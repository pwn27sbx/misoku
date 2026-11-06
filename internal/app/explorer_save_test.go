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

// Enter while typing a name saves: it shows the confirm, and Enter or y on
// it applies, creating the note or the folder on disk.
func TestEnterWhileTypingSavesTheName(t *testing.T) {
	for _, c := range []struct{ name, answer, want string }{
		{"prueba", "enter", "prueba.md"},
		{"carpeta/", "y", "carpeta"},
	} {
		h, store := newEntryHarness(t)
		h.keys("o")
		h.keys(c.name)
		h.key(tea.KeyEnter)
		v := h.view()
		if !strings.Contains(v, "CONFIRM FILE SYSTEM ACTIONS") {
			t.Fatalf("%s: Enter while typing must show the confirm:\n%s", c.name, v)
		}
		if c.answer == "enter" {
			h.key(tea.KeyEnter)
		} else {
			h.keys(c.answer)
		}
		if _, err := os.Stat(filepath.Join(store.Root(), c.want)); err != nil {
			t.Fatalf("%s: %s must exist after %s: %v", c.name, c.want, c.answer, err)
		}
		if v := h.view(); strings.Contains(v, "EDIT /") {
			t.Errorf("%s: an applied save closes the folder editor:\n%s", c.name, v)
		}
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
