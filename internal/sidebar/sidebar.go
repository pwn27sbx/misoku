// Package sidebar renders the nvim-tree style workspace explorer (<Sidebar/>).
package sidebar

import (
	"path"
	"slices"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dirbuf"
	"github.com/pwn27sbx/misoku/internal/dirplan"
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
	dimmed      bool // greyed out while another pane has the focus
	renaming    bool
	creating    bool            // the name line of a new note or folder is open
	newParent   string          // where the new entry goes, see StartCreating
	renameInput textinput.Model // also the new entry's name line
	edit        *dirbuf.Buffer  // the folder editor (m), nil when closed
	cmdOn       bool            // the editor's : command line is open
	cmdLine     string          // what was typed after :
	workspace   string
	activeID    string
	modified    map[string]bool
	keys        keymap.Map
	accordion   bool
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
		accordion:   true,
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

// SetAccordion turns accordion folders on or off. When on (the default),
// opening a folder closes its sibling folders so only one per level stays
// open; when off, any number of folders can be open at once.
func (m *Model) SetAccordion(on bool) { m.accordion = on }

// Expand opens the folder or note id in the tree.
func (m *Model) Expand(id string) { m.expand(id) }

// expand is the single place a row gets opened, so every path (keys, click,
// Expand, SetSelectedID) follows the accordion rule: with it on, the rows
// sharing id's parent are closed first. Ancestors are never siblings, so
// opening a subfolder leaves the folder holding it open. Closing a sibling
// above the cursor removes rows above it, so the cursor follows the selected
// page back to its new row.
func (m *Model) expand(id string) {
	selected, hasSel := m.SelectedPage()
	if p, ok := m.byID[id]; ok && m.accordion {
		for _, sib := range m.pages {
			if sib.ID != id && sib.ParentID == p.ParentID {
				delete(m.expanded, sib.ID)
			}
		}
	}
	m.expanded[id] = true
	if !hasSel || !m.moveCursorTo(selected.ID) {
		m.moveCursorTo(id)
	}
	m.clampCursor()
}

// moveCursorTo puts the cursor on id's visible row and reports whether it
// found one.
func (m *Model) moveCursorTo(id string) bool {
	for i, item := range m.VisibleItems() {
		if item.Page.ID == id {
			m.cursor = i
			return true
		}
	}
	return false
}

// toggle closes an open row or opens a closed one through expand.
func (m *Model) toggle(id string) {
	if m.isExpanded(id) {
		m.expanded[id] = false
		return
	}
	m.expand(id)
}

// SetSelectedID moves the cursor to id, expanding its ancestors (like
// nvim-tree's "find file") so the row is visible.
func (m *Model) SetSelectedID(id string) {
	for p, ok := m.byID[id]; ok && p.ParentID != ""; p, ok = m.byID[p.ParentID] {
		m.expand(p.ParentID)
	}
	m.moveCursorTo(id)
	m.ensureVisible()
}

// SetDimmed greys the explorer's text out while another pane has the focus.
func (m *Model) SetDimmed(d bool) { m.dimmed = d }

// SetFocused sets focus styling state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused {
		m.renaming = false
		m.creating = false
		m.renameInput.Blur()
	}
}

// IsRenaming returns whether inline renaming is active.
func (m Model) IsRenaming() bool { return m.renaming }

// IsBusy reports whether the explorer is capturing typed text (renaming,
// naming a new entry, or the folder editor).
func (m Model) IsBusy() bool { return m.renaming || m.creating || m.edit != nil }

// EditApplyMsg asks to plan and apply the folder editor's edits: every
// visited folder with its edited lines. The editor stays open until the
// app calls StopEditing, so a refused plan or a No keeps the edits.
// Leave marks the request of leaving with edits (Esc or q), whose confirm
// also closes the editor on No and offers a cancel that keeps editing.
type EditApplyMsg struct {
	Folders []dirplan.Folder
	Leave   bool
}

// IsEditing reports whether the folder editor is open.
func (m Model) IsEditing() bool { return m.edit != nil }

// StartEditing opens the folder editor on the context folder
// (ContextParentID), mini.files style: one folder as editable lines.
func (m *Model) StartEditing() { m.openEditor(m.ContextParentID()) }

// StartInsert opens the folder editor on the folder holding the selection,
// with the cursor on it, and passes it key, as mini.files does: i and a edit
// the selected name, o and O type a new line below or above it.
func (m *Model) StartInsert(key tea.KeyMsg) tea.Cmd {
	p, _ := m.SelectedPage()
	m.openEditor(p.ParentID)
	m.edit.Select(p.ID)
	_, cmd := m.edit.Update(key)
	return cmd
}

// openEditor opens the folder editor on folder id, with no pending edits.
func (m *Model) openEditor(id string) {
	m.edit, m.cmdOn = dirbuf.New(id, m.folderRows(id)), false
}

// StopEditing closes the folder editor, dropping its pending edits.
func (m *Model) StopEditing() { m.edit, m.cmdOn = nil, false }

// folderRows are the lines the editor shows for folder id: each entry's
// file name, without .md for a note and with a trailing / for a folder.
func (m Model) folderRows(id string) []dirbuf.Row {
	var rows []dirbuf.Row
	for _, p := range m.pages {
		if p.ParentID != id {
			continue
		}
		name := strings.TrimSuffix(path.Base(p.ID), ".md")
		if p.IsFolder {
			name += "/"
		}
		rows = append(rows, dirbuf.Row{OrigID: p.ID, Name: name})
	}
	return rows
}

// pendingChanges counts the editor's edits: lines new or changed in a
// folder, plus entries gone from the folder they were in.
func (m Model) pendingChanges() int {
	n := 0
	for _, id := range m.edit.Folders() {
		orig, rows := m.edit.Original(id), m.edit.Edited(id)
		kept := map[string]bool{}
		for _, r := range rows {
			kept[r.OrigID] = true
			if !slices.Contains(orig, r) {
				n++
			}
		}
		for _, r := range orig {
			if !kept[r.OrigID] {
				n++
			}
		}
	}
	return n
}

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

// CreateRequestMsg asks to create the note or folder Name names, a path
// relative to ParentID's folder ("ideas/test", "test/"). The explorer sends
// it when Enter is pressed on a non-empty new-entry line.
type CreateRequestMsg struct {
	ParentID string
	Name     string
}

// StartCreating opens an empty name line for a new note or folder under the
// context folder (ContextParentID), drawn where the entry will appear. It
// uses the rename input, so it looks the same.
func (m *Model) StartCreating() tea.Cmd {
	m.newParent = m.ContextParentID()
	if m.newParent != "" {
		m.expand(m.newParent)
	}
	m.creating = true
	m.renameInput.SetValue("")
	m.ensureVisible()
	return m.renameInput.Focus()
}

// finishCreating closes the name line. An empty name starts an Untitled
// draft, exactly what n did before it asked for a name.
func (m *Model) finishCreating() tea.Cmd {
	name := strings.TrimSpace(m.renameInput.Value())
	parent := m.newParent
	m.creating = false
	m.renameInput.Blur()
	if name == "" {
		return func() tea.Msg { return core.PageCreatedMsg{Page: core.Page{ParentID: parent}} }
	}
	return func() tea.Msg { return CreateRequestMsg{ParentID: parent, Name: name} }
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
			m.toggle(item.Page.ID)
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
