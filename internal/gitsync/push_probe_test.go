package gitsync_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// S13/S2 verifier probe: a user's own index.lock (a `git commit` open in a
// terminal) is never removed by the final push, and nothing is committed
// or pushed under it.
func TestPushVaultLeavesTheUsersIndexLock(t *testing.T) {
	t.Parallel()
	remote, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	before := git(t, desk, "rev-parse", "HEAD")
	write(t, desk, "d.md", "desk\n")
	lock := filepath.Join(desk, ".git", "index.lock")
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	res := pushVault(t, context.Background(), desk, 20*time.Second)
	if res.Outcome == gitsync.Synced || res.Err == nil {
		t.Fatalf("a user's index.lock must stop the push: %+v", res)
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatal("the user's index.lock must stay")
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if git(t, desk, "rev-parse", "HEAD") != before || remoteHead(t, remote, "main") != before {
		t.Error("nothing may be committed or pushed under the user's lock")
	}
}

// Before any sync set an upstream, PushVault asks origin for its default
// branch and pushes there, as SyncVault does.
func TestPushVaultWithoutUpstreamUsesOriginsDefault(t *testing.T) {
	t.Parallel()
	remote, _, desk := pair(t, map[string]string{"seed.md": "s\n"})
	git(t, desk, "branch", "--unset-upstream")
	write(t, desk, "d.md", "desk\n")
	res := pushVault(t, context.Background(), desk, 20*time.Second)
	if res.Outcome != gitsync.Synced || !res.Pushed || res.Branch != "main" {
		t.Fatalf("want a push to main: %+v", res)
	}
	if remoteHead(t, remote, "main") != git(t, desk, "rev-parse", "HEAD") {
		t.Fatal("origin's main must hold the new commit")
	}
}
