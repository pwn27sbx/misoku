package termsize

import "testing"

func TestFromWinsize(t *testing.T) {
	cases := []struct {
		name                   string
		cols, rows, xpix, ypix int
		wantW, wantH           int
		wantOK                 bool
	}{
		{"ghostty 9pt", 190, 50, 1900, 950, 10, 19, true},
		{"herdr reports zeros", 190, 50, 0, 0, 0, 0, false},
		{"pixels without a grid", 0, 0, 1900, 950, 0, 0, false},
		{"implausible tiny cell", 1900, 50, 1900, 950, 1, 19, false},
		{"implausible huge cell", 10, 5, 4000, 900, 0, 0, false},
	}
	for _, c := range cases {
		w, h, ok := FromWinsize(c.cols, c.rows, c.xpix, c.ypix)
		if ok != c.wantOK || (ok && (w != c.wantW || h != c.wantH)) {
			t.Errorf("%s: got %dx%d ok=%v, want %dx%d ok=%v", c.name, w, h, ok, c.wantW, c.wantH, c.wantOK)
		}
	}
}

func TestParse(t *testing.T) {
	for _, c := range []struct {
		in   string
		w, h int
		ok   bool
	}{
		{"10x19", 10, 19, true},
		{" 12×24 ", 12, 24, true},
		{"", 0, 0, false},
		{"12", 0, 0, false},
		{"1x1", 0, 0, false},
		{"12x", 0, 0, false},
		{"100x19", 0, 0, false},
	} {
		w, h, ok := Parse(c.in)
		if ok != c.ok || (ok && (w != c.w || h != c.h)) {
			t.Errorf("Parse(%q) = %d,%d,%v; want %d,%d,%v", c.in, w, h, ok, c.w, c.h, c.ok)
		}
	}
}
