package app_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// listTop is the screen row of the command list's top border: right under
// the three rows of the command box.
const listTop = cmdlineRow + 3

// sMatches are the rows ":s" lists, in catalog order (S4).
var sMatches = []string{":w name", ":status [value]", ":settings", ":sync", ":sync status", ":sync setup"}

// listUnderBox returns the rows of the command list drawn right under the
// box in the plain frame v, without their borders and padding, or nil when
// there is no list. It fails when the list does not share the box's left
// column and width.
func listUnderBox(t *testing.T, v string) []string {
	t.Helper()
	x, w := cmdlineBox(t, v)
	rows := strings.Split(v, "\n")
	top := rows[listTop]
	if plainCell(top, x) != "╭" {
		return nil
	}
	if plainCell(top, x+w-1) != "╮" {
		t.Fatalf("the list is not the box's %d columns wide:\n%s", w, v)
	}
	var out []string
	for y := listTop + 1; y < len(rows); y++ {
		if plainCell(rows[y], x) == "╰" {
			return out
		}
		if plainCell(rows[y], x) != "│" || plainCell(rows[y], x+w-1) != "│" {
			t.Fatalf("row %d of the list does not start at %d:\n%s", y, x, v)
		}
		line := ansi.Cut(rows[y], x+1, x+w-1)
		out = append(out, strings.TrimSpace(line))
	}
	t.Fatalf("the list has no bottom border:\n%s", v)
	return nil
}

// plainCell is the plain cell at column x of a plain row.
func plainCell(row string, x int) string { return ansi.Cut(row, x, x+1) }

// usagesOf keeps the Usage part of each list row (before the two spaces).
func usagesOf(rows []string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i], _, _ = strings.Cut(r, "  ")
	}
	return out
}

func sameRows(got, want []string) bool {
	return strings.Join(got, "|") == strings.Join(want, "|")
}

// S4: typing ":s" draws the matching commands under the box, same left
// column and width, in catalog order, each with its help text.
func TestColonListsMatchingCommandsUnderTheBox(t *testing.T) {
	h := openWith(t, "hello")
	h.keys(":s")
	v := h.view()
	rows := listUnderBox(t, v)
	if got := usagesOf(rows); !sameRows(got, sMatches) {
		t.Fatalf("list rows %q, want %q:\n%s", got, sMatches, v)
	}
	for i, c := range content.Commands {
		if c.Usage == ":settings" && !strings.Contains(rows[2], c.HelpText()) {
			t.Fatalf("row %q lacks the help text of Commands[%d] %q", rows[2], i, c.HelpText())
		}
	}
	last := strings.Split(v, "\n")[termH-1]
	if mode(v) != "COMMAND" || strings.Contains(last, "╰") {
		t.Fatalf("the statusline should keep the bottom row, got %q", last)
	}
}

// S4: Tab selects the first row and writes its insert text into the box;
// the list stays.
func TestTabWritesTheSelectedCommandIntoTheBox(t *testing.T) {
	h := openWith(t, "hello")
	h.keys(":s")
	h.key(tea.KeyTab)
	v := h.view()
	x, _ := cmdlineBox(t, v)
	rows := strings.Split(v, "\n")
	if got := colOf(rows[cmdlineRow+1], "│ › w "); got != x {
		t.Fatalf("Tab should write \"w \" into the box, row reads %q:\n%s", rows[cmdlineRow+1], v)
	}
	if got := usagesOf(listUnderBox(t, v)); !sameRows(got, sMatches) {
		t.Fatalf("the list should stay after Tab, got %q", got)
	}
}

// S4: "/" has no list, and neither has an empty ":" line.
func TestSearchAndEmptyLineShowNoList(t *testing.T) {
	h := openWith(t, "hello sync")
	h.keys(":")
	if rows := listUnderBox(t, h.view()); rows != nil {
		t.Fatalf("an empty line listed %q", rows)
	}
	h.key(tea.KeyEsc)
	h.keys("/s")
	if rows := listUnderBox(t, h.view()); rows != nil {
		t.Fatalf("the search box listed %q", rows)
	}
}

// openTall opens a note holding "hello" on a termW x height terminal, with
// the editor focused.
func openTall(t *testing.T, height int) *harness {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "views", "hello"); err != nil {
		t.Fatal(err)
	}
	h := newHarnessSize(t, termW, height, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	return h
}

// S3, S4: a modal opened while ":" lists commands hides the list too. On a
// tall terminal the finder sits lower than the list, so the list would
// still show above it.
func TestCmdlineListHidesUnderTheFinder(t *testing.T) {
	for _, height := range []int{termH, 80} {
		h := openTall(t, height)
		h.keys(":s")
		h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
		v := ansi.Strip(h.m.View())
		for _, u := range []string{":status [value]", ":sync setup"} {
			if strings.Contains(v, u) {
				t.Fatalf("height %d: the finder should hide the command list (%q shown):\n%s", height, u, v)
			}
		}
	}
}

// S4: on the lowest terminal the list is cut to the rows that fit above the
// statusline: height - 1 - listTop - 2. One row more of terminal, one row
// more of list.
func TestCmdlineListIsCappedAboveTheStatusline(t *testing.T) {
	for _, c := range []struct{ height, rows int }{
		{app.MinTermHeight, app.MinTermHeight - 1 - listTop - 2},
		{app.MinTermHeight + 1, app.MinTermHeight + 1 - 1 - listTop - 2},
		{termH, len(sMatches)},
	} {
		h := openTall(t, c.height)
		h.keys(":s")
		v := ansi.Strip(h.m.View())
		lines := strings.Split(v, "\n")
		if len(lines) != c.height {
			t.Fatalf("height %d: the frame has %d rows", c.height, len(lines))
		}
		if got := usagesOf(listUnderBox(t, v)); !sameRows(got, sMatches[:c.rows]) {
			t.Fatalf("height %d: list rows %q, want %q:\n%s", c.height, got, sMatches[:c.rows], v)
		}
		if mode(v) != "COMMAND" || strings.ContainsAny(lines[c.height-1], "╭╰│") {
			t.Fatalf("height %d: the statusline row was covered: %q", c.height, lines[c.height-1])
		}
	}
}

// S3: a click inside the list never reaches the pane under it; one below it
// still does.
func TestCmdlineListSwallowsClicks(t *testing.T) {
	h := openWith(t, "hello")
	h.keys(":s")
	x, w := cmdlineBox(t, h.view())
	l := app.CalculateLayout(termW, termH, true, true, false, app.Split{})
	px := l.PreviewX + 2
	if px < x || px >= x+w {
		t.Fatalf("setup: the preview column %d should be under the list %d..%d", px, x, x+w-1)
	}
	bottom := listTop + len(sMatches) + 1
	for _, y := range []int{listTop, listTop + 2, bottom} {
		h.click(px, y)
		if v := h.view(); mode(v) != "COMMAND" || listUnderBox(t, v) == nil {
			t.Fatalf("a click on list row %d reached the preview under it:\n%s", y, v)
		}
	}
	h.click(px, bottom+1)
	if v := h.view(); mode(v) == "COMMAND" {
		t.Fatalf("a click below the list should still focus the preview:\n%s", v)
	}
}

// S4: the list's border is the box's colour: blue for ":".
func TestCmdlineListBorderIsTheBoxColour(t *testing.T) {
	trueColour(t)
	want := hexOf(theme.DefaultTheme().Blue)
	h := openWith(t, "hello")
	h.keys(":s")
	rows := strings.Split(h.colourView(), "\n")
	box := screenCells(rows[cmdlineRow])
	list := screenCells(rows[listTop])
	bc, lc := cellIndex(box, '╭', 0), cellIndex(list, '╭', 0)
	if bc < 0 || lc != bc {
		t.Fatalf("the list corner at %d, the box corner at %d", lc, bc)
	}
	if box[bc].fg != want || list[lc].fg != want {
		t.Fatalf("box border %s, list border %s; want both %s", box[bc].fg, list[lc].fg, want)
	}
	side := screenCells(rows[listTop+1])
	if s := cellIndex(side, '│', bc); s != bc || side[s].fg != want {
		t.Fatalf("the list's side border should be %s at %d", want, bc)
	}
}
