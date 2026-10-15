package gitsync_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// pushVault runs PushVault on dir within limit, holding a lock the way the
// app does.
func pushVault(t *testing.T, ctx context.Context, dir string, limit time.Duration) gitsync.SyncResult {
	t.Helper()
	lock, ok, err := gitsync.TryLock(filepath.Join(t.TempDir(), "sync.lock"))
	if err != nil || !ok {
		t.Fatalf("TryLock: %v %v", ok, err)
	}
	defer lock.Unlock()
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	return gitsync.PushVault(ctx, gitsync.New(dir, limit), lock, gitsync.SyncOptions{Device: "desk"})
}

// hangingSSH makes origin an ssh remote whose ssh never answers.
func hangingSSH(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake ssh is a shell script")
	}
	unsetenv(t, "GIT_SSH")
	unsetenv(t, "GIT_SSH_VARIANT")
	script := filepath.Join(t.TempDir(), "fake-ssh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_SSH_COMMAND", "sh '"+script+"'")
	git(t, dir, "remote", "set-url", "origin", "ssh://git@example.invalid/notes.git")
}

// Without the vault's lock no git runs: nothing is committed or pushed.
func TestPushVaultNeedsTheLock(t *testing.T) {
	t.Parallel()
	remote, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	before := git(t, desk, "rev-parse", "HEAD")
	write(t, desk, "d.md", "desk\n")
	res := gitsync.PushVault(context.Background(), gitsync.New(desk, 20*time.Second), nil, gitsync.SyncOptions{})
	if res.Outcome == gitsync.Synced || res.Err == nil {
		t.Fatalf("no lock, no push: %+v", res)
	}
	if git(t, desk, "rev-parse", "HEAD") != before || remoteHead(t, remote, "main") != before {
		t.Error("without the lock nothing may be committed or pushed")
	}
}

// S14: the vault's changes are committed and pushed.
func TestPushVaultCommitsAndPushes(t *testing.T) {
	t.Parallel()
	remote, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, desk, "d.md", "desk\n")
	res := pushVault(t, context.Background(), desk, 20*time.Second)
	wantOutcome(t, res, gitsync.Synced)
	if !res.Commit.Committed || !res.Pushed || res.Pending != 0 {
		t.Errorf("want the change committed and pushed, got %+v", res)
	}
	if got := git(t, remote, "show", "main:d.md"); got != "desk" {
		t.Errorf("origin holds d.md = %q", got)
	}
}

// Quitting never applies origin's changes to the files: it only commits
// and pushes. When origin moved on, the commit waits for the next sync.
func TestPushVaultNeverAppliesOriginsChanges(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "l.md", "laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	before := remoteHead(t, remote, "main")

	write(t, desk, "d.md", "desk\n")
	res := pushVault(t, context.Background(), desk, 20*time.Second)
	wantOutcome(t, res, gitsync.SyncRetry)
	wantMissing(t, desk, "l.md")
	if res.Pushed || res.Pending != 1 || !res.Commit.Committed {
		t.Errorf("want the desk's commit kept and pending, got %+v", res)
	}
	if git(t, desk, "show", "HEAD:d.md") != "desk" {
		t.Error("the desk's note must be committed")
	}
	if remoteHead(t, remote, "main") != before {
		t.Error("origin must be left as the laptop pushed it")
	}
}

// A remote that never answers is given up at the time limit: the commit
// stays for the next sync and git leaves no lock behind.
func TestPushVaultStopsAtTheTimeLimit(t *testing.T) {
	_, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	hangingSSH(t, desk)
	write(t, desk, "d.md", "desk\n")
	start := time.Now()
	res := pushVault(t, context.Background(), desk, time.Second)
	if took := time.Since(start); took > 3500*time.Millisecond {
		t.Errorf("a 1 s limit took %s", took)
	}
	if res.Outcome == gitsync.Synced || res.Pushed || res.Err == nil {
		t.Errorf("a hung remote is not pushed to: %+v", res)
	}
	if git(t, desk, "show", "HEAD:d.md") != "desk" {
		t.Error("the commit must stay for the next sync")
	}
	wantMissing(t, desk, ".git/index.lock")
}
