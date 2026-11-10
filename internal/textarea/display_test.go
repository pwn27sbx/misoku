package textarea_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/textarea"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func raw() textarea.DisplayLine { return textarea.DisplayLine{Raw: true} }

func rendered(rows ...string) textarea.DisplayLine { return textarea.DisplayLine{Rows: rows} }

func hidden() textarea.DisplayLine { return textarea.DisplayLine{Hidden: true} }

// viewRows splits a View into its screen rows with the escape codes removed.
func viewRows(m textarea.Model) []string {
	return strings.Split(ansi.Strip(m.View()), "\n")
}

// trimmedRows returns the View's rows with the padding removed.
func trimmedRows(m textarea.Model) []string {
	rows := viewRows(m)
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return rows
}

func equalRows(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) < len(want) {
		t.Fatalf("got %d rows %q, want at least %q", len(got), got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("row %d = %q, want %q (all rows %q)", i, got[i], w, got)
		}
	}
}

func TestDisplayDrawsRowsExactly(t *testing.T) {
	m := frameEditor(20, 6, false, "alpha\nbeta\ngamma")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{
		rendered("A1", "A2", "A3"),
		raw(),
		rendered("G1"),
	}})
	equalRows(t, trimmedRows(m), "A1", "A2", "A3", "beta", "G1", "~")
}

func TestDisplayRowsGetNoCursorOrSelection(t *testing.T) {
	trueColour(t)
	m := frameEditor(20, 3, false, "alpha\nbeta")
	m.Focus()
	m.Cursor.SetMode(0)
	m.SelectionStyle = lipgloss.NewStyle().Background(lipgloss.Color("#112233"))
	m.SetRowCol(0, 1)
	m.SetAnchor(0, 0)
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("RENDERED"), rendered("OTHER")}})
	first := strings.Split(m.View(), "\n")[0]
	if !strings.Contains(first, "RENDERED") {
		t.Fatalf("row 0 = %q, want the rendered row in one piece", first)
	}
	if strings.Contains(first, "48;2;17;34;51") {
		t.Fatalf("row 0 carries the selection colour: %q", first)
	}
}

func TestDisplayZeroRowLineTakesNoSpaceAndIsALanding(t *testing.T) {
	m := frameEditor(20, 4, false, "alpha\nbeta\ngamma")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), rendered(), raw()}})
	equalRows(t, trimmedRows(m), "alpha", "gamma", "~")
	m.MoveCursorBy(1)
	if row, _ := m.RowCol(); row != 1 {
		t.Fatalf("j from line 0 landed on %d, want the zero-row line 1", row)
	}
	m.MoveCursorBy(1)
	if row, _ := m.RowCol(); row != 2 {
		t.Fatalf("j from the zero-row line landed on %d, want 2", row)
	}
}

func TestDisplayHiddenLinesSteppedOverGoingDown(t *testing.T) {
	m := frameEditor(20, 4, false, "head\nbody1\nbody2\ntail")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), hidden(), hidden(), raw()}})
	m.MoveCursorBy(1)
	if row, _ := m.RowCol(); row != 3 {
		t.Fatalf("j landed on %d, want 3", row)
	}
	m.MoveCursorBy(1)
	if row, _ := m.RowCol(); row != 3 {
		t.Fatalf("j at the last line moved to %d", row)
	}
}

func TestDisplayHiddenLinesSteppedOverGoingUp(t *testing.T) {
	m := frameEditor(20, 4, false, "head\nbody1\nbody2\ntail")
	m.SetRowCol(3, 0)
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), hidden(), hidden(), raw()}})
	m.MoveCursorBy(-1)
	if row, _ := m.RowCol(); row != 0 {
		t.Fatalf("k landed on %d, want 0", row)
	}
}

func TestDisplayHiddenAtTheEdgesIsNoLanding(t *testing.T) {
	m := frameEditor(20, 4, false, "a\nb\nc")
	m.SetRowCol(1, 0)
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{hidden(), raw(), hidden()}})
	m.MoveCursorBy(1)
	if row, _ := m.RowCol(); row != 1 {
		t.Fatalf("j into a hidden tail moved to %d", row)
	}
	m.MoveCursorBy(-1)
	if row, _ := m.RowCol(); row != 1 {
		t.Fatalf("k into a hidden head moved to %d", row)
	}
}

func TestDisplayFooterDrawnAndCounted(t *testing.T) {
	m := frameEditor(20, 3, false, "alpha\nbeta")
	m.SetDisplay(&textarea.Display{
		Lines:  []textarea.DisplayLine{raw(), rendered("B")},
		Footer: []string{"F1", "F2", "F3"},
	})
	equalRows(t, trimmedRows(m), "alpha", "B", "F1")
	m.ScrollBy(2)
	if off := m.YOffset(); off != 2 {
		t.Fatalf("YOffset = %d, want 2: footer rows must count as scrollable", off)
	}
	equalRows(t, trimmedRows(m), "F1", "F2", "F3")
}

func TestDisplayScrollCannotLeaveTheCursorAboveAShortFooter(t *testing.T) {
	m := frameEditor(20, 4, false, "alpha\nbeta")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), raw()}, Footer: []string{"F1", "F2"}})
	m.ScrollBy(100)
	if off := m.YOffset(); off != 1 {
		t.Fatalf("YOffset = %d, want 1: the last text row stays on screen", off)
	}
	if _, _, ok := m.CursorScreen(); !ok {
		t.Fatal("the cursor left the screen")
	}
}

func TestDisplayTallFooterIsReachable(t *testing.T) {
	m := frameEditor(20, 2, false, "alpha")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw()}, Footer: []string{"F1", "F2", "F3", "F4"}})
	m.ScrollBy(100)
	if off := m.YOffset(); off != 3 {
		t.Fatalf("YOffset = %d, want 3", off)
	}
	equalRows(t, trimmedRows(m), "F3", "F4")
}

func TestDisplayRowAtRawLine(t *testing.T) {
	m := frameEditor(10, 4, false, "aaaa bbbb cccc dddd\nb")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), raw()}})
	if line, offset, footer := m.DisplayRowAt(1); line != 0 || offset != 1 || footer {
		t.Fatalf("got (%d, %d, %v), want (0, 1, false)", line, offset, footer)
	}
}

func TestDisplayRowAtRenderedLine(t *testing.T) {
	m := frameEditor(20, 4, false, "a\nb\nc")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), rendered("x", "y"), raw()}})
	if line, offset, footer := m.DisplayRowAt(2); line != 1 || offset != 1 || footer {
		t.Fatalf("got (%d, %d, %v), want (1, 1, false)", line, offset, footer)
	}
	if line, offset, footer := m.DisplayRowAt(3); line != 2 || offset != 0 || footer {
		t.Fatalf("got (%d, %d, %v), want (2, 0, false)", line, offset, footer)
	}
}

func TestDisplayRowAtFooter(t *testing.T) {
	m := frameEditor(20, 4, false, "a\nb")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), raw()}, Footer: []string{"F1", "F2"}})
	if line, offset, footer := m.DisplayRowAt(3); line != 1 || offset != 1 || !footer {
		t.Fatalf("got (%d, %d, %v), want (1, 1, true)", line, offset, footer)
	}
}

func TestLineRowsCountsRowsAndFooter(t *testing.T) {
	m := frameEditor(20, 8, false, "a\nb\nc")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), rendered("x", "y", "z"), hidden()}, Footer: []string{"F1", "F2"}})
	for _, c := range []struct{ line, first, n int }{{0, 0, 1}, {1, 1, 3}, {2, 4, 0}, {3, 4, 2}} {
		if first, n := m.LineRows(c.line); first != c.first || n != c.n {
			t.Fatalf("line %d: got (%d, %d), want (%d, %d)", c.line, first, n, c.first, c.n)
		}
	}
}

func TestDisplayRowAtPastTheEnd(t *testing.T) {
	m := frameEditor(20, 8, false, "a\nb")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), raw()}, Footer: []string{"F1"}})
	line, offset, footer := m.DisplayRowAt(5)
	if line != 1 || footer || offset < 1 {
		t.Fatalf("got (%d, %d, %v), want line 1, not footer, offset past the line's rows", line, offset, footer)
	}
}

func TestDisplayRowAtNegativeIsTheFirstRow(t *testing.T) {
	m := frameEditor(20, 4, false, "a\nb")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("x", "y"), raw()}})
	if line, offset, footer := m.DisplayRowAt(-3); line != 0 || offset != 0 || footer {
		t.Fatalf("got (%d, %d, %v), want (0, 0, false)", line, offset, footer)
	}
}

func TestDisplayRowAtWithoutDisplayFollowsTheWrap(t *testing.T) {
	m := frameEditor(10, 4, false, "aaaa bbbb cccc dddd\nb")
	if line, offset, footer := m.DisplayRowAt(1); line != 0 || offset != 1 || footer {
		t.Fatalf("got (%d, %d, %v), want (0, 1, false)", line, offset, footer)
	}
}

func TestDisplayClickRawRow(t *testing.T) {
	m := frameEditor(20, 5, false, "alpha\nbeta")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("x", "y"), raw()}})
	m.ClickAt(2, 2)
	if row, col := m.RowCol(); row != 1 || col != 2 {
		t.Fatalf("got (%d, %d), want (1, 2)", row, col)
	}
}

func TestDisplayClickRenderedRowLandsOnItsLine(t *testing.T) {
	m := frameEditor(20, 8, false, "a\nb\nc")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), rendered("1", "2", "3", "4"), raw()}})
	m.ClickAt(5, 4)
	if row, col := m.RowCol(); row != 1 || col != 0 {
		t.Fatalf("got (%d, %d), want (1, 0)", row, col)
	}
}

func TestDisplayClickFooterLandsOnLastLine(t *testing.T) {
	m := frameEditor(20, 5, false, "a\nbb")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), raw()}, Footer: []string{"F"}})
	m.ClickAt(1, 2)
	if row, col := m.RowCol(); row != 1 || col != 2 {
		t.Fatalf("got (%d, %d), want (1, 2)", row, col)
	}
}

func TestDisplayClickFooterSkipsHiddenTail(t *testing.T) {
	m := frameEditor(20, 5, false, "a\nb\nc")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), hidden(), hidden()}, Footer: []string{"F"}})
	m.ClickAt(0, 1)
	if row, _ := m.RowCol(); row != 0 {
		t.Fatalf("row = %d, want the last line that is not hidden", row)
	}
}

func TestDisplayClickPastTheEndLandsOnLastLine(t *testing.T) {
	m := frameEditor(20, 8, false, "a\nbb")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{raw(), rendered("x", "y")}})
	m.ClickAt(0, 7)
	if row, _ := m.RowCol(); row != 1 {
		t.Fatalf("row = %d, want 1", row)
	}
}

func TestDisplayLengthMismatchIsIgnored(t *testing.T) {
	plain := frameEditor(20, 4, false, "alpha\nbeta")
	want := plain.View()
	m := frameEditor(20, 4, false, "alpha\nbeta")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("x")}, Footer: []string{"F"}})
	if got := m.View(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if line, offset, footer := m.DisplayRowAt(1); line != 1 || offset != 0 || footer {
		t.Fatalf("got (%d, %d, %v), want the raw (1, 0, false)", line, offset, footer)
	}
}

func TestDisplayTextEditBreakingTheLengthFallsBackToRaw(t *testing.T) {
	m := frameEditor(20, 5, false, "alpha\nbeta")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("x"), rendered("y")}})
	m.OpenLineBelow()
	equalRows(t, trimmedRows(m), "alpha", "", "beta")
}

// numbered returns n lines "LA", "LB", and so on.
func numbered(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "L" + string(rune('A'+i))
	}
	return strings.Join(lines, "\n")
}

func TestDisplayTopAnchorWhenALineAboveGrows(t *testing.T) {
	m := frameEditor(20, 3, false, numbered(8))
	m.SetRowCol(5, 0)
	m.SetRowCol(3, 0) // the cursor at the top row, so EnsureVisible cannot hide a wrong offset
	lines := make([]textarea.DisplayLine, 8)
	for i := range lines {
		lines[i] = raw()
	}
	m.SetDisplay(&textarea.Display{Lines: lines})
	top := trimmedRows(m)[0]
	before := m.YOffset()

	grown := append([]textarea.DisplayLine(nil), lines...)
	grown[0] = rendered("1", "2", "3", "4")
	m.SetDisplay(&textarea.Display{Lines: grown})
	if got := trimmedRows(m)[0]; got != top {
		t.Fatalf("top row = %q, want %q", got, top)
	}
	if off := m.YOffset(); off != before+3 {
		t.Fatalf("YOffset = %d, want %d", off, before+3)
	}
}

func TestDisplayTopAnchorWhenALineAboveShrinks(t *testing.T) {
	m := frameEditor(20, 3, false, numbered(8))
	lines := make([]textarea.DisplayLine, 8)
	for i := range lines {
		lines[i] = raw()
	}
	lines[0] = rendered("1", "2", "3", "4")
	m.SetRowCol(6, 0)
	m.SetDisplay(&textarea.Display{Lines: lines})
	top := trimmedRows(m)[0]

	shrunk := append([]textarea.DisplayLine(nil), lines...)
	shrunk[0] = raw()
	m.SetDisplay(&textarea.Display{Lines: shrunk})
	if got := trimmedRows(m)[0]; got != top {
		t.Fatalf("top row = %q, want %q", got, top)
	}
}

func TestDisplayCursorVisibleAfterSetDisplay(t *testing.T) {
	m := frameEditor(20, 3, false, numbered(6))
	m.SetRowCol(5, 0)
	lines := make([]textarea.DisplayLine, 6)
	for i := range lines {
		lines[i] = rendered("a", "b", "c", "d")
	}
	lines[5] = raw()
	m.SetDisplay(&textarea.Display{Lines: lines})
	_, y, ok := m.CursorScreen()
	if !ok || y < 0 || y >= 3 {
		t.Fatalf("cursor at y=%d ok=%v, want it on screen", y, ok)
	}
	rows := trimmedRows(m)
	if rows[y] != "LF" {
		t.Fatalf("the cursor's row shows %q, want LF (rows %q)", rows[y], rows)
	}
}

func TestDisplayNilRestoresTodaysOutput(t *testing.T) {
	plain := frameEditor(20, 4, true, "aaaa bbbb cccc dddd eeee ffff\nbeta\ngamma")
	want := plain.View()
	m := frameEditor(20, 4, true, "aaaa bbbb cccc dddd eeee ffff\nbeta\ngamma")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("x"), hidden(), raw()}, Footer: []string{"F"}})
	if got := m.View(); got == want {
		t.Fatal("the display changed nothing")
	}
	m.SetDisplay(nil)
	if got := m.View(); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDisplayNilWithoutDisplayLeavesTheViewportAlone(t *testing.T) {
	m := frameEditor(20, 2, false, numbered(6))
	m.ScrollBy(3)
	row, col := m.RowCol()
	off := m.YOffset()
	m.SetDisplay(nil)
	if r, c := m.RowCol(); r != row || c != col || m.YOffset() != off {
		t.Fatalf("got row %d col %d off %d, want %d %d %d", r, c, m.YOffset(), row, col, off)
	}
}

func TestDisplayReplacementRebuildsTheRowIndex(t *testing.T) {
	m := frameEditor(20, 8, false, "a\nb")
	first := &textarea.Display{Lines: []textarea.DisplayLine{rendered("1"), raw()}}
	second := &textarea.Display{Lines: []textarea.DisplayLine{rendered("1", "2", "3"), raw()}}
	m.SetDisplay(first)
	if line, _, _ := m.DisplayRowAt(1); line != 1 {
		t.Fatalf("under the first display row 1 is line %d, want 1", line)
	}
	m.SetDisplay(second)
	if line, _, _ := m.DisplayRowAt(1); line != 0 {
		t.Fatalf("under the second display row 1 is line %d, want 0", line)
	}
}

func TestDisplayModelCopiesKeepTheirOwnGeometry(t *testing.T) {
	a := frameEditor(20, 8, false, "a\nb")
	b := a // shares the row index with a
	a.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("1"), raw()}})
	b.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("1", "2", "3"), raw()}})
	for range 2 {
		if line, _, _ := a.DisplayRowAt(1); line != 1 {
			t.Fatalf("copy a: row 1 is line %d, want 1", line)
		}
		if line, _, _ := b.DisplayRowAt(1); line != 0 {
			t.Fatalf("copy b: row 1 is line %d, want 0", line)
		}
	}
}

func TestDisplayScrollInsideATallRowDoesNotMoveTheCursor(t *testing.T) {
	m := frameEditor(20, 3, false, "a\nb")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("1", "2", "3", "4", "5", "6"), raw()}})
	m.ScrollBy(2)
	if row, _ := m.RowCol(); row != 0 {
		t.Fatalf("row = %d: the cursor's block is still on screen, it must not move", row)
	}
	if off := m.YOffset(); off != 2 {
		t.Fatalf("YOffset = %d, want 2", off)
	}
	equalRows(t, trimmedRows(m), "3", "4", "5")
}

func TestDisplayRowsArePaddedLikeRawRows(t *testing.T) {
	m := frameEditor(20, 2, false, "a")
	m.SetDisplay(&textarea.Display{Lines: []textarea.DisplayLine{rendered("x")}})
	if w := ansi.StringWidth(strings.Split(m.View(), "\n")[0]); w != 20 {
		t.Fatalf("row width = %d, want 20", w)
	}
}
