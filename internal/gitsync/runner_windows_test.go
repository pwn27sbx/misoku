//go:build windows

package gitsync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// TestHelperHangingChild is not a test: git runs this test binary as its
// hung child (a stand-in for ssh.exe) when GITSYNC_HANG_PIDFILE is set. It
// writes its own Windows pid and sleeps.
func TestHelperHangingChild(t *testing.T) {
	pidFile := os.Getenv("GITSYNC_HANG_PIDFILE")
	if pidFile == "" {
		return
	}
	tmp := pidFile + ".tmp"
	if os.WriteFile(tmp, []byte(strconv.Itoa(os.Getpid())), 0o644) == nil {
		os.Rename(tmp, pidFile)
	}
	time.Sleep(60 * time.Second)
	os.Exit(0)
}

// On Windows, a timeout kills git's children too (through the job object):
// a hung ssh.exe left behind would keep the connection open, and a new one
// would pile up on every sync. The limit doubles until the child got to
// start, so the result never depends on how loaded the machine is.
func TestTimeoutKillsGitsChildrenOnWindows(t *testing.T) {
	isolate(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	t.Setenv("GITSYNC_HANG_PIDFILE", pidFile)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	hang := `alias.hang=!"` + filepath.ToSlash(self) + `" '-test.run=^TestHelperHangingChild$'`
	pid := 0
	for timeout := time.Second; pid <= 0; timeout *= 2 {
		if timeout > 30*time.Second {
			t.Fatal("the hung child never started")
		}
		_, err := gitsync.New(newRepo(t), timeout).Run(context.Background(), "-c", hang, "hang")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected a timeout, got %v", err)
		}
		if b, rerr := os.ReadFile(pidFile); rerr == nil {
			pid, _ = strconv.Atoi(strings.TrimSpace(string(b)))
		}
	}
	h, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		return // already gone, nothing left to open
	}
	defer windows.CloseHandle(h)
	if ev, _ := windows.WaitForSingleObject(h, 5000); ev != windows.WAIT_OBJECT_0 {
		windows.TerminateProcess(h, 1)
		t.Fatalf("child %d of git is still running after the timeout", pid)
	}
}
