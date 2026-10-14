package app

import (
	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Keyboard

func (m *Model) handleKey(k tea.KeyMsg) tea.Cmd {
	s := k.String()
	if m.keys.DirectModified(keymap.Quit, s) {
		return m.requestQuit(false)
	}
	if m.viewMode == viewModeDashboard {
		return m.handleDashboardKey(k)
	}

	sidebarBusy := m.focus == focusSidebar && m.sidebar.IsBusy()
	editorBusy := m.focus == focusEditor && m.content.Mode() != content.ModeNormal && !m.graphActive()

	// Ctrl/Alt shortcuts of the global actions work in every mode.
	for _, a := range globalActions {
		if m.keys.DirectModified(a, s) {
			if a == keymap.NewNote && sidebarBusy {
				break
			}
			return m.runAction(a)
		}
	}

	if !sidebarBusy && !editorBusy {
		if m.leaderPending {
			m.leaderPending = false
			for _, a := range generalActions {
				if m.keys.Leader(a, s) {
					return m.runAction(a)
				}
			}
			return nil
		}
		if m.keys.IsLeader(s) {
			m.leaderPending = true
			return nil
		}
		for _, a := range generalActions {
			if m.keys.Direct(a, s) {
				return m.runAction(a)
			}
		}
		if s == "esc" && m.focus != focusEditor && m.activeBuffer() != nil {
			return m.focusPane(focusEditor)
		}
	}
	m.leaderPending = false

	var cmd tea.Cmd
	switch m.focus {
	case focusSidebar:
		m.sidebar, cmd = m.sidebar.Update(k)
	case focusPreview:
		m.preview, cmd = m.preview.Update(k)
	default:
		if m.graphActive() {
			m.graph, cmd = m.graph.Update(k)
			return cmd
		}
		if m.activeBuffer() == nil && s == "i" {
			return m.newDraft(m.sidebar.ContextParentID())
		}
		m.content, cmd = m.content.Update(k)
		if b := m.activeBuffer(); b != nil {
			m.preview.SetContent(m.content.Value())
			m.refreshModified()
			if m.bufferIndex(graphID) >= 0 {
				m.liveGraph()
			}
		}
	}
	return cmd
}

func (m *Model) handleDashboardKey(k tea.KeyMsg) tea.Cmd {
	s := k.String()
	switch {
	case m.keys.DirectModified(keymap.NewNote, s):
		return m.newDraft("")
	case m.keys.DirectModified(keymap.FindNote, s):
		return m.startFind()
	case m.keys.DirectModified(keymap.ToggleExplorer, s):
		return m.focusPane(focusSidebar)
	}
	switch s {
	case "n":
		return m.newDraft("")
	case "f", "\\":
		return m.startFind()
	case "e":
		return m.focusPane(focusSidebar)
	case "?":
		m.showKeymap = true
		return nil
	case "t":
		return m.openThemePicker()
	case "g":
		return m.openGraph()
	case "q":
		return m.requestQuit(false)
	case "esc":
		if m.activeBuffer() != nil {
			return m.focusPane(focusEditor)
		}
		return nil
	case "enter":
		a, id := m.dashboard.SelectedAction()
		return m.runDashboardAction(a, id)
	}
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		if i := int(s[0] - '1'); i < len(m.dashboard.RecentPages()) {
			return m.openFile(m.dashboard.RecentPages()[i].ID)
		}
		return nil
	}
	m.dashboard, _ = m.dashboard.Update(k)
	return nil
}

func (m *Model) runDashboardAction(a dashboard.Action, id string) tea.Cmd {
	switch a {
	case dashboard.ActionNewPage:
		return m.newDraft("")
	case dashboard.ActionFind:
		return m.startFind()
	case dashboard.ActionBrowse:
		return m.focusPane(focusSidebar)
	case dashboard.ActionKeymap:
		m.showKeymap = true
	case dashboard.ActionQuit:
		return m.requestQuit(false)
	case dashboard.ActionOpenRecent:
		return m.openFile(id)
	case dashboard.ActionThemes:
		return m.openThemePicker()
	}
	return nil
}
