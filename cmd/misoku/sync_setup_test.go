package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// The help lists the assistant as the recommended way to set up sync,
// and init for any other git URL.
func TestHelpListsSyncSetup(t *testing.T) {
	t.Parallel()
	code, out := runMain(t, "--help")
	if code != 0 {
		t.Fatalf("misoku --help: exit %d:\n%s", code, out)
	}
	wantOutput(t, out, "misoku sync setup [--dir <folder>]", "(recommended)", "misoku sync init <url> [--dir <folder>]")
	code, out = runMain(t, "sync", "--help")
	if code != 0 {
		t.Fatalf("misoku sync --help: exit %d:\n%s", code, out)
	}
	wantOutput(t, out, "misoku sync setup")
}

// Inside another repository the assistant offers a repository of its own
// (Enter creates it) and goes on; cancelling at the next question
// removes it again, as nothing used it yet.
func TestSyncSetupOffersARepositoryOfItsOwnAndRemovesItOnCancel(t *testing.T) {
	t.Parallel()
	needGit(t)
	outer := t.TempDir()
	git(t, outer, "init", "-q", "-b", "main")
	dir := filepath.Join(outer, "notes")
	writeFile(t, dir, "a.md", "a\n")
	before := treeHash(t, outer)
	var out bytes.Buffer
	cfg := filepath.Join(t.TempDir(), "config.json")
	code := syncSetup(context.Background(), []string{"--dir", dir}, strings.NewReader("\n"), &out, &out, cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1 (cancelled):\n%s", code, out.String())
	}
	wantOutput(t, out.String(),
		"This folder is inside another git repository. Create a separate one for your notes? [Y/n]",
		"Start? [Y/n]", "was removed")
	if _, err := os.Lstat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Errorf("the repository the cancelled run created must go: %v", err)
	}
	if treeHash(t, outer) != before {
		t.Error("the cancelled setup changed the folder or the outer repository")
	}
}

// lockedBuffer is an output both the assistant and the test read and
// write at once.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// Ctrl+C while a question waits is a cancel: the repository this run
// created goes again, and the message says so.
func TestSyncSetupInterruptAtAQuestionCancels(t *testing.T) {
	t.Parallel()
	needGit(t)
	outer := t.TempDir()
	git(t, outer, "init", "-q", "-b", "main")
	dir := filepath.Join(outer, "notes")
	writeFile(t, dir, "a.md", "a\n")
	in, typed := io.Pipe()
	defer typed.Close()
	out := &lockedBuffer{}
	interrupt := make(chan os.Signal, 1)
	cfg := filepath.Join(t.TempDir(), "config.json")
	done := make(chan int, 1)
	go func() {
		done <- runSetup(context.Background(), []string{"--dir", dir}, in, out, out, cfg, interrupt, false)
	}()
	if _, err := typed.Write([]byte("\n")); err != nil { // create the repository
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for !strings.Contains(out.String(), "Start? [Y/n]") {
		if time.Now().After(deadline) {
			t.Fatalf("the assistant never asked to start:\n%s", out.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	interrupt <- os.Interrupt
	select {
	case code := <-done:
		if code != 1 {
			t.Errorf("exit %d, want 1 (cancelled)", code)
		}
	case <-time.After(30 * time.Second):
		t.Fatalf("Ctrl+C at a question must cancel the setup:\n%s", out.String())
	}
	wantOutput(t, out.String(), "Setup cancelled", "was removed")
	if _, err := os.Lstat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Errorf("the repository the interrupted run created must go: %v", err)
	}
}

// A vault folder that does not exist yet inside another repository is
// offered a repository of its own too; cancelling removes the folder and
// the repository this run created, and the outer repository is as it was.
func TestSyncSetupOffersARepositoryOfItsOwnForAMissingFolder(t *testing.T) {
	t.Parallel()
	needGit(t)
	outer := t.TempDir()
	git(t, outer, "init", "-q", "-b", "main")
	writeFile(t, outer, "dotfile", "x\n")
	dir := filepath.Join(outer, "notes")
	before := treeHash(t, outer)
	var out bytes.Buffer
	cfg := filepath.Join(t.TempDir(), "config.json")
	code := syncSetup(context.Background(), []string{"--dir", dir}, strings.NewReader("\n"), &out, &out, cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1 (cancelled):\n%s", code, out.String())
	}
	wantOutput(t, out.String(),
		"This folder is inside another git repository. Create a separate one for your notes? [Y/n]",
		"Start? [Y/n]", "was removed")
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Errorf("the folder the cancelled run created must go: %v", err)
	}
	if treeHash(t, outer) != before {
		t.Error("the cancelled setup changed the outer repository")
	}
}

// A missing folder the outer repository still tracks files in is
// refused, as an existing one is, and nothing is left behind.
func TestSyncSetupRefusesAMissingFolderTheOuterRepositoryTracks(t *testing.T) {
	t.Parallel()
	needGit(t)
	outer := t.TempDir()
	git(t, outer, "init", "-q", "-b", "main")
	writeFile(t, outer, "notes/a.md", "a\n")
	git(t, outer, "add", "-A")
	git(t, outer, "commit", "-q", "-m", "notes")
	dir := filepath.Join(outer, "notes")
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	cfg := filepath.Join(t.TempDir(), "config.json")
	code := syncSetup(context.Background(), []string{"--dir", dir}, strings.NewReader(""), &out, &out, cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1 (refused):\n%s", code, out.String())
	}
	wantOutput(t, out.String(), "tracks files in this folder")
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Errorf("the refused setup left the folder: %v", err)
	}
}

// Saying no to a repository of its own leaves a missing folder missing:
// looking at it creates nothing.
func TestSyncSetupDeclinedForAMissingFolderCreatesNothing(t *testing.T) {
	t.Parallel()
	needGit(t)
	outer := t.TempDir()
	git(t, outer, "init", "-q", "-b", "main")
	dir := filepath.Join(outer, "notes")
	var out bytes.Buffer
	cfg := filepath.Join(t.TempDir(), "config.json")
	code := syncSetup(context.Background(), []string{"--dir", dir}, strings.NewReader("n\n"), &out, &out, cfg)
	if code != 1 {
		t.Fatalf("exit %d, want 1 (cancelled):\n%s", code, out.String())
	}
	wantOutput(t, out.String(), "Setup cancelled; nothing was changed.")
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Errorf("the folder was created: %v", err)
	}
}

// unsafeText is what setup and init say about the home folder, the root
// of a disk or a system folder.
const unsafeText = "is your home folder, the root of a disk or a system folder: sync a folder of its own for your notes, such as ~/Notes (misoku sync setup --dir ~/Notes)."

// `misoku sync setup` and `misoku sync init` refuse the home folder (it
// would push ~/.ssh too), the root of a disk and a system folder, and
// change nothing.
func TestSyncSetupAndInitRefuseTheHomeFolderTheRootAndASystemFolder(t *testing.T) {
	home := t.TempDir()
	writeFile(t, home, ".ssh/id_x", "secret")
	writeFile(t, home, "todo.md", "a note in the home folder\n")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
	system, _ := systemExample(t)
	ctx := context.Background()
	for _, dir := range []string{home, rootOf(home), system} {
		cfg := filepath.Join(t.TempDir(), "config.json")
		var out bytes.Buffer
		if code := syncSetup(ctx, []string{"--dir", dir}, strings.NewReader(""), &out, &out, cfg); code != 1 {
			t.Errorf("sync setup %s: exit %d, want 1", dir, code)
		}
		wantOutput(t, out.String(), dir+" "+unsafeText)
		var stdout, stderr bytes.Buffer
		if code := syncInit(ctx, []string{filepath.Join(t.TempDir(), "none.git"), "--dir", dir}, &stdout, &stderr, cfg); code != 1 {
			t.Errorf("sync init %s: exit %d, want 1:\n%s%s", dir, code, stdout.String(), stderr.String())
		}
		wantOutput(t, stderr.String(), dir+" "+unsafeText)
		if _, err := os.Stat(cfg); !os.IsNotExist(err) {
			t.Errorf("%s: the config was written: %v", dir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".git")); !os.IsNotExist(err) {
		t.Errorf("the home folder became a repository: %v", err)
	}
}
