package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/config"
)

// dailyDateLayout names a daily note: 2026-11-06.md.
const dailyDateLayout = "2006-01-02"

// WithDailyDir sets the vault folder of the daily notes, daily_dir of
// config.json. A folder outside the vault is refused with a message, and
// the default one is used.
func WithDailyDir(dir string) Option {
	return func(m *Model) {
		d, err := config.VaultFolder(dir, config.DefaultDailyDir)
		if err != nil {
			m.setError("daily_dir: " + err.Error() + "; using " + d)
		}
		m.dailyDir = d
	}
}

// dailyFolder is the folder of the daily notes in use.
func (m *Model) dailyFolder() string {
	if m.dailyDir == "" {
		return config.DefaultDailyDir
	}
	return m.dailyDir
}

// openDailyNote opens today's note, <daily folder>/YYYY-MM-DD.md, dated by
// the clock when it is asked for. A note already on disk opens as any
// other; a draft of it already open is shown again rather than started
// twice. Otherwise a draft starts with that name and folder, set directly
// because the folder may not exist yet; Save As offers both and creates the
// folder, and nothing is written before that.
func (m *Model) openDailyNote() tea.Cmd {
	dir, name := m.dailyFolder(), m.clock().Format(dailyDateLayout)
	if p, err := m.store.Get(dir + "/" + name + ".md"); err == nil && !p.IsFolder {
		return m.openInCurrentBuffer(p.ID)
	}
	for _, b := range m.buffers {
		if b.draft() && b.dir == dir && b.title == name {
			m.showBuffer(b)
			m.setStatus(dailyDraftStatus(name, dir))
			return m.focusPane(focusEditor)
		}
	}
	cmd := m.newLinkedDraft(dir + "/" + name)
	m.setStatus(dailyDraftStatus(name, dir))
	return cmd
}

// dailyDraftStatus tells that today's note is a draft still to be saved.
func dailyDraftStatus(name, dir string) string {
	return "New daily note \"" + name + "\": save to create it in " + dir
}
