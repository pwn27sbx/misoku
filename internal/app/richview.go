package app

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/rich"
	"github.com/pwn27sbx/misoku/internal/rich/engines"

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
	// A refusal is never silent: say which piece is missing.
	if reason := m.richRefusal(); reason != "" {
		m.richNotice = reason
		m.setStatus(reason)
		return nil
	}
	m.setNoteView(viewRich)
	return m.focusPane(focusEditor)
}

// richRefusal is why the page cannot open here, or "" when it can. The reason
// names the first missing piece: the caller's hint (tmux), the image output,
// the terminal's graphics, the pane width, or an open saved note.
func (m *Model) richRefusal() string {
	if m.richHint != "" {
		return m.richHint
	}
	if m.richSender == nil {
		return i18n.T(i18n.AppRichNoOutput)
	}
	if !preview.GraphicsOK() {
		return i18n.T(i18n.AppRichUnavailable)
	}
	if w := m.layout().EditorW; w < minRichForce {
		return i18n.T(i18n.AppRichNarrow, minRichForce, w)
	}
	b := m.activeBuffer()
	if b == nil {
		return i18n.T(i18n.AppOpenNoteFirst)
	}
	if b.draft() {
		return i18n.T(i18n.AppRichSaveFirst)
	}
	return ""
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
	// A rebuild (theme, resize, note change) drops the diagram zoom: the
	// page under it no longer exists.
	if m.richZoomBack != nil && m.rich != nil {
		m.leaveZoom()
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
	// A new theme means new colours inside the engines too: their caches are
	// keyed by colour, so they are rebuilt rather than reused.
	if m.richEngines == nil || m.richEngines.Palette() != pal {
		m.richEngines = engines.New(m.richFonts, pal)
	}
	page := m.richEngines.Page(doc, l.EditorW, images)
	// A rebuild must not lose the reader's place: keep the note line at the top
	// of the page across a resize or a reload.
	line := 0
	if m.rich != nil && m.rich.Page != nil {
		line = m.rich.SourceLine()
	}
	if m.rich == nil {
		m.rich = rich.NewView(l.EditorW, l.BodyH, m.richFonts, pal, m.richSender)
	}
	m.rich.Palette = pal
	m.rich.SetSize(l.EditorW, l.BodyH)
	m.rich.SetPage(page)
	if line > 0 {
		m.rich.GoToLine(line)
	}
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
		reason := i18n.T(i18n.AppRichUnavailable)
		m.richNotice = reason
		m.setStatus(reason)
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
		if m.richZoomBack != nil {
			// Any click leaves the zoom; its rows are the picture's, not the
			// note's, so nothing else may read them.
			if t.Button == tea.MouseButtonLeft && t.Action == tea.MouseActionRelease {
				m.leaveZoom()
			}
			return nil, true
		}
		if t.Button == tea.MouseButtonWheelUp {
			if t.Shift {
				m.rich.WideScroll(-2)
			} else {
				m.rich.Scroll(-richScrollRows)
			}
			return nil, true
		}
		if t.Button == tea.MouseButtonWheelDown {
			if t.Shift {
				m.rich.WideScroll(2)
			} else {
				m.rich.Scroll(richScrollRows)
			}
			return nil, true
		}
		if t.Button == tea.MouseButtonLeft {
			if t.Action == tea.MouseActionRelease && m.richDrag >= 0 {
				defer func() { m.richDrag = -1 }()
			}
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
						// A click on a diagram zooms it; anywhere else it opens
						// the editor at that line.
						m.rich.ClearSelection()
						if m.zoomDiagram(row) {
							return nil, true
						}
						return m.openRichLine(row), true
					}
					m.richDrag = -1
					return nil, true
				}
			}
		}
	case tea.KeyMsg:
		if m.richZoomBack != nil {
			// While a diagram fills the pane only leaving it is the rich
			// view's business; every other key stays with the app, so quit,
			// the leader and help keep working.
			switch t.String() {
			case "esc", "enter", "i", "q":
				m.leaveZoom()
				return nil, true
			}
			return nil, false
		}
		if m.rich.HintsOn() {
			key := t.String()
			if h, ok := m.rich.HintFor(key); ok {
				return m.richFollow(h.Target), true
			}
			m.rich.ClearHints()
			if key == "esc" {
				return nil, true
			}
		}
		switch t.String() {
		case "enter":
			// Zoom the selected diagram, or the first one on screen.
			row := -1
			if a, _ := m.rich.SelectionRows(); a >= 0 {
				row = a
			} else {
				for r := m.rich.ScrollTop(); r < m.rich.ScrollTop()+m.rich.H; r++ {
					if _, ok := m.diagramAt(r); ok {
						row = r
						break
					}
				}
			}
			if row >= 0 && m.zoomDiagram(row) {
				return nil, true
			}
			m.setStatus("no diagram on screen")
			return nil, true
		case "f":
			if hints := m.rich.ShowHints(); len(hints) > 0 {
				m.setStatus("type a letter to follow a link")
			} else {
				m.setStatus("no links on screen")
			}
			return nil, true
		case "H":
			m.rich.WideScroll(-4)
			return nil, true
		case "L":
			m.rich.WideScroll(4)
			return nil, true
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
			m.scaleRich(1.15)
			return nil, true
		case "-":
			m.scaleRich(1 / 1.15)
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

// richFollow opens a link from the page: a URL goes to the system opener, a
// note target opens as a [[link]] would.
func (m *Model) richFollow(target string) tea.Cmd {
	m.rich.ClearHints()
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		if m.openURL != nil {
			_ = m.openURL(target)
		}
		return nil
	}
	return m.followWikiLink(core.ParseWikiLink(target), m.openInCurrentBuffer)
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

// scaleRich zooms the page: the faces, the cells they sit in and therefore
// every line of the page grow together, and the engines rebuild at the new
// size.
func (m *Model) scaleRich(k float64) {
	if m.richFonts == nil {
		return
	}
	fonts, err := m.richFonts.Scaled(k)
	if err != nil {
		m.setError(i18n.T(i18n.AppRichFailed, err.Error()))
		return
	}
	if fonts.CellW < 6 || fonts.CellW > 40 || fonts.CellH < 12 || fonts.CellH > 80 {
		return
	}
	m.richFonts = fonts
	m.richCellW, m.richCellH = fonts.CellW, fonts.CellH
	m.richEngines = nil
	m.refreshRich()
	m.setStatus(fmt.Sprintf("page zoom %d×%d px cells", fonts.CellW, fonts.CellH))
}

// diagramAt finds the diagram whose rows cover a page row.
func (m *Model) diagramAt(row int) (rich.DiagramOnPage, bool) {
	if m.rich == nil || m.rich.Page == nil {
		return rich.DiagramOnPage{}, false
	}
	for _, d := range m.rich.Page.Diagrams {
		if row >= d.Row && row < d.Row+d.Rows {
			return d, true
		}
	}
	return rich.DiagramOnPage{}, false
}

// zoomDiagram draws the diagram under row across the pane. It reports whether
// the row belongs to a diagram, so a click can fall through to the editor.
func (m *Model) zoomDiagram(row int) bool {
	d, ok := m.diagramAt(row)
	if !ok || m.richEngines == nil {
		return false
	}
	l := m.layout()
	zoom, ok := m.richEngines.Zoom(d.Block, l.EditorW, l.BodyH)
	if !ok {
		return false
	}
	m.richZoomBack = m.rich.Page
	m.richZoomTop = m.rich.ScrollTop()
	m.rich.SetPage(zoom)
	m.setStatus("diagram zoom — esc to go back")
	return true
}

// leaveZoom returns to the page with the reader where they were.
func (m *Model) leaveZoom() {
	if m.richZoomBack == nil || m.rich == nil {
		return
	}
	back := m.richZoomBack
	m.rich.SetPage(back)
	m.rich.Scroll(m.richZoomTop - m.rich.ScrollTop())
	m.richZoomBack = nil
	m.setStatus("")
}

var errRichUnavailable = errors.New("rich view unavailable")
