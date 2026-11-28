package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// RED test for PR #55: additional macOS and Windows executable/launcher
// extensions must never be opened directly without asking.
func TestOpenDirectlyRefusesAdditionalMacAndWindowsExecExts(t *testing.T) {
	dir := t.TempDir()
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(store)

	exts := []string{
		".terminal", ".workflow", ".pkg", ".scpt",
		".pif", ".cpl", ".msc", ".gadget", ".application",
	}
	for _, ext := range exts {
		t.Run(ext, func(t *testing.T) {
			p := filepath.Join(dir, "sample"+ext)
			if err := os.WriteFile(p, []byte("payload"), 0o644); err != nil {
				t.Fatal(err)
			}
			if m.openDirectly(p) {
				t.Fatalf("openDirectly(%q) = true, want false", p)
			}
		})
	}
}
