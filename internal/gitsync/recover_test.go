package gitsync_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// deadPid is the id of a process that has already exited.
func deadPid(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("git", "version")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return cmd.Process.Pid
}

// interrupted leaves dir as a Misoku that died mid-sync leaves it: its
// marker names git process pid, started at start, and git's index.lock is
// still there. It returns the lock's path.
func interrupted(t *testing.T, dir string, pid int, start time.Time) string {
	t.Helper()
	gitDir := git(t, dir, "rev-parse", "--absolute-git-dir")
	if err := gitsync.WriteRunMarker(gitDir, pid, start); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(gitDir, "index.lock")
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return lock
}

func wantLock(t *testing.T, lock string, want bool) {
	t.Helper()
	_, err := os.Lstat(lock)
	if got := err == nil; got != want {
		t.Fatalf("index.lock there = %v, want %v (err %v)", got, want, err)
	}
}

// S13: a lock left while Misoku's dead git ran. On Windows that git died
// with Misoku, so the lock is its own: removed, and the sync picks up where
// it stopped. Elsewhere that git outlived Misoku and removed its own lock,
// so a lock there now may be the user's: the sync stops on it, and only
// the user's confirmation (RemoveLeftoverLock) removes it.
func TestSyncLeftoverLockAfterMisokuDied(t *testing.T) {
	t.Parallel()
	remote, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "a.md", "a\n")
	lock := interrupted(t, laptop, deadPid(t), time.Now().Add(-time.Second))

	if st := detect(t, laptop); st.Status != gitsync.Ready {
		t.Fatalf("Misoku's own interrupted sync must not pause the vault: %+v", st)
	}
	res := syncVault(t, laptop, "laptop", false)
	if runtime.GOOS != "windows" {
		var locked *gitsync.LockedError
		if res.Outcome != gitsync.SyncFailed || !errors.As(res.Err, &locked) || locked.Path != lock {
			t.Fatalf("want the sync stopped on the lock, got %v: %v", res.Outcome, res.Err)
		}
		wantLock(t, lock, true)
		shown, ok, err := gitsync.StatLock(lock)
		if err != nil || !ok {
			t.Fatal(ok, err)
		}
		if err := gitsync.RemoveLeftoverLock(shown); err != nil {
			t.Fatal(err)
		}
		res = syncVault(t, laptop, "laptop", false)
	}
	wantOutcome(t, res, gitsync.Synced)
	wantLock(t, lock, false)
	wantInSync(t, remote, laptop, "main")
	gitDir := git(t, laptop, "rev-parse", "--absolute-git-dir")
	if entries, _ := filepath.Glob(filepath.Join(gitDir, "misoku-sync.run*")); len(entries) != 0 {
		t.Errorf("the marker must be gone after a sync: %q", entries)
	}
}

// S13 + S2: a lock with no marker is the user's (or another program's):
// sync pauses with a notice and touches nothing.
func TestSyncNeverTouchesAnIndexLockItDidNotLeave(t *testing.T) {
	t.Parallel()
	_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "a.md", "a\n")
	lock := filepath.Join(laptop, ".git", "index.lock")
	if err := os.WriteFile(lock, []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := git(t, laptop, "rev-parse", "HEAD")

	st := detect(t, laptop)
	wantPaused(t, st, gitsync.ReasonLocked, "index.lock")
	res := syncVault(t, laptop, "laptop", false)
	if res.Outcome == gitsync.Synced {
		t.Fatal("a sync cannot go on past someone else's lock")
	}
	if b, err := os.ReadFile(lock); err != nil || string(b) != "theirs" {
		t.Fatalf("the lock must stay as it was: %q %v", b, err)
	}
	if after := git(t, laptop, "rev-parse", "HEAD"); after != before {
		t.Error("nothing may be committed past someone else's lock")
	}
}

// S13: while the git of the interrupted sync still runs, its lock is in use.
func TestSyncWaitsWhileTheInterruptedGitStillRuns(t *testing.T) {
	t.Parallel()
	_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	lock := interrupted(t, laptop, os.Getpid(), time.Now().Add(-time.Second))
	res := syncVault(t, laptop, "laptop", false)
	wantOutcome(t, res, gitsync.SyncRetry)
	if !strings.Contains(res.Err.Error(), "still running") {
		t.Errorf("the error must say git still runs: %v", res.Err)
	}
	wantLock(t, lock, true)
}

// A lock older than the command Misoku's marker names was there before
// that command: someone else's, so it stays.
func TestSyncLeavesAnIndexLockOlderThanItsCommand(t *testing.T) {
	t.Parallel()
	_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	start := time.Now()
	lock := interrupted(t, laptop, deadPid(t), start)
	old := start.Add(-time.Hour)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatal(err)
	}
	if res := syncVault(t, laptop, "laptop", false); res.Outcome == gitsync.Synced {
		t.Fatal("a sync cannot go on past someone else's lock")
	}
	wantLock(t, lock, true)
}

// The decision itself, for both kinds of OS: only right after Misoku killed
// its git, or where git dies with Misoku (Windows), and only for a lock
// made while the command could run.
func TestLeftoverIsMisokusDecision(t *testing.T) {
	t.Parallel()
	start := time.Now()
	limit := 20 * time.Second
	in, before, after := start.Add(time.Second), start.Add(-time.Minute), start.Add(time.Minute)
	for _, c := range []struct {
		name                      string
		mod                       time.Time
		killedNow, diesWithMisoku bool
		want                      bool
	}{
		{"unix, on return, in the window", in, false, false, false},
		{"unix, just killed, in the window", in, true, false, true},
		{"unix, just killed, after the limit", after, true, false, false},
		{"windows, on return, in the window", in, false, true, true},
		{"windows, on return, before the start", before, false, true, false},
		{"windows, on return, after the limit", after, false, true, false},
	} {
		if got := gitsync.LeftoverIsMisokus(1, start, limit, c.mod, c.killedNow, c.diesWithMisoku); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// RemoveLeftoverLock removes only the very lock the user was shown.
func TestRemoveLeftoverLockOnlyRemovesTheLockAsShown(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lock := filepath.Join(dir, "index.lock")
	if err := os.WriteFile(lock, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	shown, ok, err := gitsync.StatLock(lock)
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	if err := os.WriteFile(lock, []byte("another git"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitsync.RemoveLeftoverLock(shown); !errors.Is(err, gitsync.ErrLockChanged) {
		t.Fatalf("a changed lock must stay, got %v", err)
	}
	wantLock(t, lock, true)
	if shown, _, _ = gitsync.StatLock(lock); gitsync.RemoveLeftoverLock(shown) != nil {
		t.Fatal("the lock as shown must go")
	}
	wantLock(t, lock, false)
	if err := gitsync.RemoveLeftoverLock(shown); !errors.Is(err, gitsync.ErrLockChanged) {
		t.Fatalf("a lock already gone reports it, got %v", err)
	}
}
