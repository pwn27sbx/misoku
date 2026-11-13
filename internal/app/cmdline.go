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
	screen = ui.Overlay(screen, box, x, y)
	if list, lx, ly, _, ok := m.cmdlineList(); ok {
		screen = ui.Overlay(screen, list, lx, ly)
	}
	return screen
}

// cmdlineList is the ":" command list and its place: right under the box,
// as wide and in the box's colour. Its rows stop above the statusline, and
// ok is false when there is no list or not even one row fits.
func (m *Model) cmdlineList() (list string, x, y, h int, ok bool) {
	if !m.cmdlineOpen() || !m.content.SuggestionsOpen() {
		return "", 0, 0, 0, false
	}
	x, by, w, bh := m.cmdlineRect()
	y = by + bh
	// The two borders and the statusline row stay out of the rows.
	rows := min(content.SuggestMaxRows, m.height-1-y-2)
	_, colour := m.cmdlineKind()
	list, ok = m.content.SuggestionsView(w, rows, colour)
	if !ok {
		return "", 0, 0, 0, false
	}
	return list, x, y, lipgloss.Height(list), true
}

// inCmdline reports whether the screen cell (x, y) is inside the open box or
// its command list, so a click there never reaches the pane underneath.
func (m *Model) inCmdline(x, y int) bool {
	if !m.cmdlineOpen() {
		return false
	}
	bx, by, bw, bh := m.cmdlineRect()
	if _, _, _, lh, ok := m.cmdlineList(); ok {
		bh += lh // the list sits right under the box, as wide
	}
	return x >= bx && x < bx+bw && y >= by && y < by+bh
}
