package content

import (
	"sort"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/textarea"
	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// The rendered view: every block of the note is drawn from the preview's
// compile. Only while typing (INSERT) does the cursor's block show its
// Markdown, and with a selection every block it touches does, so they edit
// and select like any editor line; in NORMAL the cursor's block stays
// rendered under the cursor-line band, and j/k step from block to block. The
// app hands the compile over after every event (ShowRendered); this package
// does not import the preview.

// RenderedBlock is one source range of the note and the compiled rows it owns.
type RenderedBlock struct {
	Line, End int      // source lines [Line, End)
	Row       int      // the compile's row of Rows[0]
	Rows      []string // compiled rows, possibly none
	Folded    bool     // a folded section: the lines after the first are hidden
}

// Rendered is one compile of the note, cut by source line.
type Rendered struct {
	Generation uint64 // changes with every compile
	PageID     string
	Failed     bool                   // the compile panicked: the note draws raw
	Current    bool                   // false while a typed edit waits for its compile
	Frozen     bool                   // a block drag: the raw lines stay as they are
	Blocks     func() []RenderedBlock // read only for a new Generation
	Footer     []string               // rows after the last line
	FooterRow  int                    // the compile's row of Footer[0]
}

// renderedState is what ShowRendered last installed.
type renderedState struct {
	on         bool // a display is installed
	page       string
	generation uint64
	version    uint64 // the text blocks were last brought up to
	blocks     []RenderedBlock
	footer     []string
	footerRow  int
	current    bool // blocks are the compile's, not shifted to a newer text
	lines      int  // the lines blocks cover
	row        int  // the cursor's line after the last call
	off        int  // the textarea's scroll after the last call
	from, to   int  // the raw lines [from, to)
	blockwise  bool // j/k step over every line but a block's first
	dim        bool
}

// ShowRendered draws the note from r, greyed when dim is set: every block
// rendered, except in INSERT the cursor's block and with a selection the
// blocks it touches, which are raw. It is cheap to call after every event:
// unless the compile, the text's lines, the raw range, the mode's kind or dim
// changed, it leaves the display alone, so a NORMAL cursor move redraws
// nothing but the band. A failed compile, or one of another note, draws the
// lines raw.
func (m *Model) ShowRendered(r Rendered, dim bool) {
	if r.Failed || r.PageID != m.page.ID {
		m.ClearRendered()
		return
	}
	s := m.rendered
	if s.page != r.PageID {
		s = renderedState{page: r.PageID}
	}
	n, version := m.LineCount(), m.Version()
	row, _ := m.textarea.RowCol()
	resized := s.on && s.lines != n

	changed := false
	switch {
	case r.Current && r.Generation != s.generation:
		s.generation, s.version = r.Generation, version
		s.blocks, s.footer, s.footerRow = r.Blocks(), r.Footer, r.FooterRow
		s.lines = blockLines(s.blocks)
		s.current, changed = true, true
	case version != s.version:
		// The text moved on and its compile has not come yet: the rendered
		// rows stay and the entries follow the lines they were compiled from.
		s.blocks = shiftBlocks(s.blocks, n-s.lines, min(s.row, row))
		s.version, changed = version, n != s.lines
		s.lines, s.current = n, false
	}
	if len(s.blocks) == 0 || s.lines != n {
		m.ClearRendered() // nothing that says how to draw these lines
		return
	}

	revealing := m.mode == ModeInsert || m.textarea.HasSelection()
	from, to := s.from, s.to
	if (!m.dragging && !r.Frozen) || !s.on { // a drag keeps the rows still under the pointer
		from, to = 0, 0
		if revealing {
			from, to = m.revealRange(s.blocks, row)
		}
	}
	from, to = min(from, n), min(to, n)
	if !s.on || changed || from != s.from || to != s.to || dim != s.dim || s.blockwise == revealing {
		s.from, s.to, s.dim, s.blockwise = from, to, dim, !revealing
		m.textarea.SetDisplay(s.display(n, m.theme.Grey, m.blankLines))
		if resized {
			m.keepScroll(s.off)
		}
		s.on = true
	}
	s.row, s.off = row, m.textarea.YOffset()
	m.rendered = s
}

// ClearRendered goes back to drawing every line raw.
func (m *Model) ClearRendered() {
	m.rendered = renderedState{}
	m.textarea.SetDisplay(nil)
}

// keepScroll puts the scroll back to off after an edit changed the line count.
// The text above the edit did not move, so off still shows it; while the line
// count disagreed the textarea measured the scroll in the wrong geometry. The
// cursor stays on screen.
func (m *Model) keepScroll(off int) {
	_, y, ok := m.textarea.CursorScreen()
	if !ok {
		return
	}
	now := m.textarea.YOffset()
	cursor := now + y
	m.textarea.ScrollBy(max(min(off, cursor), cursor-m.textarea.Height()+1) - now)
}

// revealRange returns the lines [from, to) drawn raw for a cursor on row:
// every block the selection, or the cursor, touches. In a run of blank lines
// only the touched lines go raw, and in a folded section only its first line
// unless the body is touched.
func (m Model) revealRange(blocks []RenderedBlock, row int) (from, to int) {
	first, last := row, row
	if m.textarea.HasSelection() {
		first, last = m.textarea.SelectedRows()
	}
	from, to = first, last+1
	for _, b := range blocks {
		if b.Line > last {
			break
		}
		if b.End <= first || (b.Folded && last == b.Line) || m.blankLines(b) {
			continue
		}
		from, to = min(from, b.Line), max(to, b.End)
	}
	return from, to
}

// blankLines reports whether b holds nothing but blank lines.
func (m Model) blankLines(b RenderedBlock) bool {
	for l := b.Line; l < b.End; l++ {
		if strings.TrimSpace(m.textarea.RowText(l)) != "" {
			return false
		}
	}
	return true
}

// display builds the textarea's display for n lines. Every line of a block
// names the block's first line as its Head, which holds the rows. Blockwise,
// j/k skip every other line of a block and the whole of a blank run.
func (s renderedState) display(n int, grey lipgloss.Color, blank func(RenderedBlock) bool) *textarea.Display {
	paint := func(rows []string) []string {
		if !s.dim {
			return rows
		}
		out := make([]string, len(rows))
		for i, row := range rows {
			out[i] = ui.Dim(row, grey)
		}
		return out
	}
	lines := make([]textarea.DisplayLine, n)
	for _, b := range s.blocks {
		skip := s.blockwise && blank(b)
		for l := b.Line; l < b.End; l++ {
			lines[l].Raw = l >= s.from && l < s.to
			lines[l].Head = b.Line
			lines[l].Hidden = !lines[l].Raw && (b.Folded && l > b.Line || s.blockwise && l > b.Line || skip)
		}
		if !lines[b.Line].Raw {
			lines[b.Line].Rows = paint(b.Rows)
		}
	}
	return &textarea.Display{Lines: lines, Footer: paint(s.footer)}
}

// cursorBlock returns the source lines [from, to) of the rendered block the
// cursor is on, for the NORMAL commands that act on the whole block. A raw
// line, a blank run and the editor without a display report false: there
// the commands keep acting on the line.
func (m Model) cursorBlock() (from, to int, ok bool) {
	s := m.rendered
	row, _ := m.textarea.RowCol()
	if !s.on || m.mode != ModeNormal || s.lines != m.LineCount() || (row >= s.from && row < s.to) {
		return 0, 0, false
	}
	i := sort.Search(len(s.blocks), func(i int) bool { return s.blocks[i].End > row })
	if i == len(s.blocks) || s.blocks[i].Line > row || m.blankLines(s.blocks[i]) {
		return 0, 0, false
	}
	return s.blocks[i].Line, s.blocks[i].End, true
}

// lastBlockLine is where G lands while j/k go by block: the first line of
// the last block that is not a blank run, or the last line of its body when
// that block is a folded section, which then opens as a goto into it does.
func (m Model) lastBlockLine() (int, bool) {
	s := m.rendered
	if !s.on || !s.blockwise || s.lines != m.LineCount() {
		return 0, false
	}
	for i := len(s.blocks) - 1; i >= 0; i-- {
		switch b := s.blocks[i]; {
		case m.blankLines(b):
		case b.Folded:
			return b.End - 1, true
		default:
			return b.Line, true
		}
	}
	return 0, false
}

// blockText is the Markdown of lines [from, to), each ending in a newline:
// what a linewise yank of them holds.
func (m Model) blockText(from, to int) string {
	var b strings.Builder
	for l := from; l < to; l++ {
		b.WriteString(m.textarea.RowText(l))
		b.WriteByte('\n')
	}
	return b.String()
}

// shiftable reports whether >> and << act on the rendered block starting at
// line from: only a list item, which they nest and un-nest. Any other block
// keeps its indent, since two more spaces either change nothing or, from
// four, turn it into indented code; front matter and fenced code never move.
func (m Model) shiftable(from int) bool {
	return listItem.MatchString(m.textarea.RowText(from))
}

// shiftBlock indents (in) or outdents every line of [from, to) by two
// spaces, as Tab nests a list line.
func (m *Model) shiftBlock(from, to int, in bool) {
	m.textarea.MapLines(from, to-1, func(l string) string {
		if in {
			return "  " + l
		}
		for range 2 {
			l = strings.TrimPrefix(l, " ")
		}
		return l
	})
}

// blankAt reports whether line l exists and holds nothing but spaces.
func (m Model) blankAt(l int) bool {
	return l >= 0 && l < m.LineCount() && strings.TrimSpace(m.textarea.RowText(l)) == ""
}

// frontMatter reports whether the rendered block [from, to) is the note's
// front matter, above which nothing may go.
func (m Model) frontMatter(from, to int) bool {
	if from != 0 {
		return false
	}
	lines := make([]string, to)
	for l := range lines {
		lines[l] = m.textarea.RowText(l)
	}
	return core.FrontMatterEnd(lines) >= 0
}

// newBlock puts lines in as a block of their own beside the rendered block
// [from, to): below it, or above it when above is set and the block is not
// the front matter. A blank line keeps them apart from the block and from
// what is on their other side, added only where none is there already, so
// they never join the block or the next one (a table row, a paragraph's
// continuation). The cursor goes to their first line.
func (m *Model) newBlock(from, to int, above bool, lines []string) {
	ta := &m.textarea
	at := to // the new lines go in before line at
	if above && !m.frontMatter(from, to) {
		at = from
	}
	var b []string
	if at > 0 && !m.blankAt(at-1) {
		b = append(b, "")
	}
	start := at + len(b)
	b = append(b, lines...)
	if at < m.LineCount() && !m.blankAt(at) {
		b = append(b, "")
	}
	if at < m.LineCount() {
		ta.SetRowCol(at, 0)
		ta.OpenLineAbove()
	} else {
		ta.SetRowCol(at-1, len(ta.RowText(at-1)))
		ta.OpenLineBelow()
	}
	ta.InsertString(strings.Join(b, "\n"))
	ta.SetRowCol(start, 0)
}

// deleteBlock deletes the rendered block [from, to) and one of the blank
// lines that kept it apart, so the blocks around stay one blank line apart,
// and puts the cursor on the next block, or on the previous one when none
// follows: never on a blank line.
func (m *Model) deleteBlock(from, to int) {
	blocks := m.rendered.blocks // the old text's; lines before from keep their place
	ta := &m.textarea
	ta.DeleteLines(from, to)
	switch {
	case m.blankAt(from) && (from == 0 || m.blankAt(from-1)):
		ta.DeleteLines(from, from+1)
	case from >= m.LineCount() && m.blankAt(from-1):
		ta.DeleteLines(from-1, from)
	}
	for l := from; l < m.LineCount(); l++ {
		if !m.blankAt(l) {
			ta.SetRowCol(l, 0)
			return
		}
	}
	for l := min(from, m.LineCount()) - 1; l >= 0; l-- {
		if m.blankAt(l) {
			continue
		}
		i := sort.Search(len(blocks), func(i int) bool { return blocks[i].End > l })
		if i < len(blocks) && blocks[i].Line <= l {
			l = blocks[i].Line
		}
		ta.SetRowCol(l, 0)
		return
	}
}

// blockLines is the number of lines blocks cover, which are contiguous.
func blockLines(blocks []RenderedBlock) int {
	if len(blocks) == 0 {
		return 0
	}
	return blocks[len(blocks)-1].End
}

// shiftBlocks moves the blocks' boundaries past line anchor by delta, when the
// text gained or lost lines there. Blocks left without lines go.
func shiftBlocks(blocks []RenderedBlock, delta, anchor int) []RenderedBlock {
	if delta == 0 {
		return blocks
	}
	at := func(line int) int {
		if line <= anchor {
			return line
		}
		return max(line+delta, anchor+1)
	}
	out := make([]RenderedBlock, 0, len(blocks))
	for _, b := range blocks {
		b.Line, b.End = at(b.Line), at(b.End)
		if b.Line < b.End {
			out = append(out, b)
		}
	}
	return out
}
