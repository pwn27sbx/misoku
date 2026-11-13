package app

import (
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// Timeline

func parseSpan(line string) (name string, start, end time.Time, ok bool) {
	n, span, found := strings.Cut(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- ")), ":")
	if !found {
		return "", start, end, false
	}
	a, b, _ := strings.Cut(span, "->")
	var err error
	if start, err = time.Parse("2006-01-02", strings.TrimSpace(a)); err != nil {
		return "", start, end, false
	}
	if end, err = time.Parse("2006-01-02", strings.TrimSpace(b)); err != nil || end.Before(start) {
		end = start
	}
	return strings.TrimSpace(n), start, end, true
}

func spanLine(name string, start, end time.Time) string {
	return name + ": " + start.Format("2006-01-02") + " -> " + end.Format("2006-01-02")
}

func (m *Model) timelineHit(h preview.Hit) tea.Cmd {
	switch h.Kind {
	case "tl:add":
		return m.prompt(i18n.T(i18n.AppTlNew), "", i18n.T(i18n.AppTlNewHint), func(m *Model, v string) tea.Cmd {
			if v == "" {
				return nil
			}
			m.pickDates(v, time.Time{}, time.Time{}, func(m *Model, start, end time.Time) {
				m.setDocLines(insertAt(m.docLines(), h.End, spanLine(v, start, end)))
			})
			return nil
		})
	case "tl:item":
		items := []string{i18n.T(i18n.AppTlDayLater), i18n.T(i18n.AppTlDayEarlier), i18n.T(i18n.AppTlWeekLater), i18n.T(i18n.AppTlWeekEarlier),
			i18n.T(i18n.AppTlExtend), i18n.T(i18n.AppTlShorten), i18n.T(i18n.AppTlSetDates), i18n.T(i18n.AppRename), i18n.T(i18n.AppDelete)}
		m.menu(h.Arg, items, func(m *Model, choice int) tea.Cmd {
			lines := m.docLines()
			name, start, end, ok := parseSpan(lines[h.Line])
			if !ok {
				return nil
			}
			shift := func(a, b int) {
				start, end = start.AddDate(0, 0, a), end.AddDate(0, 0, b)
				if end.Before(start) {
					end = start
				}
				lines[h.Line] = spanLine(name, start, end)
				m.setDocLines(lines)
			}
			switch choice {
			case 0:
				shift(1, 1)
			case 1:
				shift(-1, -1)
			case 2:
				shift(7, 7)
			case 3:
				shift(-7, -7)
			case 4:
				shift(0, 1)
			case 5:
				shift(0, -1)
			case 6:
				m.pickDates(name, start, end, func(m *Model, s, e time.Time) {
					lines := m.docLines()
					lines[h.Line] = spanLine(name, s, e)
					m.setDocLines(lines)
				})
			case 7:
				return m.prompt(i18n.T(i18n.AppRename), name, "", func(m *Model, v string) tea.Cmd {
					if v != "" {
						lines := m.docLines()
						lines[h.Line] = spanLine(v, start, end)
						m.setDocLines(lines)
					}
					return nil
				})
			case 8:
				m.setDocLines(removeAt(lines, h.Line, 1))
			}
			return nil
		})
	}
	return nil
}

// pickDates asks for a start date, then an end date on or after it.
func (m *Model) pickDates(name string, start, end time.Time, done func(*Model, time.Time, time.Time)) {
	m.pickDate(i18n.T(i18n.AppTlStart, name), start, time.Time{}, func(m *Model, s time.Time) tea.Cmd {
		initial := end
		if initial.IsZero() || initial.Before(s) {
			initial = s
		}
		m.pickDate(i18n.T(i18n.AppTlEnd, name), initial, s, func(m *Model, e time.Time) tea.Cmd {
			done(m, s, e)
			return nil
		})
		return nil
	})
}
