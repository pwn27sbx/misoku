package app

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// Tables

func (m *Model) tableEdited() {
	m.preview.SetContent(m.content.Value())
	m.refreshModified()
}

func (m *Model) tableHit(h preview.Hit) tea.Cmd {
	switch h.Kind {
	case "table:addrow":
		m.content.TableOpAt(h.Line, 0, "addrow")
		m.tableEdited()
	case "table:addcol":
		m.content.TableOpAt(h.Line, h.Index, "addcol")
		m.tableEdited()
	case "table:cell":
		lines := m.docLines()
		header := h.Line+1 < len(lines) && strings.Contains(lines[h.Line+1], "-") &&
			strings.Trim(strings.ReplaceAll(strings.ReplaceAll(lines[h.Line+1], "|", ""), ":", ""), " -") == ""
		// The menu dispatches on the action, never on its shown words.
		edit := i18n.TableEditCell
		if header {
			edit = i18n.TableRenameColumn
		}
		actions := []i18n.Key{edit, i18n.TableAddRowBelow, i18n.TableAddColRight, i18n.TableDeleteColumn}
		if !header {
			actions = append(actions, i18n.TableDeleteRow)
		}
		items := make([]string, len(actions))
		for i, a := range actions {
			items[i] = i18n.T(a)
		}
		label := h.Arg
		if label == "" {
			label = i18n.T(i18n.TableEmptyCell)
		}
		m.menu(label, items, func(m *Model, choice int) tea.Cmd {
			switch actions[choice] {
			case i18n.TableEditCell, i18n.TableRenameColumn:
				return m.prompt(i18n.T(edit), h.Arg, "", func(m *Model, v string) tea.Cmd {
					m.content.SetTableCell(h.Line, h.Index, v)
					m.tableEdited()
					return nil
				})
			case i18n.TableAddRowBelow:
				m.content.TableOpAt(h.Line, h.Index, "addrow")
			case i18n.TableAddColRight:
				m.content.TableOpAt(h.Line, h.Index, "addcol")
			case i18n.TableDeleteColumn:
				m.content.TableOpAt(h.Line, h.Index, "delcol")
			case i18n.TableDeleteRow:
				m.content.TableOpAt(h.Line, h.Index, "delrow")
			}
			m.tableEdited()
			return nil
		})
	}
	return nil
}
