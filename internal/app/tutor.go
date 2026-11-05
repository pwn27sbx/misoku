package app

import (
	"fmt"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// tutorGame is the tutorial being played (misoku tutor): which levels pass,
// and the note whose "- []" hint was already shown.
type tutorGame struct {
	progress *tutor.Progress
	hinted   string // buffer id the hint showed for, "" once the hint no longer applies
}

// WithTutor plays the tutorial in this vault: editing a lesson note checks its
// level live, and the statusline counts the levels passed. p comes from
// tutor.Scan, run before the program starts; a nil p leaves the tutorial off.
func WithTutor(p *tutor.Progress) Option {
	return func(m *Model) {
		if p != nil {
			m.tutor = &tutorGame{progress: p}
		}
	}
}

// tutorEdited re-checks the open lesson note after an edit: the first pass
// of its level toasts it (the last one, the whole tutorial), and a "- []"
// toasts the hint once until it is fixed. It reads only the editor and the
// explorer's notes, never the disk.
func (m *Model) tutorEdited() tea.Cmd {
	id := m.currentNoteID()
	if m.tutor == nil || id == "" || m.tutor.progress.Level(id) == 0 {
		return nil
	}
	text := m.content.Value()
	exists := func(name string) bool {
		_, ok := core.ResolveNote(m.notes, name, id)
		return ok
	}
	p := m.tutor.progress
	if p.Update(id, text, exists) {
		done, total := p.Count()
		if done == total {
			return m.showToast(fmt.Sprintf("🏆 Tutorial completo (%d/%d)", done, total))
		}
		return m.showToast(fmt.Sprintf("✓ Nivel %d superado (%d/%d)", p.Level(id), done, total))
	}
	hint := tutor.Hint(id, text)
	switch {
	case hint == "":
		if m.tutor.hinted == id {
			m.tutor.hinted = ""
		}
	case m.tutor.hinted != id:
		m.tutor.hinted = id
		return m.showToast(hint)
	}
	return nil
}

// tutorBadge is the statusline's "Tutor k/n" while the tutorial is played,
// "" otherwise.
func (m *Model) tutorBadge() string {
	if m.tutor == nil {
		return ""
	}
	done, total := m.tutor.progress.Count()
	return fmt.Sprintf("Tutor %d/%d", done, total)
}
