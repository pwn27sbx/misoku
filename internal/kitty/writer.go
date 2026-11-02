package kitty

import (
	"io"
	"sync"
)

// Writer is the program's only output: Bubble Tea writes whole frames through
// it, and graphics go through Send, which holds the lock across every chunk of
// one transmit. A frame therefore lands entirely before or entirely after a
// transmit, never inside it — the protocol requires the chunks of an image to
// arrive back to back.
type Writer struct {
	mu sync.Mutex
	w  io.Writer
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
		if _, err := io.WriteString(w.w, s); err != nil {
			return err
		}
	}
	return nil
}
