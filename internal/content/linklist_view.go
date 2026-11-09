package content

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	linkListWidth   = 44
	linkListMaxRows = 10
)

// linkListIcon marks a suggestion's kind in the list.
func linkListIcon(kind string) string {
	switch kind {
	case core.LinkKindHeading:
		return "#"
	case core.LinkKindBlock:
		return "^"
	}
	return "\U000f0337"
}

// linkListGeom places the list under the cursor, or above it when there is
// no room, the way the "/" menu is placed.
func (m Model) linkListGeom(rows int) (x, y, w, h int, ok bool) {
	cx, cy, visible := m.textarea.CursorScreen()
	if !visible {
		return 0, 0, 0, 0, false
	}
	w = min(linkListWidth, m.width)
	h = rows + 2
	x = min(max(cx-1, 0), max(m.width-w, 0))
	y = cy + 1
	if y+h > m.height && cy-h >= 0 {
		y = cy - h
	}
	return x, y, w, h, true
}

// visibleLinkItems returns the suggestions on screen, keeping the selection
// visible.
func (m Model) visibleLinkItems() []core.LinkSuggestion {
	if m.linkList == nil || len(m.linkList.items) == 0 {
		return nil
	}
	items := m.linkList.items
	start := min(m.linkList.offset, len(items)-1)
	return items[start:min(start+linkListMaxRows, len(items))]
}

// LinkListView renders the inline list of link suggestions and its position
// relative to the pane.
func (m Model) LinkListView() (string, int, int, bool) {
	if m.linkList == nil {
		return "", 0, 0, false
	}
	th := m.theme
	items := m.visibleLinkItems()
	n := max(len(items), 1)
	x, y, w, _, ok := m.linkListGeom(n)
	if !ok {
		return "", 0, 0, false
	}
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.PopupBg)

	var lines []string
	if len(items) == 0 {
		lines = append(lines, ui.FitLine(bg.Foreground(th.GreyFg).Italic(true).Render("  No matches"), inner, bg))
	}
	for i, it := range items {
		idx := m.linkList.offset + i
		rowBg := th.PopupBg
		if idx == m.linkList.sel {
			rowBg = th.OneBg2
		}
		base := lipgloss.NewStyle().Background(rowBg)
		marker := base.Render("  ")
		labelFg, iconFg := th.Fg, th.GreyFg2
		if idx == m.linkList.sel {
			marker = base.Foreground(th.Blue).Render("▌ ")
			iconFg = th.Blue
		}
		line := marker + base.Foreground(iconFg).Render(linkListIcon(it.Kind)+" ") +
			base.Foreground(labelFg).Render(ui.Truncate(it.Label, inner-4))
		if it.Detail != "" {
			detail := base.Foreground(th.GreyFg2).Render(ui.Truncate(it.Detail, inner/2))
			if gap := inner - lipgloss.Width(line) - lipgloss.Width(detail) - 1; gap > 1 {
				line += base.Render(strings.Repeat(" ", gap)) + detail + base.Render(" ")
			}
		}
		lines = append(lines, ui.FitLine(line, inner, base))
	}
	box := lipgloss.NewStyle().
		BorderStyle(ui.Border()).
		BorderForeground(th.Line).
		BorderBackground(th.PopupBg).
		Render(strings.Join(lines, "\n"))
	return box, x, y, true
}

// clickLinkList inserts the item at pane coordinates (x, y), if any. A click
// outside the box closes the list and goes on to the editor.
func (m *Model) clickLinkList(x, y int) (tea.Cmd, bool) {
	if m.linkList == nil {
		return nil, false
	}
	items := m.visibleLinkItems()
	bx, by, w, h, ok := m.linkListGeom(max(len(items), 1))
	if !ok {
		return nil, false
	}
	if x < bx || x >= bx+w || y < by || y >= by+h {
		m.linkList = nil
		return nil, false
	}
	row := y - by - 1
	if row < 0 || row >= len(items) {
		return nil, true
	}
	q, ok := m.linkListQuery()
	if !ok {
		m.linkList = nil
		return nil, false
	}
	m.insertLinkSuggestion(q, items[row].Insert)
	return nil, true
}
