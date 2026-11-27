package engines

import (
	"image/draw"
	"os"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/rich"
	"github.com/pwn27sbx/misoku/internal/theme"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// testFonts is the page's font stack: the Go fonts, so nothing depends on the
// machine's installed faces.
func testFonts(t *testing.T) *rich.Fonts {
	t.Helper()
	f, err := rich.NewFontsWithStyles(goregular.TTF, gomono.TTF, gobold.TTF, goitalic.TTF,
		rich.FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func testPage(t *testing.T, src string, width int) (*rich.Page, *rich.Fonts, rich.Palette) {
	t.Helper()
	f := testFonts(t)
	pal := rich.PaletteFromTheme(theme.DefaultTheme())
	doc := rich.Parse([]byte(src))
	return Page(doc, width, f, pal, rich.ImageOptions{}), f, pal
}

func boxes(p *rich.Page) []rich.Item {
	var out []rich.Item
	for _, it := range p.Items {
		if it.Kind == rich.ItemBox {
			out = append(out, it)
		}
	}
	return out
}

func textOf(p *rich.Page) string {
	var parts []string
	for _, it := range p.Items {
		if it.Kind == rich.ItemText {
			parts = append(parts, it.Text)
		}
	}
	// Runs are split at every style boundary; the reader sees them joined.
	return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
}

// inkIn counts non-background pixels inside a cell rectangle of the page.
func inkIn(p *rich.Page, f *rich.Fonts, pal rich.Palette, x, y, w, h int) int {
	img := (&rich.Raster{Page: p, Fonts: f, Palette: pal}).Tile(y, max(h, 1))
	n := 0
	for py := 0; py < h*f.CellH; py++ {
		for px := x * f.CellW; px < (x+w)*f.CellW; px++ {
			if py >= img.Bounds().Dy() || px >= img.Bounds().Dx() {
				continue
			}
			if img.RGBAAt(px, py) != pal.Bg {
				n++
			}
		}
	}
	return n
}

// The app and headless mode lay a note out through Page, so a display maths
// block has to arrive at the raster as a drawn box, not as its source text.
func TestPageDrawsDisplayMathAsABox(t *testing.T) {
	p, f, pal := testPage(t, "Before.\n\n$$\n\\int_0^1 x \\, dx\n$$\n\nAfter.\n", 70)

	bs := boxes(p)
	if len(bs) != 1 {
		t.Fatalf("maths boxes on the page = %d, want 1 (items: %+v)", len(bs), p.Items)
	}
	box := bs[0]
	if box.W < 4 || box.H < 1 {
		t.Fatalf("maths box = %dx%d cells, want a sized box", box.W, box.H)
	}
	if got := textOf(p); strings.Contains(got, "int_0") {
		t.Fatalf("the maths source was drawn as text: %q", got)
	}
	if n := inkIn(p, f, pal, box.X, box.Y, box.W, box.H); n == 0 {
		t.Fatal("the maths box painted no ink")
	}
	// The paragraph after the maths keeps its own source line.
	afterRow := -1
	for _, it := range p.Items {
		if it.Kind == rich.ItemText && strings.Contains(it.Text, "After.") {
			afterRow = it.Y
		}
	}
	if afterRow < 0 {
		t.Fatal("the paragraph after the maths is not drawn")
	}
	if got := p.SourceLine(afterRow); got != 6 {
		t.Fatalf("the paragraph after the maths maps to line %d, want 6", got)
	}
}

// Inline maths sits inside the text line it belongs to: same row as the words,
// with its ink on the line's baseline.
func TestPageDrawsInlineMathInItsLine(t *testing.T) {
	p, f, pal := testPage(t, "Energy $E=mc^2$ is mass, and prices $5 and $6 stay text.\n", 70)

	bs := boxes(p)
	if len(bs) != 1 {
		t.Fatalf("inline maths boxes = %d, want 1 (items: %+v)", len(bs), p.Items)
	}
	box := bs[0]
	wordRow := -1
	for _, it := range p.Items {
		if it.Kind == rich.ItemText && strings.Contains(it.Text, "Energy") {
			wordRow = it.Y
		}
	}
	if wordRow < 0 {
		t.Fatal("the paragraph is not drawn")
	}
	if box.Y != wordRow {
		t.Fatalf("inline maths row = %d, want the line's row %d", box.Y, wordRow)
	}
	if n := inkIn(p, f, pal, box.X, box.Y, box.W, box.H); n == 0 {
		t.Fatal("the inline maths box painted no ink")
	}
	// Prices are text, never maths, and the source is not drawn.
	got := textOf(p)
	for _, want := range []string{"$5 and $6", "Energy"} {
		if !strings.Contains(got, want) {
			t.Fatalf("page text %q does not contain %q", got, want)
		}
	}
	if strings.Contains(got, "E=mc") {
		t.Fatalf("the inline maths source was drawn as text: %q", got)
	}
}

// A paragraph whose maths cannot be laid out still shows its source: the pane
// is never blank and no marker character reaches the screen.
func TestPageWithoutAnEngineDrawsTheSource(t *testing.T) {
	f := testFonts(t)
	pal := rich.PaletteFromTheme(theme.DefaultTheme())
	doc := rich.Parse([]byte("Energy $E=mc^2$ here.\n\n$$x^2$$\n"))
	p := rich.Layout(doc, rich.LayoutOptions{Width: 60, Fonts: f, Palette: pal})
	if len(boxes(p)) != 0 {
		t.Fatal("a page without an engine drew a box")
	}
	got := textOf(p)
	if !strings.Contains(got, "$E=mc^2$") {
		t.Fatalf("the fallback does not show the maths source: %q", got)
	}
	for _, r := range got {
		if r >= 0xE000 && r <= 0xF8FF {
			t.Fatalf("a marker character reached the page text: %q", got)
		}
	}
}

// The page hands the engine the formula exactly as the note wrote it: a
// display block keeps its braces, underscores and backslashes. Only the
// engine decides how a group is drawn, so a broken subscript can never be the
// Markdown layer eating characters.
func TestPageHandsTheEngineTheExactFormula(t *testing.T) {
	f := testFonts(t)
	pal := rich.PaletteFromTheme(theme.DefaultTheme())
	doc := rich.Parse([]byte("Intro with $\\eta$ inline.\n\n$$\n\\theta_{t+1} = \\theta_t - \\eta \\frac{\\partial L}{\\partial \\theta}\n$$\n"))
	var blockSrc, inlineSrc []string
	page := rich.Layout(doc, rich.LayoutOptions{
		Width: 70, Fonts: f, Palette: pal,
		Math: func(b rich.Block) *rich.Box {
			blockSrc = append(blockSrc, strings.Join(b.Code, "\n"))
			return &rich.Box{W: 4, H: 1, Draw: func(dst draw.Image, x, y int) {}}
		},
		MathInline: func(src string, sizePx float64) *rich.InlineBox {
			inlineSrc = append(inlineSrc, src)
			return &rich.InlineBox{WidthPx: 12, DepthPx: 2, Draw: func(dst draw.Image, x, baselineY int) {}}
		},
	})
	if len(blockSrc) != 1 {
		t.Fatalf("engine received %d display formulas, want 1", len(blockSrc))
	}
	if len(inlineSrc) != 1 || inlineSrc[0] != `\eta` {
		t.Fatalf("engine received inline formulas %q, want one \\eta (the note has $\\eta$)", inlineSrc)
	}
	want := `\theta_{t+1} = \theta_t - \eta \frac{\partial L}{\partial \theta}`
	if blockSrc[0] != want {
		t.Fatalf("engine source = %q, want %q", blockSrc[0], want)
	}
	if page.Height <= 0 {
		t.Fatal("page is empty")
	}
}

// A bold run claims the width its bold face draws: the space after it must
// start past the ink, or "learning rate" collapses into "learningrate".
func TestPageKeepsTheSpaceAfterABoldRun(t *testing.T) {
	f := testFonts(t)
	pal := rich.PaletteFromTheme(theme.DefaultTheme())
	doc := rich.Parse([]byte("The update uses the **learning rate** here.\n"))
	p := Page(doc, 70, f, pal, rich.ImageOptions{})
	bold, ok := f.BoldFace(f.BodyPx)
	if !ok {
		t.Fatal("the test stack has no bold face")
	}
	boldW := 0
	for _, r := range "learning" {
		adv, _ := bold.GlyphAdvance(r)
		boldW += adv.Round()
	}
	var learning, space, rate *rich.Item
	for i := range p.Items {
		it := &p.Items[i]
		if it.Kind != rich.ItemText {
			continue
		}
		switch it.Text {
		case "learning":
			learning = it
		case "rate":
			rate = it
		}
		if it.Text == " " && learning != nil && space == nil {
			space = it
		}
	}
	if learning == nil || space == nil || rate == nil {
		t.Fatalf("runs missing: learning %v space %v rate %v", learning, space, rate)
	}
	inkEnd := learning.X*f.CellW + learning.XOffPx + boldW
	spaceStart := space.X*f.CellW + space.XOffPx
	if spaceStart < inkEnd {
		t.Fatalf("space starts at %d px, inside the bold ink ending at %d", spaceStart, inkEnd)
	}
	rateStart := rate.X*f.CellW + rate.XOffPx
	if gap := rateStart - spaceStart; gap <= 0 {
		t.Fatalf("\"rate\" starts at the space's own pixel (gap %d)", gap)
	}
	// The rendered gap is clean: no ink of the bold word reaches into it.
	img := (&rich.Raster{Page: p, Fonts: f, Palette: pal}).Tile(learning.Y, 1)
	for x := spaceStart; x < rateStart; x++ {
		for y := 0; y < f.CellH; y++ {
			if img.RGBAAt(x, y) != pal.Bg {
				t.Fatalf("ink at x=%d in the gap %d..%d", x, spaceStart, rateStart)
			}
		}
	}
}

// Every text run claims the cells it draws at: inline code at 0.85 in the mono
// face, bold in the bold face, italic in its own. A width sweep over a note
// that mixes all of them must never leave a run wider than its cells.
func TestPageRunsFitTheirCellsAcrossWidths(t *testing.T) {
	f := testFonts(t)
	pal := rich.PaletteFromTheme(theme.DefaultTheme())
	src := "Words with `inline code that is fairly long` and **bold words here** and *italic text* too, plus some plain words.\n"
	for width := 20; width <= 80; width++ {
		p := Page(rich.Parse([]byte(src)), width, f, pal, rich.ImageOptions{})
		for _, it := range p.Items {
			if it.Kind != rich.ItemText {
				continue
			}
			ink := it.XOffPx + f.MeasureStyled(it.Text, it.SizePx, it.Mono, it.Bold, it.Italic)
			if limit := it.W * f.CellW; ink > limit {
				t.Fatalf("width %d: run %q draws %d px into %d claimed (%d cells)", width, it.Text, ink, limit, it.W)
			}
		}
	}
}

const (
	flowSource = "flowchart LR\nA[Start] --> B{Choice}\nB -->|yes| C[(Done)]"
	seqSource  = "sequenceDiagram\nparticipant A\nparticipant B\nA->>B: hello\nB-->>A: reply"
	pieSource  = "pie\ntitle Share\n\"A\" : 60\n\"B\" : 40"
)

func mermaidNote(sources ...string) string {
	var b strings.Builder
	b.WriteString("# Diagrams\n\nText before.\n\n")
	for _, src := range sources {
		b.WriteString("```mermaid\n" + src + "\n```\n\n")
	}
	b.WriteString("Text after.\n")
	return b.String()
}

// The app and headless mode draw Mermaid blocks with the native engine: one
// box per block, painted, and the blocks around them keep their place.
func TestPageDrawsMermaidDiagrams(t *testing.T) {
	p, f, pal := testPage(t, mermaidNote(flowSource, seqSource, pieSource), 100)
	if len(p.Diagrams) != 3 {
		t.Fatalf("diagrams on the page = %d, want 3", len(p.Diagrams))
	}
	boxes := 0
	for _, it := range p.Items {
		if it.Kind != rich.ItemBox {
			continue
		}
		boxes++
		if it.W < 4 || it.H < 2 {
			t.Fatalf("diagram box = %dx%d cells, want a sized picture", it.W, it.H)
		}
		if n := inkIn(p, f, pal, it.X, it.Y, it.W, it.H); n == 0 {
			t.Fatalf("diagram box at row %d painted no ink", it.Y)
		}
	}
	if boxes != 3 {
		t.Fatalf("boxes = %d, want 3", boxes)
	}
	// Each diagram remembers its rows and source, so a click can zoom it.
	for _, d := range p.Diagrams {
		if d.Row <= 0 || d.Rows < 2 || len(d.Block.Code) == 0 {
			t.Fatalf("diagram record = %+v, want rows and its source", d)
		}
	}
	after := -1
	for _, it := range p.Items {
		if it.Kind == rich.ItemText && strings.Contains(it.Text, "after") {
			after = it.Y
		}
	}
	if after <= p.Diagrams[2].Row {
		t.Fatalf("the text after the diagrams is at row %d, diagrams end at %d", after, p.Diagrams[2].Row+p.Diagrams[2].Rows)
	}
}

// A block the engine cannot parse keeps its source and the reason, never a
// silently wrong picture, and never nothing.
func TestPageKeepsTheSourceOfAnUnsupportedDiagram(t *testing.T) {
	src := "sequenceDiagram\nparticipant A\nA ?!> B: unsupported arrow"
	p, f, pal := testPage(t, mermaidNote(src), 100)
	if len(p.Diagrams) != 1 {
		t.Fatalf("diagrams = %d, want 1", len(p.Diagrams))
	}
	d := p.Diagrams[0]
	if d.Rows < 4 {
		t.Fatalf("fallback box rows = %d, want the source panel", d.Rows)
	}
	if n := inkIn(p, f, pal, 2, d.Row, 70, d.Rows); n == 0 {
		t.Fatal("the fallback panel painted nothing")
	}
}

// Zoom lays one diagram out to fill a pane: it fits inside it, keeps its
// aspect, is centred, and covers most of the pane.
func TestZoomFillsThePane(t *testing.T) {
	f, pal := testFonts(t), rich.PaletteFromTheme(theme.DefaultTheme())
	e := New(f, pal)
	doc := rich.Parse([]byte(mermaidNote(flowSource)))
	page := e.Page(doc, 100, rich.ImageOptions{})
	if len(page.Diagrams) != 1 {
		t.Fatalf("diagrams = %d, want 1", len(page.Diagrams))
	}
	const w, h = 60, 30
	zoom, ok := e.Zoom(page.Diagrams[0].Block, w, h)
	if !ok {
		t.Fatal("Zoom refused a diagram it just drew")
	}
	if zoom.Width != w || zoom.Height != h {
		t.Fatalf("zoom page = %dx%d, want the pane %dx%d", zoom.Width, zoom.Height, w, h)
	}
	var box *rich.Item
	for i := range zoom.Items {
		if zoom.Items[i].Kind == rich.ItemBox {
			box = &zoom.Items[i]
		}
	}
	if box == nil {
		t.Fatal("the zoom page has no diagram")
	}
	if box.W > w || box.H > h {
		t.Fatalf("zoomed diagram %dx%d does not fit %dx%d", box.W, box.H, w, h)
	}
	if box.W < w/2 && box.H < h/2 {
		t.Fatalf("zoomed diagram is %dx%d in a %dx%d pane: it did not fill it", box.W, box.H, w, h)
	}
	if box.X < 0 || box.Y < 0 || box.X+box.W > w || box.Y+box.H > h {
		t.Fatalf("zoomed diagram at %d,%d is not centred inside the pane", box.X, box.Y)
	}
	if n := inkIn(zoom, f, pal, box.X, box.Y, box.W, box.H); n == 0 {
		t.Fatal("the zoomed diagram painted no ink")
	}
	// A development aid: RICH_ZOOM_PNG writes the pane to be looked at.
	if out := os.Getenv("RICH_ZOOM_PNG"); out != "" {
		img := (&rich.Raster{Page: zoom, Fonts: f, Palette: pal}).Tile(0, h)
		if err := os.WriteFile(out, rich.EncodePNG(img), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%dx%d, diagram %dx%d at %d,%d)", out, img.Bounds().Dx(), img.Bounds().Dy(), box.W, box.H, box.X, box.Y)
	}
}

// A pane too small for any text scale still gets the whole picture: the
// bitmap is fitted rather than left half outside the pane.
func TestZoomFitsADiagramThePaneCannotHold(t *testing.T) {
	f, pal := testFonts(t), rich.PaletteFromTheme(theme.DefaultTheme())
	e := New(f, pal)
	long := "sequenceDiagram\nparticipant A\nparticipant B\nparticipant C\nparticipant D\n" +
		"A->>B: hello\nB-->>A: reply\nC->>D: ping\nD-->>C: pong\nA->>D: done"
	doc := rich.Parse([]byte(mermaidNote(long)))
	page := e.Page(doc, 100, rich.ImageOptions{})
	if len(page.Diagrams) != 1 {
		t.Fatalf("diagrams = %d, want 1", len(page.Diagrams))
	}
	const w, h = 24, 8
	zoom, ok := e.Zoom(page.Diagrams[0].Block, w, h)
	if !ok {
		t.Fatal("Zoom refused the sequence diagram")
	}
	var box *rich.Item
	for i := range zoom.Items {
		box = &zoom.Items[i]
	}
	if box.W > w || box.H > h {
		t.Fatalf("fitted diagram %dx%d does not fit %dx%d", box.W, box.H, w, h)
	}
	if n := inkIn(zoom, f, pal, box.X, box.Y, box.W, box.H); n == 0 {
		t.Fatal("the fitted diagram painted no ink")
	}
}
