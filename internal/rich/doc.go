package rich

import (
	"bytes"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

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
	body = sanitizeReserved(body)
	body = markInlineMath(body)
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
			sort.SliceStable(b.Spans, func(i, j int) bool { return b.Spans[i].Start < b.Spans[j].Start })
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
		b.Spans = append(b.Spans, inlineSpans(t, src)...)
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
			b.Lines = []string{mathLiteral(string(img.Text(src)))}
			return b, true
		}
		b.Kind = KindParagraph
		b.Lines = []string{inlineText(t, src)}
		b.Spans = append(b.Spans, inlineSpans(t, src)...)
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
		text := codeLines(t, src)
		summary, body, ok := detailsText(strings.Join(text, "\n"))
		if !ok {
			return b, false
		}
		b.Kind = KindQuote
		b.Lines = []string{"▸ " + summary}
		b.Blocks = []Block{{Kind: KindParagraph, Lines: []string{body}, Line: b.Line, End: b.End}}
	case *extast.FootnoteList:
		b.Kind = KindTable
		b.Rows = footnoteRows(t, src)
	case *ast.ThematicBreak:
		b.Kind = KindRule
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
	text, _ := inlineTextSpans(n, src)
	return text
}

// inlineSpans is the bold and link spans of a block's inline text.
func inlineSpans(n ast.Node, src []byte) []Span {
	_, spans := inlineTextSpans(n, src)
	return spans
}

// inlineTextSpans flattens a block's inline Markdown and reports the bold and
// link runs it wrote, as rune offsets in the returned text.
func inlineTextSpans(n ast.Node, src []byte) (string, []Span) {
	var b strings.Builder
	var spans []Span
	markStart, kbdStart, rtStart := -1, -1, -1
	inRP := false
	var walk func(ast.Node)
	walk = func(n ast.Node) {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			switch t := c.(type) {
			case *ast.Text:
				if !inRP {
					writeWikiAware(&b, &spans, string(t.Segment.Value(src)))
					if t.SoftLineBreak() || t.HardLineBreak() {
						b.WriteString("\n")
					}
				}
			case *ast.String:
				if !inRP {
					writeWikiAware(&b, &spans, string(t.Value))
				}
			case *ast.RawHTML:
				var raw strings.Builder
				for i := 0; i < t.Segments.Len(); i++ {
					seg := t.Segments.At(i)
					raw.Write(seg.Value(src))
				}
				tag := strings.ToLower(strings.TrimSpace(raw.String()))
				now := utf8.RuneCountInString(b.String())
				switch {
				case tag == "<mark>" || strings.HasPrefix(tag, "<mark "):
					markStart = now
				case tag == "</mark>" && markStart >= 0:
					spans = append(spans, Span{Start: markStart, End: now, Mark: true})
					markStart = -1
				case tag == "<kbd>" || strings.HasPrefix(tag, "<kbd "):
					kbdStart = now
				case tag == "</kbd>" && kbdStart >= 0:
					spans = append(spans, Span{Start: kbdStart, End: now, Kbd: true})
					kbdStart = -1
				case tag == "<rp>" || strings.HasPrefix(tag, "<rp "):
					inRP = true
				case tag == "</rp>":
					inRP = false
				case tag == "<rt>" || strings.HasPrefix(tag, "<rt "):
					b.WriteString(" (")
					rtStart = utf8.RuneCountInString(b.String()) - 1
				case tag == "</rt>" && rtStart >= 0:
					b.WriteString(")")
					spans = append(spans, Span{Start: rtStart, End: utf8.RuneCountInString(b.String()), Ruby: true})
					rtStart = -1
				}
			case *ast.Emphasis:
				start := utf8.RuneCountInString(b.String())
				walkInline(c, &b, src)
				spans = append(spans, Span{Start: start, End: utf8.RuneCountInString(b.String()), Bold: t.Level == 2, Italic: t.Level == 1})
			case *ast.CodeSpan:
				start := utf8.RuneCountInString(b.String())
				walkInline(c, &b, src)
				spans = append(spans, Span{Start: start, End: utf8.RuneCountInString(b.String()), Code: true})
			case *ast.Link:
				start := utf8.RuneCountInString(b.String())
				for cc := c.FirstChild(); cc != nil; cc = cc.NextSibling() {
					walkInline(cc, &b, src)
				}
				spans = append(spans, Span{Start: start, End: utf8.RuneCountInString(b.String()), Link: string(t.Destination)})
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
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].Start < spans[j].Start })
	text, mathSpans := resolveMathMarkers(strings.TrimRight(b.String(), " "))
	spans = append(spans, mathSpans...)
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].Start < spans[j].Start })
	return text, spans
}

// writeWikiAware appends text, replacing wiki markers with their alias and
// recording each link at its final rune offset. Recording here, while
// writing, is what keeps every span aligned after the markers are gone.
func writeWikiAware(b *strings.Builder, spans *[]Span, text string) {
	matches := wikiMarker.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		b.WriteString(text)
		return
	}
	prev := 0
	runes := utf8.RuneCountInString(b.String()) // offsets are page-wide
	for _, m := range matches {
		prefix := text[prev:m[0]]
		b.WriteString(prefix)
		runes += utf8.RuneCountInString(prefix)
		start := runes
		alias := text[m[4]:m[5]]
		b.WriteString(alias)
		runes += utf8.RuneCountInString(alias)
		*spans = append(*spans, Span{Start: start, End: runes, Link: text[m[2]:m[3]]})
		prev = m[1]
	}
	b.WriteString(text[prev:])
}

// walkInline appends one inline node's text, for an emphasis or link's
// children.
func walkInline(c ast.Node, b *strings.Builder, src []byte) {
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
	default:
		for cc := c.FirstChild(); cc != nil; cc = cc.NextSibling() {
			walkInline(cc, b, src)
		}
	}
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
		// The line's terminator is not a glyph: drawing it left a tofu box
		// after the last character of every code line.
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
// wikiMark and friends wrap a wiki link in the parsed text: \uE000target
// \uE001alias\uE002. Substituting the alias at expansion time would lose its
// rune offsets as soon as Markdown removes ** or _ markers; resolving the
// markers after flattening keeps every span exact.
const (
	wikiMarkOpen  = '\uE000'
	wikiMarkMid   = '\uE001'
	wikiMarkClose = '\uE002'
)

var wikiMarker = regexp.MustCompile(`\x{E000}([^\x{E001}]*)\x{E001}([^\x{E002}]*)\x{E002}`)

// expandWikiLinks turns [[Note|alias]] into a marker that inlineTextSpans
// resolves, skipping fenced code so a documented link stays literal.
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
			target := parts[1]
			alias := target
			if len(parts) > 2 && len(parts[2]) > 0 {
				alias = parts[2]
			}
			return []byte(string(wikiMarkOpen) + string(target) + string(wikiMarkMid) + string(alias) + string(wikiMarkClose))
		}))
	}
	return []byte(strings.Join(lines, "\n"))
}

// resolveWikiMarkers removes the markers from text, appends one Link span per
// wiki link at its final offsets, and shifts the spans collected before (bold,
// Markdown links) by the characters each earlier marker removes.

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

// Inline maths is wrapped in two private-use markers before Markdown runs:
// $…$ and \(…\) would otherwise be eaten by emphasis (*, _ inside a formula)
// or by backslash escaping. The markers keep the source exact until the
// block's text is flattened, where resolveMathMarkers turns each pair into a
// readable literal plus one maths span.
const (
	mathMarkOpen  = '\uE004'
	mathMarkClose = '\uE005'
)

// markInlineMath marks inline formulas on every line outside code. Fenced
// blocks are tracked by their own character and length (a four-backtick fence
// is not closed by three), and the four-space indented form is left alone too.
// The private-use range U+E000..U+E01E is reserved for these markers and for
// the stand-ins that keep Markdown from eating a formula's characters;
// sanitizeReserved replaces any the note already carries.
func markInlineMath(src []byte) []byte {
	lines := strings.Split(string(src), "\n")
	var fence rune // 0 outside a fence
	fenceLen := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence != 0 {
			if r, n := fenceRun(trimmed); r == fence && n >= fenceLen {
				fence, fenceLen = 0, 0
			}
			continue
		}
		if isIndentedCode(line) {
			continue
		}
		if r, n := fenceRun(trimmed); n >= 3 {
			fence, fenceLen = r, n
			continue
		}
		lines[i] = markMathInLine(line)
	}
	return []byte(strings.Join(lines, "\n"))
}

// fenceRun reports the fence character and its run length at the start of a
// trimmed line, or 0 when the line does not open with a fence.
func fenceRun(s string) (rune, int) {
	if s == "" || (s[0] != '`' && s[0] != '~') {
		return 0, 0
	}
	r := rune(s[0])
	n := 0
	for _, c := range s {
		if c != r {
			break
		}
		n++
	}
	return r, n
}

// isIndentedCode reports Markdown's indented code form: four spaces or a tab
// before any other character.
func isIndentedCode(line string) bool {
	return strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ")
}

// markMathInLine replaces every inline formula of one line with its markers.
// Backtick code spans are copied untouched, so a documented `$x$` stays
// literal and can never close a formula that opened before it.
func markMathInLine(line string) string {
	runes := []rune(line)
	var b strings.Builder
	b.Grow(len(line))
	start := 0
	for i := 0; i < len(runes); {
		if runes[i] != '`' {
			i++
			continue
		}
		n := 0
		for i+n < len(runes) && runes[i+n] == '`' {
			n++
		}
		closeAt := backtickRun(runes, i+n, n)
		if closeAt < 0 {
			i += n
			continue
		}
		b.WriteString(markMathSegment(string(runes[start:i])))
		b.WriteString(string(runes[i : closeAt+n]))
		i = closeAt + n
		start = i
	}
	b.WriteString(markMathSegment(string(runes[start:])))
	return b.String()
}

// backtickRun is the index of the next run of exactly n backticks at or after
// from, or -1.
func backtickRun(runes []rune, from, n int) int {
	for i := from; i < len(runes); i++ {
		if runes[i] != '`' {
			continue
		}
		j := i
		for j < len(runes) && runes[j] == '`' {
			j++
		}
		if j-i == n {
			return i
		}
		i = j - 1
	}
	return -1
}

// markMathSegment marks the formulas of a stretch of line that holds no code.
func markMathSegment(text string) string {
	runes := []rune(text)
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(runes); {
		r := runes[i]
		switch {
		case r == '\\' && i+1 < len(runes) && runes[i+1] == '(':
			if end := parenMathEnd(runes, i+2); end > i+2 {
				writeMathMark(&b, string(runes[i+2:end]))
				i = end + 2 // past the closing "\)"
			} else {
				b.WriteRune(r)
				i++
			}
		case r == '\\':
			// Keep escapes whole for Markdown: "\$" is a literal dollar.
			b.WriteRune(r)
			if i+1 < len(runes) {
				b.WriteRune(runes[i+1])
				i += 2
			} else {
				i++
			}
		case r == '$':
			if end, ok := dollarMathEnd(runes, i); ok {
				writeMathMark(&b, string(runes[i+1:end]))
				i = end + 1
				continue
			}
			b.WriteRune(r)
			i++
		default:
			b.WriteRune(r)
			i++
		}
	}
	return b.String()
}

func writeMathMark(b *strings.Builder, src string) {
	b.WriteRune(mathMarkOpen)
	b.WriteString(escapeMath(strings.TrimSpace(src)))
	b.WriteRune(mathMarkClose)
}

// markdownActive maps each character Markdown treats as inline syntax to a
// private-use stand-in: one rune for one rune, so the flattened text keeps its
// offsets and the formula reaches the engine exactly as written. Without it,
// "$a*b*c$" loses "*b*" to emphasis and "$\{x\}$" loses its backslash.
var markdownActive = map[rune]rune{
	'\\': '\uE010', '`': '\uE011', '*': '\uE012', '_': '\uE013',
	'[': '\uE014', ']': '\uE015', '(': '\uE016', ')': '\uE017',
	'!': '\uE018', '<': '\uE019', '>': '\uE01A', '&': '\uE01B',
	'~': '\uE01C', '|': '\uE01D', '#': '\uE01E',
}

var markdownRestored = func() map[rune]rune {
	m := make(map[rune]rune, len(markdownActive))
	for r, stand := range markdownActive {
		m[stand] = r
	}
	return m
}()

func escapeMath(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	for _, r := range src {
		if stand, ok := markdownActive[r]; ok {
			b.WriteRune(stand)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func unescapeMath(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	for _, r := range src {
		if real, ok := markdownRestored[r]; ok {
			b.WriteRune(real)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// sanitizeReserved replaces the private-use runes the page reserves for its
// own markers. They are invisible junk in a note, and left alone they would
// pair with a real formula or reach the page as tofu.
func sanitizeReserved(src []byte) []byte {
	var b strings.Builder
	b.Grow(len(src))
	for _, r := range string(src) {
		if r >= 0xE000 && r <= 0xE01E {
			b.WriteRune(utf8.RuneError)
			continue
		}
		b.WriteRune(r)
	}
	return []byte(b.String())
}

// parenMathEnd finds the \) that closes a \(…\) formula, or -1.
func parenMathEnd(runes []rune, from int) int {
	for i := from; i+1 < len(runes); i++ {
		if runes[i] == '\\' {
			if runes[i+1] == ')' {
				return i
			}
			i++ // an escape inside the formula consumes the next rune
		}
	}
	return -1
}

// dollarMathEnd finds the '$' that closes an inline formula. The content
// cannot start or end with a space, and the closer cannot be followed by a
// digit or another dollar, so "$5 and $6" stays text.
func dollarMathEnd(runes []rune, open int) (int, bool) {
	if open+1 >= len(runes) || unicode.IsSpace(runes[open+1]) || runes[open+1] == '$' {
		return 0, false
	}
	for i := open + 1; i < len(runes); i++ {
		switch runes[i] {
		case '\\':
			i++ // an escape's next rune is content, never the closer
		case '$':
			if unicode.IsSpace(runes[i-1]) {
				continue
			}
			if i+1 < len(runes) && (unicode.IsDigit(runes[i+1]) || runes[i+1] == '$') {
				continue
			}
			return i, true
		}
	}
	return 0, false
}

// resolveMathMarkers replaces each marked formula with its readable literal
// and reports the span the layout turns into a box. The literal is as long in
// runes as the marked source, so spans collected before keep their offsets.
func resolveMathMarkers(text string) (string, []Span) {
	if !strings.ContainsRune(text, mathMarkOpen) {
		return text, nil
	}
	runes := []rune(text)
	var b strings.Builder
	var spans []Span
	for i := 0; i < len(runes); {
		if runes[i] == mathMarkOpen {
			if j := indexRune(runes, mathMarkClose, i+1); j > i {
				src := unescapeMath(string(runes[i+1 : j]))
				literal := "$" + src + "$"
				spans = append(spans, Span{Start: i, End: i + len([]rune(literal)), Math: src})
				b.WriteString(literal)
				i = j + 1
				continue
			}
		}
		if runes[i] == mathMarkOpen || runes[i] == mathMarkClose {
			// An unpaired marker is not the reader's text: show a replacement
			// rune and keep the offsets stable.
			b.WriteRune(utf8.RuneError)
			i++
			continue
		}
		b.WriteRune(runes[i])
		i++
	}
	return b.String(), spans
}

// indexRune is the rune index of r at or after from, or -1.
func indexRune(runes []rune, r rune, from int) int {
	for i := from; i < len(runes); i++ {
		if runes[i] == r {
			return i
		}
	}
	return -1
}

// mathLiteral is text with every formula marker replaced by its literal, for
// the places that draw a block's text without going through the span walk.
func mathLiteral(text string) string {
	out, _ := resolveMathMarkers(text)
	return out
}

// footnoteRows turns the footnote list into a two-column table: the label and
// its text, so the page draws them under a rule like a published note.
func footnoteRows(list *extast.FootnoteList, src []byte) [][]string {
	var rows [][]string
	for n := list.FirstChild(); n != nil; n = n.NextSibling() {
		label, text := "", ""
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Kind() == extast.KindFootnoteLink {
				label = inlineText(c, src)
				continue
			}
			if t := inlineText(c, src); t != "" && text == "" {
				text = t
			}
		}
		if label != "" || text != "" {
			rows = append(rows, []string{label, text})
		}
	}
	return rows
}

// detailsText reads the raw "<details><summary>…</summary>body" form: the page
// keeps the summary and the body, folded.
func detailsText(text string) (summary, body string, ok bool) {
	lower := strings.ToLower(text)
	if !strings.HasPrefix(strings.TrimSpace(lower), "<details") {
		return "", "", false
	}
	start := strings.Index(lower, "<summary>")
	end := strings.Index(lower, "</summary>")
	if start < 0 || end < 0 || end < start {
		return "", "", false
	}
	summary = mathLiteral(strings.TrimSpace(text[start+len("<summary>") : end]))
	body = mathLiteral(strings.TrimSpace(text[end+len("</summary>"):]))
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
