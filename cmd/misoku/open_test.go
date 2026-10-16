package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/config"
)

// configIn is where main(), run by runMainIn with home, reads its config:
// os.UserConfigDir under that home.
func configIn(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "misoku", "config.json")
	}
	return filepath.Join(home, "misoku", "config.json") // XDG_CONFIG_HOME, APPDATA
}

func cwdIs(dir string) func() (string, error) {
	return func() (string, error) { return dir, nil }
}

// Which folder misoku opens, for every pair of sources: --dir and a
// folder argument (both explicit) beat $MISOKU_WORKSPACE, which beats
// notes_dir, which beats the current folder.
func TestPickWorkspacePrecedence(t *testing.T) {
	t.Parallel()
	notes := t.TempDir() // notes_dir that exists
	cfg := config.Config{NotesDir: notes}
	cwd := cwdIs(filepath.Join("the", "terminal"))
	for _, tc := range []struct {
		name string
		in   workspaceInput
		dir  string
		src  workspaceSource
	}{
		{"flag beats env", workspaceInput{Flag: "f", Env: "e", Getwd: cwd}, "f", fromFlag},
		{"flag beats notes_dir", workspaceInput{Flag: "f", Config: cfg, Getwd: cwd}, "f", fromFlag},
		{"flag beats cwd", workspaceInput{Flag: "f", Getwd: cwd}, "f", fromFlag},
		{"arg beats env", workspaceInput{Arg: "a", Env: "e", Getwd: cwd}, "a", fromArg},
		{"arg beats notes_dir", workspaceInput{Arg: "a", Config: cfg, Getwd: cwd}, "a", fromArg},
		{"arg beats cwd", workspaceInput{Arg: "a", Getwd: cwd}, "a", fromArg},
		{"env beats notes_dir", workspaceInput{Env: "e", Config: cfg, Getwd: cwd}, "e", fromEnv},
		{"env beats cwd", workspaceInput{Env: "e", Getwd: cwd}, "e", fromEnv},
		{"notes_dir beats cwd", workspaceInput{Config: cfg, Getwd: cwd}, notes, fromNotesDir},
		{"cwd when nothing else", workspaceInput{Getwd: cwd}, filepath.Join("the", "terminal"), fromCwd},
		{"dot when there is no cwd", workspaceInput{Getwd: func() (string, error) { return "", errors.New("gone") }}, ".", fromCwd},
	} {
		got := pickWorkspace(tc.in)
		if got.Dir != tc.dir || got.Source != tc.src || got.Warning != "" {
			t.Errorf("%s: got %+v, want dir %q source %d", tc.name, got, tc.dir, tc.src)
		}
	}
}

// v0.7.0 behaviour of --dir, kept: the value is used as typed (relative
// stays relative), even when it does not exist yet and even with a
// notes_dir whose folder is missing.
func TestDirFlagBehavesAsBefore(t *testing.T) {
	t.Parallel()
	gone := config.Config{NotesDir: filepath.Join(t.TempDir(), "unplugged")}
	got := pickWorkspace(workspaceInput{Flag: filepath.Join("new", "notes"), Config: gone, Getwd: cwdIs("x")})
	if got.Dir != filepath.Join("new", "notes") || got.Source != fromFlag {
		t.Errorf("got %+v", got)
	}
}

// v0.7.0 behaviour of $MISOKU_WORKSPACE, kept: used as written, above a
// notes_dir whose folder is missing (the `notes` alias keeps working).
func TestEnvBehavesAsBefore(t *testing.T) {
	t.Parallel()
	gone := config.Config{NotesDir: filepath.Join(t.TempDir(), "unplugged")}
	got := pickWorkspace(workspaceInput{Env: "rel", Config: gone, Getwd: cwdIs("x")})
	if got.Dir != "rel" || got.Source != fromEnv {
		t.Errorf("got %+v", got)
	}
}

// The remembered folder is gone (an unplugged disk): the choice says so,
// with the folder, so the caller can explain instead of creating it.
func TestPickWorkspaceReportsAMissingNotesDir(t *testing.T) {
	t.Parallel()
	gone := filepath.Join(t.TempDir(), "unplugged")
	got := pickWorkspace(workspaceInput{Config: config.Config{NotesDir: gone}, Getwd: cwdIs("x")})
	if got.Dir != gone || got.Source != notesDirMissing {
		t.Errorf("got %+v, want %q missing", got, gone)
	}
}

// A notes_dir Misoku cannot use (relative) is warned about and skipped.
func TestPickWorkspaceWarnsAboutABadNotesDir(t *testing.T) {
	t.Parallel()
	got := pickWorkspace(workspaceInput{Config: config.Config{NotesDir: "notes"}, Getwd: cwdIs("x")})
	if got.Dir != "x" || got.Source != fromCwd || !strings.Contains(got.Warning, `"notes"`) {
		t.Errorf("got %+v", got)
	}
}

// `misoku .`, `misoku ~/Other`: the argument is the folder; the flags
// work before and after it.
func TestFolderArgumentWithFlagsAround(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"Other"},
		{"--theme", "nord", "Other"},
		{"Other", "--theme", "nord"},
		{"--theme", "nord", "Other", "--themes-dir", "t"},
	} {
		fs := flag.NewFlagSet("misoku", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		opts, folder, err := parseOpenArgs(fs, args)
		if err != nil || folder != "Other" {
			t.Errorf("%q: folder %q, %v", args, folder, err)
			continue
		}
		if len(args) > 1 && opts.theme != "nord" {
			t.Errorf("%q: theme %q", args, opts.theme)
		}
		if len(args) == 5 && opts.themesDir != "t" {
			t.Errorf("%q: themes-dir %q", args, opts.themesDir)
		}
	}
}

// Two folders, or a folder and --dir: which one is meant is unclear, so
// misoku prints the usage and exits 2 without opening anything.
func TestTwoFoldersIsUsage(t *testing.T) {
	t.Parallel()
	a, b := t.TempDir(), t.TempDir()
	for _, args := range [][]string{{a, b}, {"--dir", a, b}, {a, "--dir", b}} {
		code, out := runMain(t, args...)
		if code != 2 {
			t.Errorf("misoku %q: exit %d, want 2:\n%s", args, code, out)
		}
		wantOutput(t, out, "misoku [folder]")
	}
}

// The help shows `misoku [folder]`.
func TestHelpShowsTheFolderArgument(t *testing.T) {
	t.Parallel()
	code, out := runMain(t, "--help")
	if code != 0 {
		t.Fatalf("misoku --help: exit %d:\n%s", code, out)
	}
	wantOutput(t, out, "misoku [folder] [flags]", "notes_dir")
}

// notes_dir is set but its folder is gone: one clear line, exit 1, and
// the folder is not created (it may be a disk that is not plugged in).
func TestMissingNotesFolderSaysSoAndCreatesNothing(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	gone := filepath.Join(t.TempDir(), "Disk", "Notes")
	if err := config.SetNotesDir(configIn(home), gone); err != nil {
		t.Fatal(err)
	}
	code, out := runMainIn(t, home)
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	wantOutput(t, out, "Your notes folder "+gone+" is missing; open another with misoku <folder>")
	if _, err := os.Stat(filepath.Dir(gone)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the missing folder was created (%v)", err)
	}
}
