package app

import (
	"github.com/pwn27sbx/misoku/internal/content"

	tea "github.com/charmbracelet/bubbletea"
)

// The app reports what the player does to the current lesson in two ways.
// The editor records its own actions (content.SetReportActions, drained here)
// because they happen inside the editor's Update; everything else calls
// m.tutorEvent where the action is confirmed done, or m.tutorDefer from a
// place that cannot return a command. None of it costs anything without a
// tutorial: m.tutor is nil outside `misoku tutor`.

// tutorAfterUpdate runs after every update: it turns the editor's recorded
// actions into tutorial events, saves what they passed, and follows the
// look-only hints whose state just changed.
func (m *Model) tutorAfterUpdate() tea.Cmd {
	actions := m.content.TakeActions()
	g := m.tutor
	if g == nil {
		return nil
	}
	var cmds []tea.Cmd
	for _, a := range actions {
		cmds = append(cmds, m.tutorEvent(a.Name, a.Value))
	}
	cmds = append(cmds, g.takeCmds()...)
	if m.tableHintVisible() && !g.tableHint {
		g.tableHint = true
		cmds = append(cmds, m.tutorEvent("table_hint", ""))
	}
	return tea.Batch(cmds...)
}

// tutorDefer records one tutorial event from a place that cannot return the
// command its save needs; the command runs at the end of the update.
func (m *Model) tutorDefer(name, value string) {
	g := m.tutor
	if g == nil {
		return
	}
	if cmd := m.tutorEvent(name, value); cmd != nil {
		g.pending = append(g.pending, cmd)
	}
}

// takeCmds hands the deferred commands to the update that drains them.
func (g *tutorGame) takeCmds() []tea.Cmd {
	if len(g.pending) == 0 {
		return nil
	}
	out := g.pending
	g.pending = nil
	return out
}

// tableHintVisible reports whether the statusline is showing the table hint
// (bars.go's tableHint); an event needs the state, not the drawn text.
func (m *Model) tableHintVisible() bool {
	return m.viewMode == viewModeWorkspace && m.focus == focusEditor &&
		m.content.Mode() == content.ModeInsert && m.content.InTable()
}
