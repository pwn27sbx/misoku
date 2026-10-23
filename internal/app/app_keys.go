package app

import (
	"slices"

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

	// Only typing in the editor leaves an edit for the preview to catch up
	// on; everything else first brings the preview up to date.
	if m.focus != focusEditor || m.graphActive() {
		m.flushPreview()
	}

	sidebarBusy := m.focus == focusSidebar && m.sidebar.IsBusy()
	editorBusy := m.focus == focusEditor && m.content.Mode() != content.ModeNormal && !m.graphActive()

	// Ctrl/Alt shortcuts of the global actions work in every mode. One
	// typed after Space is the sequence's second key: it ends the sequence.
	for _, a := range globalActions {
		if m.keys.DirectModified(a, s) {
			if a == keymap.NewNote && sidebarBusy {
				break
			}
			m.leaderPending = false
			m.flushPreview()
			return m.runAction(a)
		}
	}

	if !sidebarBusy && !editorBusy {
		if m.leaderPending {
			guide := m.leaderGuideOpen()
			m.leaderPending = false
			for _, a := range generalActions {
				if m.keys.Leader(a, s) {
					m.flushPreview()
					return m.runAction(a)
				}
			}
			if guide && s != "esc" {
				m.setStatus(s + ": not a leader key")
			}
			return nil
		}
		if m.keys.IsLeader(s) {
			return m.startLeader()
		}
		for _, a := range generalActions {
			if m.keys.Direct(a, s) {
				m.flushPreview()
				return m.runAction(a)
			}
		}
		if s == "esc" && m.focus == focusSidebar && m.explorerOverlay() {
			return m.closeExplorerOverlay()
		}
		if s == "esc" && m.focus != focusEditor && m.activeBuffer() != nil {
			return m.focusPane(focusEditor)
		}
	}
	m.leaderPending = false

	// ":" commands are typed in the editor: reading mode makes way for it.
	if m.reading && m.focus == focusPreview && s == ":" {
		m.leaveReading()
		focus := m.focusPane(focusEditor)
		return tea.Batch(focus, m.handleKey(k))
	}

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
		ver := m.content.Version()
		m.content, cmd = m.content.Update(k)
		if b := m.activeBuffer(); b != nil {
			m.refreshModified()
			// A key that only moved the cursor or the view leaves the
			// preview and the graph as they are.
			if m.content.Version() != ver {
				cmd = tea.Batch(cmd, m.editorEdited())
			}
		}
	}
	return cmd
}

// homeActions are the general actions the home screen runs from the keymap
// (leader sequences and Ctrl/Alt shortcuts). The pane, tab, preview, save
// and export actions need an open note, so they wait for one.
var homeActions = []string{
	keymap.NewNote, keymap.FindNote, keymap.ToggleExplorer, keymap.FocusExplorer,
	keymap.Themes, keymap.Graph, keymap.SyncNow, keymap.Help, keymap.Quit,
}

func (m *Model) handleDashboardKey(k tea.KeyMsg) tea.Cmd {
	s := k.String()
	// The leader comes first: tab, for one, moves the home selection but
	// is also the second key of "space tab".
	if m.leaderPending {
		guide := m.leaderGuideOpen()
		m.leaderPending = false
		for _, a := range homeActions {
			if m.keys.Leader(a, s) {
				return m.runHomeAction(a)
			}
		}
		// The guide lists every leader key, also those that need an open
		// note: those say so instead of doing nothing.
		switch {
		case slices.ContainsFunc(generalActions, func(a string) bool { return m.keys.Leader(a, s) }):
			m.setStatus(s + ": needs an open note")
		case guide && s != "esc":
			m.setStatus(s + ": not a leader key")
		}
		return nil
	}
	if m.keys.IsLeader(s) {
		return m.startLeader()
	}
	for _, a := range homeActions {
		if m.keys.DirectModified(a, s) {
			return m.runHomeAction(a)
		}
	}
	// The letter keys are fixed: they are the ones the home screen shows.
	switch s {
	case "n":
		return m.newDraft("")
	case "f", "\\":
		return m.startFind()
	case "e":
		return m.focusPane(focusSidebar)
	case "?":
		return m.openPalette()
	case "t":
		return m.openThemePicker()
	case "g":
		return m.openGraph()
	case "c":
		return m.OpenSettings()
	case "s":
		// Only while the entry shows: a folder that syncs has nothing to
		// set up.
		if m.syncOffered() {
			return m.startSetup()
		}
		return nil
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
		i := int(s[0] - '1')
		recent, projects := m.dashboard.RecentPages(), m.dashboard.Projects()
		switch {
		case i < len(recent):
			return m.openFile(recent[i].ID)
		case i-len(recent) < len(projects):
			return m.openProject(projects[i-len(recent)].ID)
		}
		return nil
	}
	m.dashboard, _ = m.dashboard.Update(k)
	return nil
}

func (m *Model) runDashboardAction(a dashboard.Action, id string) tea.Cmd {
	switch a {
	case dashboard.ActionOpenProject:
		return m.openProject(id)
	case dashboard.ActionNewPage:
		return m.newDraft("")
	case dashboard.ActionFind:
		return m.startFind()
	case dashboard.ActionBrowse:
		return m.focusPane(focusSidebar)
	case dashboard.ActionKeymap:
		return m.openPalette()
	case dashboard.ActionQuit:
		return m.requestQuit(false)
	case dashboard.ActionOpenRecent:
		return m.openFile(id)
	case dashboard.ActionThemes:
		return m.openThemePicker()
	case dashboard.ActionSyncSetup:
		return m.startSetup()
	case dashboard.ActionGraph:
		return m.openGraph()
	case dashboard.ActionSettings:
		return m.OpenSettings()
	}
	return nil
}

// runHomeAction runs a keymap action from the home screen. A new note made
// here goes to the top of the workspace, as with n.
func (m *Model) runHomeAction(a string) tea.Cmd {
	if a == keymap.NewNote {
		return m.newDraft("")
	}
	return m.runAction(a)
}

// openProject shows a top-level folder from the home screen: the explorer
// takes focus with that folder expanded and selected.
func (m *Model) openProject(id string) tea.Cmd {
	cmd := m.focusPane(focusSidebar)
	m.sidebar.Expand(id)
	m.sidebar.SetSelectedID(id)
	return cmd
}
