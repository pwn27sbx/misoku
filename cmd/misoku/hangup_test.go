package main

// What main does once its screen has closed (issue #22). The real
// SIGHUP path runs in hangup_unix_test.go; here leave gets a p.Run
// result directly, as a closed terminal can make Run fail.

import (
	"bytes"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
)

// pushed records what leave asked the quit sync to do.
type pushed struct {
	called bool
	w      io.Writer
}

func (p *pushed) push(_ app.Model, w io.Writer) { p.called, p.w = true, w }

// Reading the closed terminal fails just before its SIGHUP arrives: the
// quit sync still runs, writing to the log, and nothing goes to the
// terminal.
func TestLeavePushesWhenRunFailsBecauseTheTerminalClosed(t *testing.T) {
	t.Parallel()
	hup := make(hangup)
	time.AfterFunc(50*time.Millisecond, func() { close(hup) })
	var stderr bytes.Buffer
	var p pushed
	code := leave(app.Model{}, errors.New("read /dev/tty: input/output error"), hup, &stderr, p.push)
	if !p.called {
		t.Fatal("the quit sync must run after the terminal closed")
	}
	if p.w != log.Writer() {
		t.Errorf("the quit sync must write to the log, not to the closed terminal")
	}
	if stderr.Len() != 0 {
		t.Errorf("nothing may be written to the closed terminal: %q", stderr.String())
	}
	if code != 1 {
		t.Errorf("exit %d, want 1 for Run's error", code)
	}
}

// Without a hang-up, an error from Run is reported and skips the quit
// sync, as before.
func TestLeaveReportsARunErrorWithoutAHangup(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	var p pushed
	code := leave(app.Model{}, errors.New("boom"), make(hangup), &stderr, p.push)
	if p.called {
		t.Error("an error without a hang-up must not run the quit sync")
	}
	if code != 1 || stderr.String() != "Error running Misoku application: boom\n" {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
}

// Quitting from inside pushes and says so on the terminal, as before.
func TestLeaveAfterQuittingPushesOnTheTerminal(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	var p pushed
	code := leave(app.Model{}, nil, make(hangup), &stderr, p.push)
	if !p.called || p.w != io.Writer(&stderr) || code != 0 {
		t.Errorf("pushed %v on %v (want the terminal), exit %d", p.called, p.w, code)
	}
}
