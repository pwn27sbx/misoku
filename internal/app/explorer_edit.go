package app

import (
	"path"
	"strings"

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
// edits pending on leaving, as mini.files does: Yes applies them and
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
			return apply(m, 0)
		case 1:
			m.sidebar.StopEditing()
		}
		return nil
	}
}

// closeEditorAfter closes the folder editor once its batch wrote
// something: its lines describe the folders as they were.
func (m *Model) closeEditorAfter(msg relinksAppliedMsg) {
	if msg.job.batch != nil && msg.job.batch.done > 0 {
		m.sidebar.StopEditing()
	}
}

// createPlannedMsg is the plan of Enter on a new named line of the folder
// editor; createdMsg is what applying it wrote.
type (
	createPlannedMsg struct{ planned relinksPlannedMsg }
	createdMsg       struct{ applied relinksAppliedMsg }
)

// requestCreatePlan plans the edits like =, in requestDirPlan's command, and
// tags the answer as Enter's create.
func (m *Model) requestCreatePlan(folders []dirplan.Folder) tea.Cmd {
	plan := m.requestDirPlan(folders)
	return func() tea.Msg {
		planned, _ := plan().(relinksPlannedMsg)
		return createPlannedMsg{planned: planned}
	}
}

// applyCreates applies Enter's plan with no confirm when every action
// creates a note or a folder, which loses nothing. A plan with any other
// action (a rename, a move, a delete) stays pending for = or leaving, with
// their confirm; a refused plan shows its error.
func (m *Model) applyCreates(msg createPlannedMsg) tea.Cmd {
	batch := msg.planned.job.batch
	switch {
	case batch == nil:
		return nil
	case batch.err != nil:
		m.setError("Can't create: " + batch.err.Error())
		return nil
	case len(batch.plan.Ops) == 0 || !onlyCreates(batch.plan.Ops):
		return nil
	}
	store, job := m.store, msg.planned.job
	return func() tea.Msg { return createdMsg{applied: applyDirPlan(store, job)} }
}

// finishCreates brings the screen up to date like any batch and, when every
// create ran, restarts the folder editor from the disk on the entry made.
func (m *Model) finishCreates(msg createdMsg) tea.Cmd {
	m.finishRelinks(msg.applied)
	batch := msg.applied.job.batch
	if batch.failed != nil || batch.done == 0 {
		return nil
	}
	return m.sidebar.SettleCreated(childOf(m.sidebar.EditFolder(), batch.plan.Ops[0].To))
}

func onlyCreates(ops []dirplan.Op) bool {
	for _, op := range ops {
		if op.Kind != dirplan.OpCreate && op.Kind != dirplan.OpMkdir {
			return false
		}
	}
	return true
}

// childOf is the entry of folder that holds id: id itself, or the folder
// of folder it is nested in (a/b.md typed in the root is under a).
func childOf(folder, id string) string {
	rel := id
	if folder != "" {
		rel = strings.TrimPrefix(id, folder+"/")
	}
	if i := strings.Index(rel, "/"); i >= 0 {
		return path.Join(folder, rel[:i])
	}
	return id
}
