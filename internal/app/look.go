package app

import (
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/preview"
	uikit "github.com/pwn27sbx/misoku/internal/ui"
)

// WithUI applies the "ui" section of config.json: the preview's look and
// heading numbers, the bars' marks, the border of dialogs, accordion
// folders, the explorer's hide width and the transparent background.
func WithUI(ui config.UI) Option {
	return func(m *Model) { m.applyUI(ui) }
}

// applyUI applies ui and keeps it as the settings in use, for the
// settings menu.
func (m *Model) applyUI(ui config.UI) {
	wasTransparent := m.prefs().ui.Transparent
	m.prefs().ui = ui
	if ui.Transparent != wasTransparent {
		// WithTheme may have run first: draw the theme again under the
		// new setting, so the first frame already honours it.
		m.applyTheme(m.baseTheme, false)
	}
	m.applyLook()
	m.modeIcon = ui.ModeIcon
	m.modifiedMark = ui.ModifiedMark
	if ui.Border != "" && !uikit.SetBorder(ui.Border) {
		m.setError(i18n.T(i18n.AppUnknownBorder, ui.Border))
	}
	m.sidebar.SetAccordion(ui.AccordionOn())
	m.applyExplorerHideWidth(ui.HideWidth())
}

// applyLook gives the preview the look of the "ui" settings in use. The
// preview renders again, so a new border shows in its blocks too.
func (m *Model) applyLook() {
	ui := m.prefs().ui
	look := preview.Look{
		// A transparent background drops the heading bands, as Neovim's
		// transparent themes do: a band is a tint of a page colour the
		// terminal no longer shows.
		HeadingBackground: ui.HeadingBackground && !ui.Transparent,
		HeadingNumbers:    ui.HeadingNumbersOn(),
		Bullets:           ui.Bullets,
		Unchecked:         ui.Checkbox.Unchecked,
		Checked:           ui.Checkbox.Checked,
		Todo:              ui.Checkbox.Todo,
	}
	copy(look.HeadingIcons[:], ui.HeadingIcons)
	m.preview.SetLook(look)
}
