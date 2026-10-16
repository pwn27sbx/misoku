package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

func readNote(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestOpenNoteReloadsWhenChangedOnDisk(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "first version"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1") // open the recent note
	h.key(tea.KeyEsc)

	writeOutside(t, store.Root(), "doc.md", "second version from outside")
	h.send(app.CheckWorkspaceMsg{})

	v := h.view()
	if !strings.Contains(v, "second version from outside") {
		t.Fatalf("the open note should show the new text:\n%s", v)
	}
	// Reloaded, not edited: saving must not ask about a conflict and must
	// leave the file as it is.
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if strings.Contains(h.view(), "changed on disk") {
		t.Fatalf("a reloaded note is in sync with the disk:\n%s", h.view())
	}
	if got := readNote(t, store.Root(), "doc.md"); got != "second version from outside" {
		t.Fatalf("file should keep the outside text, got %q", got)
	}
}

// copiesOf returns the text of each conflict copy of stem+".md" in dir.
func copiesOf(t *testing.T, dir, stem string) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, stem+" (conflict *).md"))
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		texts = append(texts, string(b))
	}
	return texts
}

func TestUnsavedEditsAreNotReplacedAndSaveKeepsBothVersions(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.keys("A mine")
	h.key(tea.KeyEsc)

	writeOutside(t, store.Root(), "doc.md", "theirs")
	h.send(app.CheckWorkspaceMsg{})

	v := h.view()
	if !strings.Contains(v, "base mine") {
		t.Fatalf("unsaved edits must stay in the editor:\n%s", v)
	}
	if !strings.Contains(v, "changed on disk") {
		t.Fatalf("the user should be told the file changed on disk:\n%s", v)
	}

	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if v := h.view(); strings.Contains(v, "Overwrite") {
		t.Fatalf("saving over an outside change keeps both versions, without a question:\n%s", v)
	}
	if got := readNote(t, store.Root(), "doc.md"); got != "base mine" {
		t.Fatalf("the note must hold the editor text, got %q", got)
	}
	if got := copiesOf(t, store.Root(), "doc"); len(got) != 1 || got[0] != "theirs" {
		t.Fatalf("the outside version must be kept in one conflict copy, got %q", got)
	}
	if v := h.view(); !strings.Contains(v, "conflict") {
		t.Fatalf("the user must be told where the outside version went:\n%s", v)
	}
}

func TestSaveWithoutOutsideChangeDoesNotAsk(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.keys("A!")
	h.key(tea.KeyEsc)
	h.send(app.CheckWorkspaceMsg{})
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if v := h.view(); strings.Contains(v, "Overwrite") {
		t.Fatalf("no outside change, no question:\n%s", v)
	}
	if got := readNote(t, store.Root(), "doc.md"); got != "base!" {
		t.Fatalf("got %q", got)
	}
}

func TestReloadKeepsTheCursorLine(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "one\ntwo\nthree"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys("jj") // line 3
	if v := h.view(); !strings.Contains(v, "Ln 3") {
		t.Fatalf("setup: cursor should be on line 3:\n%s", v)
	}
	writeOutside(t, store.Root(), "doc.md", "one\ntwo\nthree\nfour")
	h.send(app.CheckWorkspaceMsg{})
	if v := h.view(); !strings.Contains(v, "four") || !strings.Contains(v, "Ln 3") {
		t.Fatalf("reload should keep the cursor on line 3:\n%s", v)
	}
}

func TestDeleteDialogSaysTheNoteGoesToTheTrash(t *testing.T) {
	if !core.HasTrash() {
		t.Skip("deleting is permanent on this system")
	}
	store := newTestStore(t)
	if _, err := store.SaveAs("", "Doomed", "x"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("e")
	h.keys("d")
	v := h.view()
	if !strings.Contains(v, "moved to the Trash") || strings.Contains(v, "cannot be undone") {
		t.Fatalf("the dialog should say the note goes to the Trash:\n%s", v)
	}
	h.key(tea.KeyEnter)
	if len(store.List()) != 0 {
		t.Fatalf("the note should leave the workspace, got %v", store.List())
	}
}

func TestSaveAllOnQuitKeepsBothVersionsOfOutsideChanges(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"alpha", "beta"} {
		if _, err := store.SaveAs("", n, "base "+n); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e") // explorer, cursor on alpha
	h.key(tea.KeyEnter)
	h.keys("A mine")
	h.key(tea.KeyEsc)
	h.keys(" e")
	h.keys("j") // beta
	h.key(tea.KeyEnter)
	h.keys("A mine")
	h.key(tea.KeyEsc)
	if tabs := strings.Split(h.view(), "\n")[0]; !strings.Contains(tabs, "alpha.md") || !strings.Contains(tabs, "beta.md") {
		t.Fatalf("setup: both notes should be open with edits: %q", tabs)
	}
	writeOutside(t, store.Root(), "alpha.md", "theirs alpha")
	writeOutside(t, store.Root(), "beta.md", "theirs beta")
	h.send(app.CheckWorkspaceMsg{})

	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	h.keys("s") // Save All
	if !h.quit {
		t.Fatalf("Save All keeps both versions and quits:\n%s", h.view())
	}
	for _, n := range []string{"alpha", "beta"} {
		if got := readNote(t, store.Root(), n+".md"); got != "base "+n+" mine" {
			t.Fatalf("%s should hold the edits, got %q", n, got)
		}
		if got := copiesOf(t, store.Root(), n); len(got) != 1 || got[0] != "theirs "+n {
			t.Fatalf("%s's outside version must be kept in a conflict copy, got %q", n, got)
		}
	}
}

// Saving a note with no edits of the user's that changed on disk before
// the poll noticed shows the disk's version: there is nothing to keep two
// of, so no copy is made and nothing is asked.
func TestSavingAnUneditedNoteThatChangedOnDiskShowsTheDiskVersion(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	writeOutside(t, store.Root(), "doc.md", "theirs")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	v := h.view()
	if strings.Contains(v, "Overwrite") || !strings.Contains(v, "theirs") {
		t.Fatalf("the note must show the disk's version without a question:\n%s", v)
	}
	if got := readNote(t, store.Root(), "doc.md"); got != "theirs" {
		t.Fatalf("the disk's version must stay, got %q", got)
	}
	if got := copiesOf(t, store.Root(), "doc"); len(got) != 0 {
		t.Fatalf("no copy for a note without edits, got %q", got)
	}
}
