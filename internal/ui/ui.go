// Package ui holds small rendering helpers shared by every pane: exact-size
// box fitting (so panes never wrap or overflow and break the dividers) and
// modal overlays.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Fit forces s into exactly w columns by h rows: long lines are truncated,
// short ones padded, missing rows added and extra rows dropped. Padding uses
// fill, so a pane can paint its own background colour.
func Fit(s string, w, h int, fill lipgloss.Style) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	for i, l := range lines {
		lines[i] = FitLine(l, w, fill)
	}
	return strings.Join(lines, "\n")
}

// invisible lists code points whose rendered width terminals disagree on
// (emoji variation selectors, zero-width joiners). Dropping them before
// measuring keeps every row exactly as wide as we think it is.
var invisible = strings.NewReplacer(
	"\uFE0F", "", // VS16: emoji presentation (☁️ is 1 or 2 cells depending on the terminal)
	"\uFE0E", "", // VS15: text presentation
	"\u200D", "", // zero-width joiner
	"\u200B", "", // zero-width space
	"\u2060", "", // word joiner
)

// Sanitize removes characters whose display width is terminal-dependent.
func Sanitize(s string) string { return invisible.Replace(s) }

// FitLine truncates or pads a single line to exactly w columns.
func FitLine(l string, w int, fill lipgloss.Style) string {
	l = Sanitize(l)
	lw := ansi.StringWidth(l)
	if lw > w {
		l = ansi.Truncate(l, w, "")
		lw = ansi.StringWidth(l)
	}
	if lw < w {
		l += fill.Render(strings.Repeat(" ", w-lw))
	}
	return l
}

// FitLineEllipsis is FitLine for a line that must show it was cut: a line
// wider than w ends in "…" (styles are kept). When the cut lands on a wide
// character the line comes out one column short and is padded with fill.
func FitLineEllipsis(l string, w int, fill lipgloss.Style) string {
	l = Sanitize(l)
	if ansi.StringWidth(l) > w {
		l = ansi.Truncate(l, w, "…")
	}
	return FitLine(l, w, fill)
}

// Column renders a vertical divider of height h.
func Column(glyph string, h int, style lipgloss.Style) string {
	if h <= 0 {
		return ""
	}
	cell := style.Render(glyph)
	rows := make([]string, h)
	for i := range rows {
		rows[i] = cell
	}
	return strings.Join(rows, "\n")
}

// Overlay draws fg on top of bg with its top-left corner at (x, y). Both are
// multi-line ANSI strings; cells outside fg keep bg's content.
func Overlay(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	for i, fl := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			break
		}
		bl := bgLines[row]
		fw := ansi.StringWidth(fl)
		left := ansi.Truncate(bl, x, "")
		if lw := ansi.StringWidth(left); lw < x {
			left += strings.Repeat(" ", x-lw)
		}
		right := ansi.TruncateLeft(bl, x+fw, "")
		bgLines[row] = left + "\x1b[0m" + fl + "\x1b[0m" + right
	}
	return strings.Join(bgLines, "\n")
}

// Center returns the top-left position that centres a w×h box in W×H.
func Center(W, H, w, h int) (int, int) {
	x := (W - w) / 2
	y := (H - h) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

// Truncate shortens plain text to w columns with an ellipsis.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return ansi.Truncate(s, w, "…")
}

// Paint gives every cell of a rendered screen a default foreground and
// background, so a theme looks the same whatever the terminal's own colours
// are. paint is a style carrying just those two colours.
func Paint(screen string, paint lipgloss.Style) string {
	sample := paint.Render("x")
	i := strings.Index(sample, "x")
	if i <= 0 {
		return screen // no colour support (e.g. tests): nothing to do
	}
	prefix := sample[:i]
	const reset = "\x1b[0m"
	lines := strings.Split(screen, "\n")
	for n, l := range lines {
		l = strings.ReplaceAll(l, reset, reset+prefix)
		l = strings.ReplaceAll(l, "\x1b[m", reset+prefix)
		lines[n] = prefix + l + reset
	}
	return strings.Join(lines, "\n")
}

// border frames dialogs, menus and preview blocks (config ui.border).
var border = lipgloss.RoundedBorder()

// Border returns the frame style in use.
func Border() lipgloss.Border { return border }

// SetBorder chooses the frame style by name: "rounded" (default) or
// "single". It reports false for an unknown name and keeps the current one.
func SetBorder(name string) bool {
	switch name {
	case "rounded":
		border = lipgloss.RoundedBorder()
	case "single":
		border = lipgloss.NormalBorder()
	default:
		return false
	}
	return true
}
