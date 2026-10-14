package app_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/gitsync"

	tea "github.com/charmbracelet/bubbletea"
)

// fakeGit puts a "git" first on PATH that records every call in a log and
// then runs script (a shell body). It returns the log path.
func fakeGit(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake git is a shell script, which Windows cannot run")
	}
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	bin := t.TempDir()
	log := filepath.Join(bin, "calls.log")
	body := fmt.Sprintf("#!/bin/sh\necho \"$@\" >> %q\n%s\n", log, strings.ReplaceAll(script, "REALGIT", real))
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// an edit typed while the sync runs (never saved) is kept, even when
// the sync brings a remote change to that very note.
func TestSyncKeepsAnEditTypedWhileItRuns(t *testing.T) {
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")
	h.key(tea.KeyEsc)
	pushFromDesk(t, remote, "doc.md", "from desk")
	cmd := h.update(app.SyncNowMsg{}) // sync starts with a clean buffer
	h.keys("A typed during sync")
	h.key(tea.KeyEsc)
	h.finish(cmd)
	v := h.view()
	if !strings.Contains(v, "base typed during sync") {
		t.Fatalf("typed text lost:\n%s", v)
	}
	if !strings.Contains(v, "changed on disk") {
		t.Fatalf("user must be warned:\n%s", v)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if v := h.view(); !strings.Contains(v, "Overwrite") {
		t.Fatalf("save must ask:\n%s", v)
	}
	if got := readNote(t, vault, "doc.md"); got != "from desk" {
		t.Fatalf("disk must hold the remote version until the answer, got %q", got)
	}
}

// a note saved while the sync runs (concurrently with git) is never
// lost: after the sync and one more sync, both the user's text and the
// remote text exist somewhere in the vault and on the remote.
func TestSyncLosesNothingSavedWhileItRuns(t *testing.T) {
	for _, delay := range []time.Duration{0, 5 * time.Millisecond, 20 * time.Millisecond, 60 * time.Millisecond, 150 * time.Millisecond} {
		t.Run(delay.String(), func(t *testing.T) {
			remote, vault := syncedVault(t)
			h := window(t, vault, syncConfig(t, vault))
			h.keys("1")
			pushFromDesk(t, remote, "doc.md", "from desk")
			cmd := h.update(app.SyncNowMsg{})
			done := make(chan tea.Msg, 1)
			go func() { done <- cmd() }()
			time.Sleep(delay)
			h.keys("A mine")
			h.key(tea.KeyEsc)
			h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
			if strings.Contains(h.view(), "Overwrite") {
				h.key(tea.KeyEnter) // user insists on their version
			}
			select {
			case msg := <-done:
				h.send(msg)
			case <-time.After(60 * time.Second):
				t.Fatal("sync hung")
			}
			// Second sync pushes whatever is pending.
			h.finish(h.update(app.SyncNowMsg{}))
			all := vaultText(t, vault)
			if !strings.Contains(all, "base mine") {
				t.Fatalf("user's saved text lost; vault:\n%s\nview:\n%s", all, h.view())
			}
			if !strings.Contains(remoteText(t, remote), "base mine") {
				t.Fatalf("user's saved text never reached the remote:\n%s", remoteText(t, remote))
			}
		})
	}
}

func vaultText(t *testing.T, vault string) string {
	t.Helper()
	var b strings.Builder
	filepath.Walk(vault, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(p, ".md") {
			data, _ := os.ReadFile(p)
			fmt.Fprintf(&b, "== %s\n%s\n", p, data)
		}
		return nil
	})
	return b.String()
}

func remoteText(t *testing.T, remote string) string {
	t.Helper()
	out, _ := exec.Command("git", "-C", remote, "grep", "-h", "", "main").CombinedOutput()
	return string(out)
}

// a remote change to an open clean note in a background tab reloads.
func TestSyncReloadsACleanNoteInABackgroundTab(t *testing.T) {
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1") // doc.md
	h.key(tea.KeyEsc)
	writeOutside(t, vault, "other.md", "other")
	h.keys(":e other.md")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "other.md") {
		t.Fatalf("second tab not open:\n%s", v)
	}
	h.finish(h.update(app.SyncNowMsg{}))
	pushFromDesk(t, remote, "doc.md", "desk edit")
	h.finish(h.update(app.SyncNowMsg{}))
	h.keys("[") // back to doc.md
	p, err := h.store.Get("doc.md")
	if err != nil || p.Content != "desk edit" {
		t.Fatalf("disk not updated: %q %v", p.Content, err)
	}
	if v := h.view(); !strings.Contains(v, "desk edit") {
		t.Fatalf("open note not reloaded:\n%s", v)
	}
}

// Helper process: holds the vault lock until killed.
func TestSyncHelperHoldsTheVaultLock(t *testing.T) {
	dir := os.Getenv("MISOKU_TEST_HOLD_LOCK")
	if dir == "" {
		t.Skip("helper only")
	}
	_, ok, err := gitsync.LockVault(context.Background(), gitsync.New(dir, 10*time.Second))
	if err != nil || !ok {
		fmt.Println("nolock", ok, err)
		os.Exit(1)
	}
	fmt.Println("locked")
	time.Sleep(time.Hour)
}

// another process holds the lock → this window does not sync; once
// that process dies, the next sync request takes over.
func TestSyncTakesOverWhenTheHolderDies(t *testing.T) {
	remote, vault := syncedVault(t)
	holder := exec.Command(os.Args[0], "-test.run=^TestSyncHelperHoldsTheVaultLock$")
	holder.Env = append(os.Environ(), "MISOKU_TEST_HOLD_LOCK="+vault)
	out, _ := holder.StdoutPipe()
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		// Killed and waited before TempDir cleanup: Windows cannot delete
		// the lock file a live process holds open.
		holder.Process.Kill()
		holder.Wait()
	}()
	buf := make([]byte, 64)
	n, _ := out.Read(buf)
	if !strings.Contains(string(buf[:n]), "locked") {
		t.Fatalf("helper: %q", buf[:n])
	}
	h := window(t, vault, syncConfig(t, vault))
	writeOutside(t, vault, "new.md", "fresh")
	h.finish(h.update(app.SyncNowMsg{}))
	if v := h.view(); !strings.Contains(v, "another Misoku window") || onRemote(t, remote, "new.md") {
		t.Fatalf("must defer to the holder:\n%s", v)
	}
	holder.Process.Kill()
	holder.Wait()
	h.finish(h.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatalf("must take over after the holder died:\n%s", h.view())
	}
}

// two windows, one through a symlink to the vault, share the lock.
func TestSyncSymlinkedVaultSharesTheLock(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks on Windows needs Developer Mode or admin rights")
	}
	remote, vault := syncedVault(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(vault, link); err != nil {
		t.Fatal(err)
	}
	cfg := syncConfig(t, vault)
	first := window(t, vault, cfg)
	first.finish(first.update(app.SyncNowMsg{}))
	second := window(t, link, cfg)
	if !initAsksForSync(second) {
		t.Fatal("symlinked path must match the listed vault")
	}
	writeOutside(t, vault, "new.md", "fresh")
	second.finish(second.update(app.SyncNowMsg{}))
	if v := second.view(); !strings.Contains(v, "another Misoku window") || onRemote(t, remote, "new.md") {
		t.Fatalf("symlinked window must see the lock held:\n%s", v)
	}
	runtime.KeepAlive(first)
	// And the reverse: listed through the link, opened through the real path.
	_, vault2 := syncedVault(t)
	link2 := filepath.Join(t.TempDir(), "link2")
	if err := os.Symlink(vault2, link2); err != nil {
		t.Fatal(err)
	}
	cfg2 := syncConfig(t, link2)
	a := window(t, link2, cfg2)
	a.finish(a.update(app.SyncNowMsg{}))
	b := window(t, vault2, cfg2)
	b.finish(b.update(app.SyncNowMsg{}))
	if v := b.view(); !strings.Contains(v, "another Misoku window") {
		t.Fatalf("reverse symlink must share lock:\n%s", v)
	}
	runtime.KeepAlive(a)
}

// MISOKU_SYNC=0 and unlisted folders never execute git, through
// open, SyncNowMsg, :sync and space s.
func TestSyncRunsNoGitAtAllWhenOff(t *testing.T) {
	for _, c := range []struct {
		name   string
		listed bool
		env    string
	}{{"unlisted", false, ""}, {"MISOKU_SYNC=0", true, "0"}} {
		t.Run(c.name, func(t *testing.T) {
			_, vault := syncedVault(t)
			listed := t.TempDir()
			if c.listed {
				listed = vault
			}
			cfg := syncConfig(t, listed)
			t.Setenv("MISOKU_SYNC", c.env)
			log := fakeGit(t, "exit 1")
			h := window(t, vault, cfg)
			if initAsksForSync(h) {
				t.Fatal("must not sync on open")
			}
			if cmd := h.update(app.SyncNowMsg{}); cmd != nil {
				runCmd(cmd)
			}
			h.keys("1")
			h.key(tea.KeyEsc)
			h.keys(":sync")
			if cmd := h.update(tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
				drain(h, cmd)
			}
			if v := h.view(); !strings.Contains(v, "Sync is off") {
				t.Fatalf(":sync must say why:\n%s", v)
			}
			h.send(tea.KeyMsg{Type: tea.KeySpace})
			if cmd := h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}); cmd != nil {
				drain(h, cmd)
			}
			if data, err := os.ReadFile(log); err == nil {
				t.Fatalf("git ran: %s", data)
			}
		})
	}
}

// drain runs cmd and any batched commands (not ticks), feeding results back.
func drain(h *harness, cmd tea.Cmd) {
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		msg := runCmd(c)
		if b, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, b...)
			continue
		}
		if msg != nil {
			if next := h.update(msg); next != nil {
				queue = append(queue, next)
			}
		}
	}
}

// :sync and sync_now behave the same (status while running,
// one at a time, same outcome).
func TestSyncCommandAndKeyBehaveTheSame(t *testing.T) {
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")
	h.key(tea.KeyEsc)
	// key first
	h.send(tea.KeyMsg{Type: tea.KeySpace})
	keyCmd := h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if keyCmd == nil || !strings.Contains(h.view(), "Syncing") {
		t.Fatal("key must start a sync")
	}
	// :sync while it runs → already syncing, no second sync
	h.keys(":sync")
	parsed := h.update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := runCmd(parsed)
	var cmd tea.Cmd
	if b, ok := msg.(tea.BatchMsg); ok {
		for _, c := range b {
			if c != nil {
				if _, ok := runCmd(c).(core.SyncMsg); ok {
					cmd = h.update(core.SyncMsg{})
				}
			}
		}
	} else if _, ok := msg.(core.SyncMsg); ok {
		cmd = h.update(core.SyncMsg{})
	}
	if cmd != nil {
		t.Fatal(":sync must not start a second sync while one runs")
	}
	if !strings.Contains(h.view(), "Already syncing") {
		t.Fatalf(":sync while running:\n%s", h.view())
	}
	h.finish(keyCmd)
	writeOutside(t, vault, "n.md", "x")
	h.finish(h.update(core.SyncMsg{}))
	if !onRemote(t, remote, "n.md") || !strings.Contains(h.view(), "Synced") {
		t.Fatalf(":sync outcome:\n%s", h.view())
	}
}

// in INSERT mode "space s" types text and does not sync.
func TestSyncKeyTypesTextInInsertMode(t *testing.T) {
	_, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")
	h.keys("A")
	h.send(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	if cmd := h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}); cmd != nil {
		if _, ok := runCmd(cmd).(app.SyncNowMsg); ok {
			t.Fatal("typing must not sync")
		}
	}
	if strings.Contains(h.view(), "Syncing") {
		t.Fatal("typing must not sync")
	}
	if !strings.Contains(h.view(), "base s") {
		t.Fatalf("typed text missing:\n%s", h.view())
	}
}

// Update never blocks on git, even when git hangs.
func TestSyncNeverBlocksUpdateOnAHungGit(t *testing.T) {
	_, vault := syncedVault(t)
	cfg := syncConfig(t, vault)
	fakeGit(t, "sleep 5; exec REALGIT \"$@\"")
	start := time.Now()
	h := window(t, vault, cfg)
	h.m.Init()
	cmd := h.update(app.SyncNowMsg{})
	h.keys("1")
	h.key(tea.KeyEsc)
	if el := time.Since(start); el > 2*time.Second {
		t.Fatalf("New/Update blocked for %v", el)
	}
	_ = cmd // not run: Bubble Tea would run it in the background
}

// help lists sync_now with its key and :sync among the commands.
func TestSyncHelpListsTheKeyAndCommand(t *testing.T) {
	_, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("?")
	v := h.view()
	if !strings.Contains(v, "SPC s") || !strings.Contains(v, ":sync") {
		t.Fatalf("help must list sync_now:\n%s", v)
	}
}
