package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"

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

// checkWorkspace syncs open notes with the disk and refreshes the explorer
// when the folder changed,
// keeping the cursor on the same note even if rows above it came or went.
// That also covers a rename in progress: it applies to the selected note,
// which is still the same one after the refresh.
func (m *Model) checkWorkspace() {
	m.syncUserTheme()
	m.syncOpenBuffers()
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

// syncOpenBuffers brings open notes up to date with the disk. A note with no
// unsaved edits is reloaded in place, keeping the cursor; one with unsaved
// edits is left alone and the user is told, on every check while the
// conflict lasts, and saving it asks before overwriting (see saveBuffer).
func (m *Model) syncOpenBuffers() {
	for _, b := range m.buffers {
		if b.draft() || b.isGraph() {
			continue
		}
		p, err := m.store.Get(b.id)
		if err != nil || p.Content == b.saved {
			continue
		}
		if m.isDirty(b) {
			m.setError(fmt.Sprintf("%q changed on disk; saving will ask before overwriting it", b.id))
			continue
		}
		b.saved, b.text = p.Content, p.Content
		if b.id == m.active {
			m.content.ReplaceText(p.Content)
			m.preview.SetPage(b.page())
			m.refreshLinks()
		}
		m.setStatus(fmt.Sprintf("%q reloaded: it changed on disk", b.id))
	}
	m.refreshModified()
}

// syncUserTheme re-reads the user theme files when they changed on disk (a
// tool such as Omarchy's theme switcher rewrites them) and re-applies the
// theme in use, without saving anything. A bundled theme comes out the same,
// since bundled names win over user files.
func (m *Model) syncUserTheme() {
	stamp := theme.UserThemesStamp()
	if stamp == m.themeStamp {
		return
	}
	m.themeStamp = stamp
	theme.ReloadUserThemes()
	if th, ok := theme.Get(m.theme.Name); ok {
		m.applyTheme(th, false)
	}
}
