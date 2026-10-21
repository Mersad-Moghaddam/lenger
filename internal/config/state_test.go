package config

import "testing"

func TestStateWriterSupersededCommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var w StateWriter
	older := w.Queue(State{ListRatio: .30})
	newer := w.Queue(State{ListRatio: .45})
	if err := newer(); err != nil {
		t.Fatal(err)
	}
	if err := older(); err != nil {
		t.Fatal(err)
	}
	if got := LoadState().ListRatio; got != .45 {
		t.Fatalf("out-of-order writes persisted %v", got)
	}
}
