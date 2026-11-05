package app_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"
)

// fakeNvimOnPath puts an "nvim" that is never run on PATH, so the launcher
// finds Neovim without the test depending on the machine's own.
func fakeNvimOnPath(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	name := "nvim"
	if runtime.GOOS == "windows" {
		name = "nvim.exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

// exitCode is the error a program that ended with code n hands back.
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("exit status %d", int(e)) }
func (e exitCode) ExitCode() int { return int(e) }

// fileManagerHarness is a vault with one note, the explorer focused, and
// a fake exec that records the command and, while "running", calls during.
func fileManagerHarness(t *testing.T, store *core.Store, during func(*exec.Cmd) error) (*harness, *[]*exec.Cmd) {
	t.Helper()
	if _, err := store.SaveAs("", "zeta", "zeta body"); err != nil {
		t.Fatal(err)
	}
	var ran []*exec.Cmd
	h := newHarness(t, store, app.WithExec(func(c *exec.Cmd, cb tea.ExecCallback) tea.Cmd {
		ran = append(ran, c)
		err := during(c)
		return func() tea.Msg { return cb(err) }
	}))
	h.keys("e")
	return h, &ran
}

func envValue(c *exec.Cmd, name string) (string, bool) {
	for _, kv := range c.Env {
		if v, ok := strings.CutPrefix(kv, name+"="); ok {
			return v, true
		}
	}
	return "", false
}

// M in the explorer opens Neovim's mini.files; the vault folder travels in
// the environment, verbatim, and never inside the Lua source.
func TestMOpensMiniFilesInNeovimWithTheVaultInTheEnvironment(t *testing.T) {
	fakeNvimOnPath(t)
	name := "my vault's dir"
	if runtime.GOOS != "windows" {
		name = `my "vault" dir`
	}
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	newTestStore(t) // keeps HOME away from the real one
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	var actions string
	h, ran := fileManagerHarness(t, store, func(c *exec.Cmd) error {
		actions, _ = envValue(c, "MISOKU_ACTIONS")
		if _, err := os.Stat(actions); err != nil {
			t.Errorf("the action log should exist while Neovim runs: %v", err)
		}
		return nil
	})
	h.keys("M")

	if len(*ran) != 1 {
		t.Fatalf("expected one program run, got %d", len(*ran))
	}
	c := (*ran)[0]
	if base := strings.TrimSuffix(filepath.Base(c.Path), ".exe"); base != "nvim" {
		t.Fatalf("expected nvim, got %q", c.Path)
	}
	i := slices.Index(c.Args, "-c")
	if i < 0 || i+1 >= len(c.Args) {
		t.Fatalf("expected nvim -c <lua>, got %q", c.Args)
	}
	lua := c.Args[i+1]
	for _, want := range []string{"require('lazy').load", "mini.files", "cquit 3", "MiniFilesExplorerClose", "permanent_delete = false", "MISOKU_DIR"} {
		if !strings.Contains(lua, want) {
			t.Errorf("the Lua should contain %q:\n%s", want, lua)
		}
	}
	if strings.Contains(lua, store.Root()) || strings.Contains(lua, name) {
		t.Errorf("the vault path must not be spliced into the Lua:\n%s", lua)
	}
	if got, ok := envValue(c, "MISOKU_DIR"); !ok || got != store.Root() {
		t.Fatalf("MISOKU_DIR: got %q, want %q", got, store.Root())
	}
	if actions == "" {
		t.Fatal("MISOKU_ACTIONS is not set")
	}
	if _, err := os.Stat(actions); !os.IsNotExist(err) {
		t.Errorf("the action log should be removed after Neovim ends: %v", err)
	}
}

// Without nvim on PATH nothing runs and the bar says what is missing.
func TestMWithoutNeovimRunsNothingAndSaysSo(t *testing.T) {
	store := newTestStore(t)
	t.Setenv("PATH", "")
	h, ran := fileManagerHarness(t, store, func(*exec.Cmd) error { return nil })
	h.keys("M")
	if len(*ran) != 0 {
		t.Fatalf("nothing should run without nvim, got %d runs", len(*ran))
	}
	if v := h.view(); !strings.Contains(v, "M needs Neovim (nvim) on PATH") {
		t.Fatalf("expected the missing-Neovim message:\n%s", v)
	}
}

// Neovim ending with code 3 means mini.files could not be loaded.
func TestMWhenMiniFilesIsMissingSaysHowToInstallIt(t *testing.T) {
	fakeNvimOnPath(t)
	store := newTestStore(t)
	h, _ := fileManagerHarness(t, store, func(*exec.Cmd) error { return exitCode(3) })
	h.keys("M")
	if v := h.view(); !strings.Contains(v, "mini.files is not installed in Neovim") {
		t.Fatalf("expected the mini.files message:\n%s", v)
	}
}

// A note created in mini.files is in the explorer as soon as Misoku is
// back, without waiting for the workspace poll.
func TestMRereadsTheWorkspaceOnReturn(t *testing.T) {
	fakeNvimOnPath(t)
	store := newTestStore(t)
	h, _ := fileManagerHarness(t, store, func(*exec.Cmd) error {
		return os.WriteFile(filepath.Join(store.Root(), "fromnvim.md"), []byte("x"), 0o644)
	})
	h.keys("M")
	if v := app.ExplorerView(h.m); !strings.Contains(v, "fromnvim") {
		t.Fatalf("the note made in mini.files should be in the explorer:\n%s", v)
	}
}

// The folder is read when Neovim ends, in the program's callback, and not
// later in Update: what the message carries is what the explorer shows,
// even if the disk changed again before the message was handled.
func TestMReadsTheTreeWhenNeovimEndsNotInUpdate(t *testing.T) {
	fakeNvimOnPath(t)
	store := newTestStore(t)
	if _, err := store.SaveAs("", "zeta", "zeta body"); err != nil {
		t.Fatal(err)
	}
	made := filepath.Join(store.Root(), "fromnvim.md")
	var done tea.Msg
	h := newHarness(t, store, app.WithExec(func(_ *exec.Cmd, cb tea.ExecCallback) tea.Cmd {
		if err := os.WriteFile(made, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		done = cb(nil)
		return nil
	}))
	h.keys("e")
	h.keys("M")
	if done == nil {
		t.Fatal("Neovim never ran")
	}
	if err := os.Remove(made); err != nil {
		t.Fatal(err)
	}
	h.send(done)
	if v := app.ExplorerView(h.m); !strings.Contains(v, "fromnvim") {
		t.Fatalf("the explorer should show the tree read when Neovim ended:\n%s", v)
	}
}
