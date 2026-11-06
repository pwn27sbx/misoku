package app

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// historyRows is how many commits (and preview lines) the dialog shows at a
// time on a screen of height H.
func historyRows(H int) int { return max(min(H-10, 24), 3) }

// historyGeom is the dialog's box on screen, border included, and the width
// of its commit list.
func (m *Model) historyGeom() (x, y, w, h, listW int) {
	w = min(120, m.width-2)
	h = historyRows(m.height) + 5
	x, y = ui.Center(m.width, m.height, w, h)
	listW = min(46, (w-2)/2)
	return x, y, w, h, listW
}

// update handles the dialog's keys and mouse: j/k or the arrows move, Enter
// restores the selected version, Esc or q closes it.
func (d *historyDialog) update(m *Model, msg tea.Msg) tea.Cmd {
	rows := historyRows(m.height)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.history = nil
		case "enter":
			d.restore(m)
		case "down", "j", "ctrl+n", "ctrl+j":
			return d.move(m, 1, rows)
		case "up", "k", "ctrl+p", "ctrl+k":
			return d.move(m, -1, rows)
		case "pgdown", "ctrl+d":
			return d.move(m, rows, rows)
		case "pgup", "ctrl+u":
			return d.move(m, -rows, rows)
		case "home", "g":
			return d.selectAt(m, 0, rows)
		case "end", "G":
			return d.selectAt(m, len(d.versions)-1, rows)
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			return d.move(m, -1, rows)
		case tea.MouseButtonWheelDown:
			return d.move(m, 1, rows)
		case tea.MouseButtonLeft:
			x, y, w, h, listW := m.historyGeom()
			if msg.X < x || msg.X >= x+w || msg.Y < y || msg.Y >= y+h {
				m.history = nil
				return nil
			}
			// Border, title and rule above the first commit.
			i := msg.Y - y - 3
			if msg.X > x+listW || i < 0 || i >= rows || d.offset+i >= len(d.versions) {
				return nil
			}
			if d.offset+i == d.sel {
				d.restore(m) // a click on the selected commit restores it
				return nil
			}
			return d.selectAt(m, d.offset+i, rows)
		}
	}
	return nil
}

// view draws the commit list on the left and the selected version on the
// right.
func (d *historyDialog) view(m *Model) (string, int, int) {
	th := m.theme
	x, y, w, _, listW := m.historyGeom()
	inner := w - 2
	textW := max(inner-listW-1, 1)
	bg := lipgloss.NewStyle().Background(th.PopupBg)
	sel := lipgloss.NewStyle().Background(th.OneBg2)

	title := bg.Foreground(th.Blue).Render(" \U000f02da ") +
		bg.Foreground(th.Fg).Bold(true).Render(ui.Truncate("History of "+d.id, inner-4))
	out := []string{title, bg.Foreground(th.Line).Render(strings.Repeat("─", inner))}

	lines := d.previewLines()
	n := historyRows(m.height)
	for r := 0; r < n; r++ {
		left := bg.Render("")
		if i := d.offset + r; i < len(d.versions) {
			v := d.versions[i]
			base, marker := bg, "  "
			if i == d.sel {
				base, marker = sel, "▌ "
			}
			label := v.Date + "  " + v.Subject
			left = base.Foreground(th.Blue).Render(marker) +
				base.Foreground(th.Fg).Bold(i == d.sel).Render(ui.Truncate(label, listW-2))
			left = ui.FitLine(left, listW, base)
		} else {
			left = ui.FitLine(left, listW, bg)
		}
		right := ""
		if r < len(lines) {
			right = ui.Truncate(lines[r], textW-1)
		}
		out = append(out, left+bg.Foreground(th.Line).Render("│")+
			ui.FitLine(bg.Foreground(th.GreyFg2).Render(" "+right), textW, bg))
	}
	out = append(out, bg.Foreground(th.GreyFg).Render(" j/k move · Enter restore into the editor (u undoes it) · Esc close"))
	return panel(th, out, inner), x, y
}
