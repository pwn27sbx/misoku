package app

import (
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// Calendar

func (m *Model) calendarHit(h preview.Hit) tea.Cmd {
	switch h.Kind {
	case "cal:day":
		return m.prompt("New event on "+h.Arg, "", "", func(m *Model, v string) tea.Cmd {
			if v != "" {
				m.setDocLines(insertAt(m.docLines(), h.End, h.Arg+": "+v))
			}
			return nil
		})
	case "cal:event":
		date, _, _ := strings.Cut(strings.TrimSpace(m.docLines()[h.Line]), ":")
		m.menu(h.Arg, []string{"Rename", "Move to another date", "Delete"}, func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				return m.prompt("Rename event", h.Arg, "", func(m *Model, v string) tea.Cmd {
					if v != "" {
						lines := m.docLines()
						lines[h.Line] = strings.TrimSpace(date) + ": " + v
						m.setDocLines(lines)
					}
					return nil
				})
			case 1:
				from, _ := time.Parse("2006-01-02", strings.TrimSpace(date))
				m.pickDate("Move \""+h.Arg+"\" to", from, time.Time{}, func(m *Model, d time.Time) tea.Cmd {
					lines := m.docLines()
					lines[h.Line] = d.Format("2006-01-02") + ": " + h.Arg
					m.setDocLines(lines)
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
