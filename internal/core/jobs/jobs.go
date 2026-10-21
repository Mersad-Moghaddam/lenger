// Package jobs manages cancellable pulls and session-local progress history.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/OWNER/lenger/internal/engine"
)

// Puller is the minimal service used by a pull job.
type Puller interface{Pull(context.Context,string,func(engine.Progress)error)error}

// Job is a detached snapshot safe for rendering after the manager unlocks.
type Job struct{
	ID uint64
	Target,State string
	Layers []engine.Progress
	Current,Total int64
	Speed float64
	ETA time.Duration
	Started time.Time
	Err *engine.ClassifiedError
}

// Fraction returns known byte completion, not a promise about extraction time.
func(j Job)Fraction()float64{if j.State=="succeeded"{return 1};if j.Total<=0{return 0};return min(1,float64(j.Current)/float64(j.Total))}

type running struct{job Job;layers map[string]engine.Progress;cancel context.CancelFunc}

// Manager serializes job changes and coalesces notifications without blocking.
type Manager struct{mu sync.Mutex;next uint64;order []uint64;active map[uint64]*running;notify chan struct{}}

// New creates a session-local job manager with no goroutines or timers.
func New()*Manager{return &Manager{active:make(map[uint64]*running),notify:make(chan struct{},1)}}

// Notifications signals that snapshots changed; one signal may cover many frames.
func(m *Manager)Notifications()<-chan struct{}{return m.notify}

func(m *Manager)signal(){select{case m.notify<-struct{}{}:default:}}

// Start starts a cancellable pull and coalesces concurrent pulls of one reference.
func(m *Manager)Start(ctx context.Context,p Puller,target string)uint64{
	m.mu.Lock();for _,r:=range m.active{if r.job.Target==target&&r.job.State=="running"{id:=r.job.ID;m.mu.Unlock();return id}}
	m.next++;id:=m.next;ctx,cancel:=context.WithCancel(ctx);r:=&running{job:Job{ID:id,Target:target,State:"running",Started:time.Now()},layers:make(map[string]engine.Progress),cancel:cancel};m.active[id]=r;m.order=append(m.order,id);m.mu.Unlock();m.signal()
	go func(){
		var err error
		defer func(){
			if recover()!=nil{err=fmt.Errorf("pull worker panicked; operation stopped safely")}
			cancel();m.mu.Lock();r.job.State="succeeded";if err!=nil{r.job.State="failed";if errors.Is(err,context.Canceled){r.job.State="cancelled"};r.job.Err=engine.Classify(err)};m.mu.Unlock();m.signal()
		}()
		err=p.Pull(ctx,target,func(progress engine.Progress)error{
			if err:=ctx.Err();err!=nil{return err}
			m.mu.Lock();previous:=r.layers[progress.ID]
			if progress.Total==0{progress.Total=previous.Total};if progress.Current==0&&progress.Status!="Downloading"{progress.Current=previous.Current}
			if progress.Status=="Pull complete"||progress.Status=="Already exists"{progress.Current=progress.Total}
			r.layers[progress.ID]=progress;r.job.Current=0;r.job.Total=0
			for _,l:=range r.layers{r.job.Current+=l.Current;r.job.Total+=l.Total}
			seconds:=time.Since(r.job.Started).Seconds();if seconds>0{r.job.Speed=float64(r.job.Current)/seconds};if r.job.Speed>0{r.job.ETA=time.Duration(float64(max(0,r.job.Total-r.job.Current))/r.job.Speed*float64(time.Second))}
			m.mu.Unlock();m.signal();return nil
		})
	}()
	return id
}

// Snapshot returns jobs in creation order with sorted, detached layers.
func(m *Manager)Snapshot()[]Job{
	m.mu.Lock();defer m.mu.Unlock();items:=make([]Job,0,len(m.order))
	for _,id:=range m.order{r:=m.active[id];j:=r.job;j.Layers=make([]engine.Progress,0,len(r.layers));for _,l:=range r.layers{j.Layers=append(j.Layers,l)};sort.Slice(j.Layers,func(a,b int)bool{return j.Layers[a].ID<j.Layers[b].ID});items=append(items,j)}
	return items
}

// Cancel requests cancellation without waiting for network I/O.
func(m *Manager)Cancel(id uint64){m.mu.Lock();defer m.mu.Unlock();if r:=m.active[id];r!=nil&&r.job.State=="running"{r.cancel()}}

// CancelAll requests termination of all running jobs on exit.
func(m *Manager)CancelAll(){m.mu.Lock();defer m.mu.Unlock();for _,r:=range m.active{r.cancel()}}
