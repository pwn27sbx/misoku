package app

import (
	"path"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/i18n"
)

// WithTemplatesDir sets the vault folder of the note templates,
// templates_dir of config.json. A folder outside the vault is refused with
// a message, and the default one is used.
func WithTemplatesDir(dir string) Option {
	return func(m *Model) {
		d, err := config.VaultFolder(dir, config.DefaultTemplatesDir)
		if err != nil {
			m.setError(i18n.T(i18n.AppFolderRefused, "templates_dir", err.Error(), d))
		}
		m.templates = d
	}
}

// templatesFolder is the folder of the templates in use.
func (m *Model) templatesFolder() string {
	if m.templates == "" {
		return config.DefaultTemplatesDir
	}
	return m.templates
}

// inTemplates reports whether note id lies under the templates folder
// dir, in any letter case, as the link index leaves it out (SetExcluded).
func inTemplates(id, dir string) bool {
	return dir != "" && strings.HasPrefix(strings.ToLower(id), strings.ToLower(dir)+"/")
}

// openTemplatePicker is the "/plantilla" slash item: the Markdown notes
// under the templates folder, subfolders included, listed by their path in
// it. The one picked goes into the open note at the cursor.
func (m *Model) openTemplatePicker() tea.Cmd {
	if m.activeBuffer() == nil {
		return nil
	}
	dir := m.templatesFolder()
	var ids []string
	for _, p := range m.store.List() {
		if !p.IsFolder && strings.EqualFold(path.Ext(p.ID), ".md") && inTemplates(p.ID, dir) {
			ids = append(ids, p.ID)
		}
	}
	if len(ids) == 0 {
		m.setStatus(i18n.T(i18n.AppNoTemplates, dir))
		return nil
	}
	sort.Strings(ids)
	items := make([]string, len(ids))
	for i, id := range ids {
		items[i] = strings.TrimSuffix(id[len(dir)+1:], path.Ext(id))
	}
	m.searchMenu(i18n.T(i18n.AppTemplatesTitle), items, 0, func(m *Model, i int) tea.Cmd {
		m.insertTemplate(ids[i], items[i])
		return nil
	})
	return nil
}

// insertTemplate puts template id into the open note at the cursor, dated
// by the clock at this moment and titled after the note (its draft name
// when it was never saved). Nothing is written: the tab stays modified.
func (m *Model) insertTemplate(id, name string) {
	b := m.activeBuffer()
	if b == nil {
		return
	}
	p, err := m.store.Get(id)
	if err != nil {
		m.setError(i18n.T(i18n.AppTemplateError, name, err.Error()))
		return
	}
	m.content.InsertTemplate(p.Content, b.title, m.clock())
	m.preview.SetContent(m.content.Value())
	m.refreshModified()
	m.setStatus(i18n.T(i18n.AppTemplateDone, name))
}
