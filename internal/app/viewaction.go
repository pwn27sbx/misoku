package app

import (
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
	// onCancel, when set, runs once the dialog closed without an answer.
	onCancel func(m *Model) tea.Cmd
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

func (d *promptDialog) update(m *Model, msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc", "ctrl+c":
			return d.cancel(m)
		case "enter":
			m.promptBox = nil
			return d.onSubmit(m, strings.TrimSpace(d.input.Value()))
		}
	}
	if mm, ok := msg.(tea.MouseMsg); ok {
		if mm.Action == tea.MouseActionPress && mm.Button == tea.MouseButtonLeft {
			x, y, w, h := d.geom(m)
			if mm.X < x || mm.X >= x+w || mm.Y < y || mm.Y >= y+h {
				return d.cancel(m)
			}
		}
		return nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

// cancel closes the dialog without an answer.
func (d *promptDialog) cancel(m *Model) tea.Cmd {
	m.promptBox = nil
	if d.onCancel != nil {
		return d.onCancel(m)
	}
	return nil
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
