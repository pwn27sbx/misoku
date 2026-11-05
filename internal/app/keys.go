package app

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
)

// WithKeys applies the "keys" section of config.json: each listed action gets
// exactly those shortcuts. Mistakes are skipped and shown in the status bar.
func WithKeys(user map[string][]string) Option {
	return func(m *Model) {
		km, warnings := keymap.New(user)
		m.keys = km
		m.sidebar.SetKeys(km)
		m.content.SetKeys(km)
		if len(warnings) > 0 {
			m.setError(strings.Join(warnings, "; "))
		}
	}
}

// generalActions are dispatched by the app outside the explorer's own keys,
// in this order.
var generalActions = []string{
	keymap.Quit, keymap.Save, keymap.NewNote, keymap.FindNote,
	keymap.ToggleExplorer, keymap.FocusExplorer,
	keymap.NextPane, keymap.PrevPane, keymap.PaneLeft, keymap.PaneRight,
	keymap.NextTab, keymap.PrevTab, keymap.CloseTab, keymap.TogglePreview, keymap.ReadingMode,
	keymap.ExportPDF, keymap.Themes, keymap.Graph, keymap.GlobalGraph, keymap.Home, keymap.SyncNow, keymap.Help,
	keymap.PasteImage, keymap.ToggleCheckbox, keymap.RemoveCheckbox,
}

// globalActions also fire while typing (INSERT mode, renaming), but only
// through Ctrl or Alt shortcuts, which never insert text.
var globalActions = []string{keymap.Quit, keymap.Save, keymap.NewNote, keymap.ToggleExplorer, keymap.FindNote}

// runAction performs one general action.
func (m *Model) runAction(action string) tea.Cmd {
	switch action {
	case keymap.Quit:
		return m.requestQuit(false)
	case keymap.Save:
		if b := m.activeBuffer(); b != nil {
			return m.saveBuffer(b, nil)
		}
	case keymap.NewNote:
		return m.newDraft(m.sidebar.ContextParentID())
	case keymap.FindNote:
		return m.startFind()
	case keymap.ToggleExplorer:
		return m.toggleSidebar()
	case keymap.FocusExplorer:
		return m.focusPane(focusSidebar)
	case keymap.NextPane:
		return m.movePane(1, true)
	case keymap.PrevPane:
		return m.movePane(-1, true)
	case keymap.PaneLeft:
		return m.movePane(-1, false)
	case keymap.PaneRight:
		return m.movePane(1, false)
	case keymap.NextTab:
		return m.cycleBuffer(1)
	case keymap.PrevTab:
		return m.cycleBuffer(-1)
	case keymap.CloseTab:
		return m.closeBuffer(m.active, false)
	case keymap.TogglePreview:
		return m.togglePreview()
	case keymap.ReadingMode:
		return m.toggleReading()
	case keymap.ExportPDF:
		return m.exportPDF("")
	case keymap.Themes:
		return m.openThemePicker()
	case keymap.Graph:
		return m.openGraph()
	case keymap.GlobalGraph:
		return m.openGlobalGraph()
	case keymap.Home:
		m.goHome()
	case keymap.SyncNow:
		return m.requestSync()
	case keymap.PasteImage:
		return m.content.PasteClipboard()
	case keymap.ToggleCheckbox:
		return m.editCheckboxes(m.content.ToggleCheckboxes)
	case keymap.RemoveCheckbox:
		return m.editCheckboxes(m.content.RemoveCheckboxes)
	case keymap.Help:
		return m.openPalette()
	}
	return nil
}

// editCheckboxes runs a checkbox edit of the editor on the open note and,
// when it changed the text, brings the modified mark and the preview up to
// date, as typing does.
func (m *Model) editCheckboxes(edit func() bool) tea.Cmd {
	if m.activeBuffer() == nil || !edit() {
		return nil
	}
	m.refreshModified()
	return m.editorEdited()
}

func defaultKeys() keymap.Map {
	km, _ := keymap.New(nil)
	return km
}
