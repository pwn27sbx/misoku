package math

import (
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/vector"

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
)

// mathFont returns the bundled maths font, parsed once.
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

// metrics measures one already-chosen rune (glyphFont picks the alphabet and
// its fallback).
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

// scaleTolerance is how close to 1 a scale must be to take the plain drawing
// path instead of the scaled one.
const scaleTolerance = 0.01

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
	if math.Abs(sx-1) < scaleTolerance && math.Abs(sy-1) < scaleTolerance {
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

// gidMetrics measures one glyph id, the way a rune is measured.
type gidMetrics struct {
	advance, height, depth float64
	ok                     bool
}

func metricsGID(gid sfnt.GlyphIndex, sizePx float64) gidMetrics {
	otf, err := mathFont()
	if err != nil || gid == 0 {
		return gidMetrics{}
	}
	var buf sfnt.Buffer
	ppem := fixed.Int26_6(int(math.Round(sizePx * 64)))
	adv, err := otf.GlyphAdvance(&buf, gid, ppem, font.HintingNone)
	if err != nil {
		return gidMetrics{}
	}
	b, _, err := otf.GlyphBounds(&buf, gid, ppem, font.HintingNone)
	if err != nil {
		return gidMetrics{advance: float64(adv) / 64, ok: true}
	}
	m := gidMetrics{
		advance: float64(adv) / 64,
		height:  -float64(b.Min.Y) / 64,
		depth:   float64(b.Max.Y) / 64,
		ok:      true,
	}
	if m.height < 0 {
		m.height = 0
	}
	if m.depth < 0 {
		m.depth = 0
	}
	return m
}

// variantForHeight picks the smallest MATH size variant of r whose ink covers
// the target height, and reports whether one was tall enough. When none is,
// the caller gets the largest variant and can accept its height.
func variantForHeight(r rune, sizePx, target float64) (sfnt.GlyphIndex, gidMetrics, bool) {
	variants := variantGlyphs(r)
	if len(variants) == 0 {
		return 0, gidMetrics{}, false
	}
	var last sfnt.GlyphIndex
	var lastM gidMetrics
	for _, gid := range variants {
		m := metricsGID(gid, sizePx)
		if !m.ok {
			continue
		}
		last, lastM = gid, m
		if m.height+m.depth >= target {
			return gid, m, true
		}
	}
	if last != 0 {
		return last, lastM, false
	}
	return 0, gidMetrics{}, false
}

// drawGlyphID draws a glyph by id as a vector outline, which is how the MATH
// variants (real delimiter and radical shapes, not stretched copies) reach
// the page.
func drawGlyphID(dst draw.Image, gid sfnt.GlyphIndex, sizePx, x, baseline float64, col color.RGBA) {
	if col.A == 0 || gid == 0 || sizePx <= 0 {
		return
	}
	otf, err := mathFont()
	if err != nil {
		return
	}
	var buf sfnt.Buffer
	ppem := fixed.Int26_6(int(math.Round(sizePx * 64)))
	segs, err := otf.LoadGlyph(&buf, gid, ppem, nil)
	if err != nil || len(segs) == 0 {
		return
	}
	b := segs.Bounds()
	w := (b.Max.X - b.Min.X).Ceil() + 2
	h := (b.Max.Y - b.Min.Y).Ceil() + 2
	if w <= 0 || h <= 0 {
		return
	}
	z := vector.NewRasterizer(w, h)
	z.DrawOp = draw.Over
	ox := float32(-b.Min.X/64) + 1
	oy := float32(-b.Min.Y/64) + 1
	pt := func(p fixed.Point26_6) (float32, float32) {
		return float32(p.X)/64 + ox, float32(p.Y)/64 + oy
	}
	for _, s := range segs {
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			ax, ay := pt(s.Args[0])
			z.MoveTo(ax, ay)
		case sfnt.SegmentOpLineTo:
			bx, by := pt(s.Args[0])
			z.LineTo(bx, by)
		case sfnt.SegmentOpQuadTo:
			bx, by := pt(s.Args[0])
			cx, cy := pt(s.Args[1])
			z.QuadTo(bx, by, cx, cy)
		case sfnt.SegmentOpCubeTo:
			bx, by := pt(s.Args[0])
			cx, cy := pt(s.Args[1])
			dx, dy := pt(s.Args[2])
			z.CubeTo(bx, by, cx, cy, dx, dy)
		}
	}
	z.ClosePath()
	tmp := image.NewRGBA(image.Rect(0, 0, w, h))
	z.Draw(tmp, tmp.Bounds(), image.NewUniform(col), image.Point{})
	px := int(math.Round(x)) + int(math.Floor(float64(b.Min.X)/64))
	py := int(math.Round(baseline)) + int(math.Floor(float64(b.Min.Y)/64))
	draw.Draw(dst, image.Rect(px, py, px+w, py+h), tmp, image.Point{}, draw.Over)
}
