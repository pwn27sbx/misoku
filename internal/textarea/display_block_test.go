package textarea_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/textarea"

	"github.com/charmbracelet/x/ansi"
)

// The cursor's rendered block (S15): banded on all its rows, measured as one
// whole, and stepped over block by block.

// rowless is a line of a rendered block without rows of its own, whose rows
// line head holds.
func rowless(head int) textarea.DisplayLine { return textarea.DisplayLine{Head: head} }

// skipped is rowless and stepped over by j/k: a block's other line in NORMAL.
func skipped(head int) textarea.DisplayLine { return textarea.DisplayLine{Head: head, Hidden: true} }

const testBand = "48;2;48;48;48" // frameEditor's cursor-line background

// S15: with the cursor on a line without rows, every row of its block takes
// the band, padded to the width, and no other block's row does.
func TestDisplayBandOnEveryRowOfTheCursorsBlock(t *testing.T) {
	trueColour(t)
	m := frameEditor(20, 6, false, "a\nb\nc\nd")
	m.SetRowCol(1, 0)
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{
		rendered("A1", "A2", "A3"), rowless(0), rendered("C1"), raw(),
	}})
	rows := strings.Split(m.View(), "\n")
	for y := 0; y < 3; y++ {
		if !strings.Contains(rows[y], testBand) {
			t.Fatalf("row %d of the cursor's block has no band: %q", y, rows[y])
		}
		if w := ansi.StringWidth(rows[y]); w != 20 {
			t.Fatalf("row %d is %d wide, want 20", y, w)
		}
	}
	if strings.Contains(rows[3], testBand) {
		t.Fatalf("another block's row took the band: %q", rows[3])
	}
}

// S15: the band is the focused editor's: a blurred one draws none.
func TestDisplayNoBandWithoutFocus(t *testing.T) {
	trueColour(t)
	m := frameEditor(20, 6, false, "a\nb")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("A1", "A2"), raw()}})
	m.Blur()
	if v := m.View(); strings.Contains(v, testBand) {
		t.Fatalf("a blurred editor banded its rendered block: %q", v)
	}
}

// S15: the cursor on a line without rows is on its block's rows, not on the
// next block's.
func TestDisplayCursorOnARowlessLineIsOnItsBlock(t *testing.T) {
	m := frameEditor(20, 6, false, "a\nb\nc")
	m.SetRowCol(1, 0)
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{
		rendered("A1", "A2", "A3"), rowless(0), rendered("C1"),
	}})
	if _, y, ok := m.CursorScreen(); !ok || y != 0 {
		t.Fatalf("cursor at y=%d ok=%v, want the block's first row 0", y, ok)
	}
	if !m.CursorRendered() {
		t.Fatal("the cursor sits on a rendered block")
	}
}

// S15: bringing the cursor into view from above, on a line without rows of
// a tall block, shows the whole block, not the rows after it.
func TestDisplayEnsureVisibleKeepsATallBlockWhole(t *testing.T) {
	m := frameEditor(20, 3, false, numbered(9))
	lines := make([]textarea.DisplayLine, 9)
	for i := range lines {
		lines[i] = raw()
	}
	lines[5] = rendered("B1", "B2", "B3")
	lines[6] = rowless(5)
	m.SetDisplay(&textarea.Display{Lines: lines})
	m.SetRowCol(6, 0)
	m.EnsureVisible()
	equalRows(t, trimmedRows(m), "B1", "B2", "B3")
}

// S15: k from a block's other line goes to the block before, not to its own
// first line; j goes to the next block.
func TestDisplayBlockwiseStepsLeaveTheBlock(t *testing.T) {
	m := frameEditor(20, 6, false, "a\nb\nc\nd")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{
		rendered("A"), rendered("B1", "B2"), skipped(1), rendered("D"),
	}})
	m.SetRowCol(2, 0)
	m.MoveCursorBy(-1)
	if row, _ := m.RowCol(); row != 0 {
		t.Fatalf("k from line 2 landed on %d, want the block before, 0", row)
	}
	m.MoveCursorBy(1)
	m.MoveCursorBy(1)
	if row, _ := m.RowCol(); row != 3 {
		t.Fatalf("jj from line 0 landed on %d, want 3", row)
	}
}

// S15: DeleteLines takes a block's lines out at once and leaves the cursor at
// the start of what follows; taking every line leaves one empty line.
func TestDeleteLines(t *testing.T) {
	m := frameEditor(20, 6, false, "a\nb\nc\nd")
	m.SetRowCol(2, 1)
	m.DeleteLines(1, 3)
	if got, row := m.Value(), m.Line(); got != "a\nd" || row != 1 {
		t.Fatalf("got %q with the cursor on %d, want \"a\\nd\" on 1", got, row)
	}
	m.DeleteLines(1, 9)
	if got, row := m.Value(), m.Line(); got != "a" || row != 0 {
		t.Fatalf("deleting the tail left %q with the cursor on %d", got, row)
	}
	m.DeleteLines(0, 1)
	if got := m.Value(); got != "" || m.LineCount() != 1 {
		t.Fatalf("deleting every line left %q in %d lines", got, m.LineCount())
	}
}

// cutDisplay is a three-row screen over "a".."f": the first two lines a
// block of two rows, the third a tall block of four, the rest raw.
func cutDisplay(t *testing.T) textarea.Model {
	t.Helper()
	m := frameEditor(20, 3, false, "a\nb\nc\nd\ne\nf")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{
		rendered("A1", "A2"), rowless(0), rendered("C1", "C2", "C3", "C4"), raw(), raw(), raw(),
	}})
	return m
}

// S8 (B1): moving onto a block the top edge cuts brings the whole block on
// screen, not only the rows still showing.
func TestDisplayEnsureVisibleShowsABlockCutByTheTop(t *testing.T) {
	m := cutDisplay(t)
	m.SetRowCol(3, 0) // d, row 6: rows 4-6 on screen
	m.ScrollBy(-3)    // rows 1-3, the cursor dragged onto the tall block
	if off := m.YOffset(); off != 1 {
		t.Fatalf("setup: YOffset = %d, want 1", off)
	}
	m.SetRowCol(1, 0)
	if off := m.YOffset(); off != 0 {
		t.Fatalf("YOffset = %d, want 0: the cursor's block starts on row 0", off)
	}
	equalRows(t, trimmedRows(m), "A1", "A2", "C1")
}

// S8 (B1): moving onto a block taller than the screen whose top is cut
// shows its top.
func TestDisplayEnsureVisibleShowsTheTopOfATallBlock(t *testing.T) {
	m := cutDisplay(t)
	m.SetRowCol(3, 0) // d, row 6: rows 4-6, C3 C4 d
	m.SetRowCol(2, 0)
	if off := m.YOffset(); off != 2 {
		t.Fatalf("YOffset = %d, want 2: the tall block's first row", off)
	}
	equalRows(t, trimmedRows(m), "C1", "C2", "C3")
}

// S8: wheel scrolling over a block taller than the screen does not bounce
// back once something brings the cursor into view again.
func TestDisplayWheelOverATallBlockDoesNotBounce(t *testing.T) {
	m := frameEditor(20, 3, false, "a\nb")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("1", "2", "3", "4", "5", "6"), raw()}})
	for want := 1; want <= 3; want++ {
		m.ScrollBy(1)
		m.EnsureVisible()
		if row, _ := m.RowCol(); row != 0 || m.YOffset() != want {
			t.Fatalf("wheel %d: cursor on %d, YOffset %d, want 0 and %d", want, row, m.YOffset(), want)
		}
	}
	m.ScrollBy(-1)
	m.EnsureVisible()
	if m.YOffset() != 2 {
		t.Fatalf("wheel up: YOffset %d, want 2", m.YOffset())
	}
}

// S8: the wheel leaving the cursor's block cut by the top edge moves the
// cursor to the first block below it that is whole on screen, so nothing
// scrolls back.
func TestDisplayWheelCuttingTheCursorsBlockMovesTheCursor(t *testing.T) {
	m := frameEditor(20, 3, false, "a\nb\nc\nd\ne")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{
		rendered("A1", "A2"), rowless(0), rendered("C1", "C2"), raw(), raw(),
	}})
	m.SetRowCol(1, 0)
	m.ScrollBy(1)
	if row, _ := m.RowCol(); row != 2 {
		t.Fatalf("the wheel left the cursor on line %d, want 2, the next whole block", row)
	}
	m.EnsureVisible()
	if off := m.YOffset(); off != 1 {
		t.Fatalf("YOffset = %d after EnsureVisible, want 1: no bounce", off)
	}
	m.ScrollBy(-1) // row 0-2: A1 A2 C1, C2 cut by the bottom
	if row, _ := m.RowCol(); row != 0 {
		t.Fatalf("the wheel up left the cursor on line %d, want 0", row)
	}
	m.EnsureVisible()
	if off := m.YOffset(); off != 0 {
		t.Fatalf("YOffset = %d after EnsureVisible, want 0", off)
	}
}

// S8: the wheel moving the cursor off a cut block skips a hidden line (a
// blank run in NORMAL) for the next block that is shown whole.
func TestDisplayWheelSkipsHiddenLines(t *testing.T) {
	m := frameEditor(20, 3, false, "a\nb\n\nd\ne")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{
		rendered("A1", "A2"), skipped(0), {Rows: []string{""}, Hidden: true}, rendered("D1"), raw(),
	}})
	m.ScrollBy(1) // rows 1-3: A2, the blank row, D1
	if row, _ := m.RowCol(); row != 3 {
		t.Fatalf("the wheel left the cursor on line %d, want 3, the block after the blank run", row)
	}
}
