package app

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dirplan"

	tea "github.com/charmbracelet/bubbletea"
)

// hasTrash is core.HasTrash, replaceable in tests to see a system without
// a Trash (Windows).
var hasTrash = core.HasTrash

// dirPlanShown is how many actions the confirm lists before "…and N more".
const dirPlanShown = 8

// dirBatch is a Stage A batch (S8) as it goes from the plan to the answer
// and the disk. It rides on a relinkJob, so the link count is asked in the
// same dialog (Q1) and the tabs and links follow through the rename flow.
type dirBatch struct {
	plan dirplan.Plan
	err  error // the plan was refused: nothing is asked or written
	// done counts the ops applied; failed is the error that stopped the
	// batch at op done, nil when every op ran.
	done   int
	failed error
}

// requestDirPlan is what the folder editor's = asks for: it plans the
// edited folders against the workspace in a command (Update never waits
// on the disk), with the links the batch would rewrite, and the answer
// comes back as a relinksPlannedMsg that shows the confirm.
func (m *Model) requestDirPlan(folders []dirplan.Folder) tea.Cmd {
	store := m.store
	return func() tea.Msg {
		pages := store.List()
		plan, err := dirplan.Build(pages, folders)
		job := relinkJob{before: notesOf(pages), batch: &dirBatch{plan: plan, err: err}}
		if err != nil {
			return relinksPlannedMsg{job: job}
		}
		job.moves = settleMoves(job.before, planMoves(plan.Ops))
		gone := deletedIDs(plan.Ops)
		final := map[string]string{}
		for _, mv := range core.ExpandMoves(job.before, job.moves) {
			final[mv.From] = mv.To
		}
		content := map[string]string{}
		for _, p := range job.before {
			to, moved := final[p.ID]
			if !moved {
				to = p.ID
			}
			if gone(to) {
				continue // a deleted note's links need no rewrite
			}
			if got, err := store.Get(p.ID); err == nil {
				content[p.ID] = got.Content
			}
		}
		return relinksPlannedMsg{job: job, relinks: core.PlanRelinks(job.before, job.moves, content)}
	}
}

// confirmDirPlan shows "CONFIRM FILE SYSTEM ACTIONS" for a planned batch,
// listing its actions and the links it updates; Yes applies them all. With
// no Trash the confirm says a delete cannot be undone and takes a
// deliberate choice.
func (m *Model) confirmDirPlan(msg relinksPlannedMsg) {
	batch := msg.job.batch
	switch {
	case batch.err != nil:
		m.setError("Can't apply: " + batch.err.Error())
		return
	case len(batch.plan.Ops) == 0:
		m.setStatus("No changes")
		return
	}
	var lines []string
	deletes := false
	for i, op := range batch.plan.Ops {
		deletes = deletes || op.Kind == dirplan.OpDelete
		if i < dirPlanShown {
			lines = append(lines, describeOp(op))
		}
	}
	if n := len(batch.plan.Ops) - dirPlanShown; n > 0 {
		lines = append(lines, fmt.Sprintf("…and %d more", n))
	}
	if len(msg.relinks) > 0 {
		lines = append(lines, "Updates "+countLinks(msg.relinks)+".")
	}
	permanent := deletes && !hasTrash()
	switch {
	case permanent:
		lines = append(lines, "Deleted entries are removed from disk, cannot be undone.")
	case deletes:
		lines = append(lines, "Deleted entries go to the Trash.")
	}
	m.confirm = &confirmDialog{
		title:      "CONFIRM FILE SYSTEM ACTIONS",
		message:    lines,
		buttons:    []string{"Yes", "No"},
		danger:     deletes,
		deliberate: permanent,
		onChoose: func(m *Model, choice int) tea.Cmd {
			if choice != 0 {
				return nil
			}
			store, job := m.store, msg.job
			return func() tea.Msg { return applyDirPlan(store, job) }
		},
	}
}

// applyDirPlan runs the batch's ops in order and stops at the first that
// fails, then rewrites the links the ops that ran call for, once for the
// whole batch (S1, S2). It runs in a command, off the update loop.
func applyDirPlan(store *core.Store, job relinkJob) relinksAppliedMsg {
	b := *job.batch
	for _, op := range b.plan.Ops {
		if err := applyDirOp(store, op); err != nil {
			b.failed = fmt.Errorf("%s: %w", describeOp(op), err)
			break
		}
		b.done++
	}
	job.batch = &b
	job.moves = settleMoves(job.before, planMoves(b.plan.Ops[:b.done]))
	job.renamed = b.summary()
	planned := planRelinksIn(store, job)
	done, skipped, err := store.ApplyRelinks(planned.relinks)
	return relinksAppliedMsg{job: job, done: done, skipped: skipped, err: err}
}

// applyDirOp runs one op through the store: n's CreateEntry for new
// folders and notes, Rename, Move (then Rename when the name changes too)
// and Delete, which moves to the Trash where there is one.
func applyDirOp(store *core.Store, op dirplan.Op) error {
	switch op.Kind {
	case dirplan.OpMkdir, dirplan.OpCreate:
		input := op.To
		if op.Kind == dirplan.OpMkdir {
			input += "/"
		}
		e, err := store.ParseNewEntry("", input)
		if err != nil {
			return err
		}
		_, err = store.CreateEntry(e)
		return err
	case dirplan.OpRename:
		_, err := store.Rename(op.From, titleOf(op.To))
		return err
	case dirplan.OpMove:
		p, err := store.Move(op.From, destParent(store, op.To))
		if err == nil && path.Base(p.ID) != path.Base(op.To) {
			_, err = store.Rename(p.ID, titleOf(op.To))
		}
		return err
	case dirplan.OpDelete:
		return store.Delete(op.From)
	}
	return fmt.Errorf("unknown action %s", op.Kind)
}

// finishDirPlan brings the screen up to date after a batch: tabs follow
// the moves and renames, deleted notes' tabs close, the tree is read again
// and the batch counts as a save for sync. finishRelinks then rewrites the
// tabs of the notes whose links changed.
func (m *Model) finishDirPlan(job relinkJob) {
	m.followMoves(job.moves)
	for _, op := range job.batch.plan.Ops[:job.batch.done] {
		if op.Kind == dirplan.OpDelete {
			m.dropBuffersUnder(op.From)
		}
	}
	m.reloadTree()
	if job.batch.done > 0 {
		m.noteSaved()
	}
}

// summary is the status a batch leaves: "Applied N actions", or where it
// stopped, why, and what was not done.
func (b dirBatch) summary() string {
	if b.failed == nil {
		return fmt.Sprintf("Applied %d %s", b.done, plural(b.done, "action", "actions"))
	}
	var left []string
	for _, op := range b.plan.Ops[b.done:] {
		left = append(left, describeOp(op))
	}
	return fmt.Sprintf("Stopped after %d of %d actions: %v · not done: %s", b.done, len(b.plan.Ops), b.failed, strings.Join(left, ", "))
}

// describeOp is an op as the confirm lists it: "move a.md → Docs/a.md".
func describeOp(op dirplan.Op) string {
	shown := func(id string) string {
		if op.Folder {
			return id + "/"
		}
		return id
	}
	switch op.Kind {
	case dirplan.OpMkdir, dirplan.OpCreate:
		return op.Kind.String() + " " + shown(op.To)
	case dirplan.OpDelete:
		return op.Kind.String() + " " + shown(op.From)
	}
	return op.Kind.String() + " " + shown(op.From) + " → " + shown(op.To)
}

// planMoves are the renames and moves of ops, in plan order.
func planMoves(ops []dirplan.Op) []core.Move {
	var moves []core.Move
	for _, op := range ops {
		if op.Kind == dirplan.OpRename || op.Kind == dirplan.OpMove {
			moves = append(moves, core.Move{From: op.From, To: op.To})
		}
	}
	return moves
}

// deletedIDs reports whether an ID, as it stands after the batch's moves,
// is deleted by the batch: the entry itself or anything under it.
func deletedIDs(ops []dirplan.Op) func(string) bool {
	var gone []string
	for _, op := range ops {
		if op.Kind == dirplan.OpDelete {
			gone = append(gone, op.From)
		}
	}
	return func(id string) bool {
		for _, g := range gone {
			if id == g || strings.HasPrefix(id, strings.TrimSuffix(g, ".md")+"/") {
				return true
			}
		}
		return false
	}
}

// notesOf keeps the notes of pages, the list the link rewrite plans on.
func notesOf(pages []core.Page) []core.Page {
	var notes []core.Page
	for _, p := range pages {
		if !p.IsFolder {
			notes = append(notes, p)
		}
	}
	return notes
}

// titleOf is the name an ID gives its entry: "Docs/a.md" → "a".
func titleOf(id string) string { return strings.TrimSuffix(path.Base(id), ".md") }

// destParent is the parent ID Store.Move takes for an entry that ends at
// to: "" for the root, the folder, or the note whose sub-notes it joins.
func destParent(store *core.Store, to string) string {
	dir := path.Dir(to)
	if dir == "." {
		return ""
	}
	if info, err := os.Stat(filepath.Join(store.Root(), filepath.FromSlash(dir))); err == nil && info.IsDir() {
		return dir
	}
	return dir + ".md"
}
