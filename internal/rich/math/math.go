package math

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"sync"

	"github.com/pwn27sbx/misoku/internal/rich"
)

// Box lays a block's maths source out and hands the page a sized box. It is
// the engine's only integration point: the page never sees a glyph. A formula
// that cannot be parsed returns nil, which makes the page draw the source in a
// bordered box instead of nothing.
func Box(src []string, widthPx int, cfg Config) *rich.Box {
	srcText := strings.TrimSpace(strings.Join(src, "\n"))
	if srcText == "" {
		return nil
	}
	cfg = cfg.withDefaults()
	if b, ok := cacheGet(srcText, widthPx, cfg); ok {
		return b
	}
	formula, err := Parse(srcText)
	if err != nil {
		return nil
	}
	l := &layouter{base: cfg.SizePx}
	laid := l.list(formula.Nodes, cfg.SizePx, styleDisplay, 0)
	if laid.w <= 0 {
		return nil
	}
	img, _, _, _ := render(laid, cfg.Fg)
	img = scaleImage(img, widthPx)
	w := int(math.Ceil(float64(img.Bounds().Dx()) / float64(cfg.CellW)))
	h := int(math.Ceil(float64(img.Bounds().Dy()) / float64(cfg.CellH)))
	if w < 1 || h < 1 {
		return nil
	}
	b := &rich.Box{
		W: w,
		H: h,
		Draw: func(dst draw.Image, x, y int) {
			r := image.Rect(x, y, x+img.Bounds().Dx(), y+img.Bounds().Dy())
			draw.Draw(dst, r, img, img.Bounds().Min, draw.Over)
		},
	}
	cachePut(srcText, widthPx, cfg, b)
	return b
}

// FromBlock is the function the rich layout installs as LayoutOptions.Math.
func FromBlock(b rich.Block, widthPx int, cfg Config) *rich.Box {
	return Box(b.Code, widthPx, cfg)
}

// Ink is a formula laid out for a text line: its width and its extent above
// and below the baseline. The rich page cannot place it inside a paragraph
// yet (the S1 display list has no inline box), so this is the seam the inline
// form will use.
type Ink struct {
	Width, Ascent, Depth float64
	img                  *image.RGBA
	padX, baseline       int
}

// LayoutInline lays a formula out in text style for a line's baseline.
func LayoutInline(src string, cfg Config) (*Ink, error) {
	cfg = cfg.withDefaults()
	formula, err := Parse(strings.TrimSpace(src))
	if err != nil {
		return nil, err
	}
	l := &layouter{base: cfg.SizePx}
	laid := l.list(formula.Nodes, cfg.SizePx, styleText, 0)
	img, padX, _, baseline := render(laid, cfg.Fg)
	return &Ink{
		Width: laid.w, Ascent: laid.h, Depth: laid.d,
		img: img, padX: padX, baseline: baseline,
	}, nil
}

// Draw blits the formula with its baseline at baselineY and its left edge at
// x. It is safe on any draw.Image and never draws outside the image.
func (i *Ink) Draw(dst draw.Image, x, baselineY int) {
	if i == nil || i.img == nil {
		return
	}
	dx := x - i.padX
	dy := baselineY - i.baseline
	r := image.Rect(dx, dy, dx+i.img.Bounds().Dx(), dy+i.img.Bounds().Dy())
	draw.Draw(dst, r, i.img, i.img.Bounds().Min, draw.Over)
}

// Inline lays an inline formula out in text style and hands the page its ink:
// the width in pixels and a draw function that puts its baseline where the
// text line's baseline is. Nil means the page keeps the formula's source as
// text, which is what a page without an engine draws.
func Inline(src string, cfg Config) *rich.InlineBox {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil
	}
	cfg = cfg.withDefaults()
	if b, ok := inlineCacheGet(src, cfg); ok {
		return b
	}
	ink, err := LayoutInline(src, cfg)
	if err != nil || ink == nil || ink.img == nil {
		return nil
	}
	w := int(math.Ceil(ink.Width))
	if w <= 0 {
		return nil
	}
	b := &rich.InlineBox{
		WidthPx:  w,
		AscentPx: int(math.Ceil(ink.Ascent)),
		DepthPx:  int(math.Ceil(ink.Depth)),
		Draw:     ink.Draw,
	}
	inlineCachePut(src, cfg, b)
	return b
}

// A page rebuild after a keystroke reuses every inline formula that did not
// change; the key is everything that changes its pixels.
type inlineKey struct {
	src    string
	size   float64
	fg     color.RGBA
	cw, ch int
}

const inlineCacheCap = 128

var (
	inlineMu   sync.Mutex
	inlineMap  = map[inlineKey]*rich.InlineBox{}
	inlineKeys []inlineKey
)

func inlineCacheGet(src string, cfg Config) (*rich.InlineBox, bool) {
	inlineMu.Lock()
	defer inlineMu.Unlock()
	b, ok := inlineMap[inlineKey{src, cfg.SizePx, cfg.Fg, cfg.CellW, cfg.CellH}]
	return b, ok
}

func inlineCachePut(src string, cfg Config, b *rich.InlineBox) {
	inlineMu.Lock()
	defer inlineMu.Unlock()
	k := inlineKey{src, cfg.SizePx, cfg.Fg, cfg.CellW, cfg.CellH}
	if _, ok := inlineMap[k]; ok {
		return
	}
	if len(inlineKeys) >= inlineCacheCap {
		delete(inlineMap, inlineKeys[0])
		inlineKeys = inlineKeys[1:]
	}
	inlineMap[k] = b
	inlineKeys = append(inlineKeys, k)
}

// The cache key is the formula plus everything that changes its pixels. A
// page rebuild after a keystroke reuses every formula that did not change.
type cacheKey struct {
	src    string
	width  int
	size   float64
	fg     color.RGBA
	cw, ch int
}

const cacheCap = 64

var (
	cacheMu   sync.Mutex
	cacheMap  = map[cacheKey]*rich.Box{}
	cacheKeys []cacheKey
)

func cacheGet(src string, width int, cfg Config) (*rich.Box, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	b, ok := cacheMap[cacheKey{src, width, cfg.SizePx, cfg.Fg, cfg.CellW, cfg.CellH}]
	return b, ok
}

func cachePut(src string, width int, cfg Config, b *rich.Box) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	k := cacheKey{src, width, cfg.SizePx, cfg.Fg, cfg.CellW, cfg.CellH}
	if _, ok := cacheMap[k]; ok {
		return
	}
	if len(cacheKeys) >= cacheCap {
		delete(cacheMap, cacheKeys[0])
		cacheKeys = cacheKeys[1:]
	}
	cacheMap[k] = b
	cacheKeys = append(cacheKeys, k)
}

// ResetCache empties the formula cache; tests use it when they change the
// colour or cell size on purpose.
func ResetCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cacheMap = map[cacheKey]*rich.Box{}
	cacheKeys = nil
}
