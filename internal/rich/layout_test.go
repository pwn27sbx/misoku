package rich

import (
	"testing"
	"unicode/utf8"

	"github.com/pwn27sbx/misoku/internal/theme"
)

func testPalette() Palette { return PaletteFromTheme(theme.DefaultTheme()) }

func TestLayoutHeadingScaleAndMap(t *testing.T) {
	f := testFonts(t)
	doc := Parse([]byte("# Title\n\nA paragraph with enough words to wrap onto a second line in a narrow page.\n"))
	p := Layout(doc, LayoutOptions{Width: 60, Fonts: f, Palette: testPalette()})

	var heading, para *Item
	for i := range p.Items {
		it := &p.Items[i]
		if it.Kind != ItemText {
			continue
		}
		switch it.Text {
		case "Title":
			heading = it
		case "A paragraph with enough words to wrap onto a second line in a narrow page.":
			para = it
		}
	}
	if heading == nil || para == nil {
		t.Fatalf("heading %v paragraph %v not drawn", heading, para)
	}
	if heading.SizePx != f.BodyPx*2 {
		t.Fatalf("heading size = %v, want %v", heading.SizePx, f.BodyPx*2)
	}
	if para.SizePx != f.BodyPx {
		t.Fatalf("paragraph size = %v, want %v", para.SizePx, f.BodyPx)
	}
	if p.Height <= 0 {
		t.Fatalf("page height = %d", p.Height)
	}
	// The heading's row maps back to source line 0; the paragraph starts at 2.
	if got := p.SourceLine(heading.Y); got != 0 {
		t.Fatalf("heading row maps to line %d, want 0", got)
	}
	if got := p.SourceLine(para.Y); got != 2 {
		t.Fatalf("paragraph row maps to line %d, want 2", got)
	}
	if got := p.RowOf(2); got != para.Y {
		t.Fatalf("RowOf(2) = %d, want %d", got, para.Y)
	}
}

func TestLayoutCodeListAndRule(t *testing.T) {
	f := testFonts(t)
	doc := Parse([]byte("- item one\n- item two\n\n---\n\n```go\nx := 1\ny := 2\n```\n"))
	p := Layout(doc, LayoutOptions{Width: 60, Fonts: f, Palette: testPalette()})

	var bullets, codeBG, rule int
	for _, it := range p.Items {
		switch {
		case it.Kind == ItemText && it.Text == "•":
			bullets++
		case it.Kind == ItemRect && !it.Thin && it.H > 1 && it.Color == testPalette().CodeBg:
			codeBG++
		case it.Kind == ItemRect && it.Thin:
			rule++
		}
	}
	if bullets != 2 {
		t.Fatalf("bullets = %d, want 2", bullets)
	}
	if codeBG != 1 {
		t.Fatalf("code backgrounds = %d, want 1", codeBG)
	}
	if rule < 1 {
		t.Fatalf("rules = %d, want at least the thematic break", rule)
	}
	// Code lines are drawn in monospace.
	mono := 0
	for _, it := range p.Items {
		if it.Kind == ItemText && it.Mono {
			mono++
		}
	}
	if mono < 2 {
		t.Fatalf("monospace lines = %d, want the two code lines", mono)
	}
}

func TestLayoutWrapsInsideThePage(t *testing.T) {
	f := testFonts(t)
	long := "word "
	p := Layout(Document{Blocks: []Block{{Kind: KindParagraph, Lines: []string{repeat(long, 40)}, End: 1}}},
		LayoutOptions{Width: 40, Fonts: f, Palette: testPalette()})
	for _, it := range p.Items {
		if it.Kind != ItemText {
			continue
		}
		if got := int(f.Measure(it.Text, it.SizePx, it.Mono)) / 64; got > (p.Width-2*pageMargin)*f.CellW {
			t.Fatalf("line %q is %d px, over the page", it.Text, got)
		}
	}
}

func TestLayoutTableAndProperties(t *testing.T) {
	f := testFonts(t)
	doc := Parse([]byte("---\nkey: value\n---\n\n| a | b |\n|---|---|\n| 1 | 2 |\n"))
	p := Layout(doc, LayoutOptions{Width: 60, Fonts: f, Palette: testPalette()})
	var texts []string
	for _, it := range p.Items {
		if it.Kind == ItemText {
			texts = append(texts, it.Text)
		}
	}
	for _, want := range []string{"key", "value", "a", "b", "1", "2"} {
		if !contains(texts, want) {
			t.Fatalf("table/properties missing %q in %v", want, texts)
		}
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

func contains(all []string, want string) bool {
	for _, s := range all {
		if s == want {
			return true
		}
	}
	return false
}

func TestLayoutStripesAndDiffColours(t *testing.T) {
	f := testFonts(t)
	pal := testPalette()
	doc := Parse([]byte("| a | b |\n|---|---|\n| 1 | 2 |\n| 3 | 4 |\n| 5 | 6 |\n\n```diff\n+added\n-removed\n```\n"))
	p := Layout(doc, LayoutOptions{Width: 60, Fonts: f, Palette: pal})
	var stripes, seps int
	added, removed := false, false
	for _, it := range p.Items {
		switch {
		case it.Kind == ItemRect && it.Color == pal.Stripe && !it.Thin:
			stripes++
		case it.Kind == ItemRect && it.Thin:
			seps++
		case it.Kind == ItemText && it.Text == "+added":
			added = it.Color == pal.Alert[1]
		case it.Kind == ItemText && it.Text == "-removed":
			removed = it.Color == pal.Alert[4]
		}
	}
	if stripes == 0 {
		t.Fatal("no table stripe")
	}
	if seps < 2 {
		t.Fatalf("separators = %d, want the column lines and the header rule", seps)
	}
	if !added || !removed {
		t.Fatalf("diff colours: added=%v removed=%v", added, removed)
	}
}

func TestNestedListsDrawEveryItem(t *testing.T) {
	f := testFonts(t)
	doc := Parse([]byte("- a\n  - b\n    - c\n"))
	p := Layout(doc, LayoutOptions{Width: 40, Fonts: f, Palette: testPalette()})
	var texts []string
	for _, it := range p.Items {
		if it.Kind == ItemText {
			texts = append(texts, it.Text)
		}
	}
	for _, want := range []string{"a", "b", "c"} {
		if !contains(texts, want) {
			t.Fatalf("nested list lost %q: %v", want, texts)
		}
	}
}

func TestTableColumnsFitThePage(t *testing.T) {
	f := testFonts(t)
	doc := Parse([]byte("| key | value |\n|---|---|\n| a | a long value that needs room |\n"))
	p := Layout(doc, LayoutOptions{Width: 30, Fonts: f, Palette: testPalette()})
	for _, it := range p.Items {
		if it.Kind != ItemText {
			continue
		}
		if it.X+it.W > p.Width {
			t.Fatalf("table cell %q ends at %d, over the page %d", it.Text, it.X+it.W, p.Width)
		}
		if !utf8.ValidString(it.Text) {
			t.Fatalf("table cell %q is not valid UTF-8", it.Text)
		}
	}
}

func TestHeadingsDrawAtTheirSize(t *testing.T) {
	f := testFonts(t)
	body := f.Face(f.BodyPx, false).Metrics()
	head := f.Face(f.BodyPx*2, false).Metrics()
	if head.Height <= body.Height {
		t.Fatalf("heading face %v is not bigger than body %v", head.Height, body.Height)
	}
}
