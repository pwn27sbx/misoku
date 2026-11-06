package gitsync_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// historyRepo is a repository whose vault is its "notes" subfolder: the
// note was committed as a.md, renamed to "b c.md" with git mv, then edited.
func historyRepo(t *testing.T) (top, vault string) {
	t.Helper()
	isolate(t)
	top = newRepo(t)
	vault = filepath.Join(top, "notes")
	first := "line one\nline two\nline three\nline four\n"
	write(t, vault, "a.md", first)
	git(t, top, "add", "-A")
	git(t, top, "commit", "-q", "-m", "first version")
	git(t, top, "mv", "notes/a.md", "notes/b c.md")
	git(t, top, "commit", "-q", "-m", "rename")
	write(t, vault, "b c.md", first+"line five\n")
	git(t, top, "commit", "-q", "-a", "-m", "add five")
	return top, vault
}

// S9: the history follows the note across a rename, newest first, and
// every version can be read back at the path it had in that commit.
func TestFileHistoryFollowsARenameAndReadsEachVersion(t *testing.T) {
	_, vault := historyRepo(t)
	r := gitsync.New(vault, 10*time.Second)
	ctx := context.Background()
	got, err := r.FileHistory(ctx, "b c.md")
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 versions, got %+v", got)
	}
	wantSubjects := []string{"add five", "rename", "first version"}
	wantPaths := []string{"notes/b c.md", "notes/b c.md", "notes/a.md"}
	for i, v := range got {
		if v.Subject != wantSubjects[i] || v.Path != wantPaths[i] || len(v.Hash) < 40 || len(v.Date) != 10 {
			t.Errorf("version %d = %+v, want subject %q at %q with a full hash and a YYYY-MM-DD date", i, v, wantSubjects[i], wantPaths[i])
		}
	}
	text, err := r.FileAt(ctx, got[2].Hash, got[2].Path)
	if err != nil {
		t.Fatalf("FileAt: %v", err)
	}
	if string(text) != "line one\nline two\nline three\nline four\n" {
		t.Errorf("the version before the rename = %q", text)
	}
	text, err = r.FileAt(ctx, got[0].Hash, got[0].Path)
	if err != nil || string(text) != "line one\nline two\nline three\nline four\nline five\n" {
		t.Errorf("the newest version = %q, %v", text, err)
	}
}

// S9: a note git never saw has no history, and that is not an error.
func TestFileHistoryOfAnUntrackedNoteIsEmpty(t *testing.T) {
	_, vault := historyRepo(t)
	write(t, vault, "new.md", "fresh\n")
	got, err := gitsync.New(vault, 10*time.Second).FileHistory(context.Background(), "new.md")
	if err != nil || len(got) != 0 {
		t.Fatalf("untracked note: %+v, %v; want no versions and no error", got, err)
	}
}

// S9: outside a repository the error says so (ErrNotARepo).
func TestFileHistoryOutsideARepository(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	write(t, dir, "a.md", "x\n")
	_, err := gitsync.New(dir, 10*time.Second).FileHistory(context.Background(), "a.md")
	if !errors.Is(err, gitsync.ErrNotARepo) {
		t.Fatalf("err = %v, want ErrNotARepo", err)
	}
}

// S9: without git the error says so (ErrNoGit).
func TestFileHistoryWithoutGit(t *testing.T) {
	_, vault := historyRepo(t)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ProgramFiles", t.TempDir()) // Windows also looks there
	_, err := gitsync.New(vault, 10*time.Second).FileHistory(context.Background(), "b c.md")
	if !errors.Is(err, gitsync.ErrNoGit) {
		t.Fatalf("err = %v, want ErrNoGit", err)
	}
}
