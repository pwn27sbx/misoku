package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// Unsaved links of the note open before :graph stay in the graph when
// another program edits a different note (e.g. a git sync pull).
func TestGraphKeepsUnsavedLinksThroughAnOutsideEdit(t *testing.T) {
	h := graphWorkspace(t) // Untitled-2 open, graph active, centred on it
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("untitled-2")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.typeAtTop("[t](Untitled-3.md) ") // unsaved
	h.keys(":graph")
	h.key(tea.KeyEnter)
	before := editorPane(h.view())
	if !strings.Contains(before, "Untitled-2 · 2 links ·") {
		t.Fatalf("unsaved link should show:\n%s", before)
	}
	abs := filepath.Join(h.store.Root(), "Other.md")
	if err := os.WriteFile(abs, []byte("[[Untitled-2]] edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	_ = os.Chtimes(abs, later, later)
	h.send(app.CheckWorkspaceMsg{})
	if after := editorPane(h.view()); !strings.Contains(after, "Untitled-2 · 2 links ·") {
		t.Fatalf("unsaved link dropped by an external edit of another note:\n%s", after)
	}
}

// The graph opened from a new, unnamed note shows no unsaved links: an
// outside edit that changes no link must not bring back those of a note
// edited earlier.
func TestGraphFromADraftShowsNoUnsavedLinksAfterAnOutsideEdit(t *testing.T) {
	h := graphWorkspace(t)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("untitled-2")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.typeAtTop("[t](Untitled-3.md) ") // unsaved
	h.ex("enew")
	h.key(tea.KeyEsc)
	h.ex("graph")
	h.keys("---") // every link in view
	before := editorPane(h.view())
	if !strings.Contains(before, "Untitled-2") {
		t.Fatalf("unexpected graph:\n%s", before)
	}
	abs := filepath.Join(h.store.Root(), "Other.md")
	if err := os.WriteFile(abs, []byte("[[Untitled-2]] edited"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(abs, later, later); err != nil {
		t.Fatal(err)
	}
	h.send(app.CheckWorkspaceMsg{})
	if after := editorPane(h.view()); after != before {
		t.Fatalf("an outside edit changed the graph:\nbefore\n%s\nafter\n%s", before, after)
	}
}
