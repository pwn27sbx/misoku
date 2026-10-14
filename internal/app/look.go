package app

import (
	"fmt"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/preview"
	uikit "github.com/pwn27sbx/misoku/internal/ui"
)

// WithUI applies the "ui" section of config.json to the preview.
func WithUI(ui config.UI) Option {
	return func(m *Model) {
		look := preview.Look{
			HeadingBackground: ui.HeadingBackground,
			Bullets:           ui.Bullets,
			Unchecked:         ui.Checkbox.Unchecked,
			Checked:           ui.Checkbox.Checked,
			Todo:              ui.Checkbox.Todo,
		}
		copy(look.HeadingIcons[:], ui.HeadingIcons)
		m.preview.SetLook(look)
		m.modeIcon = ui.ModeIcon
		m.modifiedMark = ui.ModifiedMark
		if ui.Border != "" && !uikit.SetBorder(ui.Border) {
			m.setError(fmt.Sprintf("ui: unknown border %q (use rounded or single)", ui.Border))
		}
	}
}
