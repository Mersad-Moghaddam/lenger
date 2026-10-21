// Package theme supplies bundled palettes and validates accessible text colors.
package theme

import (
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// Names returns bundled theme names in gallery order.
func Names() []string {
	return []string{"midnight", "tokyo-night", "catppuccin-mocha", "high-contrast"}
}

// Bundled returns a semantic palette or a descriptive error for unknown names.
func Bundled(name string) (tokens.Palette, error) {
	p := tokens.Palette{Name: name}
	switch name {
	case "midnight", "lenger-midnight", "":
		p = tokens.Palette{Name: "midnight", BG: "#0b1220", Surface: "#111d2e", Raised: "#1a2940", Border: "#3b506b", BorderFocus: "#6dd6d0", Text: "#edf3ff", Muted: "#b5c5dc", Faint: "#a4b4cc", Accent: "#6dd6d0", Accent2: "#b4a1ff", Success: "#9ee6ae", Warning: "#f1ce86", Danger: "#ff9eaa", Info: "#9dcaff"}
	case "tokyo-night":
		p = tokens.Palette{Name: name, BG: "#1a1b26", Surface: "#202232", Raised: "#292e42", Border: "#545c7e", BorderFocus: "#7dcfff", Text: "#c0caf5", Muted: "#b4bdea", Faint: "#aeb8df", Accent: "#7dcfff", Accent2: "#bb9af7", Success: "#9ece6a", Warning: "#e0af68", Danger: "#f7768e", Info: "#7aa2f7"}
	case "catppuccin-mocha":
		p = tokens.Palette{Name: name, BG: "#1e1e2e", Surface: "#242436", Raised: "#313244", Border: "#585b70", BorderFocus: "#89dceb", Text: "#cdd6f4", Muted: "#bac2de", Faint: "#a6adc8", Accent: "#89dceb", Accent2: "#cba6f7", Success: "#a6e3a1", Warning: "#f9e2af", Danger: "#f38ba8", Info: "#89b4fa"}
	case "high-contrast":
		p = tokens.Palette{Name: name, BG: "#000000", Surface: "#000000", Raised: "#101010", Border: "#ffffff", BorderFocus: "#ffffff", Text: "#ffffff", Muted: "#ffffff", Faint: "#dddddd", Accent: "#80ffff", Accent2: "#ffffff", Success: "#aaffaa", Warning: "#ffffaa", Danger: "#ffaaaa", Info: "#aaddff"}
	default:
		return p, fmt.Errorf("unknown theme %q", name)
	}
	p.Series = [6]string{p.Accent, p.Accent2, p.Info, p.Success, p.Warning, p.Danger}
	return p, nil
}

// RGB validates a six-digit RGB color; malformed external colors are rejected.
func RGB(hex string) (color.RGBA, error) {
	if len(hex) != 7 || hex[0] != '#' {
		return color.RGBA{}, fmt.Errorf("invalid RGB color")
	}
	value, err := strconv.ParseUint(strings.ToLower(hex[1:]), 16, 24)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("parse RGB: %w", err)
	}
	return color.RGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 255}, nil
}

// Blend interpolates accent colors in perceptual OKLab space.
func Blend(first, last string, fraction float64) string {
	a, err := RGB(first)
	if err != nil {
		return first
	}
	b, err := RGB(last)
	if err != nil {
		return first
	}
	t := max(0, min(1, fraction))
	linear := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= .04045 {
			return x / 12.92
		}
		return math.Pow((x+.055)/1.055, 2.4)
	}
	toLab := func(c color.RGBA) [3]float64 {
		r, g, b := linear(c.R), linear(c.G), linear(c.B)
		l := math.Cbrt(.4122214708*r + .5363325363*g + .0514459929*b)
		m := math.Cbrt(.2119034982*r + .6806995451*g + .1073969566*b)
		s := math.Cbrt(.0883024619*r + .2817188376*g + .6299787005*b)
		return [3]float64{.2104542553*l + .793617785*m - .0040720468*s, 1.9779984951*l - 2.428592205*m + .4505937099*s, .0259040371*l + .7827717662*m - .808675766*s}
	}
	x, y := toLab(a), toLab(b)
	var lab [3]float64
	for n := range lab {
		lab[n] = x[n]*(1-t) + y[n]*t
	}
	l := lab[0] + .3963377774*lab[1] + .2158037573*lab[2]
	m := lab[0] - .1055613458*lab[1] - .0638541728*lab[2]
	s := lab[0] - .0894841775*lab[1] - 1.291485548*lab[2]
	l = l * l * l
	m = m * m * m
	s = s * s * s
	srgb := func(v float64) uint8 {
		v = max(0, min(1, v))
		if v <= .0031308 {
			v *= 12.92
		} else {
			v = 1.055*math.Pow(v, 1/2.4) - .055
		}
		return uint8(math.Round(v * 255))
	}
	return fmt.Sprintf("#%02x%02x%02x", srgb(4.0767416621*l-3.3077115913*m+.2309699292*s), srgb(-1.2684380046*l+2.6097574011*m-.3413193965*s), srgb(-.0041960863*l-.7034186147*m+1.707614701*s))
}

// Contrast returns the WCAG relative-luminance contrast ratio.
func Contrast(a, b string) float64 {
	lum := func(v string) float64 {
		c, err := RGB(v)
		if err != nil {
			return 0
		}
		linear := func(x uint8) float64 {
			f := float64(x) / 255
			if f <= 0.04045 {
				return f / 12.92
			}
			return math.Pow((f+0.055)/1.055, 2.4)
		}
		return .2126*linear(c.R) + .7152*linear(c.G) + .0722*linear(c.B)
	}
	x, y := lum(a), lum(b)
	if x < y {
		x, y = y, x
	}
	return (x + .05) / (y + .05)
}

// Readable replaces external low-contrast text roles with the fallback text.
func Readable(p tokens.Palette) tokens.Palette {
	for _, ptr := range []*string{&p.Text, &p.Muted, &p.Faint} {
		if Contrast(*ptr, p.Surface) < 4.5 || Contrast(*ptr, p.Raised) < 4.5 {
			*ptr = "#ffffff"
		}
	}
	return p
}
