// Package engines wires the rich page's engines into the layout. The app,
// headless mode and the page-level tests all lay a note out through Page, so
// one configuration (fonts, palette, cell size) draws every page and the three
// paths cannot drift apart.
package engines

import (
	"github.com/pwn27sbx/misoku/internal/rich"
	"github.com/pwn27sbx/misoku/internal/rich/math"
)

// Page lays a note out with the maths engine installed. width is the page
// width in cells; the block engine gets the same width in pixels so a formula
// wider than the page is scaled down into it.
//
// Diagrams are not wired yet: LayoutOptions.Diagram stays nil, so a diagram
// block keeps its bordered source box until a diagram engine is passed here.
func Page(doc rich.Document, width int, f *rich.Fonts, pal rich.Palette, img rich.ImageOptions) *rich.Page {
	cfg := math.Config{SizePx: f.BodyPx, CellW: f.CellW, CellH: f.CellH, Fg: pal.Fg}
	widthPx := max(width, 1) * f.CellW
	return rich.Layout(doc, rich.LayoutOptions{
		Width:   width,
		Fonts:   f,
		Palette: pal,
		Images:  img,
		Math: func(b rich.Block) *rich.Box {
			return math.FromBlock(b, widthPx, cfg)
		},
		MathInline: func(src string, sizePx float64) *rich.InlineBox {
			inline := cfg
			inline.SizePx = sizePx
			return math.Inline(src, inline)
		},
	})
}
