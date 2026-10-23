package config

import (
	"encoding/json"
	"reflect"
)

// DefaultExplorerHideWidth is the width below which the explorer hides
// when ui.explorer_hide_width is not set.
const DefaultExplorerHideWidth = 120

// Keys of the "ui" section that the settings menu writes.
const (
	UIAccordion         = "accordion"
	UIExplorerHideWidth = "explorer_hide_width"
	UIHeadingIcons      = "heading_icons"
	UIBullets           = "bullets"
	UIBorder            = "border"
)

// AccordionOn reports whether accordion folders are on (the default).
func (u UI) AccordionOn() bool { return u.Accordion == nil || *u.Accordion }

// HideWidth is the explorer's hide width, DefaultExplorerHideWidth when
// unset.
func (u UI) HideWidth() int {
	if u.ExplorerHideWidth <= 0 {
		return DefaultExplorerHideWidth
	}
	return u.ExplorerHideWidth
}

// SetUI writes values into the "ui" section of the config file at path,
// for the settings menu: a nil value removes its key (back to Misoku's
// default). It reports false, and leaves the file byte for byte, when
// every value already is the one in the file. Every other key stays as
// written, top level and inside "ui", those this version of Misoku does
// not know included; the file is written like SetNotesDir writes it
// (symlink followed, permissions kept, one atomic rename), and a file
// that is not a JSON object, or whose "ui" is not one, is never
// rewritten.
func SetUI(path string, values map[string]any) (bool, error) {
	return editFile(path, func(top map[string]json.RawMessage) (bool, error) {
		ui, err := object(top["ui"], `"ui" in `+path)
		if err != nil {
			return false, err
		}
		changed := false
		for key, value := range values {
			old, had := ui[key]
			if value == nil {
				if had {
					delete(ui, key)
					changed = true
				}
				continue
			}
			raw, err := marshal(value)
			if err != nil {
				return false, err
			}
			if had && sameJSON(old, raw) {
				continue
			}
			ui[key] = raw
			changed = true
		}
		if !changed {
			return false, nil
		}
		if len(ui) == 0 {
			delete(top, "ui")
			return true, nil
		}
		top["ui"], err = marshal(ui)
		return err == nil, err
	})
}

// sameJSON reports whether a and b hold the same JSON value, however each
// is spaced or escaped.
func sameJSON(a, b json.RawMessage) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}
