package rich

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// Raster draws slices of a Page into pixels. It is stateless: a tile freed by
// the terminal can be drawn again from the page, which is why the engine keeps
// no bitmap of the whole note.
type Raster struct {
	Page    *Page
	Fonts   *Fonts
	Palette Palette
}

// Tile draws rows [first, first+rows) of the page at the cell size of Fonts.
func (r *Raster) Tile(first, rows int) *image.RGBA {
	w := max(r.Page.Width, 1) * r.Fonts.CellW
	h := max(rows, 1) * r.Fonts.CellH
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	if r.Palette.Bg.A != 0 {
		draw.Draw(img, img.Bounds(), image.NewUniform(r.Palette.Bg), image.Point{}, draw.Src)
	}
	for _, it := range r.Page.Items {
		if it.Y+max(it.H, 1) <= first || it.Y >= first+rows {
			continue
		}
		switch it.Kind {
		case ItemRect:
			r.rect(img, it, first)
		case ItemText:
			r.text(img, it, first)
		case ItemBox:
			if it.Box != nil && it.Box.Draw != nil {
				it.Box.Draw(img, it.X*r.Fonts.CellW, (it.Y-first)*r.Fonts.CellH)
			}
		}
	}
	return img
}

func (r *Raster) rect(img *image.RGBA, it Item, first int) {
	x0 := it.X * r.Fonts.CellW
	y0 := (it.Y - first) * r.Fonts.CellH
	w := it.W * r.Fonts.CellW
	h := it.H * r.Fonts.CellH
	if it.Thin {
		h = 2
	}
	if w <= 0 || h <= 0 {
		return
	}
	draw.Draw(img, image.Rect(x0, y0, x0+w, y0+h), image.NewUniform(it.Color), image.Point{}, draw.Over)
}

func (r *Raster) text(img *image.RGBA, it Item, first int) {
	if it.Color.A == 0 {
		return
	}
	face := r.Fonts.Face(it.SizePx, it.Mono)
	syntheticBold := false
	if it.Bold {
		if bold, ok := r.Fonts.BoldFace(it.SizePx); ok {
			face = bold
		} else {
			syntheticBold = true
		}
	}
	d := &font.Drawer{Dst: img, Src: image.NewUniform(it.Color), Face: face}
	baseline := (it.Y-first)*r.Fonts.CellH + it.BaselinePx
	pen := it.X*r.Fonts.CellW + it.XOffPx
	limit := it.X*r.Fonts.CellW + it.W*r.Fonts.CellW
	for _, g := range r.Fonts.Shape(it.Text, it.SizePx, it.Mono) {
		step := g.Advance.Round()
		if pen+g.X.Round() >= limit && step > 0 {
			break
		}
		d.Dot = fixed.P(pen+g.X.Round(), baseline)
		d.DrawString(string(g.R))
		if syntheticBold {
			// Without a bold face, a second strike one pixel over reads as
			// bold and keeps the advances.
			d.Dot = fixed.P(pen+g.X.Round()+1, baseline)
			d.DrawString(string(g.R))
			d.Dot = fixed.P(pen+g.X.Round(), baseline)
		}
		pen += step
	}
}

// EncodePNG encodes a tile. BestSpeed keeps a scroll frame inside its budget:
// the default level costs ~42 ms for a 1440×1152 tile, BestSpeed ~9 ms, and
// the terminal decompresses it once.
func EncodePNG(img *image.RGBA) []byte {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

// tileRows is how many cell rows one image holds: one screenful, rounded up to
// the height the view asks for. The view passes its own height.
func tileRows(pageHeight, screen int) int {
	return max(screen, 1)
}

var _ = color.RGBA{}
var _ = math.Ceil
