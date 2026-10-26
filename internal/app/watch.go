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
// when the folder changed. It reads the disk where it is called; the
// periodic check does the same work off the update loop, see
// startWorkspaceScan.
func (m *Model) checkWorkspace() {
	m.syncUserTheme()
	m.syncOpenBuffers()
	m.applyScannedTree(m.store.List())
}

// applyScannedTree refreshes the explorer when pages differ from the tree
// on screen, keeping the cursor on the same note even if rows above it came
// or went. That also covers a rename in progress: it applies to the
// selected note, which is still the same one after the refresh.
func (m *Model) applyScannedTree(pages []core.Page) {
	if treeSignature(pages) == m.treeSig {
		// Notes edited outside keep the tree as it was, but their links
		// may have changed: the graph and backlinks read them again.
		if m.linkIndexOutOfDate(pages) {
			m.refreshGraph()
		}
		return
	}
	selected, ok := m.sidebar.SelectedPage()
	m.applyTree(pages)
	if ok {
		m.sidebar.SetSelectedID(selected.ID)
	}
}

// workspaceScannedMsg is what a workspace scan read from disk. Every part
// carries what it was read against, so Update drops what went out of date
// while the scan ran.
type workspaceScannedMsg struct {
	themeStamp string
	treeGen    uint64 // m.treeGen when the scan was sent
	pages      []core.Page
	notes      []scannedNote
}

// scannedNote is one open note as a scan found it on disk.
type scannedNote struct {
	b       *buffer
	id      string // b.id when the scan was sent
	saves   uint64 // b.saves when the scan was sent
	gone    bool
	content string
	read    bool // content was read; false when the file could not be
}

// startWorkspaceScan reads the disk for the periodic check in a command, so
// Update never waits on it. Only one scan runs at a time: while one is in
// flight this returns nil, and the next check is scheduled when its result
// arrives (see applyWorkspaceScan), not when it is sent.
func (m *Model) startWorkspaceScan() tea.Cmd {
	if m.scanning || m.store == nil {
		return nil
	}
	m.scanning = true
	store, gen := m.store, m.treeGen
	var notes []scannedNote
	for _, b := range m.buffers {
		if b.draft() || b.isGraph() {
			continue
		}
		notes = append(notes, scannedNote{b: b, id: b.id, saves: b.saves})
	}
	return func() tea.Msg {
		msg := workspaceScannedMsg{themeStamp: theme.UserThemesStamp(), treeGen: gen}
		for i := range notes {
			n := &notes[i]
			if n.gone = noteGoneAt(store.Root(), n.id); n.gone {
				continue
			}
			if p, err := store.Get(n.id); err == nil {
				n.content, n.read = p.Content, true
			}
		}
		msg.notes = notes
		msg.pages = store.List()
		return msg
	}
}

// applyWorkspaceScan applies a scan's result. A note saved, reloaded or
// renamed after the scan was sent keeps what it has: the scan read it
// before that and would undo it. The tree is skipped when it was redrawn
// meanwhile, and the next scan reads it again.
func (m *Model) applyWorkspaceScan(msg workspaceScannedMsg) {
	m.scanning = false
	m.applyThemeStamp(msg.themeStamp)
	for _, n := range msg.notes {
		if i := m.bufferIndex(n.id); i < 0 || m.buffers[i] != n.b {
			continue // closed, or renamed by Misoku
		}
		if n.b.saves != n.saves {
			continue // saved or reloaded while the scan ran
		}
		m.applyScannedNote(n.b, n.gone, n.content, n.read)
	}
	m.refreshModified()
	if msg.treeGen == m.treeGen {
		m.applyScannedTree(msg.pages)
	}
}

// syncOpenBuffers brings open notes up to date with the disk, reading it
// where it is called. See applyScannedNote.
func (m *Model) syncOpenBuffers() {
	for _, b := range m.buffers {
		if b.draft() || b.isGraph() {
			continue
		}
		gone := m.noteGone(b)
		var content string
		read := false
		if !gone {
			if p, err := m.store.Get(b.id); err == nil {
				content, read = p.Content, true
			}
		}
		m.applyScannedNote(b, gone, content, read)
	}
	m.refreshModified()
}

// applyScannedNote brings an open note up to date with what is on disk. A
// note with no unsaved edits is reloaded in place, keeping the cursor; one
// with unsaved edits is left alone and the user is told, on every check
// while the conflict lasts; saving it keeps the disk's version in a
// conflict copy (see saveBuffer).
// A note that is no longer on disk keeps its tab and text; the user is told
// once, when it goes, and saving it asks before recreating it.
func (m *Model) applyScannedNote(b *buffer, gone bool, content string, read bool) {
	if gone {
		// Warn only on the change: the tab stays gone until the user acts,
		// and repeating it every check would bury every other message.
		if !b.gone {
			b.gone = true
			m.warnGone(b)
		}
		return
	}
	b.gone = false
	if !read || content == b.saved {
		return
	}
	if m.isDirty(b) {
		m.setError(fmt.Sprintf("%q changed on disk; saving keeps both versions", b.id))
		return
	}
	b.setSaved(content)
	// D7: what was undoable is the text the disk replaced, so the tab's
	// history starts over from the reloaded text.
	b.hist = nil
	if b.id == m.active {
		m.content.SetHistory(b.history())
		m.content.ReplaceText(content)
		m.content.ResetHistory()
		m.preview.SetPage(b.page())
		m.refreshLinks()
	}
	m.setStatus(fmt.Sprintf("%q reloaded: it changed on disk", b.id))
}

// syncUserTheme re-reads the user theme files when they changed on disk (a
// tool such as Omarchy's theme switcher rewrites them) and re-applies the
// theme in use, without saving anything. A bundled theme comes out the same,
// since bundled names win over user files.
func (m *Model) syncUserTheme() {
	m.applyThemeStamp(theme.UserThemesStamp())
}

// applyThemeStamp reloads the user themes when stamp, read from disk, is not
// the one last seen.
func (m *Model) applyThemeStamp(stamp string) {
	if stamp == m.themeStamp {
		return
	}
	m.themeStamp = stamp
	theme.ReloadUserThemes()
	if th, ok := theme.Get(m.theme.Name); ok {
		m.applyTheme(th, false)
	}
}
