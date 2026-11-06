package app_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// inOrder fails unless every want appears in v, in that order.
func inOrder(t *testing.T, v string, want ...string) {
	t.Helper()
	at := 0
	for _, w := range want {
		i := strings.Index(v[at:], w)
		if i < 0 {
			t.Fatalf("want %q after offset %d, in order %q:\n%s", w, at, want, v)
		}
		at += i + len(w)
	}
}

func isDir(store *core.Store, rel string) bool {
	fi, err := os.Stat(filepath.Join(store.Root(), filepath.FromSlash(rel)))
	return err == nil && fi.IsDir()
}

// S1: Enter while typing only opens a new line: nothing is created, opened
// or confirmed, and the folder editor stays in the buffer.
func TestOilInsertEnterOnlyAddsLines(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("a") // on FLM/, typing at the end
	h.key(tea.KeyEnter)
	h.keys("FolderPRueba/")
	h.key(tea.KeyEnter)
	h.keys("nota")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	v := h.view()
	if strings.Contains(v, "CONFIRM") || !strings.Contains(v, "EDIT /") {
		t.Fatalf("Enter while typing asks nothing and leaves the editor open:\n%s", v)
	}
	if exists(store, "FolderPRueba") || exists(store, "nota.md") {
		t.Fatal("Enter while typing must not touch the disk")
	}
	if tabs := app.OpenTabs(h.m); len(tabs) != 0 {
		t.Fatalf("Enter while typing opened %q", tabs)
	}
	inOrder(t, app.ExplorerView(h.m), "FLM/", "FolderPRueba/", "nota", "zeta")
}

// S2, S4: the user's Oil flow. Enter in normal mode with pending lines
// confirms; No keeps the lines; Yes creates, the editor stays on the same
// folder, re-read and sorted, with the cursor on the new entry. A file
// typed on a line below follows the same way.
func TestOilNormalEnterConfirmsAppliesAndStays(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("a")
	h.key(tea.KeyEnter)
	h.keys("FolderPRueba/")
	h.key(tea.KeyEsc)
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "CONFIRM FILE SYSTEM ACTIONS") || !strings.Contains(v, "mkdir FolderPRueba/") {
		t.Fatalf("Enter with pending lines must confirm them:\n%s", v)
	}
	h.keys("n")
	if v := h.view(); strings.Contains(v, "CONFIRM") || !strings.Contains(v, "FolderPRueba/") || exists(store, "FolderPRueba") {
		t.Fatalf("No returns to the buffer with the line kept and nothing written:\n%s", v)
	}
	h.key(tea.KeyEnter)
	h.keys("y")
	if !isDir(store, "FolderPRueba") {
		t.Fatalf("Yes must create FolderPRueba/:\n%s", h.view())
	}
	if v := h.view(); !strings.Contains(v, "EDIT /") || strings.Contains(v, "CONFIRM") {
		t.Fatalf("after applying the editor stays open on the same folder:\n%s", v)
	}
	inOrder(t, app.ExplorerView(h.m), "FLM/", "FolderPRueba/", "zeta")
	h.key(tea.KeyEnter) // nothing pending: the cursor's line is entered
	if v := h.view(); !strings.Contains(v, "EDIT FolderPRueba/") {
		t.Fatalf("the cursor must be on FolderPRueba/ after applying:\n%s", v)
	}
	h.keys("h")

	h.keys("a")
	h.key(tea.KeyEnter)
	h.keys("archivo de prueba.txt")
	h.key(tea.KeyEsc)
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "create archivo de prueba.txt.md") {
		t.Fatalf("Enter must confirm the new file:\n%s", v)
	}
	h.keys("y")
	if !exists(store, "archivo de prueba.txt.md") {
		t.Fatalf("Yes must create the file:\n%s", h.view())
	}
	inOrder(t, app.ExplorerView(h.m), "FLM/", "FolderPRueba/", "archivo de prueba.txt", "zeta")
	h.key(tea.KeyEnter)
	if tabs := app.OpenTabs(h.m); !slices.Equal(tabs, []string{"archivo de prueba.txt.md"}) {
		t.Fatalf("the cursor must be on the new file: tabs = %q", tabs)
	}
}

// S3, S4: = applies after the confirm and the editor stays open.
func TestOilEqualsAppliesAndStays(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("G")
	h.keys("m")
	h.keys("o")
	h.keys("otra")
	h.key(tea.KeyEsc)
	h.keys("=")
	h.keys("y")
	if !exists(store, "otra.md") {
		t.Fatal("= then Yes must create otra.md")
	}
	if v := h.view(); !strings.Contains(v, "EDIT /") {
		t.Fatalf("= keeps the editor open after applying:\n%s", v)
	}
}

// S6: a folder made in the editor with nothing inside is listed in the
// editor and in the tree, also by a fresh store (a restart), and typing it
// again is refused instead of planned twice.
func TestOilEmptyFolderStaysListed(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("G")
	h.keys("m")
	h.keys("o")
	h.keys("Testfolder/")
	h.key(tea.KeyEsc)
	h.keys("=")
	h.keys("y")
	if !isDir(store, "Testfolder") {
		t.Fatal("Testfolder/ must be created")
	}
	if v := app.ExplorerView(h.m); !strings.Contains(v, "Testfolder/") {
		t.Fatalf("the editor lists the empty folder:\n%s", v)
	}
	h.key(tea.KeyEsc)
	if v := app.ExplorerView(h.m); !strings.Contains(v, "Testfolder") {
		t.Fatalf("the tree lists the empty folder:\n%s", v)
	}

	fresh, err := core.NewStore(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	h2 := newHarness(t, fresh)
	h2.keys("e")
	if v := app.ExplorerView(h2.m); !strings.Contains(v, "Testfolder") {
		t.Fatalf("after a restart the empty folder is still listed:\n%s", v)
	}

	h.keys("m") // on Testfolder/, selected in the tree
	h.keys("h")
	h.keys("o")
	h.keys("Testfolder/")
	h.key(tea.KeyEsc)
	h.keys("=")
	if v := h.view(); strings.Contains(v, "CONFIRM") || !strings.Contains(v, "already exists") {
		t.Fatalf("retyping an existing empty folder is refused:\n%s", v)
	}
}

// S8, S9: erasing a line's whole name with Backspace joins it into the
// line above, which plans its delete; Ctrl+S in insert confirms, Yes moves
// it to the Trash, the editor stays open and goes on typing.
func TestOilBackspaceErasesALineAndCtrlSApplies(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("G") // zeta
	h.keys("a")
	for range len("zeta") + 1 {
		h.key(tea.KeyBackspace)
	}
	if v := app.ExplorerView(h.m); strings.Contains(v, "zeta") {
		t.Fatalf("Backspace at the start of the line removes it:\n%s", v)
	}
	if !exists(store, "zeta.md") {
		t.Fatal("erasing a line touches nothing on disk")
	}
	h.key(tea.KeyCtrlS)
	if v := h.view(); !strings.Contains(v, "CONFIRM FILE SYSTEM ACTIONS") || !strings.Contains(v, "delete zeta.md") {
		t.Fatalf("Ctrl+S must confirm the delete:\n%s", v)
	}
	h.keys("n")
	if exists(store, "zeta.md") == false || strings.Contains(h.view(), "CONFIRM") {
		t.Fatal("No writes nothing and returns to the buffer")
	}
	h.key(tea.KeyCtrlS)
	h.keys("y")
	if exists(store, "zeta.md") {
		t.Fatalf("Yes must delete zeta.md:\n%s", h.view())
	}
	if v := h.view(); !strings.Contains(v, "EDIT /") {
		t.Fatalf("the editor stays open after Ctrl+S:\n%s", v)
	}
	h.keys("x") // still typing: x goes into the name on the line
	if v := app.ExplorerView(h.m); !strings.Contains(v, "FLM/x") {
		t.Fatalf("Ctrl+S keeps insert mode, typing at the end of the line:\n%s", v)
	}
}

// S9: Ctrl+S in normal mode confirms like =.
func TestOilCtrlSInNormalConfirms(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("G")
	h.keys("m")
	h.keys("G") // zeta, in the editor of the root
	h.keys("dd")
	h.key(tea.KeyCtrlS)
	if v := h.view(); !strings.Contains(v, "delete zeta.md") {
		t.Fatalf("Ctrl+S must confirm:\n%s", v)
	}
	h.keys("y")
	if exists(store, "zeta.md") {
		t.Fatal("Yes must delete zeta.md")
	}
	if v := h.view(); !strings.Contains(v, "EDIT /") {
		t.Fatalf("the editor stays open:\n%s", v)
	}
}
