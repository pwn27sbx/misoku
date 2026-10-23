package app

import (
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// leaderGuideDelay is how long after Space the leader guide waits before it
// shows: whoever types the next key sooner never sees it.
const leaderGuideDelay = 400 * time.Millisecond

// leaderGuideMsg is the leader guide's timer firing for the Space press
// numbered seq. A tick from an earlier press, or one that arrives after the
// sequence ended, is ignored.
type leaderGuideMsg struct{ seq int }

// startLeader begins a leader sequence and schedules its guide.
func (m *Model) startLeader() tea.Cmd {
	m.leaderPending = true
	m.leaderGuide = false
	m.leaderSeq++
	seq := m.leaderSeq
	return tea.Tick(leaderGuideDelay, func(time.Time) tea.Msg { return leaderGuideMsg{seq: seq} })
}

// openLeaderGuide shows the guide if msg belongs to the latest Space press.
// Once that sequence has ended, leaderGuideOpen keeps it hidden.
func (m *Model) openLeaderGuide(msg leaderGuideMsg) {
	if msg.seq == m.leaderSeq {
		m.leaderGuide = true
	}
}

// leaderGuideOpen reports whether the guide is on screen. Whatever ends the
// sequence clears leaderPending, which hides the guide with it.
func (m *Model) leaderGuideOpen() bool { return m.leaderPending && m.leaderGuide }

// leaderGuideRow is one action of the guide: the key after Space and what it
// does.
type leaderGuideRow struct{ key, help string }

// leaderGuideColumns lists, per keymap.Groups entry, the general actions
// that have a leader key in km, in keymap.Actions order.
func leaderGuideColumns(km keymap.Map) [][]leaderGuideRow {
	cols := make([][]leaderGuideRow, len(keymap.Groups))
	for _, a := range keymap.Actions {
		key := km.LeaderLabel(a.Action)
		if key == "" {
			continue
		}
		for i, g := range keymap.Groups {
			if a.Group == g {
				cols[i] = append(cols[i], leaderGuideRow{key, a.Help})
			}
		}
	}
	return cols
}

// renderLeaderGuide draws the guide: one column per group, side by side
// when the terminal is wide enough and stacked otherwise.
func (m *Model) renderLeaderGuide() string {
	th := m.theme
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	keyStyle := bg.Foreground(th.Blue).Bold(true)
	headStyle := bg.Foreground(th.Purple).Bold(true)
	descStyle := bg.Foreground(th.Fg)

	cols := leaderGuideColumns(m.keys)
	keyW := 1
	for _, c := range cols {
		for _, r := range c {
			keyW = max(keyW, lipgloss.Width(r.key))
		}
	}
	// A column fits its longest row. The groups sit side by side when three
	// columns of at least minColW fit the terminal, and stack otherwise.
	const minColW = 24
	colW := minColW
	for _, c := range cols {
		for _, r := range c {
			colW = max(colW, min(keyW, 12)+lipgloss.Width(r.help)+4)
		}
	}
	avail := m.width - 4
	sideBySide := avail/len(cols) >= minColW
	if sideBySide {
		colW = min(colW, avail/len(cols))
	} else {
		colW = max(min(colW, avail), 1)
	}
	keyW = min(keyW, colW/2)
	column := func(i int) []string {
		rows := []string{headStyle.Render(" " + strings.ToUpper(keymap.Groups[i]))}
		for _, r := range cols[i] {
			key := keyStyle.Render(" " + padRight(ui.Truncate(r.key, keyW), keyW) + "  ")
			rows = append(rows, ui.FitLine(key+descStyle.Render(ui.Truncate(r.help, max(colW-keyW-3, 1))), colW, bg))
		}
		return rows
	}

	var body []string
	width := colW
	if sideBySide {
		width = colW * len(cols)
		blocks := make([][]string, len(cols))
		n := 0
		for i := range cols {
			blocks[i] = column(i)
			n = max(n, len(blocks[i]))
		}
		for y := 0; y < n; y++ {
			var line string
			for _, b := range blocks {
				cell := ""
				if y < len(b) {
					cell = b[y]
				}
				line += ui.FitLine(cell, colW, bg)
			}
			body = append(body, line)
		}
	} else {
		for i := range cols {
			if i > 0 {
				body = append(body, "")
			}
			body = append(body, column(i)...)
		}
	}

	footer := " These keys come from config.json · Esc to close"
	width = min(max(width, lipgloss.Width(footer)+1), max(avail, 1))
	// A short terminal loses rows of the body, never the footer: the panel
	// has a title, two blank rows, the footer and its border.
	if maxBody := m.height - footerHeight - 6; len(body) > maxBody {
		body = append(body[:max(maxBody-1, 0)], bg.Foreground(th.GreyFg).Render(" … enlarge the terminal to see all"))
	}
	rows := []string{keyStyle.Render(" SPC") + bg.Foreground(th.GreyFg).Render(" then"), ""}
	rows = append(rows, body...)
	rows = append(rows, "", bg.Foreground(th.GreyFg).Render(footer))
	return panel(th, rows, width)
}

// overlayLeaderGuide draws the guide centred above the status bar.
func (m *Model) overlayLeaderGuide(screen string) string {
	box := m.renderLeaderGuide()
	x, _ := ui.Center(m.width, m.height, lipgloss.Width(box), lipgloss.Height(box))
	y := max(m.height-footerHeight-lipgloss.Height(box), 0)
	return ui.Overlay(screen, box, x, y)
}
