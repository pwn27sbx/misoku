package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// A link or a typed path must never reach outside the vault: the store
// resolves every ID under its root and refuses the ones that climb out.
func TestStoreRefusesIDsThatLeaveTheWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	base := t.TempDir()
	root := filepath.Join(base, "vault")
	outside := filepath.Join(base, "outside.md")
	if err := os.WriteFile(outside, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}

	bad := []string{"../outside.md", "../../outside.md", "sub/../../outside.md", "../", "/etc/passwd", "C:/Windows/win.ini"}

	for _, id := range bad {
		if _, err := store.Get(id); err == nil {
			t.Errorf("Get(%q) read a file outside the vault", id)
		}
		if _, err := store.Abs(id); err == nil {
			t.Errorf("Abs(%q) resolved outside the vault", id)
		}
	}

	page := core.Page{ID: "../outside.md", Title: "outside", Content: "pwned"}
	if _, err := store.Update(page); err == nil {
		t.Error("Update wrote a file outside the vault")
	}
	if got, err := os.ReadFile(outside); err != nil || string(got) != "keep me" {
		t.Errorf("the outside file changed: %q, %v", got, err)
	}

	if err := store.Delete("../outside.md"); err == nil {
		t.Error("Delete removed a file outside the vault")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("the outside file is gone: %v", err)
	}

	if _, err := store.Create("evil", ".."); err == nil {
		t.Error("Create wrote outside the vault")
	}
	if _, err := os.Stat(filepath.Join(base, "evil.md")); err == nil {
		t.Error("Create left a note outside the vault")
	}
}

// The guard must not block ordinary IDs, including a note in a sub-folder.
func TestStoreStillResolvesIDsInsideTheWorkspace(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("ideas", "note", "hello"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"ideas/note.md", "ideas"} {
		if _, err := store.Get(id); err != nil {
			t.Errorf("Get(%q): %v", id, err)
		}
		if _, err := store.Abs(id); err != nil {
			t.Errorf("Abs(%q): %v", id, err)
		}
	}
}
