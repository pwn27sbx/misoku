package app_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The hover handles and block drag of the reading view (S14). Motion goes
// through the mouse filter, as in the running app.

const handlesNote = "First\n\nSecond\n\nThird"

// gripsShown reports whether a grip is drawn in the note pane.
func gripsShown(v string) bool { return strings.Contains(strings.Join(paneRows(v), "\n"), "⠿") }

// runeAt returns the rune drawn at screen cell (x, y).
func runeAt(v string, x, y int) string {
	r := []rune(strings.Split(v, "\n")[y])
	if x < 0 || x >= len(r) {
		return ""
	}
	return string(r[x])
}

// S14: hovering a rendered block draws "+" and "⠿" left of its first row,
// where the split draws them.
func TestPrettyHoverShowsTheHandles(t *testing.T) {
	h := prettyAt(t, handlesNote)
	x, y, _ := textAt(t, h.view(), "Second")
	h.sendFiltered(motionAt(x+2, y))
	v := h.view()
	if runeAt(v, x-2, y) != "⠿" || runeAt(v, x-4, y) != "+" {
		t.Fatalf("hovering Second should draw + and ⠿ left of it:\n%s", v)
	}
	if strings.Count(strings.Join(paneRows(v), "\n"), "⠿") != 1 {
		t.Fatalf("one block's handles only:\n%s", v)
	}
	if got := saved(t, h); got != handlesNote {
		t.Fatalf("drawing the handles changed the note: %q", got)
	}
}

// S14: the raw block has no handles, hovered or once the cursor reaches the
// hovered block.
func TestPrettyNoHandlesOnTheRawBlock(t *testing.T) {
	h := prettyAt(t, handlesNote) // the cursor is on Third: it is raw
	x, y, _ := textAt(t, h.view(), "Third")
	h.sendFiltered(motionAt(x+2, y))
	if v := h.view(); gripsShown(v) {
		t.Fatalf("the raw block should have no handles:\n%s", v)
	}
	x, y, _ = textAt(t, h.view(), "Second")
	h.sendFiltered(motionAt(x+2, y))
	if v := h.view(); !gripsShown(v) {
		t.Fatalf("hovering Second should show its handles:\n%s", v)
	}
	h.keys("kk") // onto Second, which goes raw under the pointer
	if v := h.view(); gripsShown(v) {
		t.Fatalf("a block the cursor reveals should lose its handles:\n%s", v)
	}
}

// S14: no handles while typing.
func TestPrettyNoHandlesInInsert(t *testing.T) {
	h := prettyAt(t, handlesNote)
	x, y, _ := textAt(t, h.view(), "Second")
	h.sendFiltered(motionAt(x+2, y))
	if v := h.view(); !gripsShown(v) {
		t.Fatalf("hovering Second should show its handles:\n%s", v)
	}
	h.keys("A")
	if v := h.view(); gripsShown(v) {
		t.Fatalf("INSERT should hide the handles:\n%s", v)
	}
	h.sendFiltered(motionAt(x+3, y), motionAt(x+2, y))
	if v := h.view(); gripsShown(v) || !strings.Contains(v, "INSERT") {
		t.Fatalf("hovering in INSERT should show no handles:\n%s", v)
	}
}

// S14: moving off the block, onto the raw block or out of the pane, clears
// the handles.
func TestPrettyHandlesClearWhenTheMouseLeaves(t *testing.T) {
	h := prettyAt(t, handlesNote)
	v := h.view()
	sx, sy, _ := textAt(t, v, "Second")
	tx, ty, _ := textAt(t, v, "Third")
	h.sendFiltered(motionAt(sx+2, sy))
	h.sendFiltered(motionAt(tx+2, ty))
	if v := h.view(); gripsShown(v) {
		t.Fatalf("moving onto the raw block should clear the handles:\n%s", v)
	}
	h.sendFiltered(motionAt(sx+2, sy))
	h.sendFiltered(motionAt(2, sy)) // the explorer
	if v := h.view(); gripsShown(v) {
		t.Fatalf("leaving the pane should clear the handles:\n%s", v)
	}
}

// S14: dragging a block's grip below the next block moves it, as the split
// does, and one u puts it back.
func TestPrettyDragTheGripMovesTheBlock(t *testing.T) {
	const note = handlesNote + "\n\nFourth" // the cursor on Fourth: Third is rendered
	h := prettyAt(t, note)
	v := h.view()
	fx, fy, _ := textAt(t, v, "First")
	_, ty, _ := textAt(t, v, "Third")
	h.sendFiltered(motionAt(fx+2, fy))
	h.sendFiltered(
		leftAt(fx-2, fy, tea.MouseActionPress),
		leftAt(fx-2, ty, tea.MouseActionMotion),
	)
	if v := h.view(); !strings.Contains(strings.Split(v, "\n")[ty-1], "━━━") {
		t.Fatalf("the drop line should show above Third:\n%s", v)
	}
	h.sendFiltered(leftAt(fx-2, ty, tea.MouseActionRelease))
	if got := saved(t, h); got != "Second\n\nFirst\n\nThird\n\nFourth" {
		t.Fatalf("First should move below Second, got %q", got)
	}
	h.keys("u")
	if got := saved(t, h); got != note {
		t.Fatalf("one u should undo the move, got %q", got)
	}
}

// S14: "+" opens the block menu below the block, as in the split.
func TestPrettyPlusOpensTheBlockMenu(t *testing.T) {
	h := prettyAt(t, handlesNote)
	x, y, _ := textAt(t, h.view(), "First")
	h.sendFiltered(motionAt(x+2, y))
	h.sendFiltered(leftAt(x-4, y, tea.MouseActionPress), leftAt(x-4, y, tea.MouseActionRelease))
	v := h.view()
	rows := strings.Split(v, "\n")
	if !strings.Contains(rows[len(rows)-1], "INSERT") || !strings.Contains(v, "Basic blocks") || strings.Count(rows[2], "│") != 1 {
		t.Fatalf("+ should open the block menu in the view:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.key(tea.KeyEsc)
	// The block menu's "/" on the new line after First, as the split's "+".
	if got := saved(t, h); got != "First\n/\n\nSecond\n\nThird" {
		t.Fatalf("+ should add a block after First, got %q", got)
	}
}

// S14: during a block drag the reveal stays as it was, even when a wheel
// scroll moves the cursor into another block.
func TestPrettyBlockDragFreezesTheReveal(t *testing.T) {
	var b strings.Builder
	for i := range 40 {
		fmt.Fprintf(&b, "**p%d**\n\n", i)
	}
	b.WriteString("end")
	h := prettyAt(t, b.String())
	h.keys("gg") // p0 raw
	x, y, _ := textAt(t, h.view(), "p2")
	h.sendFiltered(motionAt(x+1, y), leftAt(x-2, y, tea.MouseActionPress))
	for range 6 {
		h.send(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	}
	if v := h.view(); strings.Contains(strings.Join(paneRows(v), "\n"), "**") {
		t.Fatalf("the block drag should keep the reveal where it was:\n%s", v)
	}
	h.sendFiltered(leftAt(x-2, y, tea.MouseActionRelease))
	if v := h.view(); !strings.Contains(strings.Join(paneRows(v), "\n"), "**") {
		t.Fatalf("the release should reveal the cursor's block again:\n%s", v)
	}
}
