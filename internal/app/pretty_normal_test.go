package app_test

import (
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
