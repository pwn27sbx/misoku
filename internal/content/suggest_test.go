package content_test

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// S4: the insert text of every catalog row, as the spec lists it.
func TestCommandInsertTextForEveryCatalogRow(t *testing.T) {
	want := []string{
		"w", "w ", "wq", "q", "bd", "enew", "e ", "export ", "graph", "graph all",
		"daily", "history", "tags", "status ", "messages", "colo ", "settings",
		"sync", "sync status", "sync setup", "addrow", "addcol", "delrow", "tablefmt",
	}
	if len(content.Commands) != len(want) {
		t.Fatalf("the catalog has %d entries, the spec lists %d", len(content.Commands), len(want))
	}
	for i, c := range content.Commands {
		t.Run(c.Usage, func(t *testing.T) {
			if got := c.InsertText(); got != want[i] {
				t.Errorf("InsertText(%q) = %q, want %q", c.Usage, got, want[i])
			}
		})
	}
}

// cmdline opens ":" on a note and types text.
func cmdline(t *testing.T, text string) content.Model {
	t.Helper()
	c := searchNote(t, "hello")
	c = keys(c, ":")
	if c.Mode() != content.ModeCommand {
		t.Fatalf(": should open the command line, mode is %v", c.Mode())
	}
	return keys(c, text)
}

func pressKey(c content.Model, kt tea.KeyType) content.Model {
	c, _ = c.Update(tea.KeyMsg{Type: kt})
	return c
}

// cmdValue is the text of the command line, read from its view.
func cmdValue(c content.Model) string {
	return strings.TrimPrefix(ansi.Strip(c.CommandView(80)), ":")
}

func assertValue(t *testing.T, c content.Model, want string) {
	t.Helper()
	if got := cmdValue(c); !strings.HasPrefix(got, want) || strings.TrimRight(got, " ") != strings.TrimRight(want, " ") {
		t.Errorf("command line holds %q, want %q", got, want)
	}
}

// rowText is how a catalog row reads in the list, without padding.
func rowText(usage string) string {
	for _, c := range content.Commands {
		if c.Usage == usage {
			return c.Usage + "  " + c.HelpText()
		}
	}
	panic("no command " + usage)
}

// listRows returns the rows of the suggestion box, trimmed, or nil when
// there is no box.
func listRows(t *testing.T, c content.Model, width, maxRows int) []string {
	t.Helper()
	box, ok := c.SuggestionsView(width, maxRows, theme.DefaultTheme().Blue)
	if !ok {
		if c.SuggestionsOpen() && maxRows >= 1 {
			t.Errorf("SuggestionsOpen is true but SuggestionsView gave no box")
		}
		return nil
	}
	if !c.SuggestionsOpen() {
		t.Errorf("SuggestionsView drew a box while SuggestionsOpen is false")
	}
	lines := strings.Split(ansi.Strip(box), "\n")
	if len(lines) < 3 {
		t.Fatalf("box too short: %q", box)
	}
	var rows []string
	for _, l := range lines[1 : len(lines)-1] {
		r := []rune(l)
		rows = append(rows, strings.TrimSpace(string(r[1:len(r)-1])))
	}
	return rows
}

func rowsOf(usages ...string) []string {
	var out []string
	for _, u := range usages {
		out = append(out, rowText(u))
	}
	return out
}

// S4 matching: one word matches any name by prefix, in catalog order; with
// a space the whole text must prefix the insert text; empty input, no match
// and "/" show nothing.
func TestSuggestionsMatchTheTypedText(t *testing.T) {
	tests := []struct {
		name  string
		typed string
		want  []string
	}{
		{"one word matches any name", "s", rowsOf(":w name", ":status [value]", ":settings", ":sync", ":sync status", ":sync setup")},
		{"an alias name matches", "th", rowsOf(":colo [name]")},
		{"one word w", "w", rowsOf(":w", ":wq / :x")},
		{"after a space the insert text must start with the text", "sync s", rowsOf(":sync status", ":sync setup")},
		{"graph and a space", "graph ", rowsOf(":graph all")},
		{"a placeholder row matches its own prefix", "e ", rowsOf(":e name")},
		{"an argument past the placeholder lists nothing", "e note", nil},
		{"empty input lists nothing", "", nil},
		{"no match lists nothing", "zzz", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := cmdline(t, tt.typed)
			got := listRows(t, c, 200, content.SuggestMaxRows)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("typed %q: rows\n%q\nwant\n%q", tt.typed, got, tt.want)
			}
		})
	}
}

// S4: "/" never lists commands, and its keys reach the search as before.
func TestSearchPromptHasNoSuggestions(t *testing.T) {
	c := searchNote(t, "first sync\nsecond sync")
	c = keys(c, "/")
	c = keys(c, "sync")
	if c.SuggestionsOpen() {
		t.Fatal("the / prompt opened the command list")
	}
	if _, ok := c.SuggestionsView(80, content.SuggestMaxRows, theme.DefaultTheme().Yellow); ok {
		t.Fatal("the / prompt drew the command list")
	}
	c = pressKey(c, tea.KeyTab)
	c = pressKey(c, tea.KeyDown)
	if got := ansi.Strip(c.CommandView(80)); !strings.HasPrefix(got, "/sync") {
		t.Fatalf("Tab/Down changed the search text: %q", got)
	}
	c = pressKey(c, tea.KeyEnter)
	if line, col := at(c); line != 1 || col != 7 {
		t.Errorf("Enter should still search, cursor at %d:%d", line, col)
	}
}

// S4: no row is selected at first, so Enter runs exactly what was typed.
func TestSuggestionsStartWithNoSelection(t *testing.T) {
	trueColour(t)
	c := cmdline(t, "sync")
	box, ok := c.SuggestionsView(80, content.SuggestMaxRows, theme.DefaultTheme().Blue)
	if !ok {
		t.Fatal("no list for :sync")
	}
	if sel := selectedRows(box); len(sel) != 0 {
		t.Errorf("rows selected before any key: %q", sel)
	}
	assertValue(t, c, "sync")
	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got, want := runMsg(cmd), content.RunCommand("sync", ""); !reflect.DeepEqual(got, want) {
		t.Errorf("Enter ran %#v, want %#v", got, want)
	}
}

// selectedRows returns the plain text of the rows painted with the
// selection background.
func selectedRows(box string) []string {
	bg := rgb(theme.DefaultTheme().OneBg2)
	var out []string
	for _, l := range strings.Split(box, "\n") {
		if _, bgs := colours(l); bgs[bg] {
			r := []rune(ansi.Strip(l))
			out = append(out, strings.TrimSpace(string(r[1:len(r)-1])))
		}
	}
	return out
}

// runMsg is the message a command line Enter produced, without the focus
// commands batched with it.
func runMsg(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if b, ok := msg.(tea.BatchMsg); ok {
		for _, c := range b[1:] {
			if c != nil {
				return c()
			}
		}
		return nil
	}
	return msg
}

// S4 selection: Tab/Down go forward, Shift+Tab/Up back, wrapping; the input
// takes the row's insert text and the list stays.
func TestSuggestionSelectionMovesAndWraps(t *testing.T) {
	all := rowsOf(":sync", ":sync status", ":sync setup")
	tests := []struct {
		name  string
		keys  []tea.KeyType
		value string
		sel   string
	}{
		{"tab selects the first row", []tea.KeyType{tea.KeyTab}, "sync", ":sync"},
		{"down selects the first row", []tea.KeyType{tea.KeyDown}, "sync", ":sync"},
		{"tab twice", []tea.KeyType{tea.KeyTab, tea.KeyTab}, "sync status", ":sync status"},
		{"tab wraps to the top", []tea.KeyType{tea.KeyTab, tea.KeyTab, tea.KeyTab, tea.KeyTab}, "sync", ":sync"},
		{"shift+tab selects the last row", []tea.KeyType{tea.KeyShiftTab}, "sync setup", ":sync setup"},
		{"up selects the last row", []tea.KeyType{tea.KeyUp}, "sync setup", ":sync setup"},
		{"up wraps to the bottom", []tea.KeyType{tea.KeyDown, tea.KeyUp}, "sync setup", ":sync setup"},
		{"down then up goes back", []tea.KeyType{tea.KeyDown, tea.KeyDown, tea.KeyUp}, "sync", ":sync"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trueColour(t)
			c := cmdline(t, "sync")
			for _, k := range tt.keys {
				c = pressKey(c, k)
			}
			assertValue(t, c, tt.value)
			if got := listRows(t, c, 200, content.SuggestMaxRows); !reflect.DeepEqual(got, all) {
				t.Errorf("the list changed on selection: %q", got)
			}
			box, _ := c.SuggestionsView(200, content.SuggestMaxRows, theme.DefaultTheme().Blue)
			if got := selectedRows(box); !reflect.DeepEqual(got, []string{rowText(tt.sel)}) {
				t.Errorf("selected rows %q, want %q", got, rowText(tt.sel))
			}
		})
	}
}

// S4: a selected placeholder row writes its insert text with the space.
func TestSelectingAPlaceholderRowLeavesTheSpace(t *testing.T) {
	c := cmdline(t, "e")
	// "e" lists :enew (alias e) and :e name.
	c = pressKey(c, tea.KeyTab)
	c = pressKey(c, tea.KeyTab)
	if got := cmdValue(c); !strings.HasPrefix(got, "e ") {
		t.Errorf("command line holds %q, want %q", got, "e ")
	}
}

// S4: typing or deleting clears the selection and filters again.
func TestTypingAfterASelectionFiltersAgain(t *testing.T) {
	trueColour(t)
	c := cmdline(t, "sync")
	c = pressKey(c, tea.KeyTab)
	c = pressKey(c, tea.KeyTab) // "sync status"
	c = pressKey(c, tea.KeyBackspace)
	assertValue(t, c, "sync statu")
	if got := listRows(t, c, 200, content.SuggestMaxRows); !reflect.DeepEqual(got, rowsOf(":sync status")) {
		t.Errorf("after a delete the list is %q", got)
	}
	box, _ := c.SuggestionsView(200, content.SuggestMaxRows, theme.DefaultTheme().Blue)
	if got := selectedRows(box); len(got) != 0 {
		t.Errorf("the selection survived a delete: %q", got)
	}

	c = cmdline(t, "sync")
	c = pressKey(c, tea.KeyTab)
	c = keys(c, " s")
	if got := listRows(t, c, 200, content.SuggestMaxRows); !reflect.DeepEqual(got, rowsOf(":sync status", ":sync setup")) {
		t.Errorf("after typing the list is %q", got)
	}
	box, _ = c.SuggestionsView(200, content.SuggestMaxRows, theme.DefaultTheme().Blue)
	if got := selectedRows(box); len(got) != 0 {
		t.Errorf("the selection survived typing: %q", got)
	}
}

// S4: with no list, the selection keys change nothing.
func TestSelectionKeysWithoutAListDoNothing(t *testing.T) {
	for _, k := range []tea.KeyType{tea.KeyTab, tea.KeyShiftTab, tea.KeyUp, tea.KeyDown} {
		c := cmdline(t, "zzz")
		c = pressKey(c, k)
		if c.Mode() != content.ModeCommand {
			t.Fatalf("%v closed the command line", k)
		}
		assertValue(t, c, "zzz")
		if c.SuggestionsOpen() {
			t.Errorf("%v opened a list", k)
		}
	}
}

// S4/S5: Enter runs the selected text as typed; Esc and Backspace on an
// empty line close as before, and the next ":" starts with no list.
func TestCommandLineKeysKeepTheirMeaning(t *testing.T) {
	c := cmdline(t, "sync")
	c = pressKey(c, tea.KeyTab)
	c = pressKey(c, tea.KeyTab)
	c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if got, want := runMsg(cmd), content.RunCommand("sync status", ""); !reflect.DeepEqual(got, want) {
		t.Errorf("Enter ran %#v, want %#v", got, want)
	}
	if c.Mode() != content.ModeNormal || c.SuggestionsOpen() {
		t.Errorf("Enter should close the line and the list, mode %v", c.Mode())
	}

	c = cmdline(t, "sync")
	c = pressKey(c, tea.KeyEsc)
	if c.Mode() != content.ModeNormal || c.SuggestionsOpen() {
		t.Errorf("Esc should close the line and the list, mode %v", c.Mode())
	}
	c = keys(c, ":")
	if c.SuggestionsOpen() {
		t.Error("a new : opened with the old list")
	}

	c = cmdline(t, "")
	c = pressKey(c, tea.KeyBackspace)
	if c.Mode() != content.ModeNormal {
		t.Errorf("Backspace on an empty line should close it, mode %v", c.Mode())
	}
}

// S4 view: the box is exactly width columns, rows cut with an ellipsis, at
// most maxRows rows, scrolling to keep the selection; maxRows < 1 is no box.
func TestSuggestionsViewSizeAndScroll(t *testing.T) {
	c := cmdline(t, "sync")
	box, ok := c.SuggestionsView(24, content.SuggestMaxRows, theme.DefaultTheme().Blue)
	if !ok {
		t.Fatal("no box")
	}
	lines := strings.Split(box, "\n")
	if len(lines) != 3+2 {
		t.Errorf("box has %d lines, want 5", len(lines))
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 24 {
			t.Errorf("line %d is %d columns wide, want 24: %q", i, w, l)
		}
	}
	if !strings.Contains(lines[2], "…") {
		t.Errorf("a long row should end in an ellipsis: %q", lines[2])
	}

	if got := listRows(t, c, 200, 2); !reflect.DeepEqual(got, rowsOf(":sync", ":sync status")) {
		t.Errorf("two rows with no selection: %q", got)
	}
	c = pressKey(c, tea.KeyShiftTab) // the last row
	if got := listRows(t, c, 200, 2); !reflect.DeepEqual(got, rowsOf(":sync status", ":sync setup")) {
		t.Errorf("the window should scroll to the selection: %q", got)
	}
	c = pressKey(c, tea.KeyTab) // wraps to the first row
	if got := listRows(t, c, 200, 2); !reflect.DeepEqual(got, rowsOf(":sync", ":sync status")) {
		t.Errorf("the window should scroll back up: %q", got)
	}
	if _, ok := c.SuggestionsView(200, 0, theme.DefaultTheme().Blue); ok {
		t.Error("maxRows 0 should draw nothing")
	}
	if content.SuggestMaxRows != 8 {
		t.Errorf("SuggestMaxRows = %d, the design caps it at 8", content.SuggestMaxRows)
	}
}

// S5: Tab in INSERT still indents (the list keys only live in COMMAND).
func TestTabInInsertStillIndents(t *testing.T) {
	c := searchNote(t, "- item")
	c = keys(c, "A")
	c = pressKey(c, tea.KeyTab)
	if v := c.Value(); !strings.HasPrefix(v, "  - item") && !strings.HasPrefix(v, "\t- item") {
		t.Errorf("Tab in INSERT should indent the item, got %q", v)
	}
}
