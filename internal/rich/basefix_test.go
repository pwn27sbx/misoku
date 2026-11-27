package rich

import (
	"strings"
	"testing"

	"image"
	"image/color"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// testFontsWithBold loads the Go fonts plus Go Bold, so bold text can be
// pinned without a system font.
func testFontsWithBold(t testing.TB) *Fonts {
	t.Helper()
	f, err := NewFontsWithBold(goregular.TTF, gomono.TTF, gobold.TTF, FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// TestRasterDrawsHeadingsAtTheirSize is the RED test for headings drawn with
// the body face: a heading item is twice the body size and must draw about
// twice as tall.
func TestRasterDrawsHeadingsAtTheirSize(t *testing.T) {
	f := testFontsWithBold(t)
	pal := testPalette()
	render := func(size float64) inkCount {
		p := &Page{Width: 20, Height: 4, Items: []Item{{
			Kind: ItemText, X: 1, Y: 0, W: 18, H: 2, Text: "Hg", SizePx: size,
			Color: color.RGBA{255, 255, 255, 255}, BaselinePx: 20,
		}}}
		img := (&Raster{Page: p, Fonts: f, Palette: pal}).Tile(0, 4)
		return countInk(img, pal.Bg)
	}
	body := render(f.BodyPx)
	head := render(f.BodyPx * 2)
	if head.rows < body.rows*3/2 || head.pixels < body.pixels*3/2 {
		t.Fatalf("heading ink = %+v, body ink = %+v: the heading drew at body size", head, body)
	}
}

// TestRasterDrawsBoldHeavier is the RED test for **bold** drawn as body text.
func TestRasterDrawsBoldHeavier(t *testing.T) {
	f := testFontsWithBold(t)
	pal := testPalette()
	render := func(bold bool) inkCount {
		p := &Page{Width: 20, Height: 3, Items: []Item{{
			Kind: ItemText, X: 1, Y: 0, W: 18, H: 1, Text: "Bold", SizePx: f.BodyPx, Bold: bold,
			Color: color.RGBA{255, 255, 255, 255}, BaselinePx: 16,
		}}}
		return countInk((&Raster{Page: p, Fonts: f, Palette: pal}).Tile(0, 3), pal.Bg)
	}
	plain, heavy := render(false), render(true)
	if heavy.pixels <= plain.pixels {
		t.Fatalf("bold ink %d is not heavier than plain %d", heavy.pixels, plain.pixels)
	}
}

// TestCodeLinesDropCRLF is the RED test for the tofu box drawn after code.
func TestCodeLinesDropCRLF(t *testing.T) {
	doc := Parse([]byte("```go\r\nx := 1\r\n\r\n```\r\n"))
	if got := doc.Blocks[0].Code[0]; got != "x := 1" {
		t.Fatalf("code line = %q, want no trailing terminator", got)
	}
}

// TestWikiLinkIsStyledAsALink is the RED test for wiki links drawn as body
// text: the alias run must carry the theme's link colour.
func TestWikiLinkIsStyledAsALink(t *testing.T) {
	f := testFontsWithBold(t)
	pal := testPalette()
	doc := Parse([]byte("See [[Docker|the guide]] now.\n"))
	p := Layout(doc, LayoutOptions{Width: 40, Fonts: f, Palette: pal})
	var link, plain bool
	for _, it := range p.Items {
		if it.Kind != ItemText {
			continue
		}
		if (it.Text == "the" || it.Text == "guide") && it.Color == pal.Link {
			link = true
		}
		if strings.Contains(it.Text, "now") && it.Color == pal.Fg {
			plain = true
		}
	}
	if !link || !plain {
		t.Fatalf("wiki link colour: link=%v plain=%v (items %+v)", link, plain, p.Items)
	}
}

// TestBoldSpanCoversMarkdown runs the doc side of the bold fix.
func TestBoldSpanCoversMarkdown(t *testing.T) {
	doc := Parse([]byte("Some **bold text** here.\n"))
	spans := doc.Blocks[0].Spans
	if len(spans) != 1 || !spans[0].Bold {
		t.Fatalf("spans = %+v, want one bold span", spans)
	}
	runes := []rune(doc.Blocks[0].Lines[0])
	if got := string(runes[spans[0].Start:spans[0].End]); got != "bold text" {
		t.Fatalf("bold span = %q", got)
	}
}

type inkCount struct{ rows, pixels int }

func countInk(img *image.RGBA, bg color.RGBA) inkCount {
	rows := map[int]bool{}
	pixels := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if img.RGBAAt(x, y) != bg {
				rows[y] = true
				pixels++
			}
		}
	}
	return inkCount{rows: len(rows), pixels: pixels}
}

// TestStyledRunsFitTheirCells is the RED test for glyphs dropped at a wrap:
// a run placed at a pixel offset must still have room for its whole text in
// the cells it claims.
func TestStyledRunsFitTheirCells(t *testing.T) {
	f := testFontsWithBold(t)
	doc := Parse([]byte("Some **bold text** and a [[wiki link]] in a paragraph that wraps across the page width.\n"))
	p := Layout(doc, LayoutOptions{Width: 40, Fonts: f, Palette: testPalette()})
	for _, it := range p.Items {
		if it.Kind != ItemText {
			continue
		}
		start := it.X*f.CellW + it.XOffPx
		end := start + int(f.Measure(it.Text, it.SizePx, it.Mono))/64
		if limit := it.X*f.CellW + it.W*f.CellW; end > limit {
			t.Fatalf("run %q ends at %d px, past its limit %d: its last glyph is dropped", it.Text, end, limit)
		}
	}
}

// TestStyledLineKeepsEverySpace is the RED test for words running together:
// the runs of one line, in order, must spell the source line.
func TestStyledLineKeepsEverySpace(t *testing.T) {
	f := testFontsWithBold(t)
	doc := Parse([]byte("Some **bold text** and a [[wiki link]] in a paragraph that wraps across the page width.\n"))
	p := Layout(doc, LayoutOptions{Width: 40, Fonts: f, Palette: testPalette()})
	rows := map[int]*strings.Builder{}
	for _, it := range p.Items {
		if it.Kind != ItemText || it.SizePx != f.BodyPx {
			continue
		}
		if rows[it.Y] == nil {
			rows[it.Y] = &strings.Builder{}
		}
		rows[it.Y].WriteString(it.Text)
	}
	var lines []string
	for _, b := range rows {
		lines = append(lines, b.String())
	}
	joined := strings.Join(lines, " ")
	want := "Some bold text and a wiki link in a paragraph that wraps across the page width."
	if joined != want {
		t.Fatalf("page text = %q, want %q", joined, want)
	}
}
