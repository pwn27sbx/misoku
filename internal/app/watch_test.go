package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// writeOutside creates a note the way another program would: straight on
// disk, without going through the store.
func writeOutside(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExplorerPicksUpNotesCreatedOutside(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "Inside", "x"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("e") // explorer
	if v := h.view(); strings.Contains(v, "Outside") {
		t.Fatalf("note does not exist yet:\n%s", v)
	}

	writeOutside(t, store.Root(), "Outside.md", "y")
	writeOutside(t, store.Root(), "Folder/Nested.md", "z")
	h.send(app.CheckWorkspaceMsg{})

	v := h.view()
	for _, want := range []string{"Inside", "Outside", "Folder"} {
		if !strings.Contains(v, want) {
			t.Fatalf("explorer should list %q after the check:\n%s", want, v)
		}
	}
}

func TestExplorerDropsNotesDeletedOrRenamedOutside(t *testing.T) {
	store := newTestStore(t)
	for _, name := range []string{"Keep", "Gone", "Old Name"} {
		if _, err := store.SaveAs("", name, "x"); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e")

	root := store.Root()
	if err := os.Remove(filepath.Join(root, "Gone.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "Old Name.md"), filepath.Join(root, "New Name.md")); err != nil {
		t.Fatal(err)
	}
	h.send(app.CheckWorkspaceMsg{})

	v := h.view()
	if strings.Contains(v, "Gone") || strings.Contains(v, "Old Name") {
		t.Fatalf("deleted and renamed notes should leave the explorer:\n%s", v)
	}
	if !strings.Contains(v, "Keep") || !strings.Contains(v, "New Name") {
		t.Fatalf("explorer should list Keep and New Name:\n%s", v)
	}
}

func TestExplorerKeepsSelectionWhenNotesAppearAbove(t *testing.T) {
	store := newTestStore(t)
	for _, name := range []string{"Beta", "Gamma"} {
		if _, err := store.SaveAs("", name, "body of "+name); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e")
	h.keys("j") // cursor on Gamma

	// Sorts before both: a cursor that kept its row index would land on Beta.
	writeOutside(t, store.Root(), "Alpha.md", "body of Alpha")
	h.send(app.CheckWorkspaceMsg{})
	h.key(tea.KeyEnter)

	if v := h.view(); !strings.Contains(v, "body of Gamma") {
		t.Fatalf("selection should stay on Gamma after the refresh:\n%s", v)
	}
}

func TestCheckWithoutChangesKeepsTheExplorerAsIs(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "Only", "x"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("e")
	before := h.view()
	h.send(app.CheckWorkspaceMsg{})
	if after := h.view(); after != before {
		t.Fatalf("an unchanged workspace must render the same:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestRenameInProgressSurvivesAnOutsideChange(t *testing.T) {
	store := newTestStore(t)
	for _, name := range []string{"Beta", "Gamma"} {
		if _, err := store.SaveAs("", name, "body of "+name); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e")
	h.keys("j") // Gamma
	h.keys("r")
	for range "Gamma" {
		h.key(tea.KeyBackspace)
	}
	h.keys("Del")

	writeOutside(t, store.Root(), "Alpha.md", "a")
	h.send(app.CheckWorkspaceMsg{})
	h.keys("ta")
	h.key(tea.KeyEnter)

	if got, err := store.Get("Delta.md"); err != nil || got.Content != "body of Gamma" {
		t.Fatalf("the rename must apply to Gamma, got %+v %v", got, err)
	}
	if got, err := store.Get("Beta.md"); err != nil || got.Content != "body of Beta" {
		t.Fatalf("Beta must be untouched, got %+v %v", got, err)
	}
	h.send(app.CheckWorkspaceMsg{})
	if v := h.view(); !strings.Contains(v, "Alpha") || !strings.Contains(v, "Delta") {
		t.Fatalf("after the rename the explorer should catch up:\n%s", v)
	}
}
