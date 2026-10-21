// Package zones provides deterministic terminal-cell mouse hit testing.
package zones

// Rect identifies a half-open clickable region.
type Rect struct {
	ID                  string
	X, Y, Width, Height int
}

// Registry is a frame-local list of clickable regions, back to front.
type Registry []Rect

// Hit returns the topmost region containing the point.
func (r Registry) Hit(x, y int) string {
	for n := len(r) - 1; n >= 0; n-- {
		z := r[n]
		if x >= z.X && x < z.X+z.Width && y >= z.Y && y < z.Y+z.Height {
			return z.ID
		}
	}
	return ""
}
