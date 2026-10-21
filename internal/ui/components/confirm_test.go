package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestConfirmVolumePreviewAndTypedGate(t *testing.T) {
	called := 0
	d := NewVolumeRemoveConfirmation("database-data", func() tea.Cmd {
		called++
		return func() tea.Msg { return "removed" }
	})
	preview := strings.ToLower(d.View())
	for _, word := range []string{"database-data", "volume", "data", "permanent"} {
		if !strings.Contains(preview, word) {
			t.Errorf("preview lacks %q: %s", word, preview)
		}
	}
	if !d.Destructive || d.TargetName != "database-data" {
		t.Fatalf("missing destructive target: %#v", d)
	}
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyEnter}, {Code: 'y', Text: "y"}} {
		if cmd := d.Update(key); cmd != nil {
			t.Fatal("confirmed without typing target")
		}
	}
	d.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	for _, r := range "database-data" {
		d.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if called != 0 {
		t.Fatal("preview/input invoked action")
	}
	cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || called != 1 || !d.Done() {
		t.Fatal("explicit confirmation did not return command")
	}
	if msg := cmd(); msg != "removed" {
		t.Fatalf("command message = %v", msg)
	}
	if cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil || called != 1 {
		t.Fatal("action confirmed twice")
	}
}

func TestConfirmImagePrunePreviewBeforeExecution(t *testing.T) {
	executed := false
	d := NewImagePruneConfirmation(3, 1234567, func() tea.Cmd {
		return func() tea.Msg { executed = true; return nil }
	})
	preview := strings.ToLower(d.View())
	for _, word := range []string{"3", "images", "1234567", "bytes", "reclaimable"} {
		if !strings.Contains(preview, word) {
			t.Errorf("preview lacks %q: %s", word, preview)
		}
	}
	if executed {
		t.Fatal("preview executed prune")
	}
	if cmd := d.Update(tea.KeyReleaseMsg{Code: tea.KeyEnter}); cmd != nil {
		t.Fatal("release confirmed")
	}
	if cmd := d.Update(tea.KeyPressMsg{Code: 'y', Text: "y"}); cmd != nil {
		t.Fatal("unadvertised key confirmed")
	}
	cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || executed {
		t.Fatal("command not deferred until Tea executes it")
	}
	cmd()
	if !executed {
		t.Fatal("confirmed command did not execute")
	}
}

func TestConfirmCancellationIsTerminal(t *testing.T) {
	for _, code := range []rune{tea.KeyEscape, 'n'} {
		d := NewImagePruneConfirmation(0, 0, func() tea.Cmd { t.Fatal("cancel invoked action"); return nil })
		d.Update(tea.KeyPressMsg{Code: code, Text: string(code)})
		if !d.Done() {
			t.Fatal("cancel left dialog open")
		}
		if cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil {
			t.Fatal("cancel returned action")
		}
	}
}

func TestConfirmEmptyVolumeCannotConfirm(t *testing.T) {
	d := NewVolumeRemoveConfirmation("", func() tea.Cmd { t.Fatal("empty target confirmed"); return nil })
	if cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil {
		t.Fatal("empty volume accepted")
	}
}

func TestConfirmTypedNameAcceptsShiftAndUnicodeBackspace(t *testing.T) {
	d := NewVolumeRemoveConfirmation("DB-data", func() tea.Cmd { return func() tea.Msg { return "ok" } })
	d.Update(tea.KeyPressMsg{Code: 'D', Text: "D", Mod: tea.ModShift})
	d.Update(tea.KeyPressMsg{Code: 'B', Text: "B", Mod: tea.ModShift})
	d.Update(tea.KeyPressMsg{Code: 'é', Text: "é"})
	d.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	for _, r := range "-data" {
		d.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	if cmd := d.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); cmd == nil {
		t.Fatal("could not confirm uppercase target after Unicode correction")
	}
}
