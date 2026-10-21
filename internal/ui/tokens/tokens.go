// Package tokens defines the semantic palette, spacing grid and glyph vocabulary.
package tokens

// Palette contains every color role consumed by views and components.
type Palette struct {
	Name                                                         string
	BG, Surface, Raised, Border, BorderFocus, Text, Muted, Faint string
	Accent, Accent2, Success, Warning, Danger, Info              string
	Series                                                       [6]string
}

// Layout constants express shared terminal-cell geometry.
const (
	Gap          = 2
	InsetX       = 2
	InsetY       = 1
	MinWidth     = 80
	MinHeight    = 24
	DetailWidth  = 120
	WideWidth    = 180
	HeaderHeight = 4
	FooterHeight = 2
	PanelFrame   = 2
	ModalWidth   = 72
	RowHeight    = 1
)

// Glyphs supplies a consistent, optional ASCII resource/status vocabulary.
type Glyphs struct{ Anchor, Image, Container, Volume, Network, Project, Health, Running, Stopped, Paused, Error, Pull, Selected, Cursor string }

// Icons returns a font-independent Unicode or fully ASCII vocabulary.
func Icons(ascii bool) Glyphs {
	if ascii {
		return Glyphs{Anchor: "@", Image: "I", Container: "C", Volume: "V", Network: "N", Project: "P", Health: "H", Running: "+", Stopped: "-", Paused: "=", Error: "!", Pull: "v", Selected: "*", Cursor: ">"}
	}
	return Glyphs{Anchor: "⚓", Image: "◈", Container: "▣", Volume: "▤", Network: "⌘", Project: "◫", Health: "♡", Running: "●", Stopped: "○", Paused: "◐", Error: "✕", Pull: "⇣", Selected: "◆", Cursor: "▌"}
}
