// Package keymap holds Misoku's configurable shortcuts: every action, its
// default keys (today's shortcuts) and the user's overrides from the "keys"
// section of config.json.
//
// A key is written the way Bubble Tea names it ("ctrl+b", "tab", "G", "\\")
// or as a leader sequence: "space e" means Space, then e.
package keymap

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Actions. The explorer ones only apply while the explorer has focus.
const (
	Quit           = "quit"
	Save           = "save"
	NewNote        = "new_note"
	FindNote       = "find_note"
	ToggleExplorer = "toggle_explorer"
	FocusExplorer  = "focus_explorer"
	NextPane       = "next_pane"
	PrevPane       = "prev_pane"
	PaneLeft       = "pane_left"
	PaneRight      = "pane_right"
	NextTab        = "next_tab"
	PrevTab        = "prev_tab"
	CloseTab       = "close_tab"
	TogglePreview  = "toggle_preview"
	ReadingMode    = "reading_mode"
	ExportPDF      = "export_pdf"
	Themes         = "themes"
	Graph          = "graph"
	Home           = "home"
	SyncNow        = "sync_now"
	Help           = "help"

	ExplorerFind        = "explorer.find"
	ExplorerUp          = "explorer.up"
	ExplorerDown        = "explorer.down"
	ExplorerTop         = "explorer.top"
	ExplorerBottom      = "explorer.bottom"
	ExplorerPageDown    = "explorer.page_down"
	ExplorerPageUp      = "explorer.page_up"
	ExplorerOpen        = "explorer.open"
	ExplorerPreview     = "explorer.preview"
	ExplorerExpand      = "explorer.expand"
	ExplorerCollapse    = "explorer.collapse"
	ExplorerToggleFold  = "explorer.toggle_fold"
	ExplorerCollapseAll = "explorer.collapse_all"
	ExplorerRename      = "explorer.rename"
	ExplorerNewNote     = "explorer.new_note"
	ExplorerNewSubNote  = "explorer.new_sub_note"
	ExplorerDelete      = "explorer.delete"
)

// leader starts a two-key sequence ("space e").
const leader = "space"

// Info describes one action for the help screen.
type Info struct {
	Action   string
	Section  string // "general" or "explorer"
	Help     string
	defaults []string
}

// Actions lists every action in help-screen order, with today's shortcuts as
// defaults.
var Actions = []Info{
	{NewNote, "general", "New note (unsaved tab)", []string{"ctrl+n", "space n"}},
	{Save, "general", "Save (new notes: pick folder)", []string{"ctrl+s", "space w"}},
	{FindNote, "general", "Find note (opens in this tab)", []string{"ctrl+p", "\\", "space f"}},
	{ToggleExplorer, "general", "Toggle explorer", []string{"ctrl+b"}},
	{FocusExplorer, "general", "Focus explorer", []string{"space e"}},
	{NextPane, "general", "Next pane", []string{"tab"}},
	{PrevPane, "general", "Previous pane", []string{"shift+tab"}},
	{PaneLeft, "general", "Pane to the left", []string{"ctrl+h"}},
	{PaneRight, "general", "Pane to the right", []string{"ctrl+l"}},
	{NextTab, "general", "Next tab", []string{"]", "space tab"}},
	{PrevTab, "general", "Previous tab", []string{"[", "space shift+tab"}},
	{CloseTab, "general", "Close tab", []string{"space x"}},
	{TogglePreview, "general", "Toggle preview", []string{"space p"}},
	{ReadingMode, "general", "Reading mode: preview only", []string{"space r"}},
	{ExportPDF, "general", "Export note as PDF", []string{"space P"}},
	{Themes, "general", "Themes", []string{"space t"}},
	{Graph, "general", "Graph of this note's links", []string{"space g"}},
	{Home, "general", "Home screen", []string{"space d"}},
	{SyncNow, "general", "Sync the vault with git now", []string{"space s"}},
	{Help, "general", "This help", []string{"?", "space h", "space ?"}},
	{Quit, "general", "Quit (asks to save)", []string{"ctrl+c"}},

	{ExplorerUp, "explorer", "Move up", []string{"up", "k"}},
	{ExplorerDown, "explorer", "Move down", []string{"down", "j"}},
	{ExplorerTop, "explorer", "First item", []string{"g", "home"}},
	{ExplorerBottom, "explorer", "Last item", []string{"G", "end"}},
	{ExplorerPageDown, "explorer", "Half page down", []string{"ctrl+d", "pgdown"}},
	{ExplorerPageUp, "explorer", "Half page up", []string{"ctrl+u", "pgup"}},
	{ExplorerOpen, "explorer", "Open note / expand", []string{"enter"}},
	{ExplorerPreview, "explorer", "Preview note (folder: open)", []string{"o"}},
	{ExplorerExpand, "explorer", "Expand / open", []string{"l", "right"}},
	{ExplorerCollapse, "explorer", "Collapse / go to parent", []string{"h", "left"}},
	{ExplorerToggleFold, "explorer", "Fold / unfold", []string{"z"}},
	{ExplorerCollapseAll, "explorer", "Collapse all", []string{"W"}},
	{ExplorerNewNote, "explorer", "New note in this folder", []string{"n", "%"}},
	{ExplorerNewSubNote, "explorer", "New sub-note", []string{"a"}},
	{ExplorerRename, "explorer", "Rename", []string{"r", "e"}},
	{ExplorerDelete, "explorer", "Delete (asks first)", []string{"d", "x", "delete"}},
	{ExplorerFind, "explorer", "Find note", []string{"\\", "f"}},
}

// HelpText returns the help-screen description of action.
func HelpText(action string) string {
	for _, a := range Actions {
		if a.Action == action {
			return a.Help
		}
	}
	return action
}

// Map is the resolved shortcut table.
type Map struct {
	keys map[string][]string
}

// New builds the shortcut table from the defaults and the user's "keys"
// section. A configured action replaces all of its default keys; an empty list
// unbinds it. Mistakes are skipped and described in the returned warnings.
func New(user map[string][]string) (Map, []string) {
	m := Map{keys: map[string][]string{}}
	known := map[string]bool{}
	for _, a := range Actions {
		m.keys[a.Action] = a.defaults
		known[a.Action] = true
	}

	var warnings []string
	names := make([]string, 0, len(user))
	for name := range user {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if !known[name] {
			warnings = append(warnings, fmt.Sprintf("keys: unknown action %q", name))
			continue
		}
		raw := user[name]
		keys := []string{}
		for _, k := range raw {
			n, err := normalize(k)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("keys: %s: %v", name, err))
				continue
			}
			keys = append(keys, n)
		}
		if len(raw) > 0 && len(keys) == 0 {
			continue // every key was rejected: keep the defaults
		}
		m.keys[name] = keys
	}
	return m, append(warnings, m.clashes()...)
}

// sameAction pairs an explorer action with the general action that does the
// same thing: when they share a key, the general one runs first and the
// result is identical, so the explorer key is not really shadowed.
var sameAction = map[string]string{ExplorerFind: FindNote}

// clashes reports keys bound to two actions of the same section (general
// actions among themselves, explorer actions among themselves): only one of
// them can run, so the config should unbind the other. It also reports
// explorer keys the explorer never receives, see shadowed.
func (m Map) clashes() []string {
	var out []string
	owner := map[string]string{}
	for _, a := range Actions {
		for _, k := range m.keys[a.Action] {
			id := a.Section + "\x00" + k
			if first, ok := owner[id]; ok {
				out = append(out, fmt.Sprintf("keys: %q is bound to both %s and %s; unbind one of them", k, first, a.Action))
				continue
			}
			owner[id] = a.Action
		}
	}
	return append(out, m.shadowed()...)
}

// shadowed reports explorer keys that the app consumes before the explorer
// sees them: the leader and its sequences, and keys of general actions,
// which are dispatched first even while the explorer has the focus.
func (m Map) shadowed() []string {
	general := map[string][]string{}
	for _, a := range Actions {
		if a.Section != "general" {
			continue
		}
		for _, k := range m.keys[a.Action] {
			general[k] = append(general[k], a.Action)
		}
	}
	var out []string
	for _, a := range Actions {
		if a.Section != "explorer" {
			continue
		}
		for _, k := range m.keys[a.Action] {
			if k == leader || strings.HasPrefix(k, leader+" ") {
				out = append(out, fmt.Sprintf("keys: %q on %s starts a leader sequence, so the explorer never gets it; pick another key", k, a.Action))
				continue
			}
			for _, g := range general[k] {
				if g == sameAction[a.Action] {
					continue
				}
				out = append(out, fmt.Sprintf("keys: %q is bound to both %s and %s, and %s runs first even in the explorer; unbind one of them", k, g, a.Action, g))
			}
		}
	}
	return out
}

// normalize turns "Space e" or "Ctrl+E" into "space e" or "ctrl+e". Single
// characters keep their case: "G" and "g" are different keys.
func normalize(k string) (string, error) {
	fields := strings.Fields(k)
	switch len(fields) {
	case 1:
		return token(fields[0]), nil
	case 2:
		if token(fields[0]) != leader {
			return "", fmt.Errorf("%q: a two-key shortcut must start with space", k)
		}
		return leader + " " + token(fields[1]), nil
	}
	return "", fmt.Errorf("%q: use one key or space plus one key", k)
}

// token normalizes one key: named keys and modifier combos are lower case
// ("Ctrl+E" → "ctrl+e", "Tab" → "tab"); a single character keeps its case.
func token(t string) string {
	if len([]rune(t)) == 1 {
		return t
	}
	return strings.ToLower(t)
}

// fromMsg converts a Bubble Tea key name to this package's form.
func fromMsg(s string) string {
	if s == " " {
		return leader
	}
	return s
}

// Keys returns the keys bound to action.
func (m Map) Keys(action string) []string { return m.keys[action] }

// Direct reports whether the key s, pressed on its own, triggers action.
func (m Map) Direct(action, s string) bool {
	s = fromMsg(s)
	for _, k := range m.keys[action] {
		if k == s {
			return true
		}
	}
	return false
}

// DirectModified is Direct restricted to Ctrl and Alt shortcuts: the ones
// that may fire while typing, because they never insert text.
func (m Map) DirectModified(action, s string) bool {
	s = fromMsg(s)
	if !strings.HasPrefix(s, "ctrl+") && !strings.HasPrefix(s, "alt+") {
		return false
	}
	return m.Direct(action, s)
}

// IsLeader reports whether s starts a leader sequence.
func (m Map) IsLeader(s string) bool { return fromMsg(s) == leader }

// Leader reports whether s, pressed right after the leader, triggers action.
func (m Map) Leader(action, s string) bool {
	want := leader + " " + fromMsg(s)
	for _, k := range m.keys[action] {
		if k == want {
			return true
		}
	}
	return false
}

// Label renders the keys of action for the help screen: "SPC e / Ctrl+E".
func (m Map) Label(action string) string {
	parts := make([]string, 0, len(m.keys[action]))
	for _, k := range m.keys[action] {
		parts = append(parts, keyLabel(k))
	}
	return strings.Join(parts, " / ")
}

// Hint renders only the first key of action, spelled as in Label, for
// one-line hints ("Ctrl+P"). It is empty when the action is unbound.
func (m Map) Hint(action string) string {
	if keys := m.keys[action]; len(keys) > 0 {
		return keyLabel(keys[0])
	}
	return ""
}

// keyLabel renders one key or leader sequence: "SPC Shift+P", "Ctrl+E".
func keyLabel(k string) string {
	var toks []string
	for i, t := range strings.Fields(k) {
		l := label(t)
		// After SPC a lone capital is easy to misread as the lower-case
		// key, so spell out Shift ("SPC Shift+P"). A capital on its own
		// ("G") stays as is.
		if i > 0 && len([]rune(t)) == 1 && unicode.IsUpper([]rune(t)[0]) {
			l = "Shift+" + l
		}
		toks = append(toks, l)
	}
	return strings.Join(toks, " ")
}

func label(t string) string {
	switch t {
	case leader:
		return "SPC"
	case "shift+tab":
		return "S-Tab"
	}
	if len([]rune(t)) == 1 {
		return t
	}
	segs := strings.Split(t, "+")
	for i, s := range segs {
		if len([]rune(s)) == 1 {
			segs[i] = strings.ToUpper(s)
		} else {
			segs[i] = strings.ToUpper(s[:1]) + s[1:]
		}
	}
	return strings.Join(segs, "+")
}
