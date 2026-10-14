package core

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// crossDevice makes every rename fail the way it does between two volumes.
func crossDevice(t *testing.T) {
	t.Helper()
	orig := renameFile
	renameFile = func(from, to string) error {
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: syscall.EXDEV}
	}
	t.Cleanup(func() { renameFile = orig })
}

func TestTrashFromAnotherVolumeCopiesThenRemoves(t *testing.T) {
	src := t.TempDir()
	trash := t.TempDir()
	note := filepath.Join(src, "doc.md")
	folder := filepath.Join(src, "Folder")
	if err := os.WriteFile(note, []byte("note"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(folder, "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "deep", "inside.md"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	crossDevice(t)

	for _, p := range []string{note, folder} {
		if _, err := renameUnique(p, trash, nil); err != nil {
			t.Fatalf("moving %s across volumes: %v", p, err)
		}
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Fatalf("%s should be gone after the copy", p)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(trash, "doc.md")); string(b) != "note" {
		t.Fatalf("note in the Trash = %q", b)
	}
	if info, _ := os.Stat(filepath.Join(trash, "doc.md")); info == nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions should be kept, got %v", info)
	}
	if b, _ := os.ReadFile(filepath.Join(trash, "Folder", "deep", "inside.md")); string(b) != "inside" {
		t.Fatalf("folder contents in the Trash = %q", b)
	}
}

func TestTrashFromAnotherVolumeKeepsTheNoteIfTheCopyFails(t *testing.T) {
	src := t.TempDir()
	trash := t.TempDir()
	note := filepath.Join(src, "doc.md")
	if err := os.WriteFile(note, []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(trash, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(trash, 0o755)
	crossDevice(t)

	if _, err := renameUnique(note, trash, nil); err == nil {
		t.Fatal("a copy that cannot be written must fail")
	}
	if b, _ := os.ReadFile(note); string(b) != "precious" {
		t.Fatalf("the note must stay, got %q", b)
	}
}
