package gitsync_test

// Regression tests for P4 (a vault inside another repository). They use
// the helpers of setup_nested_test.go (nestedVault, outerState, gitIn,
// hasOwnGit, stepText, ownRepoQuestion) and newSetup/isolate/newRepo.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// vaultIn makes outer/rel with a note and, when tracked is not "", has
// the outer repository commit that path (relative to outer) first.
func vaultIn(t *testing.T, rel, tracked string) (outer, dir string) {
	t.Helper()
	outer = newRepo(t)
	if err := os.WriteFile(filepath.Join(outer, "readme.txt"), []byte("outer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, outer, "add", "readme.txt")
	if tracked != "" {
		p := filepath.Join(outer, filepath.FromSlash(tracked))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("tracked\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitIn(t, outer, "add", "--", ":(literal)"+tracked)
	}
	gitIn(t, outer, "commit", "-q", "-m", "outer")
	dir = filepath.Join(outer, filepath.FromSlash(rel))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return outer, dir
}

// The tracked check takes the folder's name literally and on whole path
// components: glob characters, spaces, unicode, a leading dash, deeper
// folders, and siblings whose names start the same.
func TestTrackedCheckIsLiteralAndWholeComponent(t *testing.T) {
	t.Parallel()
	isolate(t)
	for _, c := range []struct {
		rel, tracked string
		offer        bool
	}{
		{"My Notes", "", true},
		{"My Notes", "My Notes/b.md", false},
		{"Notas ñ 日本", "Notas ñ 日本/b.md", false},
		{"-Notes", "", true},
		{"-Notes", "-Notes/b.md", false},
		{"N*", "Nx/b.md", true},
		{"N*", "N*/b.md", false},
		{"[ab]", "a/b.md", true},
		{"[ab]", "[ab]/b.md", false},
		{"Notes", "Notes2/b.md", true},
		{"Notes", "Notes.md", true},
		{"a/b/Notes", "", true},
		{"a/b/Notes", "a/b/Notes/deep/x.md", false},
		{"a/b/Notes", "a/b/x.md", true},
	} {
		outer, dir := vaultIn(t, c.rel, c.tracked)
		before := outerState(t, outer)
		step := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json")).Begin(context.Background())
		if c.offer {
			if step.Ask != gitsync.AskOwnRepo || step.Confirm != ownRepoQuestion {
				t.Errorf("%q tracked %q: want the offer, got %+v", c.rel, c.tracked, step)
			}
		} else if step.Outcome != gitsync.SetupFailed || !strings.Contains(stepText(step), "two repositories") {
			t.Errorf("%q tracked %q: want the two-repositories stop, got %+v", c.rel, c.tracked, step)
		}
		if hasOwnGit(t, dir) || outerState(t, outer) != before {
			t.Errorf("%q: Begin must change nothing", c.rel)
		}
	}
}

// An outer .gitignore that ignores the folder: it tracks nothing there,
// so the offer comes, and Enter leaves the outer repository as it was.
func TestIgnoredFolderIsOfferedAndOuterUnchanged(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, dir := vaultIn(t, "Notes", ".gitignore")
	if err := os.WriteFile(filepath.Join(outer, ".gitignore"), []byte("Notes/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, outer, "commit", "-q", "-am", "ignore")
	before := outerState(t, outer)
	ctx := context.Background()
	s := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json"))
	if step := s.Begin(ctx); step.Ask != gitsync.AskOwnRepo {
		t.Fatalf("begin: %+v", step)
	}
	if step := s.Next(ctx, gitsync.SetupAnswer{OK: true}); step.Ask != gitsync.AskStart {
		t.Fatalf("enter: %+v", step)
	}
	if outerState(t, outer) != before {
		t.Error("the outer repository changed")
	}
}

// A .git the run created but that became a symbolic link (to an empty
// repository) or a gitfile before the cancel is not the run's to delete.
func TestCancelLeavesADotGitThatIsNoLongerTheFolderItMade(t *testing.T) {
	t.Parallel()
	isolate(t)
	for _, kind := range []string{"symlink", "gitfile"} {
		_, dir := nestedVault(t)
		ctx := context.Background()
		s := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json"))
		s.Begin(ctx)
		if step := s.Next(ctx, gitsync.SetupAnswer{OK: true}); step.Ask != gitsync.AskStart {
			t.Fatalf("%s: enter: %+v", kind, step)
		}
		gitDir := filepath.Join(dir, ".git")
		other := filepath.Join(newRepo(t), ".git") // empty: no commits either
		if err := os.RemoveAll(gitDir); err != nil {
			t.Fatal(err)
		}
		switch kind {
		case "symlink":
			if err := os.Symlink(other, gitDir); err != nil {
				t.Skipf("symlink: %v", err)
			}
		case "gitfile":
			if err := os.WriteFile(gitDir, []byte("gitdir: "+other+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		step := s.Next(ctx, gitsync.SetupAnswer{})
		if step.Outcome != gitsync.SetupCancelled || !strings.Contains(stepText(step), "left in place") {
			t.Errorf("%s: cancel must say it was left: %+v", kind, step)
		}
		if _, err := os.Lstat(gitDir); err != nil {
			t.Errorf("%s: the .git that replaced the run's one was removed: %v", kind, err)
		}
		if _, err := os.Stat(filepath.Join(other, "HEAD")); err != nil {
			t.Errorf("%s: the repository it points to was touched: %v", kind, err)
		}
	}
}

// On a case-insensitive file system (core.ignorecase, macOS and Windows
// by default) the outer repository may track the folder under another
// case, e.g. after renaming notes to Notes. The folder is still tracked
// there (outer `git status` sees notes/b.md as clean), so no repository
// of its own may be created.
func TestTrackedUnderAnotherCaseStops(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, _ := vaultIn(t, "x", "notes/b.md")
	if gitIn(t, outer, "config", "--bool", "core.ignorecase") != "true" {
		t.Skip("case-sensitive file system")
	}
	if err := os.Rename(filepath.Join(outer, "notes"), filepath.Join(outer, "tmp-rename")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(outer, "tmp-rename"), filepath.Join(outer, "Notes")); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(outer, "Notes")
	if gitIn(t, outer, "status", "--porcelain", "--", "notes/b.md") != "" {
		t.Fatal("setup: the outer repository must still see notes/b.md as tracked and clean")
	}
	before := outerState(t, outer)
	ctx := context.Background()
	s := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json"))
	step := s.Begin(ctx)
	if step.Ask == gitsync.AskOwnRepo {
		step = s.Next(ctx, gitsync.SetupAnswer{OK: true})
	}
	if hasOwnGit(t, dir) {
		t.Errorf("a repository of its own was created in a folder the outer one tracks (as notes/): %+v", step)
	}
	if step.Outcome != gitsync.SetupFailed || !strings.Contains(stepText(step), "two repositories") {
		t.Errorf("want the two-repositories stop, got %+v", step)
	}
	if outerState(t, outer) != before {
		t.Error("the outer repository changed")
	}
}

// A folder that is a submodule of the outer repository under another
// case (core.ignorecase) is still that submodule.
func TestSubmoduleUnderAnotherCaseStops(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, _ := vaultIn(t, "x", "")
	if gitIn(t, outer, "config", "--bool", "core.ignorecase") != "true" {
		t.Skip("case-sensitive file system")
	}
	gitIn(t, outer, "update-index", "--add", "--cacheinfo", "160000,"+gitIn(t, outer, "rev-parse", "HEAD")+",notes")
	dir := filepath.Join(outer, "Notes")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	step := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json")).Begin(context.Background())
	if step.Outcome != gitsync.SetupFailed || !strings.Contains(stepText(step), "submodule") {
		t.Errorf("want the submodule stop, got %+v", step)
	}
	if hasOwnGit(t, dir) {
		t.Error("a repository of its own was created in a submodule's folder")
	}
}

// The outer repository's folder is shown as the vault's path is written,
// not as git resolves it (on macOS the temporary folders are under /var,
// a link to /private/var), so the messages read alike.
func TestOuterFolderIsShownAsTheVaultPathIsWritten(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, dir := nestedVault(t)
	step := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json")).Begin(context.Background())
	if !strings.Contains(stepText(step), "repository at "+outer+",") {
		t.Errorf("the offer must name %s:\n%s", outer, stepText(step))
	}
	gitIn(t, outer, "add", "notes/a.md")
	step = newSetup(t, dir, filepath.Join(t.TempDir(), "config.json")).Begin(context.Background())
	if !strings.Contains(stepText(step), "repository at "+outer+" ") || !strings.Contains(stepText(step), "outside "+outer+",") {
		t.Errorf("the stop must name %s:\n%s", outer, stepText(step))
	}
}
