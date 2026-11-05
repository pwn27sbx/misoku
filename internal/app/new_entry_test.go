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

// newEntryHarness is a workspace with a folder FLM (holding one note) and a
// root note zeta, with the explorer focused on FLM.
func newEntryHarness(t *testing.T) (*harness, *core.Store) {
	t.Helper()
	store := newTestStore(t)
	for _, n := range [][3]string{{"FLM", "x", "x body"}, {"", "zeta", "zeta body"}} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e")
	return h, store
}

// typeNew presses n in the explorer, types name and presses Enter.
func (h *harness) typeNew(name string) {
	h.t.Helper()
	h.keys("n")
	h.keys(name)
	h.key(tea.KeyEnter)
}

func exists(store *core.Store, rel string) bool {
	_, err := os.Stat(filepath.Join(store.Root(), filepath.FromSlash(rel)))
	return err == nil
}

func TestNewEntryFolderAsksThenCreatesAndSelectsIt(t *testing.T) {
	h, store := newEntryHarness(t)
	h.typeNew("test/")
	if v := h.view(); !strings.Contains(v, "CREATE FLM/test/") || !strings.Contains(v, "Yes") || !strings.Contains(v, "No") {
		t.Fatalf("expected the CREATE confirmation:\n%s", v)
	}
	if exists(store, "FLM/test") {
		t.Fatal("nothing may be created before confirming")
	}
	h.keys("y")
	if info, err := os.Stat(filepath.Join(store.Root(), "FLM", "test")); err != nil || !info.IsDir() {
		t.Fatalf("folder FLM/test not created: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(store.Root(), "FLM", "test")); len(entries) != 0 {
		t.Fatalf("a folder entry creates no note, got %v", entries)
	}
	if got := app.ExplorerSelected(h.m); got != "FLM/test" {
		t.Fatalf("the new folder should be selected, got %q", got)
	}
	if !strings.Contains(app.ExplorerView(h.m), "test") {
		t.Fatalf("the new (empty) folder should show in the explorer:\n%s", app.ExplorerView(h.m))
	}
	// The workspace poll must not hide it again.
	h.send(app.CheckWorkspaceMsg{})
	if got := app.ExplorerSelected(h.m); got != "FLM/test" {
		t.Fatalf("the folder should survive a workspace check, got %q", got)
	}
}

func TestNewEntryNestedNoteCreatesFoldersAndOpensIt(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("j") // zeta.md: the context is the root
	h.typeNew("ideas/test")
	if v := h.view(); !strings.Contains(v, "CREATE ideas/test.md") {
		t.Fatalf("expected the CREATE confirmation:\n%s", v)
	}
	h.key(tea.KeyEnter) // Yes is selected
	if got, err := store.Get("ideas/test.md"); err != nil || got.Content != "" {
		t.Fatalf("expected an empty ideas/test.md, got %+v %v", got, err)
	}
	v := h.view()
	if !strings.Contains(v, "test.md") || !strings.Contains(v, "INSERT") {
		t.Fatalf("the new note should open in the editor, in INSERT:\n%s", v)
	}
	if tree := app.ExplorerView(h.m); !strings.Contains(tree, "ideas") || !strings.Contains(tree, "test.md") {
		t.Fatalf("ideas should be expanded with test.md inside:\n%s", tree)
	}
}

func TestNewEntryDeepPaths(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("j") // zeta.md: the context is the root
	h.typeNew("a/b/")
	h.keys("y")
	if !exists(store, "a/b") || exists(store, "a/b.md") {
		t.Fatalf("expected the folders a/b/, got %v", store.List())
	}
	// a/b is now selected, so the next path is typed inside it.
	h.typeNew("c/d/e")
	if v := h.view(); !strings.Contains(v, "CREATE a/b/c/d/e.md") {
		t.Fatalf("the confirmation shows the path from the workspace root:\n%s", v)
	}
	h.keys("y")
	if !exists(store, "a/b/c/d/e.md") {
		t.Fatalf("expected a/b/c/d/e.md, got %v", store.List())
	}
}

func TestNewEntryCancelCreatesNothing(t *testing.T) {
	h, store := newEntryHarness(t)
	before := len(store.List())

	h.typeNew("gone/")
	h.keys("n")
	h.typeNew("gone2")
	h.key(tea.KeyEsc)
	h.keys("n")
	h.keys("gone3")
	h.key(tea.KeyEsc) // closes the name line: no dialog
	if v := h.view(); strings.Contains(v, "CREATE") {
		t.Fatalf("Esc on the name line must not ask:\n%s", v)
	}
	for _, rel := range []string{"FLM/gone", "FLM/gone2.md", "FLM/gone3.md"} {
		if exists(store, rel) {
			t.Fatalf("%s created after cancelling", rel)
		}
	}
	if len(store.List()) != before {
		t.Fatalf("nothing may change, got %v", store.List())
	}
}

func TestNewEntryRejectsPathsOutsideTheFolder(t *testing.T) {
	for _, name := range []string{"../escape", "../../escape/", "/escape", "a//b", "./x"} {
		t.Run(name, func(t *testing.T) {
			h, store := newEntryHarness(t)
			h.typeNew(name)
			v := h.view()
			if strings.Contains(v, "CREATE") {
				t.Fatalf("%q must not reach the confirmation:\n%s", name, v)
			}
			if !strings.Contains(v, "Can't create") {
				t.Fatalf("%q should report an error in the status bar:\n%s", name, v)
			}
			parent := filepath.Dir(store.Root())
			for _, rel := range []string{"escape", "escape.md", "a", "x.md", "FLM/a"} {
				if exists(store, rel) || fileThere(filepath.Join(parent, rel)) {
					t.Fatalf("%q created %s", name, rel)
				}
			}
		})
	}
}

func fileThere(abs string) bool { _, err := os.Stat(abs); return err == nil }

func TestNewEntryExistingNoteOpensItWithoutOverwriting(t *testing.T) {
	h, store := newEntryHarness(t)
	h.typeNew("x")
	v := h.view()
	if strings.Contains(v, "CREATE") {
		t.Fatalf("an existing note is opened, not created:\n%s", v)
	}
	if !strings.Contains(v, "already exists") || !strings.Contains(v, "x body") {
		t.Fatalf("expected FLM/x.md opened with a notice:\n%s", v)
	}
	if got, _ := store.Get("FLM/x.md"); got.Content != "x body" {
		t.Fatalf("existing note changed: %q", got.Content)
	}
	if exists(store, "FLM/x 2.md") {
		t.Fatal("no copy may be made")
	}
}

func TestNewEntryExistingFolderIsSelected(t *testing.T) {
	h, _ := newEntryHarness(t)
	h.keys("j")
	h.typeNew("FLM/")
	if v := h.view(); strings.Contains(v, "CREATE") {
		t.Fatalf("an existing folder is selected, not created:\n%s", v)
	}
	if got := app.ExplorerSelected(h.m); got != "FLM" {
		t.Fatalf("FLM should be selected, got %q", got)
	}
}

func TestNewEntryEmptyNameStartsAnUntitledDraft(t *testing.T) {
	h, store := newEntryHarness(t)
	before := len(store.List())
	h.typeNew("")
	v := h.view()
	if strings.Contains(v, "CREATE") {
		t.Fatalf("an empty name needs no confirmation:\n%s", v)
	}
	if !strings.Contains(v, "Untitled-1") || !strings.Contains(v, "INSERT") {
		t.Fatalf("expected the Untitled-1 draft in INSERT, as n did:\n%s", v)
	}
	if len(store.List()) != before {
		t.Fatalf("a draft writes nothing, got %v", store.List())
	}
}

func TestNewSubNoteStillStartsADraftUnderTheNote(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("j") // zeta.md
	h.keys("s")
	v := h.view()
	if !strings.Contains(v, "Untitled-1") || !strings.Contains(v, "INSERT") {
		t.Fatalf("s should still open an Untitled draft:\n%s", v)
	}
	if exists(store, "zeta") || len(store.List()) != 3 {
		t.Fatalf("s writes nothing before saving, got %v", store.List())
	}
}

func TestRenameStillRenamesInPlace(t *testing.T) {
	h, store := newEntryHarness(t)
	h.keys("j") // zeta.md
	h.keys("r")
	for range "zeta" {
		h.key(tea.KeyBackspace)
	}
	h.keys("omega/")
	h.key(tea.KeyEnter)
	if v := h.view(); strings.Contains(v, "CREATE") {
		t.Fatalf("rename must not ask to create:\n%s", v)
	}
	if got, err := store.Get("omega-.md"); err != nil || got.Content != "zeta body" {
		t.Fatalf("rename keeps its old rules (/ becomes -): %+v %v", got, err)
	}
}

// p previews the selected note: it opens in a tab.
func TestPPreviewsTheSelectedNote(t *testing.T) {
	h, _ := newEntryHarness(t)
	h.keys("j") // zeta.md
	h.keys("p")
	if tabs := app.OpenTabs(h.m); !slices.Contains(tabs, "zeta.md") {
		t.Fatalf("p must preview zeta.md, tabs = %q", tabs)
	}
}
