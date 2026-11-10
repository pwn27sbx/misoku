package content

// Searching inside the note: Vim's /, n and N. The prompt reuses the command
// line (ModeCommand) with a "/" prompt, so its keys and the bottom bar stay
// the ones the ":" line already has. The match itself is left to the cursor:
// the editor moves it there and scrolls it into view.

import (
	"unicode"
	"unicode/utf8"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// openSearch opens the command line as a forward search.
func (m *Model) openSearch() tea.Cmd {
	m.mode = ModeCommand
	m.search = true
	m.cmdInput.Prompt = "/"
	m.cmdInput.SetValue("")
	m.textarea.Blur()
	return m.cmdInput.Focus()
}

// closeCommandLine returns from the command line to NORMAL with the ":"
// prompt back, whichever prompt it was opened with.
func (m *Model) closeCommandLine() {
	m.mode = ModeNormal
	m.search = false
	m.cmdInput.Prompt = ":"
	m.cmdInput.Blur()
}

// searchCmd searches forward for query from the cursor, as Enter does in the
// search prompt, and remembers it for n and N.
func (m *Model) searchCmd(query string) tea.Cmd {
	return m.findCmd(query, true)
}

// repeatCmd repeats the last search: forward for n, backward for N.
func (m *Model) repeatCmd(forward bool) tea.Cmd {
	if m.lastSearch == "" {
		return errorCmd("E35: No previous regular expression")
	}
	cmd, found := m.findCmdResult(m.lastSearch, forward)
	if found {
		if forward {
			m.reportAction("search_next", "")
		} else {
			m.reportAction("search_prev", "")
		}
	}
	return cmd
}

// findCmd moves the cursor to the next match of query and reports a wrap
// around the note.
func (m *Model) findCmd(query string, forward bool) tea.Cmd {
	cmd, _ := m.findCmdResult(query, forward)
	return cmd
}

// findCmdResult is findCmd, and it also reports whether the query matched:
// a repeat that finds nothing is not a step the lesson credits.
func (m *Model) findCmdResult(query string, forward bool) (tea.Cmd, bool) {
	if query == "" {
		return nil, false
	}
	m.lastSearch = query
	row, col, wrapped, ok := m.find(query, forward)
	if !ok {
		m.reportAction("search_error", query)
		return errorCmd("E486: Pattern not found: " + query), false
	}
	m.reportAction("search_hit", query)
	m.textarea.SetRowCol(row, col) // keeps the cursor visible
	if !wrapped {
		return nil, true
	}
	m.reportAction("search_wrap", "")
	if forward {
		return statusCmd("search hit BOTTOM, continuing at TOP"), true
	}
	return statusCmd("search hit TOP, continuing at BOTTOM"), true
}

// find looks for the next (forward) or previous occurrence of query from the
// cursor, wrapping around the note. It returns the start of the match as a
// 0-based row and rune column.
func (m *Model) find(query string, forward bool) (row, col int, wrapped, found bool) {
	last := m.textarea.LineCount() - 1
	r0, c0 := m.textarea.RowCol()
	fold := !hasUpper(query) // smartcase: case-sensitive only with an upper-case letter

	if forward {
		if c := matchForward(m.textarea.RowText(r0), query, c0+1, fold); c >= 0 {
			return r0, c, false, true
		}
		for r := r0 + 1; r <= last; r++ {
			if c := matchForward(m.textarea.RowText(r), query, 0, fold); c >= 0 {
				return r, c, false, true
			}
		}
		for r := 0; r <= r0; r++ {
			if c := matchForward(m.textarea.RowText(r), query, 0, fold); c >= 0 && (r < r0 || c <= c0) {
				return r, c, true, true
			}
		}
		return 0, 0, false, false
	}

	if c := matchBackward(m.textarea.RowText(r0), query, c0-1, fold); c >= 0 {
		return r0, c, false, true
	}
	for r := r0 - 1; r >= 0; r-- {
		if c := lastMatch(m.textarea.RowText(r), query, fold); c >= 0 {
			return r, c, false, true
		}
	}
	for r := last; r >= r0; r-- {
		if c := lastMatch(m.textarea.RowText(r), query, fold); c >= 0 && (r > r0 || c >= c0) {
			return r, c, true, true
		}
	}
	return 0, 0, false, false
}

// lastMatch is the rune column of the last match of q in line, or -1.
func lastMatch(line, q string, fold bool) int {
	return matchBackward(line, q, utf8.RuneCountInString(line), fold)
}

// matchForward is the rune column of the first match of q in line at or
// after from, or -1.
func matchForward(line, q string, from int, fold bool) int {
	l, query := []rune(line), []rune(q)
	for i := max(from, 0); i+len(query) <= len(l); i++ {
		if sameRunes(l[i:i+len(query)], query, fold) {
			return i
		}
	}
	return -1
}

// matchBackward is the rune column of the last match of q in line at or
// before from, or -1.
func matchBackward(line, q string, from int, fold bool) int {
	l, query := []rune(line), []rune(q)
	for i := min(from, len(l)-len(query)); i >= 0; i-- {
		if sameRunes(l[i:i+len(query)], query, fold) {
			return i
		}
	}
	return -1
}

// sameRunes reports whether a and b are the same runes, ignoring case when
// fold is set.
func sameRunes(a, b []rune, fold bool) bool {
	for i := range b {
		if a[i] == b[i] {
			continue
		}
		if !fold || unicode.ToLower(a[i]) != unicode.ToLower(b[i]) {
			return false
		}
	}
	return true
}

// hasUpper reports whether s holds an upper-case letter: smartcase searches
// ignore case only when it does not.
func hasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// errorCmd is a status message shown as an error.
func errorCmd(text string) tea.Cmd {
	return func() tea.Msg { return core.StatusMsg{Text: text, Error: true} }
}
