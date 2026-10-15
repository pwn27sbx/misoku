package app_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// S14 verifier probes. Each one drives SyncOnQuit through the public
// interface with real git and a local bare remote.

// readFile returns name's content in dir, failing the test when missing.
func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Sync off (unlisted folder, MISOKU_SYNC=0) and a window that never held
// the lock: quitting runs no git at all and prints nothing.
func TestQuitRunsNoGitWhenOffOrWithoutTheLock(t *testing.T) {
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
			calls := fakeGit(t, "exit 1")
			h := window(t, vault, cfg)
			writeOutside(t, vault, "new.md", "fresh")
			if lines, _ := quit(h, context.Background(), 5*time.Second); lines != nil {
				t.Fatalf("nothing must be printed, got %q", lines)
			}
			if b, err := os.ReadFile(calls); err == nil {
				t.Fatalf("git ran on quit: %s", b)
			}
		})
	}
	t.Run("window without the lock", func(t *testing.T) {
		_, vault := syncedVault(t)
		cfg := syncConfig(t, vault)
		first := window(t, vault, cfg)
		first.finish(first.update(app.SyncNowMsg{}))
		second := window(t, vault, cfg)
		second.finish(second.update(app.SyncNowMsg{}))
		calls := fakeGit(t, "REALGIT \"$@\"")
		writeOutside(t, vault, "new.md", "fresh")
		if lines, _ := quit(second, context.Background(), 5*time.Second); lines != nil {
			t.Fatalf("nothing must be printed, got %q", lines)
		}
		if b, err := os.ReadFile(calls); err == nil {
			t.Fatalf("git ran on quit: %s", b)
		}
	})
}

// Origin moved on (another device pushed the same note): the final push is
// turned down, nothing on disk changes, the commit stays, and the next sync
// joins both sides without losing either version.
func TestQuitRejectedPushLosesNothing(t *testing.T) {
	t.Parallel()
	h, remote, vault := syncedWindow(t)
	pushFromDesk(t, remote, "doc.md", "from desk")
	writeOutside(t, vault, "doc.md", "from laptop")
	writeOutside(t, vault, "new.md", "fresh")
	lines, _ := quit(h, context.Background(), 20*time.Second)
	wantQuitLines(t, lines, "Not pushed, the next sync tries again")
	if got := readFile(t, vault, "doc.md"); got != "from laptop" {
		t.Fatalf("the quit push must not touch the note, doc.md = %q", got)
	}
	if !committed(t, vault, "new.md") || onRemote(t, remote, "new.md") {
		t.Fatal("the commit must stay local, waiting for the next sync")
	}
	if st := gitIn(t, vault, "status", "--porcelain"); st != "" {
		t.Fatalf("the vault must be clean after the commit:\n%s", st)
	}
	if !lockFree(t, vault) {
		t.Fatal("quitting must release the sync lock")
	}

	// The next session's sync joins both sides.
	next := window(t, vault, syncConfig(t, vault))
	next.finish(next.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the next sync must push the pending commit")
	}
	if got := readFile(t, vault, "doc.md"); got != "from laptop" {
		t.Fatalf("this device's version must stay in the note, got %q", got)
	}
	entries, _ := os.ReadDir(vault)
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "doc (conflict") {
			found = readFile(t, vault, e.Name()) == "from desk"
		}
	}
	if !found {
		t.Fatal("the other device's version must be kept in a conflict copy")
	}
}

// Ctrl+C while the final commit holds git's index.lock: git is killed,
// its own index.lock removed, the repository consistent, the note on
// disk untouched, and the next sync commits and pushes it.
func TestQuitCancelledDuringCommit(t *testing.T) {
	h, remote, vault := syncedWindow(t)
	lockFile := filepath.Join(vault, ".git", "index.lock")
	fakeGit(t, `case " $* " in *" commit "*) : > "`+lockFile+`"; sleep 30;; esac
REALGIT "$@"`)
	writeOutside(t, vault, "new.md", "fresh")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(500*time.Millisecond, cancel)
	lines, took := quit(h, ctx, 20*time.Second)
	if took > 500*time.Millisecond+slack {
		t.Fatalf("a cancelled commit took %s", took)
	}
	wantQuitLines(t, lines, "Not pushed")
	if !strings.Contains(lines[1], "cancelled") {
		t.Errorf("the line must say it was cancelled: %q", lines[1])
	}
	if _, err := os.Lstat(lockFile); err == nil {
		t.Fatal("Misoku's own index.lock must be removed after the kill")
	}
	if got := readFile(t, vault, "new.md"); got != "fresh" {
		t.Fatalf("the note on disk changed: %q", got)
	}
	os.Setenv("PATH", strings.SplitN(os.Getenv("PATH"), string(os.PathListSeparator), 2)[1])
	gitIn(t, vault, "fsck", "--no-progress")
	if !lockFree(t, vault) {
		t.Fatal("quitting must release the sync lock")
	}
	next := window(t, vault, syncConfig(t, vault))
	next.finish(next.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the next sync must commit and push the note")
	}
}

// A git that hangs on every command (not only ssh) is still given up at
// the 10 s limit used in production.
func TestQuitHungGitRespectsTheProductionLimit(t *testing.T) {
	h, _, vault := syncedWindow(t)
	fakeGit(t, `sleep 60; REALGIT "$@"`)
	writeOutside(t, vault, "new.md", "fresh")
	lines, took := quit(h, context.Background(), app.QuitSyncLimit)
	t.Logf("took %s, lines %q", took, lines)
	if took > app.QuitSyncLimit+slack {
		t.Fatalf("a %s limit took %s", app.QuitSyncLimit, took)
	}
	if len(lines) != 2 || !strings.HasPrefix(lines[1], "Not pushed") {
		t.Fatalf("want a Not pushed line, got %q", lines)
	}
	if out, _ := exec.Command("pgrep", "-f", "sleep 60").Output(); len(out) > 0 {
		t.Errorf("a hung git's children survived the limit: pids %s", out)
	}
}

// Ctrl+C with a git that hangs on every command: how long until quitting
// gives up. Ctrl+C should stop at once.
//
// Pins advisory A1: the first command resolves the ssh setting
// with `git config` under its own 5 s limit (Runner.sshConfig), which
// must follow ctx, or Ctrl+C would wait ~5 s.
func TestQuitCtrlCWithHungGit(t *testing.T) {
	h, _, vault := syncedWindow(t)
	fakeGit(t, `sleep 60; REALGIT "$@"`)
	writeOutside(t, vault, "new.md", "fresh")
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	lines, took := quit(h, ctx, app.QuitSyncLimit)
	t.Logf("took %s, lines %q", took, lines)
	if took > 300*time.Millisecond+slack {
		t.Fatalf("Ctrl+C took %s to stop the final push", took)
	}
}

// Ctrl+C while waiting for a sync still running: quitting returns at once,
// the running sync's lock is not released under it (two gits never run),
// and once Misoku is gone another window can sync the vault.
func TestQuitCtrlCWhileASyncRuns(t *testing.T) {
	h, remote, vault := syncedWindow(t)
	hangingRemote(t, vault, "3")
	writeOutside(t, vault, "new.md", "fresh")
	running := make(chan tea.Msg, 1)
	cmd := h.update(app.SyncNowMsg{})
	go func() { running <- cmd() }()
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	lines, took := quit(h, ctx, app.QuitSyncLimit)
	if took > 200*time.Millisecond+slack {
		t.Fatalf("Ctrl+C took %s", took)
	}
	wantQuitLines(t, lines, "Not pushed")
	if !strings.Contains(lines[1], "cancelled") {
		t.Errorf("the line must say it was cancelled: %q", lines[1])
	}
	if lockFree(t, vault) {
		t.Fatal("the lock of a sync still running must not be released under it")
	}
	<-running
	// Misoku exits: the OS releases the lock (simulated here).
	h.m.(app.Model).ReleaseSyncLock()
	gitIn(t, vault, "remote", "set-url", "origin", remote)
	os.Unsetenv("GIT_SSH_COMMAND")
	other := window(t, vault, syncConfig(t, vault))
	other.finish(other.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "new.md") {
		t.Fatal("another window must sync the vault after Misoku exits")
	}
}
