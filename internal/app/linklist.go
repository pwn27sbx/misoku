package app

import (
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// linkSourceMsg carries the vault's notes and text to the editor's inline
// "[[" link list, read in a command: Update never reads the disk.
type linkSourceMsg struct{ notes []core.LinkNote }

// openLinkList reads every note away from Update and opens the editor's
// inline list of "[[" suggestions with them. The editor keeps the typed
// "[[" in the note and reads what comes after it as the user types.
func (m *Model) openLinkList() tea.Cmd {
	store, pages := m.store, append([]core.Page(nil), m.notes...)
	return func() tea.Msg {
		notes := make([]core.LinkNote, 0, len(pages))
		for _, p := range pages {
			full, err := store.Get(p.ID)
			if err != nil {
				continue // a note deleted meanwhile offers no suggestion
			}
			notes = append(notes, core.LinkNote{ID: p.ID, Title: p.Title, Text: full.Content, UpdatedAt: p.UpdatedAt})
		}
		return linkSourceMsg{notes: notes}
	}
}

// inlineLinkSuggestions reports whether "[[" opens the inline list (the
// default) instead of the finder, the ui.link_suggestions setting.
func (m *Model) inlineLinkSuggestions() bool {
	return m.prefs().ui.InlineLinkSuggestions()
}
