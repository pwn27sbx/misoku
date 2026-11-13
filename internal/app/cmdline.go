package app

// The floating command line: ":" and "/" open a box near the top of the
// screen, like LazyVim's noice (its command_palette preset), instead of
// taking the bottom row. The box is no modal: the note stays bright behind
// it and the statusline keeps the bottom row with its COMMAND badge.

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/ui"
)

const (
	// cmdlineY is the screen row of the box's top border, counted from 0:
	// noice's command_palette row.
	cmdlineY = 3
	// cmdlineMinW is the box's width before the text needs more.
	cmdlineMinW = 60
	// cmdlineMargin is the columns the box leaves free across the screen.
	cmdlineMargin = 4
	// cmdlineChrome is the columns around the typed text: the two borders,
	// a blank each side, the glyph and its space, and the cursor cell.
	cmdlineChrome = 7
)

// cmdlineOpen reports whether the floating command line is on screen: the
// editor's ":" or "/" prompt, unless a modal opened over it (the finder
// opens with a global key that works while typing a command).
func (m *Model) cmdlineOpen() bool {
	return m.viewMode == viewModeWorkspace && m.focus == focusEditor &&
		m.content.Mode() == content.ModeCommand && !m.modalOpen()
}

// cmdlineRect is the box's place and size on screen, borders included. The
// suggestions list goes right under it, at y+h.
func (m *Model) cmdlineRect() (x, y, w, h int) {
	w = ansi.StringWidth(m.content.CommandText()) + cmdlineChrome
	w = min(max(w, cmdlineMinW), m.width-cmdlineMargin)
	return (m.width - w) / 2, cmdlineY, w, 3
}

// cmdlineKind is the box's title and colour: noice's Cmdline in blue for
// ":", Search in yellow for "/".
func (m *Model) cmdlineKind() (string, lipgloss.Color) {
	if m.content.IsSearch() {
		return i18n.T(i18n.AppSearchTitle), m.theme.Yellow
	}
	return i18n.T(i18n.AppCmdlineTitle), m.theme.Blue
}

// drawCmdline paints the floating command line over screen when it is open.
func (m *Model) drawCmdline(screen string) string {
	if !m.cmdlineOpen() {
		return screen
	}
	x, y, w, _ := m.cmdlineRect()
	title, colour := m.cmdlineKind()
	inner := w - 2
	bg := lipgloss.NewStyle().Background(m.theme.PopupBg)
	row := bg.Render(" ") + m.content.CommandView(inner-2, colour, m.theme.PopupBg) + bg.Render(" ")
	box := ui.TitledBox([]string{row}, inner, title, colour, m.theme.PopupBg)
	return ui.Overlay(screen, box, x, y)
}

// inCmdline reports whether the screen cell (x, y) is inside the open box, so
// a click there never reaches the pane underneath.
func (m *Model) inCmdline(x, y int) bool {
	if !m.cmdlineOpen() {
		return false
	}
	bx, by, bw, bh := m.cmdlineRect()
	return x >= bx && x < bx+bw && y >= by && y < by+bh
}
