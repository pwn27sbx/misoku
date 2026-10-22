package content

import "github.com/pwn27sbx/misoku/internal/highlight"

// ViewColors are the syntax colours View paints the text with.
func (m Model) ViewColors() highlight.Colors { return m.hl.colors(m) }
