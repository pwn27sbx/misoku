package syncout

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/charmbracelet/x/term"
)

// countingWriter records every Write it receives.
type countingWriter struct{ writes [][]byte }

func (c *countingWriter) Write(p []byte) (int, error) {
	c.writes = append(c.writes, append([]byte(nil), p...))
	return len(p), nil
}

func TestWriteWrapsOneFrameInOneWrite(t *testing.T) {
	rec := &countingWriter{}
	w := &Writer{out: rec}
	frame := []byte("\x1b[Hline one\r\nline two")

	n, err := w.Write(frame)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(frame) {
		t.Fatalf("Write returned %d, want %d (the caller's bytes)", n, len(frame))
	}
	if len(rec.writes) != 1 {
		t.Fatalf("got %d writes to the terminal, want 1", len(rec.writes))
	}
	want := "\x1b[?2026h" + string(frame) + "\x1b[?2026l"
	if got := string(rec.writes[0]); got != want {
		t.Fatalf("frame = %q, want %q", got, want)
	}

	// The buffer is reused: a second frame carries nothing of the first.
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatalf("second Write: %v", err)
	}
	if got := string(rec.writes[1]); got != "\x1b[?2026hx\x1b[?2026l" {
		t.Fatalf("second frame = %q", got)
	}
}

func TestWriterIsATerminalFile(t *testing.T) {
	// Bubble Tea finds the TTY (window size, Windows VT mode) only through
	// term.File; a plain io.Writer would lose it.
	var out io.Writer = ForTerminal(os.Stdout, func(string) string { return "" })
	f, ok := out.(term.File)
	if !ok {
		t.Fatalf("%T does not satisfy term.File", out)
	}
	if f.Fd() != os.Stdout.Fd() {
		t.Fatalf("Fd = %d, want stdout's %d", f.Fd(), os.Stdout.Fd())
	}
	if _, ok := out.(*Writer); !ok {
		t.Fatalf("ForTerminal returned %T, want *Writer when MISOKU_NO_SYNC is unset", out)
	}
}

func TestNoSyncPassesBytesThroughUnchanged(t *testing.T) {
	r, wf, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	env := func(k string) string {
		if k == "MISOKU_NO_SYNC" {
			return "1"
		}
		return ""
	}
	out := ForTerminal(wf, env)
	if out != io.Writer(wf) {
		t.Fatalf("MISOKU_NO_SYNC=1: got %T, want the file itself", out)
	}
	frame := []byte("\x1b[Hplain frame")
	if _, err := out.Write(frame); err != nil {
		t.Fatal(err)
	}
	wf.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, frame) {
		t.Fatalf("got %q, want %q unchanged", got, frame)
	}
}

func TestOtherNoSyncValuesKeepTheWrapper(t *testing.T) {
	for _, v := range []string{"", "0"} {
		env := func(string) string { return v }
		if _, ok := ForTerminal(os.Stdout, env).(*Writer); !ok {
			t.Errorf("MISOKU_NO_SYNC=%q: wrapper turned off", v)
		}
	}
}

func TestEmptyWriteSendsNothing(t *testing.T) {
	rec := &countingWriter{}
	w := &Writer{out: rec}
	if n, err := w.Write(nil); n != 0 || err != nil {
		t.Fatalf("Write(nil) = %d, %v", n, err)
	}
	if len(rec.writes) != 0 {
		t.Fatalf("empty write reached the terminal: %q", rec.writes)
	}
}
