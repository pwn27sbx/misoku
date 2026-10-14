package core_test

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

func TestUpdateReplacesTheFileInOneStep(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "old version, complete"); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(store.Root(), "doc.md")

	// A program that had the note open before the save (a sync client, a
	// backup) must keep reading the whole old version: the file is swapped,
	// never truncated and rewritten in place.
	reader, err := os.Open(abs)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	if _, err := store.Update(core.Page{ID: "doc.md", Content: "new version"}); err != nil {
		t.Fatal(err)
	}
	old, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != "old version, complete" {
		t.Fatalf("the file was rewritten in place; an open reader saw %q", old)
	}
	if got, _ := os.ReadFile(abs); string(got) != "new version" {
		t.Fatalf("file content = %q", got)
	}
	entries, _ := os.ReadDir(store.Root())
	for _, e := range entries {
		if e.Name() != "doc.md" {
			t.Fatalf("leftover file after saving: %s", e.Name())
		}
	}
}

func TestUpdateKeepsTheFilePermissions(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "private", "x"); err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(store.Root(), "private.md")
	if err := os.Chmod(abs, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(core.Page{ID: "private.md", Content: "y"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions changed to %v", info.Mode().Perm())
	}
}

func TestUpdateThroughASymlinkKeepsTheLink(t *testing.T) {
	store := newTestStore(t)
	outside := filepath.Join(t.TempDir(), "real.md")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(store.Root(), "linked.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(core.Page{ID: "linked.md", Content: "through the link"}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the note must stay a symlink, got %v %v", info, err)
	}
	if got, _ := os.ReadFile(outside); string(got) != "through the link" {
		t.Fatalf("the link target should hold the new text, got %q", got)
	}
}

func TestUpdateKeepsWindowsLineEndings(t *testing.T) {
	store := newTestStore(t)
	abs := filepath.Join(store.Root(), "crlf.md")
	if err := os.WriteFile(abs, []byte("a\r\nb\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(core.Page{ID: "crlf.md", Content: "a\nb\nc\n"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(abs); !strings.Contains(string(got), "c\r\n") {
		t.Fatalf("CRLF files keep CRLF, got %q", got)
	}
}
