package app

import (
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// WithClock makes the app read the time from now, so a test can say how
// long ago the last sync was.
func WithClock(now func() time.Time) Option {
	return func(m *Model) { m.now = now }
}

// SaveCommitDelay is how long after the last save the commit waits.
const SaveCommitDelay = saveCommitDelay

// WithExec replaces how the app runs a program outside its screen
// (tea.ExecProcess), so a test can run gh's sign-in without a terminal.
func WithExec(exec func(*exec.Cmd, tea.ExecCallback) tea.Cmd) Option {
	return func(m *Model) { m.execProcess = exec }
}

// IsWorkspaceScan reports whether msg is the result of the workspace scan a
// CheckWorkspaceMsg starts, so a test counting the git work a tick starts
// can tell it apart.
func IsWorkspaceScan(msg tea.Msg) bool {
	_, ok := msg.(workspaceScannedMsg)
	return ok
}

// OpenInCurrentBuffer shows id in the active tab, as picking a note in the
// finder does, and returns the updated model and its command.
func OpenInCurrentBuffer(tm tea.Model, id string) (tea.Model, tea.Cmd) {
	m := tm.(Model)
	cmd := m.openInCurrentBuffer(id)
	return m, cmd
}
