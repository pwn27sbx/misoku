package gitsync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// ownRepoQuestion is what the assistant asks for a folder inside another
// repository.
const ownRepoQuestion = "This folder is inside another git repository. Create a separate one for your notes?"

// gitIn runs git in dir and returns its trimmed output.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// nestedVault is a notes folder with one note inside an outer repository
// that has a commit of its own and does not track the folder.
func nestedVault(t *testing.T) (outer, dir string) {
	t.Helper()
	outer = newRepo(t)
	if err := os.WriteFile(filepath.Join(outer, "readme.txt"), []byte("outer\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, outer, "add", "readme.txt")
	gitIn(t, outer, "commit", "-q", "-m", "outer")
	dir = filepath.Join(outer, "notes")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return outer, dir
}

// outerState is what the outer repository must keep: its status, its
// commit and what it tracks.
func outerState(t *testing.T, outer string) string {
	t.Helper()
	return gitIn(t, outer, "status", "--porcelain") + "|" + gitIn(t, outer, "rev-parse", "HEAD") + "|" + gitIn(t, outer, "ls-files", "--stage")
}

// hasOwnGit reports whether dir has a .git of its own.
func hasOwnGit(t *testing.T, dir string) bool {
	t.Helper()
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return err == nil
}

func stepText(step gitsync.SetupStep) string {
	return strings.Join(append(append([]string{}, step.Lines...), step.Confirm), "\n")
}

func TestSetupInsideAnotherRepositoryOffersOneOfItsOwn(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, dir := nestedVault(t)
	before := outerState(t, outer)
	step := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json")).Begin(context.Background())
	if step.Ask != gitsync.AskOwnRepo || step.Confirm != ownRepoQuestion {
		t.Fatalf("a folder inside another repository must be offered one of its own: %+v", step)
	}
	if hasOwnGit(t, dir) {
		t.Error("asking must not create anything")
	}
	if outerState(t, outer) != before {
		t.Error("asking changed the outer repository")
	}
}

func TestSetupCreatesARepositoryOfItsOwnAndGoesOn(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, dir := nestedVault(t)
	before := outerState(t, outer)
	ctx := context.Background()
	s := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json"))
	s.Begin(ctx)
	step := s.Next(ctx, gitsync.SetupAnswer{OK: true})
	if step.Ask != gitsync.AskStart {
		t.Fatalf("after creating the repository the assistant must go on as usual: %+v", step)
	}
	if !strings.Contains(strings.Join(step.Notes, "\n"), "repository of its own") {
		t.Errorf("the progress must say the repository was created: %v", step.Notes)
	}
	if top := gitIn(t, dir, "rev-parse", "--show-toplevel"); !sameFolder(t, top, dir) {
		t.Errorf("the notes folder must be a repository of its own, its top is %s", top)
	}
	if outerState(t, outer) != before {
		t.Errorf("the outer repository changed:\nbefore %s\nafter  %s", before, outerState(t, outer))
	}
}

func sameFolder(t *testing.T, a, b string) bool {
	t.Helper()
	ia, err := os.Stat(a)
	if err != nil {
		t.Fatal(err)
	}
	ib, err := os.Stat(b)
	if err != nil {
		t.Fatal(err)
	}
	return os.SameFile(ia, ib)
}

func TestSetupStopsWhenTheOuterRepositoryTracksTheFolder(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, dir := nestedVault(t)
	gitIn(t, outer, "add", "notes/a.md")
	gitIn(t, outer, "commit", "-q", "-m", "notes")
	before := outerState(t, outer)
	step := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json")).Begin(context.Background())
	text := stepText(step)
	if step.Outcome != gitsync.SetupFailed || !strings.Contains(text, "notes/a.md") || !strings.Contains(text, "two repositories") || !strings.Contains(text, "changed nothing") {
		t.Fatalf("a folder the outer repository tracks must stop with why: %+v", step)
	}
	if strings.Contains(text, "git init") {
		t.Errorf("git init is not the fix here:\n%s", text)
	}
	if hasOwnGit(t, dir) || outerState(t, outer) != before {
		t.Error("the stop changed something")
	}
}

func TestSetupStopsWhenTheFolderIsASubmodule(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, dir := nestedVault(t)
	head := gitIn(t, outer, "rev-parse", "HEAD")
	gitIn(t, outer, "update-index", "--add", "--cacheinfo", "160000,"+head+",notes")
	before := outerState(t, outer)
	step := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json")).Begin(context.Background())
	text := stepText(step)
	if step.Outcome != gitsync.SetupFailed || !strings.Contains(text, "submodule") || !strings.Contains(text, "two repositories") {
		t.Fatalf("a folder that is a submodule must stop with why: %+v", step)
	}
	if hasOwnGit(t, dir) || outerState(t, outer) != before {
		t.Error("the stop changed something")
	}
}

func TestSetupStopsWhenTheOuterRepositoryStartsTrackingTheFolderMeanwhile(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, dir := nestedVault(t)
	ctx := context.Background()
	s := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json"))
	if step := s.Begin(ctx); step.Ask != gitsync.AskOwnRepo {
		t.Fatalf("begin: %+v", step)
	}
	gitIn(t, outer, "add", "notes/a.md")
	step := s.Next(ctx, gitsync.SetupAnswer{OK: true})
	if step.Outcome != gitsync.SetupFailed || !strings.Contains(stepText(step), "two repositories") {
		t.Fatalf("the folder is checked again before git init: %+v", step)
	}
	if hasOwnGit(t, dir) {
		t.Error("a repository was created in a folder the outer one tracks")
	}
}

func TestSetupCancelledRemovesTheRepositoryItCreated(t *testing.T) {
	t.Parallel()
	isolate(t)
	outer, dir := nestedVault(t)
	before := outerState(t, outer)
	ctx := context.Background()
	s := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json"))
	s.Begin(ctx)
	s.Next(ctx, gitsync.SetupAnswer{OK: true})
	step := s.Next(ctx, gitsync.SetupAnswer{})
	text := stepText(step)
	if step.Outcome != gitsync.SetupCancelled || !strings.Contains(text, "was removed") || !strings.Contains(text, "Nothing else changed") {
		t.Fatalf("cancel must say the repository went again: %+v", step)
	}
	if hasOwnGit(t, dir) {
		t.Error("the repository this run created and nothing used is still there")
	}
	if _, err := os.Stat(filepath.Join(dir, "a.md")); err != nil {
		t.Errorf("the note must stay: %v", err)
	}
	if outerState(t, outer) != before {
		t.Error("the outer repository changed")
	}
}

func TestSetupCancelledKeepsARepositoryWithCommits(t *testing.T) {
	t.Parallel()
	isolate(t)
	_, dir := nestedVault(t)
	ctx := context.Background()
	s := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json"))
	s.Begin(ctx)
	s.Next(ctx, gitsync.SetupAnswer{OK: true})
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "mine")
	step := s.Next(ctx, gitsync.SetupAnswer{})
	text := stepText(step)
	if step.Outcome != gitsync.SetupCancelled || !strings.Contains(text, "already has commits") || !strings.Contains(text, "left in place") {
		t.Fatalf("cancel must say the repository stays: %+v", step)
	}
	if !hasOwnGit(t, dir) || gitIn(t, dir, "log", "--format=%s") != "mine" {
		t.Error("a repository with commits must stay as it is")
	}
}

func TestSetupCancelledKeepsARepositoryItDidNotCreate(t *testing.T) {
	t.Parallel()
	isolate(t)
	_, dir := nestedVault(t)
	ctx := context.Background()
	s := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json"))
	if step := s.Begin(ctx); step.Ask != gitsync.AskOwnRepo {
		t.Fatalf("begin: %+v", step)
	}
	gitIn(t, dir, "init", "-q") // the user made it while the question waited
	if step := s.Next(ctx, gitsync.SetupAnswer{OK: true}); step.Ask != gitsync.AskStart {
		t.Fatalf("a folder that has a repository of its own goes on: %+v", step)
	}
	step := s.Next(ctx, gitsync.SetupAnswer{})
	if step.Outcome != gitsync.SetupCancelled || !strings.Contains(stepText(step), "nothing was changed") {
		t.Fatalf("cancel: %+v", step)
	}
	if !hasOwnGit(t, dir) {
		t.Error("a repository this run did not create was removed")
	}
}

// A folder that is not inside any repository goes straight to the start,
// as before, and a cancel there creates nothing.
func TestSetupOutsideAnyRepositoryStartsAsBefore(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s := newSetup(t, dir, filepath.Join(t.TempDir(), "config.json"))
	if step := s.Begin(ctx); step.Ask != gitsync.AskStart || step.Title != "Sync this vault" {
		t.Fatalf("begin: %+v", step)
	}
	if step := s.Next(ctx, gitsync.SetupAnswer{}); !strings.Contains(stepText(step), "nothing was changed") {
		t.Fatalf("cancel: %+v", step)
	}
	if hasOwnGit(t, dir) {
		t.Error("a cancelled setup created a repository")
	}
}
