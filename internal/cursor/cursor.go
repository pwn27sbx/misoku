// Package cursor draws the mode with the terminal's own cursor, like LazyVim:
// a block in NORMAL, VISUAL and VISUAL-LINE and a thin bar in INSERT, placed on
// the editor cell after every frame.
//
// Bubble Tea v1 hides nothing and has no way to place the cursor, so the
// program's output is wrapped: a frame is recognised by its leading cursor home
// (the alt-screen renderer writes one per frame) and gets the placement bytes
// appended in the same write, which keeps them inside the frame's synchronized
// update. Loose sequences pass through untouched. The shape is a DECSCUSR
// sequence, which a terminal that does not know it ignores.
package cursor

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"sync"
)

// The sequences. A terminal that ignores DECSCUSR keeps working.
const (
	show    = "\x1b[?25h"
	hide    = "\x1b[?25l"
	restore = "\x1b[0 q"
	// home is what Bubble Tea's alt-screen renderer writes first in every
	// frame (standardRenderer.flush).
	home = "\x1b[H"
)

// Shape is the cursor shape of a mode.
type Shape int

const (
	// Block is NORMAL, VISUAL and VISUAL-LINE: the thick cursor of LazyVim.
	Block Shape = iota
	// Bar is INSERT: a thin line between characters.
	Bar
)

// seq is the DECSCUSR sequence for s.
func (s Shape) seq() string {
	if s == Bar {
		return "\x1b[6 q"
	}
	return "\x1b[2 q"
}

// Enabled reports whether the real cursor is used: ui.cursor_shape, with
// MISOKU_CURSOR_SHAPE winning over it for one run. "block" turns the feature
// off (today's look); "auto", an empty or an unknown value keeps it on. Nothing
// is emitted when out is not a terminal or the terminal is TERM=dumb.
func Enabled(setting string, getenv func(string) string, isTTY bool) bool {
	if !isTTY {
		return false
	}
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if strings.EqualFold(strings.TrimSpace(getenv("TERM")), "dumb") {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(getenv("MISOKU_CURSOR_SHAPE")))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(setting))
	}
	return mode != "block"
}

// Writer is the program output with the real cursor on top: Set says where the
// cursor belongs and in which shape, Write adds the placement to every frame,
// and Restore gives the terminal its own cursor back.
//
// Every method holds the lock, so a frame, a placement and an image Send never
// interleave. It forwards Fd, Read and Close like the writers around it, so
// Bubble Tea still finds the TTY through it.
type Writer struct {
	mu  sync.Mutex
	out io.Writer
	buf []byte // reused between frames

	// What the app wants.
	x, y    int
	shape   Shape
	visible bool

	// What the terminal was last told. known is false before the shape was
	// ever written and after Restore, so the next visible state says it
	// again; hiding the cursor does not forget it, since the shape stays set
	// while the cursor is not drawn.
	known       bool
	lastX       int
	lastY       int
	lastShape   Shape
	lastVisible bool
}

// NewWriter wraps out. A nil out discards everything, which keeps tests and
// non-terminal runs harmless.
func NewWriter(out io.Writer) *Writer { return &Writer{out: out} }

// Set places the cursor on cell (x, y), 0-based from the frame's top-left
// corner, in shape s. visible false hides the terminal cursor: the editor is
// not focused, or another widget has its own drawn one.
//
// Only what changed is written, so a mode switch costs one DECSCUSR and a move
// one CUP. A state identical to the last one writes nothing, which is how a
// frame the renderer skips still leaves the cursor right.
func (w *Writer) Set(x, y int, s Shape, visible bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.out == nil {
		return
	}
	w.x, w.y, w.shape, w.visible = x, y, s, visible

	var b strings.Builder
	if visible {
		if !w.lastVisible {
			b.WriteString(show)
		}
		if !w.known || w.lastShape != s {
			b.WriteString(s.seq())
		}
		if !w.lastVisible || w.lastX != x || w.lastY != y {
			b.WriteString(cup(x, y))
		}
	} else if w.lastVisible {
		b.WriteString(hide)
	}
	if b.Len() > 0 {
		if n, err := w.out.Write([]byte(b.String())); err != nil || n < b.Len() {
			// The terminal did not take it: what it has is unknown, so the
			// next Set says it again.
			return
		}
	}
	w.mark(x, y, s, visible)
}

// Write sends p, adding the cursor placement when p is a frame and the cursor
// is visible. The frame and its placement are one write, so the placement lands
// inside the frame's synchronized update.
func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.out == nil {
		return len(p), nil
	}
	if !w.visible || !bytes.HasPrefix(p, []byte(home)) {
		return w.out.Write(p)
	}

	// The frame left the cursor on its last line: put it back on the editor
	// cell, and say the shape only when the terminal does not know it yet.
	w.buf = append(w.buf[:0], p...)
	if !w.known || !w.lastVisible {
		w.buf = append(w.buf, show...)
	}
	if !w.known || w.lastShape != w.shape {
		w.buf = append(w.buf, w.shape.seq()...)
	}
	w.buf = append(w.buf, cup(w.x, w.y)...)

	raw, err := w.out.Write(w.buf)
	n := min(raw, len(p))
	if err == nil && n < len(p) {
		err = io.ErrShortWrite
	}
	if err == nil && raw == len(w.buf) {
		// Only a write the terminal took whole says the placement happened.
		w.mark(w.x, w.y, w.shape, w.visible)
	}
	return n, err
}

// Restore gives the terminal its own cursor back: the default shape, visible.
// The program calls it when it stops and before handing the terminal to another
// program. The shape is forgotten and the cell is unknown, so the next Set says
// both again; the cursor stays visible, so hiding it afterwards is still said.
func (w *Writer) Restore() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	// The cursor is visible now, at a cell the program left it on, and its
	// shape is the terminal's own: nothing is known about either.
	w.known, w.lastX, w.lastY = false, -1, -1
	if w.out == nil {
		w.lastVisible = false
		return nil
	}
	n, err := w.out.Write([]byte(restore + show))
	w.lastVisible = err == nil && n == len(restore+show)
	return err
}

// mark records what the terminal now knows. The caller holds the lock. Only a
// visible cursor says the terminal has the shape: a hidden one was never given
// it, which is why the start screen's first hide leaves known false.
func (w *Writer) mark(x, y int, s Shape, visible bool) {
	w.lastX, w.lastY, w.lastVisible = x, y, visible
	if visible {
		w.known, w.lastShape = true, s
	}
}

// Forget drops what the writer believes the terminal has, without writing: the
// next Set says the shape, the visibility and the cell again. It is for when
// another program has used the terminal, which may have changed all three.
func (w *Writer) Forget() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.known, w.lastVisible, w.lastX, w.lastY = false, false, -1, -1
}

// Fd is the terminal's file descriptor, or -1 when the output is not a file.
func (w *Writer) Fd() uintptr {
	if f, ok := w.out.(interface{ Fd() uintptr }); ok {
		return f.Fd()
	}
	return ^uintptr(0)
}

// Read exists so the wrapper still is the terminal Bubble Tea expects to find
// (io.ReadWriteCloser). Input never comes from the program output.
func (w *Writer) Read([]byte) (int, error) { return 0, io.EOF }

// Close exists for the same reason and never closes the terminal: the real file
// outlives the program output.
func (w *Writer) Close() error { return nil }

// cup moves the cursor to cell (x, y), 0-based, as CUP is 1-based.
func cup(x, y int) string {
	return "\x1b[" + strconv.Itoa(y+1) + ";" + strconv.Itoa(x+1) + "H"
}
