package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/tutor"
)

// fakeConfigHome points HOME (and the XDG folders) at a temporary folder, so
// tutor.Dir and the Trash are never the user's. On Windows os.UserConfigDir,
// os.UserCacheDir and os.UserHomeDir read %AppData%, %LocalAppData% and
// %UserProfile% instead, so those must point there too: otherwise the tests
// share the runner's real tutorial folder (and --reset deletes a developer's).
func fakeConfigHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	t.Setenv("USERPROFILE", home)
}

// S2: misoku tutor installs the lessons once and keeps a player's edits;
// --reset brings the bundled lesson back.
func TestTutorWorkspaceKeepsEditsUnlessReset(t *testing.T) {
	fakeConfigHome(t)
	dir, err := tutorWorkspace(nil)
	if err != nil {
		t.Fatal(err)
	}
	if dir != tutor.Dir() || dir == "" {
		t.Fatalf("dir = %q, want tutor.Dir() %q", dir, tutor.Dir())
	}
	lesson := filepath.Join(dir, "02 Casillas.md")
	bundled, err := os.ReadFile(lesson)
	if err != nil {
		t.Fatalf("the lesson was not installed: %v", err)
	}
	if err := os.WriteFile(lesson, []byte("- [x] hecho\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := tutorWorkspace(nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(lesson); string(got) != "- [x] hecho\n" {
		t.Fatalf("without --reset the edited lesson was replaced:\n%s", got)
	}

	if _, err := tutorWorkspace([]string{"--reset"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(lesson); string(got) != string(bundled) {
		t.Fatalf("--reset kept the edited lesson:\n%s", got)
	}
}

func TestTutorWorkspaceRejectsOtherArguments(t *testing.T) {
	fakeConfigHome(t)
	for _, args := range [][]string{{"--nope"}, {"folder"}} {
		if _, err := tutorWorkspace(args); err == nil {
			t.Errorf("tutorWorkspace(%q): no error", args)
		}
	}
}

func TestUsageListsTutor(t *testing.T) {
	var b strings.Builder
	usage(&b)
	if !strings.Contains(b.String(), "misoku tutor [--reset]") {
		t.Fatalf("usage does not list misoku tutor:\n%s", b.String())
	}
}
