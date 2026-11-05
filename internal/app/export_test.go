package app

import (
	"os/exec"
	"time"

	"github.com/pwn27sbx/misoku/internal/dirplan"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
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

// ExecOutside runs a program outside the screen through the app's hook, as the
// GitHub sign-in does, so a test can see what the terminal is told around it.
func ExecOutside(tm tea.Model, cmd *exec.Cmd, cb tea.ExecCallback) tea.Cmd {
	m := tm.(Model)
	return m.execOutside(cmd, cb)
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

// PreviewCompiles counts the preview recompiles the editor's text caused.
func PreviewCompiles(tm tea.Model) int { return tm.(Model).pv.compiles }

// GraphLinkScans counts how often the live graph re-read the edited note's
// links.
func GraphLinkScans(tm tea.Model) int { return tm.(Model).pv.linkScans }

// PreviewGen is the generation of the latest preview refresh scheduled.
func PreviewGen(tm tea.Model) int { return tm.(Model).pv.gen }

// PreviewTick is the message the preview refresh timer of generation gen
// delivers.
func PreviewTick(gen int) tea.Msg { return PreviewRefreshMsg{gen: gen} }

// TextCopies counts the copies of the editor's whole text made so far.
func TextCopies(tm tea.Model) int64 { return tm.(Model).content.TextCopies() }

// CountWords is the status line's word count of s.
var CountWords = countWords

// LeaderGuideDelay is how long after Space the leader guide waits to show.
const LeaderGuideDelay = leaderGuideDelay

// LeaderSeq is the sequence number of the latest Space press.
func LeaderSeq(tm tea.Model) int { return tm.(Model).leaderSeq }

// LeaderGuideTick is the message the leader guide timer of the Space press
// numbered seq delivers.
func LeaderGuideTick(seq int) tea.Msg { return leaderGuideMsg{seq: seq} }

// ExplorerHideWidth is the explorer hide width the app holds, set by
// config ui.explorer_hide_width and the settings menu.
func (m Model) ExplorerHideWidth() int { return m.prefs().hideWidth }

// EditorText is the editor's text, what the active tab holds before it is
// saved.
func EditorText(tm tea.Model) string { return tm.(Model).content.Value() }

// MediaPickedForTest is the message the native file dialog produces, so a test
// can run an attachment without a desktop dialog.
func MediaPickedForTest(kind, path string) tea.Msg {
	return mediaPickedMsg{kind: kind, path: path}
}

// ExplorerSelected is the ID of the row selected in the explorer.
func ExplorerSelected(tm tea.Model) string {
	m := tm.(Model)
	p, _ := m.sidebar.SelectedPage()
	return p.ID
}

// ExplorerView is the explorer pane as drawn, without colours.
func ExplorerView(tm tea.Model) string {
	m := tm.(Model)
	return ansi.Strip(m.sidebar.View())
}

// CommitDue reports whether a save is waiting for the local commit that
// follows it (see noteSaved).
func CommitDue(tm tea.Model) bool { return tm.(Model).commitDue }

// ExpireToast hides the toast showing now, as its timer would.
func ExpireToast(tm tea.Model) tea.Model {
	m := tm.(Model)
	next, _ := m.Update(toastExpiredMsg{id: m.toastSeq})
	return next
}

// RequestDirPlan hands the edited folders of the explorer's folder editor
// to the app, as the editor's = does, and returns the planning command.
func RequestDirPlan(tm tea.Model, folders []dirplan.Folder) (tea.Model, tea.Cmd) {
	m := tm.(Model)
	cmd := m.requestDirPlan(folders)
	return m, cmd
}

// OpenFile opens id in a tab of its own, as picking it in the explorer does.
func OpenFile(tm tea.Model, id string) (tea.Model, tea.Cmd) {
	m := tm.(Model)
	cmd := m.openFile(id)
	return m, cmd
}

// OpenTabs lists the IDs of the open tabs, in tab order.
func OpenTabs(tm tea.Model) []string {
	var ids []string
	for _, b := range tm.(Model).buffers {
		ids = append(ids, b.id)
	}
	return ids
}

// StatusText is the message the status bar shows, whole.
func StatusText(tm tea.Model) string { return tm.(Model).status }

// SetHasTrash makes the app see a system with or without a Trash and
// returns the function that puts the real check back.
func SetHasTrash(has bool) (restore func()) {
	prev := hasTrash
	hasTrash = func() bool { return has }
	return func() { hasTrash = prev }
}

// DirPlanStoppedMsg is the result of a Stage A batch that stopped at failed
// and whose link rewrite then skipped the notes skipped and failed with err.
func DirPlanStoppedMsg(failed error, skipped []string, err error) tea.Msg {
	b := &dirBatch{failed: failed}
	return relinksAppliedMsg{job: relinkJob{renamed: b.summary(), batch: b}, skipped: skipped, err: err}
}
