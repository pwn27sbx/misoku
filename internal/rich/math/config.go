package math

import "image/color"

// Config is everything the page tells a formula: how big the body text is,
// the terminal's cell size (a box is measured in cells), and the ink colour.
// A zero SizePx is 16 px and a zero cell is 12x24 px. A zero Fg falls back to
// opaque white so a caller that forgets the colour still draws; a themed
// caller must pass the page's own foreground.
type Config struct {
	SizePx       float64
	CellW, CellH int
	Fg           color.RGBA
}

const (
	defaultSizePx = 16
	defaultCellW  = 12
	defaultCellH  = 24
)

func (c Config) withDefaults() Config {
	if c.SizePx <= 0 {
		c.SizePx = defaultSizePx
	}
	if c.CellW <= 0 {
		c.CellW = defaultCellW
	}
	if c.CellH <= 0 {
		c.CellH = defaultCellH
	}
	if c.Fg.A == 0 {
		c.Fg = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	}
	return c
}
