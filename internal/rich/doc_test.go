package rich

import (
	"strings"
	"testing"
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
	links := doc.Blocks[0].Links
	if len(links) != 2 {
		t.Fatalf("links = %+v, want the wiki link and the Markdown link", links)
	}
	runes := []rune(doc.Blocks[0].Lines[0])
	if got := string(runes[links[0].Start:links[0].End]); got != "the guide" || links[0].Target != "Docker" {
		t.Fatalf("wiki span = %q -> %q", got, links[0].Target)
	}
	if got := string(runes[links[1].Start:links[1].End]); got != "the docs" || links[1].Target != "https://example.com/docs" {
		t.Fatalf("md span = %q -> %q", got, links[1].Target)
	}
	if span := string(runes[links[0].Start:links[0].End]); span != "the guide" {
		t.Fatalf("wiki span text = %q", span)
	}
}

func TestCodeLinesDropCRLF(t *testing.T) {
	doc := Parse([]byte("```go\r\nx := 1\r\n\r\n```\r\n"))
	if got := doc.Blocks[0].Code[0]; got != "x := 1" {
		t.Fatalf("code line = %q, want the CRLF dropped", got)
	}
}
