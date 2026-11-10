package content

import (
	"github.com/pwn27sbx/misoku/internal/highlight"
	"github.com/pwn27sbx/misoku/internal/theme"
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
