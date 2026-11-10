package content

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/textarea"
	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// The rendered view: every block of the note is drawn from the preview's
// compile, except the block holding the cursor or the selection, which stays
// Markdown and edits like any editor line. The app hands the compile over
// after every event (ShowRendered); this package does not import the preview.

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
	dim        bool
}

// ShowRendered draws the note from r, the block under the cursor or selection
// raw and the others rendered, greyed when dim is set. It is cheap to call
// after every event: unless the compile, the text's lines, the raw range or
// dim changed, it leaves the display alone. A failed compile, or one of
// another note, draws the lines raw.
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

	from, to := s.from, s.to
	if (!m.dragging && !r.Frozen) || !s.on { // a drag keeps the rows still under the pointer
		from, to = m.revealRange(s.blocks, row)
	}
	from, to = min(from, n), min(to, n)
	if !s.on || changed || from != s.from || to != s.to || dim != s.dim {
		s.from, s.to, s.dim = from, to, dim
		m.textarea.SetDisplay(s.display(n, m.theme.Grey))
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

// display builds the textarea's display for n lines.
func (s renderedState) display(n int, grey lipgloss.Color) *textarea.Display {
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
		for l := b.Line; l < b.End; l++ {
			lines[l].Raw = l >= s.from && l < s.to
			lines[l].Hidden = b.Folded && l > b.Line && !lines[l].Raw
		}
		if !lines[b.Line].Raw {
			lines[b.Line].Rows = paint(b.Rows)
		}
	}
	return &textarea.Display{Lines: lines, Footer: paint(s.footer)}
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
