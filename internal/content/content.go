// Package content renders the Markdown editor pane (<Body/>), a Vim-style
// editor. The file name lives in the tabline only.
package content

import (
	"context"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/highlight"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/textarea"
	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the Content component state.
type Model struct {
	theme      theme.Theme
	keys       keymap.Map // only for the hints; the app dispatches shortcuts
	page       core.Page
	draft      bool
	textarea   textarea.Model
	cmdInput   textinput.Model
	mode       VimMode
	pending    string // first key of a two-key Vim command ("g", "d")
	slash      *slashMenu
	linkList   *linkList       // the inline "[[" suggestions, see linklist.go
	linkSrc    []core.LinkNote // the notes those suggestions read
	search     bool            // the command line is the "/" search prompt
	lastSearch string          // the text NORMAL's n and N repeat
	hl         *hlCache        // shared across copies so View can memoise
	live       bool            // the live view is on; see SetLive
	rendered   renderedState   // what ShowRendered installed
	hist       *History        // the shown tab's undo history (SetHistory)
	copies     *atomic.Int64
	actions    *actionLog // the actions the tutorial listens to; see actions.go
	dragging   bool
	dragRow    int
	dragCol    int
	// readImage reads a picture from the clipboard for ctrl+v; nil uses
	// clipimage. The app installs one so tests never touch a clipboard.
	readImage  func(context.Context) (core.ClipboardImageMsg, bool, error)
	width      int
	height     int
	margin     int // blank columns each side of the text; see SetMargin
	focused    bool
	dimmed     bool // greyed out while another pane has the focus
	realCursor bool // the terminal draws the cursor; see SetRealCursor

	// linkSeq numbers the "[[" triggers and linkPending is the trigger a
	// source read still waits for: a read that arrives after a close is
	// dropped, see RequestLinkList.
	linkSeq     uint64
	linkPending uint64
}

// New constructs a Content Model.
func New(th theme.Theme) Model {
	ta := textarea.New()
	ta.Placeholder = i18n.T(i18n.EditorPlaceholder)
	ta.Prompt = ""
	ta.ShowLineNumbers = true
	ta.EndOfBufferCharacter = '~'
	ta.HideBandWhileSelecting = true
	ta.WrapIndent = hangIndent // before any text: see the field
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.Cursor.Style = lipgloss.NewStyle().Foreground(th.Fg)
	configureTextareaStyles(&ta, th)

	km, _ := keymap.New(nil)
	return Model{
		theme:    th,
		keys:     km,
		textarea: ta,
		cmdInput: createCommandInput(th),
		mode:     ModeNormal,
		hl:       &hlCache{},
		hist:     NewHistory(),
		copies:   &atomic.Int64{},
		actions:  &actionLog{},
	}
}

// SetKeys sets the shortcuts named in the editor's hints.
func (m *Model) SetKeys(km keymap.Map) { m.keys = km }

// SetTheme switches colours.
func (m *Model) SetTheme(th theme.Theme) {
	m.theme = th
	m.textarea.Cursor.Style = lipgloss.NewStyle().Foreground(th.Fg)
	configureTextareaStyles(&m.textarea, th)
	// The textarea draws through a pointer to its focused or blurred style,
	// which can still point at an older copy's styles: re-apply the focus
	// state so it points at the ones just set.
	if m.textarea.Focused() {
		m.textarea.Focus()
	} else {
		m.textarea.Blur()
	}
	value := m.cmdInput.Value()
	m.cmdInput = createCommandInput(th)
	m.cmdInput.SetValue(value)
	if m.search {
		m.cmdInput.Prompt = "/" // the search prompt keeps its mark
	}
	if m.mode == ModeCommand && m.focused {
		m.cmdInput.Focus()
	}
}

// Init initializes the textarea model commands.
func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

// SetSize updates editor dimensions safely.
func (m *Model) SetSize(w, h int) {
	m.width = max(w, 0)
	m.height = max(h, 0)
	m.fitText()
	m.textarea.SetHeight(max(h, 1))
}

// SetMargin leaves n blank columns each side of the text, as the preview
// does around its page; 0, the default, lets the text fill the pane. The
// left margin is the textarea's prompt, so clicks and the cursor keep
// lining up with the text.
func (m *Model) SetMargin(n int) {
	m.margin = max(n, 0)
	m.textarea.Prompt = strings.Repeat(" ", m.margin)
	m.fitText()
}

// fitText sizes the textarea to the pane: the full width less one column
// for the cursor past the line's end, or less the right margin, which
// leaves room for it already. Call it after changing the prompt or the
// line numbers, which the textarea's width depends on.
func (m *Model) fitText() {
	w := m.width - 1
	if m.margin > 0 {
		w = m.width - m.margin
	}
	m.textarea.SetWidth(max(w, 10))
}

// SetPage loads a document into the editor. draft marks an unsaved buffer
// that has no file yet.
func (m *Model) SetPage(p core.Page) {
	m.SetBuffer(p, false)
}

// SetBuffer loads a buffer's text, resetting the cursor to the top.
// Loading is not a change: the attached history (SetHistory) is kept when
// it belongs to this note and text, and started over otherwise.
func (m *Model) SetBuffer(p core.Page, draft bool) {
	m.page = p
	m.draft = draft
	m.pending = ""
	m.slash = nil
	m.closeLinkList()
	m.textarea.SetValue(p.Content)
	m.textarea.GotoTop()
	m.syncHistory()
}

// SetDimmed greys the editor's text out while another pane has the focus.
func (m *Model) SetDimmed(d bool) { m.dimmed = d }

// SetFocused sets focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused {
		m.slash = nil
		m.closeLinkList()
		m.Blur()
		if m.mode == ModeCommand {
			m.closeCommandLine()
		}
	}
}

// Focus gives the editor keyboard focus in its current mode.
func (m *Model) Focus() tea.Cmd {
	m.focused = true
	if m.mode == ModeCommand {
		return m.cmdInput.Focus()
	}
	return m.focusTextarea()
}

func (m *Model) focusTextarea() tea.Cmd {
	cmd := m.textarea.Focus()
	cursorMode := cursor.CursorStatic
	if m.mode == ModeInsert && !m.realCursor {
		cursorMode = cursor.CursorBlink
	}
	return tea.Batch(cmd, m.textarea.Cursor.SetMode(cursorMode))
}

// SetRealCursor tells the editor that the terminal draws the cursor cell, so it
// stops drawing its own. The blink setting stays steady: a real cursor that
// blinks would disagree with the drawn one the rest of the program uses.
func (m *Model) SetRealCursor(on bool) {
	m.realCursor = on
	m.textarea.HideCursor = on
}

// CursorScreen returns the cell the terminal cursor belongs on, relative to the
// pane's top-left corner, or ok=false when there is none: no note open, the
// command line in charge, no focus, or the cell off screen.
func (m Model) CursorScreen() (x, y int, ok bool) {
	if m.page.ID == "" || !m.focused || m.mode == ModeCommand {
		return 0, 0, false
	}
	x, y, ok = m.textarea.CursorScreen()
	if !ok || x < 0 || y < 0 || x >= m.width || y >= m.height {
		return 0, 0, false
	}
	return x, y, true
}

// CursorOnRendered reports whether the cursor is on a block the view draws
// rendered, where the terminal shows no cursor: the band marks the block.
func (m Model) CursorOnRendered() bool { return m.textarea.CursorRendered() }

// Blur blurs inputs.
func (m *Model) Blur() {
	m.textarea.Blur()
	m.cmdInput.Blur()
}

// Mode returns the active Vim mode.
func (m Model) Mode() VimMode { return m.mode }

// ModeString returns the string representation of the active mode.
func (m Model) ModeString() string {
	switch m.mode {
	case ModeInsert:
		return "INSERT"
	case ModeCommand:
		return "COMMAND"
	case ModeVisual:
		return "VISUAL"
	case ModeVisualLine:
		return "V-LINE"
	default:
		return "NORMAL"
	}
}

// Value returns the current textarea content. It copies the whole note,
// so a caller that runs on every frame or key should check Version first.
func (m Model) Value() string {
	if m.copies != nil {
		m.copies.Add(1)
	}
	return m.textarea.Value()
}

// TextCopies counts the copies of the note Value has made, so a test can
// check that moving the cursor or redrawing copies nothing.
func (m Model) TextCopies() int64 {
	if m.copies == nil {
		return 0
	}
	return m.copies.Load()
}

// Version identifies the current text: it changes on every edit and stays
// the same while only the cursor or the view moves (textarea.Model.Version).
func (m Model) Version() uint64 { return m.textarea.Version() }

// HasPage reports whether a document is loaded.
func (m Model) HasPage() bool { return m.page.ID != "" }

// CursorPosition returns the 1-based line and column of the cursor.
func (m Model) CursorPosition() (int, int) { return m.textarea.CursorPosition() }

// LineCount returns the number of lines in the buffer.
func (m Model) LineCount() int { return m.textarea.LineCount() }

// CommandView renders the ":" prompt for the command-line row.
func (m Model) CommandView(width int) string {
	m.cmdInput.Width = max(width-2, 1)
	return m.cmdInput.View()
}

// EnterInsert switches to INSERT mode (used by the app, e.g. for a new
// buffer).
func (m *Model) EnterInsert() tea.Cmd {
	m.mode = ModeInsert
	if !m.focused {
		return nil
	}
	return m.focusTextarea()
}

// ExitInsert returns to NORMAL mode (e.g. when focus leaves the editor).
func (m *Model) ExitInsert() {
	if m.mode == ModeInsert {
		m.mode = ModeNormal
	}
	m.slash = nil
	m.closeLinkList()
}

func (m *Model) enterInsert() tea.Cmd {
	m.mode = ModeInsert
	return m.focusTextarea()
}

func (m *Model) halfPage() int { return max(m.textarea.Height()/2, 1) }

// listItem matches Markdown list and to-do lines, which Tab nests.
var listItem = regexp.MustCompile(`^\s*([-*+]|\d+[.)])\s`)

// indent handles Tab / Shift+Tab in INSERT mode: list lines are nested or
// un-nested as a whole; elsewhere Tab inserts two spaces.
func (m *Model) indent(in bool) {
	const width = 2
	switch {
	case !in:
		m.textarea.OutdentLine(width)
	case listItem.MatchString(m.textarea.CurrentLine()):
		m.textarea.IndentLine(width)
	default:
		m.textarea.InsertString("  ")
	}
	m.textarea.EnsureVisible()
}

// hlCache memoises syntax colours for the last text version and theme
// rendered, so a frame that only moved the cursor neither copies the note
// nor highlights it again. After an edit, the highlighter re-colours only
// the lines the edit reached.
type hlCache struct {
	version uint64
	// theme is the whole palette, not its name: a user theme can keep its
	// name while its colours change on disk.
	theme theme.Theme
	cols  highlight.Colors
	valid bool
	hl    highlight.Highlighter
}

func (c *hlCache) colors(m Model) highlight.Colors {
	if !c.valid || c.version != m.Version() || c.theme != m.theme {
		c.version, c.theme, c.cols, c.valid = m.Version(), m.theme, c.hl.Markdown(m.Value(), m.theme), true
	}
	return c.cols
}

// ReplaceText swaps the whole buffer text, keeping the cursor where it was
// (clamped). Used when the preview edits the note; it is one change that u
// undoes.
func (m *Model) ReplaceText(text string) {
	m.edit(func() {
		row, col := m.textarea.RowCol()
		m.textarea.SetValue(text)
		m.textarea.SetRowCol(row, col)
	})
}

// GotoLine puts the cursor on 1-based line n, scrolled into view.
func (m *Model) GotoLine(n int) { m.textarea.GotoLine(n) }

// ScrollBy scrolls the text by n lines, even while the pane is unfocused.
func (m *Model) ScrollBy(n int) { m.textarea.ScrollBy(n) }
