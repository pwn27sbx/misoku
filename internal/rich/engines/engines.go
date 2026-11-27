// Package engines wires the rich page's engines into the layout. The app,
// headless mode and the page-level tests all lay a note out through Engines,
// so one configuration (fonts, palette, cell size) draws every page and the
// three paths cannot drift apart.
package engines

import (
	"image"
	"image/draw"
	stdmath "math"

	xdraw "golang.org/x/image/draw"

	"github.com/pwn27sbx/misoku/internal/rich"
	"github.com/pwn27sbx/misoku/internal/rich/diagram"
	"github.com/pwn27sbx/misoku/internal/rich/math"
)

// Engines is the page's engines bound to one font stack and palette. A caller
// that lays the same note out repeatedly (the app rebuilds on a theme change,
// a resize and every keystroke) keeps one Engines, so the formula and diagram
// caches survive the rebuild.
type Engines struct {
	fonts    *rich.Fonts
	palette  rich.Palette
	cfg      math.Config
	diagrams *diagram.Renderer
}

// New binds the engines to the page's fonts and colours.
func New(f *rich.Fonts, pal rich.Palette) *Engines {
	return &Engines{
		fonts:    f,
		palette:  pal,
		cfg:      math.Config{SizePx: f.BodyPx, CellW: f.CellW, CellH: f.CellH, Fg: pal.Fg},
		diagrams: diagram.New(f, pal),
	}
}

// Palette is the palette these engines were built with, so a caller can tell
// when a theme change needs new ones.
func (e *Engines) Palette() rich.Palette { return e.palette }

// Page lays a note out with every engine installed. width is the page width in
// cells; the block engines get the same width in pixels so a formula wider
// than the page is scaled down into it.
func (e *Engines) Page(doc rich.Document, width int, img rich.ImageOptions) *rich.Page {
	widthPx := max(width, 1) * e.fonts.CellW
	return rich.Layout(doc, rich.LayoutOptions{
		Width:   width,
		Fonts:   e.fonts,
		Palette: e.palette,
		Images:  img,
		Math: func(b rich.Block) *rich.Box {
			return math.FromBlock(b, widthPx, e.cfg)
		},
		MathInline: func(src string, sizePx float64) *rich.InlineBox {
			inline := e.cfg
			inline.SizePx = sizePx
			return math.Inline(src, inline)
		},
		Diagram: e.diagrams.Box,
	})
}

// Page lays a note out with engines built for fonts and palette. Callers that
// lay pages out repeatedly keep one Engines instead, so the caches survive.
func Page(doc rich.Document, width int, f *rich.Fonts, pal rich.Palette, img rich.ImageOptions) *rich.Page {
	return New(f, pal).Page(doc, width, img)
}

// Zoom lays one diagram block out to fill a pane of w×h cells: the engine
// scales its text until the picture covers as much of the pane as it can
// without leaving it, and the page centres it. ok is false when the engine
// refuses the block, which keeps the page's own box on screen.
func (e *Engines) Zoom(block rich.Block, w, h int) (*rich.Page, bool) {
	w, h = max(w, 8), max(h, 4)
	box := e.diagrams.Box(block)
	if box == nil {
		return nil, false
	}
	// The layout scales with the text, not exactly linearly, so the fit is a
	// few multiplicative steps. Only a picture that fits becomes the answer:
	// an overshoot is a signal to shrink, never something to clip.
	scale := 1.0
	var best *rich.Box
	for step := 0; step < 6; step++ {
		k := stdmath.Min(float64(w)/float64(max(box.W, 1)), float64(h)/float64(max(box.H, 1)))
		if box.W <= w && box.H <= h {
			best = box
			if k < 1.06 {
				break
			}
		}
		next := stdmath.Max(0.4, stdmath.Min(scale*k, 8))
		if stdmath.Abs(next-scale) < 0.02 {
			break
		}
		fonts, err := e.fonts.WithTextScale(next)
		if err != nil {
			break
		}
		cand := diagram.New(fonts, e.palette).Box(block)
		if cand == nil {
			break
		}
		box, scale = cand, next
	}
	if best == nil {
		// No text scale fits a pane this small: the picture is scaled down
		// as pixels instead of leaving half of it outside the pane.
		scaled, ok := e.scaledBox(block, w, h)
		if !ok {
			return nil, false
		}
		best = scaled
	}
	page := &rich.Page{Width: w, Height: h}
	page.Items = []rich.Item{{
		Kind: rich.ItemBox,
		X:    max((w-best.W)/2, 0), Y: max((h-best.H)/2, 0),
		W: best.W, H: best.H, Box: best,
	}}
	return page, true
}

// scaledBox fits a diagram's bitmap into w×h cells, keeping its aspect when
// its own layout cannot shrink any further.
func (e *Engines) scaledBox(block rich.Block, w, h int) (*rich.Box, bool) {
	src := e.diagrams.Image(block)
	if src == nil || src.Bounds().Empty() {
		return nil, false
	}
	cw, ch := e.fonts.CellW, e.fonts.CellH
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	k := stdmath.Min(float64(w*cw)/float64(sw), float64(h*ch)/float64(sh))
	if k <= 0 {
		return nil, false
	}
	tw := max(int(stdmath.Round(float64(sw)*k)), 1)
	th := max(int(stdmath.Round(float64(sh)*k)), 1)
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	box := &rich.Box{W: (tw + cw - 1) / cw, H: (th + ch - 1) / ch}
	box.Draw = func(dstImg draw.Image, x, y int) {
		draw.Draw(dstImg, image.Rect(x, y, x+tw, y+th).Intersect(dstImg.Bounds()),
			dst, image.Point{}, draw.Over)
	}
	return box, true
}
