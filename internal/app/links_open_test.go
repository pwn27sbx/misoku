package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// Web and mail addresses and plain files inside the vault open at once;
// anything the system opener may run, or that lives outside the vault, asks
// first.
func TestOpenExternalAsksBeforeRunningOrLeavingTheVault(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "vault")
	store, err := core.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"note.md", "images/pic.png", "run.command", "script.sh"} {
		abs := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outside := filepath.Join(base, "outside.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var opened []string
	m := &Model{store: store, openURL: func(target string) error {
		opened = append(opened, target)
		return nil
	}}

	direct := []string{
		"https://example.com/a",
		"http://example.com/b.png",
		"mailto:someone@example.com",
		filepath.Join(root, "note.md"),
		filepath.Join(root, "images", "pic.png"),
	}
	for _, target := range direct {
		opened, m.confirm = nil, nil
		m.openExternal(target)
		if m.confirm != nil {
			t.Errorf("%q asked before opening", target)
		}
		if len(opened) != 1 || opened[0] != target {
			t.Errorf("%q opened %v", target, opened)
		}
	}

	guarded := []string{
		"ftp://example.com/file",
		"file:///etc/passwd",
		"ssh://host/path",
		filepath.Join(root, "run.command"),
		filepath.Join(root, "script.sh"),
		outside,
	}
	if runtime.GOOS != "windows" {
		// An executable without an extension: `open` would run it on macOS.
		exe := filepath.Join(root, "tool")
		if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
		guarded = append(guarded, exe)
	}
	for _, target := range guarded {
		opened, m.confirm = nil, nil
		m.openExternal(target)
		if m.confirm == nil {
			t.Errorf("%q opened without asking", target)
			continue
		}
		if len(opened) != 0 {
			t.Errorf("%q reached the opener before the answer", target)
		}
		if !m.confirm.deliberate || m.confirm.selected != 1 {
			t.Errorf("%q: the dialog must default to Cancel and ignore letters", target)
		}
	}

	// Open runs it once; Cancel opens nothing.
	m.confirm, opened = nil, nil
	m.openExternal("ftp://example.com/file")
	if m.confirm == nil {
		t.Fatal("no dialog before opening")
	}
	if cmd := m.confirm.onChoose(m, 0); cmd == nil {
		t.Fatal("choosing Open returned no command")
	}
	if len(opened) != 1 || opened[0] != "ftp://example.com/file" {
		t.Fatalf("choosing Open opened %v", opened)
	}
	m.confirm, opened = nil, nil
	m.openExternal("ftp://example.com/file")
	if m.confirm == nil {
		t.Fatal("no dialog before opening")
	}
	if cmd := m.confirm.onChoose(m, 1); cmd != nil {
		t.Error("Cancel returned a command")
	}
	if len(opened) != 0 {
		t.Errorf("Cancel opened %v", opened)
	}
}
