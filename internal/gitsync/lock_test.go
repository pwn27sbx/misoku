package gitsync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

func TestOnlyOneHolderAtATime(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sync.lock")
	first, ok, err := gitsync.TryLock(path)
	if err != nil || !ok {
		t.Fatalf("first lock: ok=%v err=%v", ok, err)
	}
	if _, ok, _ := gitsync.TryLock(path); ok {
		t.Fatal("a second Misoku must not get the lock while the first holds it")
	}
	first.Unlock()
	again, ok, err := gitsync.TryLock(path)
	if err != nil || !ok {
		t.Fatalf("the lock must be free after Unlock: ok=%v err=%v", ok, err)
	}
	again.Unlock()
}

// The holder is a real process that gets killed: the operating system must
// release its lock, with no stale file to clean up by hand.
func TestLockIsFreedWhenTheHolderDies(t *testing.T) {
	t.Parallel()
	if os.Getenv("MISOKU_LOCK_HOLDER") != "" {
		if _, ok, err := gitsync.TryLock(os.Getenv("MISOKU_LOCK_HOLDER")); err != nil || !ok {
			os.Exit(3)
		}
		os.Stdout.WriteString("locked\n")
		time.Sleep(time.Minute)
		return
	}
	path := filepath.Join(t.TempDir(), "sync.lock")
	child := exec.Command(os.Args[0], "-test.run=^TestLockIsFreedWhenTheHolderDies$")
	child.Env = append(os.Environ(), "MISOKU_LOCK_HOLDER="+path)
	out, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 7)
	if _, err := out.Read(buf); err != nil || string(buf) != "locked\n" {
		child.Process.Kill()
		t.Fatalf("holder did not take the lock: %q %v", buf, err)
	}
	if _, ok, _ := gitsync.TryLock(path); ok {
		child.Process.Kill()
		t.Fatal("the lock must be taken while the holder runs")
	}
	child.Process.Kill()
	child.Wait()
	deadline := time.Now().Add(3 * time.Second)
	for {
		l, ok, err := gitsync.TryLock(path)
		if ok {
			l.Unlock()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the lock was not released after the holder died (err=%v)", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// One lock per vault, kept inside its git folder: two windows on the same
// vault meet on the same file, and the lock never shows up as a note to
// commit.
func TestLockVaultIsOnePerVaultAndInsideGit(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := readyVault(t)
	r := gitsync.New(dir, 10*time.Second)
	first, ok, err := gitsync.LockVault(context.Background(), r)
	if err != nil || !ok {
		t.Fatalf("first window: ok=%v err=%v", ok, err)
	}
	defer first.Unlock()
	if _, ok, err := gitsync.LockVault(context.Background(), gitsync.New(dir, 10*time.Second)); ok || err != nil {
		t.Fatalf("a second window must see the vault locked: ok=%v err=%v", ok, err)
	}
	if changes, err := r.Status(context.Background()); err != nil || len(changes) != 0 {
		t.Fatalf("the lock must not be a change in the vault: %v %v", changes, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "misoku-sync.lock")); err != nil {
		t.Fatalf("the lock lives in the git folder: %v", err)
	}
}
