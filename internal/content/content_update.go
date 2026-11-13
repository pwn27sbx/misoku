package content

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/clipimage"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/textarea"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// Update processes key and mouse messages (mouse coordinates relative to the
// pane's top-left corner) and handles Vim editing modes. Outside INSERT,
// each command that changed the text is recorded as one change for u.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	m, cmd := m.update(msg)
	if m.mode != ModeInsert {
		m.record()
	}
	return m, cmd
}

func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	if mouse, ok := msg.(tea.MouseMsg); ok {
		m, cmd := m.handleMouse(mouse)
		m.refreshLinkList() // a click can move the cursor out of the link
		return m, cmd
	}
	if m.page.ID == "" {
		return m, nil
	}

	if k, ok := msg.(tea.KeyMsg); ok && m.textarea.HasSelection() &&
		m.mode != ModeVisual && m.mode != ModeVisualLine {
		m.textarea.ClearSelection() // a key press drops a mouse selection
		_ = k
	}

	switch m.mode {
	case ModeVisual, ModeVisualLine:
		if k, ok := msg.(tea.KeyMsg); ok {
			return m.handleVisualKey(k)
		}
		return m, nil
	case ModeInsert:
		k, isKey := msg.(tea.KeyMsg)
		if isKey && m.linkList != nil {
			if cmd, handled := m.updateLinkList(k); handled {
				return m, cmd
			}
		}
		if isKey && m.slash != nil {
			if cmd, handled := m.updateSlash(k); handled {
				return m, cmd
			}
		}
		// A picture in the clipboard becomes a link to the file; anything else
		// is the text paste of today. The read happens in a command, never
		// here. An empty bracketed paste is the shape a terminal would use for
		// a clipboard that carries no text.
		if isKey && m.slash == nil && (k.String() == "ctrl+v" || (k.Paste && len(k.Runes) == 0)) {
			return m, m.pasteCmd()
		}
		if isKey && (k.String() == "tab" || k.String() == "shift+tab") {
			if m.nextCell(k.String() == "tab") {
				m.reportAction("cell_next", "")
			} else {
				m.indent(k.String() == "tab")
			}
			return m, nil
		}
		if isKey && k.String() == "esc" {
			m.mode = ModeNormal
			m.closeLinkList()
			m.textarea.CharLeft()
			return m, m.focusTextarea()
		}
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		if isKey {
			m.completeTypedBox(k)
		}
		m.textarea.EnsureVisible()
		if isKey {
			m.alignTableWhileTyping(k)
			m.refreshLinkList()
			if m.slash != nil {
				m.afterSlashKey()
			} else if m.linkList == nil {
				m.maybeOpenSlash(k)
			}
			if m.linkList == nil && k.Type == tea.KeyRunes && string(k.Runes) == "[" && strings.HasSuffix(m.textarea.LineBeforeCursor(), "[[") {
				return m, tea.Batch(cmd, func() tea.Msg { return core.SlashActionMsg{Action: "wikilink"} })
			}
		}
		return m, cmd

	case ModeCommand:
		k, ok := msg.(tea.KeyMsg)
		if ok {
			switch k.String() {
			case "esc":
				m.closeCommandLine()
				return m, m.focusTextarea()
			case "backspace":
				if m.cmdInput.Value() == "" {
					m.closeCommandLine()
					return m, m.focusTextarea()
				}
			case "enter":
				if m.search {
					query := m.cmdInput.Value()
					m.closeCommandLine()
					focus := m.focusTextarea()
					if cmd := m.searchCmd(query); cmd != nil {
						return m, tea.Batch(focus, cmd)
					}
					return m, focus
				}
				if op, ok := tableCommands[strings.TrimSpace(m.cmdInput.Value())]; ok {
					m.closeCommandLine()
					focus := m.focusTextarea()
					if !m.tableOp(op) {
						return m, tea.Batch(focus, func() tea.Msg { return core.StatusMsg{Text: i18n.T(i18n.EditorNotInTable), Error: true} })
					}
					return m, focus
				}
				out := parseVimCommand(m.cmdInput.Value(), m.textarea.Value())
				m.reportCommandError(out)
				m.closeCommandLine()
				focus := m.focusTextarea()
				if out == nil {
					return m, focus
				}
				return m, tea.Batch(focus, func() tea.Msg { return out })
			}
		}
		var cmd tea.Cmd
		m.cmdInput, cmd = m.cmdInput.Update(msg)
		return m, cmd
	}

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	return m.handleNormalKey(k)
}

func (m Model) handleNormalKey(k tea.KeyMsg) (Model, tea.Cmd) {
	ta := &m.textarea
	s := k.String()
	m.reportMotion(s)

	if m.pending != "" {
		combo := m.pending + s
		m.pending = ""
		switch combo {
		case "gg":
			ta.GotoTop()
			m.reportAction("motion", "file-bounds")
		case "dd":
			if from, to, ok := m.cursorBlock(); ok {
				text := m.blockText(from, to)
				m.deleteBlock(from, to)
				return m, copyCmd(text)
			}
			ta.DeleteLine()
		case "yy":
			m.reportAction("yank", "line")
			if from, to, ok := m.cursorBlock(); ok {
				return m, copyCmd(m.blockText(from, to))
			}
			return m, copyCmd(ta.CurrentLine() + "\n")
		case ">>", "<<":
			if from, to, ok := m.cursorBlock(); ok && m.shiftable(from) {
				m.shiftBlock(from, to, combo == ">>")
			}
		case "gd", "gx":
			line := ta.CurrentLine()
			_, col := ta.RowCol()
			return m, func() tea.Msg { return core.FollowLinkMsg{Line: line, Col: col, Key: combo} }
		}
		return m, nil
	}

	switch s {
	case "i":
		return m, m.enterInsert()
	case "a":
		ta.CharRight()
		return m, m.enterInsert()
	case "A":
		ta.CursorEnd()
		return m, m.enterInsert()
	case "I":
		ta.CursorStart()
		return m, m.enterInsert()
	case "o", "O":
		if from, to, ok := m.cursorBlock(); ok {
			m.newBlock(from, to, s == "O", []string{""})
		} else if s == "o" {
			ta.OpenLineBelow()
		} else {
			ta.OpenLineAbove()
		}
		return m, m.enterInsert()
	case ":":
		m.mode = ModeCommand
		m.search = false
		m.cmdInput.Prompt = ":"
		m.cmdInput.SetValue("")
		m.textarea.Blur()
		m.reportAction("cmdline", "")
		return m, m.cmdInput.Focus()
	case "/":
		return m, m.openSearch()
	case "n":
		return m, m.repeatCmd(true)
	case "N":
		return m, m.repeatCmd(false)

	case "h", "left":
		ta.CharLeft()
	case "l", "right":
		ta.CharRight()
	case "j", "down":
		ta.MoveCursorBy(1)
	case "enter":
		return m, m.enter()
	case "k", "up":
		ta.MoveCursorBy(-1)
	case "w":
		ta.WordForward()
	case "b":
		ta.WordBackward()
	case "0", "^", "home":
		ta.CursorStart()
	case "$", "end":
		ta.CursorEnd()
	case "G":
		if l, ok := m.lastBlockLine(); ok {
			ta.SetRowCol(l, 0)
			ta.EnsureVisible()
		} else {
			ta.GotoBottom()
		}
	case "g", "d", "y":
		m.pending = s
	case ">", "<":
		if from, _, ok := m.cursorBlock(); ok && m.shiftable(from) {
			m.pending = s // >> and << nest a rendered list item
		}
	case "v", "V":
		m.mode = ModeVisual
		if s == "V" {
			m.mode = ModeVisualLine
		}
		ta.SelectLinewise = s == "V"
		ta.StartSelection()
	case "p":
		return m, m.paste(false)
	case "P":
		if _, _, ok := m.cursorBlock(); ok {
			return m, m.paste(true)
		}
	case "u":
		return m, m.undo()
	case "ctrl+r":
		return m, m.redo()
	case "ctrl+d":
		ta.ScrollBy(m.halfPage())
		ta.MoveCursorBy(m.halfPage())
	case "ctrl+u":
		ta.ScrollBy(-m.halfPage())
		ta.MoveCursorBy(-m.halfPage())
	case "pgdown", "ctrl+f":
		ta.MoveCursorBy(ta.Height())
	case "pgup":
		ta.MoveCursorBy(-ta.Height())
	case "ctrl+e":
		ta.ScrollBy(1)
	case "ctrl+y":
		ta.ScrollBy(-1)
	case "x", "delete":
		if _, _, ok := m.cursorBlock(); ok {
			return m, nil // a rendered block hides the column x would delete
		}
		ta.DeleteCharForward()
	}
	return m, nil
}

// reportMotion reports the movement key s, if it moves the cursor: the
// lessons teach them by groups (hjkl, arrows, words, …). gg is reported where
// its two keys complete.
func (m Model) reportMotion(s string) {
	if m.actions == nil || !m.actions.reporting() {
		return // the tutorial is not playing: nothing to classify
	}
	switch s {
	case "h", "j", "k", "l":
		m.reportAction("motion", "hjkl")
	case "left", "right", "up", "down":
		m.reportAction("motion", "arrows")
	case "0", "^", "home", "$", "end":
		m.reportAction("motion", "line-bounds")
	case "G":
		m.reportAction("motion", "file-bounds")
	case "w", "b":
		m.reportAction("motion", "words")
	case "ctrl+d", "ctrl+u":
		m.reportAction("motion", "half-page")
	case "pgdown", "ctrl+f", "pgup":
		m.reportAction("motion", "page")
	case "ctrl+e", "ctrl+y":
		m.reportAction("motion", "scroll-line")
	}
}

func (m Model) handleMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	if m.page.ID == "" {
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		// A wheel ends a text drag, as the app does: the selection stays,
		// the next motion no longer extends it and the reveal is no longer
		// frozen. The release that would have ended it never comes here.
		m.dragging = false
		if msg.Button == tea.MouseButtonWheelUp {
			m.textarea.ScrollBy(-3)
		} else {
			m.textarea.ScrollBy(3)
		}
		return m, nil
	case tea.MouseButtonLeft:
		if m.slash != nil && msg.Action == tea.MouseActionPress {
			if cmd, hit := m.clickSlash(msg.X, msg.Y); hit {
				return m, cmd
			}
		}
		if m.linkList != nil && msg.Action == tea.MouseActionPress {
			if cmd, hit := m.clickLinkList(msg.X, msg.Y); hit {
				return m, cmd
			}
		}
		switch msg.Action {
		case tea.MouseActionPress:
			if m.mode == ModeVisual || m.mode == ModeVisualLine {
				m.mode = ModeNormal
			}
			m.textarea.ClearSelection()
			m.textarea.SelectLinewise = false
			m.clickAt(msg.X, msg.Y)
			m.dragRow, m.dragCol = m.textarea.RowCol()
			m.dragging = true
		case tea.MouseActionMotion:
			if !m.dragging {
				break
			}
			m.clickAt(msg.X, msg.Y)
			if r, c := m.textarea.RowCol(); !m.textarea.HasSelection() && (r != m.dragRow || c != m.dragCol) {
				m.textarea.SetAnchor(m.dragRow, m.dragCol)
			}
		case tea.MouseActionRelease:
			m.dragging = false
			if m.textarea.HasSelection() {
				// Auto-copy on release, like selecting text in Claude Code.
				return m, copyCmd(m.textarea.SelectedText())
			}
		}
	}
	return m, nil
}

func copyCmd(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	return func() tea.Msg { return core.CopyMsg{Text: text} }
}

// handleVisualKey runs Vim visual mode: motions extend the selection,
// y copies, d/x cut, Esc cancels.
func (m Model) handleVisualKey(k tea.KeyMsg) (Model, tea.Cmd) {
	ta := &m.textarea
	exit := func() {
		m.mode = ModeNormal
		ta.ClearSelection()
		ta.SelectLinewise = false
	}
	switch s := k.String(); s {
	case "esc", "ctrl+c":
		exit()
		return m, nil
	case "v", "V":
		if (s == "v") == (m.mode == ModeVisual) {
			exit()
			return m, nil
		}
		m.mode = map[string]VimMode{"v": ModeVisual, "V": ModeVisualLine}[s]
		ta.SelectLinewise = s == "V"
		return m, nil
	case "y":
		text := ta.SelectedText()
		exit()
		return m, copyCmd(text)
	case "d", "x", "delete":
		text := ta.SelectedText()
		ta.DeleteSelection()
		exit()
		return m, copyCmd(text)
	case "h", "j", "k", "l", "left", "right", "up", "down", "w", "b", "0", "^", "$",
		"home", "end", "G", "g", "ctrl+d", "ctrl+u", "pgdown", "pgup":
		mode := m.mode
		m.mode = ModeNormal
		m, _ = m.handleNormalKey(k)
		m.mode = mode
	}
	return m, nil
}

// paste puts the clipboard after the cursor (Vim's p), or before it when
// before is set (P, which only a rendered block offers): a text ending in a
// newline as whole lines, anything else into the line.
func (m *Model) paste(before bool) tea.Cmd {
	text, err := readClipboard()
	if err != nil || text == "" {
		return func() tea.Msg { return core.StatusMsg{Text: i18n.T(i18n.EditorClipboardEmpty), Error: err != nil} }
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if from, to, ok := m.cursorBlock(); ok {
		// On a rendered block the text goes in as a block of its own, below
		// the block or above it.
		m.newBlock(from, to, before, strings.Split(strings.TrimSuffix(text, "\n"), "\n"))
		m.reportAction("paste", "")
		return nil
	}
	switch {
	case strings.HasSuffix(text, "\n") && before:
		m.textarea.OpenLineAbove()
		m.textarea.InsertString(strings.TrimSuffix(text, "\n"))
	case strings.HasSuffix(text, "\n"):
		m.textarea.OpenLineBelow()
		m.textarea.InsertString(strings.TrimSuffix(text, "\n"))
	default:
		m.textarea.CharRight()
		m.textarea.InsertString(text)
	}
	m.textarea.EnsureVisible()
	m.reportAction("paste", "")
	return nil
}

// View renders the editor at exactly width × height.

// readClipboard reads the system clipboard; tests replace it.
var readClipboard = clipboard.ReadAll

// pasteText is the text half of the paste: the textarea's own command reads the
// clipboard and returns the message it would have produced before. Tests
// replace it, so they never touch a real clipboard.
var pasteText = textarea.Paste

// readClipboardImage reads a picture from the clipboard; tests replace it. It
// reports ok=false when the clipboard holds something else.
var readClipboardImage = func(ctx context.Context) (core.ClipboardImageMsg, bool, error) {
	data, ok, err := clipimage.Read(ctx, nil)
	if !ok {
		return core.ClipboardImageMsg{}, false, err
	}
	return core.ClipboardImageMsg{Path: data.Path, MIME: data.MIME, Ext: data.Ext, Bytes: data.Bytes}, true, nil
}

// SetClipboardImage replaces how a picture is read from the clipboard (nil
// restores the real one); the app uses it so tests never touch a clipboard.
func (m *Model) SetClipboardImage(f func(context.Context) (core.ClipboardImageMsg, bool, error)) {
	m.readImage = f
}

// PasteClipboard is the "paste image" action: the same command ctrl+v runs, for
// the key the user binds to it.
func (m Model) PasteClipboard() tea.Cmd { return m.pasteCmd() }

// pasteCmd reads the clipboard away from the Update loop: a picture becomes a
// ClipboardImageMsg for the app to attach, anything else the text paste Misoku
// has always done. Nothing here may run inside Update, and nothing is replayed
// from a key.
func (m Model) pasteCmd() tea.Cmd {
	read := m.readImage
	if read == nil {
		read = readClipboardImage
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		img, ok, err := read(ctx)
		if ok {
			m.reportAction("paste", "")
			return img
		}
		// No picture: paste text exactly as before. The clipboard of the text
		// path may work through a tool this package does not probe (xsel), so a
		// missing image tool must never take the text paste away from the user.
		msg := pasteText()
		if err != nil && errors.Is(err, clipimage.ErrNoTool) {
			if _, failed := msg.(error); failed || msg == nil {
				return core.StatusMsg{Text: i18n.T(i18n.EditorPasteNeeds, pasteHint()), Error: true}
			}
		}
		if _, refused := msg.(core.StatusMsg); !refused {
			m.reportAction("paste", "")
		}
		return msg
	}
}

// pasteHint names the clipboard tool to install for this session.
func pasteHint() string { return clipimage.Hint() }
