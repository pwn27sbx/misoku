package diagram

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"sync"

	"github.com/pwn27sbx/misoku/internal/rich"
)

const maxCachedDiagrams = 64

// Renderer turns Mermaid blocks into rich.Box values. A renderer is bound to
// one font stack and palette; its cached bitmaps are therefore theme- and
// size-specific and Draw only has to blit the requested box.
type Renderer struct {
	fonts   *rich.Fonts
	palette rich.Palette

	mu    sync.Mutex
	cache map[string]*image.RGBA
	order []string
}

// New creates a renderer for the page's fonts and colours.
func New(fonts *rich.Fonts, palette rich.Palette) *Renderer {
	return &Renderer{fonts: fonts, palette: palette, cache: make(map[string]*image.RGBA)}
}

// Box implements the rich.LayoutOptions.Diagram seam. Unsupported source is
// rendered as a labelled source panel, never as a silently wrong picture.
func (r *Renderer) Box(block rich.Block) *rich.Box {
	if r == nil || r.fonts == nil {
		return nil
	}
	source := strings.Join(block.Code, "\n")
	if source == "" {
		source = strings.Join(block.Lines, "\n")
	}
	img := r.render(source)
	if img == nil {
		return nil
	}
	w := max(img.Bounds().Dx()/r.fonts.CellW, 1)
	h := max(img.Bounds().Dy()/r.fonts.CellH, 1)
	box := &rich.Box{W: w, H: h}
	box.Draw = func(dst draw.Image, x, y int) {
		if img == nil || box.W <= 0 || box.H <= 0 {
			return
		}
		maxW := box.W * r.fonts.CellW
		maxH := box.H * r.fonts.CellH
		at := image.Pt(x, y)
		rct := image.Rect(x, y, x+maxW, y+maxH).
			Intersect(img.Bounds().Add(at)).
			Intersect(dst.Bounds())
		if rct.Empty() {
			return
		}
		draw.Draw(dst, rct, img, rct.Min.Sub(at), draw.Over)
	}
	return box
}

// Layout lays out a parsed diagram with the supplied fonts and palette.
func Layout(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
	registry.RLock()
	layout, ok := registry.layouts[d.Kind]
	registry.RUnlock()
	if !ok {
		return Scene{}, unknown(d.Kind, 0, string(d.Kind))
	}
	return layout(d, fonts, palette)
}

func (r *Renderer) render(source string) *image.RGBA {
	r.mu.Lock()
	defer r.mu.Unlock()
	if img, ok := r.cache[source]; ok {
		return img
	}

	var scene Scene
	d, err := Parse(source)
	if err == nil {
		scene, err = Layout(d, r.fonts, r.palette)
	}
	if err != nil {
		scene = fallbackScene(source, err, r.fonts, r.palette)
	}
	img := renderScene(scene, r.fonts)
	if img == nil {
		return nil
	}
	r.cache[source] = img
	r.order = append(r.order, source)
	if len(r.order) > maxCachedDiagrams {
		oldest := r.order[0]
		r.order = r.order[1:]
		delete(r.cache, oldest)
	}
	return img
}

func fallbackScene(source string, cause error, fonts *rich.Fonts, palette rich.Palette) Scene {
	s := newScene()
	const width = 80
	note := "Mermaid source shown because: " + cause.Error()
	noteLines, _, noteLineH := labelSize(note, fonts, width-4, fonts.BodyPx, false)
	var code []string
	for _, line := range strings.Split(source, "\n") {
		wrapped, _, _ := labelSize(line, fonts, width-4, fonts.MonoPx, true)
		code = append(code, wrapped...)
	}
	if len(code) == 0 {
		code = []string{""}
	}
	_, _, codeLineH := labelSize("M", fonts, width-4, fonts.MonoPx, true)
	height := 2 + len(noteLines)*noteLineH + len(code)*codeLineH
	s.Width, s.Height = width, height
	if palette.CodeBg.A != 0 {
		s.Rect(0, 0, width, height, palette.CodeBg, palette.CodeBg)
	}
	s.Rect(0, 0, width, height, color.RGBA{}, rule(palette))
	s.TextLines(noteLines, 2, 1, width-4, noteLineH, fonts.BodyPx, AlignLeft, accent(palette, 3), false)
	s.TextLines(code, 2, 1+len(noteLines)*noteLineH, width-4, codeLineH, fonts.MonoPx, AlignLeft, foreground(palette), true)
	return s
}
