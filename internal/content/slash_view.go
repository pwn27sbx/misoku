package content

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// slashGeom places the menu under the cursor (or above it when there is no
// room), relative to the editor pane.
func (m Model) slashGeom(rows int) (x, y, w, h int, ok bool) {
	cx, cy, visible := m.textarea.CursorScreen()
	if !visible {
		return 0, 0, 0, 0, false
	}
	w = min(slashWidth, m.width)
	h = rows + 2
	x = min(max(cx-1, 0), max(m.width-w, 0))
	y = cy + 1
	if y+h > m.height && cy-h >= 0 {
		y = cy - h
	}
	return x, y, w, h, true
}

type slashRow struct {
	header string
	item   int // index into matches, -1 for headers
}

func (m Model) slashRows(items []slashItem) []slashRow {
	q, _ := m.slashQuery()
	var rows []slashRow
	last := ""
	for i, it := range items {
		if strings.TrimSpace(q) == "" && it.section != last {
			rows = append(rows, slashRow{header: it.section, item: -1})
			last = it.section
		}
		rows = append(rows, slashRow{item: i})
	}
	return rows
}

// visibleSlashRows returns the rows on screen, keeping the selection visible.
func (m Model) visibleSlashRows() ([]slashRow, []slashItem) {
	items := m.slashMatches()
	rows := m.slashRows(items)
	selRow := 0
	for i, r := range rows {
		if r.item == m.slash.sel {
			selRow = i
		}
	}
	start := 0
	if selRow >= slashMaxRows {
		start = selRow - slashMaxRows + 1
	}
	end := min(start+slashMaxRows, len(rows))
	return rows[start:end], items
}

// SlashView renders the menu and its position relative to the pane.
func (m Model) SlashView() (string, int, int, bool) {
	if m.slash == nil {
		return "", 0, 0, false
	}
	th := m.theme
	rows, items := m.visibleSlashRows()
	if len(items) == 0 {
		rows = nil
	}
	n := max(len(rows), 1)
	x, y, w, _, ok := m.slashGeom(n)
	if !ok {
		return "", 0, 0, false
	}
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)

	var lines []string
	if len(items) == 0 {
		lines = append(lines, ui.FitLine(bg.Foreground(th.GreyFg).Italic(true).Render("  No results"), inner, bg))
	}
	for _, r := range rows {
		if r.item < 0 {
			lines = append(lines, ui.FitLine(bg.Foreground(th.GreyFg2).Render(" "+r.header), inner, bg))
			continue
		}
		it := items[r.item]
		base := bg
		labelFg, iconFg := th.Fg, th.GreyFg2
		if r.item == m.slash.sel {
			base = lipgloss.NewStyle().Background(th.OneBg2)
			iconFg = th.Blue
		}
		left := base.Foreground(iconFg).Render(" "+it.icon+"  ") + base.Foreground(labelFg).Render(it.label)
		right := base.Foreground(th.GreyFg).Render(it.hint + " ")
		gap := inner - lipgloss.Width(left) - lipgloss.Width(right)
		lines = append(lines, left+base.Render(strings.Repeat(" ", max(gap, 1)))+right)
	}
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(th.Line).
		BorderBackground(th.DarkerBg).
		Render(strings.Join(lines, "\n"))
	return box, x, y, true
}

// clickSlash applies the item at pane coordinates (x, y), if any.
func (m *Model) clickSlash(x, y int) (tea.Cmd, bool) {
	rows, items := m.visibleSlashRows()
	bx, by, w, h, ok := m.slashGeom(max(len(rows), 1))
	if !ok || x < bx || x >= bx+w || y < by || y >= by+h {
		m.slash = nil
		return nil, false
	}
	i := y - by - 1
	if i < 0 || i >= len(rows) || rows[i].item < 0 {
		return nil, true
	}
	m.slash.sel = rows[i].item
	return m.applySlash(items[rows[i].item]), true
}
