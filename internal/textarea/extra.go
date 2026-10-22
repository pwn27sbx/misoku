package textarea

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// misoku: helpers that are not part of the upstream bubbles textarea API.

// YOffset returns the first visible visual (soft-wrapped) line.
func (m Model) YOffset() int {
	return m.viewport.YOffset
}

// EnsureVisible scrolls so the cursor is on screen. Unlike the upstream
// repositionView it does not clamp against the content rendered by the last
// View call, so it also works right after SetValue on a long document.
func (m *Model) EnsureVisible() {
	off := m.viewport.YOffset
	h := max(m.viewport.Height, 1)
	row := m.cursorLineNumber()
	if row < off {
		off = row
	} else if row > off+h-1 {
		off = row - h + 1
	}
	m.viewport.YOffset = max(off, 0)
}

// visualLineCount returns the number of soft-wrapped lines in the buffer.
func (m Model) visualLineCount() int {
	starts := m.rows()
	return starts[len(starts)-1]
}

// CursorPosition returns the 1-based logical line and column of the cursor.
func (m Model) CursorPosition() (line, col int) {
	return m.row + 1, m.col + 1
}

// moveToVisual places the cursor on visual line v, at display column col
// within that wrapped segment (clamped to the segment's length).
func (m *Model) moveToVisual(v, col int) {
	v = max(v, 0)
	r := m.lineAtRow(v)
	if r >= len(m.value) {
		// Past the end: last character of the buffer.
		m.row = len(m.value) - 1
		m.col = len(m.value[m.row])
		return
	}
	l := m.value[r]
	segs := m.memoizedWrap(l, m.width)
	k := v - m.rows()[r] // wrapped segment within the line
	start := 0
	for i := 0; i < k; i++ {
		start += len(segs[i])
	}
	segLen := len(segs[k])
	// A wrapped segment (not the last) ends where the next begins;
	// keep the cursor on this segment rather than jumping down.
	if k < len(segs)-1 && segLen > 0 {
		segLen--
	}
	m.row = r
	// The last wrapped segment carries a padding space, so clamp
	// to the real line length as well.
	m.col = min(start+clamp(col, 0, segLen), len(l))
}

// ScrollBy scrolls the view by n visual lines (negative scrolls up) without
// requiring focus, dragging the cursor along only when it would leave the
// screen — the same feel as scrolling a Vim window with the mouse.
func (m *Model) ScrollBy(n int) {
	total := m.visualLineCount()
	maxOff := max(0, total-1)
	off := clamp(m.viewport.YOffset+n, 0, maxOff)
	m.viewport.YOffset = off

	top := off
	bottom := off + m.viewport.Height - 1
	cur := m.cursorLineNumber()
	col := m.LineInfo().ColumnOffset
	switch {
	case cur < top:
		m.moveToVisual(top, col)
	case cur > bottom:
		m.moveToVisual(bottom, col)
	}
}

// ClickAt moves the cursor to the cell clicked at (x, y), relative to the
// textarea's top-left corner, accounting for the prompt and line-number gutter.
func (m *Model) ClickAt(x, y int) {
	gutter := m.promptWidth
	if m.ShowLineNumbers {
		gutter += 6
	}
	m.moveToVisual(m.viewport.YOffset+y, x-gutter)
	m.EnsureVisible()
}

// MoveCursorBy moves the cursor by n logical lines and keeps it in view.
func (m *Model) MoveCursorBy(n int) {
	for ; n > 0; n-- {
		m.CursorDown()
	}
	for ; n < 0; n++ {
		m.CursorUp()
	}
	m.EnsureVisible()
}

// GotoLine moves the cursor to the start of 1-based line n.
func (m *Model) GotoLine(n int) {
	m.row = clamp(n-1, 0, len(m.value)-1)
	m.SetCursor(0)
	m.EnsureVisible()
}

// CharLeft / CharRight move the cursor one character within the line.
func (m *Model) CharLeft() {
	if m.col > 0 {
		m.SetCursor(m.col - 1)
	}
}

// CharRight moves the cursor one character right, stopping at line end.
func (m *Model) CharRight() {
	if m.col < len(m.value[m.row]) {
		m.SetCursor(m.col + 1)
	}
}

// WordForward moves the cursor to the start of the next word.
func (m *Model) WordForward() {
	m.wordRight()
	m.EnsureVisible()
}

// WordBackward moves the cursor to the start of the previous word.
func (m *Model) WordBackward() {
	m.wordLeft()
	m.EnsureVisible()
}

// GotoTop moves the cursor to the first line.
func (m *Model) GotoTop() {
	m.moveToBegin()
	m.EnsureVisible()
}

// GotoBottom moves the cursor to the start of the last line.
func (m *Model) GotoBottom() {
	m.row = len(m.value) - 1
	m.SetCursor(0)
	m.EnsureVisible()
}

// DeleteCharForward deletes the character under the cursor (Vim's x).
func (m *Model) DeleteCharForward() {
	line := m.value[m.row]
	if m.col < len(line) {
		m.value[m.row] = append(line[:m.col:m.col], line[m.col+1:]...)
		m.touch()
		if m.col > 0 && m.col >= len(m.value[m.row]) {
			m.col = len(m.value[m.row]) - 1
		}
	}
}

// DeleteLine removes the cursor's line (Vim's dd).
func (m *Model) DeleteLine() {
	if len(m.value) == 1 {
		m.value[0] = m.value[0][:0]
		m.touch()
		m.col = 0
		return
	}
	m.value = append(m.value[:m.row:m.row], m.value[m.row+1:]...)
	m.touch()
	if m.row >= len(m.value) {
		m.row = len(m.value) - 1
	}
	m.SetCursor(0)
	m.EnsureVisible()
}

// OpenLineBelow inserts an empty line below the cursor and moves onto it (Vim's o).
func (m *Model) OpenLineBelow() {
	m.CursorEnd()
	m.splitLine(m.row, m.col)
	m.EnsureVisible()
}

// OpenLineAbove inserts an empty line above the cursor and moves onto it (Vim's O).
func (m *Model) OpenLineAbove() {
	m.CursorStart()
	m.splitLine(m.row, 0)
	m.row--
	m.col = 0
	m.EnsureVisible()
}

// CursorScreen returns the cursor cell relative to the textarea's top-left
// corner (including the line-number gutter), or ok=false when off screen.
func (m Model) CursorScreen() (x, y int, ok bool) {
	gutter := m.promptWidth
	if m.ShowLineNumbers {
		gutter += 6
	}
	y = m.cursorLineNumber() - m.viewport.YOffset
	if y < 0 || y >= m.viewport.Height {
		return 0, 0, false
	}
	return gutter + m.LineInfo().CharOffset, y, true
}

// RowCol returns the 0-based logical row and rune column of the cursor.
func (m Model) RowCol() (int, int) { return m.row, m.col }

// SetRowCol moves the cursor to a 0-based logical row and rune column.
func (m *Model) SetRowCol(row, col int) {
	m.row = clamp(row, 0, len(m.value)-1)
	m.col = clamp(col, 0, len(m.value[m.row]))
	m.lastCharOffset = 0
	m.EnsureVisible()
}

// LineBeforeCursor returns the text of the cursor's line up to the cursor.
func (m Model) LineBeforeCursor() string {
	return string(m.value[m.row][:clamp(m.col, 0, len(m.value[m.row]))])
}

// DeleteBefore removes n runes before the cursor on the current line.
func (m *Model) DeleteBefore(n int) {
	line := m.value[m.row]
	n = clamp(n, 0, m.col)
	m.value[m.row] = append(line[:m.col-n:m.col-n], line[m.col:]...)
	m.touch()
	m.col -= n
}

// IndentLine prepends n spaces to the cursor's line, keeping the cursor on
// the same character.
func (m *Model) IndentLine(n int) {
	pad := make([]rune, n)
	for i := range pad {
		pad[i] = ' '
	}
	m.value[m.row] = append(pad, m.value[m.row]...)
	m.touch()
	m.col += n
}

// OutdentLine removes up to n leading spaces from the cursor's line.
func (m *Model) OutdentLine(n int) {
	line := m.value[m.row]
	k := 0
	for k < n && k < len(line) && line[k] == ' ' {
		k++
	}
	m.value[m.row] = append([]rune(nil), line[k:]...)
	m.touch()
	m.col = max(m.col-k, 0)
}

// CurrentLine returns the cursor's whole line.
func (m Model) CurrentLine() string { return string(m.value[m.row]) }

// StartSelection anchors a selection at the cursor.
func (m *Model) StartSelection() {
	m.selecting = true
	m.anchorRow, m.anchorCol = m.row, m.col
}

// SetAnchor starts a selection anchored at row/col (the cursor is the
// other end).
func (m *Model) SetAnchor(row, col int) {
	m.selecting = true
	m.anchorRow, m.anchorCol = row, col
}

// ClearSelection drops any selection.
func (m *Model) ClearSelection() { m.selecting = false }

// HasSelection reports whether a selection is active.
func (m Model) HasSelection() bool { return m.selecting }

// selectionRange returns the selection as ordered, inclusive positions.
func (m Model) selectionRange() (r1, c1, r2, c2 int) {
	r1, c1, r2, c2 = m.anchorRow, m.anchorCol, m.row, m.col
	if r2 < r1 || (r2 == r1 && c2 < c1) {
		r1, c1, r2, c2 = r2, c2, r1, c1
	}
	r1 = clamp(r1, 0, len(m.value)-1)
	r2 = clamp(r2, 0, len(m.value)-1)
	if m.SelectLinewise {
		c1, c2 = 0, max(len(m.value[r2])-1, 0)
	}
	return
}

// inRange reports whether (row, col) falls inside the ordered, inclusive
// range r1:c1..r2:c2 returned by selectionRange.
func inRange(r1, c1, r2, c2, row, col int) bool {
	switch {
	case row < r1 || row > r2:
		return false
	case r1 == r2:
		return col >= c1 && col <= c2
	case row == r1:
		return col >= c1
	case row == r2:
		return col <= c2
	}
	return true
}

// SelectedText returns the selected text (the end character included, as
// in Vim's visual mode).
func (m Model) SelectedText() string {
	if !m.selecting {
		return ""
	}
	r1, c1, r2, c2 := m.selectionRange()
	var b strings.Builder
	for r := r1; r <= r2; r++ {
		line := m.value[r]
		from, to := 0, len(line)
		if r == r1 {
			from = min(c1, len(line))
		}
		if r == r2 {
			to = min(c2+1, len(line))
		}
		if m.SelectLinewise {
			to = len(line)
		}
		if from < to {
			b.WriteString(string(line[from:to]))
		}
		if r < r2 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// DeleteSelection removes the selected text and clears the selection.
func (m *Model) DeleteSelection() {
	if !m.selecting {
		return
	}
	r1, c1, r2, c2 := m.selectionRange()
	if m.SelectLinewise {
		if r2-r1+1 >= len(m.value) {
			m.value = [][]rune{{}}
		} else {
			m.value = append(m.value[:r1], m.value[r2+1:]...)
		}
		m.touch()
		m.row, m.col = min(r1, len(m.value)-1), 0
		m.selecting = false
		m.EnsureVisible()
		return
	}
	head := append([]rune(nil), m.value[r1][:min(c1, len(m.value[r1]))]...)
	tail := []rune{}
	if c2+1 < len(m.value[r2]) {
		tail = append(tail, m.value[r2][c2+1:]...)
	}
	merged := append(head, tail...)
	m.value = append(m.value[:r1], append([][]rune{merged}, m.value[r2+1:]...)...)
	m.touch()
	m.row, m.col = r1, min(c1, len(merged))
	m.selecting = false
	m.EnsureVisible()
}

// paint renders runes of line l starting at rune offset off, applying
// LineColors on top of style.
func (m *Model) paint(style lipgloss.Style, l, off int, runes []rune) string {
	if !m.selecting && (l >= len(m.LineColors) || len(m.LineColors[l]) == 0) {
		return style.Render(string(runes))
	}
	var colors []lipgloss.Color
	if l < len(m.LineColors) {
		colors = m.LineColors[l]
	}
	color := func(i int) lipgloss.Color {
		if i := off + i; i >= 0 && i < len(colors) {
			return colors[i]
		}
		return ""
	}
	// The selection range is computed once per call, not once per rune.
	var r1, c1, r2, c2 int
	if m.selecting {
		r1, c1, r2, c2 = m.selectionRange()
	}
	sel := func(i int) bool { return m.selecting && inRange(r1, c1, r2, c2, l, off+i) }
	var b strings.Builder
	start := 0
	for i := 1; i <= len(runes); i++ {
		if i < len(runes) && color(i) == color(start) && sel(i) == sel(start) {
			continue
		}
		st := style
		if c := color(start); c != "" {
			st = st.Foreground(c)
		}
		if sel(start) {
			st = st.Inherit(m.SelectionStyle).Background(m.SelectionStyle.GetBackground())
		}
		b.WriteString(st.Render(string(runes[start:i])))
		start = i
	}
	return b.String()
}
