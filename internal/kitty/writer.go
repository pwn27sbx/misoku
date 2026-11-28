package kitty

import (
	"io"
	"strings"
	"sync"
)

// Writer is the program's only output: Bubble Tea writes whole frames through
// it, and graphics go through Send, which holds the lock across every chunk of
// one transmit. A frame therefore lands entirely before or entirely after a
// transmit, never inside it — the protocol requires the chunks of an image to
// arrive back to back.
//
// It also forwards Read, Close and Fd to the writer underneath when that one is
// a terminal (the sync output wrapper embeds *os.File for the same reason):
// Bubble Tea finds the TTY through the program output, and a wrapper that hides
// it would leave the program without a window size.
type Writer struct {
	mu sync.Mutex
	w  io.Writer
	// passthrough wraps every transmit for tmux's passthrough DCS: only
	// graphics are wrapped, frames go out as they are.
	passthrough bool
}

// SetPassthrough turns tmux passthrough on or off. Call it once, before any
// output.
func (w *Writer) SetPassthrough(on bool) { w.passthrough = on }

// WrapPassthrough wraps one kitty sequence in tmux's passthrough DCS: the
// payload's own escape bytes are doubled, which is what tmux expects, and the
// sequence ends the DCS itself.
func WrapPassthrough(seq string) string {
	var b strings.Builder
	b.Grow(len(seq)*2 + 10)
	b.WriteString("\x1bPtmux;")
	for i := 0; i < len(seq); i++ {
		if seq[i] == 0x1b {
			b.WriteString("\x1b\x1b")
			continue
		}
		b.WriteByte(seq[i])
	}
	b.WriteString("\x1b\\")
	return b.String()
}

// NewWriter wraps w. A nil w discards everything, which keeps tests and
// non-terminal runs harmless.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Write writes one frame (or anything else Bubble Tea emits) under the lock.
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.w == nil {
		return len(p), nil
	}
	return w.w.Write(p)
}

// Send writes seqs back to back while no frame can slip between them. Every
// element must be a complete escape sequence.
func (w *Writer) Send(seqs ...string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.w == nil {
		return nil
	}
	for _, s := range seqs {
		if s == "" {
			continue
		}
		if w.passthrough {
			s = WrapPassthrough(s)
		}
		if _, err := io.WriteString(w.w, s); err != nil {
			return err
		}
	}
	return nil
}

// Fd is the terminal's file descriptor, or -1 when the output is not a file.
// Bubble Tea asks for it to read the window size.
func (w *Writer) Fd() uintptr {
	if f, ok := w.w.(interface{ Fd() uintptr }); ok {
		return f.Fd()
	}
	return ^uintptr(0)
}

// Read exists so the wrapper still is the terminal Bubble Tea expects to find
// (io.ReadWriteCloser). Input never comes from the program output, so it only
// reports the end of the stream.
func (w *Writer) Read([]byte) (int, error) { return 0, io.EOF }

// Close exists for the same reason and never closes the terminal: Bubble Tea may
// call it while leaving, and the real file outlives the program output.
func (w *Writer) Close() error { return nil }
