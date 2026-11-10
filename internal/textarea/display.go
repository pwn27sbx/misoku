package textarea

import (
	"strings"
	"sync/atomic"

	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// misoku: a display replaces the soft-wrap's geometry line by line, so a note
// can be drawn as rendered blocks of any height and stay editable. The text
// never changes; every mapping between rows and lines reads the row index,
// and the row index reads the display.

// displayGenerations hands out display generations. It is global for the same
// reason versions is: copies of a Model share one row index.
var displayGenerations atomic.Uint64

// DisplayLine says how one logical line is drawn.
type DisplayLine struct {
	Raw    bool     // the line's normal path: wrap, syntax colours, cursor, selection
	Rows   []string // when !Raw: exactly these rows, possibly none
	Hidden bool     // !Raw, and j/k step over it (a folded body, a block's other lines)
	// Head is the line holding the rows of this line's block, for a line
	// without rows of its own: the cursor there is measured, kept on screen
	// and banded as that block. It counts only when it is an earlier line
	// drawn with rows; otherwise the line stands for itself.
	Head int
}

// Display is how the whole note is drawn while it is set.
type Display struct {
	Generation uint64        // stamped by SetDisplay from a package-level counter
	Lines      []DisplayLine // one per logical line; a length mismatch means the display is ignored
	Footer     []string      // rows after the last line, counted in the row index and drawn by View
}

// SetDisplay installs d, or goes back to the plain wrap when d is nil. The
// text at the top of the viewport stays there: the first line on screen that
// still has rows keeps its screen row, so the offset moves by the height
// change of the lines above it. Then the cursor is brought into view.
//
// SetDisplay stamps d.Generation, so a display built twice from the same
// compile (another reveal, say) is still a new geometry to the row index.
func (m *Model) SetDisplay(d *Display) {
	if m.display == nil && d == nil {
		return
	}
	before := m.rows()
	off := m.viewport.YOffset
	top := m.lineAtRow(off)
	within := off - before[min(top, len(m.value))]

	if d != nil {
		d.Generation = displayGenerations.Add(1)
	}
	m.display = d

	after := m.rows()
	if top < len(m.value) {
		anchor := m.survivingLine(before, after, top, off+max(m.viewport.Height, 1))
		if anchor == top {
			within = clamp(within, 0, max(after[top+1]-after[top]-1, 0))
			off = after[top] + within
		} else {
			// The top line lost its rows: the first line that keeps some
			// stays on the screen row it had.
			off = after[anchor] - (before[anchor] - off)
		}
	} else {
		off = after[len(m.value)] + within
	}
	m.viewport.YOffset = clamp(off, 0, max(after[len(after)-1]-1, 0))
	m.EnsureVisible()
}

// survivingLine returns the first line from top, starting above row end in
// the old geometry before, that has rows in both before and after: the line
// whose screen row SetDisplay keeps. It returns top when none does.
func (m Model) survivingLine(before, after []int, top, end int) int {
	for l := top; l < len(m.value) && before[l] < end; l++ {
		if before[l+1] > before[l] && after[l+1] > after[l] {
			return l
		}
	}
	return top
}

// DisplayRowAt maps visual row v to its logical line and the row within it. A
// footer row returns the last line that is not hidden, its row within the
// footer and footer=true. A row past everything returns that same line with
// an offset past the line's own rows. Without a display it follows the wrap.
func (m Model) DisplayRowAt(v int) (line, offset int, footer bool) {
	v = max(v, 0)
	starts := m.rows()
	if l := m.lineAtRow(v); l < len(m.value) {
		return l, v - starts[l], false
	}
	line = m.lastShownLine()
	if v < starts[len(starts)-1] {
		return line, v - starts[len(m.value)], true
	}
	return line, v - starts[line], false
}

// LineRows returns the first visual row of line l and how many rows it has;
// l == LineCount() gives the footer's.
func (m Model) LineRows(l int) (first, n int) {
	starts := m.rows()
	l = clamp(l, 0, len(m.value))
	if l+1 >= len(starts) {
		return starts[l], 0
	}
	return starts[l], starts[l+1] - starts[l]
}

// activeDisplay returns the display when it applies to the current text.
func (m Model) activeDisplay() *Display {
	if m.display == nil || len(m.display.Lines) != len(m.value) {
		return nil
	}
	return m.display
}

// displayGeneration is the row index's key for the active display.
func (m Model) displayGeneration() uint64 {
	if d := m.activeDisplay(); d != nil {
		return d.Generation
	}
	return 0
}

// displayed returns line l's entry when the display draws it instead of the
// wrap, and nil when it takes the normal path.
func (m Model) displayed(l int) *DisplayLine {
	d := m.activeDisplay()
	if d == nil || d.Lines[l].Raw {
		return nil
	}
	return &d.Lines[l]
}

// hidden reports whether j/k step over line l.
func (m Model) hidden(l int) bool {
	d := m.displayed(l)
	return d != nil && d.Hidden
}

// nextShownLine returns the first line after from, in direction dir, that is
// not hidden, or -1 when there is none.
func (m Model) nextShownLine(from, dir int) int {
	for l := from + dir; l >= 0 && l < len(m.value); l += dir {
		if !m.hidden(l) {
			return l
		}
	}
	return -1
}

// lastShownLine returns the last line that is not hidden.
func (m Model) lastShownLine() int {
	l := len(m.value) - 1
	for l > 0 && m.hidden(l) {
		l--
	}
	return l
}

// head returns the line holding the rows of line l's block: its Head when l
// is drawn without rows of its own and Head is an earlier line drawn with
// rows, and l itself otherwise.
func (m Model) head(l int) int {
	d := m.displayed(l)
	if d == nil || len(d.Rows) > 0 || d.Head < 0 || d.Head >= l {
		return l
	}
	if h := m.displayed(d.Head); h != nil && len(h.Rows) > 0 {
		return d.Head
	}
	return l
}

// cursorRows returns the visual rows of the cursor: one, except on a line
// drawn by the display, whose block's rows all count as the cursor being on
// screen, also from a line of the block without rows of its own.
func (m Model) cursorRows() (first, last int) {
	if h := m.head(m.row); h != m.row {
		first = m.rows()[h]
		return first, first + len(m.displayed(h).Rows) - 1
	}
	first = m.cursorLineNumber()
	last = first
	if d := m.displayed(m.row); d != nil && len(d.Rows) > 1 {
		last = first + len(d.Rows) - 1
	}
	return first, last
}

// offsetShowing returns the scroll that brings the cursor's rows on screen
// from off, moving as little as it can: rows above the top go to the top;
// rows below the bottom come up until the last is on the bottom row, or the
// first on the top row for a block taller than the screen.
func (m Model) offsetShowing(off int) int {
	h := max(m.viewport.Height, 1)
	first, last := m.cursorRows()
	switch {
	case last < off:
		return first
	case first > off && last > off+h-1:
		return min(first, last-h+1)
	}
	return off
}

// bandLine returns the line whose display rows take the cursor-line band,
// the head of the cursor's rendered block, or -1: the band is the focused
// editor's, and a selection hides it as it hides the raw line's.
func (m Model) bandLine() int {
	if !m.focus || m.bandHidden() {
		return -1
	}
	h := m.head(m.row)
	if m.displayed(h) == nil {
		return -1
	}
	return h
}

// bandRow paints row with the cursor-line band, to the text's width.
func (m Model) bandRow(row string) string {
	bg, ok := m.style.computedCursorLine().GetBackground().(lipgloss.Color)
	if !ok {
		return row
	}
	return ui.Band(row, bg, m.width)
}

// writeDisplayRow writes one row View does not wrap itself: the prompt for
// visual row v, the line-number cell, the row and its padding.
func (m Model) writeDisplayRow(s *strings.Builder, v int, cell, row string) {
	style := m.style.computedText()
	s.WriteString(style.Render(m.style.computedPrompt().Render(m.getPromptString(v))))
	s.WriteString(cell)
	s.WriteString(row)
	s.WriteString(style.Render(strings.Repeat(" ", max(0, m.width-ansi.StringWidth(row)))))
	s.WriteRune('\n')
}

// displayCell returns the line-number cell of row wl of line l, or of a footer
// row when l is negative.
func (m Model) displayCell(l, wl int) string {
	if !m.ShowLineNumbers {
		return ""
	}
	style := m.style.computedText()
	if l < 0 {
		return style.Render(m.style.computedLineNumber().Render(m.formatLineNumber(" ")))
	}
	return m.lineNumberCell(style, l, wl)
}

// CursorRendered reports whether the cursor's line is drawn by the display,
// where no cursor cell is drawn and the terminal's cursor has no column.
func (m Model) CursorRendered() bool { return m.displayed(m.row) != nil }
