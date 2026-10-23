package sidebar

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// displayName shows real filenames, like nvim-tree.
func displayName(p core.Page) string {
	if p.IsFolder {
		return p.Title
	}
	return p.Title + ".md"
}

// View renders the explorer at exactly width × height.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	if m.dimmed {
		m.theme = greyText(m.theme)
	}
	th := m.theme
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	w := m.width

	var rows []string

	// 1. Workspace title.
	title := bg.Foreground(th.Blue).Bold(true).Render(" 󰉖 " + ui.Truncate(strings.ToUpper(m.workspace), w-4))
	rows = append(rows, ui.FitLine(title, w, bg), "")

	// 2. Find button (opens the global finder).
	fieldW := max(w-2, 4)
	field := lipgloss.NewStyle().Background(th.Bg2)
	left := field.Foreground(th.GreyFg).Render(" 󰍉 Find note")
	hint := ""
	if k := m.keys.Hint(keymap.ExplorerFind); k != "" {
		hint = field.Foreground(th.Grey).Render(k + " ")
	}
	inner := left + field.Render(strings.Repeat(" ", max(fieldW-lipgloss.Width(left)-lipgloss.Width(hint), 0))) + hint
	rows = append(rows, bg.Render(" ")+ui.FitLine(inner, fieldW, field)+bg.Render(" "))
	rows = append(rows, "")

	// 3. Tree.
	items := m.VisibleItems()
	h := m.treeHeight()
	if len(items) == 0 {
		msg := "Empty workspace — press n"
		rows = append(rows, bg.Foreground(th.GreyFg).Italic(true).Render("  "+ui.Truncate(msg, w-3)))
	}
	end := min(m.offset+h, len(items))
	for i := m.offset; i < end; i++ {
		rows = append(rows, m.renderItem(items[i], i == m.cursor))
	}

	return ui.Fit(strings.Join(rows, "\n"), w, m.height, bg)
}

func (m Model) renderItem(item TreeItem, selected bool) string {
	th := m.theme
	rowBg := th.DarkerBg
	if selected {
		rowBg = th.Bg2
		if m.focused {
			rowBg = th.OneBg2
		}
	}
	base := lipgloss.NewStyle().Background(rowBg)
	guide := base.Foreground(th.Line)

	var b strings.Builder
	b.WriteString(base.Render(" "))

	// Indent guides.
	{
		for i := 0; i < item.Level; i++ {
			switch {
			case i < item.Level-1 && i < len(item.Guides) && item.Guides[i]:
				b.WriteString(guide.Render("│ "))
			case i < item.Level-1:
				b.WriteString(base.Render("  "))
			case item.IsLast:
				b.WriteString(guide.Render("└ "))
			default:
				b.WriteString(guide.Render("│ "))
			}
		}
	}

	// Chevron + icon + name.
	name := displayName(item.Page)
	var arrow, icon string
	iconStyle := base.Foreground(th.NordBlue)
	nameStyle := base.Foreground(th.Fg)
	const (
		chevronRight = " "
		chevronDown  = " "
		folderClosed = " "
		folderOpen   = " "
		folderEmpty  = " "
		markdownIcon = " "
	)
	switch {
	case item.Page.IsFolder:
		icon = folderClosed
		if item.HasChildren {
			arrow = chevronRight
			if item.Expanded {
				arrow = chevronDown
				icon = folderOpen
			}
		} else {
			arrow = "  "
			icon = folderEmpty
		}
		iconStyle = base.Foreground(th.Folder)
		nameStyle = base.Foreground(th.Folder)
	case item.HasChildren:
		arrow = chevronRight
		if item.Expanded {
			arrow = chevronDown
		}
		icon = markdownIcon
	default:
		arrow = "  "
		icon = markdownIcon
	}
	if item.Page.ID == m.activeID {
		nameStyle = nameStyle.Bold(true).Foreground(th.Green)
	} else if selected && m.focused {
		nameStyle = nameStyle.Bold(true)
	}

	b.WriteString(base.Foreground(th.GreyFg).Render(arrow))
	b.WriteString(iconStyle.Render(icon))

	right := ""
	if m.modified[item.Page.ID] {
		right = base.Foreground(th.Yellow).Render(" ● ")
	}
	used := lipgloss.Width(b.String()) + lipgloss.Width(right)

	if m.renaming && selected {
		m.renameInput.Width = max(m.width-used-2, 1)
		m.renameInput.TextStyle = base.Foreground(th.Fg)
		b.WriteString(m.renameInput.View())
		return ui.FitLine(b.String(), m.width, base)
	}

	avail := m.width - used
	b.WriteString(nameStyle.Render(ui.Truncate(name, max(avail, 1))))
	pad := m.width - lipgloss.Width(b.String()) - lipgloss.Width(right)
	if pad > 0 {
		b.WriteString(base.Render(strings.Repeat(" ", pad)))
	}
	b.WriteString(right)
	return ui.FitLine(b.String(), m.width, base)
}

// greyText is th with every text colour set to Grey and the backgrounds
// kept: how a pane without the focus is drawn.
func greyText(th theme.Theme) theme.Theme {
	for _, c := range []*lipgloss.Color{
		&th.Fg, &th.GreyFg, &th.GreyFg2, &th.LightGrey, &th.Red, &th.Pink,
		&th.Green, &th.Blue, &th.NordBlue, &th.Yellow, &th.Purple, &th.DarkPurple,
		&th.Teal, &th.Orange, &th.Cyan, &th.Folder, &th.TitleFg, &th.MutedFg, &th.SelectedFg,
	} {
		*c = th.Grey
	}
	return th
}
