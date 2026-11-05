package app

import (
	"fmt"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// relinkJob is a rename's link rewrite as it goes from the plan to the
// answer and the disk: what was renamed, the notes as they were before it
// and the moves, so an open tab's unsaved text can be planned the same way.
type relinkJob struct {
	renamed string // the status the rename left: "Renamed to X"
	before  []core.Page
	moves   []core.Move
}

// relinksPlannedMsg carries the rewrites a rename calls for, worked out by
// planRelinks.
type relinksPlannedMsg struct {
	job     relinkJob
	relinks []core.Relink
}

// relinksAppliedMsg carries what ApplyRelinks wrote and skipped.
type relinksAppliedMsg struct {
	job     relinkJob
	done    []core.Relink
	skipped []string
	err     error
}

// planRelinks reads the notes and plans the rewrite of every link the moves
// would break (S1), in a command so Update never waits on the disk. before
// is the note list from before the moves; the files are read where the
// moves put them.
func (m *Model) planRelinks(job relinkJob) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		newID := map[string]string{}
		for _, mv := range core.ExpandMoves(job.before, job.moves) {
			newID[mv.From] = mv.To
		}
		content := map[string]string{}
		for _, p := range job.before {
			id := p.ID
			if to, ok := newID[id]; ok {
				id = to
			}
			if got, err := store.Get(id); err == nil && !got.IsFolder {
				content[p.ID] = got.Content
			}
		}
		return relinksPlannedMsg{job: job, relinks: core.PlanRelinks(job.before, job.moves, content)}
	}
}

// askRelinks asks before rewriting links in other notes (Q1); a rename no
// link names asks nothing. No keeps the rename and leaves the links.
func (m *Model) askRelinks(msg relinksPlannedMsg) {
	if len(msg.relinks) == 0 {
		return
	}
	m.confirm = &confirmDialog{
		title:   "Update " + countLinks(msg.relinks) + "?",
		message: []string{"No keeps the new name and leaves the links as written."},
		buttons: []string{"Yes", "No"},
		onChoose: func(m *Model, choice int) tea.Cmd {
			if choice != 0 {
				return nil
			}
			store := m.store
			return func() tea.Msg {
				done, skipped, err := store.ApplyRelinks(msg.relinks)
				return relinksAppliedMsg{job: msg.job, done: done, skipped: skipped, err: err}
			}
		},
	}
}

// countLinks says how many links in how many notes rs rewrites: "2 links in
// 1 note".
func countLinks(rs []core.Relink) string {
	links := 0
	for _, r := range rs {
		links += r.Links
	}
	return fmt.Sprintf("%d %s in %d %s", links, plural(links, "link", "links"), len(rs), plural(len(rs), "note", "notes"))
}

// finishRelinks brings the open tabs of the rewritten notes up to date
// (S4), counts the rewrite as a save for sync and reports it.
func (m *Model) finishRelinks(msg relinksAppliedMsg) {
	oldID := map[string]string{}
	for _, mv := range core.ExpandMoves(msg.job.before, msg.job.moves) {
		oldID[mv.To] = mv.From
	}
	for _, r := range msg.done {
		from := r.ID
		if id, ok := oldID[r.ID]; ok {
			from = id
		}
		m.relinkTab(msg.job, r, from)
	}
	if len(msg.done) > 0 {
		m.noteSaved()
		m.refreshGraph()
		m.refreshModified()
	}
	status := msg.job.renamed + " · updated " + countLinks(msg.done)
	switch {
	case msg.err != nil:
		m.setError(status + " · " + msg.err.Error())
	case len(msg.skipped) > 0:
		m.setError(status + " · " + strings.Join(msg.skipped, ", ") + " changed meanwhile, its links were left as written")
	default:
		m.setStatus(status)
	}
}

// relinkTab applies r to the open tab of its note, if any. The tab's saved
// text becomes the rewritten one; its live text gets the same rewrite,
// planned on that text so unsaved edits stay (and stay unsaved). In the
// active tab the rewrite is one undo step; another tab replays it when shown
// (see showBuffer). A tab whose saved text is not what was rewritten is out
// of step with the disk and left to the next check.
func (m *Model) relinkTab(job relinkJob, r core.Relink, oldID string) {
	i := m.bufferIndex(r.ID)
	if i < 0 || m.buffers[i].saved != r.From {
		return
	}
	b := m.buffers[i]
	live := m.liveText(b)
	next := r.To
	if live != r.From {
		next = live
		if rs := core.PlanRelinks(job.before, job.moves, map[string]string{oldID: live}); len(rs) > 0 {
			next = rs[0].To
		}
	}
	b.markSaved(r.To)
	if b.id == m.active && m.content.HasPage() {
		m.content.CommitChange()
		m.content.ReplaceText(next)
		m.content.CommitChange()
		b.text, b.textVer = next, m.content.Version()
		m.preview.SetPage(b.page())
		m.refreshLinks()
		return
	}
	b.rewriteText(next)
}
