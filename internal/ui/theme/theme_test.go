package theme

import (
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"testing"
)

func TestBundledTextContrast(t *testing.T) {
	for _, name := range Names() {
		t.Run(name, func(t *testing.T) {
			p, err := Bundled(name)
			if err != nil {
				t.Fatal(err)
			}
			checkTextRoles(t, p)
		})
	}
}
func checkTextRoles(t *testing.T, p tokens.Palette) {
	t.Helper()
	for _, fg := range []string{p.Text, p.Muted, p.Faint} {
		for _, bg := range []string{p.BG, p.Surface, p.Raised} {
			if ratio := Contrast(fg, bg); ratio < 4.5 {
				t.Errorf("%s against %s contrast %.2f < 4.5", fg, bg, ratio)
			}
		}
	}
}
