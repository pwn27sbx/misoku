package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// TitledBox frames rows in the border in use (Border) with title centred on
// the top edge between two spaces, the way Neovim's floating windows show
// theirs. Every row is fitted to inner columns on bg; the border and the
// title are drawn in fg on bg, so the box is inner+2 columns wide. A title
// wider than the box is cut.
func TitledBox(rows []string, inner int, title string, fg, bg lipgloss.Color) string {
	inner = max(inner, 0)
	b := Border()
	fill := lipgloss.NewStyle().Background(bg)
	frame := fill.Foreground(fg)

	label := Truncate(" "+title+" ", inner)
	lw := ansi.StringWidth(label)
	left := (inner - lw) / 2
	right := inner - lw - left

	out := make([]string, 0, len(rows)+2)
	out = append(out, frame.Render(b.TopLeft+strings.Repeat(b.Top, left)+label+strings.Repeat(b.Top, right)+b.TopRight))
	for _, r := range rows {
		out = append(out, frame.Render(b.Left)+FitLine(r, inner, fill)+frame.Render(b.Right))
	}
	out = append(out, frame.Render(b.BottomLeft+strings.Repeat(b.Bottom, inner)+b.BottomRight))
	return strings.Join(out, "\n")
}
