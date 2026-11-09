package app

import (
	"fmt"
	"strings"

	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// View

// View composes child component views into the complete terminal layout.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		m.syncCursor(Layout{})
		return "loading…"
	}
	if m.width < MinTermWidth || m.height < MinTermHeight {
		m.syncCursor(Layout{})
		return lipgloss.NewStyle().
			Width(m.width).Height(m.height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(m.theme.Yellow).
			Render(fmt.Sprintf("Terminal too small (%dx%d)\nResize to at least %dx%d", m.width, m.height, MinTermWidth, MinTermHeight))
	}

	var screen string
	var l Layout
	if m.viewMode == viewModeDashboard {
		screen = lipgloss.JoinVertical(lipgloss.Left, m.dashboard.View(), m.bottomBar())
	} else {
		l = m.layout()
		m.dimPanes(l)
		div := func(d divider) string {
			return ui.Column("│", l.BodyH, lipgloss.NewStyle().Foreground(m.dividerColour(d, l)))
		}
		var cols []string
		if l.SidebarW > 0 {
			cols = append(cols, m.sidebar.View(), div(dividerSidebar))
		}
		if l.EditorW > 0 {
			switch {
			case m.globalGraphActive():
				cols = append(cols, m.globalGraph.View())
			case m.graphActive():
				cols = append(cols, m.graph.View())
			default:
				cols = append(cols, padNote(m.content.View(), l.EditorW, l.PadY))
			}
			if l.PreviewW > 0 {
				cols = append(cols, div(dividerPreview))
			}
		}
		if l.PreviewW > 0 {
			cols = append(cols, padNote(m.preview.View(), l.PreviewW, l.PadY))
		}
		tabs, _ := m.tabline()
		screen = lipgloss.JoinVertical(lipgloss.Left,
			tabs,
			lipgloss.JoinHorizontal(lipgloss.Top, cols...),
			m.bottomBar(),
		)
		screen = m.drawExplorerOverlay(screen)
	}

	if m.dimsBackdrop() {
		screen = ui.DimBackdrop(screen, m.theme.Grey)
	}
	switch {
	case m.saveAs != nil:
		box, x, y := m.saveAs.view(m.theme, m.width, m.height)
		screen = ui.Overlay(screen, box, x, y)
	case m.confirm != nil:
		box, x, y := m.confirm.view(m.theme, m.width, m.height)
		screen = ui.Overlay(screen, box, x, y)
	case m.finder != nil:
		box, x, y := m.finder.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.themes != nil:
		box, x, y := m.themes.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.browser != nil:
		box, x, y := m.browser.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.history != nil:
		box, x, y := m.history.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.datePick != nil:
		box, x, y := m.datePick.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.promptBox != nil:
		box, x, y := m.promptBox.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.menuBox != nil:
		box, x, y := m.menuBox.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.viewMode == viewModeWorkspace && m.focus == focusEditor && m.content.LinkListOpen():
		if box, x, y, ok := m.content.LinkListView(); ok {
			l := m.layout()
			screen = ui.Overlay(screen, box, l.EditorX+x, l.NoteY()+y)
		}
	case m.viewMode == viewModeWorkspace && m.focus == focusEditor && m.content.SlashOpen():
		if box, x, y, ok := m.content.SlashView(); ok {
			l := m.layout()
			screen = ui.Overlay(screen, box, l.EditorX+x, l.NoteY()+y)
		}
	case m.palette != nil:
		box, x, y := m.palette.view(&m)
		screen = ui.Overlay(screen, box, x, y)
	case m.leaderGuideOpen():
		screen = m.overlayLeaderGuide(screen)
	}
	screen = m.renderToast(screen)
	// After the overlays: whether one of them owns the cursor is model state.
	m.syncCursor(l)
	return ui.Paint(screen, lipgloss.NewStyle().Foreground(m.theme.Fg).Background(m.theme.Bg))
}

// dimsBackdrop reports whether a modal box is open, so the screen behind it
// is greyed out and the box is the only coloured thing on it. The leader
// guide is a passing hint while a key chord is typed and the slash menu a
// completion attached to the cursor: neither dims what the user is working on.
func (m Model) dimsBackdrop() bool { return m.modalOpen() && !m.leaderGuideOpen() }

// padNote adds pad blank rows above and below a note pane's view of width w.
// They are plain spaces, so the frame's paint gives them the pane's
// background, or none with ui.transparent.
func padNote(view string, w, pad int) string {
	if pad == 0 {
		return view
	}
	blank := strings.Repeat(strings.Repeat(" ", w)+"\n", pad)
	return blank + view + "\n" + strings.TrimSuffix(blank, "\n")
}
