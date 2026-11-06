package content

// The read-only view: the note is drawn like the live view on every line,
// the cursor's included, and no key changes the text. Motions, search,
// visual selection with y, the command line and following links still work.

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
)

// normalEdits are the NORMAL-mode keys that change the text or start
// typing; d is listed because on its own it only starts dd.
var normalEdits = map[string]bool{
	"i": true, "a": true, "A": true, "I": true, "o": true, "O": true,
	"p": true, "u": true, "ctrl+r": true, "x": true, "delete": true, "d": true,
}

// SetReadOnly turns the read-only view on or off. On, keys that edit do
// nothing but say so, the cursor's line is drawn rendered like the others
// (with SetLive) and the real cursor is not placed; a pending INSERT or
// selection ends.
func (m *Model) SetReadOnly(on bool) {
	m.readOnly = on
	m.textarea.RenderCursorLine = on
	if !on {
		return
	}
	m.pending = ""
	switch m.mode {
	case ModeInsert:
		m.ExitInsert()
	case ModeVisual, ModeVisualLine:
		m.mode = ModeNormal
		m.textarea.ClearSelection()
		m.textarea.SelectLinewise = false
	}
}

// ReadOnly reports whether the read-only view is on.
func (m Model) ReadOnly() bool { return m.readOnly }

// editsText reports whether k, in the current mode, would change the text.
func (m Model) editsText(k tea.KeyMsg) bool {
	s := k.String()
	switch m.mode {
	case ModeNormal:
		// Only g and y wait for a second key here (d is refused first),
		// and gg, gd, gx and yy edit nothing.
		return m.pending == "" && normalEdits[s]
	case ModeVisual, ModeVisualLine:
		return s == "d" || s == "x" || s == "delete"
	case ModeCommand:
		_, table := tableCommands[strings.TrimSpace(m.cmdInput.Value())]
		return s == "enter" && table
	}
	return true // INSERT never starts in the read-only view
}

// ReadOnlyHint is the status shown for a refused edit, naming the key of
// the editor view as the user configured it; the app shows it for its own
// edit actions too.
func (m Model) ReadOnlyHint() tea.Cmd {
	text := "Read-only view"
	if k := m.keys.Hint(keymap.EditorView); k != "" {
		text = "Read-only · " + k + " to edit"
	}
	return func() tea.Msg { return core.StatusMsg{Text: text} }
}
