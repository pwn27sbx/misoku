package rich

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseBlocksAndSourceLines(t *testing.T) {
	src := []byte(`---
status: active
tags: [a, b]
---
# Title

First paragraph
wrapped here.

- one
- [x] done task
  - nested

> [!NOTE]
> a callout body

` + "```go\nfunc main() {}\n```\n\n$$ x^2 $$\n\n```mermaid\nflowchart LR\nA-->B\n```\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	doc := Parse(src)

	var kinds []Kind
	for _, b := range doc.Blocks {
		kinds = append(kinds, b.Kind)
	}
	want := []Kind{KindProperties, KindHeading, KindParagraph, KindParagraph, KindQuote, KindCode, KindMath, KindDiagram, KindTable}
	if len(kinds) != len(want) {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("block %d kind = %v, want %v (all: %v)", i, kinds[i], want[i], kinds)
		}
	}

	// The properties table keeps its top-level keys.
	props := doc.Blocks[0].Rows
	if len(props) != 2 || props[0][0] != "status" || props[1][0] != "tags" {
		t.Fatalf("properties = %v", props)
	}

	// Source lines point back at the note: the heading is line 4 (0-based).
	if got := doc.Blocks[1].Line; got != 4 {
		t.Fatalf("heading line = %d, want 4", got)
	}
	// The list keeps its items, and the checkbox becomes a drawn glyph.
	list := doc.Blocks[3]
	if len(list.Blocks) < 2 {
		t.Fatalf("list items = %d, want at least 2", len(list.Blocks))
	}
	if got := list.Blocks[1].Lines[0]; !strings.HasPrefix(got, "☑") {
		t.Fatalf("task item = %q, want a checked box", got)
	}
	if len(list.Blocks[1].Blocks) != 1 {
		t.Fatalf("nested items = %d, want 1", len(list.Blocks[1].Blocks))
	}

	// The callout's label is not a paragraph in the quote.
	quote := doc.Blocks[4]
	if quote.Lines[0] != "NOTE" {
		t.Fatalf("callout label = %q, want NOTE", quote.Lines[0])
	}
	if len(quote.Blocks) != 1 || quote.Blocks[0].Lines[0] != "a callout body" {
		t.Fatalf("callout body = %+v", quote.Blocks)
	}

	// Tables keep their header row first.
	table := doc.Blocks[8].Rows
	if len(table) != 2 || table[0][0] != "a" || table[1][1] != "2" {
		t.Fatalf("table = %v", table)
	}
}

func TestParseWikiLinksAndImages(t *testing.T) {
	doc := Parse([]byte("See [[Docker|the container guide]] and [[Plain]].\n\n![alt text](pic.png)\n"))
	if got := doc.Blocks[0].Lines[0]; got != "See the container guide and Plain." {
		t.Fatalf("wiki links = %q", got)
	}
	if doc.Blocks[1].Kind != KindImage || doc.Blocks[1].Src != "pic.png" || doc.Blocks[1].Lines[0] != "alt text" {
		t.Fatalf("image block = %+v", doc.Blocks[1])
	}
}

func TestParseKeepsHardBreaks(t *testing.T) {
	doc := Parse([]byte("line one  \nline two"))
	if got := doc.Blocks[0].Lines[0]; got != "line one\nline two" {
		t.Fatalf("paragraph = %q, want the hard break kept", got)
	}
}

func TestParseFootnotesDetailsAndDiff(t *testing.T) {
	src := []byte("Text with a note.[^1]\n\n[^1]: the footnote\n\n<details><summary>More</summary>hidden body</details>\n\n```diff\n+added\n-removed\n@@ hunk\n```\n")
	doc := Parse(src)
	var foot, det, diff *Block
	for i := range doc.Blocks {
		b := &doc.Blocks[i]
		switch {
		case b.Kind == KindTable && len(b.Rows) > 0:
			foot = b
		case b.Kind == KindQuote && len(b.Lines) > 0 && b.Lines[0] == "▸ More":
			det = b
		case b.Kind == KindCode && b.Lang == "diff":
			diff = b
		}
	}
	if foot == nil {
		t.Fatal("the footnote list was not parsed")
	}
	if len(foot.Rows[0]) != 2 || foot.Rows[0][1] != "the footnote" {
		t.Fatalf("footnote row = %v", foot.Rows[0])
	}
	if det == nil || len(det.Blocks) != 1 || det.Blocks[0].Lines[0] != "hidden body" {
		t.Fatalf("details block = %+v", det)
	}
	if diff == nil || len(diff.Code) != 3 {
		t.Fatalf("diff block = %+v", diff)
	}
}

func TestParseHTMLInlineMarkKbdRuby(t *testing.T) {
	src := []byte("Press <kbd>Space</kbd> for <mark>important</mark> <ruby>漢字<rp>(</rp><rt>かんじ</rt><rp>)</rp></ruby>.\n")
	doc := Parse(src)
	if len(doc.Blocks) != 1 {
		t.Fatalf("blocks = %d, want 1", len(doc.Blocks))
	}
	b := doc.Blocks[0]
	runes := []rune(b.Lines[0])
	var kbd, mark, ruby string
	for _, sp := range b.Spans {
		seg := string(runes[sp.Start:sp.End])
		if sp.Kbd {
			kbd = seg
		}
		if sp.Mark {
			mark = seg
		}
		if sp.Ruby {
			ruby = seg
		}
	}
	if kbd != "Space" {
		t.Errorf("kbd = %q, want Space (line=%q, spans=%+v)", kbd, b.Lines[0], b.Spans)
	}
	if mark != "important" {
		t.Errorf("mark = %q, want important (line=%q, spans=%+v)", mark, b.Lines[0], b.Spans)
	}
	if ruby != "(かんじ)" {
		t.Errorf("ruby = %q, want (かんじ) (line=%q, spans=%+v)", ruby, b.Lines[0], b.Spans)
	}
	if strings.Contains(b.Lines[0], "((") {
		t.Errorf("rp doubled parens in %q", b.Lines[0])
	}
}

func TestWikiLinksInsideFencesStayLiteral(t *testing.T) {
	doc := Parse([]byte("```\n[[Note|alias]]\n```\n\noutside [[Note|alias]]\n"))
	if got := doc.Blocks[0].Code[0]; got != "[[Note|alias]]" {
		t.Fatalf("fence text = %q, want the literal link", got)
	}
	if got := doc.Blocks[1].Lines[0]; got != "outside alias" {
		t.Fatalf("paragraph = %q", got)
	}
}

func TestParseLinkSpans(t *testing.T) {
	doc := Parse([]byte("Read [[Docker|the guide]] or [the docs](https://example.com/docs).\n"))
	var links []Span
	for _, sp := range doc.Blocks[0].Spans {
		if sp.Link != "" {
			links = append(links, sp)
		}
	}
	if len(links) != 2 {
		t.Fatalf("links = %+v, want the wiki link and the Markdown link", links)
	}
	runes := []rune(doc.Blocks[0].Lines[0])
	if got := string(runes[links[0].Start:links[0].End]); got != "the guide" || links[0].Link != "Docker" {
		t.Fatalf("wiki span = %q -> %q", got, links[0].Link)
	}
	if got := string(runes[links[1].Start:links[1].End]); got != "the docs" || links[1].Link != "https://example.com/docs" {
		t.Fatalf("md span = %q -> %q", got, links[1].Link)
	}
}

// Inline maths is parsed into a marker span: the page lays the formula out
// instead of drawing its source, while prices and code stay literal text.
func TestParseMarksInlineMath(t *testing.T) {
	src := "Energy $E=mc^2$ and \\(a+b\\) here; prices $5 and $6 stay; code `$x$` stays; a \\$9 bill; emphasis $a*b*c + x_1 + x_2$ and braces $\\{x\\}$ kept.\n"
	doc := Parse([]byte(src))
	if len(doc.Blocks) == 0 {
		t.Fatal("no blocks parsed")
	}
	b := doc.Blocks[0]
	runes := []rune(b.Lines[0])
	var got []string
	for _, sp := range b.Spans {
		if sp.Math == "" {
			continue
		}
		got = append(got, sp.Math)
		if sp.Start < 0 || sp.End > len(runes) || sp.Start >= sp.End {
			t.Fatalf("maths span %d..%d outside the line", sp.Start, sp.End)
		}
		if literal := string(runes[sp.Start:sp.End]); !strings.Contains(literal, sp.Math) {
			t.Fatalf("maths span %q does not cover its literal %q", sp.Math, literal)
		}
	}
	want := []string{"E=mc^2", "a+b", "a*b*c + x_1 + x_2", `\{x\}`}
	if len(got) != len(want) {
		t.Fatalf("maths spans = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("maths span %d = %q, want %q", i, got[i], want[i])
		}
	}
	// The flattened text keeps a readable literal of every formula; the spans
	// carry the source the engine draws instead.
	text := b.Lines[0]
	for _, keep := range []string{"$E=mc^2$", "$a+b$", "$5 and $6", "$x$", "$9"} {
		if !strings.Contains(text, keep) {
			t.Fatalf("text %q lost %q", text, keep)
		}
	}
	if strings.Contains(text, `\(`) {
		t.Fatalf("text %q still carries the \\( delimiter", text)
	}
}

// A maths block written as a ```math fence is still a block, not an inline
// run: the pre-pass leaves fenced content alone.
func TestParseLeavesFencedMathAlone(t *testing.T) {
	doc := Parse([]byte("```math\n\\frac{a}{b}\n```\n\n```go\nx := \"$y$\"\n```\n"))
	if len(doc.Blocks) != 2 || doc.Blocks[0].Kind != KindMath || doc.Blocks[1].Kind != KindCode {
		t.Fatalf("blocks = %+v", doc.Blocks)
	}
	if doc.Blocks[1].Code[0] != `x := "$y$"` {
		t.Fatalf("code line = %q, want the dollar signs untouched", doc.Blocks[1].Code[0])
	}
}

// The pre-pass must leave code alone whatever fence it uses: a longer fence is
// not closed by a shorter one, and the indented form counts as code too.
func TestParseInlineMathSkipsEveryKindOfCode(t *testing.T) {
	src := "````md\n```go\n$x$ is documented\n```\n````\n\n    $y$ is indent code\n\nAfter $z$ maths.\n"
	doc := Parse([]byte(src))
	var text string
	for _, b := range doc.Blocks {
		text += strings.Join(b.Lines, "") + "\n"
		text += strings.Join(b.Code, "") + "\n"
	}
	if !strings.Contains(text, "$y$ is indent code") {
		t.Fatalf("indented code lost its dollars: %q", text)
	}
	var maths []string
	for _, b := range doc.Blocks {
		for _, sp := range b.Spans {
			if sp.Math != "" {
				maths = append(maths, sp.Math)
			}
		}
	}
	if len(maths) != 1 || maths[0] != "z" {
		t.Fatalf("maths spans = %q, want only z after the fences", maths)
	}
}

// A marker rune the note already carried never reaches the page as tofu, and a
// lone one cannot pair with a later formula.
func TestParseStrayMathMarkerDoesNotLeak(t *testing.T) {
	doc := Parse([]byte("A lone \uE004 marker and $x$ maths.\n"))
	text := doc.Blocks[0].Lines[0]
	for _, r := range text {
		if r == mathMarkOpen || r == mathMarkClose {
			t.Fatalf("marker reached the page: %q", text)
		}
	}
	if !strings.ContainsRune(text, utf8.RuneError) {
		t.Fatalf("stray marker was not marked as invalid: %q", text)
	}
	if len(doc.Blocks[0].Spans) != 1 || doc.Blocks[0].Spans[0].Math != "x" {
		t.Fatalf("spans = %+v, want one x formula", doc.Blocks[0].Spans)
	}
}

// Table cells and footnotes draw their text without spans, but a formula
// there still reads as its source: no marker rune ever reaches them.
func TestParseTableAndFootnoteKeepMathsReadable(t *testing.T) {
	doc := Parse([]byte("| a | b |\n|---|---|\n| $x^2$ | y |\n\nA note[^1].\n\n[^1]: with $y_1$ inside.\n"))
	var cells []string
	for _, b := range doc.Blocks {
		for _, row := range b.Rows {
			cells = append(cells, row...)
		}
	}
	joined := strings.Join(cells, " | ")
	if !strings.Contains(joined, "$x^2$") || !strings.Contains(joined, "$y_1$") {
		t.Fatalf("table/footnote text lost a formula: %q", joined)
	}
	for _, r := range joined {
		if r >= 0xE000 && r <= 0xE01E {
			t.Fatalf("a marker reached a table or footnote: %q", joined)
		}
	}
}
