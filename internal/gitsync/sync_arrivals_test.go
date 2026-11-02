package gitsync_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

func wantArrivals(t *testing.T, res gitsync.SyncResult, wantNew, wantUpdated []string) {
	t.Helper()
	if !slices.Equal(res.NewNotes, wantNew) {
		t.Errorf("NewNotes = %q, want %q", res.NewNotes, wantNew)
	}
	if !slices.Equal(res.UpdatedNotes, wantUpdated) {
		t.Errorf("UpdatedNotes = %q, want %q", res.UpdatedNotes, wantUpdated)
	}
}

// S1, A1: what another computer pushed is named by its path in the vault,
// new notes apart from updated ones. Deleted notes and anything Misoku does
// not list as a note are not named.
func TestSyncNamesTheNotesThatArrived(t *testing.T) {
	t.Parallel()
	const body = "a note long enough for git to recognise it after a rename\n"
	for _, c := range []struct {
		name             string
		change           func(t *testing.T, laptop string)
		wantNew, wantUpd []string
	}{
		{name: "nothing arrived"},
		{
			name: "a new note",
			change: func(t *testing.T, d string) {
				write(t, d, "n.md", "new\n")
			},
			wantNew: []string{"n.md"},
		},
		{
			name: "a new note in a folder",
			change: func(t *testing.T, d string) {
				write(t, d, "ideas/my note.md", "new\n")
			},
			wantNew: []string{"ideas/my note.md"},
		},
		{
			name: "a name that is not ASCII",
			change: func(t *testing.T, d string) {
				write(t, d, "reunión.md", "new\n")
			},
			wantNew: []string{"reunión.md"},
		},
		{
			name: "an updated note",
			change: func(t *testing.T, d string) {
				write(t, d, "seed.md", "edited\n")
			},
			wantUpd: []string{"seed.md"},
		},
		{
			name: "both",
			change: func(t *testing.T, d string) {
				write(t, d, "a.md", "new\n")
				write(t, d, "b.md", "new\n")
				write(t, d, "seed.md", "edited\n")
			},
			wantNew: []string{"a.md", "b.md"},
			wantUpd: []string{"seed.md"},
		},
		{
			name: "a deleted note",
			change: func(t *testing.T, d string) {
				if err := os.Remove(filepath.Join(d, "gone.md")); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "files that are not notes",
			change: func(t *testing.T, d string) {
				write(t, d, "assets/pic.png", "png")
				write(t, d, "list.txt", "text\n")
				write(t, d, "keep.md.bak", "text\n")
			},
		},
		{
			name: "a note in a hidden folder",
			change: func(t *testing.T, d string) {
				write(t, d, ".obsidian/help.md", "not listed as a note\n")
			},
		},
		{
			name: "a renamed note is updated under its new name",
			change: func(t *testing.T, d string) {
				rename(t, d, "moved.md", "ideas/moved.md")
			},
			wantUpd: []string{"ideas/moved.md"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, laptop, desk := pair(t, map[string]string{"seed.md": "seed\n", "gone.md": "x\n", "moved.md": body})
			if c.change != nil {
				c.change(t, laptop)
				wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
			}
			res := syncVault(t, desk, "desk", false)
			wantOutcome(t, res, gitsync.Synced)
			wantArrivals(t, res, c.wantNew, c.wantUpd)
		})
	}
}

// What this computer wrote did not arrive from anywhere, also when the two
// sides are joined by a union commit.
func TestSyncNamesOnlyTheOtherComputersNotesWhenJoining(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"seed.md": "one\n\n\n\n\ntwo\n", "desk-only.md": "x\n"})
	write(t, laptop, "from-laptop.md", "new\n")
	write(t, laptop, "seed.md", "one edited\n\n\n\n\ntwo\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "from-desk.md", "new\n")
	write(t, desk, "desk-only.md", "edited\n")
	write(t, desk, "seed.md", "one\n\n\n\n\ntwo edited\n")

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if !res.Pushed {
		t.Fatalf("setup: want a union commit, got %+v", res)
	}
	wantArrivals(t, res, []string{"from-laptop.md"}, []string{"seed.md"})
}

// S3: a conflict copy is reported in Conflicts, so it is not an arrival,
// while what came with it still is.
func TestSyncDoesNotNameConflictCopiesAsArrivals(t *testing.T) {
	t.Parallel()
	_, laptop, desk := pair(t, map[string]string{"n.md": "one\n"})
	write(t, laptop, "n.md", "from laptop\n")
	write(t, laptop, "extra.md", "new\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "n.md", "from desk\n")

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if want := []string{"n (conflict laptop " + stamp + ").md"}; !slices.Equal(res.Conflicts, want) {
		t.Fatalf("setup: Conflicts = %q, want %q", res.Conflicts, want)
	}
	wantArrivals(t, res, []string{"extra.md"}, nil)
}

// A vault with no commit yet has no base to compare with: all of origin's
// notes are new to it.
func TestSyncNamesOriginsNotesOnAVaultWithNoCommit(t *testing.T) {
	t.Parallel()
	isolate(t)
	remote := newRemote(t)
	laptop, desk := clone(t, remote), clone(t, remote)
	write(t, laptop, "a.md", "new\n")
	write(t, laptop, "ideas/reunión.md", "new\n")
	write(t, laptop, "pic.png", "png")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	wantArrivals(t, res, []string{"a.md", "ideas/reunión.md"}, nil)
}

// The notes are in the vault even when the push after them fails.
func TestSyncNamesArrivalsWhenThePushAfterThemFails(t *testing.T) {
	t.Parallel()
	remote, laptop, desk := pair(t, map[string]string{"seed.md": "seed\n"})
	write(t, laptop, "n.md", "new\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	hook := filepath.Join(remote, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, desk, "mine.md", "mine\n")

	res := syncVault(t, desk, "desk", false)
	if res.Outcome == gitsync.Synced || !res.Updated {
		t.Fatalf("setup: want a failed push after applying origin's notes, got %+v", res)
	}
	wantArrivals(t, res, []string{"n.md"}, nil)
}

// Listing what arrived only informs: when that git command fails the notes
// are already in the vault, so the sync still succeeds.
func TestSyncSucceedsWhenListingTheArrivalsFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake git is a shell script, which Windows cannot run")
	}
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	_, laptop, desk := pair(t, map[string]string{"seed.md": "seed\n"})
	write(t, laptop, "n.md", "new\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)

	bin := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = diff-tree ] && exit 1; done\nexec %q \"$@\"\n", real)
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.Synced)
	if !res.Updated {
		t.Errorf("origin's changes must count as applied, got %+v", res)
	}
	wantFile(t, desk, "n.md", "new\n")
	wantArrivals(t, res, nil, nil)
}
