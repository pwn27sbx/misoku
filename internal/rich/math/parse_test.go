package math

import (
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
		{`\binom{n}{k}`, "*math.Fenced"},
	}
	for _, c := range cases {
		e, err := Parse(c.src)
		if err != nil {
			t.Fatalf("%s: %v", c.src, err)
		}
		if len(e.Nodes) == 0 {
			t.Fatalf("%s: no nodes", c.src)
		}
		if got := typeName(e.Nodes[0]); got != c.want {
			t.Fatalf("%s: first node %s, want %s", c.src, got, c.want)
		}
	}
}

func typeName(n Node) string {
	switch n.(type) {
	case *Sym:
		return "*math.Sym"
	case *Space:
		return "*math.Space"
	case *Text:
		return "*math.Text"
	case *Fonted:
		return "*math.Fonted"
	case *Group:
		return "*math.Group"
	case *Frac:
		return "*math.Frac"
	case *Sqrt:
		return "*math.Sqrt"
	case *Script:
		return "*math.Script"
	case *Fenced:
		return "*math.Fenced"
	case *BigOp:
		return "*math.BigOp"
	case *Accent:
		return "*math.Accent"
	case *Env:
		return "*math.Env"
	case *Styled:
		return "*math.Styled"
	case *Stack:
		return "*math.Stack"
	case *XArrow:
		return "*math.XArrow"
	case *Not:
		return "*math.Not"
	case *FnName:
		return "*math.FnName"
	case *ClassWrap:
		return "*math.ClassWrap"
	case *RawText:
		return "*math.RawText"
	}
	return "?"
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
	for _, src := range []string{`\frac{1}{2`, `{a`, `\left(a`} {
		if _, err := Parse(src); err == nil {
			t.Fatalf("%q parsed without error", src)
		}
	}
}
