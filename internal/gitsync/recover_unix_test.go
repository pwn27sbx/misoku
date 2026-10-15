//go:build !windows

package gitsync_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// slowWhileLocked makes git hang while it holds the vault's index.lock: an
// fsmonitor hook (git asks it while refreshing the index) that writes
// git's pid to pidFile and sleeps whenever index.lock exists.
func slowWhileLocked(t *testing.T, dir string) (pidFile string) {
	t.Helper()
	pidFile = filepath.Join(t.TempDir(), "git.pid")
	hook := fakeProgram(t, t.TempDir(), "fsmonitor",
		`[ -e .git/index.lock ] || exit 0`+"\n"+`echo $PPID > "`+pidFile+`"`+"\n"+"sleep 30\n")
	git(t, dir, "config", "core.fsmonitor", hook)
	return pidFile
}

// R4 (F2.4): a command killed at its time limit while it holds index.lock
// must not leave the lock behind, or every later sync fails on it.
func TestTimeoutLeavesNoIndexLockBehind(t *testing.T) {
	remote, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	slowWhileLocked(t, laptop)
	write(t, laptop, "a.md", "a\n")

	lock, ok, err := gitsync.TryLock(filepath.Join(t.TempDir(), "sync.lock"))
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	defer lock.Unlock()
	res := gitsync.SyncVault(context.Background(), gitsync.New(laptop, 2*time.Second), lock, gitsync.SyncOptions{Device: "laptop"})
	if res.Outcome != gitsync.SyncFailed || !errors.Is(res.Err, context.DeadlineExceeded) {
		t.Fatalf("want a timeout, got %v: %v", res.Outcome, res.Err)
	}
	wantLock(t, filepath.Join(laptop, ".git", "index.lock"), false)

	git(t, laptop, "config", "--unset", "core.fsmonitor")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantInSync(t, remote, laptop, "main")
}

// S13 end to end: Misoku is killed mid-sync while its git holds index.lock.
// While that git runs its lock is left alone. Once git is gone too, the
// lock still is not removed on its own (on macOS and Linux it may be the
// user's by then): the sync stops on it, and once the user confirms its
// removal the next sync finishes the work.
func TestSyncRecoversAfterMisokuDiesMidSync(t *testing.T) {
	if dir := os.Getenv("MISOKU_SYNC_CHILD"); dir != "" {
		syncVault(t, dir, "laptop", false) // killed while it hangs
		return
	}
	remote, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	pidFile := slowWhileLocked(t, laptop)
	write(t, laptop, "a.md", "a\n")

	child := exec.Command(os.Args[0], "-test.run=^TestSyncRecoversAfterMisokuDiesMidSync$")
	child.Env = append(os.Environ(), "MISOKU_SYNC_CHILD="+laptop)
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	pid := 0
	for deadline := time.Now().Add(20 * time.Second); pid == 0; time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			child.Process.Kill()
			t.Fatal("the child's git never took the index lock")
		}
		if b, err := os.ReadFile(pidFile); err == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		}
	}
	t.Cleanup(func() { syscall.Kill(-pid, syscall.SIGKILL) })
	child.Process.Kill()
	child.Wait()

	lock := filepath.Join(laptop, ".git", "index.lock")
	git(t, laptop, "config", "--unset", "core.fsmonitor")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.SyncRetry)
	wantLock(t, lock, true) // its git still runs

	syscall.Kill(-pid, syscall.SIGKILL) // git's session: git and its hook
	for deadline := time.Now().Add(5 * time.Second); syscall.Kill(pid, 0) == nil; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("git did not die")
		}
	}
	wantLock(t, lock, true)
	res := syncVault(t, laptop, "laptop", false)
	var locked *gitsync.LockedError
	if !errors.As(res.Err, &locked) {
		t.Fatalf("want the sync stopped on the lock, got %v: %v", res.Outcome, res.Err)
	}
	wantLock(t, lock, true)
	shown, _, err := gitsync.StatLock(locked.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := gitsync.RemoveLeftoverLock(shown); err != nil {
		t.Fatal(err)
	}
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantInSync(t, remote, laptop, "main")
	wantFile(t, laptop, "a.md", "a\n")
}

// R4 (F2.5): when git cannot count what is left to push, the sync must not
// claim the vault is synced.
func TestSyncIsNotSyncedWhenThePendingCountFails(t *testing.T) {
	remote, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "a.md", "a\n")
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	fakeProgram(t, bin, "git", `case "$*" in
*"rev-list --count refs/remotes/origin/main..HEAD"*) echo "fatal: cannot count" >&2; exit 128 ;;
esac
exec "`+real+`" "$@"`+"\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := syncVault(t, laptop, "laptop", false)
	wantOutcome(t, res, gitsync.SyncFailed)
	if !strings.Contains(res.Err.Error(), "cannot count") {
		t.Errorf("the error must be git's: %v", res.Err)
	}
	if onRemote := git(t, remote, "ls-tree", "--name-only", "main"); strings.Contains(onRemote, "a.md") {
		t.Errorf("nothing was meant to be pushed, remote has:\n%s", onRemote)
	}
}

// An origin whose answer to ls-remote makes no sense (HEAD with no branch
// named) was reached: that is an error, not "offline".
func TestSyncUnexpectedAnswerFromOriginIsAnError(t *testing.T) {
	_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	fakeProgram(t, bin, "git", `case "$*" in
*"ls-remote --symref origin HEAD"*) printf '0123456789012345678901234567890123456789\tHEAD\n'; exit 0 ;;
esac
exec "`+real+`" "$@"`+"\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := syncVault(t, laptop, "laptop", false)
	wantOutcome(t, res, gitsync.SyncFailed)
	if !strings.Contains(res.Err.Error(), "did not say which branch") {
		t.Errorf("the error must say what origin answered: %v", res.Err)
	}
}

// Runner.Status never holds index.lock (it is not marked, so a lock left
// after it timed out could be anyone's). A status that takes it rewrites
// the index through it when a file's stat data is stale; this one must
// leave the index file as it was.
func TestStatusNeverTakesTheIndexLock(t *testing.T) {
	_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(laptop, "seed.md"), later, later); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(laptop, ".git", "index")
	before, err := os.Stat(index)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gitsync.New(laptop, 10*time.Second).Status(context.Background()); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(index)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("status rewrote the index, so it took the index lock")
	}
}
