package gitsync_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"
)

func vaultAt(path string) config.Vault {
	return config.Vault{Path: path, Interval: time.Minute, Device: "test"}
}

func TestOnlyAListedFolderMatches(t *testing.T) {
	t.Setenv("MISOKU_SYNC", "")
	listed, other := t.TempDir(), t.TempDir()
	sub := filepath.Join(listed, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	vaults := []config.Vault{vaultAt(filepath.Join(t.TempDir(), "missing")), vaultAt(listed)}

	v, ok := gitsync.Match(vaults, listed)
	if !ok || v.Path != listed {
		t.Fatalf("the listed folder must match, got %+v %v", v, ok)
	}
	if _, ok := gitsync.Match(vaults, other); ok {
		t.Error("a folder that is not listed must not match")
	}
	if _, ok := gitsync.Match(vaults, sub); ok {
		t.Error("a subfolder of a listed vault is not the vault and must not match")
	}
	if _, ok := gitsync.Match(nil, listed); ok {
		t.Error("with no vaults nothing matches")
	}
}

func TestMatchSeesThroughSymlinks(t *testing.T) {
	t.Setenv("MISOKU_SYNC", "")
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, ok := gitsync.Match([]config.Vault{vaultAt(link)}, real); !ok {
		t.Error("a vault listed through a symlink must match the real folder")
	}
	if _, ok := gitsync.Match([]config.Vault{vaultAt(real)}, link); !ok {
		t.Error("a vault opened through a symlink must match its listed real folder")
	}
}

// On macOS the temporary folder lives under /var, a symlink to /private/var:
// the same folder written both ways must match.
func TestMatchIgnoresHowTheSystemSpellsTheFolder(t *testing.T) {
	t.Setenv("MISOKU_SYNC", "")
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := gitsync.Match([]config.Vault{vaultAt(dir)}, resolved); !ok {
		t.Errorf("%s and %s are the same folder", dir, resolved)
	}
	if _, ok := gitsync.Match([]config.Vault{vaultAt(resolved)}, dir); !ok {
		t.Errorf("%s and %s are the same folder", resolved, dir)
	}
}

// On a case-insensitive file system (the default on macOS and Windows)
// "Notes" and "notes" are one folder and must match.
func TestMatchOnACaseInsensitiveFileSystem(t *testing.T) {
	t.Setenv("MISOKU_SYNC", "")
	dir := filepath.Join(t.TempDir(), "Notes")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	lower := filepath.Join(filepath.Dir(dir), "notes")
	if _, err := os.Stat(lower); err != nil {
		t.Skip("case-sensitive file system")
	}
	if _, ok := gitsync.Match([]config.Vault{vaultAt(lower)}, dir); !ok {
		t.Error("the same folder in another case must match")
	}
}

func TestMisokuSyncZeroTurnsSyncOff(t *testing.T) {
	dir := t.TempDir()
	vaults := []config.Vault{vaultAt(dir)}
	t.Setenv("MISOKU_SYNC", "0")
	if _, ok := gitsync.Match(vaults, dir); ok {
		t.Error("MISOKU_SYNC=0 must turn sync off for every vault")
	}
	if !gitsync.Disabled() {
		t.Error("Disabled must report MISOKU_SYNC=0")
	}
	t.Setenv("MISOKU_SYNC", "1")
	if _, ok := gitsync.Match(vaults, dir); !ok {
		t.Error("only 0 turns sync off")
	}
}
