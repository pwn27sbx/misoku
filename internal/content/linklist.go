package content

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// linkList is the inline list of "[[" link suggestions: the text typed after
// the "[[" decides what it offers, and sel is the item Enter inserts.
type linkList struct {
	sel     int
	offset  int
	items   []core.LinkSuggestion
	version uint64 // the text the items were worked out for
}

// OpenLinkList opens the inline list of link suggestions for the "[[" before
// the cursor, reading the vault's notes from src. The app reads them in a
// command: Update never touches the disk. Nothing happens when no "[[" is
// open under the cursor.
func (m *Model) OpenLinkList(src []core.LinkNote) {
	m.linkSrc = src
	if _, ok := m.linkListQuery(); !ok {
		m.linkList = nil
		return
	}
	m.linkList = &linkList{}
	m.refreshLinkList()
}

// LinkListOpen reports whether the inline link list is showing.
func (m Model) LinkListOpen() bool { return m.linkList != nil }

// linkListQuery is the text typed after the "[[" the cursor is in, up to the
// cursor. ok is false when the cursor is not inside a link: no "[[" before it
// on its line, or a "]]" typed.
func (m Model) linkListQuery() (string, bool) {
	before := m.textarea.LineBeforeCursor()
	i := strings.LastIndex(before, "[[")
	if i < 0 {
		return "", false
	}
	q := before[i+2:]
	if strings.Contains(q, "]]") {
		return "", false
	}
	return q, true
}

// refreshLinkList works the list out again from the text under the cursor,
// and closes it when the cursor left the link. The suggestions are only
// recomputed when the text changed.
func (m *Model) refreshLinkList() {
	if m.linkList == nil {
		return
	}
	q, ok := m.linkListQuery()
	if !ok {
		m.linkList = nil
		return
	}
	if m.linkList.items == nil || m.linkList.version != m.textarea.Version() {
		m.linkList.items = core.LinkSuggestions(m.linkSrc, m.page.ID, m.textarea.Value(), q)
		m.linkList.version = m.textarea.Version()
	}
	if n := len(m.linkList.items); m.linkList.sel >= n {
		m.linkList.sel = max(n-1, 0)
	}
	if m.linkList.sel < m.linkList.offset {
		m.linkList.offset = m.linkList.sel
	}
	if m.linkList.sel >= m.linkList.offset+linkListMaxRows {
		m.linkList.offset = m.linkList.sel - linkListMaxRows + 1
	}
}

// updateLinkList handles a key while the list is open. handled=false lets the
// key reach the editor, which types it and filters the list.
func (m *Model) updateLinkList(k tea.KeyMsg) (tea.Cmd, bool) {
	switch k.String() {
	case "esc":
		m.linkList = nil
		return nil, true
	case "up", "ctrl+p", "ctrl+k":
		m.moveLinkList(-1)
		return nil, true
	case "down", "ctrl+n", "ctrl+j":
		m.moveLinkList(1)
		return nil, true
	case "enter":
		return m.applyLinkList(false)
	case "tab":
		return m.applyLinkList(true)
	}
	return nil, false
}

// applyLinkList inserts the selected suggestion, or the typed name when it
// matches no note. passThrough is Tab's: with nothing to insert it reaches
// the editor, which indents, while Enter never does.
func (m *Model) applyLinkList(passThrough bool) (tea.Cmd, bool) {
	q, ok := m.linkListQuery()
	if !ok {
		m.linkList = nil
		return nil, false
	}
	if it, found := m.selectedLinkItem(); found {
		m.insertLinkSuggestion(q, it.Insert)
		return nil, true
	}
	m.linkList = nil
	done := core.LinkCompletion(q)
	if done == "" {
		return nil, !passThrough
	}
	m.insertLinkSuggestion(q, done)
	return nil, true
}

// selectedLinkItem is the suggestion Enter inserts, if there is one.
func (m Model) selectedLinkItem() (core.LinkSuggestion, bool) {
	if m.linkList == nil || m.linkList.sel < 0 || m.linkList.sel >= len(m.linkList.items) {
		return core.LinkSuggestion{}, false
	}
	return m.linkList.items[m.linkList.sel], true
}

// insertLinkSuggestion replaces the text typed after the "[[" with insert,
// closing the link. It is part of the INSERT session, not one undo step.
func (m *Model) insertLinkSuggestion(query, insert string) {
	ta := &m.textarea
	ta.DeleteBefore(len([]rune(query)))
	ta.InsertString(insert + "]]")
	m.linkList = nil
	ta.EnsureVisible()
}

// moveLinkList moves the selection, keeping it on screen.
func (m *Model) moveLinkList(delta int) {
	if m.linkList == nil || len(m.linkList.items) == 0 {
		return
	}
	n := len(m.linkList.items)
	m.linkList.sel = (m.linkList.sel + delta + n) % n
	if m.linkList.sel < m.linkList.offset {
		m.linkList.offset = m.linkList.sel
	}
	if m.linkList.sel >= m.linkList.offset+linkListMaxRows {
		m.linkList.offset = m.linkList.sel - linkListMaxRows + 1
	}
}
