package sidebar

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// Tree icons (Nerd Font).
const (
	chevronRight = " "
	chevronDown  = " "
	folderClosed = " "
	folderOpen   = " "
	folderEmpty  = " "
	markdownIcon = " "
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

	if m.edit != nil {
		return m.editView(bg)
	}

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
	at, level := m.newEntryRow(items)
	if len(items) == 0 && !m.creating {
		msg := "Empty workspace — press n"
		rows = append(rows, bg.Foreground(th.GreyFg).Italic(true).Render("  "+ui.Truncate(msg, w-3)))
	}
	end := min(m.offset+h, len(items))
	for i := m.offset; i <= end; i++ {
		if m.creating && i == max(at, m.offset) {
			rows = append(rows, m.renderNewEntry(level))
		}
		if i < end {
			rows = append(rows, m.renderItem(items[i], i == m.cursor))
		}
	}

	return ui.Fit(strings.Join(rows, "\n"), w, m.height, bg)
}

// editView draws the folder editor: the folder's path, how to apply or
// leave, its lines with the selected one (or the name being typed)
// highlighted, and the : command line while it is open.
func (m Model) editView(bg lipgloss.Style) string {
	th, w := m.theme, m.width
	folder := "/"
	if id := m.edit.Folder(); id != "" {
		folder = strings.TrimSuffix(id, ".md") + "/"
	}
	title := bg.Foreground(th.Blue).Bold(true).Render(" 󰉖 EDIT " + ui.Truncate(folder, w-9))
	hint := bg.Foreground(th.GreyFg).Render(" " + ui.Truncate("= apply · esc leave", w-2))
	out := []string{ui.FitLine(title, w, bg), ui.FitLine(hint, w, bg), ""}

	first, h := m.editWindow()
	lines, cur := m.edit.Rows(), m.edit.Cursor()
	if len(lines) == 0 && !m.edit.Editing() {
		out = append(out, bg.Foreground(th.GreyFg).Italic(true).Render("  "+ui.Truncate("Empty — o adds a line", w-3)))
	}
	for i := first; i < len(lines) && len(out) < h+editHeader; i++ {
		base := bg
		if i == cur {
			base = lipgloss.NewStyle().Background(th.OneBg2)
		}
		icon, style := editIcon(lines[i].Name, base, th)
		text := base.Foreground(th.Fg).Render(ui.Truncate(lines[i].Name, max(w-5, 1)))
		if i == cur && m.edit.Editing() {
			text = m.edit.InputView()
		}
		out = append(out, ui.FitLine(base.Render("  ")+style.Render(icon)+text, w, base))
	}
	body := strings.Join(out, "\n")
	if m.cmdOn {
		body = ui.Fit(body, w, m.height-1, bg) + "\n" + ui.FitLine(bg.Foreground(th.Fg).Render(":"+m.cmdLine), w, bg)
	}
	return ui.Fit(body, w, m.height, bg)
}

// editHeader is how many rows the folder editor draws above its lines: the
// path, the hint and a blank.
const editHeader = 3

// editWindow is the first line the folder editor shows and how many fit, so
// the selected one stays on screen.
func (m Model) editWindow() (first, h int) {
	h = max(m.height-editHeader-1, 1)
	return max(0, m.edit.Cursor()-h+1), h
}

// editIcon is the icon of a folder editor line and its style on base: a
// folder for a name ending in "/", a note otherwise.
func editIcon(name string, base lipgloss.Style, th theme.Theme) (string, lipgloss.Style) {
	if strings.HasSuffix(name, "/") {
		return folderClosed, base.Foreground(th.Folder)
	}
	return markdownIcon, base.Foreground(th.NordBlue)
}

// CursorCell is the cell, relative to the explorer's top-left corner, where
// the next character of the name being typed in the folder editor goes,
// after the indentation and the icon. ok is false when no name is being
// typed, the : line has the keys, or the cell falls outside the explorer.
func (m Model) CursorCell() (x, y int, ok bool) {
	if m.edit == nil || !m.edit.Editing() || m.cmdOn {
		return 0, 0, false
	}
	row, ok := m.edit.Current()
	if !ok {
		return 0, 0, false
	}
	first, _ := m.editWindow()
	icon, _ := editIcon(row.Name, lipgloss.NewStyle(), m.theme)
	x = lipgloss.Width("  "+icon) + m.edit.InputCursorX()
	y = editHeader + m.edit.Cursor() - first
	if x >= m.width || y >= m.height {
		return 0, 0, false
	}
	return x, y, true
}

// newEntryRow is where the new entry's name line goes: just below its
// folder, one level in, or at the top of the tree for the workspace root.
func (m Model) newEntryRow(items []TreeItem) (at, level int) {
	for i, it := range items {
		if it.Page.ID == m.newParent {
			return i + 1, it.Level + 1
		}
	}
	return 0, 0
}

// renderNewEntry draws the name line of a new note or folder, with the
// rename input and the look of a selected row.
func (m Model) renderNewEntry(level int) string {
	th := m.theme
	base := lipgloss.NewStyle().Background(th.OneBg2)
	prefix := base.Render(" "+strings.Repeat("  ", level)+"  ") + base.Foreground(th.NordBlue).Render(markdownIcon)
	m.renameInput.Width = max(m.width-lipgloss.Width(prefix)-2, 1)
	m.renameInput.TextStyle = base.Foreground(th.Fg)
	return ui.FitLine(prefix+m.renameInput.View(), m.width, base)
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
