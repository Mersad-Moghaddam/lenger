package components

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

// ConfirmationDialog holds a preview and a one-shot, explicit confirmation gate.
// It is a child component, not a tea.Model: the owner routes messages to Update,
// wraps View in its modal, and closes the overlay when Done reports true.
// Confirmed must construct a command; daemon work belongs inside that command.
type ConfirmationDialog struct {
	Title       string
	Message     string
	TargetName  string
	Destructive bool
	Confirmed   func() tea.Cmd

	typedTarget bool
	input       string
	done        bool
}

// NewVolumeRemoveConfirmation requires the exact volume name before Enter.
func NewVolumeRemoveConfirmation(name string, confirmed func() tea.Cmd) ConfirmationDialog {
	return ConfirmationDialog{
		Title: "Remove volume", TargetName: name, Destructive: true,
		Message:   "Permanently delete this volume and all of its data. This cannot be undone.",
		Confirmed: confirmed, typedTarget: true,
	}
}

// NewImagePruneConfirmation previews the caller's snapshot, never queries or
// prunes the daemon. The owner must provide the matching, non-forced prune cmd.
func NewImagePruneConfirmation(count int, reclaimableBytes int64, confirmed func() tea.Cmd) ConfirmationDialog {
	return ConfirmationDialog{
		Title: "Prune images", Destructive: true,
		Message:   fmt.Sprintf("Remove %d images. Reclaimable: %d bytes. Review this preview before confirming.", count, reclaimableBytes),
		Confirmed: confirmed,
	}
}

// View returns a side-effect-free preview, including explicit action hints.
func (d ConfirmationDialog) View() string {
	lines := []string{style.Clean(d.Title), style.Clean(d.Message)}
	if d.TargetName != "" {
		lines = append(lines, "Target: "+style.Clean(d.TargetName))
	}
	if d.typedTarget {
		lines = append(lines, "Type the exact volume name: "+style.Clean(d.input), "enter confirm after typing name · esc cancel")
	} else {
		lines = append(lines, "enter confirm · esc / n cancel")
	}
	return strings.Join(lines, "\n")
}

// Done reports either cancellation or accepted confirmation; both are terminal.
func (d ConfirmationDialog) Done() bool { return d.done }

// Update only accepts key presses, never release or unrelated messages. It
// returns the action command at most once and never executes that command.
func (d *ConfirmationDialog) Update(msg tea.Msg) tea.Cmd {
	if d.done {
		return nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	switch key.Code {
	case tea.KeyEscape:
		d.done = true
	case tea.KeyEnter:
		if d.typedTarget && (d.TargetName == "" || d.input != d.TargetName) {
			return nil
		}
		d.done = true
		if d.Confirmed != nil {
			return d.Confirmed()
		}
	case tea.KeyBackspace:
		_, size := utf8.DecodeLastRuneInString(d.input)
		d.input = d.input[:len(d.input)-size]
	default:
		if !d.typedTarget && key.Text == "n" {
			d.done = true
		} else if d.typedTarget && key.Mod & ^tea.ModShift == 0 && key.Text != "" {
			d.input += key.Text
		}
	}
	return nil
}
