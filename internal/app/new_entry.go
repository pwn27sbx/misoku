package app

import (
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"

	tea "github.com/charmbracelet/bubbletea"
)

// requestCreate handles the explorer's new-entry line (n): "test/" makes a
// folder, "test" a note and "ideas/test" both, after a CREATE confirmation.
// An existing note is opened instead and an existing folder selected, so
// nothing is ever overwritten.
func (m *Model) requestCreate(parentID, name string) tea.Cmd {
	e, err := m.store.ParseNewEntry(parentID, name)
	if err != nil {
		m.setError(i18n.T(i18n.AppCantCreate, name, err.Error()))
		return nil
	}
	if e.Exists {
		if e.Note.ID != "" {
			cmd := m.openFile(e.Note.ID)
			m.setStatus(i18n.T(i18n.AppExistsOpened, e.Path))
			return cmd
		}
		m.showCreatedFolders(e.Folders)
		m.setStatus(i18n.T(i18n.AppExists, e.Path))
		return nil
	}
	what := i18n.T(i18n.AppCreatesNote)
	if e.Note.ID == "" {
		what = i18n.T(i18n.AppCreatesFolder)
	}
	m.confirm = &confirmDialog{
		title:   i18n.T(i18n.AppCreateTitle, e.Path),
		message: []string{what},
		buttons: []string{i18n.T(i18n.AppYes), i18n.T(i18n.AppNo)},
		onChoose: func(m *Model, choice int) tea.Cmd {
			if choice != 0 {
				return nil
			}
			return m.createEntry(e)
		},
	}
	return nil
}

// createEntry writes a confirmed entry and shows it: the folders on its path
// open in the explorer, a note opens in the editor (in INSERT, as a new
// draft does) and a folder is selected.
func (m *Model) createEntry(e core.NewEntry) tea.Cmd {
	p, err := m.store.CreateEntry(e)
	if err != nil {
		m.setError(i18n.T(i18n.AppCreateFailed, err.Error()))
		return nil
	}
	m.showCreatedFolders(e.Folders)
	m.setStatus(i18n.T(i18n.AppCreated, e.Path))
	if p.IsFolder || e.Note.ID == "" {
		return nil
	}
	m.sidebar.SetSelectedID(p.ID)
	return tea.Batch(m.openFile(p.ID), m.content.EnterInsert())
}

// showCreatedFolders reads the tree again, so new folders show (an empty
// one too, as the store lists it), then opens each one and selects the
// last.
func (m *Model) showCreatedFolders(folders []core.Page) {
	m.reloadTree()
	for _, f := range folders {
		m.sidebar.Expand(f.ID)
	}
	if len(folders) > 0 {
		m.sidebar.SetSelectedID(folders[len(folders)-1].ID)
	}
}
