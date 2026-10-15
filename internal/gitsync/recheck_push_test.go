package gitsync_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// Another device pushes between desk's fetch and its push: retry, with the
// plain message and no git hint; the next sync joins both.
func TestRecheckPushedFirstStillRetries(t *testing.T) {
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, desk, "d.md", "desk\n")
	restore := gitsync.SetSyncHooks(nil, func() {
		write(t, laptop, "l.md", "laptop\n")
		git(t, laptop, "add", "-A")
		git(t, laptop, "commit", "-q", "-m", "laptop")
		git(t, laptop, "push", "-q", "origin", "main")
	})
	res := syncVault(t, desk, "desk", false)
	restore()
	wantOutcome(t, res, gitsync.SyncRetry)
	if msg := res.Err.Error(); msg != "another device pushed first; the next sync joins its changes" {
		t.Errorf("message %q", msg)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantAllVersions(t, desk, "desk\n", "laptop\n")
	if git(t, desk, "rev-parse", "HEAD") != remoteHead(t, remote, "main") {
		t.Error("desk and origin differ")
	}
}

// Quit (PushVault, no fetch) after another device pushed: not "rewritten".
func TestRecheckQuitAfterAnotherPushIsNotRewritten(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "l.md", "laptop\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "d.md", "desk\n")
	before := remoteHead(t, remote, "main")
	res := pushVault(t, context.Background(), desk, 10*time.Second)
	if res.Outcome != gitsync.SyncRetry || res.Err == nil || !strings.Contains(res.Err.Error(), "another device pushed first") || strings.Contains(res.Err.Error(), "git pull") {
		t.Errorf("got %v: %v", res.Outcome, res.Err)
	}
	if remoteHead(t, remote, "main") != before {
		t.Error("origin changed")
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantAllVersions(t, desk, "desk\n", "laptop\n")
}
