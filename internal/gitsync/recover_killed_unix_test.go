//go:build !windows

package gitsync_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// killedMidIndexCommand runs a sync in a child process (Misoku) with the
// given per-command time limit, and kills that child while its git holds
// index.lock (an fsmonitor hook makes git wait 2 s there). The orphaned
// git then finishes by itself and removes its own lock; the marker stays.
// It returns once that git is gone.
func killedMidIndexCommand(t *testing.T, name string, limit time.Duration) string {
	t.Helper()
	_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	flag := filepath.Join(t.TempDir(), "in-lock")
	hook := fakeProgram(t, t.TempDir(), "fsmonitor",
		`[ -e .git/index.lock ] || exit 0`+"\n"+`echo $PPID > "`+flag+`"`+"\nsleep 2\n")
	git(t, laptop, "config", "core.fsmonitor", hook)
	write(t, laptop, "a.md", "a\n")

	child := exec.Command(os.Args[0], "-test.run=^"+name+"$")
	child.Env = append(os.Environ(), "MISOKU_SYNC_KILLED_CHILD="+laptop, "MISOKU_SYNC_KILLED_LIMIT="+limit.String())
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	pid := 0
	for deadline := time.Now().Add(20 * time.Second); pid == 0; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			child.Process.Kill()
			t.Fatal("the child's git never took index.lock")
		}
		if b, err := os.ReadFile(flag); err == nil {
			pid, _ = strconv.Atoi(string(b[:len(b)-1]))
		}
	}
	child.Process.Kill() // Misoku dies while its git holds index.lock
	child.Wait()
	for deadline := time.Now().Add(15 * time.Second); syscall.Kill(pid, 0) == nil; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the orphaned git never finished")
		}
	}
	git(t, laptop, "config", "--unset", "core.fsmonitor")
	if _, err := os.Lstat(filepath.Join(laptop, ".git", "misoku-sync.running")); err != nil {
		t.Fatalf("expected a stale marker: %v", err)
	}
	wantLock(t, filepath.Join(laptop, ".git", "index.lock"), false) // git cleaned up
	return laptop
}

func childSync(t *testing.T) bool {
	dir := os.Getenv("MISOKU_SYNC_KILLED_CHILD")
	if dir == "" {
		return false
	}
	limit, _ := time.ParseDuration(os.Getenv("MISOKU_SYNC_KILLED_LIMIT"))
	held, ok, err := gitsync.TryLock(filepath.Join(t.TempDir(), "sync.lock"))
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	gitsync.SyncVault(context.Background(), gitsync.New(dir, limit), held, gitsync.SyncOptions{Device: "laptop"})
	return true
}

// Misoku killed during an index command; the user opens
// git within that command's time limit (here the app's 1 minute).
func TestSyncKilledMidIndexNeverDeletesAUsersLockWithinTheLimit(t *testing.T) {
	if childSync(t) {
		return
	}
	laptop := killedMidIndexCommand(t, "TestSyncKilledMidIndexNeverDeletesAUsersLockWithinTheLimit", time.Minute)
	write(t, laptop, "seed.md", "edited by the user\n")
	lock, _ := usersCommit(t, laptop)
	res := syncVault(t, laptop, "laptop", false)
	t.Logf("%v: %v", res.Outcome, res.Err)
	wantLock(t, lock, true)
}

// Same, but the user's git takes the lock after the killed
// command's limit (+2 s slack) has passed.
func TestSyncKilledMidIndexNeverDeletesAUsersLockAfterTheLimit(t *testing.T) {
	if childSync(t) {
		return
	}
	limit := 3 * time.Second
	start := time.Now()
	laptop := killedMidIndexCommand(t, "TestSyncKilledMidIndexNeverDeletesAUsersLockAfterTheLimit", limit)
	time.Sleep(time.Until(start.Add(limit + 4*time.Second)))
	write(t, laptop, "seed.md", "edited by the user\n")
	lock, _ := usersCommit(t, laptop)
	res := syncVault(t, laptop, "laptop", false)
	t.Logf("%v: %v", res.Outcome, res.Err)
	wantLock(t, lock, true)
	if res.Outcome == gitsync.Synced {
		t.Error("must not sync past the user's lock")
	}
}

// A marker from long ago (old limit), dead pid, user's live lock.
func TestSyncOldMarkerNeverDeletesAUsersLiveLock(t *testing.T) {
	for _, age := range []time.Duration{25 * time.Second, time.Hour, 30 * 24 * time.Hour} {
		t.Run(fmt.Sprint(age), func(t *testing.T) {
			_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
			write(t, laptop, "seed.md", "edited by the user\n")
			gitDir := git(t, laptop, "rev-parse", "--absolute-git-dir")
			if err := gitsync.WriteRunMarker(gitDir, deadPid(t), time.Now().Add(-age)); err != nil {
				t.Fatal(err)
			}
			lock, _ := usersCommit(t, laptop)
			syncVault(t, laptop, "laptop", false)
			wantLock(t, lock, true)
			if st := detect(t, laptop); st.Status != gitsync.Paused || st.Reason != gitsync.ReasonLocked {
				t.Errorf("after recovery declines, the vault must pause on the lock: %+v", st)
			}
		})
	}
}
