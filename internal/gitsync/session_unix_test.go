//go:build !windows

package gitsync_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
func TestTimeoutKillsGitsChildren(t *testing.T) {
	isolate(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	r := gitsync.New(newRepo(t), 300*time.Millisecond)
	_, err := r.Run(context.Background(), "-c", "alias.hang=!sh -c 'echo $$ > "+pidFile+"; exec sleep 30'", "hang")
	if err == nil {
		t.Fatal("expected a timeout")
	}
	b, rerr := os.ReadFile(pidFile)
	if rerr != nil {
		t.Fatalf("the hung child never started: %v", rerr)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("child %d of git is still running after the timeout", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
