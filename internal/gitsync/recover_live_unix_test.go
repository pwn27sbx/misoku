//go:build !windows

package gitsync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// usersCommit starts, like the user in another terminal, `git commit -a`
// whose editor stays open: git holds the vault's index.lock all that time.
// It returns once the lock is there; the user's git is killed at cleanup.
func usersCommit(t *testing.T, dir string) (lock string, cmd *exec.Cmd) {
	t.Helper()
	editor := fakeProgram(t, t.TempDir(), "editor", "sleep 30\n")
	cmd = exec.Command("git", "-C", dir, "commit", "-a")
	cmd.Env = append(os.Environ(), "GIT_EDITOR="+editor)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	lock = filepath.Join(dir, ".git", "index.lock")
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if _, err := os.Lstat(lock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the user's git never took index.lock")
		}
	}
	if syscall.Kill(cmd.Process.Pid, 0) != nil {
		t.Fatal("the user's git must be running")
	}
	return lock, cmd
}

// a marker left by a Misoku that died (its git then exited by
// itself, so its pid is dead) must not make Misoku delete a lock that the
// user's own, live git holds now.
func TestSyncStaleMarkerNeverDeletesALiveUsersLock(t *testing.T) {
	_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "seed.md", "edited by the user\n")
	gitDir := git(t, laptop, "rev-parse", "--absolute-git-dir")
	if err := gitsync.WriteRunMarker(gitDir, deadPid(t), time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	lock, _ := usersCommit(t, laptop)

	syncVault(t, laptop, "laptop", false)
	wantLock(t, lock, true)
}

// The usual origin of a stale marker: Misoku is killed while git waits on
// the network, and git (in its own session) exits by itself later. A
// network command never takes the index lock, so it must leave no marker,
// and a lock the user's live git takes afterwards stays.
func TestSyncMisokuDyingDuringFetchLeavesNoMarker(t *testing.T) {
	if dir := os.Getenv("MISOKU_SYNC_FETCH_CHILD"); dir != "" {
		syncVault(t, dir, "laptop", false)
		return
	}
	_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	flag := filepath.Join(t.TempDir(), "ssh-started")
	ssh := fakeProgram(t, t.TempDir(), "ssh", "touch '"+flag+"'\nsleep 2\ntouch '"+flag+".done'\nexit 255\n")
	unsetenv(t, "GIT_SSH")
	t.Setenv("GIT_SSH_COMMAND", "sh '"+ssh+"'")
	git(t, laptop, "remote", "set-url", "origin", "ssh://git@example.invalid/notes.git")

	child := exec.Command(os.Args[0], "-test.run=^TestSyncMisokuDyingDuringFetchLeavesNoMarker$")
	child.Env = append(os.Environ(), "MISOKU_SYNC_FETCH_CHILD="+laptop)
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if _, err := os.Lstat(flag); err == nil {
			break
		}
		if time.Now().After(deadline) {
			child.Process.Kill()
			t.Fatal("ssh never started")
		}
	}
	child.Process.Kill() // Misoku dies
	child.Wait()
	for deadline := time.Now().Add(20 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		if _, err := os.Lstat(flag + ".done"); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the orphaned ssh never ended")
		}
	}
	time.Sleep(300 * time.Millisecond) // its git ends by itself

	marker := filepath.Join(laptop, ".git", "misoku-sync.running")
	if _, err := os.Lstat(marker); err == nil {
		t.Fatal("a network command must leave no marker behind")
	}
	write(t, laptop, "seed.md", "edited by the user\n")
	lock, _ := usersCommit(t, laptop)
	res := syncVault(t, laptop, "laptop", false)
	t.Logf("outcome %v: %v", res.Outcome, res.Err)
	wantLock(t, lock, true)
}

// a Misoku git command killed at its time limit while it
// waits on the network must not remove a lock the user's live git took
// meanwhile (after that command started).
func TestSyncTimeoutNeverDeletesALockTheUserTookMeanwhile(t *testing.T) {
	_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	flag := filepath.Join(t.TempDir(), "ssh-started")
	ssh := fakeProgram(t, t.TempDir(), "ssh", "touch '"+flag+"'\nsleep 30\n")
	unsetenv(t, "GIT_SSH")
	t.Setenv("GIT_SSH_COMMAND", "sh '"+ssh+"'")
	git(t, laptop, "remote", "set-url", "origin", "ssh://git@example.invalid/notes.git")

	lockc := make(chan string, 1)
	go func() {
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			if _, err := os.Lstat(flag); err == nil {
				break
			}
		}
		write(t, laptop, "seed.md", "edited by the user\n")
		l, _ := usersCommit(t, laptop)
		lockc <- l
	}()
	held, ok, err := gitsync.TryLock(filepath.Join(t.TempDir(), "sync.lock"))
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	defer held.Unlock()
	res := gitsync.SyncVault(context.Background(), gitsync.New(laptop, 4*time.Second), held, gitsync.SyncOptions{Device: "laptop"})
	t.Logf("outcome %v: %v", res.Outcome, res.Err)
	wantLock(t, <-lockc, true)
}
