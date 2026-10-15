package gitsync_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

func mv(t *testing.T, dir, from, to string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, filepath.FromSlash(to))), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(dir, filepath.FromSlash(from)), filepath.Join(dir, filepath.FromSlash(to))); err != nil {
		t.Fatal(err)
	}
}

func exists(dir, name string) bool {
	_, err := os.Lstat(filepath.Join(dir, filepath.FromSlash(name)))
	return err == nil
}

// Rename to an invalid name and edit it in the same sync window: desk keeps
// the old note (old content), laptop keeps the edited file.
func TestSyncRenameToAWindowsNameAndEditKeepsTheOldNoteElsewhere(t *testing.T) {
	t.Parallel()
	skipColonOnWindows(t)
	_, laptop, desk := pair(t, map[string]string{"Meeting.md": "agenda\n"})
	mv(t, laptop, "Meeting.md", "Meeting 10:30.md")
	write(t, laptop, "Meeting 10:30.md", "agenda\nedited\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "Meeting.md", "agenda\n")
	wantFile(t, laptop, "Meeting 10:30.md", "agenda\nedited\n")
	if exists(laptop, "Meeting.md") {
		t.Errorf("laptop got the old name back on disk")
	}
}

// Folder with many files renamed to an invalid name: nothing deleted on desk.
func TestSyncFolderOfManyNotesRenamedToAWindowsNameDeletesNothing(t *testing.T) {
	t.Parallel()
	seed := map[string]string{}
	for i := 0; i < 60; i++ {
		seed[fmt.Sprintf("Ideas/sub%d/n%02d.md", i%5, i)] = fmt.Sprintf("n%d\n", i)
	}
	_, laptop, desk := pair(t, seed)
	mv(t, laptop, "Ideas", "Ideas.")
	for i := 0; i < 2; i++ {
		res := syncVault(t, laptop, "laptop", false)
		wantOutcome(t, res, gitsync.Synced)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	for name, content := range seed {
		wantFile(t, desk, name, content)
		rel, _ := filepath.Rel("Ideas", name)
		wantFile(t, laptop, "Ideas./"+filepath.ToSlash(rel), content)
	}
}

// Rename to invalid, sync, then rename to valid: desk ends with only the
// valid name, content intact, nothing duplicated.
func TestSyncHeldRenameGoesThroughOnceTheNameIsValid(t *testing.T) {
	t.Parallel()
	skipColonOnWindows(t)
	_, laptop, desk := pair(t, map[string]string{"Meeting.md": "agenda\n", "keep.md": "k\n"})
	mv(t, laptop, "Meeting.md", "Meeting 10:30.md")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	mv(t, laptop, "Meeting 10:30.md", "Meeting 10-30.md")
	res := syncVault(t, laptop, "laptop", false)
	wantOutcome(t, res, gitsync.Synced)
	if len(res.Commit.Notices) != 0 {
		t.Errorf("notices after the valid rename: %+v", res.Commit.Notices)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantTree(t, desk, "Meeting 10-30.md", "keep.md")
	wantFile(t, desk, "Meeting 10-30.md", "agenda\n")
	wantTree(t, laptop, "Meeting 10-30.md", "keep.md")
	if exists(laptop, "Meeting.md") || exists(desk, "Meeting.md") {
		t.Errorf("old name still on disk")
	}
}

// An unrelated note deleted while an invalid name is pending: held, then
// the deletion goes through once the name is valid.
func TestSyncUnrelatedDeletionWaitsForTheNameToBeValid(t *testing.T) {
	t.Parallel()
	skipColonOnWindows(t)
	_, laptop, desk := pair(t, map[string]string{"old.md": "old\n", "n.md": "n\n"})
	write(t, laptop, "a:b.md", "new\n")
	os.Remove(filepath.Join(laptop, "old.md"))
	res := syncVault(t, laptop, "laptop", false)
	wantOutcome(t, res, gitsync.Synced)
	held := false
	for _, n := range res.Commit.Notices {
		held = held || (n.Kind == gitsync.HeldDeletion && n.Path == "old.md")
	}
	if !held {
		t.Errorf("notices %+v, want old.md held", res.Commit.Notices)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "old.md", "old\n")
	mv(t, laptop, "a:b.md", "a-b.md")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	if exists(desk, "old.md") {
		t.Errorf("the held deletion never reached desk")
	}
	wantFile(t, desk, "a-b.md", "new\n")
	wantTree(t, laptop, "a-b.md", "n.md")
}

// A held deletion while desk edits the old note: no data lost, and the
// sync does not stay stuck.
func TestSyncHeldDeletionKeepsAnotherDevicesEdit(t *testing.T) {
	t.Parallel()
	skipColonOnWindows(t)
	_, laptop, desk := pair(t, map[string]string{"Meeting.md": "agenda\n"})
	mv(t, laptop, "Meeting.md", "Meeting 10:30.md")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	write(t, desk, "Meeting.md", "agenda\ndesk edit\n")
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	var last gitsync.SyncResult
	for i := 0; i < 3; i++ {
		last = syncVault(t, laptop, "laptop", false)
	}
	wantFile(t, laptop, "Meeting 10:30.md", "agenda\n")
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "Meeting.md", "agenda\ndesk edit\n")
	if last.Outcome != gitsync.Synced {
		t.Errorf("laptop stays stuck while the deletion is held: %v %v", last.Outcome, last.Err)
	}
	// Release: rename to valid; nothing lost anywhere.
	mv(t, laptop, "Meeting 10:30.md", "Meeting 10-30.md")
	for i := 0; i < 2; i++ {
		wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	found := false
	for _, p := range tree(t, desk) {
		if c := read(t, desk, p); c == "agenda\ndesk edit\n" {
			found = true
		}
	}
	if !found {
		t.Errorf("desk's edit was lost")
	}
}
