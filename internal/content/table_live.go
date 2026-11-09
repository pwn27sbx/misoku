package content

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Live alignment of a pipe table while its cell is being typed in.
//
// Without it, typing lands before the cell's padding, so the closing pipe
// drifts right until the next Tab; see MISOKU-05. It reuses formatTable, the
// one formatter, and rewrites only the table's lines through MapLines.

// alignTableWhileTyping realigns the table holding the cursor after a key
// that changed the text in INSERT. It never records a change, so the whole
// INSERT session stays one undo step, and it leaves the cursor in the same
// cell at the same offset within the cell's text.
func (m *Model) alignTableWhileTyping(k tea.KeyMsg) {
	if !tableTypingKey(k) || m.slash != nil {
		return
	}
	m.alignCursorTable()
}

// tableTypingKey reports whether k inserts or deletes text one key at a
// time. A paste may open a new line or a pipe, which is Tab's or
// :tablefmt's decision, not the live path's; a typed pipe is skipped too, so
// it stays text until then (A5).
func tableTypingKey(k tea.KeyMsg) bool {
	switch k.Type {
	case tea.KeyRunes:
		return !k.Paste && len(k.Runes) == 1 && k.Runes[0] != '|'
	case tea.KeyBackspace, tea.KeyDelete:
		return true
	}
	return false
}

// tableRowInFence reports whether row sits inside a fenced code block. A
// note without a fence line anywhere cannot hold one, and that common case
// skips the row-by-row walk inFence does from the top of the document.
func (m *Model) tableRowInFence(row int) bool {
	v := m.textarea.Value()
	if !strings.Contains(v, "```") && !strings.Contains(v, "~~~") {
		return false
	}
	return inFence(m.textarea.RowText, row)
}

// alignCursorTable reformats the table at the cursor, if there is one and it
// is not inside a fenced code block, and restores the cursor in its cell.
func (m *Model) alignCursorTable() {
	row, col := m.textarea.RowCol()
	text := m.textarea.RowText
	if !isTableLine(text(row)) {
		return
	}
	// Walk the table's own rows only: a long note must not pay for a split of
	// the whole document per key.
	start := row
	for start > 0 && isTableLine(text(start-1)) {
		start--
	}
	end := row + 1
	for end < m.textarea.LineCount() && isTableLine(text(end)) {
		end++
	}
	if m.tableRowInFence(row) {
		return
	}
	cells := splitCells(text(row))
	if isSeparator(cells) {
		return
	}
	rows := make([][]string, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, splitCells(text(i)))
	}
	// The separator row declares the table's columns; without one, the
	// widest other row does, so a delimiter typed in this row is still
	// undecided.
	cols := 0
	if len(rows) > 1 && isSeparator(rows[1]) {
		cols = len(rows[1])
	} else {
		for i, cells := range rows {
			if i != row-start {
				cols = max(cols, len(cells))
			}
		}
		if cols == 0 { // a single-row table: its own count
			cols = len(cells)
		}
	}
	// More cells than the separator, or a cursor in a cell past its columns,
	// means the text is outside the table: a "|" was typed there (Tab or
	// :tablefmt decides that column), or the row's last pipe already closed
	// it. Either way the row is left exactly as typed, and a row whose last
	// cell simply has no trailing pipe is still a cell.
	c := cellIndex(text(row), col)
	if len(cells) > cols || c >= cols {
		return
	}
	cs := cellStart(text(row), c)
	if col < cs {
		return // before the cell's text, e.g. on the opening pipe
	}
	offset := col - cs
	formatted := formatTable(rows)
	i := 0
	if !m.textarea.MapLines(start, end-1, func(string) string {
		s := formatted[i]
		i++
		return s
	}) {
		return
	}
	// splitCells trims, so the offset may sit in the new padding: restore it
	// without clamping to the cell's text length, so a typed space survives
	// and the next key lands after it (A1). The row does not change, so
	// SetCursor is enough: SetRowCol would scroll the whole document to find
	// the same line again.
	m.textarea.SetCursor(cellStart(m.textarea.RowText(row), c) + offset)
}
