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
		WidthPx: w,
		DepthPx: int(math.Ceil(ink.Depth)),
		Draw:    ink.Draw,
	}
	inlineCachePut(src, cfg, b)
	return b
}

func inlineCacheGet(src string, cfg Config) (*rich.InlineBox, bool) {
	return inlineCache.get(cacheKey{src: src, size: cfg.SizePx, fg: cfg.Fg, cw: cfg.CellW, ch: cfg.CellH})
}

func inlineCachePut(src string, cfg Config, b *rich.InlineBox) {
	inlineCache.put(cacheKey{src: src, size: cfg.SizePx, fg: cfg.Fg, cw: cfg.CellW, ch: cfg.CellH}, b)
}

// cache is a small bounded FIFO of laid-out formulas, keyed by everything that
// changes their pixels: a page rebuilt after a keystroke reuses every formula
// that did not change. Both the block and the inline formulas share it.
type cacheKey struct {
	src    string
	width  int
	size   float64
	fg     color.RGBA
	cw, ch int
}

const cacheCap = 64

type cache[T any] struct {
	mu   sync.Mutex
	m    map[cacheKey]T
	keys []cacheKey
}

func (c *cache[T]) get(k cacheKey) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[k]
	return v, ok
}

func (c *cache[T]) put(k cacheKey, v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.m[k]; ok {
		return
	}
	if len(c.keys) >= cacheCap {
		delete(c.m, c.keys[0])
		c.keys = c.keys[1:]
	}
	c.m[k] = v
	c.keys = append(c.keys, k)
}

func (c *cache[T]) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = map[cacheKey]T{}
	c.keys = nil
}

var (
	blockCache  = &cache[*rich.Box]{m: map[cacheKey]*rich.Box{}}
	inlineCache = &cache[*rich.InlineBox]{m: map[cacheKey]*rich.InlineBox{}}
)

func cacheGet(src string, width int, cfg Config) (*rich.Box, bool) {
	return blockCache.get(cacheKey{src, width, cfg.SizePx, cfg.Fg, cfg.CellW, cfg.CellH})
}

func cachePut(src string, width int, cfg Config, b *rich.Box) {
	blockCache.put(cacheKey{src, width, cfg.SizePx, cfg.Fg, cfg.CellW, cfg.CellH}, b)
}

// ResetCache empties the formula caches; tests use it when they change the
// colour or cell size on purpose.
func ResetCache() {
	blockCache.reset()
	inlineCache.reset()
}
