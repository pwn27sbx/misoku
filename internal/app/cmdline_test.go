package app_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// cmdlineRow is the screen row of the floating command line's top border:
// row 3 counting from 0, noice's command_palette position.
const cmdlineRow = 3

// colOf is the screen column where sub starts in the plain row line, or -1.
func colOf(line, sub string) int {
	i := strings.Index(line, sub)
	if i < 0 {
		return -1
	}
	return ansi.StringWidth(line[:i])
}

// cmdlineBox returns the left column and the width of the floating box drawn
// in the plain frame v, failing when its top border is not on cmdlineRow.
func cmdlineBox(t *testing.T, v string) (x, w int) {
	t.Helper()
	rows := strings.Split(v, "\n")
	top := rows[cmdlineRow]
	x, right := colOf(top, "╭"), colOf(top, "╮")
	if x < 0 || right < x {
		t.Fatalf("no box on row %d:\n%s", cmdlineRow, v)
	}
	if got := colOf(rows[cmdlineRow+2], "╰"); got != x {
		t.Fatalf("the bottom border starts at %d, want %d:\n%s", got, x, v)
	}
	return x, right - x + 1
}

// S1, S3: ":" draws a floating box at row 3, centred and 60 columns wide,
// titled on its top border, the prompt "›" and the typed text inside; the
// bottom row keeps the statusline with its COMMAND badge.
func TestColonOpensTheFloatingCommandLine(t *testing.T) {
	h := openWith(t, "hello")
	h.keys(":ab")
	v := h.view()
	x, w := cmdlineBox(t, v)
	if w != 60 || x != (termW-60)/2 {
		t.Fatalf("box at %d, %d wide; want %d, 60:\n%s", x, w, (termW-60)/2, v)
	}
	rows := strings.Split(v, "\n")
	if got, want := colOf(rows[cmdlineRow], " Cmdline "), x+1+(w-2-len(" Cmdline "))/2; got != want {
		t.Fatalf("the title starts at %d, want %d (centred):\n%s", got, want, v)
	}
	if got := colOf(rows[cmdlineRow+1], "│ › ab"); got != x {
		t.Fatalf("the prompt row should read \"│ › ab\" from %d, got %d:\n%s", x, got, v)
	}
	last := rows[len(rows)-1]
	if strings.HasPrefix(last, ":") || mode(v) != "COMMAND" {
		t.Fatalf("the bottom row should be the statusline with COMMAND, got %q", last)
	}
	h.key(tea.KeyEsc)
	if v := h.view(); strings.Contains(v, " Cmdline ") || mode(v) != "NORMAL" {
		t.Fatalf("Esc should close the box:\n%s", v)
	}
}

// S1: the box grows with the text and stops 4 columns short of the screen.
func TestCmdlineBoxGrowsWithTheText(t *testing.T) {
	h := openWith(t, "hello")
	h.keys(":" + strings.Repeat("a", 70))
	if x, w := cmdlineBox(t, h.view()); w != 77 || x != (termW-77)/2 {
		t.Fatalf("70 typed columns: box at %d, %d wide; want %d, 77", x, w, (termW-77)/2)
	}
	h.keys(strings.Repeat("b", 200))
	if x, w := cmdlineBox(t, h.view()); w != termW-4 || x != 2 {
		t.Fatalf("a long line: box at %d, %d wide; want 2, %d", x, w, termW-4)
	}
}

// S1: on the narrowest screen the box is the screen minus 4 columns.
func TestCmdlineBoxOnTheNarrowestScreen(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "views", "hello"); err != nil {
		t.Fatal(err)
	}
	h := newHarnessSize(t, app.MinTermWidth, app.MinTermHeight, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys(":")
	v := ansi.Strip(h.m.View())
	if x, w := cmdlineBox(t, v); w != app.MinTermWidth-4 || x != 2 {
		t.Fatalf("box at %d, %d wide; want 2, %d:\n%s", x, w, app.MinTermWidth-4, v)
	}
}

// S1, S2: "/" opens the same box titled Search, the prompt "/".
func TestSearchOpensTheFloatingSearchBox(t *testing.T) {
	h := openWith(t, "hello foo")
	h.keys("/foo")
	v := h.view()
	x, _ := cmdlineBox(t, v)
	rows := strings.Split(v, "\n")
	if !strings.Contains(rows[cmdlineRow], " Search ") || strings.Contains(v, "Cmdline") {
		t.Fatalf("the search box should be titled Search:\n%s", v)
	}
	if got := colOf(rows[cmdlineRow+1], "│ / foo"); got != x {
		t.Fatalf("the prompt row should read \"│ / foo\" from %d, got %d:\n%s", x, got, v)
	}
	h.key(tea.KeyEnter)
	if v := h.view(); strings.Contains(v, " Search ") || mode(v) != "NORMAL" {
		t.Fatalf("Enter should close the box:\n%s", v)
	}
}

// S2: the titles are in the interface language.
func TestCmdlineTitlesFollowTheLanguage(t *testing.T) {
	useLang(t, i18n.ES)
	h := openWith(t, "hola")
	h.keys(":")
	if v := h.view(); !strings.Contains(strings.Split(v, "\n")[cmdlineRow], " Comando ") {
		t.Fatalf("the Spanish command box should be titled Comando:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.keys("/")
	if v := h.view(); !strings.Contains(strings.Split(v, "\n")[cmdlineRow], " Buscar ") {
		t.Fatalf("the Spanish search box should be titled Buscar:\n%s", v)
	}
}

// S2: the border, the title and the glyph are blue for ":" and yellow for
// "/", noice's Cmdline and Search colours.
func TestCmdlineBoxColourFollowsTheKind(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	for _, c := range []struct {
		key, title, glyph string
		want              string
	}{
		{":", "Cmdline", "›", hexOf(th.Blue)},
		{"/", "Search", "/", hexOf(th.Yellow)},
	} {
		h := openWith(t, "hello")
		h.keys(c.key)
		rows := strings.Split(h.colourView(), "\n")
		top := screenCells(rows[cmdlineRow])
		corner := cellIndex(top, '╭', 0)
		titleAt := cellIndex(top, rune(c.title[0]), corner)
		if corner < 0 || titleAt < 0 {
			t.Fatalf("%s: no box or title on row %d", c.key, cmdlineRow)
		}
		if top[corner].fg != c.want || top[titleAt].fg != c.want {
			t.Errorf("%s: border %s, title %s; want %s", c.key, top[corner].fg, top[titleAt].fg, c.want)
		}
		mid := screenCells(rows[cmdlineRow+1])
		g := cellIndex(mid, []rune(c.glyph)[0], corner+1)
		if g < 0 || mid[g].fg != c.want {
			t.Errorf("%s: the glyph %q should be %s", c.key, c.glyph, c.want)
		}
	}
}

// cellIndex is the first cell at or after from holding r, or -1.
func cellIndex(cells []screenCell, r rune, from int) int {
	for i := max(from, 0); i < len(cells); i++ {
		if cells[i].r == r {
			return i
		}
	}
	return -1
}

// S3: the box is no modal: the note behind it keeps its colours.
func TestCmdlineBoxDoesNotDimTheNote(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	h := openWith(t, "hello")
	h.keys(":")
	for _, row := range strings.Split(h.colourView(), "\n") {
		cells := screenCells(row)
		for i := 0; i+5 <= len(cells); i++ {
			word := string([]rune{cells[i].r, cells[i+1].r, cells[i+2].r, cells[i+3].r, cells[i+4].r})
			if word != "hello" {
				continue
			}
			if cells[i].fg == hexOf(th.Grey) {
				t.Fatalf("the note was dimmed behind the box: %q", ansi.Strip(row))
			}
			return
		}
	}
	t.Fatal("the note is no longer on screen")
}

// S3: a modal opened while ":" is active wins: the box is not drawn under it.
func TestCmdlineBoxHidesUnderTheFinder(t *testing.T) {
	h := openWith(t, "hello")
	h.keys(":")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	if v := h.view(); strings.Contains(v, " Cmdline ") {
		t.Fatalf("the finder should hide the command box:\n%s", v)
	}
}

// S3: a click inside the box never reaches the pane under it; one outside
// still does.
func TestCmdlineBoxSwallowsClicks(t *testing.T) {
	h := openWith(t, "hello")
	h.keys(":")
	x, w := cmdlineBox(t, h.view())
	l := app.CalculateLayout(termW, termH, true, true, false, app.Split{})
	px := l.PreviewX + 2
	if px < x || px >= x+w {
		t.Fatalf("setup: the preview column %d should be under the box %d..%d", px, x, x+w-1)
	}
	h.click(px, cmdlineRow+1)
	if v := h.view(); mode(v) != "COMMAND" || !strings.Contains(v, " Cmdline ") {
		t.Fatalf("a click in the box reached the preview under it:\n%s", v)
	}
	h.click(px, cmdlineRow+10)
	if v := h.view(); mode(v) == "COMMAND" || strings.Contains(v, " Cmdline ") {
		t.Fatalf("a click outside the box should still focus the preview:\n%s", v)
	}
}

// S3: the box is drawn above the narrow-screen explorer overlay.
func TestCmdlineBoxIsDrawnAboveTheExplorerOverlay(t *testing.T) {
	h := newHarnessSize(t, narrowW, narrowH, narrowStore(t))
	openRecent(h)
	h.keys(":")
	h.m = app.ShowExplorerOverlay(h.m)
	v := h.screen()
	if !explorerShown(v) {
		t.Fatalf("setup: the explorer overlay should show:\n%s", v)
	}
	x, w := cmdlineBox(t, v)
	if x >= app.SidebarWidth || w != 60 || !strings.Contains(v, " Cmdline ") {
		t.Fatalf("the whole box should cover the overlay's columns from %d:\n%s", x, v)
	}
}
