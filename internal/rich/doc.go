package rich

import (
	"bytes"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Parse turns a note into a Document: one block per top-level Markdown
// construct, each with the source lines it came from. It is the rich view's
// only Markdown entry point; everything after it works on blocks, never on
// Markdown.
func Parse(src []byte) Document {
	body, props, consumed := splitFrontMatter(src)
	body, lineMap := fenceMathBlocks(body, consumed)
	body = expandWikiLinks(body)

	var doc Document
	if len(props) > 0 {
		doc.Blocks = append(doc.Blocks, Block{Kind: KindProperties, Rows: props, Line: 0, End: 0})
	}

	md := goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote))
	tree := md.Parser().Parse(text.NewReader(body))
	starts := lineStarts(body)
	// The pre-passes may add lines (a one-line $$…$$ becomes a fence); the
	// map puts every transformed line back on its note line.
	lineOf := func(off int) int {
		i := sort.Search(len(starts), func(i int) bool { return starts[i] > off }) - 1
		if i < 0 {
			i = 0
		}
		if i >= len(lineMap) {
			i = max(len(lineMap)-1, 0)
		}
		if len(lineMap) == 0 {
			return 0
		}
		return lineMap[i]
	}

	var blocks []Block
	for n := tree.FirstChild(); n != nil; n = n.NextSibling() {
		if b, ok := blockFrom(n, body, lineOf); ok {
			blocks = append(blocks, b)
		}
	}
	doc.Blocks = append(doc.Blocks, blocks...)
	return doc
}

// blockFrom converts one top-level node. ok is false for nodes that draw
// nothing (HTML comments, link reference definitions).
func blockFrom(n ast.Node, src []byte, lineOf func(int) int) (Block, bool) {
	start, end := nodeLines(n, lineOf)
	b := Block{Line: start, End: end}
	switch t := n.(type) {
	case *ast.Heading:
		b.Kind = KindHeading
		b.Level = t.Level
		b.Lines = []string{inlineText(t, src)}
	case *ast.Paragraph:
		if summary, body, ok := detailsText(strings.TrimSpace(inlineText(t, src))); ok {
			b.Kind = KindQuote
			b.Lines = []string{"▸ " + summary}
			b.Blocks = []Block{{Kind: KindParagraph, Lines: []string{body}, Line: b.Line, End: b.End}}
			return b, true
		}
		if img, ok := onlyImage(t); ok {
			b.Kind = KindImage
			b.Src = string(img.Destination)
			b.Lines = []string{string(img.Text(src))}
			return b, true
		}
		b.Kind = KindParagraph
		b.Lines = []string{inlineText(t, src)}
	case *ast.List:
		b.Kind = KindParagraph
		b.Blocks = listItems(t, src, lineOf)
		if len(b.Blocks) == 0 {
			return b, false
		}
	case *ast.Blockquote:
		b.Kind = KindQuote
		items := quoteBlocks(t, src, lineOf)
		if len(items) == 0 {
			return b, false
		}
		// A callout keeps its label out of the text: "> [!NOTE]" is the
		// block's colour, not a paragraph that starts with brackets.
		if label, rest, ok := callout(items[0]); ok {
			b.Lines = []string{label}
			items[0] = rest
		}
		b.Blocks = items
	case *ast.FencedCodeBlock:
		b.Kind = KindCode
		b.Lang = string(t.Language(src))
		switch b.Lang {
		case "math":
			b.Kind = KindMath
		case "mermaid", "flow":
			b.Kind = KindDiagram
		}
		b.Code = codeLines(t, src)
	case *ast.CodeBlock:
		b.Kind = KindCode
		b.Code = codeLines(t, src)
	case *ast.HTMLBlock:
		// A block of raw HTML: <details> keeps its summary and body, the
		// rest of the tag soup draws nothing (the text compiler strips it).
		text := codeLines(t, src)
		summary, body, ok := detailsText(strings.Join(text, "\n"))
		if !ok {
			return b, false
		}
		b.Kind = KindQuote
		b.Lines = []string{"▸ " + summary}
		b.Blocks = []Block{{Kind: KindParagraph, Lines: []string{body}, Line: b.Line, End: b.End}}
	case *ast.ThematicBreak:
		b.Kind = KindRule
	case *extast.FootnoteList:
		b.Kind = KindTable
		b.Rows = footnoteRows(t, src)
	case *extast.Table:
		b.Kind = KindTable
		b.Rows = tableRows(t, src)
	default:
		return b, false
	}
	if b.Kind == KindParagraph && len(b.Blocks) == 0 && strings.TrimSpace(strings.Join(b.Lines, "")) == "" {
		return b, false
	}
	return b, true
}

// nodeLines reports the source lines of a node. Containers (lists, quotes,
// tables) have no line segments of their own, so their children are used.
func nodeLines(n ast.Node, lineOf func(int) int) (int, int) {
	first, last := -1, -1
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || c.Type() != ast.TypeBlock {
			return ast.WalkContinue, nil
		}
		lines := c.Lines()
		for i := 0; i < lines.Len(); i++ {
			s := lines.At(i)
			a, b := lineOf(s.Start), lineOf(max(s.Stop-1, s.Start))
			if first < 0 || a < first {
				first = a
			}
			if b > last {
				last = b
			}
		}
		return ast.WalkContinue, nil
	})
	if first < 0 {
		return 0, 0
	}
	return first, last + 1
}

// inlineText flattens a block's inline Markdown to the text the page draws,
// with one line per soft or hard break.
func inlineText(n ast.Node, src []byte) string {
	var b strings.Builder
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch t := c.(type) {
			case *ast.Text:
				b.Write(t.Segment.Value(src))
				if t.SoftLineBreak() || t.HardLineBreak() {
					b.WriteString("\n")
				}
			case *ast.String:
				b.Write(t.Value)
			case *ast.CodeSpan:
				for cc := c.FirstChild(); cc != nil; cc = cc.NextSibling() {
					if tt, ok := cc.(*ast.Text); ok {
						b.Write(tt.Segment.Value(src))
					}
				}
			case *ast.AutoLink:
				b.Write(t.Label(src))
			case *ast.Image:
				b.WriteString(string(t.Text(src)))
			default:
				walk(c)
			}
		}
	}
	walk(n)
	return strings.TrimRight(b.String(), " ")
}

func onlyImage(p *ast.Paragraph) (*ast.Image, bool) {
	if p.ChildCount() != 1 {
		return nil, false
	}
	img, ok := p.FirstChild().(*ast.Image)
	return img, ok
}

func listItems(l *ast.List, src []byte, lineOf func(int) int) []Block {
	var out []Block
	for item := l.FirstChild(); item != nil; item = item.NextSibling() {
		li, ok := item.(*ast.ListItem)
		if !ok {
			continue
		}
		text := ""
		var kids []Block
		for c := li.FirstChild(); c != nil; c = c.NextSibling() {
			switch t := c.(type) {
			case *ast.TextBlock, *ast.Paragraph:
				if text == "" {
					text = inlineText(c, src)
				}
			case *ast.List:
				kids = append(kids, listItems(t, src, lineOf)...)
			}
		}
		start, end := nodeLines(li, lineOf)
		b := Block{Kind: KindListItem, Lines: []string{text}, Line: start, End: end}
		if box := taskBox(li); box != "" {
			b.Lines[0] = box + " " + b.Lines[0]
		}
		b.Blocks = kids
		out = append(out, b)
	}
	return out
}

func taskBox(li *ast.ListItem) string {
	for c := li.FirstChild(); c != nil; c = c.NextSibling() {
		if _, ok := c.(*ast.TextBlock); !ok {
			continue
		}
		for cc := c.FirstChild(); cc != nil; cc = cc.NextSibling() {
			if box, ok := cc.(*extast.TaskCheckBox); ok {
				if box.IsChecked {
					return "☑"
				}
				return "☐"
			}
		}
	}
	return ""
}

func quoteBlocks(q *ast.Blockquote, src []byte, lineOf func(int) int) []Block {
	var out []Block
	for c := q.FirstChild(); c != nil; c = c.NextSibling() {
		if b, ok := blockFrom(c, src, lineOf); ok {
			out = append(out, b)
		}
	}
	return out
}

var calloutRe = regexp.MustCompile(`^\[!([A-Za-z]+)\]\s*(.*)$`)

// callout splits a quote's first paragraph into its label and the rest.
func callout(b Block) (label string, rest Block, ok bool) {
	if b.Kind != KindParagraph || len(b.Lines) == 0 {
		return "", b, false
	}
	m := calloutRe.FindStringSubmatch(b.Lines[0])
	if m == nil {
		return "", b, false
	}
	b.Lines = b.Lines[1:]
	if m[2] != "" {
		b.Lines = append([]string{m[2]}, b.Lines...)
	}
	if len(b.Lines) == 0 {
		b.Lines = []string{""}
	}
	return strings.ToUpper(m[1]), b, true
}

func codeLines(n ast.Node, src []byte) []string {
	lines := n.Lines()
	var out []string
	for i := 0; i < lines.Len(); i++ {
		s := lines.At(i)
		out = append(out, strings.TrimRight(string(s.Value(src)), "\r\n"))
	}
	return out
}

func tableRows(t *extast.Table, src []byte) [][]string {
	var rows [][]string
	cellText := func(cell ast.Node) string { return inlineText(cell, src) }
	collect := func(row ast.Node) []string {
		var cells []string
		for c := row.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Kind() == extast.KindTableCell {
				cells = append(cells, strings.TrimSpace(cellText(c)))
			}
		}
		return cells
	}
	for child := t.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case extast.KindTableHeader:
			// The header's children are the cells themselves.
			rows = append(rows, collect(child))
		case extast.KindTableRow:
			rows = append(rows, collect(child))
		}
	}
	return rows
}

// splitFrontMatter takes a leading "---" block off the note and returns its
// top-level keys as table rows and how many note lines it consumed, so the
// blocks that follow keep their real line numbers.
func splitFrontMatter(src []byte) ([]byte, [][]string, int) {
	text := string(src)
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return src, nil, 0
	}
	rest := text[strings.IndexByte(text, '\n')+1:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return src, nil, 0
	}
	block := rest[:end]
	after := rest[end+4:]
	after = strings.TrimPrefix(after, "\n")
	consumed := 3 + strings.Count(block, "\n")

	var rows [][]string
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		rows = append(rows, []string{strings.TrimSpace(k), strings.TrimSpace(v)})
	}
	return []byte(after), rows, consumed
}

var wikiRe = regexp.MustCompile(`!?\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// expandWikiLinks turns "[[Note|alias]]" into its alias and a bare
// "[[Note]]" into its target, so the page draws readable text; the link
// itself becomes clickable when inline runs land.
func expandWikiLinks(src []byte) []byte {
	lines := strings.Split(string(src), "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		lines[i] = string(wikiRe.ReplaceAllFunc([]byte(line), func(m []byte) []byte {
			parts := wikiRe.FindSubmatch(m)
			if len(parts) > 2 && len(parts[2]) > 0 {
				return parts[2]
			}
			return parts[1]
		}))
	}
	return []byte(strings.Join(lines, "\n"))
}

// fenceMathBlocks turns standalone "$$ … $$" blocks into ```math fences, so
// the Markdown parser sees them as blocks and the page can hand them to the
// maths engine. A "$$" inside a code fence is left alone. It returns the new
// source and the note line of every new line, so block lines stay exact.
func fenceMathBlocks(src []byte, baseLine int) ([]byte, []int) {
	lines := strings.Split(string(src), "\n")
	var out []string
	var mapOut []int
	inFence := false
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			out = append(out, lines[i])
			mapOut = append(mapOut, baseLine+i)
			continue
		}
		if !inFence && strings.HasPrefix(trimmed, "$$") {
			// One line: $$ … $$ with the closing pair.
			if rest := strings.TrimPrefix(trimmed, "$$"); strings.HasSuffix(rest, "$$") && strings.TrimSpace(rest) != "" {
				out = append(out, "```math", strings.TrimSuffix(rest, "$$"), "```")
				mapOut = append(mapOut, baseLine+i, baseLine+i, baseLine+i)
				continue
			}
			// Several lines until a closing "$$".
			var body []string
			if rest := strings.TrimPrefix(trimmed, "$$"); rest != "" {
				body = append(body, rest)
			}
			j := i + 1
			for ; j < len(lines); j++ {
				if strings.HasSuffix(strings.TrimSpace(lines[j]), "$$") {
					if pre := strings.TrimSuffix(lines[j], "$$"); strings.TrimSpace(pre) != "" {
						body = append(body, pre)
					}
					break
				}
				body = append(body, lines[j])
			}
			out = append(out, "```math")
			mapOut = append(mapOut, baseLine+i)
			for k, line := range body {
				out = append(out, line)
				mapOut = append(mapOut, baseLine+i+1+k)
			}
			out = append(out, "```")
			mapOut = append(mapOut, baseLine+min(j, len(lines)-1))
			i = j
			continue
		}
		out = append(out, lines[i])
		mapOut = append(mapOut, baseLine+i)
	}
	return []byte(strings.Join(out, "\n")), mapOut
}

// footnoteRows turns the footnote list into a two-column table: the label and
// its text, so the page draws them under a rule like a published note.
func footnoteRows(list *extast.FootnoteList, src []byte) [][]string {
	var rows [][]string
	for n := list.FirstChild(); n != nil; n = n.NextSibling() {
		label := ""
		text := ""
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch c.Kind() {
			case extast.KindFootnoteLink:
				label = inlineText(c, src)
			default:
				if t := inlineText(c, src); t != "" && text == "" {
					text = t
				}
			}
		}
		if label != "" || text != "" {
			rows = append(rows, []string{label, text})
		}
	}
	return rows
}

// detailsBlock reads the raw "<details><summary>…</summary>body" form the text
// compiler strips: the page keeps the summary and the body, folded.
func detailsText(text string) (summary, body string, ok bool) {
	lower := strings.ToLower(text)
	if !strings.HasPrefix(lower, "<details") {
		return "", "", false
	}
	start := strings.Index(lower, "<summary>")
	end := strings.Index(lower, "</summary>")
	if start < 0 || end < 0 || end < start {
		return "", "", false
	}
	summary = strings.TrimSpace(text[start+len("<summary>") : end])
	body = strings.TrimSpace(text[end+len("</summary>"):])
	body = strings.ReplaceAll(body, "</details>", "")
	return summary, strings.TrimSpace(body), true
}

// lineStarts returns the byte offset where each 0-based line begins.
func lineStarts(src []byte) []int {
	starts := []int{0}
	for i, c := range src {
		if c == '\n' && i+1 < len(src) {
			starts = append(starts, i+1)
		}
	}
	return starts
}

var _ = bytes.MinRead
