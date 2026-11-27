package core

import (
	"os"
	"path/filepath"
	"testing"
)

// RED test for #8: renaming to a path that refers to the same underlying file
// (case-only rename on case-insensitive filesystems like APFS/NTFS) must not be
// rejected as "already exists", while a distinct existing file still is.
func TestDestOccupiedByOtherAllowsSameFile(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "note.md")
	if err := os.WriteFile(orig, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	sameAlias := filepath.Join(dir, "Note.md")
	if err := os.Link(orig, sameAlias); err != nil {
		t.Skipf("hardlinks unsupported on this filesystem: %v", err)
	}
	distinct := filepath.Join(dir, "other.md")
	if err := os.WriteFile(distinct, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}

	if destOccupiedByOther(orig, sameAlias) {
		t.Fatal("destOccupiedByOther returned true for the same underlying file (os.SameFile)")
	}
	if !destOccupiedByOther(orig, distinct) {
		t.Fatal("destOccupiedByOther returned false for a distinct existing file")
	}
	if destOccupiedByOther(orig, filepath.Join(dir, "missing.md")) {
		t.Fatal("destOccupiedByOther returned true for a non-existent destination")
	}
}
