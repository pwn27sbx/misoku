package app_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// gitIn runs git in dir and returns its trimmed output.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// testIsolationEnv carries the folder TestMain made to a test binary that a
// test starts again as a helper, so the helper reuses it instead of making
// (and, when it is killed, leaving behind) one of its own.
const testIsolationEnv = "MISOKU_APP_TEST_ISOLATION"

// testHome is the throwaway HOME TestMain gives the package.
var testHome string

// TestMain gives every test a throwaway HOME (deleting moves notes to the
// Trash: keep the tests away from the real one), sync not turned off by the
// environment, and git an empty global config and no system config, so the
// user's (or the CI image's) settings never leak into a test. It is the same
// for every test, so it is set once here with os.Setenv: t.Setenv would
// forbid t.Parallel, and the sync tests run in parallel.
func TestMain(m *testing.M) {
	os.Exit(runIsolated(m))
}

func runIsolated(m *testing.M) int {
	if os.Getenv(testIsolationEnv) == "" {
		dir, err := os.MkdirTemp("", "misoku-app-test-")
		if err != nil {
			fmt.Fprintln(os.Stderr, "TestMain:", err)
			return 1
		}
		defer os.RemoveAll(dir)
		home := filepath.Join(dir, "home")
		global := filepath.Join(dir, "gitconfig")
		cfg := "[user]\n\tname = Test\n\temail = test@example.com\n[init]\n\tdefaultBranch = main\n"
		if err := os.Mkdir(home, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "TestMain:", err)
			return 1
		}
		if err := os.WriteFile(global, []byte(cfg), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "TestMain:", err)
			return 1
		}
		os.Setenv(testIsolationEnv, dir)
		os.Setenv("HOME", home)
		os.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
		os.Unsetenv("MISOKU_SYNC")
		os.Setenv("GIT_CONFIG_GLOBAL", global)
		os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	}
	testHome = os.Getenv("HOME")
	return m.Run()
}

// syncedVault is a vault cloned from a local bare remote, holding doc.md
// ("base") committed and pushed, with git configured only for the test (in
// TestMain).
func syncedVault(t *testing.T) (remote, vault string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	parent := t.TempDir()
	gitIn(t, parent, "init", "-q", "--bare", "notes.git")
	remote = filepath.Join(parent, "notes.git")
	vault = clone(t, remote)
	writeOutside(t, vault, "doc.md", "base")
	gitIn(t, vault, "add", "doc.md")
	gitIn(t, vault, "commit", "-q", "-m", "first")
	gitIn(t, vault, "push", "-q", "-u", "origin", "main")
	return remote, vault
}

func clone(t *testing.T, remote string) string {
	t.Helper()
	parent := t.TempDir()
	gitIn(t, parent, "clone", "-q", remote, "vault")
	return filepath.Join(parent, "vault")
}

// pushFromDesk writes name on another device and pushes it.
func pushFromDesk(t *testing.T, remote, name, content string) {
	t.Helper()
	desk := clone(t, remote)
	writeOutside(t, desk, name, content)
	gitIn(t, desk, "add", "-A")
	gitIn(t, desk, "commit", "-q", "-m", "from desk")
	gitIn(t, desk, "push", "-q", "origin", "HEAD:main")
}

// onRemote reports whether origin's main holds name.
func onRemote(t *testing.T, remote, name string) bool {
	t.Helper()
	return exec.Command("git", "-C", remote, "cat-file", "-e", "main:"+name).Run() == nil
}

// syncConfig writes a config.json that lists vault in sync.vaults.
func syncConfig(t *testing.T, vault string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Config{Sync: config.Sync{Vaults: map[string]config.SyncVault{vault: {Device: "laptop"}}}}
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	return path
}

// window opens Misoku on vault with the given config file.
func window(t *testing.T, vault, cfgPath string, opts ...app.Option) *harness {
	t.Helper()
	store, err := core.NewStore(vault)
	if err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, append([]app.Option{app.WithConfigPath(cfgPath)}, opts...)...)
	// The vault's TempDir cleanup was registered earlier, so this runs
	// first: Windows cannot delete the lock file while it is open.
	t.Cleanup(func() { h.m.(app.Model).ReleaseSyncLock() })
	return h
}

// update feeds msg to the model and returns its command without running it.
func (h *harness) update(msg tea.Msg) tea.Cmd {
	h.t.Helper()
	var cmd tea.Cmd
	h.m, cmd = h.m.Update(msg)
	return cmd
}

// finish runs a sync command to the end, as the Bubble Tea runtime would in
// the background, and feeds its result back.
func (h *harness) finish(cmd tea.Cmd) {
	h.t.Helper()
	if cmd == nil {
		h.t.Fatal("no sync command was returned")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		h.send(msg)
	case <-time.After(60 * time.Second):
		h.t.Fatal("the sync did not finish")
	}
}

// initAsksForSync reports whether Init asks the app to sync.
func initAsksForSync(h *harness) bool {
	queue := []tea.Cmd{h.m.Init()}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := runCmd(c).(type) {
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case app.SyncNowMsg:
			return true
		}
	}
	return false
}

func TestOpeningAListedVaultSyncs(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	writeOutside(t, vault, "new.md", "written before opening")
	h := window(t, vault, syncConfig(t, vault))
	if !initAsksForSync(h) {
		t.Fatal("opening a listed vault must start a sync")
	}
	h.finish(h.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the sync on opening must push the new note")
	}
	if v := h.view(); !strings.Contains(v, "Synced") {
		t.Fatalf("the outcome must be shown:\n%s", v)
	}
}

func TestSyncNowKeySyncs(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")
	h.key(tea.KeyEsc)
	writeOutside(t, vault, "new.md", "fresh")
	h.send(tea.KeyMsg{Type: tea.KeySpace})
	h.finish(h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal("space s must sync the vault")
	}
}

func TestSyncNowKeyIsConfigurable(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault), app.WithKeys(map[string][]string{"sync_now": {"ctrl+y"}}))
	h.keys("1")
	h.key(tea.KeyEsc)
	writeOutside(t, vault, "new.md", "fresh")
	h.send(tea.KeyMsg{Type: tea.KeySpace})
	if cmd := h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}); cmd != nil {
		t.Fatal("a configured sync_now replaces the default space s")
	}
	h.finish(h.update(tea.KeyMsg{Type: tea.KeyCtrlY}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the configured key must sync the vault")
	}
}

func TestSyncIsOffForUnlistedVaultsAndMISOKU_SYNC0(t *testing.T) {
	for _, c := range []struct {
		name   string
		listed bool
		env    string
	}{
		{"not listed", false, ""},
		{"MISOKU_SYNC=0", true, "0"},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, vault := syncedVault(t)
			t.Setenv("MISOKU_SYNC", c.env)
			listed := t.TempDir()
			if c.listed {
				listed = vault
			}
			before := gitIn(t, vault, "rev-parse", "HEAD")
			writeOutside(t, vault, "new.md", "fresh")
			h := window(t, vault, syncConfig(t, listed))
			if initAsksForSync(h) {
				t.Fatal("opening must not sync")
			}
			if cmd := h.update(app.SyncNowMsg{}); cmd != nil {
				t.Fatal("a sync request must run nothing")
			}
			h.keys("1")
			h.key(tea.KeyEsc)
			h.send(tea.KeyMsg{Type: tea.KeySpace})
			if cmd := h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}); cmd != nil {
				t.Fatal("space s must run nothing")
			}
			if v := h.view(); !strings.Contains(v, "Sync is off") {
				t.Fatalf("pressing the key says why nothing happens:\n%s", v)
			}
			if after := gitIn(t, vault, "rev-parse", "HEAD"); after != before {
				t.Fatal("nothing may be committed")
			}
			if _, err := os.Stat(filepath.Join(vault, ".git", "misoku-sync.lock")); err == nil {
				t.Fatal("no lock may be taken")
			}
		})
	}
}

func TestSyncRunsInTheBackgroundOneAtATime(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	writeOutside(t, vault, "new.md", "fresh")
	before := gitIn(t, vault, "rev-parse", "HEAD")
	cmd := h.update(app.SyncNowMsg{})
	if cmd == nil {
		t.Fatal("the sync must be returned as a command")
	}
	if gitIn(t, vault, "rev-parse", "HEAD") != before || onRemote(t, remote, "new.md") {
		t.Fatal("Update must not run git: the sync belongs in the command")
	}
	if v := h.view(); !strings.Contains(v, "Syncing") {
		t.Fatalf("a running sync is shown:\n%s", v)
	}
	if again := h.update(app.SyncNowMsg{}); again != nil {
		t.Fatal("a second sync must not start while one runs")
	}
	h.finish(cmd)
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the command must sync the vault")
	}
	writeOutside(t, vault, "later.md", "later")
	h.finish(h.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "later.md") {
		t.Fatal("a sync may start again once the previous one ended")
	}
}

func TestSyncReloadsAnOpenUnmodifiedNote(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")
	h.key(tea.KeyEsc)
	pushFromDesk(t, remote, "doc.md", "from desk")
	h.finish(h.update(app.SyncNowMsg{}))
	if v := h.view(); !strings.Contains(v, "from desk") {
		t.Fatalf("the open note must show what the sync brought:\n%s", v)
	}
}

func TestSyncNeverOverwritesAnUnsavedBuffer(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")
	h.keys("A mine")
	h.key(tea.KeyEsc)
	pushFromDesk(t, remote, "doc.md", "from desk")
	h.finish(h.update(app.SyncNowMsg{}))
	v := h.view()
	if !strings.Contains(v, "base mine") {
		t.Fatalf("unsaved edits must stay in the editor:\n%s", v)
	}
	if !strings.Contains(v, "changed on disk") {
		t.Fatalf("the user must be told the note changed on disk:\n%s", v)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if v := h.view(); !strings.Contains(v, "Overwrite") {
		t.Fatalf("saving over what the sync brought must ask first:\n%s", v)
	}
	if got := readNote(t, vault, "doc.md"); got != "from desk" {
		t.Fatalf("nothing is written before the answer, got %q", got)
	}
}

func TestOnlyOneWindowSyncsAVault(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	cfg := syncConfig(t, vault)
	first := window(t, vault, cfg)
	first.finish(first.update(app.SyncNowMsg{}))

	second := window(t, vault, cfg)
	writeOutside(t, vault, "new.md", "fresh")
	before := gitIn(t, vault, "rev-parse", "HEAD")
	second.finish(second.update(app.SyncNowMsg{}))
	if v := second.view(); !strings.Contains(v, "another Misoku window") {
		t.Fatalf("the second window must say another one syncs:\n%s", v)
	}
	if gitIn(t, vault, "rev-parse", "HEAD") != before || onRemote(t, remote, "new.md") {
		t.Fatal("the second window must not commit or push")
	}

	// The first window keeps the lock for its whole session.
	first.finish(first.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the window that holds the lock keeps syncing")
	}
}

func TestSyncPausesWhenTheVaultIsNotReady(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	gitIn(t, vault, "checkout", "-q", "--detach")
	writeOutside(t, vault, "new.md", "fresh")
	before := gitIn(t, vault, "rev-parse", "HEAD")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	if v := h.view(); !strings.Contains(v, "Sync paused") {
		t.Fatalf("a vault that cannot sync says why:\n%s", v)
	}
	if gitIn(t, vault, "rev-parse", "HEAD") != before || onRemote(t, remote, "new.md") {
		t.Fatal("a paused vault is left untouched")
	}
}

func TestSyncCommandSyncs(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")
	h.key(tea.KeyEsc)
	writeOutside(t, vault, "new.md", "fresh")
	h.keys(":sync")
	parsed := h.update(tea.KeyMsg{Type: tea.KeyEnter})
	if parsed == nil {
		t.Fatal(":sync must produce a command")
	}
	var asked bool
	queue := []tea.Cmd{parsed}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := runCmd(c).(type) {
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case core.SyncMsg:
			asked = true
		}
	}
	if !asked {
		t.Fatal(":sync must ask for a sync")
	}
	h.finish(h.update(core.SyncMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal(":sync must sync the vault")
	}
}

// skippedVaultConfig lists vault under a relative path, which Parse skips
// with a warning.
func skippedVaultConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Config{Sync: config.Sync{Vaults: map[string]config.SyncVault{"notes": {}}}}
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	return path
}

const skippedWarning = `sync: vault "notes" skipped`

func TestSkippedVaultEntryShowsItsWarning(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	h := window(t, vault, skippedVaultConfig(t))
	if v := h.view(); !strings.Contains(v, skippedWarning) {
		t.Fatalf("a skipped sync.vaults entry must be reported:\n%s", v)
	}
}

func TestSyncRequestGivesTheRealReasonWhenAnEntryWasSkipped(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	h := window(t, vault, skippedVaultConfig(t))
	if cmd := h.update(app.SyncNowMsg{}); cmd != nil {
		t.Fatal("a skipped entry must not sync")
	}
	v := h.view()
	if !strings.Contains(v, "Sync is off") || !strings.Contains(v, skippedWarning) {
		t.Fatalf("the request must name the skipped entry, not only say the folder is not listed:\n%s", v)
	}
}

func TestReleasingTheSyncLockLetsAnotherWindowSync(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	cfg := syncConfig(t, vault)
	first := window(t, vault, cfg)
	first.finish(first.update(app.SyncNowMsg{}))
	first.m.(app.Model).ReleaseSyncLock()

	second := window(t, vault, cfg)
	writeOutside(t, vault, "new.md", "fresh")
	second.finish(second.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatalf("once released, another window takes the lock and syncs:\n%s", second.view())
	}
}
