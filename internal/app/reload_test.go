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

func TestUnsavedEditsAreNotReplacedAndSaveAsksFirst(t *testing.T) {
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
	if v := h.view(); !strings.Contains(v, "Overwrite") {
		t.Fatalf("saving over an outside change must ask first:\n%s", v)
	}
	if got := readNote(t, store.Root(), "doc.md"); got != "theirs" {
		t.Fatalf("nothing is written before the answer, got %q", got)
	}
	h.key(tea.KeyEsc) // cancel
	if got := readNote(t, store.Root(), "doc.md"); got != "theirs" {
		t.Fatalf("cancel must keep the outside text, got %q", got)
	}

	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	h.key(tea.KeyEnter) // Overwrite
	if got := readNote(t, store.Root(), "doc.md"); got != "base mine" {
		t.Fatalf("overwrite must write the editor text, got %q", got)
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
