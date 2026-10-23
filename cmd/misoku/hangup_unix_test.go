//go:build darwin || linux

// Closing the terminal pane or window Misoku runs in hangs it up with
// SIGHUP (issue #22). Here main() runs on a pseudo-terminal, as in a
// terminal window, and closing the window is closing the pty's master:
// the kernel then sends SIGHUP, as it does for a real one.

package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// terminal is main() running on a pseudo-terminal in a synced vault, its
// opening sync done.
type terminal struct {
	cmd                 *exec.Cmd
	master              *os.File
	home, vault, remote string
	slow                string // holding seconds, makes git push wait that long
	mu                  sync.Mutex
	screen              bytes.Buffer  // everything drawn so far
	exited              chan struct{} // closed once Misoku has exited
	err                 error         // how it exited; read once exited is closed
}

func openTerminal(t *testing.T) *terminal {
	t.Helper()
	needGit(t)
	home := t.TempDir()
	tm := &terminal{home: home, vault: filepath.Join(home, "notes"), remote: newRemote(t), exited: make(chan struct{})}
	writeFile(t, tm.vault, "a.md", "a\n")
	if code, out, errOut := initVault(t, tm.vault, tm.remote, configIn(home)); code != 0 {
		t.Fatalf("sync init: exit %d\n%s\n%s", code, out, errOut)
	}
	// git, but a push waits while the file slow says so.
	bin := t.TempDir()
	tm.slow = filepath.Join(bin, "slow")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nfor a in \"$@\"; do\n\tif [ \"$a\" = push ] && [ -f '" + tm.slow + "' ]; then sleep \"$(cat '" + tm.slow + "')\"; fi\ndone\nexec '" + realGit + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	master, slave := openPTY(t)
	tm.master = master
	if err := unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 40, Col: 120}); err != nil {
		t.Fatal(err)
	}
	tm.cmd = exec.Command(os.Args[0], "-test.run=^$")
	tm.cmd.Env = append(os.Environ(), mainArgsEnv+"="+tm.vault, "MISOKU_WORKSPACE=", "TERM=xterm-256color",
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+home, "XDG_CACHE_HOME="+home, "XDG_CONFIG_HOME="+home)
	tm.cmd.Stdin, tm.cmd.Stdout, tm.cmd.Stderr = slave, slave, slave
	// Misoku leads its own session with the pty as its terminal, so the
	// kernel hangs it up when the window closes.
	tm.cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	if err := tm.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	slave.Close()
	go func() {
		buf := make([]byte, 4096)
		answered := false
		for {
			n, err := master.Read(buf)
			tm.mu.Lock()
			tm.screen.Write(buf[:n])
			asked := strings.Contains(tm.screen.String(), "\x1b[6n")
			tm.mu.Unlock()
			if err != nil {
				return
			}
			// Answer the background colour and cursor queries at start
			// as a terminal does; unanswered, they wait 5 s.
			if asked && !answered {
				answered = true
				master.Write([]byte("\x1b]11;rgb:0000/0000/0000\x1b\\\x1b[1;1R"))
			}
		}
	}()
	go func() { tm.err = tm.cmd.Wait(); close(tm.exited) }()
	t.Cleanup(func() {
		tm.cmd.Process.Kill()
		master.Close()
	})
	tm.waitFor(t, "Synced")
	return tm
}

// waitFor waits until the screen has shown text.
func (tm *terminal) waitFor(t *testing.T, text string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		tm.mu.Lock()
		seen := strings.Contains(tm.screen.String(), text)
		tm.mu.Unlock()
		if seen {
			return
		}
		select {
		case <-tm.exited:
			if !strings.Contains(tm.drawn(), text) {
				t.Fatalf("misoku exited (%v) before showing %q:\n%q", tm.err, text, tm.drawn())
			}
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("misoku never showed %q:\n%q", text, tm.drawn())
}

func (tm *terminal) drawn() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.screen.String()
}

// closeWindow closes the terminal Misoku runs in.
func (tm *terminal) closeWindow(t *testing.T) {
	t.Helper()
	if err := tm.master.Close(); err != nil {
		t.Fatal(err)
	}
}

// exit waits for Misoku to end and fails when a signal killed it.
func (tm *terminal) exit(t *testing.T) {
	t.Helper()
	select {
	case <-tm.exited:
		var ee *exec.ExitError
		if errors.As(tm.err, &ee) {
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				t.Fatalf("misoku was killed by %v", ws.Signal())
			}
		}
	case <-time.After(30 * time.Second):
		t.Fatal("misoku did not exit")
	}
}

// logged is Misoku's debug log.
func (tm *terminal) logged(t *testing.T) string {
	t.Helper()
	cache := tm.home
	if runtime.GOOS == "darwin" {
		cache = filepath.Join(tm.home, "Library", "Caches")
	}
	data, err := os.ReadFile(filepath.Join(cache, "misoku", "misoku.log"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Closing the window leaves through the normal quit path: what was saved
// is pushed, and as nobody can read the terminal any more, the outcome
// goes to the log.
func TestClosingTheTerminalPushesWhatWasSaved(t *testing.T) {
	t.Parallel()
	tm := openTerminal(t)
	writeFile(t, tm.vault, "closed.md", "saved before the window closed\n")
	tm.closeWindow(t)
	tm.exit(t)
	if got := remoteFiles(t, tm.remote)["closed.md"]; got != "saved before the window closed\n" {
		t.Fatalf("closed.md on the remote = %q: closing the window must push it", got)
	}
	wantOutput(t, tm.logged(t), "Pushing changes…", "Changes pushed.")
}

// The window closes while the quit sync pushes: the hang-up is no Ctrl+C,
// the push goes on and finishes.
func TestClosingTheTerminalDuringTheQuitSyncLetsItFinish(t *testing.T) {
	t.Parallel()
	tm := openTerminal(t)
	writeFile(t, tm.vault, "late.md", "late\n")
	writeFile(t, filepath.Dir(tm.slow), "slow", "2")
	tm.master.Write([]byte{3}) // Ctrl+C: quit
	tm.waitFor(t, "Pushing changes…")
	tm.closeWindow(t)
	tm.exit(t)
	if got := remoteFiles(t, tm.remote)["late.md"]; got != "late\n" {
		t.Fatalf("late.md on the remote = %q: closing the window must not stop the push", got)
	}
}

// SIGTERM during the quit sync still stops it at once, as Ctrl+C does:
// the commit waits for the next sync.
func TestSIGTERMDuringTheQuitSyncStopsIt(t *testing.T) {
	t.Parallel()
	tm := openTerminal(t)
	writeFile(t, tm.vault, "late.md", "late\n")
	writeFile(t, filepath.Dir(tm.slow), "slow", "30")
	tm.master.Write([]byte{3})
	tm.waitFor(t, "Pushing changes…")
	start := time.Now()
	if err := tm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	tm.waitFor(t, "Not pushed, the next sync tries again: cancelled")
	tm.exit(t)
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("SIGTERM took %s to stop the quit sync", took)
	}
	if _, ok := remoteFiles(t, tm.remote)["late.md"]; ok {
		t.Fatal("a cancelled quit sync must not have pushed late.md")
	}
}
