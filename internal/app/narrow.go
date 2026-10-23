package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/ui"
)

// DefaultExplorerHideWidth is the terminal width below which the explorer
// starts hidden and, when toggled, opens over the editor instead of beside
// it, so a narrow window keeps the editor's columns.
const DefaultExplorerHideWidth = config.DefaultExplorerHideWidth

// WithExplorerHideWidth sets the width below which the explorer is hidden
// and opens as an overlay (DefaultExplorerHideWidth when not set). n <= 0
// never hides it: the explorer always docks beside the editor. Applying it
// to a running model re-evaluates the current width at once.
func WithExplorerHideWidth(n int) Option {
	return func(m *Model) {
		m.explorerHideWidth = max(n, 0)
		m.explorerHideSet = true
		if m.width > 0 {
			m.updateLayout()
		}
	}
}

// hideWidth is the effective explorer-hide threshold.
func (m *Model) hideWidth() int {
	if !m.explorerHideSet {
		return DefaultExplorerHideWidth
	}
	return m.explorerHideWidth
}

// narrowWindow reports whether the window is under the threshold, where the
// explorer opens over the editor.
func (m *Model) narrowWindow() bool {
	return m.width > 0 && m.width < m.hideWidth()
}

// explorerOverlay reports whether the explorer is shown over the editor.
func (m *Model) explorerOverlay() bool {
	return m.sidebarOpen && m.narrowWindow()
}

// applyExplorerWidth runs on every resize. Crossing the threshold resets the
// explorer: hidden when the window gets narrow, docked when it gets wide.
// While the width stays on one side, the last manual toggle is kept.
func (m *Model) applyExplorerWidth() {
	narrow := m.narrowWindow()
	if narrow == m.explorerNarrow {
		return
	}
	m.explorerNarrow = narrow
	m.sidebarOpen = !narrow
	if narrow && m.focus == focusSidebar && m.viewMode == viewModeWorkspace {
		m.focusPane(focusEditor)
	}
}

// explorerW is the explorer's width: its docked column, or its overlay
// width (clamped like a docked one) while it opens over the editor.
func (m *Model) explorerW(l Layout) int {
	if m.explorerOverlay() {
		return CalculateLayout(m.width, m.height, true, false, false, m.split).SidebarW
	}
	return l.SidebarW
}

// hitLayout is the layout mouse events are matched against: the overlay
// takes the explorer's columns over the editor, so its cells and its edge
// behave like a docked explorer's.
func (m *Model) hitLayout() Layout {
	l := m.layout()
	l.SidebarW = m.explorerW(l)
	return l
}

// closeExplorerOverlay hides the overlay and gives the editor the focus.
func (m *Model) closeExplorerOverlay() tea.Cmd {
	m.sidebarOpen = false
	return m.focusPane(focusEditor)
}

// drawExplorerOverlay paints the explorer and its edge over the editor.
func (m *Model) drawExplorerOverlay(screen string) string {
	if !m.explorerOverlay() {
		return screen
	}
	l := m.layout()
	c := m.theme.Line
	if m.resizing == dividerSidebar {
		c = m.theme.Blue
	}
	edge := ui.Column("│", l.BodyH, lipgloss.NewStyle().Foreground(c))
	box := lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar.View(), edge)
	return ui.Overlay(screen, box, 0, l.BodyY)
}

// chooseFromOverlay turns a click on a note in the overlay into a choice: a
// docked explorer opens a clicked note and keeps the focus (like VSCode),
// but the overlay covers the editor, so it gives the focus to the note and
// closes. Clicks in a docked explorer pass unchanged.
func (m *Model) chooseFromOverlay(cmd tea.Cmd) tea.Cmd {
	if cmd == nil || !m.explorerOverlay() {
		return cmd
	}
	return func() tea.Msg {
		msg := cmd()
		if sel, ok := msg.(core.PageSelectedMsg); ok {
			sel.KeepFocus = false
			return sel
		}
		return msg
	}
}
