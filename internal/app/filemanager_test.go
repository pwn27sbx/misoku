package app_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
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
	for _, want := range []string{"require('lazy').load", "mini.files", "cquit 3", "MiniFilesExplorerClose", "permanent_delete = false", "MISOKU_DIR", "MISOKU_ACTIONS", "MiniFilesActionRename", "MiniFilesActionMove", "vim.json.encode(ev.data)"} {
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

// miniFilesRun is a vault with zeta.md linking to doc.md, doc.md open in
// a tab (with " mine" typed and unsaved when edit is set), and a fake
// mini.files that runs during on the vault and appends its action events
// to the log the way the Lua does.
func miniFilesRun(t *testing.T, edit bool, during func(root string, log func(action, from, to string))) *harness {
	t.Helper()
	fakeNvimOnPath(t)
	store := newTestStore(t)
	if _, err := store.SaveAs("", "zeta", "see [[doc]]"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "doc", "base"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithExec(func(c *exec.Cmd, cb tea.ExecCallback) tea.Cmd {
		actions, _ := envValue(c, "MISOKU_ACTIONS")
		root, _ := envValue(c, "MISOKU_DIR")
		during(root, func(action, from, to string) {
			line := fmt.Sprintf("{\"action\":%q,\"from\":%q,\"to\":%q}\n", action, from, to)
			f, err := os.OpenFile(actions, os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			if _, err := f.WriteString(line); err != nil {
				t.Fatal(err)
			}
		})
		return func() tea.Msg { return cb(nil) }
	}))
	h.keys("1")
	if edit {
		h.keys("A mine")
	}
	h.key(tea.KeyEsc)
	h.keys(" e") // the explorer
	return h
}

func moveOnDisk(t *testing.T, root, from, to string) (string, string) {
	t.Helper()
	src, dst := filepath.Join(root, filepath.FromSlash(from)), filepath.Join(root, filepath.FromSlash(to))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}
	return src, dst
}

// A rename or a move made in mini.files is read from its action log: the
// open tab follows the note instead of going "gone", and the links that
// named it are rewritten after the same question as a rename in Misoku. A
// move keeps the name, so [[doc]] still finds it and nothing is asked.
func TestMiniFilesRenameAndMoveKeepTheTabAndRewriteTheLinks(t *testing.T) {
	for _, tc := range []struct{ action, to, link string }{
		{"rename", "moved.md", "[[moved]]"},
		{"move", "sub/doc.md", "[[doc]]"},
	} {
		for _, edit := range []bool{false, true} {
			h := miniFilesRun(t, edit, func(root string, log func(action, from, to string)) {
				from, to := moveOnDisk(t, root, "doc.md", tc.to)
				log(tc.action, from, to)
			})
			h.keys("M")
			if asked := strings.Contains(h.view(), "Update 1 link in 1 note?"); asked != (tc.action == "rename") {
				t.Fatalf("%s edit=%v: relink question shown=%v:\n%s", tc.action, edit, asked, h.view())
			}
			if tc.action == "rename" {
				h.key(tea.KeyEnter) // Yes
			}
			if got := readNote(t, h.store.Root(), "zeta.md"); got != "see "+tc.link {
				t.Fatalf("%s edit=%v: the backlink should follow, got %q", tc.action, edit, got)
			}
			if edit { // saved at once, before any check: no "Recreate" question
				h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
				if v := h.view(); strings.Contains(v, "Recreate") {
					t.Fatalf("%s edit=%v: the moved tab must save in place:\n%s", tc.action, edit, v)
				}
				if got := readNote(t, h.store.Root(), tc.to); got != "base mine" {
					t.Fatalf("%s edit=%v: the save should land at %s, got %q", tc.action, edit, tc.to, got)
				}
			}
			h.send(app.CheckWorkspaceMsg{})
			v := h.view()
			if strings.Contains(v, goneWarning) {
				t.Fatalf("%s edit=%v: the tab must follow the note, not go gone:\n%s", tc.action, edit, v)
			}
			if !strings.Contains(strings.Split(v, "\n")[0], path.Base(tc.to)) {
				t.Fatalf("%s edit=%v: the tab should be named after the note's new place:\n%s", tc.action, edit, v)
			}
			h.send(tea.KeyMsg{Type: tea.KeyCtrlL}) // back to the tab, then not even a warning that went by
			h.ex("messages")
			if v := h.view(); !strings.Contains(v, "Messages") || strings.Contains(v, goneWarning) {
				t.Fatalf("%s edit=%v: the tab must never have been gone:\n%s", tc.action, edit, v)
			}
		}
	}
}

// A delete in mini.files goes to its trash, outside the vault: that is no
// move, and the tab warns that its note is gone, as before.
func TestMiniFilesDeleteLeavesTheGoneWarning(t *testing.T) {
	h := miniFilesRun(t, false, func(root string, log func(action, from, to string)) {
		trash := filepath.Join(t.TempDir(), "doc.md")
		if err := os.Rename(filepath.Join(root, "doc.md"), trash); err != nil {
			t.Fatal(err)
		}
		log("delete", filepath.Join(root, "doc.md"), trash)
	})
	h.keys("M")
	h.send(app.CheckWorkspaceMsg{})
	if v := h.view(); !strings.Contains(v, goneWarning) || strings.Contains(v, "Update ") {
		t.Fatalf("a deleted note's tab should warn, with no relink question:\n%s", v)
	}
	if got := readNote(t, h.store.Root(), "zeta.md"); got != "see [[doc]]" {
		t.Fatalf("a delete must not touch links, got %q", got)
	}
}

// With no events in the log (another file manager, a crash) the tree is
// read again and nothing else happens.
func TestMiniFilesWithoutALogOnlyRereadsTheTree(t *testing.T) {
	h := miniFilesRun(t, false, func(root string, _ func(action, from, to string)) {
		moveOnDisk(t, root, "doc.md", "moved.md")
	})
	h.keys("M")
	if v := app.ExplorerView(h.m); !strings.Contains(v, "moved") {
		t.Fatalf("the explorer should show the renamed note:\n%s", v)
	}
	if v := h.view(); strings.Contains(v, "Update ") {
		t.Fatalf("no log, no relink question:\n%s", v)
	}
}

// A note renamed in mini.files takes its sub-note folder along (Q2), even
// when the log names the vault by another path (a symlink, /var against
// /private/var on macOS).
func TestMiniFilesRenameMovesTheSubNoteFolderAlong(t *testing.T) {
	h := miniFilesRun(t, false, func(root string, log func(action, from, to string)) {
		if err := os.MkdirAll(filepath.Join(root, "doc"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "doc", "child.md"), []byte("kid"), 0o644); err != nil {
			t.Fatal(err)
		}
		real, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatal(err)
		}
		moveOnDisk(t, root, "doc.md", "moved.md")
		log("rename", filepath.Join(real, "doc.md"), filepath.Join(real, "moved.md"))
	})
	h.keys("M")
	if got := readNote(t, h.store.Root(), "moved/child.md"); got != "kid" {
		t.Fatalf("the sub-note should follow its note, got %q", got)
	}
	if noteExists(t, h.store.Root(), "doc") {
		t.Fatal("the old sub-note folder should be gone")
	}
}
