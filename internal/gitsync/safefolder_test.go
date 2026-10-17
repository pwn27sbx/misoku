package gitsync_test

// The home folder, the root of a disk and the system's folders are never
// synced: syncing the home folder would push ~/.ssh too. These tests
// change HOME, so the ones that set it are not parallel.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// systemExample is a system folder of this OS, and one inside it.
func systemExample() (string, string) {
	if runtime.GOOS == "windows" {
		root := os.Getenv("SystemRoot")
		if root == "" {
			root = `C:\Windows`
		}
		return root, filepath.Join(root, "System32")
	}
	return "/etc", "/usr/local/bin"
}

// rootOf is the root of the file system dir is on.
func rootOf(dir string) string {
	return filepath.VolumeName(dir) + string(filepath.Separator)
}

// setHome makes home the user's home folder for this test, on every OS.
func setHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
}

// The home folder, the root of a disk and system folders are no place
// for notes; a folder of their own is.
func TestUnsafeFolders(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	system, inside := systemExample()
	for dir, want := range map[string]bool{
		home:                            true,
		rootOf(home):                    true,
		system:                          true,
		inside:                          true,
		filepath.Join(home, "Notes"):    false,
		filepath.Join(t.TempDir(), "x"): false,
	} {
		if got := gitsync.UnsafeFolder(dir, home); got != want {
			t.Errorf("UnsafeFolder(%s) = %v, want %v", dir, got, want)
		}
	}
}

// What counts as a system folder, on this OS; a removable disk and the
// folders under the home folder do not.
func TestSystemFolders(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	var system, not []string
	if runtime.GOOS == "windows" {
		root, inside := systemExample()
		system = []string{root, inside}
		if pf := os.Getenv("ProgramFiles"); pf != "" {
			system = append(system, filepath.Join(pf, "App"))
		}
		not = []string{filepath.Join(home, "Notes"), `D:\Notes`}
	} else {
		system = []string{"/etc", "/usr", "/usr/local/bin", "/var", "/tmp", "/Volumes", "/home", "/Users", "/bin", "/System/Library"}
		not = []string{filepath.Join(home, "Notes"), "/Volumes/USB/Notes", "/run/media/ana/USB/Notes", "/mnt/disk/Notes", "/home/ana/Notes"}
	}
	for _, d := range system {
		if !gitsync.UnsafeFolder(d, home) {
			t.Errorf("%s: want a system folder", d)
		}
	}
	for _, d := range not {
		if gitsync.UnsafeFolder(d, home) {
			t.Errorf("%s: not a system folder", d)
		}
	}
}

// unsafeText is what setup and init say about such a folder.
const unsafeText = "is your home folder, the root of a disk or a system folder: sync a folder of its own for your notes, such as ~/Notes (misoku sync setup --dir ~/Notes)."

// Setup refuses the home folder, the root of a disk and a system folder
// at once, before gh or git run, and changes nothing.
func TestSetupRefusesTheHomeFolderTheRootAndASystemFolder(t *testing.T) {
	home := t.TempDir()
	write(t, home, ".ssh/id_x", "secret")
	write(t, home, "todo.md", "a note in the home folder\n")
	setHome(t, home)
	system, _ := systemExample()
	for _, dir := range []string{home, rootOf(home), system} {
		cfg := filepath.Join(t.TempDir(), "config.json")
		step := newSetup(t, dir, cfg).Begin(context.Background())
		text := strings.Join(step.Lines, "\n")
		if step.Outcome != gitsync.SetupFailed || !strings.Contains(text, dir+" "+unsafeText) {
			t.Errorf("%s: want the refusal, got %+v", dir, step)
		}
		if _, err := os.Stat(cfg); !os.IsNotExist(err) {
			t.Errorf("%s: the config was written: %v", dir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".git")); !os.IsNotExist(err) {
		t.Errorf("the home folder became a repository: %v", err)
	}
}

// RefuseSyncFolder is the same check for `misoku sync init`: nil for a
// folder of its own.
func TestRefuseSyncFolder(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := gitsync.RefuseSyncFolder(home, home); err == nil || err.Error() != home+" "+unsafeText {
		t.Errorf("home: %v", err)
	}
	if err := gitsync.RefuseSyncFolder(filepath.Join(home, "Notes"), home); err != nil {
		t.Errorf("~/Notes: %v", err)
	}
}

// A home folder that v0.7.0 already set up and listed in sync.vaults is
// left as it is: setup says it is set up, as for any synced vault.
func TestSetupOnAListedHomeFolderSaysItIsSetUp(t *testing.T) {
	isolate(t)
	home := newRepo(t)
	setHome(t, home)
	if out, err := exec.Command("git", "-C", home, "remote", "add", "origin", "https://example.com/ana/notes.git").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(cfg, config.Config{Sync: config.Sync{Vaults: map[string]config.SyncVault{home: {}}}}); err != nil {
		t.Fatal(err)
	}
	step := newSetup(t, home, cfg).Begin(context.Background())
	if step.Outcome != gitsync.SetupAlready {
		t.Fatalf("a listed home folder must stay set up: %+v", step)
	}
}
