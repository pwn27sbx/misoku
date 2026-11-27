package math

import (
	_ "embed"
	"image"
	"image/color"
	"image/draw"

	xdraw "golang.org/x/image/draw"
	"math"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// The one bundled face: Latin Modern Math (GUST Font License, see
// fonts/GUST-FONT-LICENSE.txt). Maths glyphs must be TeX shaped, and a system
// font that covers them cannot be assumed on Linux, macOS or Windows, so this
// is the only face the binary carries. Text faces stay the system's.
//
//go:embed fonts/latinmodern-math.otf
var fontData []byte

var (
	fontOnce sync.Once
	fontOTF  *sfnt.Font
	fontErr  error

	faceMu sync.Mutex
	faces  = map[int]font.Face{}
	sfnts  = map[int]*sfnt.Font{}
)

// Face returns the bundled maths font, parsed once.
func mathFont() (*sfnt.Font, error) {
	fontOnce.Do(func() {
		fontOTF, fontErr = opentype.Parse(fontData)
	})
	return fontOTF, fontErr
}

// faceAt returns a drawing face at sizePx, cached down to 1/16 px.
func faceAt(sizePx float64) (font.Face, error) {
	key := int(math.Round(sizePx * 16))
	faceMu.Lock()
	defer faceMu.Unlock()
	if f, ok := faces[key]; ok {
		return f, nil
	}
	otf, err := mathFont()
	if err != nil {
		return nil, err
	}
	f, err := opentype.NewFace(otf, &opentype.FaceOptions{
		Size: float64(key) / 16, DPI: 72, Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}
	faces[key] = f
	return f, nil
}

// hasGlyph reports whether the bundled face draws r.
func hasGlyph(r rune) bool {
	otf, err := mathFont()
	if err != nil {
		return false
	}
	var buf sfnt.Buffer
	gid, err := otf.GlyphIndex(&buf, r)
	return err == nil && gid != 0
}

// glyphMetrics is one glyph's ink: its advance and its extent above and below
// the baseline, in pixels at size.
type glyphMetrics struct {
	advance, height, depth, left, right float64
	ok                                  bool
}

// metrics measures r. When the styled version is missing the plain rune is
// used, so \mathbb{R} falls back to an R instead of a hole.
func metrics(r rune, sizePx float64) glyphMetrics {
	if !hasGlyph(r) {
		return glyphMetrics{}
	}
	face, err := faceAt(sizePx)
	if err != nil {
		return glyphMetrics{}
	}
	adv, ok := face.GlyphAdvance(r)
	if !ok {
		return glyphMetrics{}
	}
	b, adv2, ok := face.GlyphBounds(r)
	if !ok {
		return glyphMetrics{advance: float64(adv) / 64, ok: true}
	}
	m := glyphMetrics{
		advance: float64(max(adv, adv2)) / 64,
		left:    float64(b.Min.X) / 64,
		right:   float64(b.Max.X) / 64,
		ok:      true,
	}
	m.height = -float64(b.Min.Y) / 64
	m.depth = float64(b.Max.Y) / 64
	if m.height < 0 {
		m.height = 0
	}
	if m.depth < 0 {
		m.depth = 0
	}
	return m
}

// glyphFont is the rune actually drawn for a symbol after its alphabet is
// applied, falling back to the base rune when the styled block lacks it.
func glyphFont(r rune, f Font) rune {
	out := mathAlnum(r, f)
	if out != r && !hasGlyph(out) {
		return r
	}
	return out
}

// drawGlyph draws one glyph with its pen at (x, baseline). Scale stretches it
// about the pen: 1 is the size it was measured at.
func drawGlyph(dst draw.Image, r rune, sizePx, x, baseline, sx, sy float64, col color.RGBA) {
	if col.A == 0 || sx == 0 || sy == 0 {
		return
	}
	face, err := faceAt(sizePx)
	if err != nil {
		return
	}
	if math.Abs(sx-1) < 0.01 && math.Abs(sy-1) < 0.01 {
		d := &font.Drawer{Dst: dst, Src: image.NewUniform(col), Face: face}
		d.Dot = fixed.P(int(math.Round(x)), int(math.Round(baseline)))
		d.DrawString(string(r))
		return
	}
	b, _, ok := face.GlyphBounds(r)
	if !ok {
		return
	}
	w := b.Max.X - b.Min.X
	h := b.Max.Y - b.Min.Y
	if w <= 0 || h <= 0 {
		return
	}
	mask := image.NewAlpha(image.Rect(0, 0, w.Ceil(), h.Ceil()))
	d := &font.Drawer{Dst: mask, Src: image.NewUniform(color.Alpha{A: 0xff}), Face: face}
	d.Dot = fixed.Point26_6{X: -b.Min.X, Y: -b.Min.Y}
	d.DrawString(string(r))
	tw := int(math.Round(float64(w) / 64 * sx))
	th := int(math.Round(float64(h) / 64 * sy))
	if tw <= 0 || th <= 0 {
		return
	}
	scaled := image.NewAlpha(image.Rect(0, 0, tw, th))
	x0 := int(math.Round(x)) + int(math.Round(float64(b.Min.X)/64*sx))
	y0 := int(math.Round(baseline)) + int(math.Round(float64(b.Min.Y)/64*sy))
	xdraw.BiLinear.Scale(scaled, scaled.Bounds(), mask, mask.Bounds(), draw.Src, nil)
	draw.DrawMask(dst, image.Rect(x0, y0, x0+tw, y0+th),
		image.NewUniform(col), image.Point{}, scaled, image.Point{}, draw.Over)
}

// glyphHeight is the ink height of one glyph at size, used to pick a
// delimiter size before it is stretched.
func glyphHeight(r rune, sizePx float64) float64 {
	m := metrics(r, sizePx)
	return m.height + m.depth
}

// ascentOf is the face's ascent at size, for baselines inside a box.
func ascentOf(sizePx float64) float64 {
	face, err := faceAt(sizePx)
	if err != nil {
		return sizePx * 0.75
	}
	return float64(face.Metrics().Ascent) / 64
}
