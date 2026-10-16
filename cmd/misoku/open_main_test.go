package main

// Which folder misoku opens (P1), checked through the real
// main() in a child process. Every candidate folder is a plain file, so
// main stops at "Could not open workspace <chosen>" and never starts the
// TUI: the message names the folder misoku picked.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
)

// runProbe runs main() with args in cwd, home as home/config/cache, and
// env added (MISOKU_WORKSPACE cleared unless env sets it).
func runProbe(t *testing.T, home, cwd string, env []string, args ...string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$")
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), mainArgsEnv+"="+strings.Join(args, "\n"), "MISOKU_WORKSPACE=",
		"HOME="+home, "USERPROFILE="+home, "XDG_CACHE_HOME="+home, "XDG_CONFIG_HOME="+home, "LOCALAPPDATA="+home, "APPDATA="+home)
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return code, string(out)
}

func touch(t *testing.T, path string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpenPrecedenceThroughMain(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	files := t.TempDir()
	flagF := touch(t, filepath.Join(files, "flag"))
	argF := touch(t, filepath.Join(files, "arg"))
	envF := touch(t, filepath.Join(files, "env"))
	notesF := touch(t, filepath.Join(files, "notes"))
	if err := config.SetNotesDir(configIn(home), notesF); err != nil {
		t.Fatal(err)
	}
	env := []string{"MISOKU_WORKSPACE=" + envF}
	tests := []struct {
		name string
		env  []string
		args []string
		want string
	}{
		{"--dir beats env and notes_dir", env, []string{"--dir", flagF}, flagF},
		{"folder beats env and notes_dir", env, []string{argF}, argF},
		{"env beats notes_dir", env, nil, envF},
		{"notes_dir when nothing else", nil, nil, notesF},
		{"flags after the folder", nil, []string{argF, "--theme", "nord"}, argF},
		{"flags before the folder", nil, []string{"--theme", "nord", argF}, argF},
		{"--dir alone, as in v0.7.0", nil, []string{"--dir", flagF}, flagF},
	}
	for _, tt := range tests {
		code, out := runProbe(t, home, t.TempDir(), tt.env, tt.args...)
		// main prints the folder with %q, which doubles the backslashes of
		// a Windows path: quote it the same way.
		if code != 1 || !strings.Contains(out, "Could not open workspace "+strconv.Quote(tt.want)+":") {
			t.Errorf("%s: exit %d, want 1 opening %s:\n%s", tt.name, code, tt.want, out)
		}
	}
}

func TestTwoFoldersOrFolderWithDirIsUsage(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"a", "b"}, {"--dir", "a", "b"}, {"a", "--dir", "b"}} {
		cwd := t.TempDir()
		code, out := runProbe(t, t.TempDir(), cwd, nil, args...)
		if code != 2 || !strings.Contains(out, "misoku [folder] [flags]") {
			t.Errorf("misoku %v: exit %d, want 2 with usage:\n%s", args, code, out)
		}
		if entries, _ := os.ReadDir(cwd); len(entries) != 0 {
			t.Errorf("misoku %v created %v", args, entries)
		}
	}
}

// A remembered folder that is a dangling symlink (a Stow link to an
// unplugged disk) counts as missing: nothing is created, at the link or
// its target.
func TestDanglingNotesDirLinkCreatesNothing(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	target := filepath.Join(t.TempDir(), "Disk", "Notes")
	link := filepath.Join(t.TempDir(), "Notes")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("no symlinks:", err)
	}
	if err := config.SetNotesDir(configIn(home), link); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	code, out := runProbe(t, home, cwd, nil)
	if code != 1 || !strings.Contains(out, "Your notes folder "+link+" is missing") {
		t.Errorf("exit %d, want 1 with the missing message:\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Dir(target)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the link target was created (%v)", err)
	}
	if entries, _ := os.ReadDir(cwd); len(entries) != 0 {
		t.Errorf("the current folder got %v", entries)
	}
}

// A relative notes_dir is skipped with a warning; it never creates a
// folder named after it in the current folder.
func TestRelativeNotesDirIsIgnored(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	path := configIn(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"notes_dir":"relnotes"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	code, out := runProbe(t, home, cwd, []string{"MISOKU_WORKSPACE=" + touch(t, filepath.Join(t.TempDir(), "f"))})
	_ = code
	if strings.Contains(out, "Ignoring notes_dir") {
		t.Errorf("env set: notes_dir must not even be read:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(cwd, "relnotes")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("relnotes was created (%v)", err)
	}
}
