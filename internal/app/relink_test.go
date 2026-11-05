package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// relinkNotes makes a workspace holding notes (name, text pairs) and opens
// Misoku on it with the explorer focused on the first row.
func relinkNotes(t *testing.T, notes ...string) *harness {
	t.Helper()
	store := newTestStore(t)
	for i := 0; i < len(notes); i += 2 {
		if _, err := store.SaveAs("", notes[i], notes[i+1]); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e")
	return h
}

// renameRow renames the explorer's row (0 = first) from old to name. The
// explorer must have the focus.
func (h *harness) renameRow(row int, old, name string) {
	h.t.Helper()
	h.keys("g")
	h.keys(strings.Repeat("j", row))
	h.keys("r")
	for range old {
		h.key(tea.KeyBackspace)
	}
	h.keys(name)
	h.key(tea.KeyEnter)
}

// openRow opens the explorer's row in a new tab and leaves the editor in
// NORMAL mode.
func (h *harness) openRow(row int) {
	h.t.Helper()
	h.keys("g")
	h.keys(strings.Repeat("j", row))
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
}

func (h *harness) disk(id string) string {
	h.t.Helper()
	data, err := os.ReadFile(filepath.Join(h.store.Root(), filepath.FromSlash(id)))
	if err != nil {
		h.t.Fatal(err)
	}
	return string(data)
}

// S1, S3, Q1: a rename asks before rewriting the links that named the old
// note, then rewrites them and says so.
func TestRenameAsksThenUpdatesLinks(t *testing.T) {
	h := relinkNotes(t, "aaa", "aaa body", "ref", "see [[aaa]] and [x](aaa.md)")
	h.renameRow(0, "aaa", "bbb")

	if v := h.view(); !strings.Contains(v, "Update 2 links in 1 note?") || !strings.Contains(v, "Yes") || !strings.Contains(v, "No") {
		t.Fatalf("expected the Update links question:\n%s", v)
	}
	if got := h.disk("ref.md"); got != "see [[aaa]] and [x](aaa.md)" {
		t.Fatalf("nothing is rewritten before the answer, ref.md = %q", got)
	}
	h.keys("y")
	if got := h.disk("ref.md"); got != "see [[bbb]] and [x](bbb.md)" {
		t.Fatalf("ref.md = %q", got)
	}
	if v := h.view(); !strings.Contains(v, "Renamed to bbb.md · updated 2 links in 1 note") {
		t.Fatalf("expected the status line:\n%s", v)
	}
}

// Q1: No keeps the rename and leaves the links as written.
func TestRenameNoKeepsTheRenameAndTheLinks(t *testing.T) {
	h := relinkNotes(t, "aaa", "aaa body", "ref", "see [[aaa]]")
	h.renameRow(0, "aaa", "bbb")
	h.keys("n")
	if got := h.disk("bbb.md"); got != "aaa body" {
		t.Fatalf("the rename must stay, bbb.md = %q", got)
	}
	if got := h.disk("ref.md"); got != "see [[aaa]]" {
		t.Fatalf("No leaves the links, ref.md = %q", got)
	}
	if v := h.view(); strings.Contains(v, "Update") || strings.Contains(v, "updated") {
		t.Fatalf("no question or update after No:\n%s", v)
	}
}

// Existing behaviour: a rename that no link names asks nothing.
func TestRenameWithoutLinksAsksNothing(t *testing.T) {
	h := relinkNotes(t, "aaa", "aaa body", "ref", "no links here")
	h.renameRow(0, "aaa", "bbb")
	v := h.view()
	if strings.Contains(v, "Update") || !strings.Contains(v, "Renamed to bbb.md") {
		t.Fatalf("expected a plain rename:\n%s", v)
	}
	if got := h.disk("ref.md"); got != "no links here" {
		t.Fatalf("ref.md = %q", got)
	}
}

// S4: a note changed on disk after the rewrite was planned is skipped and
// reported, never clobbered.
func TestRenameSkipsANoteChangedBeforeTheAnswer(t *testing.T) {
	h := relinkNotes(t, "aaa", "aaa body", "ref", "see [[aaa]]", "zed", "z [[aaa]]")
	h.renameRow(0, "aaa", "bbb")
	writeOutside(t, h.store.Root(), "ref.md", "changed [[aaa]]")
	h.keys("y")
	if got := h.disk("ref.md"); got != "changed [[aaa]]" {
		t.Fatalf("a changed note must be left alone, ref.md = %q", got)
	}
	if got := h.disk("zed.md"); got != "z [[bbb]]" {
		t.Fatalf("zed.md = %q", got)
	}
	if v := h.view(); !strings.Contains(v, "updated 1 link in 1 note") || !strings.Contains(v, "ref.md changed meanwhile") {
		t.Fatalf("expected the skip to be reported:\n%s", v)
	}
}

// S5: renaming back makes the links follow back.
func TestRenameBackRestoresTheLinks(t *testing.T) {
	const ref = "see [[aaa]], [[aaa|alias]] and [x](aaa.md#top)"
	h := relinkNotes(t, "aaa", "aaa body", "ref", ref)
	h.renameRow(0, "aaa", "bbb")
	h.keys("y")
	h.renameRow(0, "bbb", "aaa")
	h.keys("y")
	if got := h.disk("ref.md"); got != ref {
		t.Fatalf("renaming back must restore the text, ref.md = %q", got)
	}
}

// S4, S5: open tabs get the rewrite. The active tab with unsaved edits keeps
// them, the rewrite is one undo step, and saving it raises no conflict; an
// inactive clean tab shows the rewrite, also undoable.
func TestRenameRewritesOpenTabsKeepingEditsAndUndo(t *testing.T) {
	h := relinkNotes(t, "aaa", "aaa body", "ref", "see [[aaa]]", "zed", "z [[aaa]]")
	h.openRow(2) // zed
	h.keys(" e")
	h.openRow(1) // ref, active
	h.keys("u")  // a history before the rewrite must survive it
	h.keys("A mine")
	h.key(tea.KeyEsc)

	h.keys(" e")
	h.renameRow(0, "aaa", "bbb")
	h.keys("y")

	if got := h.editor(); got != "see [[bbb]] mine" {
		t.Fatalf("the dirty tab must get the rewrite and keep its edit, editor holds %q", got)
	}
	if got := h.disk("ref.md"); got != "see [[bbb]]" {
		t.Fatalf("only the saved text goes to disk, ref.md = %q", got)
	}
	h.key(tea.KeyEsc)
	h.send(app.CheckWorkspaceMsg{})
	if v := h.view(); strings.Contains(v, "changed on disk") {
		t.Fatalf("a check after the rewrite must not see a conflict:\n%s", v)
	}
	if !h.tabDirty("ref.md") || h.tabDirty("zed.md") {
		t.Fatalf("ref.md stays dirty with the user's edit only, zed.md is clean: %q", h.tabline())
	}

	h.keys("u")
	if got := h.editor(); got != "see [[aaa]] mine" {
		t.Fatalf("u undoes the rewrite as one step, editor holds %q", got)
	}
	h.keys("u")
	if got := h.editor(); got != "see [[aaa]]" {
		t.Fatalf("the user's edit is still undoable after it, editor holds %q", got)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlR}, tea.KeyMsg{Type: tea.KeyCtrlR})
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got := h.disk("ref.md"); got != "see [[bbb]] mine" {
		t.Fatalf("saving writes the tab, ref.md = %q", got)
	}
	if v := h.view(); strings.Contains(v, "changed on disk") || strings.Contains(v, "conflict") {
		t.Fatalf("saving after the rewrite must not raise a conflict:\n%s", v)
	}
	if n := len(h.store.List()); n != 3 {
		t.Fatalf("no conflict copy may be written, the workspace holds %d entries", n)
	}

	h.keys("]") // zed
	if got := h.editor(); got != "z [[bbb]]" {
		t.Fatalf("the inactive tab shows the rewrite, editor holds %q", got)
	}
	h.keys("u")
	if got := h.editor(); got != "z [[aaa]]" {
		t.Fatalf("u undoes the rewrite in the other tab too, editor holds %q", got)
	}
}

// S4: the rewrite counts as a save for sync, so the commit after a save
// takes the rename and the rewrites together.
func TestRenameWithLinksIsCommittedLikeASave(t *testing.T) {
	_, vault := syncedVault(t)
	writeOutside(t, vault, "aaa.md", "aaa body")
	writeOutside(t, vault, "doc.md", "see [[aaa]]")
	h := window(t, vault, syncConfig(t, vault))
	h.keys("e")
	h.renameRow(0, "aaa", "bbb")
	if app.CommitDue(h.m) {
		t.Fatal("nothing is due before the answer")
	}
	h.keys("y")
	if !app.CommitDue(h.m) {
		t.Fatal("the rewrite must count as a save")
	}
}
