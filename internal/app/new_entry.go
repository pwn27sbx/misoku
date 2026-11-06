package app

import (
	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// requestCreate handles the explorer's new-entry line (n): "test/" makes a
// folder, "test" a note and "ideas/test" both, after a CREATE confirmation.
// An existing note is opened instead and an existing folder selected, so
// nothing is ever overwritten.
func (m *Model) requestCreate(parentID, name string) tea.Cmd {
	e, err := m.store.ParseNewEntry(parentID, name)
	if err != nil {
		m.setError("Can't create " + name + ": " + err.Error())
		return nil
	}
	if e.Exists {
		if e.Note.ID != "" {
			cmd := m.openFile(e.Note.ID)
			m.setStatus(e.Path + " already exists: opened it")
			return cmd
		}
		m.showCreatedFolders(e.Folders)
		m.setStatus(e.Path + " already exists")
		return nil
	}
	what := "Creates the note and opens it."
	if e.Note.ID == "" {
		what = "Creates the folder."
	}
	m.confirm = &confirmDialog{
		title:   "CREATE " + e.Path,
		message: []string{what},
		buttons: []string{"Yes", "No"},
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
		m.setError("Create failed: " + err.Error())
		return nil
	}
	m.showCreatedFolders(e.Folders)
	m.setStatus("Created " + e.Path)
	if p.IsFolder || e.Note.ID == "" {
		return nil
	}
	m.sidebar.SetSelectedID(p.ID)
	m.leaveReading() // a new note is typed
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
