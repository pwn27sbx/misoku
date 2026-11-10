package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Mouse

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	// A click is not part of a leader sequence: it cancels one.
	if msg.Action == tea.MouseActionPress {
		m.leaderPending = false
	}
	if m.viewMode == viewModeDashboard {
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if a, id, ok := m.dashboard.Click(msg.X, msg.Y); ok {
				return m.runDashboardAction(a, id)
			}
		}
		return nil
	}
	l := m.hitLayout()
	// The preview handles the event with what it compiled: hover handles,
	// block drags and clicks on interactive views must match the text.
	overPreview := l.overPreview(msg.X, msg.Y)
	if overPreview || m.preview.Dragging() {
		m.flushPreview()
	}
	if m.resizing != dividerNone {
		switch msg.Action {
		case tea.MouseActionMotion:
			m.dragDivider(msg.X)
		case tea.MouseActionRelease:
			m.resizing = dividerNone
		}
		return nil
	}
	// Drags that start in the editor keep going there (text selection).
	if msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionRelease {
		if !m.dragging {
			// Hover shows the preview's block handles; a block drag keeps
			// going wherever the mouse goes.
			if !overPreview && !m.preview.Dragging() {
				m.preview.ClearHover()
				return nil
			}
			local := msg
			local.X -= l.PreviewX
			var inside bool
			local.Y, inside = l.noteRow(msg.Y)
			if !inside && !m.preview.Dragging() {
				// A padding row holds no block: no handle to show.
				m.preview.ClearHover()
				return nil
			}
			var cmd tea.Cmd
			m.preview, cmd = m.preview.Update(local)
			return tea.Batch(cmd, m.tutorCHover())
		}
		if msg.Action == tea.MouseActionRelease {
			m.dragging = false
		}
		local := msg
		local.X -= l.EditorX
		local.Y = m.editorRow(l, msg.Y)
		return m.editorPaneUpdate(local)
	}
	if msg.Action != tea.MouseActionPress {
		return nil
	}

	if msg.Y == 0 {
		return m.handleTabClick(msg)
	}
	if msg.Y < l.BodyY || msg.Y >= l.BodyY+l.BodyH {
		return nil
	}

	wheel := msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
	if !wheel && msg.Button != tea.MouseButtonLeft {
		return nil
	}

	local := msg
	local.Y -= l.BodyY
	noteY, onText := l.noteRow(msg.Y)
	var target focus
	switch {
	case msg.X < l.SidebarW:
		target = focusSidebar
	case msg.X >= l.EditorX && msg.X < l.EditorX+l.EditorW && dividerAt(l, msg.X) == dividerNone:
		target, local.X = focusEditor, msg.X-l.EditorX
	case l.PreviewW > 0 && msg.X >= l.PreviewX:
		target, local.X = focusPreview, msg.X-l.PreviewX
	default: // a divider
		if msg.Button == tea.MouseButtonLeft {
			m.resizing = dividerAt(l, msg.X)
		}
		return nil
	}

	var cmds []tea.Cmd
	if !wheel && m.focus != target {
		cmds = append(cmds, m.focusPane(target))
	}
	var cmd tea.Cmd
	switch target {
	case focusSidebar:
		m.sidebar, cmd = m.sidebar.Update(local)
		cmd = m.chooseFromOverlay(cmd)
	case focusEditor:
		local.Y = m.editorRow(l, msg.Y)
		if !wheel && m.noteView == viewReading {
			local.Y = m.flushUnder(local.Y)
		}
		if pressed, ok := m.pressRendered(local, wheel); ok {
			cmd = pressed // what the preview acts on: no cursor, no selection
			break
		}
		m.dragging = msg.Button == tea.MouseButtonLeft
		cmd = m.editorPaneUpdate(local)
	case focusPreview:
		if !onText && !wheel {
			break // a padding row holds nothing to click; the press only focuses
		}
		local.Y = noteY
		hit, hasHit := m.preview.HitAt(local.X, local.Y)
		m.preview, cmd = m.preview.Update(local)
		m.tutorBPreviewClick(hit, hasHit)
	}
	return tea.Batch(append(cmds, cmd)...)
}

// foldArrowW is the width of a heading's fold arrow. In the reading view
// only the arrow folds: the rest of the heading row is text to click into.
const foldArrowW = 2

// flushUnder applies a pending compile before a press in the reading view
// is mapped, so it never acts on an older text's line, and returns the pane
// row that now shows what row y showed: the compile can move rows.
func (m *Model) flushUnder(y int) int {
	if !m.pv.pending {
		return y
	}
	spot := m.content.SpotAt(y)
	m.flushPreview()
	m.syncNoteDisplay()
	if now, ok := m.content.SpotY(spot); ok {
		return now
	}
	return y
}

// pressRendered does, in the reading view, what the preview does for a press
// on the rendered row or footer at pane cell (msg.X, msg.Y): follow a link,
// tick a box, fold, turn a calendar, run a view's control. It reports false
// when the press hits none of these and is the editor's.
func (m *Model) pressRendered(msg tea.MouseMsg, wheel bool) (tea.Cmd, bool) {
	if wheel || m.noteView != viewReading || m.graphActive() {
		return nil, false
	}
	row, col, ok := m.content.RenderedCell(msg.X, msg.Y)
	if !ok {
		return nil, false
	}
	hit, ok := m.preview.HitAtRow(row, col)
	if !ok || (hit.Kind == "fold" && col-hit.X0 >= foldArrowW) {
		return nil, false
	}
	cmd, _ := m.preview.PressRow(row, col)
	m.tutorBPreviewClick(hit, true)
	return cmd, true
}

// noteRow maps screen row y to the row of the editor's or the preview's text
// under it, clamped into the text rows, and reports whether y is on one of
// them rather than on a padding row.
func (l Layout) noteRow(y int) (row int, onText bool) {
	r := y - l.NoteY()
	return clamp(r, 0, l.NoteH()-1), r >= 0 && r < l.NoteH()
}

// editorRow maps screen row y into the editor's pane. A graph fills the whole
// body; the note keeps its padding rows, and a click on one lands on the text
// row next to it.
func (m *Model) editorRow(l Layout, y int) int {
	if m.graphActive() || m.globalGraphActive() {
		return y - l.BodyY
	}
	row, _ := l.noteRow(y)
	return row
}

// overPreview reports whether screen cell (x, y) is inside the preview pane.
func (l Layout) overPreview(x, y int) bool {
	return l.PreviewW > 0 && x >= l.PreviewX && y >= l.BodyY && y < l.BodyY+l.BodyH
}

// divider names a draggable pane divider.
type divider int

const (
	dividerNone    divider = iota
	dividerSidebar         // between the explorer and the editor
	dividerPreview         // between the editor and the preview
)

// dividerAt returns the divider in screen column x.
func dividerAt(l Layout, x int) divider {
	switch {
	case l.SidebarW > 0 && x == l.SidebarW:
		return dividerSidebar
	case l.PreviewW > 0 && x == l.PreviewX-1:
		return dividerPreview
	}
	return dividerNone
}

// dragDivider moves the divider being dragged to screen column x.
func (m *Model) dragDivider(x int) {
	l := m.layout()
	switch m.resizing {
	case dividerSidebar:
		m.split.SidebarW = x
	case dividerPreview:
		if avail := l.EditorW + l.PreviewW; avail > 0 {
			m.split.EditorFrac = float64(x-l.EditorX) / float64(avail)
		}
	}
	m.updateLayout()
	m.tutorBPaneResized()
	// Keep the stored position inside the limits the layout applied, so a
	// drag past the edge doesn't need to travel back before it moves again.
	l = m.layout()
	if m.resizing == dividerSidebar && l.SidebarW > 0 {
		m.split.SidebarW = l.SidebarW
	}
	if m.resizing == dividerPreview && l.EditorW+l.PreviewW > 0 {
		m.split.EditorFrac = float64(l.EditorW) / float64(l.EditorW+l.PreviewW)
	}
}

func (m *Model) handleTabClick(msg tea.MouseMsg) tea.Cmd {
	hit := m.tabAt(msg.X)
	switch msg.Button {
	case tea.MouseButtonMiddle:
		if hit.kind == hitTab || hit.kind == hitClose {
			return m.closeBuffer(hit.id, false)
		}
		return nil
	case tea.MouseButtonWheelUp:
		return m.cycleBuffer(-1)
	case tea.MouseButtonWheelDown:
		return m.cycleBuffer(1)
	case tea.MouseButtonLeft:
	default:
		return nil
	}
	switch hit.kind {
	case hitTab:
		if i := m.bufferIndex(hit.id); i >= 0 {
			m.showBuffer(m.buffers[i])
			return m.focusPane(focusEditor)
		}
	case hitClose:
		return m.closeBuffer(hit.id, false)
	case hitNew:
		return m.newDraft(m.sidebar.ContextParentID())
	case hitExplorer:
		return m.focusPane(focusSidebar)
	}
	return nil
}
