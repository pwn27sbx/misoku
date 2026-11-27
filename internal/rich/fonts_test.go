package rich

import (
	"testing"

	"golang.org/x/image/font"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// testFonts is the page's font stack for tests: the Go fonts, so golden pixels
// do not depend on what the machine has installed.
func testFonts(t testing.TB) *Fonts {
	t.Helper()
	f, err := NewFonts(goregular.TTF, gomono.TTF, FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestFontsMeasureAndWrap(t *testing.T) {
	f := testFonts(t)
	if got := f.Measure("hello", f.BodyPx, false); got <= 0 {
		t.Fatalf("measure = %v, want positive", got)
	}
	// Wrapping never overflows the requested width.
	const widthPx = 120
	lines := f.Wrap("the quick brown fox jumps over the lazy dog and keeps going", widthPx, f.BodyPx, false)
	if len(lines) < 2 {
		t.Fatalf("wrap returned %d lines, want several", len(lines))
	}
	for _, l := range lines {
		if got := int(f.Measure(l, f.BodyPx, false)) / 64; got > widthPx {
			t.Fatalf("line %q is %d px wide, over %d", l, got, widthPx)
		}
	}
	// A word wider than the line is split, never overflowed.
	long := f.Wrap("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 40, f.BodyPx, false)
	for _, l := range long {
		if got := int(f.Measure(l, f.BodyPx, false)) / 64; got > 40 {
			t.Fatalf("split line %q is %d px, over 40", l, got)
		}
	}
}

func TestFontsLinesHaveAHeuristicHeight(t *testing.T) {
	f := testFonts(t)
	if h := f.LineHeightPx(f.BodyPx, false); h < 10 || h > 40 {
		t.Fatalf("line height = %v px at 16 px", h)
	}
	if a := f.AscentPx(f.BodyPx, false); a <= 0 || a > f.LineHeightPx(f.BodyPx, false) {
		t.Fatalf("ascent = %v, line height = %v", a, f.LineHeightPx(f.BodyPx, false))
	}
}

// MeasureStyled agrees with Measure when no styled face is installed: the run
// is drawn with the body face, and synthetic bold only adds its extra strike.
func TestMeasureStyledWithoutStyledFaces(t *testing.T) {
	f, err := NewFonts(goregular.TTF, gomono.TTF, FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"learning rate", "x", ""} {
		plain := int(f.Measure(s, f.BodyPx, false)) / 64
		if got := f.MeasureStyled(s, f.BodyPx, false, false, false); got != plain {
			t.Fatalf("plain %q = %d, want %d", s, got, plain)
		}
		bold := plain
		if bold > 0 {
			bold++ // synthetic bold strikes one pixel over
		}
		if got := f.MeasureStyled(s, f.BodyPx, false, true, false); got != bold {
			t.Fatalf("synthetic bold %q = %d, want %d", s, got, bold)
		}
	}
}

// A styled face that lacks a glyph keeps the shaped advance, the same one the
// raster moves the pen by, so the claim never falls short of what is drawn.
// Bold, italic and inline code are all covered: every styled run the page can
// draw takes this path.
func TestMeasureStyledMatchesTheDrawnAdvance(t *testing.T) {
	f, err := NewFontsWithStyles(goregular.TTF, gomono.TTF, gobold.TTF, goitalic.TTF,
		FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
	if err != nil {
		t.Fatal(err)
	}
	bold, ok := f.BoldFace(f.BodyPx)
	if !ok {
		t.Fatal("the test stack has no bold face")
	}
	italic, ok := f.ItalicFace(f.BodyPx)
	if !ok {
		t.Fatal("the test stack has no italic face")
	}
	type styled struct {
		name               string
		face               font.Face
		mono, bold, italic bool
	}
	cases := []styled{
		{"bold", bold, false, true, false},
		{"italic", italic, false, false, true},
		{"code", nil, true, false, false},
	}
	for _, c := range cases {
		for _, s := range []string{"learning rate", "école", "日本語", "x"} {
			var want int
			if c.face == nil {
				// Unstyled runs keep the layout's long-standing metric: the
				// shaped total, truncated once, exactly like Measure.
				want = int(f.Measure(s, f.BodyPx, c.mono)) / 64
			} else {
				for _, g := range f.Shape(s, f.BodyPx, c.mono) {
					step := g.Advance.Round()
					if adv, ok := c.face.GlyphAdvance(g.R); ok {
						step = adv.Round()
					}
					want += step
				}
			}
			if got := f.MeasureStyled(s, f.BodyPx, c.mono, c.bold, c.italic); got != want {
				t.Fatalf("%s %q measured %d, drawn %d", c.name, s, got, want)
			}
		}
	}
}
