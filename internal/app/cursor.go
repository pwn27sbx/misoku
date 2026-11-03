package app

import (
	"os/exec"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/cursor"

	tea "github.com/charmbracelet/bubbletea"
)

// CursorSink is the terminal's own cursor: where it goes, what shape it takes
// and how the terminal gets its own back. *cursor.Writer implements it.
type CursorSink interface {
	Set(x, y int, shape cursor.Shape, visible bool)
	Restore() error
}

// WithCursor draws the mode with the terminal's cursor instead of a fake cell.
// main passes the writer that also places it; without one the editor keeps its
// drawn cursor, which is today's look.
func WithCursor(s CursorSink) Option {
	return func(m *Model) {
		m.cursor = s
		if s != nil {
			m.content.SetRealCursor(true)
		}
	}
}

// syncCursor tells the terminal where its cursor goes and how it looks. View
// calls it on its own copy of the model, so the sink sees every state change,
// including the ones whose frame text the renderer skips.
func (m Model) syncCursor(l Layout) {
	if m.cursor == nil {
		return
	}
	x, y, shape, visible := m.cursorCell(l)
	m.cursor.Set(x, y, shape, visible)
}

// cursorCell is the cell and shape of the real cursor, or visible=false when
// another widget, pane or screen owns it.
func (m Model) cursorCell(l Layout) (x, y int, shape cursor.Shape, visible bool) {
	shape = cursor.Block
	if m.content.Mode() == content.ModeInsert {
		shape = cursor.Bar
	}
	if !m.editorHasTheCursor(l) {
		return 0, 0, shape, false
	}
	cx, cy, ok := m.content.CursorScreen()
	if !ok {
		return 0, 0, shape, false
	}
	return l.EditorX + cx, l.BodyY + cy, shape, true
}

// editorHasTheCursor reports whether the editor is what typing goes to with
// nothing drawn over it: only then is the terminal cursor its cursor. Every
// other widget keeps its own drawn cursor.
func (m Model) editorHasTheCursor(l Layout) bool {
	switch {
	case m.viewMode != viewModeWorkspace, m.focus != focusEditor, l.EditorW <= 0,
		m.graphActive(), m.globalGraphActive(),
		m.saveAs != nil, m.confirm != nil, m.finder != nil, m.themes != nil,
		m.browser != nil, m.datePick != nil, m.promptBox != nil, m.menuBox != nil,
		m.palette != nil, m.leaderGuideOpen():
		return false
	}
	return true
}

// execOutside runs cmd on the terminal in place of Misoku's screen: the cursor
// goes back to the terminal's default first, and the editor's shape returns
// with the next frame once the program ends.
func (m *Model) execOutside(cmd *exec.Cmd, cb tea.ExecCallback) tea.Cmd {
	if m.cursor != nil {
		_ = m.cursor.Restore()
	}
	return m.execProcess(cmd, cb)
}
