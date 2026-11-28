package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// PR #54 regression check: on Unix, a note with a literal backslash in its name
// must stay openable, while traversal IDs remain refused.
func TestStoreOpensLiteralBackslashNameOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("backslash is a path separator on Windows")
	}
	dir := t.TempDir()
	name := `a\b.md`
	if err := os.WriteFile(filepath.Join(dir, name), []byte("# Backslash\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Get(name)
	if err != nil {
		t.Fatalf("Get(%q) failed: %v", name, err)
	}
	if p.Content != "# Backslash\nbody" {
		t.Fatalf("Content = %q", p.Content)
	}
	for _, bad := range []string{"../escape.md", "sub/../../escape.md", "/etc/passwd"} {
		if _, err := s.Abs(bad); err == nil {
			t.Fatalf("Abs(%q) should fail containment", bad)
		}
	}
}
