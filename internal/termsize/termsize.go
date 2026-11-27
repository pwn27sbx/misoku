// Package termsize reads the terminal's own cell size in pixels, so the rich
// page can be drawn at the size the terminal really uses: one image pixel per
// screen pixel, and body text the size of the terminal's text.
package termsize

import (
	"strconv"
	"strings"
)

// Cells reports the terminal cell size in pixels for fd. ok is false when the
// terminal does not report pixel geometry (a pipe, Windows' console API, or a
// multiplexer that answers zeros): callers keep their default then.
func Cells(fd uintptr) (w, h int, ok bool) { return cells(fd) }

// Parse reads a "WxH" cell size ("10x19"), the form the config and the
// environment use for a terminal that does not report its pixels.
func Parse(v string) (w, h int, ok bool) {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(v)), func(r rune) bool { return r == 'x' || r == '×' })
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w < 4 || w > 64 || h < 8 || h > 128 {
		return 0, 0, false
	}
	return w, h, true
}

// FromWinsize turns a winsize into a cell size. Zero or negative pixel
// geometry means "unknown" (herdr and some multiplexers report it), and so
// does a cell outside the plausible range, so a bogus answer never shrinks or
// stretches the page.
func FromWinsize(cols, rows, xpix, ypix int) (int, int, bool) {
	if cols <= 0 || rows <= 0 || xpix <= 0 || ypix <= 0 {
		return 0, 0, false
	}
	w, h := xpix/cols, ypix/rows
	if w < 4 || w > 64 || h < 8 || h > 128 {
		return 0, 0, false
	}
	return w, h, true
}
