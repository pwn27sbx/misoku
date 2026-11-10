package sidebar

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dirbuf"
	"github.com/pwn27sbx/misoku/internal/dirplan"
	"github.com/pwn27sbx/misoku/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
)

// Update processes key and mouse messages (mouse coordinates must be relative
// to the sidebar's top-left corner) and emits core domain events.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if mouse, ok := msg.(tea.MouseMsg); ok {
		return m.handleMouse(mouse)
	}

	var cmd tea.Cmd
	key, isKey := msg.(tea.KeyMsg)

	if m.edit != nil {
		if isKey {
			return m.updateEdit(key)
		}
		return m, nil
	}
	if m.renaming {
		if isKey {
			switch key.String() {
			case "enter":
				newTitle := strings.TrimSpace(m.renameInput.Value())
				m.renaming = false
				m.renameInput.Blur()
				if p, ok := m.SelectedPage(); ok && newTitle != "" {
					newTitle = strings.TrimSuffix(newTitle, ".md")
					p.Title = newTitle
					return m, func() tea.Msg { return core.PageUpdatedMsg{Page: p} }
				}
				return m, nil
			case "esc":
				m.renaming = false
				m.renameInput.Blur()
				return m, nil
			}
		}
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}
	if m.creating {
		if isKey {
			switch key.String() {
			case "enter":
				return m, m.finishCreating()
			case "esc":
				m.creating = false
				m.renameInput.Blur()
				return m, nil
			}
		}
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}

	if !isKey {
		return m, nil
	}

	items := m.VisibleItems()
	var curr TreeItem
	hasCurr := m.cursor >= 0 && m.cursor < len(items)
	if hasCurr {
		curr = items[m.cursor]
	}

	k := key.String()
	is := func(action string) bool { return m.keys.Direct(action, k) }
	switch {
	case is(keymap.ExplorerFind):
		return m, findCmd

	case is(keymap.ExplorerUp):
		m.cursor--
	case is(keymap.ExplorerDown):
		m.cursor++
	case is(keymap.ExplorerTop):
		m.cursor = 0
	case is(keymap.ExplorerBottom):
		m.cursor = len(items) - 1
	case is(keymap.ExplorerPageDown):
		m.cursor += m.treeHeight() / 2
	case is(keymap.ExplorerPageUp):
		m.cursor -= m.treeHeight() / 2

	case is(keymap.ExplorerOpen):
		if hasCurr {
			return m, m.activate(curr)
		}
	case is(keymap.ExplorerPreview):
		if hasCurr {
			if !curr.Page.IsFolder {
				return m, previewCmd(curr.Page.ID)
			}
			return m, m.activate(curr)
		}
	case is(keymap.ExplorerExpand):
		if hasCurr {
			if curr.HasChildren && !curr.Expanded {
				m.expand(curr.Page.ID)
			} else if !curr.HasChildren && !curr.Page.IsFolder {
				return m, selectCmd(curr.Page.ID)
			}
		}
	case is(keymap.ExplorerCollapse):
		if hasCurr {
			if curr.HasChildren && curr.Expanded {
				m.expanded[curr.Page.ID] = false
			} else if curr.Page.ParentID != "" {
				// Jump to the parent, like nvim-tree's close-node.
				for i, it := range items {
					if it.Page.ID == curr.Page.ParentID {
						m.cursor = i
						break
					}
				}
			}
		}
	case is(keymap.ExplorerToggleFold):
		if hasCurr && curr.HasChildren {
			m.toggle(curr.Page.ID)
		}
	case is(keymap.ExplorerCollapseAll):
		m.expanded = map[string]bool{}

	case is(keymap.ExplorerEditFolder):
		m.StartEditing()
		return m, nil
	case is(keymap.ExplorerInsert):
		return m, m.StartInsert(key)
	case is(keymap.ExplorerRename):
		return m, m.StartRenaming()
	case is(keymap.ExplorerNewNote):
		return m, m.StartCreating()
	case is(keymap.ExplorerNewSubNote):
		if hasCurr {
			parentID := curr.Page.ID
			m.expand(parentID)
			return m, func() tea.Msg { return core.PageCreatedMsg{Page: core.Page{ParentID: parentID}} }
		}
		return m, func() tea.Msg { return core.PageCreatedMsg{} }
	case is(keymap.ExplorerDelete):
		if hasCurr {
			id := curr.Page.ID
			return m, func() tea.Msg { return core.PageDeletedMsg{ID: id} }
		}
	}
	m.ensureVisible()
	return m, nil
}

// updateEdit handles a key of the folder editor, as Oil does: =, :w and
// Ctrl+S (the save key, also while typing) ask to apply, Esc, q and :q
// leave, h and l walk folders, and Enter asks to apply when edits are
// pending, otherwise it enters a folder or opens a note; the rest edits
// the lines. While a name is typed every other key is text: Enter only
// opens a new line, and nothing touches the disk until the confirm.
func (m Model) updateEdit(key tea.KeyMsg) (Model, tea.Cmd) {
	k := key.String()
	if m.keys.DirectModified(keymap.Save, k) {
		m.cmdOn = false
		return m, m.applyEdits(false)
	}
	if m.cmdOn {
		switch {
		case k == "enter":
			m.cmdOn = false
			switch m.cmdLine {
			case "w":
				return m, m.applyEdits(false)
			case "q":
				return m, m.leaveEditing()
			}
		case k == "esc" || k == "backspace" && m.cmdLine == "":
			m.cmdOn = false
		case k == "backspace":
			m.cmdLine = m.cmdLine[:len(m.cmdLine)-1]
		case key.Type == tea.KeyRunes:
			m.cmdLine += string(key.Runes)
		}
		return m, nil
	}
	if !m.edit.Editing() {
		switch k {
		case "=":
			return m, m.applyEdits(false)
		case ":":
			m.cmdOn, m.cmdLine = true, ""
			return m, nil
		case "esc", "q":
			return m, m.leaveEditing()
		}
	}
	nav, cmd := m.edit.Update(key)
	switch nav {
	case dirbuf.NavParent:
		if p, ok := m.byID[m.edit.Folder()]; ok {
			m.edit.Open(p.ParentID, m.folderRows(p.ParentID))
		}
	case dirbuf.NavChild:
		m.enterFolder()
	case dirbuf.NavOpen:
		if m.pendingChanges() > 0 {
			return m, m.applyEdits(false)
		}
		return m, m.activateLine()
	}
	return m, cmd
}

// enterFolder shows the folder under the editor's cursor, keeping the
// edits of the one it leaves; it reports false on a note or a new line.
func (m *Model) enterFolder() bool {
	r, _ := m.edit.Current()
	p, ok := m.byID[r.OrigID]
	if !ok || !p.IsFolder && len(m.folderRows(p.ID)) == 0 {
		return false
	}
	m.edit.Open(p.ID, m.folderRows(p.ID))
	return true
}

// activateLine is Enter on a line of the folder editor with nothing
// pending: a folder is entered, as l does, and a note opens in the editor,
// closing the folder editor behind it.
func (m *Model) activateLine() tea.Cmd {
	if m.enterFolder() {
		return nil
	}
	r, _ := m.edit.Current()
	p, ok := m.byID[r.OrigID]
	if !ok {
		return nil
	}
	m.StopEditing()
	return selectCmd(p.ID)
}

// applyEdits asks the app to plan every visited folder's edited lines;
// leave says the request comes from leaving the editor.
func (m Model) applyEdits(leave bool) tea.Cmd {
	return m.editApply(EditApplyMsg{Leave: leave})
}

// editApply sends msg with every visited folder's edited lines, taken now.
func (m Model) editApply(msg EditApplyMsg) tea.Cmd {
	var folders []dirplan.Folder
	for _, id := range m.edit.Folders() {
		f := dirplan.Folder{ID: id}
		for _, r := range m.edit.Edited(id) {
			f.Rows = append(f.Rows, dirplan.Row{OrigID: r.OrigID, Name: r.Name})
		}
		folders = append(folders, f)
	}
	msg.Folders = folders
	return func() tea.Msg { return msg }
}

// leaveEditing closes an editor without edits; with some it asks the app
// for the same confirm as =, as Oil saves on leaving.
func (m *Model) leaveEditing() tea.Cmd {
	if m.pendingChanges() == 0 {
		m.StopEditing()
		return nil
	}
	return m.applyEdits(true)
}

func (m Model) handleMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.Scroll(-3)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.Scroll(3)
		return m, nil
	}
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	if m.edit != nil {
		return m, nil // the folder editor's rows are not the tree's
	}
	if msg.Y == findRow {
		return m, findCmd
	}
	if msg.Y < treeTop {
		return m, nil
	}
	if m.renaming || m.creating {
		m.renaming = false
		m.creating = false
		m.renameInput.Blur()
	}
	idx := m.offset + msg.Y - treeTop
	items := m.VisibleItems()
	if idx < 0 || idx >= len(items) {
		return m, nil
	}
	m.cursor = idx
	item := items[idx]
	// Clicking the chevron (or a folder anywhere) toggles; clicking a note opens it.
	arrowX := 1 + item.Level*2
	if item.HasChildren && (item.Page.IsFolder || msg.X <= arrowX+1) {
		m.toggle(item.Page.ID)
		m.ensureVisible()
		return m, nil
	}
	if item.HasChildren || item.Page.IsFolder {
		return m, m.activate(item)
	}
	return m, openQuietly(item.Page.ID)
}

// displayName shows real filenames, like nvim-tree.
