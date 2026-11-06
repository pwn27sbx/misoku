package app

import (
	"github.com/pwn27sbx/misoku/internal/dirplan"

	tea "github.com/charmbracelet/bubbletea"
)

// leavePlannedMsg is the plan of the folder editor's edits asked for on
// leaving it (Esc or q): the same plan = asks for, confirmed with a cancel.
type leavePlannedMsg struct{ planned relinksPlannedMsg }

// requestLeavePlan plans the edits like =, in requestDirPlan's command, and
// tags the answer as a leave.
func (m *Model) requestLeavePlan(folders []dirplan.Folder) tea.Cmd {
	plan := m.requestDirPlan(folders)
	return func() tea.Msg {
		planned, _ := plan().(relinksPlannedMsg)
		return leavePlannedMsg{planned: planned}
	}
}

// confirmLeaveEditor shows the "CONFIRM FILE SYSTEM ACTIONS" of = for the
// edits pending on leaving, as Oil saves on leaving: Yes applies them and
// closes, No drops them and closes, Cancel keeps editing. Edits that plan
// no action close at once; a refused plan shows its error and stays.
func (m *Model) confirmLeaveEditor(msg leavePlannedMsg) {
	batch := msg.planned.job.batch
	if batch == nil || batch.err == nil && len(batch.plan.Ops) == 0 {
		m.sidebar.StopEditing()
		return
	}
	m.confirmDirPlan(msg.planned)
	if m.confirm == nil {
		return
	}
	apply := m.confirm.onChoose
	m.confirm.buttons = []string{"Yes", "No", "Cancel"}
	m.confirm.onChoose = func(m *Model, choice int) tea.Cmd {
		switch choice {
		case 0:
			m.leavingEditor = true
			return apply(m, 0)
		case 1:
			m.sidebar.StopEditing()
		}
		return nil
	}
}

// settleEditor brings the folder editor up to date after a batch, as Oil
// does: once something was written it stays open and reloads its folder
// from the disk, cursor on the entry the cursor's line made. A batch asked
// for on leaving closes it instead when every action ran.
func (m *Model) settleEditor(msg relinksAppliedMsg) tea.Cmd {
	batch := msg.job.batch
	if batch == nil {
		return nil
	}
	if m.leavingEditor {
		m.leavingEditor = false
		if batch.failed == nil {
			m.sidebar.StopEditing()
			return nil
		}
	}
	if batch.done == 0 {
		return nil
	}
	return m.sidebar.ReloadEditor()
}
