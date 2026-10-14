package app

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

type keymapSection struct {
	title string
	keys  [][2]string
}

// helpGeneral and helpExplorer pick the configurable actions shown in the
// help, in order; their keys come from the active keymap (config "keys").
var helpGeneral = []string{
	keymap.NewNote, keymap.Save, keymap.FindNote, keymap.ToggleExplorer,
	keymap.FocusExplorer, keymap.NextPane, keymap.NextTab, keymap.PrevTab,
	keymap.CloseTab, keymap.TogglePreview, keymap.Themes, keymap.Graph,
	keymap.Home, keymap.Quit,
}

var helpExplorer = []string{
	keymap.ExplorerDown, keymap.ExplorerUp, keymap.ExplorerOpen,
	keymap.ExplorerCollapse, keymap.ExplorerNewNote, keymap.ExplorerNewSubNote,
	keymap.ExplorerRename, keymap.ExplorerDelete, keymap.ExplorerFind,
}

// editorHelp is the fixed half of the help: Vim editing is not configurable.
var editorHelp = []keymapSection{
	{" EDITOR · NORMAL", [][2]string{
		{"i a A I", "Insert mode"},
		{"Tab/S-Tab", "Indent / outdent (insert)"},
		{"o / O", "New line below / above"},
		{"h j k l", "Move"},
		{"w / b", "Next / previous word"},
		{"0 / $", "Line start / end"},
		{"gg / G", "Top / bottom"},
		{"Ctrl+D/U", "Half page down / up"},
		{"x / dd", "Delete char / line"},
		{"yy / p", "Copy line / paste"},
		{"v / V", "Select, then y copy, d cut"},
		{"/ (insert)", "Block menu: headings, lists…"},
		{"[[ (insert)", "Link to a note"},
		{"gd / gx", "Follow the link under the cursor"},
	}},
	{" COMMANDS", [][2]string{
		{":w", "Save"},
		{":w name", "Save as name.md"},
		{":wq / :x", "Save and quit"},
		{":q / :q!", "Quit / discard"},
		{":bd", "Close tab"},
		{":enew", "New note"},
		{":colo name", "Switch theme"},
		{":export [file]", "Export note as PDF"},
		{":addrow/:addcol", "Table row / column"},
	}},
	{"󰍽 MOUSE", [][2]string{
		{"Click", "Tabs, tree, cursor"},
		{"Wheel", "Scroll any pane"},
		{"< / > / T", "Calendar month (preview)"},
		{"zM / zR", "Fold / unfold all (preview)"},
	}},
}

func keymapSections(km keymap.Map) [][]keymapSection {
	section := func(title string, actions []string) keymapSection {
		s := keymapSection{title: title}
		for _, a := range actions {
			if label := km.Label(a); label != "" {
				s.keys = append(s.keys, [2]string{label, keymap.HelpText(a)})
			}
		}
		return s
	}
	return [][]keymapSection{
		{section("󰕭 GENERAL", helpGeneral), section("󰙅 EXPLORER", helpExplorer)},
		editorHelp,
	}
}

// RenderKeymapModal renders the cheatsheet panel (without positioning).
func RenderKeymapModal(th theme.Theme, km keymap.Map, termWidth, termHeight int) string {
	colW := 40
	twoCols := termWidth >= colW*2+8
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	keyStyle := bg.Foreground(th.Blue).Bold(true)
	descStyle := bg.Foreground(th.Fg)
	headStyle := bg.Foreground(th.Purple).Bold(true)

	sections := keymapSections(km)
	// The key column fits the longest label, up to half the column.
	keyW := 11
	for _, col := range sections {
		for _, sec := range col {
			for _, k := range sec.keys {
				keyW = max(keyW, lipgloss.Width(k[0]))
			}
		}
	}
	keyW = min(keyW, colW/2)

	column := func(sections []keymapSection) []string {
		var rows []string
		for i, s := range sections {
			if i > 0 {
				rows = append(rows, "")
			}
			rows = append(rows, headStyle.Render(" "+s.title))
			for _, k := range s.keys {
				key := keyStyle.Render(" " + padRight(ui.Truncate(k[0], keyW), keyW))
				rows = append(rows, ui.FitLine(key+descStyle.Render(ui.Truncate(k[1], colW-keyW-2)), colW, bg))
			}
		}
		return rows
	}

	left := column(sections[0])
	right := column(sections[1])
	var body []string
	if twoCols {
		n := max(len(left), len(right))
		for i := 0; i < n; i++ {
			l, r := "", ""
			if i < len(left) {
				l = left[i]
			}
			if i < len(right) {
				r = right[i]
			}
			body = append(body, ui.FitLine(l, colW, bg)+bg.Render("  ")+ui.FitLine(r, colW, bg))
		}
	} else {
		body = append(left, append([]string{""}, right...)...)
	}

	width := colW
	if twoCols {
		width = colW*2 + 2
	}
	center := func(s string) string {
		pad := max((width-lipgloss.Width(s))/2, 0)
		return bg.Render(strings.Repeat(" ", pad)) + s
	}
	title := bg.Foreground(th.Blue).Render(" ") + bg.Foreground(th.Fg).Bold(true).Render("MISOKU KEYMAPS")
	rows := []string{center(title), ""}
	rows = append(rows, body...)
	rows = append(rows, "", center(bg.Foreground(th.GreyFg).Render("Esc / q / ? to close")))

	if maxRows := termHeight - 2; len(rows) > maxRows && maxRows > 3 {
		rows = append(rows[:maxRows-1], bg.Foreground(th.GreyFg).Render(" … enlarge the terminal to see all"))
	}
	return panel(th, rows, width)
}

func padRight(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}
