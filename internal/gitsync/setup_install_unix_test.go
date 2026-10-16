//go:build !windows

// The fake gh is a shell script: these tests run on macOS and Linux. They
// change PATH, so none runs in parallel.

package gitsync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// The installer puts gh next to misoku, in a folder that may not be on
// PATH yet: setup finds it there.
func TestSetupFindsGHNextToMisoku(t *testing.T) {
	f := newEngineGH(t, false, "")
	exeDir := t.TempDir()
	if err := os.Rename(filepath.Join(f.bin, "gh"), filepath.Join(exeDir, "gh")); err != nil {
		t.Fatal(err)
	}
	defer gitsync.SetExecutable(filepath.Join(exeDir, "misoku"))()
	s, step := engineSetup(t, "linux")
	if step.Ask != gitsync.AskSignIn {
		t.Fatalf("gh next to misoku was not found: %+v", step)
	}
	if !strings.Contains(strings.Join(step.Notes, "\n"), "✓ gh 2.97.0") {
		t.Errorf("notes lack gh's version: %v", step.Notes)
	}
	if got := s.SignInCommand().Path; got != filepath.Join(exeDir, "gh") {
		t.Errorf("sign-in runs %q, want the gh next to misoku", got)
	}
}

// A gh on PATH still wins over the one next to misoku.
func TestSetupPrefersGHOnPath(t *testing.T) {
	f := newEngineGH(t, false, "")
	exeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(exeDir, "gh"), []byte("#!/bin/sh\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	defer gitsync.SetExecutable(filepath.Join(exeDir, "misoku"))()
	s, step := engineSetup(t, "linux")
	if step.Ask != gitsync.AskSignIn {
		t.Fatalf("want the sign-in, got %+v", step)
	}
	if got := s.SignInCommand().Path; got != filepath.Join(f.bin, "gh") {
		t.Errorf("sign-in runs %q, want the gh on PATH", got)
	}
}

// With gh neither on PATH nor next to misoku, setup stops with the hint,
// as before.
func TestSetupWithoutGHAnywhereSaysHowToInstallIt(t *testing.T) {
	f := newEngineGH(t, false, "")
	if err := os.Remove(filepath.Join(f.bin, "gh")); err != nil {
		t.Fatal(err)
	}
	defer gitsync.SetExecutable(filepath.Join(t.TempDir(), "misoku"))()
	_, step := engineSetup(t, "linux")
	if step.Outcome != gitsync.SetupFailed || !strings.Contains(strings.Join(step.Lines, "\n"), gitsync.GHInstallHint("linux")) {
		t.Fatalf("want the install hint, got %+v", step)
	}
}

// noGit leaves PATH without git, and a vault with one note.
func noGit(t *testing.T) string {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const appleProblem = "Apple is installing git; press Enter when it finishes."

// On a Mac, while Apple's installer for git runs, setup asks to wait
// instead of failing, and Enter checks git again.
func TestSetupWaitsWhileAppleInstallsGit(t *testing.T) {
	dir := noGit(t)
	running := true
	defer gitsync.SetAppleInstallingGit(func() bool { return running })()
	s := gitsync.NewSetup(gitsync.SetupOptions{
		Dir: dir, ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Timeout: 30 * time.Second, OS: "darwin", GH: filepath.Join(t.TempDir(), "no-gh"),
	})
	ctx := context.Background()
	if step := s.Begin(ctx); step.Ask != gitsync.AskStart {
		t.Fatalf("begin: %+v", step)
	}
	step := s.Next(ctx, gitsync.SetupAnswer{OK: true})
	if step.Ask != gitsync.AskStart || step.Problem != appleProblem {
		t.Fatalf("want to wait for Apple's installer, got %+v", step)
	}
	step = s.Next(ctx, gitsync.SetupAnswer{OK: true})
	if step.Ask != gitsync.AskStart || step.Problem != appleProblem {
		t.Fatalf("still installing: want to wait again, got %+v", step)
	}
	running = false
	step = s.Next(ctx, gitsync.SetupAnswer{OK: true})
	if step.Outcome != gitsync.SetupFailed || !strings.Contains(strings.Join(step.Lines, "\n"), "git did not run") {
		t.Fatalf("installer gone and still no git: want the git failure, got %+v", step)
	}
}

// Without Apple's installer, or on another system, a missing git stops
// setup as before.
func TestSetupWithoutGitFailsAsBefore(t *testing.T) {
	for _, c := range []struct {
		goos    string
		running bool
	}{{"darwin", false}, {"linux", true}} {
		dir := noGit(t)
		restore := gitsync.SetAppleInstallingGit(func() bool { return c.running })
		s := gitsync.NewSetup(gitsync.SetupOptions{
			Dir: dir, ConfigPath: filepath.Join(t.TempDir(), "config.json"),
			Timeout: 30 * time.Second, OS: c.goos, GH: filepath.Join(t.TempDir(), "no-gh"),
		})
		ctx := context.Background()
		s.Begin(ctx)
		step := s.Next(ctx, gitsync.SetupAnswer{OK: true})
		restore()
		if step.Outcome != gitsync.SetupFailed || !strings.Contains(strings.Join(step.Lines, "\n"), "git did not run") {
			t.Errorf("%s, installer %v: want the git failure, got %+v", c.goos, c.running, step)
		}
	}
}

// Once Apple's installer is done, the folder is looked at as Begin would
// have: one inside another repository is offered its own.
func TestSetupLooksAtTheFolderOnceAppleInstalledGit(t *testing.T) {
	isolate(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	outer := newRepo(t)
	dir := filepath.Join(outer, "notes")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	running := true
	defer gitsync.SetAppleInstallingGit(func() bool { return running })()
	s := gitsync.NewSetup(gitsync.SetupOptions{
		Dir: dir, ConfigPath: filepath.Join(t.TempDir(), "config.json"),
		Timeout: 30 * time.Second, OS: "darwin", GH: filepath.Join(t.TempDir(), "no-gh"),
	})
	ctx := context.Background()
	if step := s.Begin(ctx); step.Ask != gitsync.AskStart {
		t.Fatalf("begin without git: %+v", step)
	}
	if step := s.Next(ctx, gitsync.SetupAnswer{OK: true}); step.Problem != appleProblem {
		t.Fatalf("want to wait for Apple's installer, got %+v", step)
	}
	running = false
	t.Setenv("PATH", filepath.Dir(gitPath))
	step := s.Next(ctx, gitsync.SetupAnswer{OK: true})
	if step.Ask != gitsync.AskOwnRepo {
		t.Fatalf("want the folder's own repository offered, got %+v", step)
	}
}
