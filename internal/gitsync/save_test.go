package gitsync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// commitLocal runs CommitLocal on dir, holding a lock the way the app does.
func commitLocal(t *testing.T, dir string) gitsync.SyncResult {
	t.Helper()
	lock, ok, err := gitsync.TryLock(filepath.Join(t.TempDir(), "sync.lock"))
	if err != nil || !ok {
		t.Fatalf("TryLock: %v %v", ok, err)
	}
	defer lock.Unlock()
	return gitsync.CommitLocal(context.Background(), gitsync.New(dir, 20*time.Second), lock, gitsync.SyncOptions{Device: "desk"})
}

// Without the vault's lock no git runs: nothing is committed.
func TestCommitLocalNeedsTheLock(t *testing.T) {
	t.Parallel()
	_, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	before := git(t, desk, "rev-parse", "HEAD")
	write(t, desk, "d.md", "desk\n")
	res := gitsync.CommitLocal(context.Background(), gitsync.New(desk, 20*time.Second), nil, gitsync.SyncOptions{})
	if res.Outcome == gitsync.Synced || res.Err == nil {
		t.Fatalf("no lock, no commit: %+v", res)
	}
	if git(t, desk, "rev-parse", "HEAD") != before {
		t.Error("without the lock nothing may be committed")
	}
}

// The commit after a save is local (S3): the change is committed and
// counted as pending, and origin is neither asked nor changed. Here origin
// cannot even be reached, and what another device pushed is not fetched.
func TestCommitLocalCommitsWithoutTheNetwork(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "l.md", "laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	pushed := remoteHead(t, remote, "main")
	tracking := git(t, desk, "rev-parse", "refs/remotes/origin/main")
	git(t, desk, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "unreachable.git"))

	write(t, desk, "d.md", "desk\n")
	res := commitLocal(t, desk)
	wantOutcome(t, res, gitsync.Synced)
	if !res.Commit.Committed || res.Pushed || res.Updated || res.Pending != 1 {
		t.Errorf("want one local commit, pending, nothing pushed or applied: %+v", res)
	}
	if git(t, desk, "show", "HEAD:d.md") != "desk" {
		t.Error("the saved note must be committed")
	}
	if git(t, desk, "rev-parse", "refs/remotes/origin/main") != tracking {
		t.Error("nothing may be fetched")
	}
	if remoteHead(t, remote, "main") != pushed {
		t.Error("nothing may be pushed")
	}
	wantMissing(t, desk, "l.md")

	again := commitLocal(t, desk)
	wantOutcome(t, again, gitsync.Synced)
	if again.Commit.Committed || again.Pending != 1 {
		t.Errorf("with nothing new, no commit: %+v", again)
	}
}

// A git lock file Misoku did not leave stops the commit before anything is
// staged, as it stops a sync (S13).
func TestCommitLocalStopsOnSomeoneElsesIndexLock(t *testing.T) {
	t.Parallel()
	_, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	before := git(t, desk, "rev-parse", "HEAD")
	lock := filepath.Join(desk, ".git", "index.lock")
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	write(t, desk, "d.md", "desk\n")
	res := commitLocal(t, desk)
	var locked *gitsync.LockedError
	if res.Outcome == gitsync.Synced || !errors.As(res.Err, &locked) {
		t.Fatalf("want a stop on the lock file, got %+v", res)
	}
	if git(t, desk, "rev-parse", "HEAD") != before {
		t.Error("nothing may be committed past someone else's lock")
	}
	if _, err := os.Stat(lock); err != nil {
		t.Errorf("the lock must stay: %v", err)
	}
}

// The app names the copy it keeps on save as a sync names conflict copies
// (S6), so FindConflictCopies lists it; a taken name gets " 2".
func TestConflictCopyNameMatchesSyncCopies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	at := time.Date(2026, 10, 16, 9, 5, 0, 0, time.Local)
	name, err := gitsync.ConflictCopyName(dir, "notes/doc.md", "", at)
	if err != nil || name != "notes/doc (conflict 2026-10-16 0905).md" {
		t.Fatalf("got %q %v", name, err)
	}
	write(t, dir, name, "theirs")
	again, err := gitsync.ConflictCopyName(dir, "notes/doc.md", "desk", at)
	if err != nil || again != "notes/doc (conflict desk 2026-10-16 0905).md" {
		t.Fatalf("with a device: %q %v", again, err)
	}
	taken, err := gitsync.ConflictCopyName(dir, "notes/doc.md", "", at)
	if err != nil || taken != "notes/doc (conflict 2026-10-16 0905) 2.md" {
		t.Fatalf("a taken name gets a suffix: %q %v", taken, err)
	}
	copies, err := gitsync.FindConflictCopies(dir)
	if err != nil || len(copies) != 1 || copies[0] != name {
		t.Fatalf("FindConflictCopies must list it: %q %v", copies, err)
	}
}
