package app

import (
	"fmt"

	"github.com/pwn27sbx/misoku/internal/sidebar"

	tea "github.com/charmbracelet/bubbletea"
)

// confirmLeaveEditor asks before the folder editor drops its pending
// edits (S8): Yes closes it and writes nothing, No keeps editing.
func (m *Model) confirmLeaveEditor(msg sidebar.EditLeaveMsg) {
	m.confirm = &confirmDialog{
		title:   fmt.Sprintf("Discard %d %s?", msg.Changes, plural(msg.Changes, "change", "changes")),
		message: []string{"The folder editor's edits are dropped; nothing is written."},
		buttons: []string{"Yes", "No"},
		danger:  true,
		onChoose: func(m *Model, choice int) tea.Cmd {
			if choice == 0 {
				m.sidebar.StopEditing()
			}
			return nil
		},
	}
}

// closeEditorAfter closes the folder editor once its batch wrote
// something: its lines describe the folders as they were.
func (m *Model) closeEditorAfter(msg relinksAppliedMsg) {
	if msg.job.batch != nil && msg.job.batch.done > 0 {
		m.sidebar.StopEditing()
	}
}
