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
