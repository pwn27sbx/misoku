package app

import (
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// Charts

func (m *Model) chartHit(h preview.Hit) tea.Cmd {
	switch h.Kind {
	case "chart:add":
		return m.prompt(i18n.T(i18n.AppChartAdd), "", i18n.T(i18n.AppChartAddHint), func(m *Model, v string) tea.Cmd {
			if label, num, ok := strings.Cut(v, ":"); ok {
				if _, err := strconv.ParseFloat(strings.TrimSpace(num), 64); err == nil {
					m.setDocLines(insertAt(m.docLines(), h.End, strings.TrimSpace(label)+": "+strings.TrimSpace(num)))
					return nil
				}
			}
			m.setError(i18n.T(i18n.AppChartUse))
			return nil
		})
	case "chart:value":
		_, current, _ := strings.Cut(m.docLines()[h.Line], ":")
		m.menu(h.Arg+" ="+current, []string{i18n.T(i18n.AppChartEdit), i18n.T(i18n.AppChartRenameLabel), i18n.T(i18n.AppDelete)}, func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				return m.prompt(i18n.T(i18n.AppChartValueFor, h.Arg), strings.TrimSpace(current), "", func(m *Model, v string) tea.Cmd {
					if _, err := strconv.ParseFloat(v, 64); err != nil {
						m.setError(i18n.T(i18n.AppChartNotNumber, v))
						return nil
					}
					lines := m.docLines()
					lines[h.Line] = h.Arg + ": " + v
					m.setDocLines(lines)
					return nil
				})
			case 1:
				return m.prompt(i18n.T(i18n.AppChartRename, h.Arg), h.Arg, "", func(m *Model, v string) tea.Cmd {
					if v != "" {
						lines := m.docLines()
						lines[h.Line] = v + ":" + current
						m.setDocLines(lines)
					}
					return nil
				})
			default:
				m.setDocLines(removeAt(m.docLines(), h.Line, 1))
			}
			return nil
		})
	}
	return nil
}
