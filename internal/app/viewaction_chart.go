package app

import (
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// Charts

func (m *Model) chartHit(h preview.Hit) tea.Cmd {
	switch h.Kind {
	case "chart:add":
		return m.prompt("Add a value", "", "Label: number, e.g. May: 18", func(m *Model, v string) tea.Cmd {
			if label, num, ok := strings.Cut(v, ":"); ok {
				if _, err := strconv.ParseFloat(strings.TrimSpace(num), 64); err == nil {
					m.setDocLines(insertAt(m.docLines(), h.End, strings.TrimSpace(label)+": "+strings.TrimSpace(num)))
					return nil
				}
			}
			m.setError("Use Label: number")
			return nil
		})
	case "chart:value":
		_, current, _ := strings.Cut(m.docLines()[h.Line], ":")
		m.menu(h.Arg+" ="+current, []string{"Edit value", "Rename label", "Delete"}, func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				return m.prompt("Value for "+h.Arg, strings.TrimSpace(current), "", func(m *Model, v string) tea.Cmd {
					if _, err := strconv.ParseFloat(v, 64); err != nil {
						m.setError("Not a number: " + v)
						return nil
					}
					lines := m.docLines()
					lines[h.Line] = h.Arg + ": " + v
					m.setDocLines(lines)
					return nil
				})
			case 1:
				return m.prompt("Rename "+h.Arg, h.Arg, "", func(m *Model, v string) tea.Cmd {
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
