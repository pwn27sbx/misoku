package gitsync_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// S6/S9: a note on one device and a folder of the same name on another
// keep both versions (they do today), and the version moved aside should
// be a conflict copy the user is told about: named like one and listed by
// :sync status. ADVISORY probe: today git's own "name~<commit>" is kept,
// which FindConflictCopies does not list.
func TestFinalFileFolderClashIsAListedConflictCopy(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "ideas/a.md", "in folder\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "ideas", "a file\n")
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	wantAllVersions(t, desk, "in folder\n", "a file\n")
	copies, err := gitsync.FindConflictCopies(desk)
	if err != nil {
		t.Fatal(err)
	}
	if len(copies) == 0 || len(res.Conflicts) == 0 {
		t.Errorf("the version moved aside is not a listed conflict copy: res %v, on disk %v, files %v", res.Conflicts, copies, tree(t, desk))
	}
}

// S2/S14: after origin was rewritten, quitting must not tell the user the
// next sync joins it (the next sync stops with "rewritten"). ADVISORY probe.
func TestFinalQuitAfterRewrittenOriginSaysSo(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "s\n"})
	write(t, laptop, "rw.md", "rewritten\n")
	git(t, laptop, "add", "-A")
	git(t, laptop, "commit", "-q", "--amend", "-m", "rewrite")
	git(t, laptop, "push", "-q", "-f", "origin", "main")
	write(t, desk, "d.md", "desk\n")
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.SyncRewritten)
	before := remoteHead(t, remote, "main")
	res := pushVault(t, context.Background(), desk, 10*time.Second)
	if remoteHead(t, remote, "main") != before {
		t.Fatal("quit must never overwrite a rewritten origin")
	}
	if res.Err != nil && strings.Contains(res.Err.Error(), "next sync joins") {
		t.Errorf("quit message promises a join that will not happen: %v", res.Err)
	}
}
