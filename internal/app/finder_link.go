package app

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// openLinkPicker reuses the finder to choose a note for "/ Link to page".
func (m *Model) openLinkPicker() tea.Cmd {
	cmd := m.openFinder()
	m.finder.link = true
	return cmd
}

// insertLinkTo writes a Markdown link to target, relative to the note being
// edited, at the editor cursor.
func (m *Model) insertLinkTo(target core.Page) {
	b := m.activeBuffer()
	if b == nil {
		return
	}
	from := b.dir
	if !b.draft() {
		from = path.Dir(b.id)
	}
	rel, err := filepath.Rel(filepath.FromSlash(from), filepath.FromSlash(target.ID))
	if err != nil {
		rel = target.ID
	}
	link := filepath.ToSlash(rel)
	if strings.ContainsAny(link, " ()") {
		link = "<" + link + ">"
	}
	m.content.InsertText("[" + target.Title + "](" + link + ")")
	m.preview.SetContent(m.content.Value())
	m.refreshModified()
}
