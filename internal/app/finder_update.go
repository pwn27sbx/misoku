package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (f *finder) update(m *Model, msg tea.Msg) tea.Cmd {
	g := finderLayout(m.width, m.height)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.finder = nil
			return nil
		case "enter":
			return m.finderOpen(f, false)
		case "ctrl+t":
			return m.finderOpen(f, true)
		case "down", "ctrl+n", "ctrl+j":
			f.move(1, g.listRows)
			return nil
		case "up", "ctrl+p", "ctrl+k":
			f.move(-1, g.listRows)
			return nil
		case "pgdown", "ctrl+d":
			f.move(g.listRows/2, g.listRows)
			return nil
		case "pgup", "ctrl+u":
			f.move(-g.listRows/2, g.listRows)
			return nil
		}
		before := f.input.Value()
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		if f.input.Value() != before {
			f.refilter()
		}
		return cmd

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			f.move(-1, g.listRows)
			return nil
		case tea.MouseButtonWheelDown:
			f.move(1, g.listRows)
			return nil
		case tea.MouseButtonLeft:
		default:
			return nil
		}
		inside := msg.X >= g.x && msg.X < g.x+g.w && msg.Y >= g.y && msg.Y < g.y+g.h
		if !inside {
			m.finder = nil
			return nil
		}
		row := msg.Y - g.y - 1 - g.listTop
		col := msg.X - g.x - 1
		if row >= 0 && row < g.listRows && col < g.listW {
			if idx := f.offset + row; idx < f.count() {
				f.sel = idx
				return m.finderOpen(f, false)
			}
		}
		return nil

	default:
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		return cmd
	}
}
