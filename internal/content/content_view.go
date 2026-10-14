package content

import (
	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	plain := lipgloss.NewStyle()

	if m.page.ID == "" {
		th := m.theme
		lines := []string{
			lipgloss.NewStyle().Foreground(th.Grey).Render("󰠮"),
			"",
			lipgloss.NewStyle().Foreground(th.GreyFg2).Bold(true).Render("No note open"),
			"",
			lipgloss.NewStyle().Foreground(th.GreyFg).Render("Ctrl+N  new note    /  search    Tab  explorer"),
		}
		block := lipgloss.JoinVertical(lipgloss.Center, lines...)
		return ui.Fit(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block), m.width, m.height, plain)
	}

	m.textarea.LineColors = m.hl.colors(m.textarea.Value(), m.theme)
	return ui.Fit(m.textarea.View(), m.width, m.height, plain)
}
