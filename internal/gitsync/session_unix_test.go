//go:build !windows

package gitsync_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// A new session has no controlling terminal: this holds even where the test
// itself has none, unlike opening /dev/tty. A new session is also a new
// process group, and the group id is what ps reports reliably everywhere
// (macOS prints 0 for "sess").
func TestRunnerStartsGitInItsOwnSession(t *testing.T) {
	isolate(t)
	ours, err := syscall.Getpgid(0)
	if err != nil {
		t.Fatal(err)
	}
	r := gitsync.New(newRepo(t), 5*time.Second)
	out, err := r.Run(context.Background(), "-c", `alias.sid=!ps -o pgid= -p $$`, "sid")
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("could not read the process group from %q", out)
	}
	if theirs == ours {
		t.Fatalf("git ran in Misoku's process group (%d), so in its session: it could reach the terminal", ours)
	}
}

// On timeout every process git started dies too: a hung ssh left behind
// would keep a connection and a lock on the repository.
//
// The child has to be running when the limit hits, and on a loaded machine
// starting git and a shell can take longer than any fixed limit. So the
// limit starts short and doubles until the child got to write its pid:
// every attempt still ends by timeout, never by the test stopping git.
func TestTimeoutKillsGitsChildren(t *testing.T) {
	isolate(t)
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	// Written to a temporary name and renamed, so a pid file that exists is
	// complete: a half-written one would read as pid 0, and kill(0, ...)
	// targets the test's own process group.
	hang := "alias.hang=!sh -c 'echo $$ > " + pidFile + ".tmp && mv " + pidFile + ".tmp " + pidFile + "; exec sleep 30'"
	pid := 0
	for timeout := 300 * time.Millisecond; pid <= 0; timeout *= 2 {
		if timeout > 20*time.Second {
			t.Fatal("the hung child never started")
		}
		r := gitsync.New(newRepo(t), timeout)
		_, err := r.Run(context.Background(), "-c", hang, "hang")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected a timeout, got %v", err)
		}
		if b, rerr := os.ReadFile(pidFile); rerr == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("child %d of git is still running after the timeout", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// fakeProgram writes an executable shell script named name into dir.
func fakeProgram(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// End to end: git really runs the program from GIT_SSH (a path with a
// space in it), and that program gets plink's no-prompt flag.
func TestGitRunsTheUsersGitSSHProgram(t *testing.T) {
	isolate(t)
	unsetenv(t, "GIT_SSH_COMMAND")
	unsetenv(t, "GIT_SSH_VARIANT")
	dir := filepath.Join(t.TempDir(), "my tools")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(t.TempDir(), "args")
	t.Setenv("GIT_SSH", fakeProgram(t, dir, "plink", `printf '%s\n' "$@" > "`+argsFile+`"; exit 1`+"\n"))
	r := gitsync.New(newRepo(t), 5*time.Second)
	if _, err := r.Run(context.Background(), "ls-remote", "ssh://git@example.invalid/notes.git"); err == nil {
		t.Fatal("the fake plink fails, so ls-remote must fail")
	}
	b, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("git did not run the GIT_SSH program: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(b)), "\n")
	if args[0] != "-batch" || strings.Contains(string(b), "BatchMode") {
		t.Fatalf("plink must get -batch first and no OpenSSH options, got %q", args)
	}
}

// Reading the user's ssh setting costs a git process; it is done once per
// Runner, not once per command, even when commands run at the same time.
func TestSSHSettingIsReadOncePerRunner(t *testing.T) {
	isolate(t)
	unsetenv(t, "GIT_SSH_COMMAND")
	real, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin, log := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	fakeProgram(t, bin, "git", `echo "$*" >> "`+log+`"; exec "`+real+`" "$@"`+"\n")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	r := gitsync.New(newRepo(t), 5*time.Second)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.Run(context.Background(), "status", "--porcelain"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(b), " config "); n != 1 {
		t.Fatalf("the ssh setting was read %d times for 4 commands, want 1:\n%s", n, b)
	}
}
