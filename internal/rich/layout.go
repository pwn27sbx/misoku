package rich

import (
	"image/color"
	"image/draw"
	"math"
	"sort"
	"strings"
)

// ItemKind is what one display-list item draws.
type ItemKind int

const (
	ItemText ItemKind = iota
	ItemRect
	ItemBox // a maths or diagram box, drawn by its own engine
)

// Item is one piece of the page, in cell units: X and Y are cells from the
// page's top-left, W and H its size in cells. Text is drawn at SizePx with
// its baseline BaselinePx pixels below Y's top edge.
type Item struct {
	Kind       ItemKind
	X, Y, W, H int
	Text       string
	SizePx     float64
	Mono       bool
	Thin       bool // a rect drawn as a 2 px line: rules, borders, separators
	Color      color.RGBA
	BaselinePx int
	// XOffPx is the pixel offset inside X's cell: styled runs are placed at
	// their measured position, not rounded to a cell, so bold and links keep
	// the page's spacing.
	XOffPx int
	Bold   bool
	Italic bool
	Box    *Box
}

// Box is what a maths or diagram engine hands the page: its size in cells and
// a function that draws its pixels at (x, y) of the tile. The page never
// knows what is inside; the engine obeys the box.
type Box struct {
	W, H int
	Draw func(dst draw.Image, x, y int)
}

// InlineBox is an inline formula laid out for a text baseline: its width in
// pixels, how deep its ink reaches below the baseline, and a draw function
// that puts that ink's baseline at baselineY. The page places it inside a text
// line, so a formula keeps its size and sits on the same baseline as the words.
type InlineBox struct {
	WidthPx int
	DepthPx int // ink below the baseline, for the rows the item must claim
	Draw    func(dst draw.Image, x, baselineY int)
}

// Page is a note laid out in cells: the display list plus the mapping back to
// the note's source lines, which keeps the reader's place and opens the
// editable view at the same line.
type Page struct {
	Items  []Item
	Width  int // cells
	Height int // cells

	rows []rowSpan

	Headings []Heading
	Links    []LinkBox
	// Wide marks the row ranges of code blocks and tables wider than the
	// page: the view scrolls them sideways without moving the page.
	Wide []RowSpan
}

// RowSpan is a range of cell rows.
type RowSpan struct{ Row, Rows int }

// Heading is one outline entry: its level, text, source line and page row.
type Heading struct {
	Level int
	Text  string
	Line  int
	Row   int
}

// LinkBox is one clickable link on a page row, in cells.
type LinkBox struct {
	Row, X0, X1 int
	Target      string
}

type rowSpan struct {
	Line, Row, Rows int
}

// SourceLine is the note line the given cell row belongs to; a row between
// blocks answers with the next block's line, so scrolling never points at
// nothing.
func (p *Page) SourceLine(row int) int {
	if p == nil || len(p.rows) == 0 {
		return 0
	}
	i := sort.Search(len(p.rows), func(i int) bool { return p.rows[i].Row+p.rows[i].Rows > row })
	if i >= len(p.rows) {
		return p.rows[len(p.rows)-1].Line
	}
	return p.rows[i].Line
}

// RowOf is the first cell row of the block that starts at line, for position
// keeping after a reload.
func (p *Page) RowOf(line int) int {
	if p == nil || len(p.rows) == 0 {
		return 0
	}
	i := sort.Search(len(p.rows), func(i int) bool { return p.rows[i].Line >= line })
	if i >= len(p.rows) {
		return max(p.Height-1, 0)
	}
	return p.rows[i].Row
}

// LayoutOptions are what a page needs to know before it can be laid out.
type LayoutOptions struct {
	Width   int // cells
	Fonts   *Fonts
	Palette Palette
	// Math and Diagram turn a block into a sized box. Nil draws the source
	// in a bordered box instead, which is what the text fallback shows.
	Math    func(Block) *Box
	Diagram func(Block) *Box
	// MathInline turns an inline $…$ formula into ink for a text line. Nil
	// draws the formula's source as text, so a page without an engine still
	// shows something readable.
	MathInline func(src string, sizePx float64) *InlineBox
	// Images says what pictures the page may draw and from where. A source it
	// refuses keeps the placeholder box.
	Images ImageOptions
}

const (
	pageMargin  = 2 // cells left and right of the text
	blockGap    = 1 // blank cells between blocks
	headingGap  = 1
	listIndent  = 2
	codePadding = 1
)

// Layout turns a Document into a Page. Every block keeps its source lines so
// the page can be scrolled, searched and edited in place later.
func Layout(doc Document, opts LayoutOptions) *Page {
	p := &Page{Width: max(opts.Width, 8)}
	l := &layouter{page: p, f: opts.Fonts, pal: opts.Palette, math: opts.Math, diagram: opts.Diagram,
		inline: opts.MathInline, images: opts.Images}
	content := p.Width - 2*pageMargin
	if content < 8 {
		content = p.Width
	}
	l.content = content
	l.x0 = pageMargin
	for _, b := range doc.Blocks {
		l.block(b, 0)
	}
	p.Height = l.y
	return p
}

type layouter struct {
	page    *Page
	math    func(Block) *Box
	diagram func(Block) *Box
	inline  func(string, float64) *InlineBox
	f       *Fonts
	pal     Palette
	content int
	x0      int
	y       int
	images  ImageOptions
}

func (l *layouter) emit(it Item) { l.page.Items = append(l.page.Items, it) }

// block lays one block out at indent cells and records its source mapping.
func (l *layouter) block(b Block, indent int) {
	start := l.y
	switch b.Kind {
	case KindHeading:
		l.gapIfNeeded(headingGap)
		l.heading(b, indent)
	case KindParagraph:
		l.gapIfNeeded(0)
		if len(b.Blocks) > 0 {
			l.list(b, indent)
		} else if len(b.Spans) > 0 {
			l.linkBoxes(b, indent, l.f.BodyPx)
			l.styledLines(b, indent, l.f.BodyPx, l.pal.Fg)
			l.y += blockGap
		} else {
			l.paragraph(b.Lines, indent, l.f.BodyPx, l.pal.Fg)
			l.y += blockGap
		}
	case KindListItem:
		l.list(b, indent)
	case KindQuote:
		l.quote(b, indent)
	case KindCode, KindMath, KindDiagram:
		if box := l.engineBox(b, indent); box != nil {
			l.emit(Item{Kind: ItemBox, X: l.x0 + indent, Y: l.y, W: box.W, H: box.H, Box: box})
			l.y += box.H
			l.y += blockGap
			break
		}
		l.code(b, indent)
	case KindRule:
		l.gapIfNeeded(0)
		l.emit(Item{Kind: ItemRect, X: l.x0 + indent, Y: l.y, W: l.content - indent, H: 1, Thin: true, Color: l.pal.Rule})
		l.y++
	case KindTable:
		l.gapIfNeeded(0)
		l.table(b.Rows, indent)
		l.y += blockGap
	case KindProperties:
		l.gapIfNeeded(0)
		l.table(b.Rows, indent)
		l.y += blockGap
	case KindImage:
		l.gapIfNeeded(0)
		if box, ok := imageBox(b, l.images, l.content-indent, l.f.CellW, l.f.CellH); ok {
			l.emit(Item{Kind: ItemBox, X: l.x0 + indent, Y: l.y, W: box.W, H: box.H, Box: box})
			l.y += box.H + blockGap
			break
		}
		l.placeholder("🖼 "+firstLine(b.Lines), indent, 4)
		l.y += blockGap
	}
	if l.y > start {
		l.page.rows = append(l.page.rows, rowSpan{Line: b.Line, Row: start, Rows: l.y - start})
	}
}

func (l *layouter) gapIfNeeded(n int) {
	if len(l.page.Items) == 0 || l.y == 0 {
		return
	}
	l.y += n
}

func (l *layouter) heading(b Block, indent int) {
	level := min(max(b.Level, 1), 6)
	scale := []float64{2, 1.5, 1.25, 1, 0.875, 0.85}[level-1]
	size := l.f.BodyPx * scale
	col := l.pal.Heading[level-1]
	if col.A == 0 {
		col = l.pal.Fg
	}
	l.page.Headings = append(l.page.Headings, Heading{Level: level, Text: firstLine(b.Lines), Line: b.Line, Row: l.y})
	if len(b.Spans) > 0 {
		l.linkBoxes(b, indent, size)
		l.styledLines(b, indent, size, col)
		l.y++
		return
	}
	lines := l.f.Wrap(firstLine(b.Lines), (l.content-indent)*l.f.CellW, size, false)
	l.textLines(lines, indent, size, col, false)
	l.y++
}

// linkBoxes records where the block's links land on the page: rune offsets
// turned into cells with the same measurement the text uses.
func (l *layouter) linkBoxes(b Block, indent int, size float64) {
	if len(b.Spans) == 0 || len(b.Lines) == 0 {
		return
	}
	runes := []rune(b.Lines[0])
	cellWidth := func(n int) int {
		if n <= 0 || n > len(runes) {
			return 0
		}
		return (int(l.f.Measure(string(runes[:n]), size, false)) / 64) / l.f.CellW
	}
	for _, sp := range b.Spans {
		if sp.Link == "" {
			continue
		}
		x0 := l.x0 + indent + cellWidth(sp.Start)
		x1 := l.x0 + indent + cellWidth(sp.End)
		if x1 <= x0 {
			x1 = x0 + 1
		}
		l.page.Links = append(l.page.Links, LinkBox{Row: l.y, X0: x0, X1: x1, Target: sp.Link})
	}
}

// styledText is one run of a line: a link draws in the link colour, bold in
// the bold face, inline maths as a box, and the rest in the block's colour.
type styledText struct {
	Text   string
	Bold   bool
	Italic bool
	Code   bool
	Link   bool
	// Math is the formula's source when this run is inline maths; Box is the
	// ink the engine laid out for it. A run with Math but no Box draws Text.
	Math string
	Box  *InlineBox
}

// sameStyle reports whether two runs draw the same way, so adjacent segments
// can be merged back into one.
func (s styledText) sameStyle(o styledText) bool {
	return s.Bold == o.Bold && s.Italic == o.Italic && s.Code == o.Code &&
		s.Link == o.Link && s.Math == o.Math && s.Box == o.Box
}

// runsOf splits a block's first line into styled runs from its spans. The cuts
// are every span boundary, so an emphasis wrapped around a maths span keeps
// both: the inner run carries the formula, the outer its style. Spans are
// swept once in start order, so a line with many spans stays cheap.
func runsOf(b Block) []styledText {
	if len(b.Lines) == 0 {
		return nil
	}
	runes := []rune(b.Lines[0])
	spans := make([]Span, 0, len(b.Spans))
	cuts := []int{0, len(runes)}
	for _, sp := range b.Spans {
		start, end := clampSpan(sp.Start, sp.End, len(runes))
		if end <= start {
			continue
		}
		sp.Start, sp.End = start, end
		spans = append(spans, sp)
		cuts = append(cuts, start, end)
	}
	sort.Ints(cuts)
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].Start < spans[j].Start })
	var out []styledText
	var active []Span
	next := 0
	for i := 0; i+1 < len(cuts); i++ {
		start, end := cuts[i], cuts[i+1]
		if end <= start {
			continue
		}
		for next < len(spans) && spans[next].Start <= start {
			active = append(active, spans[next])
			next++
		}
		r := styledText{Text: string(runes[start:end])}
		kept := active[:0]
		for _, sp := range active {
			if sp.End <= start {
				continue
			}
			kept = append(kept, sp)
			if sp.Start > start {
				continue
			}
			r.Bold = r.Bold || sp.Bold
			r.Italic = r.Italic || sp.Italic
			r.Code = r.Code || sp.Code
			r.Link = r.Link || sp.Link != ""
			if sp.Math != "" {
				r.Math = sp.Math
			}
		}
		active = kept
		if n := len(out); n > 0 && out[n-1].sameStyle(r) {
			out[n-1].Text += r.Text
			continue
		}
		out = append(out, r)
	}
	return out
}

func clampSpan(start, end, n int) (int, int) {
	start = min(max(start, 0), n)
	end = min(max(end, start), n)
	return start, end
}

// styledLines lays a block out run by run: every line is a sequence of runs so
// bold, links and inline maths keep their style while wrapping.
func (l *layouter) styledLines(b Block, indent int, size float64, col color.RGBA) {
	maxPx := (l.content - indent) * l.f.CellW
	runs := runsOf(b)
	for i := range runs {
		if runs[i].Math == "" {
			continue
		}
		// The engine lays the formula out once per run; without one (or when
		// it cannot parse it) the source stays as text.
		if l.inline == nil {
			runs[i].Math = ""
			continue
		}
		if box := l.inline(runs[i].Math, size); box != nil && box.Draw != nil && box.WidthPx > 0 {
			runs[i].Box = box
		} else {
			runs[i].Math = ""
		}
	}
	lines := l.wrapRuns(runs, maxPx, size)
	lineCells := max(int(math.Ceil(l.f.LineHeightPx(size, false)/float64(l.f.CellH))), 1)
	base := l.baseline(size)
	for _, line := range lines {
		// A formula taller than the text line gets the rows its ink needs, so
		// a fraction never paints over the line below it.
		lineRows := lineCells
		for _, run := range line {
			if run.Box == nil {
				continue
			}
			if need := (base + run.Box.DepthPx + l.f.CellH - 1) / l.f.CellH; need > lineRows {
				lineRows = need
			}
		}
		prefixPx := 0
		for _, run := range line {
			runCol := col
			if run.Link {
				runCol = l.pal.Link
			}
			runSize := size
			if run.Code {
				runSize = size * 0.85
			}
			px := 0
			if run.Box != nil {
				px = run.Box.WidthPx
			} else {
				px = int(l.f.Measure(run.Text, runSize, run.Code)) / 64
			}
			xCells := l.x0 + indent + prefixPx/l.f.CellW
			xOff := prefixPx % l.f.CellW
			// The cells claimed must hold the offset inside the first cell
			// too: otherwise the raster's clip drops the run's last glyph.
			w := max((px+xOff+l.f.CellW-1)/l.f.CellW, 1)
			if run.Box != nil {
				l.emit(l.inlineItem(run, xCells, xOff, w, lineRows, base))
				prefixPx += px
				continue
			}
			if run.Code {
				// The background starts at the run's exact pixel: a rect at
				// the cell's left edge would paint over the word before it.
				l.emit(Item{Kind: ItemRect, X: xCells, Y: l.y, W: w, H: lineCells, Color: l.pal.CodeBg, XOffPx: xOff})
			}
			l.emit(Item{Kind: ItemText, X: xCells, Y: l.y, W: w, H: lineCells, Text: run.Text,
				SizePx: runSize, Mono: run.Code, Bold: run.Bold, Italic: run.Italic,
				Color: runCol, BaselinePx: base, XOffPx: xOff})
			prefixPx += px
		}
		l.y += lineRows
	}
}

// inlineItem turns an inline formula into a page item: a box on the text
// line, drawn with its baseline at the line's baseline and its exact pixel
// offset inside the cells it claims.
func (l *layouter) inlineItem(run styledText, xCells, xOff, w, rows, base int) Item {
	box := run.Box
	// XOffPx stays zero: a box is drawn from its cell's left edge, and the
	// sub-cell offset lives inside Draw, so nothing shifts it twice.
	return Item{Kind: ItemBox, X: xCells, Y: l.y, W: w, H: rows,
		Box: &Box{W: w, H: rows, Draw: func(dst draw.Image, x, y int) {
			box.Draw(dst, x+xOff, y+base)
		}}}
}

// wrapRuns wraps styled runs to maxPx, word by word, keeping each word's
// style and splitting a word that cannot fit on a line of its own. An inline
// maths run is one unbreakable word measured by its ink, whatever spaces its
// source has.
func (l *layouter) wrapRuns(runs []styledText, maxPx int, size float64) [][]styledText {
	type word struct {
		text               string
		bold, italic, code bool
		link               bool
		math               string
		box                *InlineBox
		space              bool // the note has a space before this word
	}
	var words []word
	prevEndedSpace := true // the first word has no space before it
	for _, r := range runs {
		if r.Box != nil {
			words = append(words, word{text: r.Text, math: r.Math, box: r.Box,
				space: len(words) > 0 && prevEndedSpace})
			prevEndedSpace = strings.HasSuffix(r.Text, " ")
			continue
		}
		leading := strings.HasPrefix(r.Text, " ")
		for i, f := range strings.Fields(r.Text) {
			// A space is a space only where the note has one: "`code`." must
			// stay "code.", never "code .", and "**bold** text" keeps the
			// space that ended the previous run.
			space := (i > 0 || leading || prevEndedSpace) && len(words) > 0
			words = append(words, word{text: f, bold: r.Bold, italic: r.Italic, code: r.Code, link: r.Link, space: space})
			prevEndedSpace = false
		}
		prevEndedSpace = strings.HasSuffix(r.Text, " ") || strings.HasSuffix(r.Text, "\n")
	}
	spacePx := func() int { return int(l.f.Measure(" ", size, false)) / 64 }
	widthOf := func(w word) int {
		if w.box != nil {
			return w.box.WidthPx
		}
		return int(l.f.Measure(w.text, size, false)) / 64
	}
	var lines [][]styledText
	var line []styledText
	linePx := 0
	for _, w := range words {
		space := 0
		if w.space && len(line) > 0 {
			space = spacePx()
		}
		width := widthOf(w)
		if len(line) > 0 && linePx+space+width > maxPx {
			lines = append(lines, line)
			line, linePx, space = nil, 0, 0
		}
		if space > 0 {
			line = append(line, styledText{Text: " "})
			linePx += space
		}
		line = append(line, styledText{Text: w.text, Bold: w.bold, Italic: w.italic, Code: w.code,
			Link: w.link, Math: w.math, Box: w.box})
		linePx += width
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}
	return lines
}

func (l *layouter) paragraph(lines []string, indent int, size float64, col color.RGBA) {
	for _, line := range lines {
		wrapped := l.f.Wrap(line, (l.content-indent)*l.f.CellW, size, false)
		l.textLines(wrapped, indent, size, col, false)
	}
}

func (l *layouter) textLines(lines []string, indent int, size float64, col color.RGBA, mono bool) {
	lineCells := max(int(math.Ceil(l.f.LineHeightPx(size, mono)/float64(l.f.CellH))), 1)
	ascent := l.f.AscentPx(size, mono)
	base := int(math.Round((float64(lineCells*l.f.CellH) + ascent - l.f.LineHeightPx(size, mono)/2) / 2))
	for _, line := range lines {
		l.emit(Item{
			Kind: ItemText, X: l.x0 + indent, Y: l.y, W: l.content - indent, H: lineCells,
			Text: line, SizePx: size, Mono: mono, Color: col, BaselinePx: max(base, 0),
		})
		l.y += lineCells
	}
}

func (l *layouter) list(b Block, indent int) {
	if b.Kind == KindListItem {
		l.listItems([]Block{b}, indent)
		return
	}
	l.listItems(b.Blocks, indent)
	if b.Kind == KindParagraph {
		l.y += blockGap
	}
}

// listItems draws every item at indent and then its children one level in, so
// a nested item is never skipped for the sake of its own children.
func (l *layouter) listItems(items []Block, indent int) {
	for _, it := range items {
		text := firstLine(it.Lines)
		bullet := "•"
		if strings.HasPrefix(text, "☐") || strings.HasPrefix(text, "☑") {
			r := []rune(text)[0]
			bullet = string(r)
			text = trimLeftSpace(strings.TrimPrefix(text, string(r)))
		}
		l.emit(Item{Kind: ItemText, X: l.x0 + indent, Y: l.y, W: 2, H: 1, Text: bullet,
			SizePx: l.f.BodyPx, Color: l.pal.Muted, BaselinePx: l.baseline(l.f.BodyPx)})
		wrapped := l.f.Wrap(text, (l.content-indent-listIndent)*l.f.CellW, l.f.BodyPx, false)
		start := l.y
		l.textLines(wrapped, indent+listIndent, l.f.BodyPx, l.pal.Fg, false)
		_ = start
		l.listItems(it.Blocks, indent+listIndent)
	}
}

func (l *layouter) quote(b Block, indent int) {
	start := l.y
	l.y++
	col := l.pal.Muted
	if len(b.Lines) > 0 {
		col = l.calloutColor(b.Lines[0])
	}
	for _, child := range b.Blocks {
		l.block(child, indent+listIndent)
	}
	for i := start; i < l.y; i++ {
		l.emit(Item{Kind: ItemRect, X: l.x0 + indent, Y: i, W: 1, H: 1, Color: col})
	}
	if len(b.Lines) > 0 {
		l.emit(Item{Kind: ItemText, X: l.x0 + indent + listIndent, Y: start, W: l.content - indent,
			Text: b.Lines[0], SizePx: l.f.BodyPx * 0.85, Color: col, BaselinePx: l.baseline(l.f.BodyPx)})
	}
	l.y++
}

func (l *layouter) calloutColor(label string) color.RGBA {
	idx := map[string]int{"NOTE": 0, "TIP": 1, "IMPORTANT": 2, "WARNING": 3, "CAUTION": 4}[label]
	if c := l.pal.Alert[idx]; c.A != 0 {
		return c
	}
	return l.pal.Muted
}

func (l *layouter) code(b Block, indent int) {
	l.gapIfNeeded(0)
	size := l.f.MonoPx
	var lines []string
	if b.Kind == KindCode {
		lines = b.Code
	} else if len(b.Code) > 0 {
		lines = append([]string{l.kindLabel(b)}, b.Code...)
	}
	lines = append(lines, "")
	height := len(lines) + 2*codePadding - 2
	widest := 0
	for _, line := range lines {
		if px := int(l.f.Measure(line, size, true)) / 64; px > widest {
			widest = px
		}
	}
	widthCells := max(l.content-indent, (widest+l.f.CellW-1)/l.f.CellW)
	if widthCells > l.content-indent {
		l.page.Wide = append(l.page.Wide, RowSpan{Row: l.y, Rows: height})
	}
	l.emit(Item{Kind: ItemRect, X: l.x0 + indent, Y: l.y, W: widthCells, H: height, Color: l.pal.CodeBg})
	l.y += codePadding
	for _, line := range lines {
		if line == "" && b.Kind == KindCode {
			l.y++
			continue
		}
		col := l.pal.Fg
		if b.Kind != KindCode && line == lines[0] {
			col = l.pal.Muted
		}
		if b.Lang == "diff" {
			switch {
			case strings.HasPrefix(line, "+"):
				col = l.pal.Alert[1]
			case strings.HasPrefix(line, "-"):
				col = l.pal.Alert[4]
			case strings.HasPrefix(line, "@@"):
				col = l.pal.Link
			}
		}
		l.emit(Item{Kind: ItemText, X: l.x0 + indent + 1, Y: l.y, W: widthCells - 2, H: 1,
			Text: line, SizePx: size, Mono: true, Color: col, BaselinePx: l.baseline(size)})
		l.y++
	}
	l.y += codePadding
	l.y += blockGap
}

// engineBox asks the maths or diagram engine for a sized box, when one is
// installed and can draw the block.
func (l *layouter) engineBox(b Block, indent int) *Box {
	var drawit func(Block) *Box
	switch b.Kind {
	case KindMath:
		drawit = l.math
	case KindDiagram:
		drawit = l.diagram
	}
	if drawit == nil {
		return nil
	}
	box := drawit(b)
	if box == nil || box.W <= 0 || box.H <= 0 || box.Draw == nil {
		return nil
	}
	box.W = min(box.W, l.content-indent)
	return box
}

func (l *layouter) kindLabel(b Block) string {
	if b.Kind == KindMath {
		return "∑ " + firstLine(b.Code) + " (math)"
	}
	name := b.Lang
	if name == "" || name == "flow" {
		name = "mermaid"
	}
	return "◇ " + name + " diagram"
}

// table lays rows out as a simple grid: one header separator and one row per
// entry. Stripes, borders and horizontal scroll come with the blocks task.
func (l *layouter) table(rows [][]string, indent int) {
	if len(rows) == 0 {
		return
	}
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		return
	}
	widths := make([]int, cols)
	for _, r := range rows {
		for i, c := range r {
			px := int(math.Ceil(float64(l.f.Measure(c, l.f.BodyPx, false)) / 64))
			cells := max((px+l.f.CellW-1)/l.f.CellW, 1) + 1 // one blank either side
			widths[i] = max(widths[i], cells)
		}
	}
	total := 0
	for _, w := range widths {
		total += w
	}
	room := l.content - indent
	wide := total > room
	if wide {
		start := l.y
		defer func() { l.page.Wide = append(l.page.Wide, RowSpan{Row: start, Rows: l.y - start}) }()
	} else if cols > 0 {
		widths[cols-1] = max(widths[cols-1]-(total-room), 3)
	}
	for ri, r := range rows {
		x := l.x0 + indent
		if ri > 0 && ri%2 == 0 {
			l.emit(Item{Kind: ItemRect, X: l.x0 + indent, Y: l.y, W: l.content - indent, H: 1, Color: l.pal.Stripe})
		}
		for ci, c := range r {
			if ci >= len(widths) {
				break
			}
			if ci == cols-1 && len(r) == cols {
				c = fitCellPx(l.f, c, max(widths[ci]-1, 1)*l.f.CellW)
			}
			col := l.pal.Fg
			if ri == 0 {
				col = l.pal.Muted
			}
			l.emit(Item{Kind: ItemText, X: x, Y: l.y, W: widths[ci], H: 1, Text: c,
				SizePx: l.f.BodyPx, Color: col, BaselinePx: l.baseline(l.f.BodyPx)})
			x += widths[ci]
		}
		if cols > 1 {
			sx := l.x0 + indent
			for ci := 0; ci < cols-1; ci++ {
				sx += widths[ci]
				l.emit(Item{Kind: ItemRect, X: sx, Y: l.y, W: 1, H: 1, Thin: true, Color: l.pal.Rule})
			}
		}
		l.y++
		if ri == 0 {
			l.emit(Item{Kind: ItemRect, X: l.x0 + indent, Y: l.y, W: l.content - indent, H: 1, Thin: true, Color: l.pal.Rule})
		}
	}
}

// placeholder draws the bordered box the engine tasks replace: images get a
// frame and their alt text, maths and diagrams their source.
func (l *layouter) placeholder(label string, indent, rows int) {
	w := l.content - indent
	l.emit(Item{Kind: ItemRect, X: l.x0 + indent, Y: l.y, W: w, H: 1, Thin: true, Color: l.pal.Rule})
	l.emit(Item{Kind: ItemRect, X: l.x0 + indent, Y: l.y + rows, W: w, H: 1, Thin: true, Color: l.pal.Rule})
	l.emit(Item{Kind: ItemText, X: l.x0 + indent + 1, Y: l.y + 1, W: w - 2, H: 1, Text: label,
		SizePx: l.f.BodyPx, Color: l.pal.Muted, BaselinePx: l.baseline(l.f.BodyPx)})
	l.y += rows + 1
}

func (l *layouter) baseline(size float64) int {
	ascent := l.f.AscentPx(size, false)
	return int(math.Round((float64(l.f.CellH) + ascent - l.f.LineHeightPx(size, false)/2) / 2))
}

func firstLine(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

// fitCellPx cuts s to the pixels the column has, counting glyphs, never
// bytes, and marks the cut with an ellipsis when one fits.
func fitCellPx(f *Fonts, s string, px int) string {
	if px <= 0 || int(f.Measure(s, f.BodyPx, false))/64 <= px {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if px > f.CellW && int(f.Measure(string(runes)+"…", f.BodyPx, false))/64 <= px {
			return string(runes) + "…"
		}
		if int(f.Measure(string(runes), f.BodyPx, false))/64 <= px {
			return string(runes)
		}
	}
	return ""
}

func fitCell(s string, cells int) string {
	if len(s) <= cells {
		return s
	}
	if cells <= 1 {
		return "…"
	}
	return s[:cells-1] + "…"
}

func trimLeftSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	return s
}
