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
		done <- runSetup(context.Background(), []string{"--dir", dir}, in, out, out, cfg, interrupt)
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
