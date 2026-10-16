package gitsync_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// gitAt writes an executable git in dir (git.exe on Windows and for
// Program Files, which is where Git for Windows puts it) and returns its
// path. It prints a git version on systems that can run a script.
func gitAt(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\necho 'git version 2.45.1'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// pathGit is the file name git has on PATH on this system.
func pathGit() string {
	if runtime.GOOS == "windows" {
		return "git.exe"
	}
	return "git"
}

// The installer's winget puts git in Program Files and adds it to a PATH
// the open window does not see yet: Misoku finds it there on Windows.
func TestGitFoundInProgramFilesWhenNotOnPath(t *testing.T) {
	defer gitsync.SetGOOS("windows")()
	pf := t.TempDir()
	want := gitAt(t, filepath.Join(pf, "Git", "cmd"), "git.exe")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ProgramFiles", pf)

	if got := gitsync.GitProgram(); got != want {
		t.Fatalf("git program = %q, want %q", got, want)
	}
}

// A git on PATH wins over the one in Program Files.
func TestGitOnPathPreferredOverProgramFiles(t *testing.T) {
	defer gitsync.SetGOOS("windows")()
	pf := t.TempDir()
	gitAt(t, filepath.Join(pf, "Git", "cmd"), "git.exe")
	bin := t.TempDir()
	want := gitAt(t, bin, pathGit())
	t.Setenv("PATH", bin)
	t.Setenv("ProgramFiles", pf)

	if got := gitsync.GitProgram(); got != want {
		t.Fatalf("git program = %q, want the one on PATH %q", got, want)
	}
}

// Program Files is looked at only on Windows.
func TestGitProgramFilesOnlyOnWindows(t *testing.T) {
	defer gitsync.SetGOOS("linux")()
	pf := t.TempDir()
	gitAt(t, filepath.Join(pf, "Git", "cmd"), "git.exe")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ProgramFiles", pf)

	if got := gitsync.GitProgram(); got != "git" {
		t.Fatalf("git program = %q, want plain git", got)
	}
}

// With git nowhere, the error is the one Misoku always gave.
func TestGitMissingKeepsTheMessage(t *testing.T) {
	defer gitsync.SetGOOS("windows")()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ProgramFiles", t.TempDir())

	if got := gitsync.GitProgram(); got != "git" {
		t.Fatalf("git program = %q, want plain git", got)
	}
	_, _, err := gitsync.New(t.TempDir(), 10*time.Second).Version(context.Background())
	if err == nil || !strings.Contains(err.Error(), `exec: "git": executable file not found`) {
		t.Fatalf("Version error = %v, want git's not-found error", err)
	}
}
