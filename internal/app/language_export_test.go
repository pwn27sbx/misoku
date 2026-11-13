package app

import (
	"github.com/pwn27sbx/misoku/internal/i18n"

	tea "github.com/charmbracelet/bubbletea"
)

// WithSystemLanguage makes "Auto (system)" in the settings menu resolve to
// l, so a test does not read the machine's language. It returns the
// function that puts the real reader back.
func WithSystemLanguage(l i18n.Lang) (restore func()) {
	old := systemLanguage
	systemLanguage = func() i18n.Lang { return l }
	return func() { systemLanguage = old }
}

// Status is the message the bar holds.
func Status(m Model) string { return m.status }

// TutorSaveFailed is a progress write of the tutorial that failed with err.
func TutorSaveFailed(err error) tea.Msg { return tutorSaveMsg{err: err} }
