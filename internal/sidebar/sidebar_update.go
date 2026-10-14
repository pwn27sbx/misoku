package sidebar

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"

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

	if !isKey {
		return m, nil
	}

	items := m.VisibleItems()
	var curr TreeItem
	hasCurr := m.cursor >= 0 && m.cursor < len(items)
	if hasCurr {
		curr = items[m.cursor]
	}

	switch key.String() {
	case "\\", "f":
		return m, findCmd

	case "up", "k":
		m.cursor--
	case "down", "j":
		m.cursor++
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		m.cursor = len(items) - 1
	case "ctrl+d", "pgdown":
		m.cursor += m.treeHeight() / 2
	case "ctrl+u", "pgup":
		m.cursor -= m.treeHeight() / 2

	case "enter", "o":
		if hasCurr {
			if key.String() == "o" && !curr.Page.IsFolder {
				return m, selectCmd(curr.Page.ID)
			}
			return m, m.activate(curr)
		}
	case "l", "right":
		if hasCurr {
			if curr.HasChildren && !curr.Expanded {
				m.expanded[curr.Page.ID] = true
			} else if !curr.HasChildren && !curr.Page.IsFolder {
				return m, selectCmd(curr.Page.ID)
			}
		}
	case "h", "left":
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
	case "z", " ", "space", "tab":
		if hasCurr && curr.HasChildren {
			m.expanded[curr.Page.ID] = !curr.Expanded
		}
	case "W":
		m.expanded = map[string]bool{}

	case "r", "e":
		return m, m.StartRenaming()
	case "n", "%":
		parent := m.ContextParentID()
		return m, func() tea.Msg { return core.PageCreatedMsg{Page: core.Page{ParentID: parent}} }
	case "a":
		if hasCurr {
			parentID := curr.Page.ID
			m.expanded[parentID] = true
			return m, func() tea.Msg { return core.PageCreatedMsg{Page: core.Page{ParentID: parentID}} }
		}
		return m, func() tea.Msg { return core.PageCreatedMsg{} }
	case "d", "x", "delete":
		if hasCurr {
			id := curr.Page.ID
			return m, func() tea.Msg { return core.PageDeletedMsg{ID: id} }
		}
	}

	m.ensureVisible()
	return m, nil
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
	if msg.Y == findRow {
		return m, findCmd
	}
	if msg.Y < treeTop {
		return m, nil
	}
	if m.renaming {
		m.renaming = false
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
		m.expanded[item.Page.ID] = !item.Expanded
		m.ensureVisible()
		return m, nil
	}
	if item.HasChildren || item.Page.IsFolder {
		return m, m.activate(item)
	}
	return m, openQuietly(item.Page.ID)
}

// displayName shows real filenames, like nvim-tree.
