package math

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestSymbolCommandsHaveGlyphs keeps the symbol table honest: every command
// must resolve to a glyph the bundled face actually has, or a formula would
// silently draw a hole.
func TestSymbolCommandsHaveGlyphs(t *testing.T) {
	var missing []string
	for name, d := range symbols {
		if !hasGlyph(d.r) {
			missing = append(missing, name+" -> "+string(d.r))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("%d commands have no glyph in the bundled face:\n%s", len(missing), strings.Join(missing, "\n"))
	}
}

func TestParseClassesAndScripts(t *testing.T) {
	e, err := Parse(`a+b=x_1^2`)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Nodes) != 5 {
		t.Fatalf("got %d nodes, want 5", len(e.Nodes))
	}
	classes := []Class{}
	for _, n := range e.Nodes[:4] {
		s, ok := n.(*Sym)
		if !ok {
			t.Fatalf("node %T is not a symbol", n)
		}
		classes = append(classes, s.Class)
	}
	want := []Class{Ord, Bin, Ord, Rel}
	for i := range want {
		if classes[i] != want[i] {
			t.Fatalf("class %d = %v, want %v", i, classes[i], want[i])
		}
	}
	script, ok := e.Nodes[4].(*Script)
	if !ok {
		t.Fatalf("x_1^2 is %T, want *Script", e.Nodes[4])
	}
	if len(script.Sub) != 1 || len(script.Sup) != 1 {
		t.Fatalf("scripts = %d sub, %d sup", len(script.Sub), len(script.Sup))
	}
}

func TestParseStructure(t *testing.T) {
	cases := []struct {
		src  string
		want string // %T of the first node
	}{
		{`\frac{1}{2}`, "*math.Frac"},
		{`\binom{n}{k}`, "*math.Fenced"},
		{`\sqrt[3]{x}`, "*math.Sqrt"},
		{`\left(\frac{a}{b}\right)`, "*math.Fenced"},
		{`\begin{pmatrix}a&b\\c&d\end{pmatrix}`, "*math.Env"},
		{`\hat{x}`, "*math.Accent"},
		{`\overrightarrow{AB}`, "*math.Accent"},
		{`\sum_{i=1}^{n} i`, "*math.Script"},
		{`\lim_{x\to0} f(x)`, "*math.Script"},
		{`\overset{a}{b}`, "*math.Stack"},
		{`\xrightarrow[below]{above}`, "*math.XArrow"},
		{`\not=`, "*math.Not"},
		{`\mathbf{A}`, "*math.Fonted"},
		{`\text{where B is a batch}`, "*math.Text"},
		{`\operatorname{softmax}`, "*math.FnName"},
		{`\displaystyle x`, "*math.Styled"},
	}
	for _, c := range cases {
		e, err := Parse(c.src)
		if err != nil {
			t.Fatalf("%s: %v", c.src, err)
		}
		if len(e.Nodes) == 0 {
			t.Fatalf("%s: no nodes", c.src)
		}
		if got := fmt.Sprintf("%T", e.Nodes[0]); got != c.want {
			t.Fatalf("%s: first node %s, want %s", c.src, got, c.want)
		}
	}
}

func TestParseUnknownCommandIsLiteral(t *testing.T) {
	e, err := Parse(`\nosuchcommand{x}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Unknown) != 1 || e.Unknown[0] != "nosuchcommand" {
		t.Fatalf("unknown = %v", e.Unknown)
	}
	if _, ok := e.Nodes[0].(*RawText); !ok {
		t.Fatalf("unknown command is %T, want literal text", e.Nodes[0])
	}
}

func TestParseMalformedErrors(t *testing.T) {
	for _, src := range []string{
		`\frac{1}{2`, `{a`, `\left(a`, `a}+b`,
		`\begin{pmatrix}a&b\end{array}`,
		`\begin{pmatrix}a&b`,
	} {
		if _, err := Parse(src); err == nil {
			t.Fatalf("%q parsed without error", src)
		}
	}
}

func TestParseMinusUsesUnicodeMinus(t *testing.T) {
	e, err := Parse(`\theta_t - \eta`)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Nodes) < 2 {
		t.Fatalf("got %d nodes, want at least 2", len(e.Nodes))
	}
	sym, ok := e.Nodes[1].(*Sym)
	if !ok {
		t.Fatalf("second node is %T, want *Sym", e.Nodes[1])
	}
	if sym.R != '−' || sym.Class != Bin {
		t.Fatalf("minus = %q (U+%04X, class %v), want U+2212 Bin", sym.R, sym.R, sym.Class)
	}
}

// TestParseArgumentsHoldContents pins what the tree carries, not only its Go
// type: the values are what layout and drawing consume.
func TestParseArgumentsHoldContents(t *testing.T) {
	e, err := Parse(`\text{where B is a batch}`)
	if err != nil {
		t.Fatal(err)
	}
	txt, ok := e.Nodes[0].(*Text)
	if !ok || txt.S != "where B is a batch" {
		t.Fatalf("text = %#v, want the argument's own spaces", e.Nodes[0])
	}

	e, err = Parse(`\sqrt[3]{x}`)
	if err != nil {
		t.Fatal(err)
	}
	root, ok := e.Nodes[0].(*Sqrt)
	if !ok || len(root.Index) != 1 || len(root.Body) != 1 {
		t.Fatalf("sqrt = %#v, want index and body", e.Nodes[0])
	}

	e, err = Parse(`\frac{a+b}{c}`)
	if err != nil {
		t.Fatal(err)
	}
	fr, ok := e.Nodes[0].(*Frac)
	if !ok || len(fr.Num) != 3 || len(fr.Den) != 1 {
		t.Fatalf("frac = %#v, want three numerator nodes", e.Nodes[0])
	}

	e, err = Parse(`\begin{pmatrix}a&b\\c&d\end{pmatrix}`)
	if err != nil {
		t.Fatal(err)
	}
	env, ok := e.Nodes[0].(*Env)
	if !ok || len(env.Rows) != 2 || len(env.Rows[0]) != 2 || len(env.Rows[1]) != 2 {
		t.Fatalf("matrix = %#v, want a 2x2 grid", e.Nodes[0])
	}

	// A trailing & leaves a pending row that must still be kept.
	e, err = Parse(`\begin{pmatrix}a&\end{pmatrix}`)
	if err != nil {
		t.Fatal(err)
	}
	if env, ok := e.Nodes[0].(*Env); !ok || len(env.Rows) != 1 || len(env.Rows[0]) != 2 {
		t.Fatalf("trailing & = %#v, want one row with two cells", e.Nodes[0])
	}

	// A row break at top level becomes the synthetic gather grid.
	e, err = Parse(`a = b\\c = d`)
	if err != nil {
		t.Fatal(err)
	}
	g, ok := e.Nodes[0].(*Env)
	if !ok || g.Name != "gather" || len(g.Rows) != 2 {
		t.Fatalf("multi-line = %#v, want two gather rows", e.Nodes[0])
	}
}

// TestParseFixes locks the behaviour the first review moved: a leading script
// keeps its body, \operatorname* asks for limits, a plain \sum takes them,
// and \bigm is a relation.
func TestParseFixes(t *testing.T) {
	e, err := Parse(`^2`)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Nodes) != 1 || e.Nodes[0] == nil {
		t.Fatalf("leading script = %#v, want one real node", e.Nodes)
	}
	if b := layoutSource(t, `^2`); b.w <= 0 {
		t.Fatal("leading script drew nothing")
	}

	e, err = Parse(`\operatorname*{argmax}_x f`)
	if err != nil {
		t.Fatal(err)
	}
	sc, ok := e.Nodes[0].(*Script)
	if !ok || !sc.Limits {
		t.Fatalf("operatorname* = %#v, want a limits script", e.Nodes[0])
	}

	e, err = Parse(`\operatorname{softmax}_x f`)
	if err != nil {
		t.Fatal(err)
	}
	if sc, ok := e.Nodes[0].(*Script); !ok || sc.Limits {
		t.Fatalf("plain operatorname = %#v, want side scripts", e.Nodes[0])
	}

	e, err = Parse(`\sum_{i=1}^{n} i`)
	if err != nil {
		t.Fatal(err)
	}
	sc, ok = e.Nodes[0].(*Script)
	if !ok || !sc.Limits {
		t.Fatalf("sum = %#v, want a limits script", e.Nodes[0])
	}
	if len(sc.Base) != 1 {
		t.Fatalf("sum base = %d nodes", len(sc.Base))
	}
	if _, ok := sc.Base[0].(*BigOp); !ok {
		t.Fatalf("sum base = %T, want *BigOp", sc.Base[0])
	}

	e, err = Parse(`a \bigm| b`)
	if err != nil {
		t.Fatal(err)
	}
	mid, ok := e.Nodes[1].(*Sym)
	if !ok || mid.Class != Rel {
		t.Fatalf("bigm = %#v, want a relation", e.Nodes[1])
	}

	e, err = Parse(``)
	if err != nil || len(e.Nodes) != 0 {
		t.Fatalf("empty formula = %#v, %v", e, err)
	}

	// A declaration with no body leaves no node behind.
	e, err = Parse(`\limits`)
	if err != nil || len(e.Nodes) != 0 {
		t.Fatalf(`\limits alone = %#v, %v`, e, err)
	}

	// TeX's interword space is one third of an em, whether written ~ or \ .
	for _, src := range []string{`a~b`, `a\ b`, `a\space b`} {
		e, err = Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		sp, ok := e.Nodes[1].(*Space)
		if !ok || sp.Em != interwordSpace {
			t.Fatalf("%s space = %#v, want %v em", src, e.Nodes[1], interwordSpace)
		}
	}
}
