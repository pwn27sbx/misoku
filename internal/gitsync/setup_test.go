package gitsync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"
)

func TestGHInstallHintPerSystem(t *testing.T) {
	t.Parallel()
	for goos, want := range map[string]string{
		"darwin":  "brew install gh",
		"linux":   "apt install gh",
		"windows": "winget install --id GitHub.cli",
		"freebsd": "package manager",
	} {
		if got := gitsync.GHInstallHint(goos); !strings.Contains(got, want) {
			t.Errorf("%s: %q lacks %q", goos, got, want)
		}
	}
}

func TestRepoNameFromTheVaultFolder(t *testing.T) {
	t.Parallel()
	for dir, want := range map[string]string{
		"notes":         "notes",
		"Mis notas":     "Mis-notas",
		"ObsidianVault": "ObsidianVault",
		"my.notes_2":    "my.notes_2",
		"...":           "notes",
	} {
		if got := gitsync.RepoName(filepath.Join(t.TempDir(), dir)); got != want {
			t.Errorf("%q: %q, want %q", dir, got, want)
		}
	}
}

// newSetup is the assistant for dir with gh at a path where there is
// none, so a test that passes never ran gh.
func newSetup(t *testing.T, dir, cfg string) *gitsync.Setup {
	t.Helper()
	return gitsync.NewSetup(gitsync.SetupOptions{
		Dir: dir, ConfigPath: cfg, Timeout: 30 * time.Second,
		GH: filepath.Join(t.TempDir(), "no-gh"),
	})
}

func readAll(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		files[p] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestSetupWithoutGHSaysHowToInstallIt(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(t.TempDir(), "config.json")
	ctx := context.Background()
	s := newSetup(t, dir, cfg)
	if step := s.Begin(ctx); step.Ask != gitsync.AskStart {
		t.Fatalf("begin: %+v", step)
	}
	step := s.Next(ctx, gitsync.SetupAnswer{OK: true})
	if step.Ask != gitsync.SetupFinished || step.Outcome != gitsync.SetupFailed {
		t.Fatalf("without gh setup must stop: %+v", step)
	}
	text := strings.Join(step.Lines, "\n")
	if !strings.Contains(text, gitsync.GHInstallHint(runtime.GOOS)) {
		t.Errorf("the hint for this system is missing:\n%s", text)
	}
	if files := readAll(t, dir); len(files) != 1 {
		t.Errorf("the vault changed: %v", files)
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Errorf("the config was written: %v", err)
	}
}

func TestSetupCancelledAtTheStartChangesNothing(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "config.json")
	ctx := context.Background()
	s := newSetup(t, dir, cfg)
	s.Begin(ctx)
	step := s.Next(ctx, gitsync.SetupAnswer{})
	if step.Outcome != gitsync.SetupCancelled || !strings.Contains(strings.Join(step.Lines, " "), "nothing was changed") {
		t.Fatalf("cancel: %+v", step)
	}
	if files := readAll(t, dir); len(files) != 0 {
		t.Errorf("the vault changed: %v", files)
	}
}

func TestSetupOnAVaultThatAlreadySyncsSaysSo(t *testing.T) {
	t.Parallel()
	isolate(t)
	dir := newRepo(t)
	if out, err := exec.Command("git", "-C", dir, "remote", "add", "origin", "https://user:secret@example.com/ana/notes.git").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(cfg, config.Config{Sync: config.Sync{Vaults: map[string]config.SyncVault{dir: {}}}}); err != nil {
		t.Fatal(err)
	}
	before := readAll(t, dir)
	step := newSetup(t, dir, cfg).Begin(context.Background())
	text := strings.Join(step.Lines, "\n")
	if step.Outcome != gitsync.SetupAlready || !strings.Contains(text, "already set up") {
		t.Fatalf("a vault that syncs must be reported as set up: %+v", step)
	}
	if strings.Contains(text, "secret") {
		t.Errorf("the message shows the URL's password:\n%s", text)
	}
	if after := readAll(t, dir); len(after) != len(before) {
		t.Error("setup changed the vault")
	}
	for p, data := range before {
		if readAll(t, dir)[p] != data {
			t.Errorf("%s changed", p)
		}
	}
}
