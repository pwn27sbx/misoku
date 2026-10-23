package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// openNotes opens every note (name -> text) in its own tab, in order, and
// leaves the editor on the last one in NORMAL mode.
func openNotes(t *testing.T, notes ...string) *harness {
	t.Helper()
	store := newTestStore(t)
	for _, n := range notes {
		if _, err := store.SaveAs("", n, n+" body"); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e") // explorer, on the first note
	h.key(tea.KeyEnter)
	for range notes[1:] {
		// The explorer opens a new tab; the finder would reuse a clean one.
		h.key(tea.KeyEsc)
		h.keys(" e")
		h.keys("j")
		h.key(tea.KeyEnter)
	}
	h.key(tea.KeyEsc)
	return h
}

func (h *harness) editor() string { h.t.Helper(); return app.EditorText(h.m) }

// S7: u in the editor undoes, Ctrl+R redoes, the bar says which change,
// and nothing is written to disk.
func TestUndoRedoInTheEditorNeverWritesToDisk(t *testing.T) {
	h := openNotes(t, "alpha")
	h.keys("A!")
	h.key(tea.KeyEsc)
	h.keys("u")
	if got := h.editor(); got != "alpha body" {
		t.Fatalf("u: editor holds %q", got)
	}
	if v := h.view(); !strings.Contains(v, "1 change; before #1") {
		t.Fatalf("the bar should say which change was undone:\n%s", v)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlR})
	if got := h.editor(); got != "alpha body!" {
		t.Fatalf("Ctrl+R: editor holds %q", got)
	}
	if got := readNote(t, h.store.Root(), "alpha.md"); got != "alpha body" {
		t.Fatalf("undo and redo must not write to disk, file has %q", got)
	}
}

// S7: undoing back to the saved text clears the unsaved mark; redoing
// brings it back.
func TestUndoToSavedTextIsNotDirty(t *testing.T) {
	h := openNotes(t, "alpha")
	h.keys("A!")
	h.key(tea.KeyEsc)
	if !h.tabDirty("alpha.md") {
		t.Fatal("typing should mark the tab")
	}
	h.keys("u")
	if h.dirty() {
		t.Fatalf("undoing to the saved text leaves nothing unsaved:\n%s", h.view())
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlR})
	if !h.tabDirty("alpha.md") {
		t.Fatal("redo makes the tab unsaved again")
	}
}

// S7: each tab keeps its own history, and switching tabs is not a change.
func TestUndoHistoryIsPerTab(t *testing.T) {
	h := openNotes(t, "alpha", "beta")
	h.keys("A2")
	h.key(tea.KeyEsc)
	h.keys("[") // to alpha
	if got := h.editor(); got != "alpha body" {
		t.Fatalf("expected alpha, editor holds %q", got)
	}
	h.keys("A1")
	h.key(tea.KeyEsc)
	h.keys("]") // to beta
	h.keys("[") // and back
	h.keys("u")
	if got := h.editor(); got != "alpha body" {
		t.Fatalf("u in alpha: editor holds %q", got)
	}
	h.keys("u")
	if got := h.editor(); got != "alpha body" {
		t.Fatalf("alpha's history must not hold beta's change, editor holds %q", got)
	}
	h.keys("]")
	if got := h.editor(); got != "beta body2" {
		t.Fatalf("beta keeps its edit, editor holds %q", got)
	}
	h.keys("u")
	if got := h.editor(); got != "beta body" {
		t.Fatalf("u in beta: editor holds %q", got)
	}
}

// S7: an INSERT session left open when the tab changes is still one change
// of that tab.
func TestInsertSessionOpenAcrossTabSwitchStaysInItsTab(t *testing.T) {
	h := openNotes(t, "alpha", "beta")
	h.keys("A2")
	h.key(tea.KeyEsc)
	h.keys("[") // alpha
	h.keys("A1")
	// Leave alpha while still typing: the tab bar works with the mouse.
	x, y := find(t, h.view(), "beta.md")
	h.click(x, y)
	h.key(tea.KeyEsc)
	if got := h.editor(); got != "beta body2" {
		t.Fatalf("setup: expected beta, editor holds %q", got)
	}
	h.keys("u")
	if got := h.editor(); got != "beta body" {
		t.Fatalf("u in beta undoes beta's change, editor holds %q", got)
	}
	h.keys("[")
	h.keys("u")
	if got := h.editor(); got != "alpha body" {
		t.Fatalf("u in alpha undoes alpha's change, editor holds %q", got)
	}
}

// S7 / D7: a reload from disk empties that tab's history, active or not.
func TestReloadFromDiskClearsUndo(t *testing.T) {
	h := openNotes(t, "alpha", "beta")
	h.keys("A2")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	h.keys("[")
	h.keys("A1")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})

	writeOutside(t, h.store.Root(), "alpha.md", "alpha outside")
	writeOutside(t, h.store.Root(), "beta.md", "beta outside")
	h.send(app.CheckWorkspaceMsg{})
	if got := h.editor(); got != "alpha outside" {
		t.Fatalf("alpha should be reloaded, editor holds %q", got)
	}
	h.keys("u")
	if got := h.editor(); got != "alpha outside" {
		t.Fatalf("a reload cannot be undone (active tab), editor holds %q", got)
	}
	if v := h.view(); !strings.Contains(v, "Already at oldest change") {
		t.Fatalf("expected the oldest-change message:\n%s", v)
	}
	h.keys("]")
	h.keys("u")
	if got := h.editor(); got != "beta outside" {
		t.Fatalf("a reload cannot be undone (other tab), editor holds %q", got)
	}
}

// S7: edits made from the preview (tables, boards) are undone with u.
func TestPreviewEditsAreUndone(t *testing.T) {
	const table = "| Name | Role |\n| --- | --- |\n| Ada | Eng |"
	h := openWith(t, table)
	h.clickText("+ Add row")
	h.key(tea.KeyEsc) // back to the editor
	if got := h.editor(); got == table {
		t.Fatal("setup: + Add row changed nothing")
	}
	h.keys("u")
	if got := h.editor(); got != table {
		t.Fatalf("u should undo the table edit, editor holds %q", got)
	}

	const board = "```board\n## Todo\n- Write\n## Done\n- Ship\n```"
	h = openWith(t, board)
	h.clickLeftmost("+ New page")
	h.keys("Review")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	if got := h.editor(); !strings.Contains(got, "- Review") {
		t.Fatalf("setup: card not added, editor holds %q", got)
	}
	h.keys("u")
	if got := h.editor(); got != board {
		t.Fatalf("u should undo the board edit, editor holds %q", got)
	}
}

// S7: setting a cover from the "/" menu is undone with the INSERT session
// that asked for it.
func TestCoverFromSlashMenuIsUndone(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
	t.Setenv("MISOKU_NATIVE_PICKER", "0")
	_ = os.MkdirAll(filepath.Join(home, "Downloads"), 0o755)
	if err := os.WriteFile(filepath.Join(home, "Downloads", "gcp.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := newTestStore(t)
	if _, err := store.SaveAs("", "ace", "# GCP- ACE"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.keys("o/cover")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEnter) // pick gcp.png
	if got := h.editor(); !strings.HasPrefix(got, "---\ncover: ") {
		t.Fatalf("setup: no cover, editor holds %q", got)
	}
	h.key(tea.KeyEsc)
	h.keys("u")
	if got := h.editor(); got != "# GCP- ACE" {
		t.Fatalf("u should undo the cover, editor holds %q", got)
	}
}

// S7: renaming a note keeps its history.
func TestRenameKeepsUndo(t *testing.T) {
	h := openNotes(t, "alpha")
	h.keys("A!")
	h.key(tea.KeyEsc)
	h.keys(" e")
	h.keys("r")
	for range "alpha" {
		h.key(tea.KeyBackspace)
	}
	h.keys("renamed")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc) // back to the editor
	if v := h.view(); !strings.Contains(strings.Split(v, "\n")[0], "renamed.md") {
		t.Fatalf("setup: tab not renamed:\n%s", v)
	}
	h.keys("u")
	if got := h.editor(); got != "alpha body" {
		t.Fatalf("u after a rename: editor holds %q", got)
	}
}

// S7: saving a new note under a name keeps its history.
func TestSaveAsKeepsUndo(t *testing.T) {
	store := newTestStore(t)
	h := newHarness(t, store)
	h.keys("n")
	h.keys("hello")
	h.key(tea.KeyEsc)
	h.keys(":w")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEnter) // save as Untitled-1 at the root
	if _, err := store.Get("Untitled-1.md"); err != nil {
		t.Fatalf("setup: not saved: %v\n%s", err, h.view())
	}
	h.keys("u")
	if got := h.editor(); got != "" {
		t.Fatalf("u after Save As: editor holds %q", got)
	}
}

// S7: a tab the finder reuses for another note starts a new history: the
// first note's text never comes back in the second.
func TestReusedTabStartsANewHistory(t *testing.T) {
	h := openNotes(t, "alpha")
	if _, err := h.store.SaveAs("", "gamma", "gamma body"); err != nil {
		t.Fatal(err)
	}
	h.keys("A!")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS}) // clean, so the tab is reused
	var cmd tea.Cmd
	h.m, cmd = app.OpenInCurrentBuffer(h.m, "gamma.md")
	_ = cmd
	h.key(tea.KeyEsc)
	if v := h.view(); strings.Contains(strings.Split(v, "\n")[0], "alpha.md") {
		t.Fatalf("setup: the tab should now be gamma:\n%s", v)
	}
	h.keys("u")
	if got := h.editor(); got != "gamma body" {
		t.Fatalf("u must not bring alpha's text into gamma, editor holds %q", got)
	}
}
