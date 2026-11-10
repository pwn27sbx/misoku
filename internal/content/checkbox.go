package content

// Checkboxes and the Enter key, ported from the LazyVim companion
// (~/.config/nvim/lua/config/misoku.lua: toggle_line, uncheck_line,
// is_checkbox_line, in_fence, code_spans, link_at) so both editors act
// alike on the same note.

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

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

// ToggleLineAt toggles the box of document line row, as one change that u
// undoes, for the preview's clicks. It reports whether the text changed.
func (m *Model) ToggleLineAt(row int) (changed bool) {
	m.edit(func() { changed = m.textarea.MapLines(row, row, ToggleLine) })
	return changed
}

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

// completeTypedBox turns "- [] " into the standard "- [ ] " when k is the
// space that just finished it. It waits for the space, not the "]", because
// a task always has one there and a link with empty text ("- [](url)") never
// does. A paste stays as it came: its k.String() is bracketed.
func (m *Model) completeTypedBox(k tea.KeyMsg) {
	if k.String() != " " {
		return
	}
	before := m.textarea.LineBeforeCursor()
	head := listItem.FindString(before)
	if head == "" || strings.TrimLeft(before[len(head):], " \t") != "[] " {
		return
	}
	if row, _ := m.textarea.RowCol(); inFence(m.textarea.RowText, row) {
		return
	}
	m.textarea.DeleteBefore(len("] "))
	m.textarea.InsertString(" ] ")
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
// one.
func inFence(rowText func(int) string, row int) bool {
	return fencedRows(rowText, row+1)[row]
}

// fencedRows reports, for each of the first n rows, whether it sits inside
// a fenced code block or opens one. A fence closes only on its own
// character, with a run at least as long as the opening one and nothing
// after it (CommonMark).
func fencedRows(rowText func(int) string, n int) []bool {
	fenced := make([]bool, n)
	var open byte
	openLen := 0
	for r := range n {
		t := strings.TrimSpace(rowText(r))
		c, k := fenceRun(t)
		fenced[r] = open != 0 || c != 0
		switch {
		case open != 0:
			if c == open && k >= openLen && strings.TrimSpace(t[k:]) == "" {
				open = 0
			}
		case c != 0:
			open, openLen = c, k
		}
	}
	return fenced
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

// wikiLinkSpans calls visit with each [[link]] (or ![[embed]]) of line and
// the byte offset and length of its text, left to right.
func wikiLinkSpans(line string, visit func(l core.WikiLink, at, n int)) {
	for i := 0; i < len(line); i++ {
		if line[i] != '[' && line[i] != '!' {
			continue
		}
		if l, n, ok := core.WikiLinkAt(line[i:]); ok {
			visit(l, i, n)
			i += n - 1
		}
	}
}

// wikiLinkAt is the [[link]] (or ![[embed]]) under rune column col of
// line, unless it is inside inline code.
func wikiLinkAt(line string, col int) (link core.WikiLink, found bool) {
	runes := []rune(line)
	at := len(string(runes[:min(max(col, 0), len(runes))]))
	if inInlineCode(line, at) {
		return core.WikiLink{}, false
	}
	wikiLinkSpans(line, func(l core.WikiLink, i, n int) {
		if !found && at >= i && at < i+n {
			link, found = l, true
		}
	})
	return link, found
}

// linkStarts is the row and rune column of the start of every [[link]] (or
// ![[embed]]) of the first n rows, in order, leaving out those in fenced or
// inline code, as the LazyVim companion's link_positions.
func linkStarts(rowText func(int) string, n int) [][2]int {
	var starts [][2]int
	for row, fenced := range fencedRows(rowText, n) {
		if fenced {
			continue
		}
		line := rowText(row)
		wikiLinkSpans(line, func(_ core.WikiLink, i, _ int) {
			if !inInlineCode(line, i) {
				starts = append(starts, [2]int{row, utf8.RuneCountInString(line[:i])})
			}
		})
	}
	return starts
}

// JumpLink moves the cursor to the start of the next (dir > 0) or previous
// [[link]] outside code, wrapping around the note, as the LazyVim
// companion's ]o and [o. It reports false, and leaves the cursor, when the
// note has no such link.
func (m *Model) JumpLink(dir int) bool {
	starts := linkStarts(m.textarea.RowText, m.textarea.LineCount())
	if len(starts) == 0 {
		return false
	}
	row, col := m.textarea.RowCol()
	after := func(p [2]int) bool { return p[0] > row || (p[0] == row && p[1] > col) }
	target := starts[0]
	if dir > 0 {
		if i := slices.IndexFunc(starts, after); i >= 0 {
			target = starts[i]
		}
	} else {
		target = starts[len(starts)-1]
		for i := len(starts) - 1; i >= 0; i-- {
			if p := starts[i]; p[0] < row || (p[0] == row && p[1] < col) {
				target = p
				break
			}
		}
	}
	m.textarea.SetRowCol(target[0], target[1])
	return true
}

// enter is Enter in NORMAL, as the LazyVim companion's: follow the [[link]]
// under the cursor outside code, else toggle the line's box (not in the
// read-only view), else move down.
func (m *Model) enter() tea.Cmd {
	row, col := m.textarea.RowCol()
	line := m.textarea.CurrentLine()
	if !inFence(m.textarea.RowText, row) {
		if l, ok := wikiLinkAt(line, col); ok {
			return func() tea.Msg { return FollowWikiLinkMsg{Link: l} }
		}
	}
	if IsBoxLine(line) && !m.readOnly {
		m.ToggleCheckboxes()
		return nil
	}
	m.textarea.MoveCursorBy(1)
	return nil
}
