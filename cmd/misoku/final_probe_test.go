package main

import (
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/config"
)

// S1/S10: a vault listed by hand (with its device) before its folder
// exists, then set up with init, which creates the folder, keeps one
// entry and its device. ADVISORY probe: today init adds a second entry,
// the hand-written one is then skipped with a warning and its device is
// ignored.
func TestFinalInitKeepsAVaultListedBeforeItsFolderExists(t *testing.T) {
	needGit(t) // not parallel: HOME is set for "~"
	remote, first := newRemote(t), t.TempDir()
	writeFile(t, first, "a.md", "a\n")
	if code, out, errOut := initVault(t, first, remote, filepath.Join(t.TempDir(), "c.json")); code != 0 {
		t.Fatalf("first device: exit %d\n%s\n%s", code, out, errOut)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dir := filepath.Join(home, "notes")
	cfgDir := t.TempDir()
	cfg := filepath.Join(cfgDir, "config.json")
	writeFile(t, cfgDir, "config.json", `{"sync":{"vaults":{"~/notes":{"device":"laptop"}}}}`)
	if code, out, errOut := initVault(t, dir, remote, cfg); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errOut)
	}
	if n := len(listedVaults(t, cfg)); n != 1 {
		t.Errorf("sync.vaults has %d entries for one folder: %v", n, listedVaults(t, cfg))
	}
	c, err := config.Load(cfg)
	if err != nil {
		t.Fatal(err)
	}
	vaults, warnings := c.Sync.Parse()
	if len(warnings) != 0 {
		t.Errorf("warnings: %v", warnings)
	}
	if len(vaults) != 1 || vaults[0].Device != "laptop" {
		t.Errorf("the device written by hand is lost: %+v", vaults)
	}
}

// jsonString quotes s as a JSON string.
func jsonString(s string) string {
	b := []byte{'"'}
	for _, r := range s {
		if r == '"' || r == '\\' {
			b = append(b, '\\')
		}
		b = append(b, string(r)...)
	}
	return string(append(b, '"'))
}
