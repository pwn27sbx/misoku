package app_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

const goneWarning = "deleted or renamed outside Misoku"

func removeOutside(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
		t.Fatal(err)
	}
}

func noteExists(t *testing.T, root, rel string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	return err == nil
}

// openDoc opens doc.md (content base) and, when edit is set, types " mine"
// at the end without saving.
func openDoc(t *testing.T, edit bool) *harness {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	if edit {
		h.keys("A mine")
	}
	h.key(tea.KeyEsc)
	return h
}

func TestDeletedOpenNoteWarnsAndKeepsItsText(t *testing.T) {
	for _, edit := range []bool{false, true} {
		h := openDoc(t, edit)
		removeOutside(t, h.store.Root(), "doc.md")
		h.send(app.CheckWorkspaceMsg{})

		v := h.view()
		if !strings.Contains(v, goneWarning) {
			t.Fatalf("edit=%v: the tab should warn that its note is gone:\n%s", edit, v)
		}
		want := "base"
		if edit {
			want = "base mine"
		}
		if !strings.Contains(v, want) || !strings.Contains(strings.Split(v, "\n")[0], "doc.md") {
			t.Fatalf("edit=%v: the tab must stay open with %q:\n%s", edit, want, v)
		}
		if strings.Contains(v, "Recreate") {
			t.Fatalf("edit=%v: a check must not open a dialog:\n%s", edit, v)
		}
		if noteExists(t, h.store.Root(), "doc.md") {
			t.Fatalf("edit=%v: a check must not recreate the note", edit)
		}
	}
}

func TestRenamedOpenNoteWarnsAndSavingAsksBeforeDuplicating(t *testing.T) {
	h := openDoc(t, true)
	root := h.store.Root()
	if err := os.Rename(filepath.Join(root, "doc.md"), filepath.Join(root, "moved.md")); err != nil {
		t.Fatal(err)
	}
	h.send(app.CheckWorkspaceMsg{})
	if v := h.view(); !strings.Contains(v, goneWarning) || !strings.Contains(v, "moved") {
		t.Fatalf("the tab should warn and the explorer show the new name:\n%s", v)
	}

	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if v := h.view(); !strings.Contains(v, "Recreate") {
		t.Fatalf("saving a note that is gone must ask first:\n%s", v)
	}
	if noteExists(t, root, "doc.md") {
		t.Fatal("nothing is written before the answer")
	}
	h.key(tea.KeyEsc) // Cancel
	if noteExists(t, root, "doc.md") {
		t.Fatal("cancel must not recreate the note")
	}
	if got := readNote(t, root, "moved.md"); got != "base" {
		t.Fatalf("the renamed note must be untouched, got %q", got)
	}
}

func TestRecreatingAGoneNoteWritesTheEditorTextAtItsOldPath(t *testing.T) {
	h := openDoc(t, true)
	root := h.store.Root()
	removeOutside(t, root, "doc.md")
	h.send(app.CheckWorkspaceMsg{})

	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	h.key(tea.KeyEnter) // Recreate
	if got := readNote(t, root, "doc.md"); got != "base mine" {
		t.Fatalf("recreate must write the editor text, got %q", got)
	}
	// Back in sync: the next check and save behave as for any note.
	h.send(app.CheckWorkspaceMsg{})
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if v := h.view(); strings.Contains(v, "Recreate") || strings.Contains(v, "Overwrite") {
		t.Fatalf("a recreated note saves without asking:\n%s", v)
	}
}

func TestVimWriteAsksBeforeRecreatingAGoneNote(t *testing.T) {
	h := openDoc(t, true)
	removeOutside(t, h.store.Root(), "doc.md")
	h.keys(":w")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Recreate") {
		t.Fatalf(":w on a note that is gone must ask first:\n%s", v)
	}
}

func TestGoneWarningIsNotRepeatedOnEveryCheck(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"alpha", "beta"} {
		if _, err := store.SaveAs("", n, "base "+n); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e") // explorer, cursor on alpha
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.keys(" e")
	h.keys("j") // beta
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)

	removeOutside(t, store.Root(), "alpha.md")
	h.send(app.CheckWorkspaceMsg{})
	if v := h.view(); !strings.Contains(v, goneWarning) {
		t.Fatalf("setup: expected the warning:\n%s", v)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS}) // save beta: replaces the status
	h.send(app.CheckWorkspaceMsg{})
	v := h.view()
	if strings.Contains(v, goneWarning) || !strings.Contains(v, "written") {
		t.Fatalf("the warning must not come back on every check:\n%s", v)
	}
}

func TestGoneNoteThatReappearsBehavesAsBefore(t *testing.T) {
	h := openDoc(t, false)
	root := h.store.Root()
	removeOutside(t, root, "doc.md")
	h.send(app.CheckWorkspaceMsg{})
	writeOutside(t, root, "doc.md", "back again")
	h.send(app.CheckWorkspaceMsg{})
	if v := h.view(); !strings.Contains(v, "back again") || !strings.Contains(v, "reloaded") {
		t.Fatalf("a note that came back reloads as usual:\n%s", v)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if v := h.view(); strings.Contains(v, "Recreate") {
		t.Fatalf("a note that came back saves without asking:\n%s", v)
	}
	if got := readNote(t, root, "doc.md"); got != "back again" {
		t.Fatalf("got %q", got)
	}
}

func TestSaveAllOnQuitWaitsForTheRecreateAnswer(t *testing.T) {
	h := openDoc(t, true)
	root := h.store.Root()
	removeOutside(t, root, "doc.md")
	h.send(app.CheckWorkspaceMsg{})

	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	h.keys("s") // Save All
	if h.quit {
		t.Fatal("Save All must wait for the answer, not quit")
	}
	if !strings.Contains(h.view(), "Recreate") {
		t.Fatalf("expected the recreate question:\n%s", h.view())
	}
	h.key(tea.KeyEsc) // Cancel
	if h.quit || noteExists(t, root, "doc.md") {
		t.Fatalf("cancel must stay open and write nothing (quit=%v)", h.quit)
	}

	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	h.keys("s")
	h.key(tea.KeyEnter) // Recreate
	if !h.quit {
		t.Fatalf("after recreating, Save All should quit:\n%s", h.view())
	}
	if got := readNote(t, root, "doc.md"); got != "base mine" {
		t.Fatalf("got %q", got)
	}
}

func TestSavingAGoneNoteOnCloseAsksBeforeRecreating(t *testing.T) {
	h := openDoc(t, true)
	root := h.store.Root()
	removeOutside(t, root, "doc.md")
	h.send(app.CheckWorkspaceMsg{})
	h.keys(" x") // close the tab
	h.keys("s")  // Save
	if v := h.view(); !strings.Contains(v, "Recreate") {
		t.Fatalf("saving on close must ask first:\n%s", v)
	}
	h.key(tea.KeyEnter) // Recreate
	if got := readNote(t, root, "doc.md"); got != "base mine" {
		t.Fatalf("got %q", got)
	}
	if tabs := strings.Split(h.view(), "\n")[0]; strings.Contains(tabs, "doc.md") {
		t.Fatalf("the tab should close once saved: %q", tabs)
	}
}

func TestNoteThatGoesAgainWarnsAgain(t *testing.T) {
	h := openDoc(t, false)
	root := h.store.Root()
	removeOutside(t, root, "doc.md")
	h.send(app.CheckWorkspaceMsg{})
	writeOutside(t, root, "doc.md", "back again")
	h.send(app.CheckWorkspaceMsg{})
	removeOutside(t, root, "doc.md")
	h.send(app.CheckWorkspaceMsg{})
	if v := h.view(); !strings.Contains(v, goneWarning) {
		t.Fatalf("a note that came back and went again must warn again:\n%s", v)
	}
}

func TestRecreateNeverOverwritesANoteThatReappeared(t *testing.T) {
	h := openDoc(t, true)
	root := h.store.Root()
	removeOutside(t, root, "doc.md")
	h.send(app.CheckWorkspaceMsg{})

	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	h.keys("s") // Save All asks to recreate doc.md
	writeOutside(t, root, "doc.md", "theirs")
	h.key(tea.KeyEnter) // Recreate, but the file is back
	if h.quit {
		t.Fatal("a failed recreate must stop Save All and keep Misoku open")
	}
	if got := readNote(t, root, "doc.md"); got != "theirs" {
		t.Fatalf("recreate must not overwrite a file that reappeared, got %q", got)
	}
	if v := h.view(); !strings.Contains(v, "base mine") {
		t.Fatalf("the edits must still be in the editor:\n%s", v)
	}
}
