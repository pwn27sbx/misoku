package math

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestReducedFixturesParse is the real-document gate: formulas reduced from
// the fixtures mddog tests with must parse and use only known constructs.
func TestReducedFixturesParse(t *testing.T) {
	dir := filepath.Join("testdata", "reduced")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var unknown = map[string][]string{}
	seen := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tex") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		seen++
		expr, err := Parse(strings.TrimSpace(string(b)))
		if err != nil {
			t.Errorf("%s: %v", e.Name(), err)
			continue
		}
		for _, u := range expr.Unknown {
			unknown["\\"+u] = append(unknown["\\"+u], e.Name())
		}
	}
	if seen < 100 {
		t.Fatalf("only %d reduced fixtures found; the corpus is too thin", seen)
	}
	if len(unknown) > 0 {
		var lines []string
		for cmd, files := range unknown {
			lines = append(lines, cmd+" ("+strings.Join(files, ", ")+")")
		}
		sort.Strings(lines)
		t.Fatalf("%d unknown commands in the reduced corpus:\n%s", len(unknown), strings.Join(lines, "\n"))
	}
}

// TestReducedFixturesLayOut runs the reduced real-document corpus through the
// whole pipeline: parse, lay out and rasterise. A real formula that cannot be
// drawn is a parity gap, not a silent hole.
func TestReducedFixturesLayOut(t *testing.T) {
	dir := filepath.Join("testdata", "reduced")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := testConfig()
	var failed []string
	var count int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tex") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		count++
		src := strings.TrimSpace(string(b))
		expr, err := Parse(src)
		if err != nil {
			failed = append(failed, e.Name()+": "+err.Error())
			continue
		}
		if len(expr.Unknown) > 0 {
			failed = append(failed, e.Name()+": unknown "+strings.Join(expr.Unknown, ","))
			continue
		}
		l := &layouter{base: cfg.SizePx}
		laid := l.list(expr.Nodes, cfg.SizePx, styleDisplay, 0)
		if laid.w <= 0 || laid.h+laid.d <= 0 {
			failed = append(failed, e.Name()+": empty layout")
			continue
		}
		box := Box([]string{src}, 900, cfg)
		if box == nil || box.W < 1 || box.H < 1 {
			failed = append(failed, e.Name()+": no box")
		}
	}
	if count < 100 {
		t.Fatalf("only %d fixtures read", count)
	}
	if len(failed) > 0 {
		sort.Strings(failed)
		t.Fatalf("%d of %d reduced formulas failed:\n%s", len(failed), count, strings.Join(failed, "\n"))
	}
	t.Logf("%d reduced formulas laid out", count)
}

// TestAccentGlyphsExist keeps the accent table honest, like the symbol table.
func TestAccentGlyphsExist(t *testing.T) {
	for kind, g := range accentGlyph {
		if !hasGlyph(g) {
			t.Fatalf("accent %d uses %q, which the bundled face lacks", kind, string(g))
		}
	}
	for _, g := range []rune("√⏞⏟") {
		if !hasGlyph(g) {
			t.Fatalf("glyph %q is missing", string(g))
		}
	}
}

// TestMathBoxFitsWidth proves a formula wider than the page is scaled down
// instead of overflowing the tile.
func TestMathBoxFitsWidth(t *testing.T) {
	cfg := testConfig()
	box := Box([]string{`\frac{\partial L}{\partial W_j} = -\frac{1}{n}\sum_{i=0}^{n-1} X_{ij}`}, 120, cfg)
	if box == nil {
		t.Fatal("no box")
	}
	if got := box.W * cfg.CellW; got > 120+cfg.CellW {
		t.Fatalf("box is %d px wide, page allows 120", got)
	}
}
