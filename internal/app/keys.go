package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines application-wide global keyboard shortcuts. These work in
// every editor mode, VSCode-style.
type KeyMap struct {
	Quit          key.Binding
	Save          key.Binding
	NewPage       key.Binding
	Find          key.Binding
	ToggleSidebar key.Binding
}

// DefaultKeyMap returns the default keybindings for Misoku.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:          key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Save:          key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
		NewPage:       key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new note")),
		Find:          key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "find note")),
		ToggleSidebar: key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("ctrl+b", "toggle explorer")),
	}
}
