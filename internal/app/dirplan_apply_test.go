package app_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/dirplan"

	tea "github.com/charmbracelet/bubbletea"
)

// dirPlanVault opens Misoku on a.md, b.md, d.md, ref.md and Docs/x.md, with
// tabs open on b.md and Docs/x.md.
func dirPlanVault(t *testing.T) *harness {
	t.Helper()
	store := newTestStore(t)
	for _, n := range [][3]string{
		{"", "a", "A"}, {"", "b", "B"}, {"", "d", "D"},
		{"", "ref", "see [[a]] and [x](Docs/x.md)"}, {"Docs", "x", "X"},
	} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	for _, id := range []string{"b.md", "Docs/x.md"} {
		m, cmd := app.OpenFile(h.m, id)
		h.m = m
		if cmd != nil {
			h.send(runCmd(cmd))
		}
	}
	return h
}

// requestDirPlan hands folders to the app and runs the planning command.
func (h *harness) requestDirPlan(folders ...dirplan.Folder) {
	h.t.Helper()
	m, cmd := app.RequestDirPlan(h.m, folders)
	h.m = m
	if cmd != nil {
		h.send(runCmd(cmd))
	}
}

func (h *harness) exists(id string) bool {
	_, err := os.Stat(filepath.Join(h.store.Root(), filepath.FromSlash(id)))
	return err == nil
}

// S8, S4: a batch asks once, listing its actions and the links it updates,
// then applies them in order (the move out of Docs before Docs is
// deleted), tabs follow, deleted tabs close and links are rewritten once.
func TestDirPlanAppliesInOrderTabsAndLinksFollow(t *testing.T) {
	defer app.SetHasTrash(true)()
	h := dirPlanVault(t)
	h.requestDirPlan(dirplan.Folder{ID: "", Rows: []dirplan.Row{
		{OrigID: "a.md", Name: "aa"}, {OrigID: "d.md", Name: "d"},
		{OrigID: "ref.md", Name: "ref"}, {OrigID: "Docs/x.md", Name: "x"},
	}})

	v := h.view()
	for _, want := range []string{
		"CONFIRM FILE SYSTEM ACTIONS", "move Docs/x.md → x.md", "rename a.md → aa.md",
		"delete Docs/", "delete b.md", "Updates 2 links in 1 note", "go to the Trash",
	} {
		if !strings.Contains(v, want) {
			t.Fatalf("the confirm must show %q:\n%s", want, v)
		}
	}
	if !h.exists("a.md") || !h.exists("b.md") {
		t.Fatal("nothing is applied before the answer")
	}
	h.keys("y")

	for id, want := range map[string]bool{"aa.md": true, "x.md": true, "a.md": false, "b.md": false, "Docs": false, "d.md": true} {
		if h.exists(id) != want {
			t.Errorf("%s exists = %v, want %v", id, !want, want)
		}
	}
	if got := h.disk("ref.md"); got != "see [[aa]] and [x](x.md)" {
		t.Errorf("ref.md = %q", got)
	}
	if tabs := app.OpenTabs(h.m); !slices.Equal(tabs, []string{"x.md"}) {
		t.Errorf("tabs = %q, want the moved x.md only (b.md was deleted)", tabs)
	}
	if got := app.StatusText(h.m); got != "Applied 4 actions · updated 2 links in 1 note" {
		t.Errorf("status = %q", got)
	}
}

// S8: No applies nothing.
func TestDirPlanNoAppliesNothing(t *testing.T) {
	h := dirPlanVault(t)
	h.requestDirPlan(dirplan.Folder{ID: "", Rows: []dirplan.Row{
		{OrigID: "b.md", Name: "b"}, {OrigID: "d.md", Name: "d"}, {OrigID: "ref.md", Name: "ref"}, {OrigID: "Docs", Name: "Docs/"},
	}})
	h.keys("n")
	if !h.exists("a.md") {
		t.Fatal("No must leave a.md where it is")
	}
}

// S8: a refused plan says why and writes nothing.
func TestDirPlanRefusalSaysWhy(t *testing.T) {
	h := dirPlanVault(t)
	h.requestDirPlan(dirplan.Folder{ID: "", Rows: []dirplan.Row{
		{OrigID: "a.md", Name: "b"}, {OrigID: "b.md", Name: "b"}, {OrigID: "d.md", Name: "d"},
		{OrigID: "ref.md", Name: "ref"}, {OrigID: "Docs", Name: "Docs/"},
	}})
	if got := app.StatusText(h.m); !strings.Contains(got, "already exists") {
		t.Fatalf("status = %q", got)
	}
	if strings.Contains(h.view(), "CONFIRM") || !h.exists("a.md") {
		t.Fatal("a refused plan asks nothing and writes nothing")
	}
}

// S8: a failing action stops the batch; the message says what was done and
// what was not.
func TestDirPlanStopsAtTheFirstFailure(t *testing.T) {
	defer app.SetHasTrash(true)() // Windows has no Trash; the confirm still takes y
	h := dirPlanVault(t)
	h.requestDirPlan(dirplan.Folder{ID: "", Rows: []dirplan.Row{
		{OrigID: "a.md", Name: "aa"}, {OrigID: "b.md", Name: "b"}, {Name: "n"},
		{OrigID: "ref.md", Name: "ref"}, {OrigID: "Docs", Name: "Docs/"},
	}})
	writeOutside(t, h.store.Root(), "n.md", "made meanwhile")
	h.keys("y")

	if !h.exists("aa.md") || !h.exists("d.md") || h.disk("n.md") != "made meanwhile" {
		t.Fatal("the rename runs, the create fails and the delete after it does not run")
	}
	got := app.StatusText(h.m)
	for _, want := range []string{"Stopped after 1 of 3 actions", "create n.md", "already exists", "not done: create n.md, delete d.md"} {
		if !strings.Contains(got, want) {
			t.Errorf("status %q lacks %q", got, want)
		}
	}
}

// S8: with no Trash (Windows) the confirm says deletes cannot be undone and
// takes a deliberate choice: a letter does not pick Yes.
func TestDirPlanWithoutTrashIsDeliberate(t *testing.T) {
	defer app.SetHasTrash(false)()
	h := dirPlanVault(t)
	h.requestDirPlan(dirplan.Folder{ID: "", Rows: []dirplan.Row{
		{OrigID: "a.md", Name: "a"}, {OrigID: "b.md", Name: "b"}, {OrigID: "ref.md", Name: "ref"}, {OrigID: "Docs", Name: "Docs/"},
	}})
	if v := h.view(); !strings.Contains(v, "removed from disk, cannot be undone") {
		t.Fatalf("expected the no-Trash warning:\n%s", v)
	}
	h.keys("y")
	if !h.exists("d.md") {
		t.Fatal("a letter must not confirm a permanent delete")
	}
	h.key(tea.KeyEnter) // Yes is selected
	if h.exists("d.md") {
		t.Fatal("Enter on Yes deletes d.md")
	}
}

// S8: a long batch lists what fits and counts the rest.
func TestDirPlanListsWhatFits(t *testing.T) {
	h := dirPlanVault(t)
	rows := []dirplan.Row{{OrigID: "a.md", Name: "a"}, {OrigID: "b.md", Name: "b"}, {OrigID: "d.md", Name: "d"}, {OrigID: "ref.md", Name: "ref"}, {OrigID: "Docs", Name: "Docs/"}}
	for _, n := range strings.Split("n1 n2 n3 n4 n5 n6 n7 n8 n9 n10 n11 n12", " ") {
		rows = append(rows, dirplan.Row{Name: n})
	}
	h.requestDirPlan(dirplan.Folder{ID: "", Rows: rows})
	if v := h.view(); !strings.Contains(v, "…and 4 more") {
		t.Fatalf("expected the overflow line:\n%s", v)
	}
}

// S4, S8: a batch that stopped halfway still reports the link rewrite's
// error and the notes it skipped.
func TestDirPlanStoppedReportsLinkErrors(t *testing.T) {
	h := dirPlanVault(t)
	h.send(app.DirPlanStoppedMsg(errors.New("create n.md: boom"), []string{"ref.md"}, errors.New("disk full")))
	got := app.StatusText(h.m)
	for _, want := range []string{"Stopped after 0 of 0 actions: create n.md: boom", "disk full", "ref.md changed meanwhile"} {
		if !strings.Contains(got, want) {
			t.Errorf("status %q lacks %q", got, want)
		}
	}
}
