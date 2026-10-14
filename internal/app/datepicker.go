package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// datePicker is a small month calendar used wherever a date is needed.
type datePicker struct {
	title  string
	date   time.Time // selected day (UTC midnight)
	min    time.Time // earliest allowed day (zero = no limit)
	onPick func(m *Model, d time.Time) tea.Cmd
}

func day(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }

// pickDate opens the date picker on initial (today when zero).
func (m *Model) pickDate(title string, initial, min time.Time, onPick func(*Model, time.Time) tea.Cmd) {
	if initial.IsZero() {
		initial = time.Now()
	}
	d := day(initial)
	if !min.IsZero() && d.Before(day(min)) {
		d = day(min)
	}
	m.datePick = &datePicker{title: title, date: d, min: min, onPick: onPick}
}

func (p *datePicker) move(days, months int) {
	d := p.date.AddDate(0, months, days)
	if !p.min.IsZero() && d.Before(day(p.min)) {
		d = day(p.min)
	}
	p.date = d
}

func (p *datePicker) firstCell() time.Time {
	first := time.Date(p.date.Year(), p.date.Month(), 1, 0, 0, 0, 0, time.UTC)
	return first.AddDate(0, 0, -int(first.Weekday()))
}

const pickerCell = 4 // columns per day

func (p *datePicker) geom(m *Model) (x, y, w, h int) {
	w = 7*pickerCell + 6
	h = 13 + 2 // rows + border
	x, y = ui.Center(m.width, m.height, w, h)
	return
}

func (p *datePicker) update(m *Model, msg tea.Msg) tea.Cmd {
	pick := func() tea.Cmd {
		m.datePick = nil
		return p.onPick(m, p.date)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.datePick = nil
		case "enter", " ":
			return pick()
		case "left", "h":
			p.move(-1, 0)
		case "right", "l":
			p.move(1, 0)
		case "up", "k":
			p.move(-7, 0)
		case "down", "j":
			p.move(7, 0)
		case "[", "pgup", "<", "H":
			p.move(0, -1)
		case "]", "pgdown", ">", "L":
			p.move(0, 1)
		case "t":
			p.date = day(time.Now())
			p.move(0, 0)
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		x, y, w, h := p.geom(m)
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			p.move(0, -1)
			return nil
		case tea.MouseButtonWheelDown:
			p.move(0, 1)
			return nil
		case tea.MouseButtonLeft:
		default:
			return nil
		}
		if msg.X < x || msg.X >= x+w || msg.Y < y || msg.Y >= y+h {
			m.datePick = nil
			return nil
		}
		row, col := msg.Y-y-1, msg.X-x-1
		if row == 2 { // month header: ‹ on the left, › on the right
			if col < 4 {
				p.move(0, -1)
			} else if col >= w-6 {
				p.move(0, 1)
			}
			return nil
		}
		if r := row - 5; r >= 0 && r < 6 && col >= 2 && col < 2+7*pickerCell {
			d := p.firstCell().AddDate(0, 0, r*7+(col-2)/pickerCell)
			if p.min.IsZero() || !d.Before(day(p.min)) {
				p.date = d
				return pick()
			}
		}
	}
	return nil
}

func (p *datePicker) view(m *Model) (string, int, int) {
	th := m.theme
	x, y, w, _ := p.geom(m)
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	rows := []string{
		bg.Foreground(th.Blue).Render(" \U000f00ed ") + bg.Foreground(th.Fg).Bold(true).Render(ui.Truncate(p.title, inner-4)),
		bg.Foreground(th.GreyFg2).Render(" " + p.date.Format("Mon, 2 Jan 2006")),
	}
	month := p.date.Format("January 2006")
	pad := max(inner-4-lipgloss.Width(month)-4, 0)
	rows = append(rows, bg.Foreground(th.Blue).Render("  ‹ ")+bg.Render(strings.Repeat(" ", pad/2))+
		bg.Foreground(th.Fg).Bold(true).Render(month)+bg.Render(strings.Repeat(" ", pad-pad/2))+bg.Foreground(th.Blue).Render(" ›  "))
	rows = append(rows, "")
	var head strings.Builder
	head.WriteString(bg.Render("  "))
	for _, d := range []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"} {
		head.WriteString(bg.Foreground(th.GreyFg).Render(fmt.Sprintf("%*s  ", pickerCell-2, d)))
	}
	rows = append(rows, head.String())

	today := day(time.Now())
	cell := p.firstCell()
	for r := 0; r < 6; r++ {
		var line strings.Builder
		line.WriteString(bg.Render("  "))
		for c := 0; c < 7; c++ {
			d := cell.AddDate(0, 0, r*7+c)
			st := bg.Foreground(th.Fg)
			switch {
			case d.Equal(p.date):
				st = lipgloss.NewStyle().Background(th.Blue).Foreground(th.Bg).Bold(true)
			case !p.min.IsZero() && d.Before(day(p.min)):
				st = bg.Foreground(th.Grey)
			case d.Month() != p.date.Month():
				st = bg.Foreground(th.GreyFg)
			case d.Equal(today):
				st = bg.Foreground(th.Red).Bold(true)
			}
			line.WriteString(st.Render(fmt.Sprintf("%*s ", pickerCell-1, strconv.Itoa(d.Day()))))
		}
		rows = append(rows, line.String())
	}
	rows = append(rows, "", bg.Foreground(th.GreyFg).Render(" ←→↑↓ day · [ ] month · t today · Enter"))
	return panel(th, rows, inner), x, y
}
