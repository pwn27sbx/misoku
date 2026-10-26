// Package dashboard renders the NvChad "nvdash" style start screen.
package dashboard

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Action defines an action triggered by a dashboard entry.
type Action int

const (
	ActionNewPage Action = iota
	ActionFind
	ActionBrowse
	ActionKeymap
	ActionQuit
	ActionOpenRecent
	ActionThemes
	ActionOpenProject
	// ActionSyncSetup opens the sync setup assistant (`:sync setup`).
	ActionSyncSetup
	// ActionGraph opens the global graph: every note and the links between
	// them (home has no open note to centre a local graph on).
	ActionGraph
	// ActionSettings opens the settings menu (`:settings`).
	ActionSettings
	ActionNone
)

// MenuItem represents a selectable button on the start screen.
type MenuItem struct {
	Key    string
	Icon   string
	Label  string
	Action Action
}

const (
	blockWidth = 56
	maxRecent  = 5
	// Projects share the 1–9 keys with Recent, after it.
	maxKeys = 9
)

// Model represents the start screen state.
type Model struct {
	theme       theme.Theme
	width       int
	height      int
	cursor      int
	menuItems   []MenuItem
	recentPages []core.Page
	projects    []core.Page
	totalNotes  int
	workspace   string
}

// banner is MISOKU in the figlet "Bloody" font, the same as the LazyVim
// dashboard header. Every row has the same width so centring keeps the
// letters aligned.
var banner = []string{
	` ███▄ ▄███▓ ██▓  ██████  ▒█████   ██ ▄█▀ █    ██ `,
	`▓██▒▀█▀ ██▒▓██▒▒██    ▒ ▒██▒  ██▒ ██▄█▒  ██  ▓██▒`,
	`▓██    ▓██░▒██▒░ ▓██▄   ▒██░  ██▒▓███▄░ ▓██  ▒██░`,
	`▒██    ▒██ ░██░  ▒   ██▒▒██   ██░▓██ █▄ ▓▓█  ░██░`,
	`▒██▒   ░██▒░██░▒██████▒▒░ ████▓▒░▒██▒ █▄▒▒█████▓ `,
	`░ ▒░   ░  ░░▓  ▒ ▒▓▒ ▒ ░░ ▒░▒░▒░ ▒ ▒▒ ▓▒░▒▓▒ ▒ ▒ `,
	`░  ░      ░ ▒ ░░ ░▒  ░ ░  ░ ▒ ▒░ ░ ░▒ ▒░░░▒░ ░ ░ `,
	`░      ░    ▒ ░░  ░  ░  ░ ░ ░ ▒  ░ ░░ ░  ░░░ ░ ░ `,
	`       ░    ░        ░      ░ ░  ░  ░      ░     `,
}

// New constructs a Dashboard model.
func New(th theme.Theme) Model {
	return Model{
		theme: th,
		menuItems: []MenuItem{
			{Key: "n", Icon: "", Label: "New Note", Action: ActionNewPage},
			{Key: "f", Icon: "󰍉", Label: "Find Note", Action: ActionFind},
			{Key: "e", Icon: "󰙅", Label: "Open Explorer", Action: ActionBrowse},
			{Key: "t", Icon: "\U000f03d8", Label: "Themes", Action: ActionThemes},
			{Key: "?", Icon: "", Label: "Keymaps", Action: ActionKeymap},
			{Key: "g", Icon: "\U000f0431", Label: "Graph", Action: ActionGraph},
			{Key: "c", Icon: "\U000f0493", Label: "Settings", Action: ActionSettings},
			{Key: "q", Icon: "󰩈", Label: "Quit", Action: ActionQuit},
		},
	}
}

// syncItem is the entry that sets up sync; SetSyncSetup shows it before
// Keymaps. Its icon is nf-md-sync, as in the status bar.
var syncItem = MenuItem{Key: "s", Icon: "\U000f04e6", Label: "Sync between computers", Action: ActionSyncSetup}

// SetSyncSetup shows the "Sync between computers" entry (key s) when
// offered is true, and removes it otherwise: the app offers it while the
// open folder does not sync. The cursor stays on the item it was on.
func (m *Model) SetSyncSetup(offered bool) {
	items := make([]MenuItem, 0, len(m.menuItems)+1)
	for i, it := range m.menuItems {
		if it.Action == ActionSyncSetup {
			if m.cursor > i {
				m.cursor-- // the items below move up a row
			}
			continue
		}
		if offered && it.Action == ActionKeymap {
			if m.cursor >= len(items) {
				m.cursor++ // the items from here move down a row
			}
			items = append(items, syncItem)
		}
		items = append(items, it)
	}
	m.menuItems = items
	m.cursor = min(m.cursor, m.itemCount()-1)
}

// SetSize updates dimensions.
func (m *Model) SetSize(w, h int) {
	m.width = max(w, 0)
	m.height = max(h, 0)
}

// SetTheme switches colours.
func (m *Model) SetTheme(th theme.Theme) { m.theme = th }

// SetWorkspace sets the path shown in the footer.
func (m *Model) SetWorkspace(p string) { m.workspace = p }

// SetRecentPages takes every note in the workspace and keeps the most
// recently modified ones.
func (m *Model) SetRecentPages(pages []core.Page) {
	recent := make([]core.Page, 0, len(pages))
	for _, p := range pages {
		if !p.IsFolder {
			recent = append(recent, p)
		}
	}
	m.totalNotes = len(recent)
	sort.SliceStable(recent, func(i, j int) bool { return recent[i].UpdatedAt.After(recent[j].UpdatedAt) })
	if len(recent) > maxRecent {
		recent = recent[:maxRecent]
	}
	m.recentPages = recent

	// Projects are the top-level folders, alphabetically, like the project
	// list of a Neovim dashboard but inside this workspace.
	var projects []core.Page
	for _, p := range pages {
		if p.IsFolder && p.ParentID == "" {
			projects = append(projects, p)
		}
	}
	sort.SliceStable(projects, func(i, j int) bool {
		return strings.ToLower(projects[i].Title) < strings.ToLower(projects[j].Title)
	})
	if room := maxKeys - len(recent); len(projects) > room {
		projects = projects[:max(room, 0)]
	}
	m.projects = projects
	m.cursor = min(m.cursor, m.itemCount()-1)
}

// Projects returns the folders listed under "FOLDERS"; their keys follow
// the recent notes' ones.
func (m Model) Projects() []core.Page { return m.projects }

// RecentPages returns the notes listed under "RECENT", newest first.
func (m Model) RecentPages() []core.Page { return m.recentPages }

func (m Model) itemCount() int { return len(m.menuItems) + len(m.recentPages) + len(m.projects) }

// itemAt resolves a selectable index to its action and (for recents) page ID.
func (m Model) itemAt(i int) (Action, string) {
	if i < 0 || i >= m.itemCount() {
		return ActionNone, ""
	}
	if i < len(m.menuItems) {
		return m.menuItems[i].Action, ""
	}
	i -= len(m.menuItems)
	if i < len(m.recentPages) {
		return ActionOpenRecent, m.recentPages[i].ID
	}
	return ActionOpenProject, m.projects[i-len(m.recentPages)].ID
}

// SelectedAction returns the action under the cursor (and the page ID for a
// recent note).
func (m Model) SelectedAction() (Action, string) { return m.itemAt(m.cursor) }

// Update handles cursor movement.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "up", "k", "shift+tab":
			m.cursor = (m.cursor - 1 + m.itemCount()) % m.itemCount()
		case "down", "j", "tab":
			m.cursor = (m.cursor + 1) % m.itemCount()
		}
	}
	return m, nil
}

type dashLine struct {
	text string
	item int // selectable index, or -1
}

func (m Model) lines() []dashLine {
	th := m.theme
	plain := lipgloss.NewStyle()
	center := func(s string) string {
		w := lipgloss.Width(s)
		left := max((blockWidth-w)/2, 0)
		return ui.FitLine(strings.Repeat(" ", left)+s, blockWidth, plain)
	}

	var out []dashLine
	add := func(s string, item int) { out = append(out, dashLine{text: s, item: item}) }

	recentRows := len(m.recentPages)
	if recentRows > 0 {
		recentRows += 2
	}
	if len(m.projects) > 0 {
		recentRows += len(m.projects) + 2
	}
	needed := len(banner) + 4 + len(m.menuItems) + recentRows + 2 + 2 // + the hint
	if m.height >= needed+2 {
		bannerStyle := lipgloss.NewStyle().Foreground(th.Blue).Bold(true)
		for _, l := range banner {
			add(center(bannerStyle.Render(l)), -1)
		}
	} else {
		add(center(lipgloss.NewStyle().Foreground(th.Blue).Bold(true).Render("M I S O K U")), -1)
	}
	add("", -1)
	add(center(lipgloss.NewStyle().Foreground(th.GreyFg2).Render("notes that live in plain Markdown files")), -1)
	add("", -1)

	button := func(idx int, icon, label, sub, key string, iconColor lipgloss.Color) string {
		selected := idx == m.cursor
		bg := plain
		fg := th.Fg
		if selected {
			bg = lipgloss.NewStyle().Background(th.OneBg)
			fg = th.Blue
		}
		left := bg.Foreground(iconColor).Render("  "+icon+"  ") + bg.Foreground(fg).Bold(selected).Render(label)
		if sub != "" {
			left += bg.Foreground(th.GreyFg).Render("  " + sub)
		}
		right := bg.Foreground(th.GreyFg2).Render(key + "  ")
		gap := blockWidth - lipgloss.Width(left) - lipgloss.Width(right)
		return left + bg.Render(strings.Repeat(" ", max(gap, 1))) + right
	}

	// Section titles are grey small caps with no icon, so they do not read
	// as folders.
	section := func(title string) string {
		return lipgloss.NewStyle().Foreground(th.GreyFg).Bold(true).Render("  " + strings.ToUpper(title))
	}

	for i, it := range m.menuItems {
		add(button(i, it.Icon, it.Label, "", it.Key, th.Blue), i)
	}

	if len(m.recentPages) > 0 {
		add("", -1)
		add(section("Recent"), -1)
		for i, p := range m.recentPages {
			sub := ""
			if dir := path.Dir(p.ID); dir != "." {
				sub = ui.Truncate(dir, 18)
			}
			idx := len(m.menuItems) + i
			add(button(idx, "", ui.Truncate(p.Title+".md", 26), sub, fmt.Sprint(i+1), th.NordBlue), idx)
		}
	}

	if len(m.projects) > 0 {
		add("", -1)
		add(section("Folders"), -1)
		for i, p := range m.projects {
			idx := len(m.menuItems) + len(m.recentPages) + i
			key := fmt.Sprint(len(m.recentPages) + i + 1)
			add(button(idx, "󰉋", ui.Truncate(p.Title, 30), "", key, th.Yellow), idx)
		}
	}

	add("", -1)
	noun := "notes"
	if m.totalNotes == 1 {
		noun = "note"
	}
	footer := lipgloss.NewStyle().Foreground(th.Green).Render("") +
		lipgloss.NewStyle().Foreground(th.GreyFg2).Render(fmt.Sprintf("  %d %s  ·  %s", m.totalNotes, noun, ui.Truncate(m.workspace, 34)))
	add(center(footer), -1)
	add("", -1)
	add(lipgloss.NewStyle().Foreground(th.GreyFg).Render("  SPC opens every other command · ? shows them all"), -1)
	return out
}

func (m Model) origin(n int) (int, int) {
	return ui.Center(m.width, m.height, blockWidth, n)
}

// Click resolves a left click at screen (x, y) to a dashboard entry,
// moving the cursor there. ok is false when nothing selectable was hit.
func (m *Model) Click(x, y int) (Action, string, bool) {
	lines := m.lines()
	left, top := m.origin(len(lines))
	row := y - top
	if row < 0 || row >= len(lines) || x < left || x >= left+blockWidth {
		return ActionNone, "", false
	}
	idx := lines[row].item
	if idx < 0 {
		return ActionNone, "", false
	}
	m.cursor = idx
	a, id := m.itemAt(idx)
	return a, id, true
}

// View renders the centered start screen.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	lines := m.lines()
	left, top := m.origin(len(lines))
	pad := strings.Repeat(" ", left)
	rows := make([]string, 0, m.height)
	for i := 0; i < top; i++ {
		rows = append(rows, "")
	}
	for _, l := range lines {
		rows = append(rows, pad+l.text)
	}
	return ui.Fit(strings.Join(rows, "\n"), m.width, m.height, lipgloss.NewStyle())
}
