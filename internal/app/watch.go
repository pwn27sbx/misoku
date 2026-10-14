package app

import (
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// workspaceCheckInterval is how often the workspace folder is re-read, so
// notes and folders created, deleted or renamed by another program (an
// editor, git, a sync client) show up in the explorer without restarting.
// Polling keeps it dependency-free and identical on every OS; a notes folder
// is small enough that a walk every couple of seconds costs nothing.
const workspaceCheckInterval = 2 * time.Second

// CheckWorkspaceMsg asks the app to re-read the workspace folder. The app
// schedules one every workspaceCheckInterval; it is exported so tests and
// embedders can trigger a check without waiting for the timer.
type CheckWorkspaceMsg struct{}

func checkWorkspaceLater() tea.Cmd {
	return tea.Tick(workspaceCheckInterval, func(time.Time) tea.Msg { return CheckWorkspaceMsg{} })
}

// treeSignature identifies the shape of the workspace tree: which notes and
// folders exist and where. Modification times are left out on purpose, so
// saving a note does not count as a change to the tree.
func treeSignature(pages []core.Page) string {
	var b strings.Builder
	for _, p := range pages {
		b.WriteString(p.ID)
		b.WriteByte(0)
		b.WriteString(p.ParentID)
		if p.IsFolder {
			b.WriteString("/")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// checkWorkspace refreshes the explorer when the folder changed on disk,
// keeping the cursor on the same note even if rows above it came or went.
// That also covers a rename in progress: it applies to the selected note,
// which is still the same one after the refresh.
func (m *Model) checkWorkspace() {
	pages := m.store.List()
	if treeSignature(pages) == m.treeSig {
		return
	}
	selected, ok := m.sidebar.SelectedPage()
	m.applyTree(pages)
	if ok {
		m.sidebar.SetSelectedID(selected.ID)
	}
}
