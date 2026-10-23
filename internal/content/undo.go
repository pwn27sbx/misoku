package content

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// Undo limits per tab (D5): whichever is reached first drops the oldest
// changes. The byte limit counts every snapshot kept, the current text too.
const (
	undoMaxChanges = 200
	undoMaxBytes   = 8 << 20
)

// History is one tab's undo history. It keeps a copy of the whole text
// after each change (D5), in memory only. A single editor serves every tab,
// so the app keeps one History per tab and attaches it with SetHistory.
type History struct {
	id string // note the snapshots belong to
	// states[pos] is the text as of the last recorded change; states before
	// it can be undone to, states after it redone to.
	states []string
	pos    int
	bytes  int // total length of states
	// dropped counts the changes trimmed off the front, so a change keeps
	// its number ("before #3") however many older ones were dropped.
	dropped int
	// ver is the editor text version last compared with states[pos]: while
	// the text keeps it, nothing changed and nothing needs copying.
	ver uint64
}

// NewHistory returns an empty history; the first text it is attached to
// becomes its starting point.
func NewHistory() *History { return &History{} }

// Rename moves the history to the note's new ID, so renaming a note or
// saving a draft under a name keeps what can be undone.
func (h *History) Rename(id string) { h.id = id }

// reset drops every change and starts over from text.
func (h *History) reset(id, text string) {
	h.id, h.states, h.pos, h.bytes, h.dropped = id, []string{text}, 0, len(text), 0
}

// push records text as a new change, dropping the changes that were undone
// and then the oldest ones beyond the limits.
func (h *History) push(text string) {
	for _, s := range h.states[h.pos+1:] {
		h.bytes -= len(s)
	}
	clear(h.states[h.pos+1:])
	h.states = append(h.states[:h.pos+1], text)
	h.pos++
	h.bytes += len(text)
	for len(h.states) > 1 && (len(h.states)-1 > undoMaxChanges || h.bytes > undoMaxBytes) {
		h.bytes -= len(h.states[0])
		h.states[0] = ""
		h.states = h.states[1:]
		h.pos--
		h.dropped++
	}
}

// SetHistory attaches the undo history of the tab about to be shown. Call
// it before SetBuffer, which checks the history belongs to the text loaded.
// nil attaches a new, empty one.
func (m *Model) SetHistory(h *History) {
	if h == nil {
		h = NewHistory()
	}
	m.hist = h
}

// ResetHistory drops every change of the attached history, keeping the
// current text as its starting point (a reload from disk, D7).
func (m *Model) ResetHistory() {
	if m.hist == nil {
		return
	}
	m.hist.reset(m.page.ID, m.textarea.Value())
	m.hist.ver = m.textarea.Version()
}

// syncHistory runs after a note is loaded: loading is not a change. A
// history that belongs to another note, or whose current text is not the
// one loaded, starts over: undoing it would put foreign text in this note.
func (m *Model) syncHistory() {
	h := m.hist
	if h == nil {
		return
	}
	text := m.textarea.Value()
	if h.id != m.page.ID || len(h.states) == 0 || h.states[h.pos] != text {
		h.reset(m.page.ID, text)
	}
	h.ver = m.textarea.Version()
}

// record closes the change in progress: if the text differs from the last
// recorded one, it becomes a new change. Cheap when nothing was edited.
func (m *Model) record() {
	h := m.hist
	if h == nil || m.page.ID == "" || h.ver == m.textarea.Version() {
		return
	}
	h.ver = m.textarea.Version()
	text := m.textarea.Value()
	if len(h.states) == 0 {
		h.reset(m.page.ID, text)
		return
	}
	if text != h.states[h.pos] {
		h.push(text)
	}
}

// CommitChange records the edit in progress, an INSERT session included,
// as one change. The app calls it before the tab changes.
func (m *Model) CommitChange() { m.record() }

// edit runs an edit the app makes (preview, media, tables) as one change
// of its own. Inside an INSERT session it is part of that session.
func (m *Model) edit(f func()) {
	if m.mode == ModeInsert {
		f()
		return
	}
	m.record()
	f()
	m.record()
}

// undo puts back the text before the last change (Vim's u).
func (m *Model) undo() tea.Cmd {
	m.record() // an edit not recorded yet becomes a change, never lost
	h := m.hist
	if h == nil || h.pos == 0 {
		return statusCmd("Already at oldest change")
	}
	n := h.dropped + h.pos
	h.pos--
	m.restore(h.states[h.pos])
	return statusCmd(fmt.Sprintf("1 change; before #%d", n))
}

// redo puts back the change undo took away (Vim's Ctrl+R).
func (m *Model) redo() tea.Cmd {
	m.record()
	h := m.hist
	if h == nil || h.pos >= len(h.states)-1 {
		return statusCmd("Already at newest change")
	}
	h.pos++
	m.restore(h.states[h.pos])
	return statusCmd(fmt.Sprintf("1 change; after #%d", h.dropped+h.pos))
}

// restore shows text, with the cursor on the first line that changed.
func (m *Model) restore(text string) {
	old := m.textarea.Value()
	row, col := m.textarea.RowCol()
	if i := commonPrefix(old, text); i < len(old) || i < len(text) {
		row = strings.Count(text[:i], "\n")
		col = utf8.RuneCountInString(text[strings.LastIndexByte(text[:i], '\n')+1 : i])
	}
	m.textarea.SetValue(text)
	m.textarea.SetRowCol(row, col)
	m.textarea.EnsureVisible()
	m.hist.ver = m.textarea.Version()
}

// commonPrefix is the length of the longest common prefix of a and b,
// backed off to a rune boundary.
func commonPrefix(a, b string) int {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	for i > 0 && i < len(b) && !utf8.RuneStart(b[i]) {
		i--
	}
	return i
}

func statusCmd(text string) tea.Cmd {
	return func() tea.Msg { return core.StatusMsg{Text: text} }
}
