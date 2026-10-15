package app_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"

	tea "github.com/charmbracelet/bubbletea"
)

// quit runs what Misoku does once its screen has closed (S14) and returns
// the lines it printed and how long it took.
func quit(h *harness, ctx context.Context, limit time.Duration) ([]string, time.Duration) {
	h.t.Helper()
	var out bytes.Buffer
	start := time.Now()
	h.m.(app.Model).SyncOnQuit(ctx, &out, limit)
	took := time.Since(start)
	text := strings.TrimRight(out.String(), "\n")
	if text == "" {
		return nil, took
	}
	return strings.Split(text, "\n"), took
}

// wantQuitLines checks the two lines quitting prints: what it does, then
// how it went, starting with outcome.
func wantQuitLines(t *testing.T, lines []string, outcome string) {
	t.Helper()
	if len(lines) != 2 || lines[0] != "Pushing changes…" || !strings.HasPrefix(lines[1], outcome) {
		t.Fatalf("want %q then a line starting %q, got %q", "Pushing changes…", outcome, lines)
	}
}

// committed reports whether the vault's HEAD holds name.
func committed(t *testing.T, vault, name string) bool {
	t.Helper()
	return exec.Command("git", "-C", vault, "cat-file", "-e", "HEAD:"+name).Run() == nil
}

// lockFree reports whether another window could take the vault's sync lock.
func lockFree(t *testing.T, vault string) bool {
	t.Helper()
	l, ok, err := gitsync.LockVault(context.Background(), gitsync.New(vault, 20*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	l.Unlock()
	return ok
}

// syncedWindow is a window on a synced vault that has synced once, so it
// holds the vault's lock as Misoku does for the whole session.
func syncedWindow(t *testing.T) (h *harness, remote, vault string) {
	t.Helper()
	remote, vault = syncedVault(t)
	h = window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	if lockFree(t, vault) {
		t.Fatal("the window must hold the lock after its first sync")
	}
	return h, remote, vault
}

// S14: what was saved is committed and pushed after the screen closes,
// saying so on the terminal, and the lock is let go.
func TestQuitPushesTheChanges(t *testing.T) {
	t.Parallel()
	h, remote, vault := syncedWindow(t)
	writeOutside(t, vault, "new.md", "fresh")
	lines, _ := quit(h, context.Background(), 20*time.Second)
	wantQuitLines(t, lines, "Changes pushed")
	if !onRemote(t, remote, "new.md") {
		t.Fatal("quitting must push the saved note")
	}
	if !lockFree(t, vault) {
		t.Fatal("quitting must release the sync lock")
	}
}

// Offline: one line says why, the commit stays for the next sync, and
// quitting goes on as usual.
func TestQuitOfflineKeepsTheCommitAndSaysWhy(t *testing.T) {
	t.Parallel()
	h, remote, vault := syncedWindow(t)
	if err := os.Rename(remote, remote+".gone"); err != nil {
		t.Fatal(err)
	}
	writeOutside(t, vault, "new.md", "fresh")
	lines, _ := quit(h, context.Background(), 20*time.Second)
	wantQuitLines(t, lines, "Not pushed")
	if !strings.Contains(lines[1], "next sync") {
		t.Errorf("the line must say the next sync pushes it: %q", lines[1])
	}
	if !committed(t, vault, "new.md") {
		t.Fatal("the note must be committed, waiting for the next sync")
	}
	if !lockFree(t, vault) {
		t.Fatal("quitting must release the sync lock")
	}
}

// A folder that does not sync runs no git and prints nothing on quit.
func TestQuitDoesNothingWhenSyncIsOff(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(cfg, config.Config{}); err != nil {
		t.Fatal(err)
	}
	h := window(t, vault, cfg)
	before := gitIn(t, vault, "rev-parse", "HEAD")
	writeOutside(t, vault, "new.md", "fresh")
	if lines, _ := quit(h, context.Background(), 20*time.Second); lines != nil {
		t.Fatalf("nothing must be printed, got %q", lines)
	}
	if gitIn(t, vault, "rev-parse", "HEAD") != before || committed(t, vault, "new.md") {
		t.Fatal("a folder that does not sync is never committed")
	}
}

// A window that never took the lock (another window syncs the vault)
// leaves the syncing to the holder: nothing is printed or pushed.
func TestQuitLeavesTheVaultToTheWindowHoldingTheLock(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	cfg := syncConfig(t, vault)
	first := window(t, vault, cfg)
	first.finish(first.update(app.SyncNowMsg{}))
	second := window(t, vault, cfg)
	second.finish(second.update(app.SyncNowMsg{}))
	writeOutside(t, vault, "new.md", "fresh")
	if lines, _ := quit(second, context.Background(), 20*time.Second); lines != nil {
		t.Fatalf("nothing must be printed, got %q", lines)
	}
	if onRemote(t, remote, "new.md") || committed(t, vault, "new.md") {
		t.Fatal("only the window holding the lock runs git")
	}
}

// slowPushGit is a fake git that makes every push take pause. It returns
// its log of calls and the file where it records any git that starts while
// another one runs.
func slowPushGit(t *testing.T, pause string) (calls, overlap string) {
	t.Helper()
	dir := t.TempDir()
	busy, overlap := filepath.Join(dir, "busy"), filepath.Join(dir, "overlap")
	calls = fakeGit(t, `if ! mkdir "`+busy+`" 2>/dev/null; then echo "$@" >> "`+overlap+`"; fi
case " $* " in *" push "*) sleep `+pause+`;; esac
REALGIT "$@"; s=$?; rmdir "`+busy+`" 2>/dev/null; exit $s`)
	return calls, overlap
}

// waitForCall waits until the fake git's log holds a call with word.
func waitForCall(t *testing.T, calls, word string) {
	t.Helper()
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if b, _ := os.ReadFile(calls); strings.Contains(" "+string(b), " "+word+" ") {
			return
		}
	}
	t.Fatalf("git %s never ran", word)
}

// A sync still running when the screen closes is waited for, never run
// alongside a second one; then what was saved after it started is pushed.
func TestQuitWaitsForARunningSync(t *testing.T) {
	h, remote, vault := syncedWindow(t)
	calls, overlap := slowPushGit(t, "1")
	writeOutside(t, vault, "new.md", "fresh")
	running := make(chan tea.Msg, 1)
	cmd := h.update(app.SyncNowMsg{})
	go func() { running <- cmd() }()
	// Saved once that sync committed: only a sync after it can push it.
	waitForCall(t, calls, "push")
	writeOutside(t, vault, "later.md", "saved while it ran")

	lines, _ := quit(h, context.Background(), 20*time.Second)
	<-running
	wantQuitLines(t, lines, "Changes pushed")
	if !onRemote(t, remote, "new.md") || !onRemote(t, remote, "later.md") {
		t.Fatal("both the running sync's note and the one saved after it must be pushed")
	}
	if b, err := os.ReadFile(overlap); err == nil {
		t.Fatalf("two gits ran at once:\n%s", b)
	}
	if !lockFree(t, vault) {
		t.Fatal("quitting must release the sync lock")
	}
}

// hangingRemote makes origin an ssh remote whose ssh answers only after
// pause (a shell sleep).
//
// It returns the file the fake ssh creates as it starts, so a test can
// tell when git reached the remote.
func hangingRemote(t *testing.T, vault, pause string) (started string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake ssh is a shell script")
	}
	t.Setenv("GIT_SSH", "")
	os.Unsetenv("GIT_SSH")
	dir := t.TempDir()
	script, started := filepath.Join(dir, "fake-ssh"), filepath.Join(dir, "started")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n: > '"+started+"'\nsleep "+pause+"\nexit 255\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_SSH_COMMAND", "sh '"+script+"'")
	gitIn(t, vault, "remote", "set-url", "origin", "ssh://git@example.invalid/notes.git")
	return started
}

// slack is how much past its limit quitting may run: the time to kill git
// and collect its output.
const slack = 2500 * time.Millisecond

// A remote that never answers is given up at the limit: one line says
// so, the commit stays, git leaves no lock behind.
func TestQuitNeverRunsPastTheLimit(t *testing.T) {
	h, _, vault := syncedWindow(t)
	hangingRemote(t, vault, "30")
	writeOutside(t, vault, "new.md", "fresh")
	lines, took := quit(h, context.Background(), time.Second)
	if took > time.Second+slack {
		t.Fatalf("a 1 s limit took %s", took)
	}
	wantQuitLines(t, lines, "Not pushed")
	if !strings.Contains(lines[1], "1s") {
		t.Errorf("the line must say the limit ran out: %q", lines[1])
	}
	if !committed(t, vault, "new.md") {
		t.Fatal("the note must be committed, waiting for the next sync")
	}
	if _, err := os.Lstat(filepath.Join(vault, ".git", "index.lock")); err == nil {
		t.Fatal("git must leave no index.lock behind")
	}
	if !lockFree(t, vault) {
		t.Fatal("quitting must release the sync lock")
	}
}

// A running sync that outlasts the limit is not waited for past it, and
// no second git starts beside it.
func TestQuitGivesUpOnARunningSyncAtTheLimit(t *testing.T) {
	h, _, vault := syncedWindow(t)
	hangingRemote(t, vault, "3")
	writeOutside(t, vault, "new.md", "fresh")
	running := make(chan tea.Msg, 1)
	cmd := h.update(app.SyncNowMsg{})
	go func() { running <- cmd() }()
	// Windows cannot remove the vault while that sync's git still runs.
	t.Cleanup(func() { <-running })

	lines, took := quit(h, context.Background(), time.Second)
	if took > time.Second+slack {
		t.Fatalf("a 1 s limit took %s", took)
	}
	wantQuitLines(t, lines, "Not pushed")
	if !strings.Contains(lines[1], "still running") {
		t.Errorf("the line must say the running sync did not finish: %q", lines[1])
	}
}

// Ctrl+C while pushing (the context is cancelled) stops git at once and
// leaves the repository whole, the commit waiting for the next sync.
func TestQuitCancelledStopsGitCleanly(t *testing.T) {
	h, _, vault := syncedWindow(t)
	started := hangingRemote(t, vault, "30")
	writeOutside(t, vault, "new.md", "fresh")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Ctrl+C once the push is waiting on the remote, whatever the load:
	// by then the note is committed.
	cancelled := make(chan time.Time, 1)
	go func() {
		for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
			if _, err := os.Stat(started); err == nil {
				break
			}
		}
		cancelled <- time.Now()
		cancel()
	}()
	lines, _ := quit(h, ctx, 20*time.Second)
	if _, err := os.Stat(started); err != nil {
		t.Fatalf("the push never reached the remote: %v", err)
	}
	if took := time.Since(<-cancelled); took > slack {
		t.Fatalf("a cancelled push took %s to stop", took)
	}
	wantQuitLines(t, lines, "Not pushed")
	if !strings.Contains(lines[1], "cancelled") {
		t.Errorf("the line must say it was cancelled: %q", lines[1])
	}
	if !committed(t, vault, "new.md") {
		t.Fatal("the note must be committed, waiting for the next sync")
	}
	if _, err := os.Lstat(filepath.Join(vault, ".git", "index.lock")); err == nil {
		t.Fatal("git must leave no index.lock behind")
	}
	gitIn(t, vault, "fsck", "--no-progress")
}

// The first sync, which takes the lock, may still be running at quit: its
// lock is the one pushed with and then let go.
func TestQuitTakesTheLockOfTheFirstSync(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	writeOutside(t, vault, "new.md", "fresh")
	running := make(chan tea.Msg, 1)
	cmd := h.update(app.SyncNowMsg{})
	go func() { running <- cmd() }()
	lines, _ := quit(h, context.Background(), 20*time.Second)
	<-running
	if len(lines) != 2 || strings.HasPrefix(lines[1], "Not pushed") {
		t.Fatalf("quitting must push with the first sync's lock, got %q", lines)
	}
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the note must reach origin")
	}
	if !lockFree(t, vault) {
		t.Fatal("quitting must release the first sync's lock")
	}
}

// A vault the user left mid-way (here a detached HEAD from a terminal)
// pauses the final push as it pauses every sync (S2): nothing is touched.
func TestQuitRespectsAPausedVault(t *testing.T) {
	t.Parallel()
	h, remote, vault := syncedWindow(t)
	gitIn(t, vault, "checkout", "-q", "--detach")
	before := gitIn(t, vault, "rev-parse", "HEAD")
	writeOutside(t, vault, "new.md", "fresh")
	lines, _ := quit(h, context.Background(), 20*time.Second)
	wantQuitLines(t, lines, "Not pushed, sync is paused")
	if gitIn(t, vault, "rev-parse", "HEAD") != before || onRemote(t, remote, "new.md") {
		t.Fatal("a paused vault is neither committed nor pushed")
	}
	if !lockFree(t, vault) {
		t.Fatal("quitting must release the sync lock")
	}
}

// S2/S14: origin was rewritten (a force-push) after the window's last
// sync, which paused: quitting does not push and says sync is paused
// because origin was rewritten, not that the next sync joins it.
func TestQuitAfterARewrittenOriginSaysSyncIsPaused(t *testing.T) {
	t.Parallel()
	h, remote, vault := syncedWindow(t)
	desk := clone(t, remote)
	gitIn(t, desk, "commit", "-q", "--amend", "-m", "rewrite")
	gitIn(t, desk, "push", "-q", "-f", "origin", "HEAD:main")
	h.finish(h.update(app.SyncNowMsg{}))
	writeOutside(t, vault, "new.md", "fresh")
	lines, _ := quit(h, context.Background(), 20*time.Second)
	wantQuitLines(t, lines, "Not pushed, sync is paused: origin's main was rewritten")
	if onRemote(t, remote, "new.md") {
		t.Fatal("quitting must never push over a rewritten origin")
	}
}
