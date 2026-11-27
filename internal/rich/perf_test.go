package rich

import (
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// The plan's budget (amendment 5): the first rich paint of a 1.000-line note
// at 120×40, and a scroll frame, at 1× and 2× cell scale. These benchmarks
// measure the whole pipeline the app runs (parse, lay out, raster the screen,
// encode the PNG the terminal receives), and the guard below fails only on a
// regression far past the budget so a slow CI never blocks a change.
const (
	budgetFirstPaint = 250 * time.Millisecond
	budgetScroll     = 33 * time.Millisecond
)

// perfNote is a 1000-line note with prose, heads, lists, code and maths: the
// mix a real note has, so the layout cost is representative.
func perfNote() []byte {
	var b strings.Builder
	for lines := 0; lines < 1000; lines += 14 {
		b.WriteString("# Section\n\nA paragraph with **bold**, *italic*, `code` and a [[wiki link]] that wraps over a few words.\n\n")
		b.WriteString("- first item\n- second item with `code`\n- third item\n\n")
		b.WriteString("```go\nfunc main() { fmt.Println(\"hello\") }\n```\n\n")
		b.WriteString("Inline $E=mc^2$ and a display formula:\n\n$$\n\\theta_{t+1} = \\theta_t - \\eta \\frac{\\partial L}{\\partial \\theta}\n$$\n\n")
	}
	return []byte(b.String())
}

func perfFonts(tb testing.TB, cellW, cellH int, bodyPx float64) *Fonts {
	tb.Helper()
	f, err := NewFontsWithStyles(goregular.TTF, gomono.TTF, gobold.TTF, goitalic.TTF,
		FontConfig{CellW: cellW, CellH: cellH, BodyPx: bodyPx})
	if err != nil {
		tb.Fatal(err)
	}
	return f
}

// firstPaint is what the reader waits for: the page built and the first tile
// encoded, ready to send.
func firstPaint(b *testing.B, f *Fonts, pal Palette, doc Document) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := Layout(doc, LayoutOptions{Width: 120, Fonts: f, Palette: pal})
		img := (&Raster{Page: p, Fonts: f, Palette: pal}).Tile(0, 40)
		if len(EncodePNG(img)) == 0 {
			b.Fatal("no png")
		}
	}
}

// scrollFrame is one wheel step: the page already laid out, the next strip
// rastered and encoded.
func scrollFrame(b *testing.B, f *Fonts, pal Palette, p *Page) {
	r := &Raster{Page: p, Fonts: f, Palette: pal}
	last := max(p.Height-40, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		img := r.Tile((i*3)%last, 40)
		if len(EncodePNG(img)) == 0 {
			b.Fatal("no png")
		}
	}
}

func BenchmarkRichFirstPaint(b *testing.B) {
	pal := testPalette()
	doc := Parse(perfNote())
	b.Run("1x", func(b *testing.B) { firstPaint(b, perfFonts(b, 12, 24, 16), pal, doc) })
	b.Run("2x", func(b *testing.B) { firstPaint(b, perfFonts(b, 24, 48, 32), pal, doc) })
}

func BenchmarkRichScrollFrame(b *testing.B) {
	pal := testPalette()
	doc := Parse(perfNote())
	b.Run("1x", func(b *testing.B) {
		f := perfFonts(b, 12, 24, 16)
		scrollFrame(b, f, pal, Layout(doc, LayoutOptions{Width: 120, Fonts: f, Palette: pal}))
	})
	b.Run("2x", func(b *testing.B) {
		f := perfFonts(b, 24, 48, 32)
		scrollFrame(b, f, pal, Layout(doc, LayoutOptions{Width: 120, Fonts: f, Palette: pal}))
	})
}

// TestRichPaintBudgets is the guard: it measures the same pipeline the
// benchmarks do and fails only when a change is far past the plan's budget.
// The exact numbers live in docs/rich-view.md and are re-measured by hand.
func TestRichPaintBudgets(t *testing.T) {
	if testing.Short() || (os.Getenv("CI") != "" && os.Getenv("MISOKU_PERF") == "") {
		t.Skip("timing guard runs outside CI or with MISOKU_PERF=1")
	}
	pal := testPalette()
	doc := Parse(perfNote())
	for _, tc := range []struct {
		name string
		f    *Fonts
	}{{"1x", perfFonts(t, 12, 24, 16)}, {"2x", perfFonts(t, 24, 48, 32)}} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			p := Layout(doc, LayoutOptions{Width: 120, Fonts: tc.f, Palette: pal})
			img := (&Raster{Page: p, Fonts: tc.f, Palette: pal}).Tile(0, 40)
			if len(EncodePNG(img)) == 0 {
				t.Fatal("no png")
			}
			paint := time.Since(start)
			t.Logf("first paint %s (budget %s)", paint.Round(time.Millisecond), budgetFirstPaint)
			if paint > 6*budgetFirstPaint {
				t.Fatalf("first paint %s is far past the %s budget", paint.Round(time.Millisecond), budgetFirstPaint)
			}

			r := &Raster{Page: p, Fonts: tc.f, Palette: pal}
			worst, last := time.Duration(0), max(p.Height-40, 1)
			for i := 0; i < 20; i++ {
				start := time.Now()
				EncodePNG(r.Tile((i*7)%last, 40))
				if d := time.Since(start); d > worst {
					worst = d
				}
			}
			t.Logf("scroll frame p100 %s (budget %s)", worst.Round(time.Millisecond), budgetScroll)
			if worst > 6*budgetScroll {
				t.Fatalf("scroll frame %s is far past the %s budget", worst.Round(time.Millisecond), budgetScroll)
			}
		})
	}
}
