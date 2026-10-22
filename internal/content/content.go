// Package content renders the Markdown editor pane (<Body/>), a Vim-style
// editor. The file name lives in the tabline only.
package content

import (
	"regexp"
	"sync/atomic"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/highlight"
	"github.com/pwn27sbx/misoku/internal/textarea"
	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the Content component state.
type Model struct {
	theme    theme.Theme
	page     core.Page
	draft    bool
	textarea textarea.Model
	cmdInput textinput.Model
	mode     VimMode
	pending  string // first key of a two-key Vim command ("g", "d")
	slash    *slashMenu
	hl       *hlCache // shared across copies so View can memoise
	copies   *atomic.Int64
	dragging bool
	dragRow  int
	dragCol  int
	width    int
	height   int
	focused  bool
}

// New constructs a Content Model.
func New(th theme.Theme) Model {
	ta := textarea.New()
	ta.Placeholder = "Start writing… (i to insert)"
	ta.Prompt = ""
	ta.ShowLineNumbers = true
	ta.EndOfBufferCharacter = '~'
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.Cursor.Style = lipgloss.NewStyle().Foreground(th.Fg)
	configureTextareaStyles(&ta, th)

	return Model{
		theme:    th,
		textarea: ta,
		cmdInput: createCommandInput(th),
		mode:     ModeNormal,
		hl:       &hlCache{},
		copies:   &atomic.Int64{},
	}
}

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
	m.textarea.SetWidth(max(w-1, 10))
	m.textarea.SetHeight(max(h, 1))
}

// SetPage loads a document into the editor. draft marks an unsaved buffer
// that has no file yet.
func (m *Model) SetPage(p core.Page) {
	m.SetBuffer(p, false)
}

// SetBuffer loads a buffer's text, resetting the cursor to the top.
func (m *Model) SetBuffer(p core.Page, draft bool) {
	m.page = p
	m.draft = draft
	m.pending = ""
	m.slash = nil
	m.textarea.SetValue(p.Content)
	m.textarea.GotoTop()
}

// SetFocused sets focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused {
		m.slash = nil
		m.Blur()
		if m.mode == ModeCommand {
			m.mode = ModeNormal
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
	if m.mode == ModeInsert {
		cursorMode = cursor.CursorBlink
	}
	return tea.Batch(cmd, m.textarea.Cursor.SetMode(cursorMode))
}

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

// EnterInsert switches to INSERT mode (used by the app, e.g. for a new buffer).
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
// (clamped). Used when the preview edits the note.
func (m *Model) ReplaceText(text string) {
	row, col := m.textarea.RowCol()
	m.textarea.SetValue(text)
	m.textarea.SetRowCol(row, col)
}

// GotoLine puts the cursor on 1-based line n, scrolled into view.
func (m *Model) GotoLine(n int) { m.textarea.GotoLine(n) }

// ScrollBy scrolls the text by n lines, even while the pane is unfocused.
func (m *Model) ScrollBy(n int) { m.textarea.ScrollBy(n) }
