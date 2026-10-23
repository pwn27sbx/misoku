package app

import (
	"fmt"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/preview"
	uikit "github.com/pwn27sbx/misoku/internal/ui"
)

// WithUI applies the "ui" section of config.json: the preview's look, the
// bars' marks, the border of dialogs, accordion folders and the explorer's
// hide width.
func WithUI(ui config.UI) Option {
	return func(m *Model) { m.applyUI(ui) }
}

// applyUI applies ui and keeps it as the settings in use, for the
// settings menu.
func (m *Model) applyUI(ui config.UI) {
	m.prefs().ui = ui
	m.applyLook()
	m.modeIcon = ui.ModeIcon
	m.modifiedMark = ui.ModifiedMark
	if ui.Border != "" && !uikit.SetBorder(ui.Border) {
		m.setError(fmt.Sprintf("ui: unknown border %q (use rounded or single)", ui.Border))
	}
	m.sidebar.SetAccordion(ui.AccordionOn())
	m.applyExplorerHideWidth(ui.HideWidth())
}

// applyLook gives the preview the look of the "ui" settings in use. The
// preview renders again, so a new border shows in its blocks too.
func (m *Model) applyLook() {
	ui := m.prefs().ui
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
