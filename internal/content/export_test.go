package content

import (
	"github.com/pwn27sbx/misoku/internal/highlight"
	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// ViewColors are the syntax colours View paints the text with.
func (m Model) ViewColors() highlight.Colors { return m.hl.colors(m) }

// SetClipboard makes p read the clipboard from read, so a test does not
// need a system clipboard. It returns a function that puts the real one back.
func SetClipboard(read func() (string, error)) (restore func()) {
	old := readClipboard
	readClipboard = read
	return func() { readClipboard = old }
}

// HighlightTheme is the palette the syntax colours were last computed for.
func (m Model) HighlightTheme() theme.Theme { return m.hl.theme }

// ConcealedView draws with the concealer of live.go, which nothing installs any
// more: the tests of its rules read it through here until it is deleted.
func (m Model) ConcealedView() string {
	if !m.dimmed {
		m.textarea.LineColors = m.hl.colors(m)
		m.textarea.RenderLine = m.lv.renderLine(m, m.textarea.LineColors)
	}
	return ui.Fit(m.textarea.View(), m.width, m.height, lipgloss.NewStyle())
}
