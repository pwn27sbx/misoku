package app

import (
	"errors"

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
	page := rich.Layout(doc, rich.LayoutOptions{Width: l.EditorW, Fonts: m.richFonts, Palette: pal})
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
		case "esc", "i":
			// Back to the editable reading view at the line on screen.
			line := m.rich.SourceLine()
			m.setNoteView(viewReading)
			m.content.GotoLine(line + 1)
			return m.focusPane(focusEditor), true
		}
	}
	return nil, false
}

var errRichUnavailable = errors.New("rich view unavailable")
