package app_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// S1: clicking the box of a to-do line in the preview toggles it, as one
// change u undoes; clicking the text of the line does not toggle it.
func TestPreviewCheckboxClickTogglesTheLine(t *testing.T) {
	look := preview.DefaultLook()
	h := openWith(t, "- [ ] alpha")

	h.clickText(look.Unchecked)
	if got := h.editor(); got != "- [x] alpha" {
		t.Fatalf("clicking the box should check the line: %q", got)
	}
	h.clickText(look.Checked)
	if got := h.editor(); got != "- [ ] alpha" {
		t.Fatalf("clicking the box again should uncheck the line: %q", got)
	}

	h.clickText(look.Unchecked)
	h.key(tea.KeyEsc) // back to the editor, where u undoes
	h.keys("u")
	if got := h.editor(); got != "- [ ] alpha" {
		t.Fatalf("one u should undo the box: %q", got)
	}

	h.clickText("alpha")
	if got := h.editor(); got != "- [ ] alpha" {
		t.Fatalf("clicking the text of the line should not toggle it: %q", got)
	}
}

// A4: a box clicked while the preview shows another note (the graph) edits
// nothing of the editor.
func TestPreviewCheckboxKeepsTheActiveNote(t *testing.T) {
	look := preview.DefaultLook()
	store := newTestStore(t)
	for _, n := range [][2]string{
		{"Center", "plain text"},
		{"Other", "- [ ] todo\n[[Center]]"},
	} {
		if _, err := store.SaveAs("", n[0], n[1]); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("\\")
	h.keys("center")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.keys(":graph")
	h.key(tea.KeyEnter)
	h.clickGraph("Other") // the graph previews it; Center is still the note

	h.clickText(look.Unchecked)
	if got := h.editor(); got != "" {
		t.Fatalf("a box of another note must edit nothing, got %q", got)
	}
}
