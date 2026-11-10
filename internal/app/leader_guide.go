package app

import (
	"slices"
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
	m.leaderKeys = nil
	return m.scheduleLeaderGuide()
}

// scheduleLeaderGuide starts the guide's timer for the keys typed so far. A
// guide already on screen stays and shows the next page at once.
func (m *Model) scheduleLeaderGuide() tea.Cmd {
	if m.leaderGuide {
		return nil
	}
	m.leaderSeq++
	seq := m.leaderSeq
	return tea.Tick(leaderGuideDelay, func(time.Time) tea.Msg { return leaderGuideMsg{seq: seq} })
}

// visualLeaderActions are the only leader actions a VISUAL selection takes,
// as the LazyVim companion maps only these in visual mode.
var visualLeaderActions = []string{keymap.ToggleCheckbox, keymap.RemoveCheckbox}

// leaderNext adds s to the pending leader sequence. When the keys so far
// start a longer sequence of one of reach, the sequence waits for the next
// key (leaderPending stays true) and wait is its guide's timer. Otherwise the sequence ends and action
// is the one of reach the keys complete, or "".
func (m *Model) leaderNext(s string, reach []string) (keys []string, action string, wait tea.Cmd) {
	keys = append(slices.Clone(m.leaderKeys), s)
	for _, a := range reach {
		if m.keys.LeaderSeq(a, keys) {
			m.endLeader()
			return keys, a, nil
		}
	}
	if s != "esc" && m.keys.LeaderPrefix(reach, keys) {
		m.leaderKeys = keys
		return keys, "", m.scheduleLeaderGuide()
	}
	m.endLeader()
	return keys, "", nil
}

// endLeader ends the pending leader sequence and hides its guide.
func (m *Model) endLeader() {
	m.leaderPending = false
	m.leaderKeys = nil
}

// openLeaderGuide shows the guide if msg belongs to the latest Space press.
// Once that sequence has ended, leaderGuideOpen keeps it hidden.
func (m *Model) openLeaderGuide(msg leaderGuideMsg) {
	if msg.seq == m.leaderSeq {
		m.leaderGuide = true
		m.tutorCLeaderGuide()
	}
}

// leaderGuideOpen reports whether the guide is on screen. Whatever ends the
// sequence clears leaderPending, which hides the guide with it.
func (m *Model) leaderGuideOpen() bool { return m.leaderPending && m.leaderGuide }

// endLeaderUnderDialog cancels a pending leader once a dialog is open: one
// that opened without a key (a sync conflict, :messages sent from outside)
// takes the keys meanwhile, and closing it must not bring the guide back or
// read the next key as a leader key.
func (m *Model) endLeaderUnderDialog() {
	if m.saveAs != nil || m.confirm != nil || m.finder != nil || m.themes != nil ||
		m.browser != nil || m.datePick != nil || m.promptBox != nil || m.menuBox != nil || m.palette != nil {
		m.leaderPending = false
	}
}

// leaderGuideRow is one action of the guide: the key after Space and what it
// does.
type leaderGuideRow struct{ key, help string }

// leaderGuideColumn is one group of the guide: its title and its rows.
type leaderGuideColumn struct {
	title string
	rows  []leaderGuideRow
}

// leaderGuideColumns lists, per keymap.Groups entry, the actions of visible
// that have a key after Space and prefix in km, in keymap.Actions order. A
// key that starts a longer sequence is one "+name" row of
// keymap.PrefixGroup. Empty groups are left out, unless all are empty.
func leaderGuideColumns(km keymap.Map, prefix, visible []string) []leaderGuideColumn {
	cols := make([]leaderGuideColumn, len(keymap.Groups))
	for i, g := range keymap.Groups {
		cols[i].title = strings.ToUpper(g)
	}
	add := func(group string, r leaderGuideRow) {
		if i := slices.Index(keymap.Groups, group); i >= 0 {
			cols[i].rows = append(cols[i].rows, r)
		}
	}
	for _, a := range keymap.Actions {
		if !slices.Contains(visible, a.Action) {
			continue
		}
		if key := km.LeaderLabelAfter(a.Action, prefix); key != "" {
			add(a.Group, leaderGuideRow{key, a.Help})
		}
	}
	for _, sg := range km.LeaderSubgroups(visible, prefix) {
		add(keymap.PrefixGroup, leaderGuideRow{sg.Key, "+" + sg.Name})
	}
	shown := slices.DeleteFunc(slices.Clone(cols), func(c leaderGuideColumn) bool { return len(c.rows) == 0 })
	if len(shown) == 0 {
		return cols
	}
	return shown
}

// leaderVisible lists the actions the pending sequence can reach: only the
// checkbox ones on a VISUAL selection.
func (m *Model) leaderVisible() []string {
	if m.visualSelecting() {
		return visualLeaderActions
	}
	return generalActions
}

// renderLeaderGuide draws the guide: one column per group, side by side
// when the terminal is wide enough and stacked otherwise. After a key that
// leads to more ("space o"), it shows that group's keys under its name.
func (m *Model) renderLeaderGuide() string {
	th := m.theme
	bg := lipgloss.NewStyle().Background(th.PopupBg)
	keyStyle := bg.Foreground(th.Blue).Bold(true)
	headStyle := bg.Foreground(th.Purple).Bold(true)
	descStyle := bg.Foreground(th.Fg)

	cols := leaderGuideColumns(m.keys, m.leaderKeys, m.leaderVisible())
	keyW := 1
	for _, c := range cols {
		for _, r := range c.rows {
			keyW = max(keyW, lipgloss.Width(r.key))
		}
	}
	// A column fits its longest row. The groups sit side by side when every
	// column of at least minColW fits the terminal, and stack otherwise.
	const minColW = 24
	colW := minColW
	for _, c := range cols {
		for _, r := range c.rows {
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
		rows := []string{headStyle.Render(" " + cols[i].title)}
		for _, r := range cols[i].rows {
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
	title := keyStyle.Render(" "+keymap.SequenceLabel(m.leaderKeys)) + bg.Foreground(th.GreyFg).Render(" then")
	if name := keymap.GroupTitle(m.leaderKeys); name != "" {
		title += headStyle.Render("  " + name)
	}
	rows := []string{title, ""}
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
