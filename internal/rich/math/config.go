package math

import "image/color"

// Config is everything the page tells a formula: how big the body text is,
// the terminal's cell size (a box is measured in cells), and the ink colour.
// A zero SizePx is 16 px; a zero Fg is the page's text colour, so a caller
// that forgets the colour still draws something visible.
type Config struct {
	SizePx       float64
	CellW, CellH int
	Fg           color.RGBA
}

func (c Config) withDefaults() Config {
	if c.SizePx <= 0 {
		c.SizePx = 16
	}
	if c.CellW <= 0 {
		c.CellW = 12
	}
	if c.CellH <= 0 {
		c.CellH = 24
	}
	if c.Fg.A == 0 {
		c.Fg = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	}
	return c
}
