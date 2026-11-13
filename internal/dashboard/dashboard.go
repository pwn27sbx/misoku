// Package dashboard renders the NvChad "nvdash" style start screen.
package dashboard

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/sidebar"
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
	// ActionLearn starts the tutorial (`misoku tutor`) from the home screen.
	ActionLearn
	ActionNone
)

// MenuItem represents a selectable button on the start screen.
type MenuItem struct {
	Key  string
	ID   string
	Icon string
	// Label is the text shown as given, such as a learn entry's; it is
	// used when the item has no text key.
	Label  string
	Action Action
	// text names the label in the i18n catalog, so the item is drawn in
	// the language in use when it is drawn.
	text i18n.Key
}

// label is the text the item shows.
func (it MenuItem) label() string {
	if it.text != "" {
		return i18n.T(it.text)
	}
	return it.Label
}

// LearnEntry is one route on the home screen: its ID and the label, such as
// "Learn · Básico 3/26".
type LearnEntry struct {
	Route string
	Label string
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
	statuses    map[string]core.NoteStatus // drawn in place of a recent note's icon
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
			{Key: "n", Icon: "", text: i18n.DashNewNote, Action: ActionNewPage},
			{Key: "f", Icon: "󰍉", text: i18n.DashFindNote, Action: ActionFind},
			{Key: "e", Icon: "󰙅", text: i18n.DashOpenExplorer, Action: ActionBrowse},
			{Key: "t", Icon: "\U000f03d8", text: i18n.DashThemes, Action: ActionThemes},
			{Key: "?", Icon: "", text: i18n.DashKeymaps, Action: ActionKeymap},
			{Key: "g", Icon: "\U000f0431", text: i18n.DashGraph, Action: ActionGraph},
			{Key: "c", Icon: "\U000f0493", text: i18n.DashSettings, Action: ActionSettings},
			{Key: "q", Icon: "󰩈", text: i18n.DashQuit, Action: ActionQuit},
		},
	}
}

// syncItem is the entry that sets up sync; SetSyncSetup shows it before
// Keymaps. Its icon is nf-md-sync, as in the status bar.
var syncItem = MenuItem{Key: "s", Icon: "\U000f04e6", text: i18n.DashSync, Action: ActionSyncSetup}

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

// learnItem is the entry that starts the tutorial, before Settings. SetLearn
// fills its label with the levels passed. Its icon is nf-fa-book, as the
// tutorial's own title.
var learnItem = MenuItem{Key: "l", Icon: "\uf02d", Action: ActionLearn}

// SetLearn shows one "Learn" entry per route, under Graph, with its label
// (such as "Learn · Básico 3/26"), and removes them when entries is empty:
// the app offers them while there is a tutorial folder to open and this is
// not a tutorial session. The first route not complete keeps the l key. The
// cursor stays on the item it was on.
func (m *Model) SetLearn(entries []LearnEntry) {
	action, id := m.itemAt(m.cursor)
	items := make([]MenuItem, 0, len(m.menuItems)+len(entries))
	for _, it := range m.menuItems {
		if it.Action == ActionLearn {
			continue
		}
		if it.Action == ActionSettings && len(entries) > 0 {
			keyed := false
			for _, e := range entries {
				learn := learnItem
				learn.ID = e.Route
				learn.Label = e.Label
				learn.Key = ""
				if !keyed && !strings.HasSuffix(e.Label, "✓") {
					learn.Key = "l"
					keyed = true
				}
				items = append(items, learn)
			}
		}
		items = append(items, it)
	}
	m.menuItems = items
	for i := 0; i < m.itemCount(); i++ {
		if a, aid := m.itemAt(i); a == action && aid == id {
			m.cursor = i
			return
		}
	}
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

// SetStatuses sets the status of each note, by ID; a recent note with one
// shows its glyph (see sidebar.StatusGlyph) in place of its icon. A nil map
// clears them.
func (m *Model) SetStatuses(statuses map[string]core.NoteStatus) { m.statuses = statuses }

// RecentPages returns the notes listed under "RECENT", newest first.
func (m Model) RecentPages() []core.Page { return m.recentPages }

func (m Model) itemCount() int { return len(m.menuItems) + len(m.recentPages) + len(m.projects) }

// itemAt resolves a selectable index to its action and (for recents) page ID.
func (m Model) itemAt(i int) (Action, string) {
	if i < 0 || i >= m.itemCount() {
		return ActionNone, ""
	}
	if i < len(m.menuItems) {
		return m.menuItems[i].Action, m.menuItems[i].ID
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
	add(center(lipgloss.NewStyle().Foreground(th.GreyFg2).Render(i18n.T(i18n.DashTagline))), -1)
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
		add(button(i, it.Icon, it.label(), "", it.Key, th.Blue), i)
	}

	if len(m.recentPages) > 0 {
		add("", -1)
		add(section(i18n.T(i18n.DashRecent)), -1)
		for i, p := range m.recentPages {
			sub := ""
			if dir := path.Dir(p.ID); dir != "." {
				sub = ui.Truncate(dir, 18)
			}
			idx := len(m.menuItems) + i
			icon, color := "", th.NordBlue
			if glyph, c, ok := sidebar.StatusGlyph(m.statuses[p.ID], th); ok {
				// Same cells as the note icon, so the labels stay aligned.
				icon, color = glyph+strings.Repeat(" ", max(lipgloss.Width(icon)-lipgloss.Width(glyph), 0)), c
			}
			add(button(idx, icon, ui.Truncate(p.Title+".md", 26), sub, fmt.Sprint(i+1), color), idx)
		}
	}

	if len(m.projects) > 0 {
		add("", -1)
		add(section(i18n.T(i18n.DashFolders)), -1)
		for i, p := range m.projects {
			idx := len(m.menuItems) + len(m.recentPages) + i
			key := fmt.Sprint(len(m.recentPages) + i + 1)
			add(button(idx, "󰉋", ui.Truncate(p.Title, 30), "", key, th.Yellow), idx)
		}
	}

	add("", -1)
	footer := lipgloss.NewStyle().Foreground(th.Green).Render("") +
		lipgloss.NewStyle().Foreground(th.GreyFg2).Render("  "+i18n.N(i18n.DashNotes, m.totalNotes, m.totalNotes)+"  ·  "+ui.Truncate(m.workspace, 34))
	add(center(footer), -1)
	add("", -1)
	add(lipgloss.NewStyle().Foreground(th.GreyFg).Render("  "+i18n.T(i18n.DashFooterHint)), -1)
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
