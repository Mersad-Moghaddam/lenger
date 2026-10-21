package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// State is the small, non-secret user presentation preference file.
type State struct {
	ListRatio float64 `json:"list_ratio"`
}

func LoadState() State {
	dir, err := os.UserConfigDir()
	if err != nil {
		return State{}
	}
	data, err := os.ReadFile(filepath.Join(dir, "lenger", "state.json"))
	if err != nil {
		return State{}
	}
	var state State
	_ = json.Unmarshal(data, &state)
	return state
}

func SaveState(state State) (result error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "lenger")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "state-*.json")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() {
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, err)
		}
	}()
	if _, err = f.Write(data); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(dir, "state.json"))
}

// StateWriter serializes actual writes and drops superseded queued revisions.
type StateWriter struct {
	mu       sync.Mutex
	revision uint64
}

func (w *StateWriter) Queue(state State) func() error {
	w.mu.Lock()
	w.revision++
	rev := w.revision
	w.mu.Unlock()
	return func() error {
		w.mu.Lock()
		defer w.mu.Unlock()
		if rev != w.revision {
			return nil
		}
		return SaveState(state)
	}
}
