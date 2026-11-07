package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// A rename whose destination folder already exists must move nothing: the
// note and its sub-pages stay together instead of leaving a.md behind as b.md.
func TestRenameIntoAnExistingSidecarFolderMovesNothing(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "a", "A"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("a", "child", "C"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("b", "keep", "K"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get("a/child.md"); err != nil {
		t.Fatalf("the fixture has no sub-note: %v", err)
	}

	if _, err := store.Update(core.Page{ID: "a.md", Title: "b", Content: "A"}); err == nil {
		t.Fatal("renaming a note onto an existing folder must fail")
	}

	if _, err := store.Get("a.md"); err != nil {
		t.Errorf("the note was moved away: %v", err)
	}
	if _, err := store.Get("a/child.md"); err != nil {
		t.Errorf("the sub-note followed the note: %v", err)
	}
	if _, err := store.Get("b/keep.md"); err != nil {
		t.Errorf("the destination folder changed: %v", err)
	}
	if _, err := store.Get("b.md"); err == nil {
		t.Error("a half-renamed note was left behind")
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "b.md")); err == nil {
		t.Error("b.md exists on disk")
	}
}
