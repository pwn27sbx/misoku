//go:build darwin || linux

// The cursor shape is a terminal protocol, so it is checked where it is used:
// main() runs on a pseudo-terminal, as in a terminal window, and the test reads
// the bytes it writes. A block after opening a note, a bar after i, the
// terminal's own cursor back on the way out.

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// cursorTerminal is main() running on a pseudo-terminal in a plain notes
// folder, with no sync.
type cursorTerminal struct {
	cmd    *exec.Cmd
	master *os.File
	mu     sync.Mutex
	screen bytes.Buffer
	exited chan struct{}
	err    error
}

func openCursorTerminal(t *testing.T) *cursorTerminal {
	t.Helper()
	home := t.TempDir()
	vault := filepath.Join(home, "notes")
	if err := os.MkdirAll(vault, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, "a.md"), []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return openCursorTerminalIn(t, home, vault)
}

// openCursorTerminalIn runs main() in home, on the notes folder vault.
func openCursorTerminalIn(t *testing.T, home, vault string) *cursorTerminal {
	t.Helper()
	master, slave := openPTY(t)
	tm := &cursorTerminal{master: master, exited: make(chan struct{})}
	if err := unix.IoctlSetWinsize(int(slave.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 40, Col: 120}); err != nil {
		t.Fatal(err)
	}
	tm.cmd = exec.Command(os.Args[0], "-test.run=^$")
	tm.cmd.Env = append(os.Environ(), mainArgsEnv+"=", "MISOKU_WORKSPACE="+vault,
		"TERM=xterm-256color", "HOME="+home, "XDG_CACHE_HOME="+home, "XDG_CONFIG_HOME="+home)
	tm.cmd.Stdin, tm.cmd.Stdout, tm.cmd.Stderr = slave, slave, slave
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
			// The terminal answers the colour and cursor queries at start;
			// unanswered, they wait 5 s.
			asked := strings.Contains(tm.screen.String(), "\x1b[6n")
			tm.mu.Unlock()
			if err != nil {
				return
			}
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
	return tm
}

func (tm *cursorTerminal) drawn() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.screen.String()
}

// waitFor waits until the screen shows text.
func (tm *cursorTerminal) waitFor(t *testing.T, text string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(tm.drawn(), text) {
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

// waitForExit waits for Misoku to end.
func (tm *cursorTerminal) waitForExit(t *testing.T) {
	t.Helper()
	select {
	case <-tm.exited:
	case <-time.After(30 * time.Second):
		t.Fatal("misoku did not exit")
	}
}

// The whole rule, from the bytes a terminal reads: NORMAL shows a block on the
// editor cell, INSERT a bar, and quitting gives the terminal its own cursor
// back so the shell is left with the default one.
func TestTheRealCursorFollowsTheModeAndIsRestored(t *testing.T) {
	t.Parallel()
	tm := openCursorTerminal(t)
	tm.waitFor(t, "a.md")

	tm.master.Write([]byte("1")) // open the note
	tm.waitFor(t, "\x1b[2 q")
	block := tm.drawn()
	if !strings.Contains(block, "\x1b[?25h") {
		t.Fatalf("the cursor was never shown:\n%q", block)
	}
	if !strings.Contains(block, "\x1b[2;38H") {
		t.Fatalf("NORMAL did not place the cursor on the editor cell:\n%q", block)
	}

	tm.master.Write([]byte("i"))
	tm.waitFor(t, "\x1b[6 q")

	tm.master.Write([]byte{3}) // Ctrl+C: quit
	tm.waitForExit(t)
	if got := tm.drawn(); !strings.HasSuffix(got, "\x1b[0 q\x1b[?25h") {
		t.Fatalf("the terminal cursor was not restored on the way out: %q", got[max(0, len(got)-80):])
	}
}

// ui.cursor_shape "block" keeps today's look: the terminal's cursor is never
// touched.
func TestCursorShapeBlockLeavesTheTerminalAlone(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	cfg := configIn(home)
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"ui": {"cursor_shape": "block"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	vault := filepath.Join(home, "notes")
	if err := os.MkdirAll(vault, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, "a.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tm := openCursorTerminalIn(t, home, vault)
	tm.waitFor(t, "a.md")
	tm.master.Write([]byte("1"))
	tm.waitFor(t, "hello")
	tm.master.Write([]byte{3})
	tm.waitForExit(t)

	for _, seq := range []string{"\x1b[2 q", "\x1b[6 q", "\x1b[0 q"} {
		if strings.Contains(tm.drawn(), seq) {
			t.Fatalf("ui.cursor_shape=block still wrote %q", seq)
		}
	}
}
