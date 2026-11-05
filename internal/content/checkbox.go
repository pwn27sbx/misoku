package content

// Checkboxes and the Enter key, ported from the LazyVim companion
// (~/.config/nvim/lua/config/misoku.lua: toggle_line, uncheck_line,
// is_checkbox_line, in_fence, code_spans, link_at) so both editors act
// alike on the same note.

import (
	"regexp"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// FollowWikiLinkMsg is emitted by Enter in NORMAL on a [[link]] outside
// code: open the note it names, as the LazyVim companion does.
type FollowWikiLinkMsg struct{ Link core.WikiLink }

var (
	bulletItem  = regexp.MustCompile(`^(\s*[-*+]\s+)(.*)$`)
	orderedItem = regexp.MustCompile(`^(\s*\d+\.\s+)(.*)$`)
)

// splitItem cuts a list item into its head (indent, marker and spaces) and
// the text after it.
func splitItem(line string) (head, rest string, ok bool) {
	for _, re := range [...]*regexp.Regexp{bulletItem, orderedItem} {
		if m := re.FindStringSubmatch(line); m != nil {
			return m[1], m[2], true
		}
	}
	return "", "", false
}

// boxMark is the mark of the "[?]" rest starts with, or 0.
func boxMark(rest string) byte {
	if len(rest) < 3 || rest[0] != '[' || rest[2] != ']' {
		return 0
	}
	return rest[1]
}

// ToggleLine is the line with its checkbox toggled: "[ ]" becomes "[x]",
// "[x]", "[X]" and "[-]" become "[ ]", a list item without a box gains one
// and any other line gains "- [ ] " in front.
func ToggleLine(line string) string {
	head, rest, ok := splitItem(line)
	if !ok {
		return "- [ ] " + line
	}
	switch boxMark(rest) {
	case ' ':
		return head + "[x]" + rest[3:]
	case 'x', 'X', '-':
		return head + "[ ]" + rest[3:]
	}
	return head + "[ ] " + rest
}

// IsBoxLine reports whether line is a list item carrying a box.
func IsBoxLine(line string) bool {
	_, rest, ok := splitItem(line)
	return ok && strings.IndexByte(" xX-", boxMark(rest)) >= 0
}

// RemoveBox is the line without its checkbox ("- [x] task" becomes
// "- task"); a line without a box is returned unchanged.
func RemoveBox(line string) string {
	if !IsBoxLine(line) {
		return line
	}
	head, rest, _ := splitItem(line)
	rest = rest[3:]
	if rest != "" && strings.ContainsRune(" \t\r\n\v\f", rune(rest[0])) {
		rest = rest[1:]
	}
	return head + rest
}

// ToggleCheckboxes toggles the box of the cursor's line, or of every line
// the VISUAL selection touches (the selection stays), as one change. It
// reports whether the text changed.
func (m *Model) ToggleCheckboxes() bool { return m.mapRows(ToggleLine) }

// RemoveCheckboxes removes the box of the cursor's line or of the VISUAL
// selection's lines, as one change. It reports whether the text changed.
func (m *Model) RemoveCheckboxes() bool { return m.mapRows(RemoveBox) }

// mapRows rewrites the cursor's row, or the rows of a VISUAL selection, with
// f as one change.
func (m *Model) mapRows(f func(string) string) (changed bool) {
	if m.page.ID == "" {
		return false
	}
	first, _ := m.textarea.RowCol()
	last := first
	if m.mode == ModeVisual || m.mode == ModeVisualLine {
		first, last = m.textarea.SelectedRows()
	}
	m.edit(func() { changed = m.textarea.MapLines(first, last, f) })
	return changed
}

// fenceRun is the character and length of the fence run text opens with,
// or 0 when text is not a fence.
func fenceRun(text string) (byte, int) {
	if text == "" || (text[0] != '`' && text[0] != '~') {
		return 0, 0
	}
	n := 1
	for n < len(text) && text[n] == text[0] {
		n++
	}
	if n < 3 {
		return 0, 0
	}
	return text[0], n
}

// inFence reports whether row sits inside a fenced code block, or opens
// one. A fence closes only on its own character, with a run at least as
// long as the opening one and nothing after it (CommonMark).
func inFence(rowText func(int) string, row int) bool {
	var open byte
	openLen := 0
	for r := range row {
		t := strings.TrimSpace(rowText(r))
		c, n := fenceRun(t)
		switch {
		case open != 0:
			if c == open && n >= openLen && strings.TrimSpace(t[n:]) == "" {
				open = 0
			}
		case c != 0:
			open, openLen = c, n
		}
	}
	if open != 0 {
		return true
	}
	c, _ := fenceRun(strings.TrimSpace(rowText(row)))
	return c != 0
}

// backtickRun is the start and length of the first run of backticks at or
// after byte from, or -1.
func backtickRun(line string, from int) (int, int) {
	i := strings.IndexByte(line[from:], '`')
	if i < 0 {
		return -1, 0
	}
	i += from
	n := 1
	for i+n < len(line) && line[i+n] == '`' {
		n++
	}
	return i, n
}

// inInlineCode reports whether byte at of line is inside a `code` span: a
// run of N backticks opens and the next run of exactly N closes.
func inInlineCode(line string, at int) bool {
	for from := 0; ; {
		open, n := backtickRun(line, from)
		if open < 0 {
			return false
		}
		closing := -1
		for s := open + n; ; {
			i, k := backtickRun(line, s)
			if i < 0 || k == n {
				closing = i
				break
			}
			s = i + k
		}
		if closing < 0 {
			return false
		}
		if at >= open && at < closing+n {
			return true
		}
		from = closing + n
	}
}

// wikiLinkAt is the [[link]] (or ![[embed]]) under rune column col of
// line, unless it is inside inline code.
func wikiLinkAt(line string, col int) (core.WikiLink, bool) {
	runes := []rune(line)
	at := len(string(runes[:min(max(col, 0), len(runes))]))
	if inInlineCode(line, at) {
		return core.WikiLink{}, false
	}
	for i := 0; i < len(line); i++ {
		if line[i] != '[' && line[i] != '!' {
			continue
		}
		if l, n, ok := core.WikiLinkAt(line[i:]); ok {
			if at >= i && at < i+n {
				return l, true
			}
			i += n - 1
		}
	}
	return core.WikiLink{}, false
}

// enter is Enter in NORMAL, as the LazyVim companion's: follow the [[link]]
// under the cursor outside code, else toggle the line's box, else move down.
func (m *Model) enter() tea.Cmd {
	row, col := m.textarea.RowCol()
	line := m.textarea.CurrentLine()
	if !inFence(m.textarea.RowText, row) {
		if l, ok := wikiLinkAt(line, col); ok {
			return func() tea.Msg { return FollowWikiLinkMsg{Link: l} }
		}
	}
	if IsBoxLine(line) {
		m.ToggleCheckboxes()
		return nil
	}
	m.textarea.MoveCursorBy(1)
	return nil
}
