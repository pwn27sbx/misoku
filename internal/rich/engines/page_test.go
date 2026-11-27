package engines

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/rich"
	"github.com/pwn27sbx/misoku/internal/theme"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// testFonts is the page's font stack: the Go fonts, so nothing depends on the
// machine's installed faces.
func testFonts(t *testing.T) *rich.Fonts {
	t.Helper()
	f, err := rich.NewFonts(goregular.TTF, gomono.TTF, rich.FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
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
