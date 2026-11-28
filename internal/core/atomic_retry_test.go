package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// RED test for #13: transient rename errors (common on Windows when Defender or
// a sync client briefly holds a handle) must be retried before falling back.
func TestWriteFileAtomicRetriesTransientRenameError(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "note.md")
	if err := os.WriteFile(dst, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	origRename := atomicRename
	origRetry := retryAtomicRename
	origDelay := atomicRetryDelay
	t.Cleanup(func() {
		atomicRename = origRename
		retryAtomicRename = origRetry
		atomicRetryDelay = origDelay
	})

	attempts := 0
	retryAtomicRename = true
	atomicRetryDelay = 0
	atomicRename = func(oldpath, newpath string) error {
		attempts++
		if attempts <= 2 {
			return errors.New("transient lock")
		}
		return os.Rename(oldpath, newpath)
	}

	if err := WriteFileAtomic(dst, []byte("new")); err != nil {
		t.Fatalf("WriteFileAtomic failed after transient errors: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 rename attempts, got %d", attempts)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("file content = %q, want %q", string(got), "new")
	}
}
