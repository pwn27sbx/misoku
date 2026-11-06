package app_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// tallNote is a note taller than the editor and the preview (one paragraph
// per line), so text would reach both ends of the panes without the padding
// rows.
func tallNote() string {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	return strings.Join(lines, "\n\n")
}

// blankFrom reports whether screen row y holds nothing but spaces and pane
// dividers from column x to the right edge.
func blankFrom(v string, y, x int) bool {
	rows := strings.Split(v, "\n")
	r := []rune(rows[y])
	if x > len(r) {
		return true
	}
	return strings.Trim(string(r[x:]), " │") == ""
}

// assertNotePadding checks the empty row under the tab bar and above the
// statusline across the note's panes, and that the note's first line sits
// right under the top one.
func assertNotePadding(t *testing.T, v string) {
	t.Helper()
	_, y, paneX := editorTextAt(t, v, "line 1")
	if y != 2 {
		t.Fatalf("first note line on row %d, want 2 (tab bar, then one blank row):\n%s", y, v)
	}
	if !blankFrom(v, 1, paneX) {
		t.Fatalf("row 1 under the tab bar is not blank in the note's panes:\n%s", v)
	}
	if last := termH - 2; !blankFrom(v, last, paneX) {
		t.Fatalf("row %d above the statusline is not blank in the note's panes:\n%s", last, v)
	}
	// The text still uses the rows just above the bottom padding (one of the
	// last two: a blank line may separate paragraphs).
	if blankFrom(v, termH-3, paneX) && blankFrom(v, termH-4, paneX) {
		t.Fatalf("rows %d-%d should still hold note text:\n%s", termH-4, termH-3, v)
	}
}

func TestSplitPanesHaveABlankRowTopAndBottom(t *testing.T) {
	h := openNoteWith(t, tallNote())
	v := h.view()
	assertNotePadding(t, v)
	// Both panes, not only the editor: the preview's text starts on row 2.
	row := strings.Split(v, "\n")[2]
	if i := strings.LastIndex(row, "│"); i < 0 || strings.TrimSpace(row[i+len("│"):]) == "" {
		t.Fatalf("the preview should start on row 2:\n%s", v)
	}
}

func TestReadingModeHasABlankRowTopAndBottom(t *testing.T) {
	h := openNoteWith(t, tallNote())
	h.keys(" r")
	assertNotePadding(t, h.view())
}

// The cursor is never drawn on a padding row: going to the last line scrolls
// it into the rows above the bottom one.
func TestScrolledCursorStaysAboveTheBottomPadding(t *testing.T) {
	out, opt := newCursorOut()
	h := openWithCursor(t, tallNote(), opt)
	h.keys("G")
	frame := h.m.View()
	v := h.view()
	_, y, paneX := editorTextAt(t, v, "line 100")
	if !blankFrom(v, termH-2, paneX) {
		t.Fatalf("bottom padding row not blank after G:\n%s", v)
	}
	if y != termH-3 {
		t.Fatalf("last line on row %d after G, want %d:\n%s", y, termH-3, v)
	}
	x, cy := out.placed(t)
	if cy != y {
		t.Fatalf("real cursor on row %d, want %d (the line under it)", cy, y)
	}
	if got := cursorCellAt(t, frame, x, cy); got != "l" {
		t.Fatalf("real cursor on %q, want the l of line 100\n%s", got, v)
	}
}

// editorTextAt is textAt limited to the editor, the left half of the split:
// the preview may show the same words on another row.
func editorTextAt(t *testing.T, v, text string) (x, y, paneX int) {
	t.Helper()
	rows := strings.Split(v, "\n")
	for i, row := range rows {
		r := []rune(row)
		if len(r) > termW/2 {
			r = r[:termW/2]
		}
		if strings.Contains(string(r), text) {
			x, _, paneX = textAt(t, row, text)
			return x, i, paneX
		}
	}
	t.Fatalf("%q not in the editor:\n%s", text, v)
	return 0, 0, 0
}

func TestRealCursorSitsBelowTheTopPadding(t *testing.T) {
	out, opt := newCursorOut()
	openWithCursor(t, "hello\nworld", opt)
	if _, y := out.placed(t); y != 2 {
		t.Fatalf("real cursor on row %d, want 2 (tab bar, padding, line 1)", y)
	}
}

// A click maps to the line drawn under it, padding row included in the math.
func TestClickLandsOnTheLineUnderItWithPadding(t *testing.T) {
	h := openNoteWith(t, "one\ntwo\nthree")
	x, y, _ := editorTextAt(t, h.view(), "three")
	h.click(x+1, y)
	h.keys("iX")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	got, _ := h.store.Get("doc.md")
	if got.Content != "one\ntwo\ntXhree" {
		t.Fatalf("click on row %d should land on line 3, got %q", y, got.Content)
	}
}

func TestNotePaddingLayout(t *testing.T) {
	l := app.CalculateLayout(140, 40, true, true, false, app.Split{})
	if l.PadY != 1 || l.NoteY() != l.BodyY+1 || l.NoteH() != l.BodyH-2 {
		t.Fatalf("note area not padded by one row each side: %+v", l)
	}
	// A body too short to spare two rows keeps them for text.
	short := app.CalculateLayout(140, 9, true, true, false, app.Split{})
	if short.PadY != 0 || short.NoteH() != short.BodyH {
		t.Fatalf("a short body should drop the padding: %+v", short)
	}
}
