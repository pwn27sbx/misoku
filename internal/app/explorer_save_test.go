package app_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// The user's flow, from the tree: o, a name, Esc, q and the confirm creates
// the note on disk.
func TestTreeONameLeaveConfirmCreatesTheNote(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("o")
	h.keys("prueba")
	h.key(tea.KeyEsc)
	h.keys("q")
	h.key(tea.KeyEnter)
	if _, err := os.Stat(filepath.Join(store.Root(), "prueba.md")); err != nil {
		t.Fatalf("prueba.md must exist after the confirm: %v\n%s", err, h.view())
	}
}

// Enter while typing a new name creates it on disk at once, with no
// confirm, and opens an empty line below in insert, as Vim does: typing a
// second name and Enter creates that one too. The editor stays open.
func TestEnterWhileTypingCreatesTheNameAndOpensALine(t *testing.T) {
	for _, c := range []struct{ name, want string }{
		{"idea", "idea.md"},
		{"carpeta/", "carpeta"},
		{"a/b", "a/b.md"},
	} {
		h, store := newEntryHarness(t)
		h.keys("o")
		h.keys(c.name)
		h.key(tea.KeyEnter)
		if v := h.view(); strings.Contains(v, "CONFIRM FILE SYSTEM ACTIONS") {
			t.Fatalf("%s: creating asks no confirm:\n%s", c.name, v)
		}
		if !exists(store, c.want) {
			t.Fatalf("%s: %s must exist after Enter:\n%s", c.name, c.want, h.view())
		}
		if v := h.view(); !strings.Contains(v, "EDIT /") {
			t.Fatalf("%s: the folder editor stays open:\n%s", c.name, v)
		}
		h.keys("otra")
		h.key(tea.KeyEnter)
		if !exists(store, "otra.md") {
			t.Fatalf("%s: Enter opens a new line in insert: otra.md must exist:\n%s", c.name, h.view())
		}
		if tabs := app.OpenTabs(h.m); len(tabs) != 0 {
			t.Fatalf("%s: Enter while typing opened %q", c.name, tabs)
		}
	}
}

// Enter while typing on an existing line opens nothing: it opens a new
// line below in insert. A renamed line stays a pending edit.
func TestEnterWhileTypingAnExistingLineOpensALine(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("G") // zeta, at the root
	h.keys("i")
	h.key(tea.KeyEnter)
	if tabs := app.OpenTabs(h.m); len(tabs) != 0 {
		t.Fatalf("Enter while typing opened %q", tabs)
	}
	if v := h.view(); !strings.Contains(v, "EDIT /") || strings.Contains(v, "CONFIRM") {
		t.Fatalf("the folder editor stays open, with no confirm:\n%s", v)
	}
	h.keys("nueva")
	h.key(tea.KeyEnter)
	if !exists(store, "nueva.md") {
		t.Fatalf("Enter must have opened a new line: nueva.md must exist:\n%s", h.view())
	}

	h, store = newEntryHarness(t)
	h.keys("G")
	h.keys("ix")
	h.key(tea.KeyEnter)
	if v := h.view(); strings.Contains(v, "CONFIRM") || !exists(store, "zeta.md") {
		t.Fatalf("a rename stays pending on Enter:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.keys("=")
	if v := h.view(); !strings.Contains(v, "CONFIRM FILE SYSTEM ACTIONS") {
		t.Fatalf("= must confirm the pending rename:\n%s", v)
	}
	h.keys("y")
	if !exists(store, "xzeta.md") {
		t.Fatalf("the rename applies after =:\n%s", h.view())
	}
}

// Enter on an empty new line ends typing like Esc: nothing is created.
func TestEnterOnAnEmptyNewLineEndsTyping(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("o")
	h.key(tea.KeyEnter)
	h.keys("j")
	if tabs := app.OpenTabs(h.m); len(tabs) != 0 {
		t.Fatalf("Enter on an empty line opened %q", tabs)
	}
	if exists(store, "j.md") {
		t.Fatal("after Enter on an empty line, keys are commands again")
	}
}

// Enter on a folder line of the folder editor enters that folder; it opens
// no note.
func TestEditorEnterOnAFolderEntersIt(t *testing.T) {
	h, _ := newEntryHarness(t)
	h.keys("G") // zeta, at the root
	h.keys("m")
	if v := h.view(); !strings.Contains(v, "EDIT /") {
		t.Fatalf("setup: m on zeta must edit the root:\n%s", v)
	}
	h.keys("gg")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "EDIT FLM/") {
		t.Fatalf("Enter on FLM/ must enter it:\n%s", v)
	}
	if tabs := app.OpenTabs(h.m); len(tabs) != 0 {
		t.Fatalf("Enter on a folder opened %q", tabs)
	}
}

// Enter on a note line of the folder editor opens that note in the editor
// and, with nothing pending, closes the folder editor.
func TestEditorEnterOnANoteOpensIt(t *testing.T) {
	h, _ := newEntryHarness(t)
	h.keys("G") // zeta, at the root
	h.keys("m")
	h.keys("G")
	h.key(tea.KeyEnter)
	if tabs := app.OpenTabs(h.m); !slices.Equal(tabs, []string{"zeta.md"}) {
		t.Fatalf("tabs = %q, want zeta.md", tabs)
	}
	if v := h.view(); strings.Contains(v, "EDIT /") {
		t.Errorf("opening a note with nothing pending closes the folder editor:\n%s", v)
	}
}

// Enter on a folder in the tree expands it and opens no note.
func TestTreeEnterOnAFolderOpensNoNote(t *testing.T) {
	h, _ := newEntryHarness(t) // FLM/ selected
	h.key(tea.KeyEnter)
	if tabs := app.OpenTabs(h.m); len(tabs) != 0 {
		t.Fatalf("Enter on a folder opened %q", tabs)
	}
	if v := app.ExplorerView(h.m); !strings.Contains(v, "x.md") {
		t.Fatalf("Enter on FLM must expand it:\n%s", v)
	}
}
