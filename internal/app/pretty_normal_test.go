package app_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/cursor"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// NORMAL in the reading view (S15): nothing raw, the cursor's block under
// the band, j/k by block; raw only while typing or selecting.

// bgOf is the background screenCells reports for colour c.
func bgOf(c lipgloss.Color) string {
	return screenCells(lipgloss.NewStyle().Background(c).Render("x"))[0].bg
}

// rowWith returns the screen row holding text, or fails.
func rowWith(t *testing.T, v, text string) string {
	t.Helper()
	for _, row := range strings.Split(v, "\n") {
		if strings.Contains(ansi.Strip(row), text) {
			return row
		}
	}
	t.Fatalf("no row with %q:\n%s", text, ansi.Strip(v))
	return ""
}

// banded reports whether a note cell of row other than the divider carries
// the background bg.
func banded(row, bg string) bool {
	cells := screenCells(row)
	for _, c := range cells[min(app.SidebarWidth+1, len(cells)):] {
		if c.bg == bg {
			return true
		}
	}
	return false
}

// S15: in NORMAL the cursor's table is rendered, and only INSERT shows its
// Markdown; Esc renders it again.
func TestPrettyNormalDrawsTheCursorsBlockRendered(t *testing.T) {
	h := prettyAt(t, prettyNote)
	h.keys("ggj") // onto the table
	if l, _, _ := app.NoteCursor(h.m); l != 3 {
		t.Fatalf("setup: the cursor is on line %d, want the table's 3", l)
	}
	wantOnScreen(t, h.view(), []string{"Title", "Head", "end"}, []string{"|---|", "# Title", "[!note]"})
	h.keys("i")
	wantOnScreen(t, h.view(), []string{"| a | b |", "|---|---|", "| 1 | 2 |"}, []string{"# Title"})
	h.key(tea.KeyEsc)
	wantOnScreen(t, h.view(), []string{"Title"}, []string{"|---|"})
}

// S15: every row of the cursor's block carries the band, while the editor
// has the focus; the other blocks do not.
func TestPrettyNormalBandsEveryRowOfTheCursorsBlock(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	band := bgOf(th.Bg2)
	h := prettyAt(t, "intro\n\n> [!note] Head\n> first\n> second\n\nend")
	h.keys("ggj") // onto the callout
	v := h.colourView()
	for _, text := range []string{"Head", "first", "second"} {
		if !banded(rowWith(t, v, text), band) {
			t.Fatalf("the callout's %q row has no band:\n%q", text, rowWith(t, v, text))
		}
	}
	if banded(rowWith(t, v, "intro"), band) || banded(rowWith(t, v, "end"), band) {
		t.Fatal("a block without the cursor took the band")
	}
	h.send(tea.KeyMsg{Type: tea.KeyTab}) // the explorer takes the focus
	if banded(rowWith(t, h.colourView(), "first"), band) {
		t.Fatal("the band should go with the focus")
	}
}

// lastCursor records what the app last told the terminal's cursor.
type lastCursor struct{ visible bool }

func (c *lastCursor) Set(_, _ int, _ cursor.Shape, visible bool) { c.visible = visible }
func (c *lastCursor) Restore() error                             { return nil }
func (c *lastCursor) Forget()                                    {}

// S15: the terminal cursor is hidden on a rendered block in NORMAL and comes
// back on the raw line in INSERT.
func TestPrettyNormalHidesTheTerminalCursor(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "# Title\n\nend"); err != nil {
		t.Fatal(err)
	}
	sink := &lastCursor{}
	h := newHarness(t, store, app.WithCursor(sink))
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys(" p")
	h.keys("gg")
	h.view()
	if sink.visible {
		t.Fatal("the terminal cursor should be hidden on a rendered heading")
	}
	h.keys("A")
	h.view()
	if !sink.visible {
		t.Fatal("INSERT should show the terminal cursor on the raw line")
	}
}

// S15: j and k go block by block over blank runs; G lands on the first line
// of the last block.
func TestPrettyNormalMovesBlockByBlock(t *testing.T) {
	h := prettyAt(t, prettyNote)
	h.keys("gg")
	for _, step := range []struct {
		keys string
		line int
	}{{"j", 3}, {"j", 7}, {"j", 10}, {"j", 10}, {"k", 7}, {"k", 3}, {"k", 1}, {"G", 10}} {
		h.keys(step.keys)
		if l, _, ok := app.NoteCursor(h.m); l != step.line || !ok {
			t.Fatalf("%s landed on line %d (on screen %v), want %d", step.keys, l, ok, step.line)
		}
	}
	h = prettyAt(t, "intro\n\n> [!note] Head\n> body\n> more")
	if l, _, _ := app.NoteCursor(h.m); l != 3 {
		t.Fatalf("G should land on the callout's first line 3, got %d", l)
	}
}

// S15: Esc from INSERT compiles at once when a compile waits, so the block
// is never drawn rendered with its old text.
func TestPrettyEscCompilesAPendingEdit(t *testing.T) {
	h := openNoteWith(t, longNote("# Title\n**Bold**"))
	h.keys(" p")
	h.keys("A more")
	compiles := app.PreviewCompiles(h.m)
	h.key(tea.KeyEsc)
	if got := app.PreviewCompiles(h.m); got != compiles+1 {
		t.Fatalf("Esc should compile the pending edit: %d compiles, was %d", got, compiles)
	}
	wantOnScreen(t, h.view(), []string{"Title more"}, []string{"# Title"})
}

// S4, S15: a mouse selection in NORMAL reveals the blocks it touches, which
// NORMAL alone draws rendered.
func TestPrettyNormalMouseSelectionReveals(t *testing.T) {
	h := prettyAt(t, "**one**\n\n**two**\n\n**end**")
	v := h.view()
	wantOnScreen(t, v, []string{"one", "end"}, []string{"**"})
	x, y, _ := textAt(t, v, "one")
	h.send(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	h.send(tea.MouseMsg{X: x + 2, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion})
	h.send(tea.MouseMsg{X: x + 2, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	wantOnScreen(t, h.view(), []string{"**one**", "two"}, []string{"**two**", "**end**"})
}

// S8 (B1): gg onto a block the top edge cuts shows the whole block.
func TestPrettyGgShowsTheTopOfACutBlock(t *testing.T) {
	var b strings.Builder
	b.WriteString("# Title\n\nfirst para\n")
	for i := range 60 {
		fmt.Fprintf(&b, "\npara %d\n", i)
	}
	h := openNoteWith(t, b.String())
	h.keys(" p")
	h.keys("jj")
	h.m = app.ScrollNoteBy(h.m, 1)
	h.view()
	h.keys("gg")
	v := h.view()
	if l, _, ok := app.NoteCursor(h.m); l != 1 || !ok || !strings.Contains(v, "Title") {
		t.Fatalf("gg: line %d on screen %v\n%s", l, ok, v)
	}
}

// S8 (B1): k into a block the top edge cuts keeps the cursor on screen.
func TestPrettyKIntoABlockCutByTheTop(t *testing.T) {
	var b strings.Builder
	for i := range 30 {
		fmt.Fprintf(&b, "para %d\n\n", i)
		if i%5 == 2 {
			b.WriteString("| a | b |\n|---|---|\n| 1 | 2 |\n| 3 | 4 |\n| 5 | 6 |\n| 7 | 8 |\n\n")
		}
	}
	h := openNoteWith(t, b.String()+"end")
	h.keys(" p")
	h.keys("G")
	for i := range 120 {
		h.keys("k")
		h.view()
		if l, _, ok := app.NoteCursor(h.m); !ok {
			t.Fatalf("k #%d: line %d off screen", i+1, l)
		}
	}
}

// S8: the wheel over a block taller than the screen scrolls through it, the
// cursor staying on the block, and never bounces back to its top.
func TestPrettyWheelThroughATallBlock(t *testing.T) {
	var b strings.Builder
	b.WriteString("intro\n\n| n | v |\n|---|---|\n")
	for i := range 80 {
		fmt.Fprintf(&b, "| r%02d | x |\n", i)
	}
	b.WriteString("\nend")
	h := prettyAt(t, b.String())
	h.keys("ggj")
	if l, _, _ := app.NoteCursor(h.m); l != 3 {
		t.Fatalf("setup: the cursor is on line %d, want the table's 3", l)
	}
	x, y := noteTextAt(t, h.view(), "r00")
	prev := strings.Join(paneRows(h.view()), "\n")
	for i := range 6 {
		h.send(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
		h.keys("$") // any key brings the cursor into view again
		pane := strings.Join(paneRows(h.view()), "\n")
		if l, _, _ := app.NoteCursor(h.m); l != 3 || pane == prev || strings.Contains(pane, "intro") {
			t.Fatalf("wheel %d: cursor on line %d, the pane did not move on:\n%s", i+1, l, pane)
		}
		prev = pane
	}
}

// S15: dd on a block lands on the next block, so a second dd deletes that
// block, never a blank line; the blocks left stay one blank line apart.
func TestPrettyDdTwiceDeletesTwoBlocks(t *testing.T) {
	h := prettyAt(t, prettyNote)
	h.keys("ggj") // onto the table
	h.keys("dd")
	if got, want := app.EditorText(h.m), "# Title\n\n> [!note] Head\n> body\n\nend"; got != want {
		t.Fatalf("dd on the table left %q, want %q", got, want)
	}
	if l, _, ok := app.NoteCursor(h.m); l != 3 || !ok {
		t.Fatalf("after dd the cursor is on line %d (on screen %v), want the callout's 3", l, ok)
	}
	h.keys("dd")
	if got, want := app.EditorText(h.m), "# Title\n\nend"; got != want {
		t.Fatalf("a second dd left %q, want %q", got, want)
	}
}

// S15: o under a table opens a paragraph of its own, not a table row.
func TestPrettyOUnderATableOpensAParagraph(t *testing.T) {
	h := prettyAt(t, prettyNote)
	h.keys("ggj")
	h.keys("onew")
	h.key(tea.KeyEsc)
	want := "# Title\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\nnew\n\n> [!note] Head\n> body\n\nend"
	if got := app.EditorText(h.m); got != want {
		t.Fatalf("o under the table left %q, want %q", got, want)
	}
	wantOnScreen(t, h.view(), []string{"new", "Head"}, []string{"|---|"})
}

// S15 (L17): x over a rendered fenced block in NORMAL does nothing; it would
// delete an invisible backtick and break the fence.
func TestPrettyXOnAFenceChangesNothing(t *testing.T) {
	const note = "intro\n\n```\ncode\n```\n\nend"
	h := prettyAt(t, note)
	h.keys("ggj")
	if l, _, ok := app.NoteCursor(h.m); l != 3 || !ok {
		t.Fatalf("setup: the cursor is on line %d (on screen %v), want the fence's 3", l, ok)
	}
	h.keys("x")
	if got := app.EditorText(h.m); got != note {
		t.Fatalf("x on the fence changed the note to %q", got)
	}
	wantOnScreen(t, h.view(), []string{"code"}, []string{"```"})
}

// S15: in the split and in space r, x deletes the character under the
// cursor as always.
func TestSplitAndRawViewXDeletesACharacter(t *testing.T) {
	h := openNoteWith(t, "# Head\n\nend")
	h.keys("x")
	if got := app.EditorText(h.m); got != " Head\n\nend" {
		t.Fatalf("x in the split should delete the character under the cursor, left %q", got)
	}
	h = openNoteWith(t, "# Head\n\nend")
	h.keys(" r")
	h.keys("x")
	if got := app.EditorText(h.m); got != " Head\n\nend" {
		t.Fatalf("x in space r should delete the character under the cursor, left %q", got)
	}
}
