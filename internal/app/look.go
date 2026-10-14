package app

import (
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/preview"
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
	}
}
