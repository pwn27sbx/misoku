// Package sidebar renders the nvim-tree style workspace explorer (<Sidebar/>).
package sidebar

import (
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Rows above the tree: workspace title, spacer, find button, spacer.
const (
	findRow = 2
	treeTop = 4
)

// TreeItem represents a flattened visible node in the sidebar tree.
type TreeItem struct {
	Page        core.Page
	Level       int
	HasChildren bool
	Expanded    bool
	// IsLast reports whether this is the last child of its parent; Guides[i]
	// reports whether ancestor level i still has siblings below (draw "│").
	IsLast bool
	Guides []bool
}

// Model represents the Sidebar component state.
type Model struct {
	theme       theme.Theme
	pages       []core.Page
	byID        map[string]core.Page
	expanded    map[string]bool
	cursor      int
	offset      int
	focused     bool
	renaming    bool
	renameInput textinput.Model
	workspace   string
	activeID    string
	modified    map[string]bool
	keys        keymap.Map
	width       int
	height      int
}

// New constructs the explorer.
func New(th theme.Theme) Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "name"
	ti.CharLimit = 128

	return Model{
		theme:       th,
		byID:        map[string]core.Page{},
		expanded:    map[string]bool{},
		modified:    map[string]bool{},
		renameInput: ti,
		keys:        defaultKeys(),
		workspace:   "workspace",
	}
}

// SetSize updates dimensions cleanly, clamping negative values to zero.
func (m *Model) SetSize(w, h int) {
	m.width = max(w, 0)
	m.height = max(h, 0)
	m.ensureVisible()
}

// SetTheme switches colours.
func (m *Model) SetTheme(th theme.Theme) {
	m.theme = th
	m.renameInput.TextStyle = lipgloss.NewStyle().Foreground(th.Fg)
}

// SetWorkspaceName sets the label shown above the tree.
func (m *Model) SetWorkspaceName(name string) {
	if name != "" {
		m.workspace = name
	}
}

// SetActiveID marks the page shown in the editor.
func (m *Model) SetActiveID(id string) { m.activeID = id }

// SetModified marks which pages have unsaved edits.
func (m *Model) SetModified(ids map[string]bool) {
	if ids == nil {
		ids = map[string]bool{}
	}
	m.modified = ids
}

func (m Model) isExpanded(id string) bool { return m.expanded[id] }

// VisibleItems constructs the flattened, filtered list of rows displayed.
func (m Model) VisibleItems() []TreeItem {
	children := make(map[string][]core.Page)
	var roots []core.Page
	for _, p := range m.pages {
		if p.ParentID == "" {
			roots = append(roots, p)
		} else {
			children[p.ParentID] = append(children[p.ParentID], p)
		}
	}

	var items []TreeItem
	var walk func(list []core.Page, level int, guides []bool)
	walk = func(list []core.Page, level int, guides []bool) {
		for i, p := range list {
			kids := children[p.ID]
			last := i == len(list)-1
			exp := m.isExpanded(p.ID)
			items = append(items, TreeItem{
				Page:        p,
				Level:       level,
				HasChildren: len(kids) > 0,
				Expanded:    exp,
				IsLast:      last,
				Guides:      append([]bool(nil), guides...),
			})
			if len(kids) > 0 && exp {
				walk(kids, level+1, append(guides, !last))
			}
		}
	}
	walk(roots, 0, nil)
	return items
}

func (m *Model) clampCursor() {
	n := len(m.VisibleItems())
	if n == 0 {
		m.cursor = 0
		return
	}
	m.cursor = max(0, min(m.cursor, n-1))
}

func (m Model) treeHeight() int { return max(m.height-treeTop, 1) }

// ensureVisible scrolls so the cursor row is on screen.
func (m *Model) ensureVisible() {
	m.clampCursor()
	h := m.treeHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
	n := len(m.VisibleItems())
	m.offset = max(0, min(m.offset, max(0, n-h)))
}

// SetKeys sets the explorer shortcuts (the explorer.* actions).
func (m *Model) SetKeys(km keymap.Map) { m.keys = km }

func defaultKeys() keymap.Map {
	km, _ := keymap.New(nil)
	return km
}

// SetPages updates the list of pages and clamps the selection cursor.
func (m *Model) SetPages(pages []core.Page) {
	m.pages = pages
	m.byID = make(map[string]core.Page, len(pages))
	for _, p := range pages {
		m.byID[p.ID] = p
	}
	m.ensureVisible()
}

// Expand opens the folder or note id in the tree.
func (m *Model) Expand(id string) { m.expanded[id] = true }

// SetSelectedID moves the cursor to id, expanding its ancestors (like
// nvim-tree's "find file") so the row is visible.
func (m *Model) SetSelectedID(id string) {
	for p, ok := m.byID[id]; ok && p.ParentID != ""; p, ok = m.byID[p.ParentID] {
		m.expanded[p.ParentID] = true
	}
	for i, item := range m.VisibleItems() {
		if item.Page.ID == id {
			m.cursor = i
			break
		}
	}
	m.ensureVisible()
}

// SetFocused sets focus styling state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused {
		m.renaming = false
		m.renameInput.Blur()
	}
}

// IsRenaming returns whether inline renaming is active.
func (m Model) IsRenaming() bool { return m.renaming }

// IsBusy reports whether the explorer is capturing typed text (renaming).
func (m Model) IsBusy() bool { return m.renaming }

// SelectedPage returns the currently selected page item safely.
func (m Model) SelectedPage() (core.Page, bool) {
	items := m.VisibleItems()
	if m.cursor < 0 || m.cursor >= len(items) {
		return core.Page{}, false
	}
	return items[m.cursor].Page, true
}

// ContextParentID is the parent a new note should go under: the selected
// folder itself, or the folder holding the selected note.
func (m Model) ContextParentID() string {
	p, ok := m.SelectedPage()
	if !ok {
		return ""
	}
	if p.IsFolder {
		return p.ID
	}
	return p.ParentID
}

// StartRenaming activates inline title editing for the currently selected item.
func (m *Model) StartRenaming() tea.Cmd {
	p, ok := m.SelectedPage()
	if !ok {
		return nil
	}
	m.renaming = true
	m.renameInput.SetValue(p.Title)
	m.renameInput.CursorEnd()
	return m.renameInput.Focus()
}

func selectCmd(id string) tea.Cmd {
	return func() tea.Msg { return core.PageSelectedMsg{ID: id} }
}

// openQuietly opens a note but leaves keyboard focus in the explorer, like a
// single click in VSCode's explorer.
func openQuietly(id string) tea.Cmd {
	return func() tea.Msg { return core.PageSelectedMsg{ID: id, KeepFocus: true} }
}

func findCmd() tea.Msg { return core.FindRequestMsg{} }

// activate is nvim-tree's "edit" action: toggle a folder, open a note.
func (m *Model) activate(item TreeItem) tea.Cmd {
	if item.HasChildren || item.Page.IsFolder {
		if item.HasChildren {
			m.expanded[item.Page.ID] = !m.isExpanded(item.Page.ID)
			m.ensureVisible()
		}
		return nil
	}
	return selectCmd(item.Page.ID)
}

// Scroll moves the view by n rows, keeping the cursor on screen.
func (m *Model) Scroll(n int) {
	items := len(m.VisibleItems())
	h := m.treeHeight()
	m.offset = max(0, min(m.offset+n, max(0, items-h)))
	if m.cursor < m.offset {
		m.cursor = m.offset
	}
	if m.cursor >= m.offset+h {
		m.cursor = m.offset + h - 1
	}
	m.clampCursor()
}
