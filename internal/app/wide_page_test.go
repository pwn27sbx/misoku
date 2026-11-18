package app_test

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// The reading view on a wide terminal: prose grows to its cap, tables and
// code may use the right margin, and the mouse keeps working on them.

const wideW = 250

// wideCellText is a table cell wider than the 120-column prose page.
var wideCellText = strings.Repeat("w", 130)

// wideTableNote has a table whose last column, "zed", lies past the prose
// page's right edge on a 250-column terminal.
var wideTableNote = "| a | middle | last |\n|---|---|---|\n| 1 | " + wideCellText + " | zed |\n\nend"

// wideAt opens body in the reading view of a wideW-column terminal, with
// the cursor on its last line.
func wideAt(t *testing.T, body string) *harness {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", body); err != nil {
		t.Fatal(err)
	}
	h := newHarnessSize(t, wideW, termH, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys(" p")
	h.keys("G")
	return h
}

// wideView is the plain frame of a wideW-column harness.
func wideView(h *harness) string { return ansi.Strip(h.m.View()) }

// proseEdge is the screen column right after the prose page, found from the
// row of "end", which starts at the page's left margin.
func proseEdge(t *testing.T, h *harness) int {
	t.Helper()
	x, _, _ := textAt(t, wideView(h), "end")
	return x + 120
}

// S2, S4: on the reading view of a wide terminal a wide table is drawn whole,
// and a click on a cell past the prose page edits that cell.
func TestWideReadingViewClickACellPastTheProse(t *testing.T) {
	h := wideAt(t, wideTableNote)
	v := wideView(h)
	x, y, _ := textAt(t, v, "zed")
	if edge := proseEdge(t, h); x < edge {
		t.Fatalf("setup: zed (column %d) should be past the prose edge %d:\n%s", x, edge, v)
	}
	if row := strings.Split(v, "\n")[y]; !strings.Contains(row, wideCellText) || strings.Contains(row, "…") {
		t.Fatalf("the wide row should be drawn whole: %q", row)
	}
	h.click(x+1, y)
	if v := wideView(h); !strings.Contains(v, "Edit cell") {
		t.Fatalf("the click should open the cell's menu:\n%s", v)
	}
	h.keys("1") // Edit cell
	for range "zed" {
		h.key(tea.KeyBackspace)
	}
	h.keys("six")
	h.key(tea.KeyEnter)
	if got := saved(t, h); !regexp.MustCompile(`\| six +\|`).MatchString(got) || strings.Contains(got, "zed") {
		t.Fatalf("the cell past the prose should change, got %q", got)
	}
}

// S4: "+ Add column" under a widened table adds a column.
func TestWideReadingViewAddColumn(t *testing.T) {
	h := wideAt(t, wideTableNote)
	x, y, _ := textAt(t, wideView(h), "+ Add column")
	h.click(x+2, y)
	got := saved(t, h)
	if first := strings.SplitN(got, "\n", 2)[0]; strings.Count(first, "|") != 5 {
		t.Fatalf("the table should have four columns, got %q", got)
	}
}

// S4: hovering a widened table past the prose page shows its handles, and
// dragging its grip onto "end" moves it below "mid".
func TestWideReadingViewHoverAndDragAWideTable(t *testing.T) {
	h := wideAt(t, strings.Replace(wideTableNote, "\n\nend", "\n\nmid\n\nend", 1))
	v := wideView(h)
	if gripsShown(v) {
		t.Fatalf("setup: no handles before hovering:\n%s", v)
	}
	zx, zy, _ := textAt(t, v, "zed")
	h.sendFiltered(motionAt(zx+1, zy))
	v = wideView(h)
	gx, gy, _ := textAt(t, v, "⠿")
	if !gripsShown(v) {
		t.Fatalf("hovering the table past the prose should show its handles:\n%s", v)
	}
	_, ey, _ := textAt(t, v, "end")
	h.sendFiltered(
		leftAt(gx, gy, tea.MouseActionPress),
		leftAt(gx, ey, tea.MouseActionMotion),
		leftAt(gx, ey, tea.MouseActionRelease),
	)
	got := saved(t, h)
	if !strings.HasPrefix(got, "mid\n\n| a |") || !strings.HasSuffix(got, "zed |\n\nend") {
		t.Fatalf("the table should move between mid and end, got %q", got)
	}
}

// S6: the raw text INSERT reveals in the reading view wraps at the prose
// width, not at the wide width tables use.
func TestWideReadingViewRawLineWrapsAtProse(t *testing.T) {
	long := strings.Repeat("lorem ipsum dolor sit amet ", 30)
	h := wideAt(t, long+"\n\nend")
	h.keys("gg")
	h.keys("A")
	v := wideView(h)
	if !strings.Contains(v, "INSERT") {
		t.Fatalf("setup: want INSERT:\n%s", v)
	}
	widest := 0
	for _, row := range strings.Split(v, "\n") {
		if !strings.Contains(row, "lorem") {
			continue
		}
		_, pane, _ := strings.Cut(row, "│") // right of the explorer
		text := strings.TrimSpace(pane)
		widest = max(widest, ansi.StringWidth(text))
	}
	if widest > 120 || widest < 114 {
		t.Fatalf("the raw line should wrap at the 120-column prose, widest row %d:\n%s", widest, v)
	}
}

// S4: with the cursor on a wide table its rows keep their cells under the
// cursor-line band, and every frame row stays exactly the terminal's width.
func TestWideReadingViewCursorOnAWideTable(t *testing.T) {
	h := wideAt(t, wideTableNote)
	h.keys("gg") // the cursor onto the table, still rendered in NORMAL
	v := wideView(h)
	if !strings.Contains(v, wideCellText+" │ zed") {
		t.Fatalf("the table under the cursor should be drawn whole:\n%s", v)
	}
	for i, row := range strings.Split(h.m.View(), "\n") {
		if w := ansi.StringWidth(row); w != wideW {
			t.Fatalf("frame row %d is %d wide, want %d", i, w, wideW)
		}
	}
}

// S2: resizing the terminal recompiles at the new wide width: a narrower
// one cuts the wide cell, a wider one shows it whole again.
func TestWideReadingViewFollowsTheTerminalWidth(t *testing.T) {
	h := wideAt(t, wideTableNote)
	h.send(tea.WindowSizeMsg{Width: termW, Height: termH})
	if v := h.view(); strings.Contains(v, wideCellText) || !strings.Contains(v, "…") {
		t.Fatalf("at %d columns the wide cell should be cut:\n%s", termW, v)
	}
	h.send(tea.WindowSizeMsg{Width: wideW, Height: termH})
	if v := wideView(h); !strings.Contains(v, wideCellText) {
		t.Fatalf("back at %d columns the wide cell should be whole:\n%s", wideW, v)
	}
}
