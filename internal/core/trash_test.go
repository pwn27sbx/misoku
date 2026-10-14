package core_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// trashFiles is where the Trash keeps deleted items on this OS, under the
// fake HOME set by newTestStore.
func trashFiles(t *testing.T) string {
	t.Helper()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), ".Trash")
	case "linux":
		return filepath.Join(os.Getenv("XDG_DATA_HOME"), "Trash", "files")
	}
	t.Skip("no Trash on " + runtime.GOOS + ": deleting is permanent there")
	return ""
}

func inTrash(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(trashFiles(t), filepath.FromSlash(name)))
	if err != nil {
		t.Fatalf("%s should be in the Trash: %v", name, err)
	}
	return string(b)
}

func TestDeleteMovesTheNoteToTheTrash(t *testing.T) {
	store := newTestStore(t)
	trashFiles(t)
	if _, err := store.SaveAs("", "doc", "keep me"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("doc.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "doc.md")); !os.IsNotExist(err) {
		t.Fatal("the note should be gone from the workspace")
	}
	if got := inTrash(t, "doc.md"); got != "keep me" {
		t.Fatalf("trashed note content = %q", got)
	}
	if runtime.GOOS == "linux" {
		info, err := os.ReadFile(filepath.Join(os.Getenv("XDG_DATA_HOME"), "Trash", "info", "doc.md.trashinfo"))
		if err != nil || !strings.Contains(string(info), "Path="+filepath.Join(store.Root(), "doc.md")) || !strings.Contains(string(info), "DeletionDate=") {
			t.Fatalf("missing or wrong .trashinfo: %q %v", info, err)
		}
	}
}

func TestDeletingTheSameNameTwiceKeepsBoth(t *testing.T) {
	store := newTestStore(t)
	dir := trashFiles(t)
	for _, body := range []string{"first", "second"} {
		if _, err := store.SaveAs("", "doc", body); err != nil {
			t.Fatal(err)
		}
		if err := store.Delete("doc.md"); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, e := range entries {
		b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		bodies = append(bodies, string(b))
	}
	if len(bodies) != 2 || !strings.Contains(strings.Join(bodies, "|"), "first") || !strings.Contains(strings.Join(bodies, "|"), "second") {
		t.Fatalf("both deleted versions must be kept, got %v", bodies)
	}
}

func TestDeleteMovesSubNotesAndFoldersToTheTrash(t *testing.T) {
	store := newTestStore(t)
	trashFiles(t)
	if _, err := store.SaveAs("", "parent", "p"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("parent", "child", "c"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("Folder", "inside", "i"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("parent.md"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("Folder"); err != nil {
		t.Fatal(err)
	}
	if got := inTrash(t, "parent/child.md"); got != "c" {
		t.Fatalf("sub-note should be in the Trash, got %q", got)
	}
	if got := inTrash(t, "Folder/inside.md"); got != "i" {
		t.Fatalf("folder should be in the Trash, got %q", got)
	}
	if left := store.List(); len(left) != 0 {
		t.Fatalf("workspace should be empty, got %v", left)
	}
}

func TestDeleteKeepsTheNoteWhenTheTrashFails(t *testing.T) {
	store := newTestStore(t)
	dir := trashFiles(t)
	if _, err := store.SaveAs("", "doc", "precious"); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)

	if err := store.Delete("doc.md"); err == nil {
		t.Fatal("a Trash that cannot be written must make Delete fail")
	}
	if got, err := store.Get("doc.md"); err != nil || got.Content != "precious" {
		t.Fatalf("the note must not be lost, got %+v %v", got, err)
	}
}

func TestDeleteReportsSubNotesItCouldNotMove(t *testing.T) {
	store := newTestStore(t)
	trashFiles(t)
	if _, err := store.SaveAs("", "parent", "p"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("parent", "child", "c"); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(store.Root(), "parent")
	// Moving a folder to another parent rewrites its ".." entry, which needs
	// write permission on the folder itself.
	if err := os.Chmod(sidecar, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(sidecar, 0o755)

	err := store.Delete("parent.md")
	if err == nil || !strings.Contains(err.Error(), "sub-pages") {
		t.Fatalf("a sub-pages folder left behind must be reported, got %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(sidecar, "child.md")); string(got) != "c" {
		t.Fatalf("the sub-note must still be there, got %q", got)
	}
}
