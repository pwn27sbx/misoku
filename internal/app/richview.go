package app

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/rich"

	tea "github.com/charmbracelet/bubbletea"
)

// The rich reading view: the note drawn as a page of pixels, sent to the
// terminal as Kitty tiles behind placeholder cells. The text reading view
// stays the fallback everywhere: a pane that cannot draw the page shows the
// compile instead, never a blank.

// minRichWidth is the narrowest editor column, in cells, that draws the page
// on its own (ui.rich_view=auto). ":rich" asks for it from 40 cells.
const (
	minRichWidth   = 80
	minRichForce   = 40
	richScrollRows = 3
)

// richAvailable reports whether this terminal and pane can draw the page now.
func (m *Model) richAvailable(force bool) bool {
	if m.richSender == nil || !preview.GraphicsOK() {
		return false
	}
	w := m.layout().EditorW
	if force {
		return w >= minRichForce
	}
	return w >= minRichWidth
}

// toggleRichView is Space V and :rich: the page when the terminal can draw it,
// the editable reading view otherwise.
func (m *Model) toggleRichView() tea.Cmd {
	if m.noteView == viewRich {
		m.setNoteView(viewReading)
		return m.focusPane(focusEditor)
	}
	if !m.richAvailable(true) {
		m.setStatus(i18n.T(i18n.AppRichUnavailable))
		return nil
	}
	m.setNoteView(viewRich)
	return m.focusPane(focusEditor)
}

// refreshRich lays the active note out for the current width and theme. It is
// called when the view opens, the note changes, the pane resizes or the theme
// does; typing re-renders once the input task lands.
func (m *Model) refreshRich() {
	if m.noteView != viewRich {
		return
	}
	b := m.activeBuffer()
	if b == nil || b.draft() {
		m.setNoteView(viewReading)
		return
	}
	l := m.layout()
	if l.EditorW < 20 || l.BodyH < 4 {
		return
	}
	if m.richFonts == nil {
		fonts, err := rich.LoadFonts(rich.FontConfig{
			BodyPath: m.richFontBody, MonoPath: m.richFontMono,
			CellW: m.richCellW, CellH: m.richCellH,
		})
		if err != nil {
			m.setError(i18n.T(i18n.AppRichFailed, err.Error()))
			m.setNoteView(viewReading)
			return
		}
		m.richFonts = fonts
	}
	pal := rich.PaletteFromTheme(m.theme)
	doc := rich.Parse([]byte(b.text))
	noteDir := filepath.Join(m.store.Root(), filepath.FromSlash(path.Dir(b.id)))
	images := rich.ImageOptions{
		BaseDir: noteDir,
		Root:    m.store.Root(),
		Remote:  os.Getenv("MISOKU_RICH_REMOTE_IMAGES") == "1",
	}
	page := rich.Layout(doc, rich.LayoutOptions{Width: l.EditorW, Fonts: m.richFonts, Palette: pal, Images: images})
	if m.rich == nil {
		m.rich = rich.NewView(l.EditorW, l.BodyH, m.richFonts, pal, m.richSender)
	}
	m.rich.Palette = pal
	m.rich.SetSize(l.EditorW, l.BodyH)
	m.rich.SetPage(page)
}

// prepareRich sends the tiles the next frame needs. A refusal falls back to
// the text reading view with the reason in the bar, so no frame is blank.
func (m *Model) prepareRich() {
	if m.noteView != viewRich {
		return
	}
	if m.rich == nil {
		m.refreshRich()
	}
	if m.rich == nil || !m.rich.Ready() {
		m.setStatus(i18n.T(i18n.AppRichUnavailable))
		m.setNoteView(viewReading)
		return
	}
	if err := m.rich.Prepare(); err != nil {
		m.setError(i18n.T(i18n.AppRichFailed, err.Error()))
		m.setNoteView(viewReading)
	}
}

// richKey scrolls the page while the rich view has the keys. It reports
// handled=false for keys the rest of the app should see (leader, tabs, quit).
func (m *Model) richKey(msg tea.Msg) (tea.Cmd, bool) {
	switch t := msg.(type) {
	case tea.MouseMsg:
		if t.Button == tea.MouseButtonWheelUp {
			m.rich.Scroll(-richScrollRows)
			return nil, true
		}
		if t.Button == tea.MouseButtonWheelDown {
			m.rich.Scroll(richScrollRows)
			return nil, true
		}
		if t.Button == tea.MouseButtonLeft {
			if row, ok := m.richRowAt(t.Y); ok {
				switch t.Action {
				case tea.MouseActionPress:
					m.richDrag = row
					m.rich.Select(row, row)
					return nil, true
				case tea.MouseActionMotion:
					if m.richDrag >= 0 {
						m.rich.Select(m.richDrag, row)
						return nil, true
					}
				case tea.MouseActionRelease:
					if m.richDrag == row {
						// A click without a drag opens the editor here.
						m.rich.ClearSelection()
						return m.openRichLine(row), true
					}
					m.richDrag = -1
					return nil, true
				}
			}
		}
	case tea.KeyMsg:
		switch t.String() {
		case "j", "down":
			m.rich.Scroll(1)
			return nil, true
		case "k", "up":
			m.rich.Scroll(-1)
			return nil, true
		case "d", "ctrl+d", "pagedown":
			m.rich.Scroll(m.rich.H / 2)
			return nil, true
		case "u", "ctrl+u", "pageup":
			m.rich.Scroll(-m.rich.H / 2)
			return nil, true
		case "g", "home":
			m.rich.Scroll(-m.rich.MaxScroll())
			return nil, true
		case "G", "end":
			m.rich.Scroll(m.rich.MaxScroll())
			return nil, true
		case "/":
			return m.prompt("Search", "", "in this page", func(mm *Model, value string) tea.Cmd {
				if n := mm.rich.Search(value); n == 0 {
					mm.setStatus("no matches")
				} else {
					mm.setStatus(fmt.Sprintf("%d matches", n))
				}
				return nil
			}), true
		case "n":
			if m.rich.SearchNext(1) {
				return nil, true
			}
		case "N":
			if m.rich.SearchNext(-1) {
				return nil, true
			}
		case "y":
			text := m.richSelectedText()
			if strings.TrimSpace(text) == "" {
				m.setStatus("nothing to copy")
				return nil, true
			}
			return m.copyText(text), true
		case "+", "=":
			m.richCellW += 2
			m.richCellH += 4
			m.refreshRich()
			return nil, true
		case "-":
			m.richCellW = max(m.richCellW-2, 6)
			m.richCellH = max(m.richCellH-4, 12)
			m.refreshRich()
			return nil, true
		case "esc":
			if a, b := m.rich.SelectionRows(); a >= 0 || b >= 0 || len(m.richOutline()) > 0 && m.richHasOverlay() {
				m.rich.ClearSelection()
				return nil, true
			}
			fallthrough
		case "i":
			// Back to the editable reading view at the line on screen.
			line := m.rich.SourceLine()
			m.setNoteView(viewReading)
			m.content.GotoLine(line + 1)
			return m.focusPane(focusEditor), true
		}
	}
	return nil, false
}

// richRowAt maps a screen row to the page's cell row, or false when the
// pointer is outside the page.
func (m *Model) richRowAt(screenY int) (int, bool) {
	l := m.layout()
	top := l.NoteY()
	if screenY < top || screenY >= top+m.rich.H {
		return 0, false
	}
	return m.rich.RowAt(screenY - top), true
}

// openRichLine opens the editable reading view at the note line the page row
// belongs to.
func (m *Model) openRichLine(row int) tea.Cmd {
	line := m.rich.PageLine(row)
	m.setNoteView(viewReading)
	m.content.GotoLine(line + 1)
	return m.focusPane(focusEditor)
}

// richSelectedText is the selected rows' text, or the visible page when there
// is no selection.
func (m *Model) richSelectedText() string {
	if a, b := m.rich.SelectionRows(); a >= 0 || b >= 0 {
		return m.rich.Select(a, b)
	}
	return m.rich.VisibleText()
}

// richOutline is the page's table of contents.
func (m *Model) richOutline() []rich.Heading { return m.rich.Outline() }

// richHasOverlay reports whether a search or selection is showing.
func (m *Model) richHasOverlay() bool {
	a, b := m.rich.SelectionRows()
	return a >= 0 || b >= 0
}

var errRichUnavailable = errors.New("rich view unavailable")
