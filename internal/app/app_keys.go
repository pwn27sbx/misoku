package app

import (
	"slices"
	"strings"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Keyboard

// handleKey runs a key and reports a mode change to the current tutorial
// lesson (E9).
func (m *Model) handleKey(k tea.KeyMsg) tea.Cmd {
	before := m.content.Mode()
	cmd := m.handleKeyAction(k)
	if m.content.Mode() != before {
		return tea.Batch(cmd, m.tutorEvent("mode_changed", modeValue(m.content.Mode())), m.tutorCCursorShape(m.content.Mode()))
	}
	return cmd
}

func (m *Model) handleKeyAction(k tea.KeyMsg) tea.Cmd {
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
	// Typing in the editor, or in the global graph's filter prompt: letters,
	// the space and tab are text there, not the app's keys.
	editorBusy := m.focus == focusEditor && (m.content.Mode() != content.ModeNormal && !m.graphActive() ||
		m.globalGraphActive() && m.globalGraph.Typing())

	// Ctrl/Alt shortcuts of the global actions work in every mode. One
	// typed after Space is the sequence's second key: it ends the sequence.
	for _, a := range globalActions {
		if m.keys.DirectModified(a, s) {
			if a == keymap.NewNote && sidebarBusy {
				break
			}
			if a == keymap.Save && m.focus == focusSidebar && m.sidebar.IsEditing() {
				break // the folder editor's save: it applies the edits
			}
			m.leaderPending = false
			m.flushPreview()
			return m.runAction(a)
		}
	}

	// A VISUAL selection takes the leader too, but only for the keys of
	// visualLeaderActions; every other key still goes to the selection.
	if m.visualSelecting() && (m.leaderPending || m.keys.IsLeader(s)) {
		if !m.leaderPending {
			return m.startLeader()
		}
		return m.editorLeaderKey(s, visualLeaderActions)
	}

	// The preview's own keys win on that pane: H and L shift its calendars
	// (preview.go), whatever the general actions bind them to.
	previewCalendarKey := m.focus == focusPreview && (s == "H" || s == "L")

	if !sidebarBusy && !editorBusy {
		if m.leaderPending {
			return m.editorLeaderKey(s, generalActions)
		}
		if m.keys.IsLeader(s) {
			return m.startLeader()
		}
		if !previewCalendarKey {
			for _, a := range generalActions {
				if m.keys.Direct(a, s) {
					m.flushPreview()
					return m.runAction(a)
				}
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

	var cmd tea.Cmd
	switch m.focus {
	case focusSidebar:
		tree := m.tutorBTreeSnapshot()
		editing := m.sidebar.IsEditing()
		m.sidebar, cmd = m.sidebar.Update(k)
		m.tutorBTreeKey(tree)
		cmd = tea.Batch(cmd, m.tutorCDirbuf(k.String(), editing))
	case focusPreview:
		m.tutorBPreviewKey(k)
		m.preview, cmd = m.preview.Update(k)
	default:
		if m.graphActive() {
			return m.editorPaneUpdate(k)
		}
		if m.activeBuffer() == nil && s == "i" {
			return m.newDraft(m.sidebar.ContextParentID())
		}
		ver, typing := m.content.Version(), m.content.Mode() == content.ModeInsert && !k.Paste
		m.content, cmd = m.content.Update(k)
		if b := m.activeBuffer(); b != nil {
			m.refreshModified()
			// A key that only moved the cursor or the view leaves the
			// preview and the graph as they are.
			if m.content.Version() != ver {
				cmd = tea.Batch(cmd, m.editorEdited())
				if m.noteView == viewReading && !typing {
					m.flushPreview() // only typing can wait: this view draws the compile
				}
			}
		}
	}
	return cmd
}

// visualSelecting reports whether the editor holds a VISUAL or V-LINE
// selection.
func (m *Model) visualSelecting() bool {
	mode := m.content.Mode()
	return m.focus == focusEditor && !m.graphActive() && (mode == content.ModeVisual || mode == content.ModeVisualLine)
}

// editorLeaderKey handles s typed after Space outside the home screen: it
// runs the action of actions the keys complete, waits when they start a
// longer sequence, and otherwise ends the sequence, saying so once the guide
// is open.
func (m *Model) editorLeaderKey(s string, actions []string) tea.Cmd {
	guide := m.leaderGuideOpen()
	keys, a, wait := m.leaderNext(s, actions)
	switch {
	case a != "":
		m.flushPreview()
		return m.runAction(a)
	case m.leaderPending:
		return wait
	case guide && s != "esc":
		m.setStatus(strings.Join(keys, " ") + ": not a leader key")
	}
	return nil
}

// homeActions are the general actions the home screen runs from the keymap
// (leader sequences and Ctrl/Alt shortcuts). The pane, tab, preview, save
// and export actions need an open note, so they wait for one.
var homeActions = []string{
	keymap.NewNote, keymap.FindNote, keymap.ToggleExplorer, keymap.FocusExplorer,
	keymap.Themes, keymap.Graph, keymap.GlobalGraph, keymap.DailyNote, keymap.SyncNow, keymap.Help, keymap.Quit,
}

func (m *Model) handleDashboardKey(k tea.KeyMsg) tea.Cmd {
	s := k.String()
	// The leader comes first: tab, for one, moves the home selection but
	// is also the second key of "space tab".
	if m.leaderPending {
		guide := m.leaderGuideOpen()
		keys, a, wait := m.leaderNext(s, generalActions)
		// The guide lists every leader key, also those that need an open
		// note: those say so instead of doing nothing.
		switch {
		case slices.Contains(homeActions, a):
			return m.runHomeAction(a)
		case a != "":
			m.setStatus(strings.Join(keys, " ") + ": needs an open note")
		case m.leaderPending:
			return wait
		case guide && s != "esc":
			m.setStatus(strings.Join(keys, " ") + ": not a leader key")
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
	// The lesson asks the player to try each one, so the press is the
	// event even where this vault gives the key nothing to open.
	switch s {
	case "n", "f", "e", "t", "?", "g", "c", "s", "l", "q":
		m.tutorBHomeKey(s)
	}
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
		return m.openGlobalGraph()
	case "l":
		// Only while the entries show: a tutorial session has none, so
		// sessions cannot nest.
		if m.learn != nil {
			return m.learnChosen(m.learn.firstIncomplete())
		}
		return nil
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
			m.tutorBHomeOpen("recent")
			return m.openFile(recent[i].ID)
		case i-len(recent) < len(projects):
			m.tutorBHomeOpen("project")
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
		return m.openGlobalGraph()
	case dashboard.ActionSettings:
		return m.OpenSettings()
	case dashboard.ActionLearn:
		return m.learnChosen(id)
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

// followEnterLink follows the [[link]] Enter was pressed on in a new tab,
// where gd stays in this one. A link naming no note says so instead of
// starting a draft, as the LazyVim companion's Enter does. A link to a
// section of this note ([[#Heading]]) jumps there.
func (m *Model) followEnterLink(l core.WikiLink) tea.Cmd {
	if l.Target != "" {
		if _, ok := core.ResolveNote(m.notes, l.Target, m.currentNoteID()); !ok {
			m.setStatus("No note found for this link")
			return nil
		}
	}
	m.tutorBFollowedLink("Enter")
	return m.followWikiLink(l, m.openFile)
}
