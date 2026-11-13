package app

import (
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/preview"

	tea "github.com/charmbracelet/bubbletea"
)

// Properties

// propertyHit edits a front-matter value clicked in the preview's
// properties table: a prompt prefilled with the value as the note has it
// now, written back through content.SetFrontMatter as one undoable edit
// that keeps the key's spelling.
func (m *Model) propertyHit(h preview.Hit) tea.Cmd {
	if h.Kind != preview.HitProperty {
		return nil
	}
	f, ok := core.ParseFrontMatter(m.content.Value()).Get(h.Arg)
	if !ok || f.List != nil || f.EndLine > f.Line {
		return nil // gone or turned into a list since the preview was drawn
	}
	return m.prompt(i18n.T(i18n.AppEditProperty), f.Value, f.Key, func(m *Model, v string) tea.Cmd {
		m.content.SetFrontMatter(f.Key, v)
		m.tableEdited()
		return nil
	})
}
