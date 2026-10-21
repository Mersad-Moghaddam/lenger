// Package docker adapts the official Moby client to Lenger's engine interface.
package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/moby/moby/client"
	"github.com/OWNER/lenger/internal/engine"
)

// Engine owns one API-negotiating Moby client.
type Engine struct { client *client.Client }

// New respects DOCKER_HOST, TLS settings and the standard proxy environment.
func New() (*Engine,error) {
	c,err:=client.New(client.FromEnv)
	if err!=nil{return nil,fmt.Errorf("create Docker client: %w",err)}
	return &Engine{client:c},nil
}

// ListImages reads all local image summaries.
func (e *Engine) ListImages(ctx context.Context)([]engine.Image,error){
	result,err:=e.client.ImageList(ctx,client.ImageListOptions{All:true,SharedSize:true})
	if err!=nil{return nil,fmt.Errorf("list images: %w",err)}
	items:=make([]engine.Image,0,len(result.Items))
	for _,i:=range result.Items{
		tags:=i.RepoTags; if len(tags)==1 && tags[0]=="<none>:<none>" {tags=nil}
		items=append(items,engine.Image{ID:i.ID,RepoTags:tags,Digests:i.RepoDigests,Size:i.Size,Shared:i.SharedSize,Created:time.Unix(i.Created,0),Dangling:len(tags)==0})
	}
	return items,nil
}

// ListContainers includes stopped containers when computing image usage.
func(e *Engine) ListContainers(ctx context.Context)([]engine.Container,error){
	result,err:=e.client.ContainerList(ctx,client.ContainerListOptions{All:true});if err!=nil{return nil,fmt.Errorf("list containers: %w",err)}
	items:=make([]engine.Container,0,len(result.Items))
	for _,c:=range result.Items{name:=engine.ShortID(c.ID);if len(c.Names)>0{name=strings.TrimPrefix(c.Names[0],"/")};items=append(items,engine.Container{ID:c.ID,Name:name,ImageID:c.ImageID,Image:c.Image,State:string(c.State),Status:c.Status,Labels:c.Labels})}
	return items,nil
}

// InspectImage joins config and history for the image details pane.
func(e *Engine) InspectImage(ctx context.Context,id string)(engine.Inspection,error){
	i,err:=e.client.ImageInspect(ctx,id);if err!=nil{return engine.Inspection{},fmt.Errorf("inspect image: %w",err)}
	out:=engine.Inspection{ID:i.ID,Platform:i.Os+"/"+i.Architecture,Digests:i.RepoDigests}
	if i.Config!=nil{out.Env=i.Config.Env;out.Entrypoint=i.Config.Entrypoint;out.Command=i.Config.Cmd;out.Labels=i.Config.Labels;for port:=range i.Config.ExposedPorts{out.Ports=append(out.Ports,string(port))}}
	h,err:=e.client.ImageHistory(ctx,id);if err!=nil{return engine.Inspection{},fmt.Errorf("image history: %w",err)}
	for _,l:=range h.Items{out.Layers=append(out.Layers,engine.Layer{ID:l.ID,Command:l.CreatedBy,Size:l.Size})}
	return out,nil
}

// RemoveImage never forces deletion or prunes parent images implicitly.
func(e *Engine) RemoveImage(ctx context.Context,id string)error{
	_,err:=e.client.ImageRemove(ctx,id,client.ImageRemoveOptions{Force:false,PruneChildren:false});if err!=nil{return fmt.Errorf("remove image: %w",err)};return nil
}

// Pull streams progress without retaining a response body after completion.
func(e *Engine) Pull(ctx context.Context,ref string,emit func(engine.Progress)error)error{
	auth,err:=registryAuth(ctx,ref);if err!=nil{return fmt.Errorf("resolve registry credentials: %w",err)}
	body,err:=e.client.ImagePull(ctx,ref,client.ImagePullOptions{RegistryAuth:auth});if err!=nil{return fmt.Errorf("pull image: %w",err)}
	defer body.Close() //nolint:errcheck // The decoder/context error is authoritative.
	return DecodePull(ctx,body,emit)
}

// Events reconnects with cancellable exponential backoff and a replay cursor.
func(e *Engine) Events(ctx context.Context,emit func(engine.Event)error)error{
	backoff:=time.Second;since:=""
	for{
		if err:=ctx.Err();err!=nil{return err}
		streamCtx,cancel:=context.WithCancel(ctx)
		stream:=e.client.Events(streamCtx,client.EventsListOptions{Since:since})
		if err:=emit(engine.Event{Kind:"connection",Action:"connecting",At:time.Now()});err!=nil{cancel();return err}
		var streamErr error
	loop: for{select{
		case <-ctx.Done():cancel();return ctx.Err()
		case ev,ok:=<-stream.Messages:
			if !ok{streamErr=fmt.Errorf("event stream closed");break loop}
			backoff=time.Second;at:=time.Unix(0,ev.TimeNano);since=at.Format(time.RFC3339Nano)
			if err:=emit(engine.Event{ID:ev.Actor.ID,Kind:string(ev.Type),Action:string(ev.Action),At:at});err!=nil{cancel();return err}
		case err,ok:=<-stream.Err:
			if !ok||err==nil{streamErr=fmt.Errorf("event stream ended")}else{streamErr=err};break loop
		}}
		cancel()
		if ctx.Err()!=nil{return ctx.Err()}
		if err:=emit(engine.Event{Kind:"connection",Action:"reconnecting",At:time.Now(),Err:engine.Classify(streamErr)});err!=nil{return err}
		timer:=time.NewTimer(backoff);select{case <-ctx.Done():timer.Stop();return ctx.Err();case <-timer.C:}
		backoff=min(30*time.Second,backoff*2)
	}
}

// Close closes idle transport connections.
func(e *Engine) Close()error{return e.client.Close()}
