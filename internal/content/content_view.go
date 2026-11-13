package content

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	if m.dimmed {
		// A copy: the grey styles and the blur last for this frame only.
		m.theme = greyText(m.theme)
		configureTextareaStyles(&m.textarea, m.theme)
		m.textarea.Blur()
	}
	plain := lipgloss.NewStyle()

	if m.page.ID == "" {
		th := m.theme
		lines := []string{
			lipgloss.NewStyle().Foreground(th.Grey).Render("󰠮"),
			"",
			lipgloss.NewStyle().Foreground(th.GreyFg2).Bold(true).Render(i18n.T(i18n.EditorNoNote)),
			"",
			lipgloss.NewStyle().Foreground(th.GreyFg).Render(m.emptyHint()),
		}
		block := lipgloss.JoinVertical(lipgloss.Center, lines...)
		return ui.Fit(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, block), m.width, m.height, plain)
	}

	// Drawn in the language in use now, which may have changed since New.
	m.textarea.Placeholder = i18n.T(i18n.EditorPlaceholder)
	m.textarea.LineColors = nil
	if !m.dimmed {
		m.textarea.LineColors = m.hl.colors(m)
	}
	return ui.Fit(m.textarea.View(), m.width, m.height, plain)
}

// emptyHint names the shortcuts worth knowing with no note open, as the user
// configured them; an unbound action is left out.
func (m Model) emptyHint() string {
	hints := []struct {
		action string
		what   i18n.Key
	}{
		{keymap.NewNote, i18n.EditorHintNewNote},
		{keymap.FindNote, i18n.EditorHintSearch},
		{keymap.NextPane, i18n.EditorHintExplorer},
	}
	var parts []string
	for _, h := range hints {
		if k := m.keys.Hint(h.action); k != "" {
			parts = append(parts, k+"  "+i18n.T(h.what))
		}
	}
	return strings.Join(parts, "    ")
}

// greyText is th with every text colour set to Grey and the backgrounds
// kept: how a pane without the focus is drawn.
func greyText(th theme.Theme) theme.Theme {
	for _, c := range []*lipgloss.Color{
		&th.Fg, &th.GreyFg, &th.GreyFg2, &th.LightGrey, &th.Red, &th.Pink,
		&th.Green, &th.Blue, &th.NordBlue, &th.Yellow, &th.Purple, &th.DarkPurple,
		&th.Teal, &th.Orange, &th.Cyan, &th.Folder, &th.TitleFg, &th.MutedFg, &th.SelectedFg,
	} {
		*c = th.Grey
	}
	return th
}
