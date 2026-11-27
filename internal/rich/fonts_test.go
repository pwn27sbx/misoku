package rich

import (
	"testing"

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
