package math

import (
	"image/color"
	"testing"
)

func testConfig() Config {
	return Config{SizePx: 16, CellW: 12, CellH: 24, Fg: color.RGBA{255, 255, 255, 255}}
}

func TestMathMatrixColumnSpec(t *testing.T) {
	got := parseColSpec("lcr|p{2cm}")
	want := []ColAlign{AlignLeft, AlignCenter, AlignRight, AlignLeft}
	if len(got) != len(want) {
		t.Fatalf("spec parsed %d columns, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("column %d = %v, want %v", i, got[i], want[i])
		}
	}
	left := layoutSource(t, `\begin{array}{lll}aaa&b&c\\d&e&fff\end{array}`)
	right := layoutSource(t, `\begin{array}{rrr}aaa&b&c\\d&e&fff\end{array}`)
	if left.w != right.w {
		t.Fatalf("same cells, different widths: %v vs %v", left.w, right.w)
	}
	xs := func(b box) []float64 {
		var out []float64
		for _, it := range b.items {
			if it.kind == itGlyph {
				out = append(out, it.x)
			}
		}
		return out
	}
	l, r := xs(left), xs(right)
	if len(l) != len(r) || len(l) == 0 {
		t.Fatalf("glyph items: %d vs %d", len(l), len(r))
	}
	same := true
	for i := range l {
		if l[i] != r[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("left and right alignment produced the same glyph positions")
	}
}

func layoutSource(t *testing.T, src string) box {
	t.Helper()
	e, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	l := &layouter{base: 16}
	return l.list(e.Nodes, 16, styleDisplay, 0)
}

func TestMathFractionAndScriptGeometry(t *testing.T) {
	plain := layoutSource(t, `x`)
	sup := layoutSource(t, `x^2`)
	if sup.h <= plain.h {
		t.Fatalf("superscript height %v, base %v", sup.h, plain.h)
	}
	if sup.d != plain.d {
		t.Fatalf("superscript changed depth: %v, base %v", sup.d, plain.d)
	}
	frac := layoutSource(t, `\frac{a}{b}`)
	if frac.h <= plain.h || frac.d <= plain.d {
		t.Fatalf("fraction %v/%v, letter %v/%v", frac.h, frac.d, plain.h, plain.d)
	}
	// The rule sits at the maths axis, not on the baseline.
	var rules int
	for _, it := range frac.items {
		if it.kind == itRule {
			rules++
			if it.y < 0.15*16 || it.y > 0.35*16 {
				t.Fatalf("fraction bar at y=%v, want the axis near 0.25em", it.y)
			}
		}
	}
	if rules != 1 {
		t.Fatalf("fraction drew %d bars, want 1", rules)
	}
}

func TestMathDelimitersStretch(t *testing.T) {
	plain := layoutSource(t, `(x)`)
	tall := layoutSource(t, `\left(\frac{\frac{a}{b}}{\frac{c}{d}}\right)`)
	if tall.h <= plain.h {
		t.Fatalf("tall fence height %v, plain %v", tall.h, plain.h)
	}
	// The fence is centred on the axis: height and depth both grow.
	if tall.d <= plain.d {
		t.Fatalf("tall fence depth %v, plain %v", tall.d, plain.d)
	}
}

func TestMathBigOpLimits(t *testing.T) {
	display := layoutSource(t, `\sum_{i=1}^{n} x_i`)
	l := &layouter{base: 16}
	e, _ := Parse(`\sum_{i=1}^{n} x_i`)
	text := l.list(e.Nodes, 16, styleText, 0)
	if display.w >= text.w {
		t.Logf("display width %v, text width %v", display.w, text.w)
	}
	if display.h <= text.h {
		t.Fatalf("display limits above: %v, text: %v", display.h, text.h)
	}
}

func TestMathEnvironmentsLayout(t *testing.T) {
	for _, src := range []string{
		`\begin{matrix}a&b\\c&d\end{matrix}`,
		`\begin{pmatrix}a&b\\c&d\end{pmatrix}`,
		`\begin{bmatrix}a&b\\c&d\end{bmatrix}`,
		`\begin{vmatrix}a&b\\c&d\end{vmatrix}`,
		`\begin{cases}0&\text{if } x\\1&\text{else}\end{cases}`,
		`\begin{aligned}a&=b\\c&=d\end{aligned}`,
		`\left\{x \middle| x>0\right\}`, // \middle is unknown: must not crash
	} {
		b := layoutSource(t, src)
		if b.w <= 0 || b.h+b.d <= 0 {
			t.Fatalf("%s laid out as %v x %v", src, b.w, b.h+b.d)
		}
	}
}

// TeX sets a superscript or subscript in script style even in a display
// formula: display->script, never display->text. The whole group must be the
// script, so exactly one glyph of \theta_{t+1} sits at body size.
func TestDisplayScriptsAreSetInScriptStyle(t *testing.T) {
	b := layoutSource(t, `\theta_{t+1}`)
	body, script := 0, 0
	for _, it := range b.items {
		if it.kind != itGlyph {
			continue
		}
		switch {
		case it.size >= 16:
			body++
		case it.size <= 12:
			script++
		default:
			t.Fatalf("glyph %q drawn at unexpected size %v", it.r, it.size)
		}
	}
	if body != 1 || script != 3 {
		t.Fatalf("glyph sizes: %d at body, %d at script size; want 1 and 3 for theta_{t+1}", body, script)
	}
}
