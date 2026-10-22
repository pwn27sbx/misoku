// Package syncout wraps the terminal output so the terminal paints each
// frame at once (synchronized output, DEC private mode 2026) instead of
// showing it half drawn while it arrives.
package syncout

import (
	"io"
	"os"
	"sync"
)

const (
	// begin and end bracket one synchronized update. A terminal that does
	// not know mode 2026 ignores both.
	begin = "\x1b[?2026h"
	end   = "\x1b[?2026l"
)

// Writer is f with every Write wrapped in begin … end and sent to the
// terminal as a single Write.
//
// It embeds *os.File so it still satisfies term.File: Bubble Tea finds the
// TTY through it (window size, raw mode, Windows VT processing). Only Write
// is wrapped. Methods promoted from *os.File bypass the markers, notably
// WriteString, which io.WriteString uses: Bubble Tea's renderer sends loose
// sequences that way (standardRenderer.execute) and they need no wrapping,
// while every frame goes through Write (standardRenderer.flush).
//
// Programs run with tea.ExecProcess inherit the program output as their
// stdout when they set none; since Writer is not an *os.File, os/exec would
// hand them a pipe instead of the terminal. Such commands must set
// cmd.Stdout themselves.
type Writer struct {
	*os.File
	out io.Writer // f itself; a test records the writes here

	mu  sync.Mutex
	buf []byte // reused between frames
}

// New wraps f.
func New(f *os.File) *Writer { return &Writer{File: f, out: f} }

// ForTerminal is the output to give Bubble Tea for f: the wrapper, or f
// itself when MISOKU_NO_SYNC is "1", for a terminal that misdraws the
// markers.
func ForTerminal(f *os.File, getenv func(string) string) io.Writer {
	if getenv("MISOKU_NO_SYNC") == "1" {
		return f
	}
	return New(f)
}

// Write sends p to the terminal between the markers, in one Write. It
// reports how many bytes of p were written, not counting the markers.
func (w *Writer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf[:0], begin...)
	w.buf = append(w.buf, p...)
	w.buf = append(w.buf, end...)
	n, err := w.out.Write(w.buf)
	n = min(max(n-len(begin), 0), len(p))
	if err == nil && n < len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}
