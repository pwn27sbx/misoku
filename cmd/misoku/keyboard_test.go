package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// cmdC is what Ghostty sends for Cmd+C while another program left the
// kitty keyboard protocol on: "c" with Super, as CSI u.
const cmdC = "\x1b[99;9u"

// Cmd+C typed before the answer (the real case: Cmd+C, then 1 and Enter)
// is not part of it: 1 is chosen with no complaint.
func TestAnEscapeSequenceBeforeTheAnswerIsIgnored(t *testing.T) {
	g := newGreeting(t)
	dir, code, out := g.run(t, cmdC+"1\n\n")
	if want := filepath.Join(g.home, "Notes"); dir != want || code != 0 {
		t.Fatalf("got %q exit %d, want %q to open:\n%q", dir, code, want, out)
	}
	if strings.Contains(out, "! Type") {
		t.Errorf("the escape sequence was taken as an answer:\n%q", out)
	}
}

// Cmd+C, Enter, then 1: the line with only Cmd+C is asked again, with no
// complaint, and does not cancel.
func TestAnAnswerOfOnlyEscapeSequencesIsAskedAgain(t *testing.T) {
	g := newGreeting(t)
	dir, code, out := g.run(t, cmdC+"\n1\n\n")
	if want := filepath.Join(g.home, "Notes"); dir != want || code != 0 {
		t.Fatalf("got %q exit %d, want %q to open:\n%q", dir, code, want, out)
	}
	if strings.Contains(out, "! Type") {
		t.Errorf("the escape sequence was taken as an answer:\n%q", out)
	}
	if n := strings.Count(out, "Number (Enter cancels)"); n != 2 {
		t.Errorf("asked %d times, want 2:\n%q", n, out)
	}
}

// A line of only escape sequences does not cancel the way Enter does: the
// question is asked again (here the input then ends).
func TestAnAnswerOfOnlyEscapeSequencesDoesNotCancel(t *testing.T) {
	g := newGreeting(t)
	_, _, out := g.run(t, cmdC+"\n")
	if n := strings.Count(out, "Number (Enter cancels)"); n != 2 {
		t.Errorf("asked %d times, want 2:\n%q", n, out)
	}
	if strings.Contains(out, "! Type") {
		t.Errorf("the escape sequence was taken as an answer:\n%q", out)
	}
}

// The questions of `misoku sync setup` read their answers the same way.
func TestSyncSetupQuestionsIgnoreEscapeSequences(t *testing.T) {
	var out bytes.Buffer
	st := &setupTerminal{in: bufio.NewReader(strings.NewReader(cmdC + "y\n")), out: &out}
	if !st.yes("Start?", false) {
		t.Errorf("answered y after Cmd+C, got no:\n%q", out.String())
	}
}

// keyModes reports whether out pushed the classic keyboard mode and then
// popped it, once each and in that order.
func keyModes(t *testing.T, out string) {
	t.Helper()
	push, pop := strings.Index(out, classicKeys), strings.Index(out, restoreKeys)
	if strings.Count(out, classicKeys) != 1 || strings.Count(out, restoreKeys) != 1 || push > pop {
		t.Errorf("want the classic keyboard pushed once, then popped once:\n%q", out)
	}
}

func TestTheWelcomeAsksInTheClassicKeyboardMode(t *testing.T) {
	g := newGreeting(t)
	var out bytes.Buffer
	runWelcome(context.Background(), welcomeOptions{
		Stdin: strings.NewReader("\n"), Out: &out, Err: &out,
		ConfigPath: g.cfg, Home: g.home, ClassicKeys: true,
	})
	keyModes(t, out.String())
	if !strings.HasPrefix(out.String(), classicKeys) || !strings.HasSuffix(out.String(), restoreKeys) {
		t.Errorf("the mode must wrap every question:\n%q", out.String())
	}
}

// Ctrl+C at a question still gives the keyboard mode back.
func TestCtrlCInTheWelcomeRestoresTheKeyboardMode(t *testing.T) {
	g := newGreeting(t)
	in, typed := io.Pipe()
	defer typed.Close()
	out := &lockedBuffer{}
	interrupt := make(chan os.Signal, 1)
	done := make(chan struct{})
	go func() {
		runWelcome(context.Background(), welcomeOptions{
			Stdin: in, Out: out, Err: out, ConfigPath: g.cfg, Home: g.home,
			Interrupt: interrupt, ClassicKeys: true,
		})
		close(done)
	}()
	deadline := time.Now().Add(30 * time.Second)
	for !strings.Contains(out.String(), "Number (Enter cancels)") {
		if time.Now().After(deadline) {
			t.Fatalf("the welcome never asked:\n%q", out.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	interrupt <- os.Interrupt
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatalf("Ctrl+C must end the welcome:\n%q", out.String())
	}
	keyModes(t, out.String())
}

func TestSyncSetupAsksInTheClassicKeyboardMode(t *testing.T) {
	var out bytes.Buffer
	cfg := filepath.Join(t.TempDir(), "config.json")
	runSetup(context.Background(), []string{"--dir", rootOf(t.TempDir())}, strings.NewReader(""), &out, &out, cfg, nil, true)
	keyModes(t, out.String())
}

// Without a terminal (a pipe, a file, CI) no keyboard mode is written.
func TestNoKeyboardModeWithoutATerminal(t *testing.T) {
	var out bytes.Buffer
	cfg := filepath.Join(t.TempDir(), "config.json")
	syncSetup(context.Background(), []string{"--dir", rootOf(t.TempDir())}, strings.NewReader(""), &out, &out, cfg)
	if strings.Contains(out.String(), "\x1b") {
		t.Errorf("an escape sequence was written without a terminal:\n%q", out.String())
	}
}

// The keyboard modes are written only to a terminal, and never on
// Windows, whose console without VT mode would print them.
func TestClassicKeysOnlyOnATerminalOutsideWindows(t *testing.T) {
	for _, tc := range []struct {
		goos     string
		terminal bool
		want     bool
	}{
		{"darwin", true, true},
		{"linux", true, true},
		{"windows", true, false},
		{"darwin", false, false},
	} {
		if got := classicKeysOn(tc.goos, tc.terminal); got != tc.want {
			t.Errorf("classicKeysOn(%s, %v) = %v, want %v", tc.goos, tc.terminal, got, tc.want)
		}
	}
}
