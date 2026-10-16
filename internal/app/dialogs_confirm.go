package app

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// panel wraps pre-painted rows (each exactly inner wide) in a rounded border.
func panel(th theme.Theme, rows []string, inner int) string {
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	for i, r := range rows {
		rows[i] = ui.FitLine(r, inner, bg)
	}
	return lipgloss.NewStyle().
		BorderStyle(ui.Border()).
		BorderForeground(th.Blue).
		BorderBackground(th.DarkerBg).
		Render(strings.Join(rows, "\n"))
}

// ---------------------------------------------------------------------------
// Confirm dialog

type confirmDialog struct {
	title   string
	message []string
	buttons []string // the last button is always "cancel"
	danger  bool     // paint the first button red (e.g. Delete)
	// deliberate takes a choice only from the arrow keys, Tab and Enter
	// (or a click): no letter picks or moves to a button. For a dialog that
	// can open on its own while the user types, where a stray letter must
	// not choose for them.
	deliberate bool
	selected   int
	onChoose   func(m *Model, choice int) tea.Cmd
}

type confirmGeom struct {
	x, y, w, h int
	buttonRow  int
	spans      [][2]int // inner-x start/end per button
}

func (d *confirmDialog) innerWidth() int {
	w := 46
	for _, l := range d.message {
		w = max(w, lipgloss.Width(l)+6)
	}
	return w
}

func (d *confirmDialog) geom(W, H int) confirmGeom {
	iw := d.innerWidth()
	h := 2 + len(d.message) + 2 + 1
	x, y := ui.Center(W, H, iw+2, h+2)
	g := confirmGeom{x: x, y: y, w: iw + 2, h: h + 2, buttonRow: h - 1}
	cx := 2
	for _, b := range d.buttons {
		bw := lipgloss.Width(b) + 4
		g.spans = append(g.spans, [2]int{cx, cx + bw})
		cx += bw + 2
	}
	return g
}

func (d *confirmDialog) view(th theme.Theme, W, H int) (string, int, int) {
	g := d.geom(W, H)
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	icon := bg.Foreground(th.Yellow).Render("  ")
	if d.danger {
		icon = bg.Foreground(th.Red).Render(" 󰆴 ")
	}
	rows := []string{icon + bg.Foreground(th.Fg).Bold(true).Render(d.title), ""}
	for i, l := range d.message {
		fg := th.Fg
		if i > 0 {
			fg = th.GreyFg2
		}
		rows = append(rows, bg.Foreground(fg).Render("  "+l))
	}
	rows = append(rows, "")

	var btns strings.Builder
	btns.WriteString(bg.Render("  "))
	for i, b := range d.buttons {
		st := lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Fg)
		if i == d.selected {
			accent := th.Blue
			if d.danger && i == 0 {
				accent = th.Red
			}
			st = lipgloss.NewStyle().Background(accent).Foreground(th.Bg).Bold(true)
		}
		btns.WriteString(st.Render("  " + b + "  "))
		btns.WriteString(bg.Render("  "))
	}
	rows = append(rows, btns.String())
	return panel(th, rows, g.w-2), g.x, g.y
}

// update handles keys and mouse; it returns done=true once the dialog closed.
func (d *confirmDialog) update(m *Model, msg tea.Msg) (tea.Cmd, bool) {
	choose := func(i int) (tea.Cmd, bool) {
		m.confirm = nil
		if i == len(d.buttons)-1 {
			m.quitting = false
		}
		return d.onChoose(m, i), true
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		if d.deliberate {
			switch key {
			case "left", "shift+tab", "up", "right", "tab", "down", "enter", "esc", "ctrl+c":
			default:
				return nil, false
			}
		}
		switch key {
		case "left", "h", "shift+tab", "up", "k":
			d.selected = (d.selected - 1 + len(d.buttons)) % len(d.buttons)
		case "right", "l", "tab", "down", "j":
			d.selected = (d.selected + 1) % len(d.buttons)
		case "enter", " ":
			return choose(d.selected)
		case "esc", "q":
			return choose(len(d.buttons) - 1)
		case "ctrl+c":
			if m.quitHeld() {
				return nil, false
			}
			return tea.Quit, true
		default:
			for i, b := range d.buttons {
				if strings.EqualFold(msg.String(), b[:1]) {
					return choose(i)
				}
			}
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
			return nil, false
		}
		g := d.geom(m.width, m.height)
		if msg.Y != g.y+1+g.buttonRow {
			return nil, false
		}
		for i, sp := range g.spans {
			if x := msg.X - g.x - 1; x >= sp[0] && x < sp[1] {
				return choose(i)
			}
		}
	}
	return nil, false
}
