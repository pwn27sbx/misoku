package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// noteTextAt returns the screen cell of the first occurrence of text right
// of the explorer, where the note is drawn.
func noteTextAt(t *testing.T, v, text string) (x, y int) {
	t.Helper()
	for y, row := range strings.Split(v, "\n") {
		r := []rune(row)
		if len(r) <= app.SidebarWidth+1 {
			continue
		}
		pane := string(r[app.SidebarWidth+1:])
		if i := strings.Index(pane, text); i >= 0 {
			return app.SidebarWidth + 1 + len([]rune(pane[:i])), y
		}
	}
	t.Fatalf("%q not in the note:\n%s", text, v)
	return 0, 0
}

// clickTextAt presses the left button on the cell dx past the first
// occurrence of text in the note.
func clickTextAt(t *testing.T, h *harness, text string, dx int) {
	t.Helper()
	x, y := noteTextAt(t, h.view(), text)
	h.click(x+dx, y)
}

// typeAtCursor types X where the cursor is and returns the note's text.
func typeAtCursor(t *testing.T, h *harness) string {
	t.Helper()
	h.keys("iX")
	h.key(tea.KeyEsc)
	return saved(t, h)
}

// S5: a click on a word of a rendered paragraph lands on that word's column
// in the source, past the markers the paragraph hides.
func TestPrettyClickLandsOnTheWordsSourceColumn(t *testing.T) {
	h := prettyAt(t, "hello **bold** wonder\n\nend")
	clickTextAt(t, h, "wonder", 2)
	if got := typeAtCursor(t, h); got != "hello **bold** woXnder\n\nend" {
		t.Fatalf("the click should land inside wonder, got %q", got)
	}
}

// S5: on a paragraph of two source lines the click picks the line.
func TestPrettyClickPicksTheParagraphsSourceLine(t *testing.T) {
	h := prettyAt(t, "alpha beta\ngamma delta\n\nend")
	clickTextAt(t, h, "delta", 1)
	if got := typeAtCursor(t, h); got != "alpha beta\ngamma dXelta\n\nend" {
		t.Fatalf("the click should land in the second line, got %q", got)
	}
}

// S5: a click on a heading's text lands after its hidden "#".
func TestPrettyClickOnAHeadingLandsAfterTheMarker(t *testing.T) {
	h := prettyAt(t, "# Title\n\nend")
	clickTextAt(t, h, "Title", 0)
	if got := typeAtCursor(t, h); got != "# XTitle\n\nend" {
		t.Fatalf("the click should land on Title, got %q", got)
	}
}

// S6: a click in a rendered table cell opens the cell's menu, as in the
// split, and its edit changes that cell.
func TestPrettyClickInATableCell(t *testing.T) {
	h := prettyAt(t, "| head | col |\n|---|---|\n| one | two |\n\nend")
	clickTextAt(t, h, "two", 1)
	if v := h.view(); !strings.Contains(v, "Edit cell") || !strings.Contains(v, "Delete row") {
		t.Fatalf("the click should open the cell's menu:\n%s", v)
	}
	h.keys("1") // Edit cell
	for range "two" {
		h.key(tea.KeyBackspace)
	}
	h.keys("six")
	h.key(tea.KeyEnter)
	if got := saved(t, h); !strings.Contains(got, "| six |") || strings.Contains(got, "two") {
		t.Fatalf("the cell should change, got %q", got)
	}
}

// S6: a click on a box ticks it as one undo step, and neither moves the
// cursor nor starts a selection.
func TestPrettyClickTicksABox(t *testing.T) {
	look := preview.DefaultLook()
	h := prettyAt(t, "- [ ] alpha\n\nend")
	x, y := noteTextAt(t, h.view(), look.Unchecked)
	h.click(x, y)
	h.send(tea.MouseMsg{X: x + 4, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion})
	h.send(tea.MouseMsg{X: x + 4, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	if v := h.view(); mode(v) != "NORMAL" || strings.Contains(v, "Copied") {
		t.Fatalf("the click should start no selection:\n%s", v)
	}
	if got := typeAtCursor(t, h); got != "- [x] alpha\n\nXend" {
		t.Fatalf("the click should tick the box and leave the cursor on end, got %q", got)
	}
	h.keys("uu") // the X, then the box
	if got := saved(t, h); got != "- [ ] alpha\n\nend" {
		t.Fatalf("one u should undo the box, got %q", got)
	}
}

// S6: a click on a [[link]] follows it, in this view.
func TestPrettyClickFollowsAWikiLink(t *testing.T) {
	h := openStart(t, linkedNotes(t))
	h.key(tea.KeyEsc)
	h.keys(" p")
	h.keys("G")
	clickTextAt(t, h, "about Docker", 7)
	if v := h.view(); !strings.Contains(v, "containers") || !inReadingView(v) {
		t.Fatalf("the click should open Docker.md in the view:\n%s", v)
	}
}

// S6: a click on a backlink under the note opens the note that links here.
func TestPrettyClickOnAFooterBacklink(t *testing.T) {
	h := openStart(t, linkedNotes(t))
	h.key(tea.KeyEsc)
	h.keys(" p")
	h.keys("G")
	clickTextAt(t, h, "about Docker", 7)
	clickTextAt(t, h, "󰈙 Start", 2)
	if v := h.view(); !strings.Contains(v, "Learn about") || strings.Contains(v, "containers") {
		t.Fatalf("the backlink should open Start.md:\n%s", v)
	}
}

// S6: the fold arrow folds and unfolds, and earns the tutorial's credit; the
// calendar's next button turns the month.
func TestPrettyClickFoldsAndTurnsTheCalendar(t *testing.T) {
	steps := []tutor.Step{
		actionStep("preview_fold", "collapse"),
		actionStep("preview_fold", "expand"),
	}
	h, p, l := eventFixture(t, "# Uno\n\ntexto\n\n# Dos\n\nend", steps)
	h.keys(" p")
	h.keys("G")
	clickTextAt(t, h, "▾", 0)
	if v := h.view(); strings.Contains(v, "texto") || !strings.Contains(v, "▸") {
		t.Fatalf("the arrow should fold Uno:\n%s", v)
	}
	assertStep(t, p, l, 1)
	clickTextAt(t, h, "▸", 0)
	if v := h.view(); !strings.Contains(v, "texto") {
		t.Fatalf("the arrow should unfold Uno:\n%s", v)
	}
	assertStep(t, p, l, 2)

	h = prettyAt(t, "```calendar\nmonth: 2026-10\n```\n\nend")
	clickTextAt(t, h, "›", 0)
	if v := h.view(); !strings.Contains(v, "November 2026") {
		t.Fatalf("the next button should show November:\n%s", v)
	}
}

// S6: a form's choice makes the same edit as in the split.
func TestPrettyClickAnswersAForm(t *testing.T) {
	h := prettyAt(t, "```form\ntitle: Survey\n? Pick (choice): A | B\n```\n\nend")
	clickTextAt(t, h, "○ B", 0)
	if got := saved(t, h); !strings.Contains(got, "? Pick (choice): A | B\n  = B\n```") {
		t.Fatalf("the choice should be stored, got %q", got)
	}
}

// S9: a click while a typed edit's compile waits acts on the current text.
func TestPrettyClickFlushesAPendingCompile(t *testing.T) {
	look := preview.DefaultLook()
	h := openNoteWith(t, longNote("first\n\n- [ ] alpha"))
	h.keys(" p")
	h.keys("A")
	compiles := app.PreviewCompiles(h.m)
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	if got := app.PreviewCompiles(h.m); got != compiles {
		t.Fatalf("setup: the Enter should wait for its compile")
	}
	clickTextAt(t, h, look.Unchecked, 0)
	if got := saved(t, h); !strings.HasPrefix(got, "first\n\n\n- [x] alpha\n") {
		t.Fatalf("the click should tick alpha, got %q", got[:40])
	}
}

// S5: a drag that starts on a rendered row selects from the mapped column.
func TestPrettyDragSelectsFromTheMappedColumn(t *testing.T) {
	h := prettyAt(t, "hello **bold** wonder\n\nend")
	bx, by := noteTextAt(t, h.view(), "bold")
	wx, wy := noteTextAt(t, h.view(), "wonder")
	h.click(bx+1, by)
	h.send(tea.MouseMsg{X: wx + 2, Y: wy, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion})
	h.send(tea.MouseMsg{X: wx + 2, Y: wy, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	// From the o of bold to the n of wonder, both in: "old** won".
	if v := h.view(); !strings.Contains(v, "Copied 9 characters") {
		t.Fatalf("the drag should copy from bold to wonder:\n%s", v)
	}
}

// S8: a wheel scroll that leaves the cursor on screen keeps its block raw.
func TestPrettyWheelKeepsTheReveal(t *testing.T) {
	var b strings.Builder
	for i := range 60 {
		b.WriteString("**p" + string(rune('A'+i%26)) + string(rune('a'+i/26)) + "**\n\n")
	}
	b.WriteString("end")
	h := prettyAt(t, b.String())
	h.keys("gg")
	h.keys(strings.Repeat("j", 20)) // well inside the screen
	v := h.view()
	x, y := noteTextAt(t, v, "**")
	raw := []rune(strings.Split(v, "\n")[y])[x : x+7] // "**pXy**"
	h.send(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	after := h.view()
	if strings.Count(after, "**") != 2 || !strings.Contains(after, string(raw)) || after == v {
		t.Fatalf("the wheel should scroll and keep %q raw:\n%s\nwas:\n%s", string(raw), after, v)
	}
}

// S8: a search into a folded body unfolds it and lands there.
func TestPrettySearchIntoAFoldUnfoldsIt(t *testing.T) {
	h := prettyAt(t, "# Sec\n\nhidden word\n\n# Next\n\nend")
	clickTextAt(t, h, "▾", 0)
	if v := h.view(); strings.Contains(v, "hidden word") {
		t.Fatalf("setup: Sec should fold:\n%s", v)
	}
	h.keys("/hidden")
	h.key(tea.KeyEnter)
	if got := typeAtCursor(t, h); !strings.Contains(got, "Xhidden word") {
		t.Fatalf("the search should land on the hidden line, got %q", got)
	}
	h.keys("G")
	if v := h.view(); !strings.Contains(v, "hidden word") || strings.Contains(v, "▸") {
		t.Fatalf("the search should unfold Sec:\n%s", v)
	}
}

// S8: G into a folded body unfolds it.
func TestPrettyGotoIntoAFoldUnfoldsIt(t *testing.T) {
	h := prettyAt(t, "# Top\n\nfirst\n\n# Sec\n\nlast line")
	h.keys("gg")
	clickTextAt(t, h, "▾", 0) // Top is raw: the arrow is Sec's
	if v := h.view(); strings.Contains(v, "last line") {
		t.Fatalf("setup: Sec should fold:\n%s", v)
	}
	h.keys("G")
	h.keys("gg")
	if v := h.view(); !strings.Contains(v, "last line") || strings.Contains(v, "▸") {
		t.Fatalf("G should unfold Sec:\n%s", v)
	}
}

// S6: a press on a control during INSERT closes the typed text as its own
// change before the control edits, so one u undoes only the control.
func TestPrettyClickInInsertIsItsOwnUndoStep(t *testing.T) {
	look := preview.DefaultLook()
	h := openNoteWith(t, "- [ ] alpha\n\nend")
	h.keys(" p")
	h.keys("G")
	h.keys("A typed")
	x, y := noteTextAt(t, h.view(), look.Unchecked)
	h.click(x, y)
	h.key(tea.KeyEsc)
	if got := saved(t, h); got != "- [x] alpha\n\nend typed" {
		t.Fatalf("setup %q", got)
	}
	h.keys("u")
	if got := saved(t, h); got != "- [ ] alpha\n\nend typed" {
		t.Fatalf("one u should undo only the box, got %q", got)
	}
}

// S6: a press on a control ends INSERT, as the same press in the split does
// by moving the focus to the preview.
func TestPrettyClickInInsertReturnsToNormal(t *testing.T) {
	look := preview.DefaultLook()
	h := openNoteWith(t, "- [ ] alpha\n\nend")
	h.keys(" p")
	h.keys("G")
	h.keys("A typed")
	x, y := noteTextAt(t, h.view(), look.Unchecked)
	h.click(x, y)
	if got := mode(h.view()); got != "NORMAL" {
		t.Fatalf("a press on a control should leave INSERT, mode %q", got)
	}
}
