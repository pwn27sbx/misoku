package app_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// folderHash hashes every file and folder under dir, .git included, and
// the file cfg, so a test can show nothing changed. .git/FETCH_HEAD is
// git's note of the last fetch, left out.
func folderHash(t *testing.T, dir, cfg string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if filepath.ToSlash(rel) == ".git/FETCH_HEAD" {
			return nil
		}
		h.Write([]byte(filepath.ToSlash(rel) + "\n"))
		if !d.IsDir() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			h.Write(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cfg)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// Typing ":sync setup" in the editor asks the app for the assistant.
func TestTypingSyncSetupAsksForTheAssistant(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys(":sync setup")
	queue := []tea.Cmd{h.update(tea.KeyMsg{Type: tea.KeyEnter})}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := runCmd(c).(type) {
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case core.SyncSetupMsg:
			return
		}
	}
	t.Fatal(":sync setup must ask for the setup assistant")
}

// A vault that already syncs: the assistant says so and changes nothing,
// with no gh needed.
func TestSyncSetupInsideTheAppOnASyncedVaultChangesNothing(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	cfg := syncConfig(t, vault)
	before := folderHash(t, vault, cfg)
	h := window(t, vault, cfg)
	h.send(core.SyncSetupMsg{})
	text := noSpace(dialogText(h.view()))
	if !strings.Contains(text, noSpace("already set up")) || !strings.Contains(text, noSpace("nothing changed")) {
		t.Fatalf("the assistant must say the vault is already set up:\n%s", h.view())
	}
	h.key(tea.KeyEnter) // Close
	if after := folderHash(t, vault, cfg); after != before {
		t.Error("setup on a synced vault changed the vault or the config")
	}
}

// Cancelling the first dialog changes nothing and says so.
func TestSyncSetupInsideTheAppCancelledChangesNothing(t *testing.T) {
	t.Parallel()
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	cfg := filepath.Join(t.TempDir(), "config.json")
	before := folderHash(t, vault, cfg)
	h := window(t, vault, cfg)
	h.send(core.SyncSetupMsg{})
	if !strings.Contains(h.view(), "Sync this vault") {
		t.Fatalf("the assistant must open with what will happen:\n%s", h.view())
	}
	h.key(tea.KeyEsc)
	wantBar(t, h, "nothing was changed")
	if after := folderHash(t, vault, cfg); after != before {
		t.Error("cancelling the setup changed the vault or the config")
	}
}

// Quitting while a setup step runs would leave it half done: the quit
// waits, says why, and works once the step has ended.
func TestQuittingWaitsForARunningSyncSetupStep(t *testing.T) {
	t.Parallel()
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	h := window(t, vault, filepath.Join(t.TempDir(), "config.json"))
	step := h.update(core.SyncSetupMsg{}) // running, not finished
	for _, quit := range []tea.Msg{core.VimQuitMsg{}, core.VimQuitMsg{Force: true}, tea.KeyMsg{Type: tea.KeyCtrlC}} {
		if cmd := h.update(quit); cmd != nil {
			if _, ok := runCmd(cmd).(tea.QuitMsg); ok {
				t.Fatalf("%#v quit while setup was running", quit)
			}
		}
		wantBar(t, h, "Sync setup is running")
	}
	h.finish(step)
	h.key(tea.KeyEsc) // cancel the setup
	cmd := h.update(core.VimQuitMsg{Force: true})
	if cmd == nil {
		t.Fatal("once setup ended, :q! must quit")
	}
	if _, ok := runCmd(cmd).(tea.QuitMsg); !ok {
		t.Fatal("once setup ended, :q! must quit")
	}
}

// Inside another repository the assistant asks before creating one of
// its own; Enter creates it and goes on, and cancelling the next dialog
// removes it again.
func TestSyncSetupInsideTheAppOffersARepositoryOfItsOwn(t *testing.T) {
	t.Parallel()
	outer := t.TempDir()
	if out, err := exec.Command("git", "-C", outer, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init: %v %s", err, out)
	}
	vault := filepath.Join(outer, "notes")
	writeOutside(t, vault, "a.md", "hello")
	cfg := filepath.Join(t.TempDir(), "config.json")
	h := window(t, vault, cfg)
	h.send(core.SyncSetupMsg{})
	if text := noSpace(dialogText(h.view())); !strings.Contains(text, noSpace("This folder is inside another git repository. Create a separate one for your notes?")) {
		t.Fatalf("the assistant must offer a repository of its own:\n%s", h.view())
	}
	if _, err := os.Lstat(filepath.Join(vault, ".git")); !os.IsNotExist(err) {
		t.Fatalf("asking must not create anything: %v", err)
	}
	h.key(tea.KeyEnter) // Create
	if !strings.Contains(h.view(), "Sync this vault") {
		t.Fatalf("after creating it the assistant must go on:\n%s", h.view())
	}
	if _, err := os.Stat(filepath.Join(vault, ".git")); err != nil {
		t.Fatalf("Enter must create the repository: %v", err)
	}
	h.key(tea.KeyEsc)
	wantBar(t, h, "was removed")
	if _, err := os.Lstat(filepath.Join(vault, ".git")); !os.IsNotExist(err) {
		t.Errorf("cancelling must remove the repository this run created: %v", err)
	}
}

// The home screen offers "Sync between computers" while the folder is not
// connected (not listed in sync.vaults), and s opens the assistant.
func TestHomeOffersSyncBetweenComputersWhileTheFolderIsNotConnected(t *testing.T) {
	t.Parallel()
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	h := window(t, vault, filepath.Join(t.TempDir(), "config.json"))
	if !strings.Contains(h.view(), "Sync between computers") {
		t.Fatalf("a folder that does not sync must be offered sync:\n%s", h.view())
	}
	h.keys("s")
	if !strings.Contains(h.view(), "Sync this vault") {
		t.Fatalf("s must open the setup assistant:\n%s", h.view())
	}
}

// A connected folder (listed in sync.vaults) is not offered sync, and s
// does nothing there.
func TestHomeHidesSyncBetweenComputersOnAConnectedFolder(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	if strings.Contains(h.view(), "Sync between computers") {
		t.Fatalf("a folder that syncs must not be offered sync:\n%s", h.view())
	}
	if cmd := h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}); cmd != nil {
		t.Error("s must do nothing on the home screen of a folder that syncs")
	}
}

// With no config file to list the folder in, the assistant cannot run:
// the entry is not offered.
func TestHomeHidesSyncBetweenComputersWithoutAConfigFile(t *testing.T) {
	t.Parallel()
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	h := window(t, vault, "")
	if strings.Contains(h.view(), "Sync between computers") {
		t.Fatalf("without a config file sync must not be offered:\n%s", h.view())
	}
}

// Enter on the selected "Sync between computers" opens the assistant too.
func TestHomeSyncBetweenComputersOpensTheAssistantOnEnter(t *testing.T) {
	t.Parallel()
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	h := window(t, vault, filepath.Join(t.TempDir(), "config.json"))
	for i := 0; i < 4; i++ { // New Note, Find Note, Open Explorer, Themes
		h.key(tea.KeyDown)
	}
	h.key(tea.KeyEnter)
	if !strings.Contains(h.view(), "Sync this vault") {
		t.Fatalf("Enter on the sync entry must open the setup assistant:\n%s", h.view())
	}
}

// MISOKU_SYNC=0 turns sync off: the home screen does not offer it.
func TestHomeHidesSyncBetweenComputersWhenSyncIsTurnedOff(t *testing.T) {
	t.Setenv("MISOKU_SYNC", "0")
	vault := t.TempDir()
	writeOutside(t, vault, "a.md", "hello")
	h := window(t, vault, filepath.Join(t.TempDir(), "config.json"))
	if strings.Contains(h.view(), "Sync between computers") {
		t.Fatalf("with sync turned off, sync must not be offered:\n%s", h.view())
	}
}

// The home folder is no place to sync (it would push ~/.ssh too): the
// home screen does not offer "Sync between computers" there, s does
// nothing, and :sync setup refuses, changing nothing.
func TestTheHomeFolderIsNotOfferedSyncAndSetupRefusesIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
	writeOutside(t, home, "a.md", "hello")
	cfg := filepath.Join(t.TempDir(), "config.json")
	before := folderHash(t, home, cfg)
	h := window(t, home, cfg)
	if strings.Contains(h.view(), "Sync between computers") {
		t.Fatalf("the home folder must not be offered sync:\n%s", h.view())
	}
	if cmd := h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}); cmd != nil {
		t.Error("s must do nothing on the home screen of the home folder")
	}
	h.send(core.SyncSetupMsg{})
	text := noSpace(dialogText(h.view()))
	if !strings.Contains(text, noSpace("is your home folder, the root of a disk or a system folder: sync a folder of its own for your notes")) {
		t.Fatalf(":sync setup must refuse the home folder:\n%s", h.view())
	}
	h.key(tea.KeyEnter) // Close
	if after := folderHash(t, home, cfg); after != before {
		t.Error("setup on the home folder changed it or the config")
	}
}

// A home folder that v0.7.0 already listed in sync.vaults keeps syncing:
// only setting up a new one is refused.
func TestAListedHomeFolderKeepsSyncing(t *testing.T) {
	remote, vault := syncedVault(t)
	t.Setenv("HOME", vault)
	t.Setenv("USERPROFILE", vault) // Windows
	writeOutside(t, vault, "new.md", "written before opening")
	h := window(t, vault, syncConfig(t, vault))
	if !initAsksForSync(h) {
		t.Fatal("opening a listed home folder must still start a sync")
	}
	h.finish(h.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the sync on opening must still push the new note")
	}
}
