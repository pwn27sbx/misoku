package app_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// S7: clicking a property value in the preview, typing and Enter rewrites
// that field only, as one change u undoes.
func TestPropertyValueEditedFromPreview(t *testing.T) {
	const note = "---\nTitle: Plan\nstatus: active\ntags: [go, tui]\n---\n# Body"
	h := openWith(t, note)
	h.clickText("active")
	if v := h.view(); !strings.Contains(v, "Edit property") {
		t.Fatalf("clicking a value should open the prompt:\n%s", v)
	}
	for range "active" {
		h.key(tea.KeyBackspace)
	}
	h.keys("on hold")
	h.key(tea.KeyEnter)
	want := "---\nTitle: Plan\nstatus: on hold\ntags: [go, tui]\n---\n# Body"
	if got := h.editor(); got != want {
		t.Fatalf("only status should change:\n got %q\nwant %q", got, want)
	}
	if !strings.Contains(h.view(), "on hold") {
		t.Fatalf("the preview should show the new value:\n%s", h.view())
	}

	// The key keeps the note's spelling.
	h.clickText("Plan")
	h.keys("!")
	h.key(tea.KeyEnter)
	if got := h.editor(); !strings.Contains(got, "\nTitle: Plan!\n") || strings.Contains(got, "title:") {
		t.Fatalf("Title should keep its spelling: %q", got)
	}

	h.key(tea.KeyEsc) // back to the editor, where u undoes
	h.keys("u")
	if got := h.editor(); got != want {
		t.Fatalf("one u should undo the last property edit:\n got %q\nwant %q", got, want)
	}
}

// S7: list values are shown read-only: clicking them opens nothing.
func TestPropertyListValueIsReadOnly(t *testing.T) {
	const note = "---\ntags: [go, tui]\naliases:\n  - one\n  - two\n---\n# Body"
	h := openWith(t, note)
	for _, v := range []string{"go, tui", "one, two"} {
		h.clickText(v)
		if view := h.view(); strings.Contains(view, "Edit property") {
			t.Fatalf("a list value (%s) should not open the prompt:\n%s", v, view)
		}
	}
	if got := h.editor(); got != note {
		t.Fatalf("the note should not change: %q", got)
	}
}
