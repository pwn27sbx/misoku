package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Interactive views: clicks in the preview edit the note's Markdown source.

// ---------------------------------------------------------------------------
// Generic prompt and menu dialogs

type promptDialog struct {
	title    string
	hint     string
	input    textinput.Model
	onSubmit func(m *Model, value string) tea.Cmd
}

type menuDialog struct {
	title    string
	items    []string
	sel      int
	onChoose func(m *Model, i int) tea.Cmd
}

func (m *Model) prompt(title, value, hint string, onSubmit func(*Model, string) tea.Cmd) tea.Cmd {
	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = 300
	in.SetValue(value)
	in.CursorEnd()
	d := &promptDialog{title: title, hint: hint, input: in, onSubmit: onSubmit}
	m.promptBox = d
	return d.input.Focus()
}

func (m *Model) menu(title string, items []string, onChoose func(*Model, int) tea.Cmd) {
	m.menuBox = &menuDialog{title: title, items: items, onChoose: onChoose}
}

func (d *promptDialog) update(m *Model, msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc", "ctrl+c":
			m.promptBox = nil
			return nil
		case "enter":
			m.promptBox = nil
			return d.onSubmit(m, strings.TrimSpace(d.input.Value()))
		}
	}
	if mm, ok := msg.(tea.MouseMsg); ok {
		if mm.Action == tea.MouseActionPress && mm.Button == tea.MouseButtonLeft {
			x, y, w, h := d.geom(m)
			if mm.X < x || mm.X >= x+w || mm.Y < y || mm.Y >= y+h {
				m.promptBox = nil
			}
		}
		return nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

func (d *promptDialog) geom(m *Model) (x, y, w, h int) {
	w = min(60, m.width-4)
	h = 6
	if d.hint != "" {
		h++
	}
	x, y = ui.Center(m.width, m.height, w, h)
	return
}

func (d *promptDialog) view(m *Model) (string, int, int) {
	th := m.theme
	x, y, w, _ := d.geom(m)
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	field := lipgloss.NewStyle().Background(th.Bg2)
	d.input.TextStyle = field.Foreground(th.Fg)
	d.input.Cursor.Style = lipgloss.NewStyle().Foreground(th.Blue)
	d.input.Cursor.TextStyle = field.Foreground(th.Fg)
	d.input.Width = inner - 6
	rows := []string{
		bg.Foreground(th.Blue).Render(" \U000f03eb ") + bg.Foreground(th.Fg).Bold(true).Render(ui.Truncate(d.title, inner-4)),
		"",
		bg.Render(" ") + ui.FitLine(field.Foreground(th.Blue).Render(" ▏")+d.input.View(), inner-2, field) + bg.Render(" "),
	}
	if d.hint != "" {
		rows = append(rows, bg.Foreground(th.GreyFg2).Render(" "+ui.Truncate(d.hint, inner-2)))
	}
	rows = append(rows, bg.Foreground(th.GreyFg).Render(" Enter save · Esc cancel"))
	return panel(th, rows, inner), x, y
}

func (d *menuDialog) geom(m *Model) (x, y, w, h int) {
	w = 34
	for _, it := range d.items {
		w = max(w, lipgloss.Width(it)+8)
	}
	w = min(w, m.width-4)
	h = len(d.items) + 4
	x, y = ui.Center(m.width, m.height, w, h)
	return
}

func (d *menuDialog) update(m *Model, msg tea.Msg) tea.Cmd {
	choose := func(i int) tea.Cmd {
		m.menuBox = nil
		return d.onChoose(m, i)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.menuBox = nil
		case "up", "k", "shift+tab":
			d.sel = (d.sel - 1 + len(d.items)) % len(d.items)
		case "down", "j", "tab":
			d.sel = (d.sel + 1) % len(d.items)
		case "enter", " ":
			return choose(d.sel)
		default:
			if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= len(d.items) {
				return choose(n - 1)
			}
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		x, y, w, h := d.geom(m)
		if msg.X < x || msg.X >= x+w || msg.Y < y || msg.Y >= y+h {
			m.menuBox = nil
			return nil
		}
		if i := msg.Y - y - 3; msg.Button == tea.MouseButtonLeft && i >= 0 && i < len(d.items) {
			return choose(i)
		}
	}
	return nil
}

func (d *menuDialog) view(m *Model) (string, int, int) {
	th := m.theme
	x, y, w, _ := d.geom(m)
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	rows := []string{bg.Foreground(th.Fg).Bold(true).Render(" " + ui.Truncate(d.title, inner-2)), ""}
	for i, it := range d.items {
		st := bg.Foreground(th.Fg)
		marker := "  "
		if i == d.sel {
			st = lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Blue).Bold(true)
			marker = "▌ "
		}
		rows = append(rows, ui.FitLine(st.Render(marker+fmt.Sprintf("%d  ", i+1)+it), inner, st))
	}
	rows = append(rows, bg.Foreground(th.GreyFg).Render(" ↑↓ Enter · Esc"))
	return panel(th, rows, inner), x, y
}
