package content_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// searchNote is an editor showing text with the cursor at its top.
func searchNote(t *testing.T, text string) content.Model {
	t.Helper()
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 10)
	c.SetPage(core.Page{ID: "a.md", Content: text})
	c = keys(c, "gg")
	return c
}

// statusOf runs a command and returns the status message it carries.
func statusOf(cmd tea.Cmd) (core.StatusMsg, bool) {
	if cmd == nil {
		return core.StatusMsg{}, false
	}
	return findStatus(cmd())
}

func findStatus(msg tea.Msg) (core.StatusMsg, bool) {
	switch msg := msg.(type) {
	case core.StatusMsg:
		return msg, true
	case tea.BatchMsg:
		for _, c := range msg {
			if s, ok := statusOf(c); ok {
				return s, true
			}
		}
	}
	return core.StatusMsg{}, false
}

// doSearch types "/" and query, then Enter, and returns the editor with the
// status it emitted, if any.
func doSearch(t *testing.T, c content.Model, query string) (content.Model, core.StatusMsg) {
	t.Helper()
	c = keys(c, "/")
	if c.Mode() != content.ModeCommand {
		t.Fatalf("/ should open the search prompt, mode is %v", c.Mode())
	}
	c = keys(c, query)
	c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	s, _ := statusOf(cmd)
	return c, s
}

// repeatSearch runs n (forward) or N and returns the status it emitted.
func repeatSearch(t *testing.T, c content.Model, key string) (content.Model, core.StatusMsg) {
	t.Helper()
	c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	s, _ := statusOf(cmd)
	return c, s
}

// at reports the 1-based line and column of the cursor.
func at(c content.Model) (int, int) { return c.CursorPosition() }

// S2: / searches forward from the cursor; n and N repeat it both ways.
func TestSearchFindsTheNextMatchAndRepeats(t *testing.T) {
	c := searchNote(t, "first foo\nsecond foo\nthird foo")

	c, s := doSearch(t, c, "foo")
	if line, col := at(c); line != 1 || col != 7 {
		t.Fatalf("/foo should land on the first foo, got %d:%d", line, col)
	}
	if s.Text != "" {
		t.Errorf("a plain hit says nothing, got %q", s.Text)
	}

	c, _ = repeatSearch(t, c, "n")
	if line, col := at(c); line != 2 || col != 8 {
		t.Fatalf("n should go to the next foo, got %d:%d", line, col)
	}
	c, _ = repeatSearch(t, c, "N")
	if line, col := at(c); line != 1 || col != 7 {
		t.Fatalf("N should go back to the first foo, got %d:%d", line, col)
	}
}

// S2: the search wraps around the note, saying so, both ways.
func TestSearchWrapsBothWays(t *testing.T) {
	c := searchNote(t, "first foo\nsecond foo\nthird foo")
	c, _ = doSearch(t, c, "foo")
	c, _ = repeatSearch(t, c, "n")
	c, _ = repeatSearch(t, c, "n") // the last match

	c, s := repeatSearch(t, c, "n")
	if line, col := at(c); line != 1 || col != 7 {
		t.Fatalf("n past the last match should wrap, got %d:%d", line, col)
	}
	if s.Text != "search hit BOTTOM, continuing at TOP" {
		t.Errorf("wrap message = %q", s.Text)
	}

	c, s = repeatSearch(t, c, "N")
	if line, col := at(c); line != 3 || col != 7 {
		t.Fatalf("N before the first match should wrap, got %d:%d", line, col)
	}
	if s.Text != "search hit TOP, continuing at BOTTOM" {
		t.Errorf("wrap message = %q", s.Text)
	}
}

// S2: smartcase, as in Vim.
func TestSearchSmartcase(t *testing.T) {
	c := searchNote(t, "x Foo\ny foo")

	c, _ = doSearch(t, c, "foo") // lower case: case-insensitive
	if line, col := at(c); line != 1 || col != 3 {
		t.Fatalf("/foo should match Foo, got %d:%d", line, col)
	}
	c, _ = repeatSearch(t, c, "n")
	if line, col := at(c); line != 2 || col != 3 {
		t.Fatalf("n should reach the lower-case foo, got %d:%d", line, col)
	}

	c, _ = doSearch(t, c, "Foo") // an upper case letter: exact case
	c, _ = repeatSearch(t, c, "n")
	if line, col := at(c); line != 1 || col != 3 {
		t.Fatalf("/Foo must not match foo, got %d:%d", line, col)
	}
}

// S2: a missing match is E486, and the cursor stays.
func TestSearchNotFound(t *testing.T) {
	c := searchNote(t, "one\ntwo")
	c = keys(c, "j") // cursor on "two"
	c, s := doSearch(t, c, "zzz")
	if s.Text != "E486: Pattern not found: zzz" || !s.Error {
		t.Fatalf("status = %+v, want the E486 error", s)
	}
	if line, col := at(c); line != 2 || col != 1 {
		t.Fatalf("a failed search must not move the cursor, got %d:%d", line, col)
	}
	if c.Mode() != content.ModeNormal {
		t.Errorf("the prompt should close, mode is %v", c.Mode())
	}
}

// S2: Esc cancels, and so does Backspace on an empty prompt; the next ":" is
// the command line again.
func TestSearchCancels(t *testing.T) {
	c := searchNote(t, "one foo\ntwo")
	c = keys(c, "/")
	c = keys(c, "fo")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if c.Mode() != content.ModeNormal || c.Value() != "one foo\ntwo" {
		t.Fatalf("Esc should cancel the search, mode %v text %q", c.Mode(), c.Value())
	}
	if line, _ := at(c); line != 1 {
		t.Errorf("a cancelled search must not move the cursor, at line %d", line)
	}

	c = keys(c, "/")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if c.Mode() != content.ModeNormal {
		t.Fatalf("Backspace on an empty prompt should cancel, mode %v", c.Mode())
	}

	// The command line keeps its ":".
	c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	s, _ := statusOf(cmd)
	if c.Mode() != content.ModeCommand || s.Text != "" {
		t.Fatalf(": should open the command line, mode %v status %q", c.Mode(), s.Text)
	}
	if v := c.CommandView(80); !strings.HasPrefix(v, ":") {
		t.Fatalf("the command prompt should be \":\", got %q", v)
	}
}

// S2: "/" at the start of a line in INSERT still opens the block menu.
func TestSlashInInsertStillOpensTheBlockMenu(t *testing.T) {
	c := searchNote(t, "")
	c = keys(c, "i/")
	if !c.SlashOpen() {
		t.Fatal("/ in INSERT should open the block menu")
	}
}

// S2: n and N without a search say so, as Vim does.
func TestRepeatSearchWithoutASearch(t *testing.T) {
	c := searchNote(t, "one\ntwo")
	c, s := repeatSearch(t, c, "n")
	if s.Text != "E35: No previous regular expression" || !s.Error {
		t.Fatalf("status = %+v, want E35", s)
	}
	if line, _ := at(c); line != 1 {
		t.Errorf("the cursor should stay, at line %d", line)
	}
}
