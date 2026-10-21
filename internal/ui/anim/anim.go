// Package anim supplies finite, on-demand spring motion with no idle ticks.
package anim

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/harmonica"
	"math"
	"time"
)

// Tick marks a frame for finite active motion.
type Tick time.Time

// Spring tracks a single animated value.
type Spring struct {
	Value, Velocity, Target float64
	Reduced                 bool
	frames                  int
}

// Move updates a target and requests a frame only when motion is allowed.
func (s *Spring) Move(target float64) tea.Cmd {
	s.Target = target
	if s.Reduced {
		s.Value = target
		s.Velocity = 0
		s.frames = 0
		return nil
	}
	s.frames = 0
	return s.Next()
}

// Step advances one frame and settles within a finite frame budget.
func (s *Spring) Step() {
	if s.Reduced {
		s.Value = s.Target
		s.Velocity = 0
		return
	}
	s.Value, s.Velocity = harmonica.NewSpring(harmonica.FPS(30), 18, 1).Update(s.Value, s.Velocity, s.Target)
	s.frames++
	if s.frames >= 18 || math.Abs(s.Target-s.Value) < .01 && math.Abs(s.Velocity) < .01 {
		s.Value = s.Target
		s.Velocity = 0
	}
}

// Next schedules nothing while idle or reduced.
func (s Spring) Next() tea.Cmd {
	if s.Reduced || s.Value == s.Target && s.Velocity == 0 {
		return nil
	}
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return Tick(t) })
}
