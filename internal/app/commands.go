package app

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/platform/omarchy"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

// Command factories capture dependencies before leaving the model thread.
// A command must never read model fields while Update might mutate them.
func (m *Model) poll() tea.Cmd {
	ctx := m.ctx
	interval := time.Duration(m.config.Docker.SafetyPoll) * time.Second
	return func() tea.Msg {
		timer := time.NewTimer(interval)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			return pollMsg{}
		}
	}
}

func (m *Model) startEvents() tea.Cmd {
	ctx, service, events := m.ctx, m.service, m.events
	return func() tea.Msg {
		err := service.Events(ctx, func(event engine.Event) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case events <- event:
				return nil
			}
		})
		if ctx.Err() == nil && err != nil {
			return eventStreamStoppedMsg{err: err}
		}
		return nil
	}
}

type eventStreamStoppedMsg struct{ err error }

func (m *Model) waitEvent() tea.Cmd {
	ctx, events := m.ctx, m.events
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events:
			return eventMsg(event)
		}
	}
}

func (m *Model) waitJobs() tea.Cmd {
	ctx, manager := m.ctx, m.jobs
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case <-manager.Notifications():
			return jobsMsg(manager.Snapshot())
		}
	}
}

func (m *Model) startTheme() tea.Cmd {
	ctx, themes := m.ctx, m.themes
	return func() tea.Msg {
		_ = omarchy.DefaultSource().Watch(ctx, func(palette tokens.Palette) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case themes <- palette:
				return nil
			}
		})
		return nil
	}
}

func (m *Model) waitTheme() tea.Cmd {
	ctx, themes := m.ctx, m.themes
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case palette := <-themes:
			return themeMsg(palette)
		}
	}
}

func waitLog(ctx context.Context, ch chan tea.Msg, gen uint64) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return nil
		case msg := <-ch:
			return msg
		}
	}
}
