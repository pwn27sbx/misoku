package preview

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Blocks can be dragged by the "⠿" handle that appears in the left margin
// next to the block under the mouse, like Notion. "+" next to it adds a new
// block below.

// MoveBlockMsg asks for document lines [Line, End) to be moved so they
// start at line To (a line number in the document before the move).
type MoveBlockMsg struct{ Line, End, To int }

// AddBlockMsg asks for a new empty block after document line After.
type AddBlockMsg struct{ After int }

// handleCols returns the margin columns of the "+" and "⠿" handles; -1 when
// the margin is too narrow to show them.
func (m Model) handleCols() (plus, grip int) {
	switch {
	case m.margin >= 4:
		return m.margin - 4, m.margin - 2
	case m.margin >= 2:
		return -1, m.margin - 2
	}
	return -1, -1
}

// blockAt returns the innermost block drawn on content row row, or -1.
func (m Model) blockAt(row int) int {
	found := -1
	for i, b := range m.blocks {
		if row >= b.Row && row < b.Row+b.H {
			found = i // sorted by row, so later matches are nested deeper
		}
	}
	return found
}

// Dragging reports whether a block is being dragged.
func (m Model) Dragging() bool { return m.drag >= 0 }

// Hovering reports whether a block is under the mouse, so its handles show.
func (m Model) Hovering() bool { return m.hover >= 0 }

// ClearHover hides the block handles (the mouse left the pane).
func (m *Model) ClearHover() {
	if m.drag < 0 {
		m.hover = -1
	}
}

// mouse handles hover, the handles and dragging. ok is false for events it
// leaves to the normal click handling.
func (m *Model) mouse(msg tea.MouseMsg) (tea.Cmd, bool) {
	row := m.viewport.YOffset + msg.Y
	switch msg.Action {
	case tea.MouseActionMotion:
		if m.drag >= 0 {
			// Scroll when dragging against the top or bottom edge.
			if msg.Y <= 0 {
				m.ScrollBy(-1)
			} else if msg.Y >= m.viewport.Height-1 {
				m.ScrollBy(1)
			}
			m.updateDrop(m.viewport.YOffset + msg.Y)
		} else {
			m.hover = m.blockAt(row)
		}
		return nil, true
	case tea.MouseActionRelease:
		if m.drag < 0 {
			return nil, false
		}
		cmd := m.drop()
		m.drag, m.dropRow = -1, -1
		m.hover = m.blockAt(row)
		return cmd, true
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return nil, false
		}
		plus, grip := m.handleCols()
		b := m.blockAt(row)
		if b < 0 || row != m.blocks[b].Row || (msg.X != grip && msg.X != plus) || grip < 0 {
			return nil, false
		}
		if msg.X == plus {
			after := m.blocks[b].End
			return func() tea.Msg { return AddBlockMsg{After: after} }, true
		}
		m.drag, m.hover = b, b
		m.updateDrop(row)
		return nil, true
	}
	return nil, false
}

// updateDrop picks the gap nearest to content row row as the drop target.
func (m *Model) updateDrop(row int) {
	d := m.blocks[m.drag]
	m.dropLine, m.dropRow, m.dropGap = -1, -1, false
	last := -1
	for i, b := range m.blocks {
		if b.Line >= d.Line && b.End <= d.End {
			continue // the dragged block and its children
		}
		if row < b.Row+(b.H+1)/2 {
			m.dropLine, m.dropRow = b.Line, b.Row-1
			m.dropGap = b.Row > 0 && i > 0 && m.blocks[i-1].Row+m.blocks[i-1].H < b.Row
			if i == 0 {
				m.dropGap = b.Row > 0
			}
			return
		}
		last = i
	}
	if last >= 0 {
		b := m.blocks[last]
		m.dropLine, m.dropRow, m.dropGap = b.End, b.Row+b.H, true
	}
}

// drop reports the move, unless the block would land where it already is.
func (m *Model) drop() tea.Cmd {
	if m.drag < 0 || m.dropLine < 0 {
		return nil
	}
	d := m.blocks[m.drag]
	if m.dropLine >= d.Line && m.dropLine <= d.End {
		return nil
	}
	msg := MoveBlockMsg{Line: d.Line, End: d.End, To: m.dropLine}
	return func() tea.Msg { return msg }
}

// decorate draws the handles and the drop line onto the visible rows
// (already prefixed with the margin).
func (m Model) decorate(lines []string) {
	th := m.theme
	plus, grip := m.handleCols()
	top := m.viewport.YOffset
	if grip >= 0 && m.hover >= 0 && m.hover < len(m.blocks) {
		if r := m.blocks[m.hover].Row - top; r >= 0 && r < len(lines) {
			st := lipgloss.NewStyle().Foreground(th.GreyFg2)
			if m.drag >= 0 {
				st = st.Foreground(th.Blue)
			}
			margin := []rune(strings.Repeat(" ", m.margin))
			if plus >= 0 {
				margin[plus] = '+'
			}
			margin[grip] = '⠿'
			lines[r] = st.Render(string(margin)) + strings.TrimPrefix(lines[r], strings.Repeat(" ", m.margin))
		}
	}
	if m.drag >= 0 && m.dropRow >= 0 {
		r := m.dropRow - top
		if r < 0 || r >= len(lines) {
			return
		}
		blue := lipgloss.NewStyle().Foreground(th.Blue)
		pad := strings.Repeat(" ", m.margin)
		if m.dropGap {
			lines[r] = pad + blue.Render(strings.Repeat("━", m.viewport.Width))
		} else if m.margin >= 1 {
			lines[r+min(1, len(lines)-1-r)] = blue.Render(strings.Repeat(" ", m.margin-1)+"▸") +
				strings.TrimPrefix(lines[r+min(1, len(lines)-1-r)], pad)
		}
	}
}
