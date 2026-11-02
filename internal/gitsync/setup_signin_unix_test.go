//go:build !windows

package gitsync_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// signInGH is the fake gh for FinishSignIn; nosetup makes `auth setup-git` fail.
const signInGH = `#!/bin/sh
d="$FAKE_GH_DIR"
printf '%s\n' "$*" >> "$d/calls"
case "$1 $2" in
"auth status") [ -f "$d/loggedin" ] && exit 0; echo "not logged in" >&2; exit 1 ;;
"auth setup-git") [ -f "$d/nosetup" ] && { echo "denied, token ghp_abcdefgh12345678" >&2; exit 1; }; exit 0 ;;
*) exit 2 ;;
esac
`

func signInSetup(t *testing.T, signedIn bool) (*engineGH, *gitsync.Setup) {
	t.Helper()
	f := newEngineGH(t, signedIn, "")
	f.write(t, "bin/gh", signInGH)
	return f, gitsync.NewSetup(gitsync.SetupOptions{Dir: t.TempDir()})
}

func TestFindGHResolvesTheProgramTheSignInRuns(t *testing.T) {
	f, s := signInSetup(t, false)
	if !s.FindGH() {
		t.Fatal("gh is on PATH")
	}
	if got, want := s.SignInCommand().Path, filepath.Join(f.bin, "gh"); got != want {
		t.Errorf("the sign-in runs %q, want %q", got, want)
	}
}

// The installer's gh, next to Misoku and not on PATH, is the one the sign-in runs.
func TestFindGHFindsTheOneNextToMisoku(t *testing.T) {
	f, s := signInSetup(t, false)
	t.Cleanup(gitsync.SetExecutable(filepath.Join(f.bin, "misoku")))
	t.Setenv("PATH", t.TempDir())
	if !s.FindGH() {
		t.Fatal("gh is next to the executable")
	}
	if got, want := s.SignInCommand().Path, filepath.Join(f.bin, "gh"); got != want {
		t.Errorf("the sign-in runs %q, want %q", got, want)
	}
}

func TestFindGHSaysWhenThereIsNone(t *testing.T) {
	_, s := signInSetup(t, false)
	t.Setenv("PATH", t.TempDir())
	if s.FindGH() {
		t.Error("no gh is on PATH or next to the executable")
	}
}

func TestFinishSignInGivesGitTheAccess(t *testing.T) {
	f, s := signInSetup(t, true)
	if !s.FindGH() {
		t.Fatal("gh is on PATH")
	}
	if err := s.FinishSignIn(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "auth status --hostname github.com\nauth setup-git --hostname github.com\n"
	if got := f.read(t, "calls"); got != want {
		t.Errorf("gh ran:\n%s\nwant:\n%s", got, want)
	}
}

func TestFinishSignInStopsWhenTheSignInDidNotFinish(t *testing.T) {
	f, s := signInSetup(t, false)
	s.FindGH()
	if err := s.FinishSignIn(context.Background()); err == nil {
		t.Fatal("not signed in must be an error")
	}
	if got := f.read(t, "calls"); strings.Contains(got, "setup-git") {
		t.Errorf("git's access must not change when not signed in:\n%s", got)
	}
}

func TestFinishSignInSaysWhenGitGetsNoAccessAndHidesTokens(t *testing.T) {
	f, s := signInSetup(t, true)
	f.write(t, "nosetup", "")
	s.FindGH()
	err := s.FinishSignIn(context.Background())
	if err == nil {
		t.Fatal("a failed setup-git must be an error")
	}
	if strings.Contains(err.Error(), "ghp_") {
		t.Errorf("a token in gh's message must be hidden: %v", err)
	}
}
