// Package keymap holds Misoku's configurable shortcuts: every action, its
// default keys (today's shortcuts) and the user's overrides from the "keys"
// section of config.json.
//
// A key is written the way Bubble Tea names it ("ctrl+b", "tab", "G", "\\")
// or as a leader sequence: "space e" means Space, then e, and "space o c"
// Space, then o, then c.
package keymap

import (
	"fmt"
	"slices"
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
	GoBack         = "go_back"
	CloseTab       = "close_tab"
	ReadingView    = "reading_view"
	EditorView     = "editor_view"
	ExportPDF      = "export_pdf"
	Themes         = "themes"
	Graph          = "graph"
	GlobalGraph    = "global_graph"
	Home           = "home"
	DailyNote      = "daily_note"
	SyncNow        = "sync_now"
	PasteImage     = "paste_image"
	ToggleCheckbox = "toggle_checkbox"
	RemoveCheckbox = "remove_checkbox"
	NextLink       = "next_link"
	PrevLink       = "prev_link"
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
	ExplorerFileManager = "explorer.file_manager"
	ExplorerEditFolder  = "explorer.edit_folder"
	ExplorerInsert      = "explorer.insert"
)

// leader starts a sequence of one or two more keys ("space e", "space o c").
const leader = "space"

// PrefixGroup is the leader guide column that lists a key starting a longer
// sequence, such as o for the "space o …" keys.
const PrefixGroup = GroupNote

// prefixNames names the leader groups whose keys take one more key, as
// which-key names a group; the guide shows "+misoku" for o.
var prefixNames = map[string]string{"o": "misoku"}

// Leader guide groups: the columns of the panel that lists the keys
// following Space.
const (
	GroupNote = "note"
	GroupView = "view"
	GroupGo   = "go"
)

// Groups lists the leader guide groups in display order.
var Groups = []string{GroupNote, GroupView, GroupGo}

// Info describes one action for the help screen and the command palette.
type Info struct {
	Action  string
	Section string // "general" or "explorer"
	Group   string // leader guide group of a general action; "" for the explorer
	Help    string // one line, next to the keys
	// Long explains the action in a sentence or two, at the bottom of the
	// command palette when the action is selected.
	Long     string
	defaults []string
}

// Actions lists every action in help-screen order, with today's shortcuts as
// defaults.
var Actions = []Info{
	{Action: NewNote, Section: "general", Group: GroupNote, Help: "New note (unsaved tab)",
		Long: "Opens an empty tab for a new note. It is written to disk on the first save, which asks for its folder.", defaults: []string{"ctrl+n", "space n", "space o n"}},
	{Action: Save, Section: "general", Group: GroupNote, Help: "Save (new notes: pick folder)",
		Long: "Writes the note in this tab to disk. A note that was never saved asks which folder it goes in.", defaults: []string{"ctrl+s", "space w"}},
	{Action: FindNote, Section: "general", Group: GroupGo, Help: "Find note (opens in this tab)",
		Long: "Searches the names and the text of every note in the vault; the note you pick opens in this tab.", defaults: []string{"ctrl+p", "\\", "space f", "space o f", "space o g"}},
	{Action: ToggleExplorer, Section: "general", Group: GroupView, Help: "Toggle explorer",
		Long: "Shows or hides the explorer, the tree of folders and notes on the left. On a narrow window it opens over the editor.", defaults: []string{"ctrl+b"}},
	{Action: FocusExplorer, Section: "general", Group: GroupGo, Help: "Focus / close explorer",
		Long: "Focuses the explorer, showing it if it was hidden, so its own keys work; when it already has the focus it closes and gives the focus back to the editor. With no note open it keeps the explorer, as there is nothing to go back to.", defaults: []string{"space e"}},
	{Action: NextPane, Section: "general", Group: GroupGo, Help: "Next pane",
		Long: "Moves the focus to the next pane: explorer, editor, preview.", defaults: []string{"tab"}},
	{Action: PrevPane, Section: "general", Group: GroupGo, Help: "Previous pane",
		Long: "Moves the focus to the previous pane: preview, editor, explorer.", defaults: []string{"shift+tab"}},
	{Action: PaneLeft, Section: "general", Group: GroupGo, Help: "Pane to the left",
		Long: "Moves the focus to the pane on the left of this one, and stops at the first.", defaults: []string{"ctrl+h", "ctrl+left"}},
	{Action: PaneRight, Section: "general", Group: GroupGo, Help: "Pane to the right",
		Long: "Moves the focus to the pane on the right of this one, and stops at the last.", defaults: []string{"ctrl+l", "ctrl+right"}},
	{Action: NextTab, Section: "general", Group: GroupGo, Help: "Next tab",
		Long: "Shows the next open note in the tab bar, wrapping around after the last.", defaults: []string{"]", "space tab", "L"}},
	{Action: PrevTab, Section: "general", Group: GroupGo, Help: "Previous tab",
		Long: "Shows the previous open note in the tab bar, wrapping around before the first.", defaults: []string{"[", "space shift+tab", "H"}},
	{Action: GoBack, Section: "general", Group: GroupGo, Help: "Back to the previous note of this tab",
		Long: "Returns this tab to the note it showed before gd, the finder or :e replaced it, at the line the cursor was on. Press it again to go further back.", defaults: []string{"ctrl+o"}},
	{Action: CloseTab, Section: "general", Group: GroupNote, Help: "Close tab",
		Long: "Closes the tab of this note. With unsaved changes it asks whether to save them first.", defaults: []string{"space x"}},
	{Action: ReadingView, Section: "general", Group: GroupView, Help: "Reading view: rendered and editable, full width",
		Long: "Shows only the note at full width, without line numbers, drawn as the side preview draws it. It stays editable: the block under the cursor shows its Markdown and every other block stays rendered, every editing key works, and what is clickable in the preview works here too. The same key goes back to the split; the editor view key shows all the Markdown with line numbers.", defaults: []string{"space p"}},
	{Action: EditorView, Section: "general", Group: GroupView, Help: "Editor view: Markdown full width",
		Long: "Shows only the Markdown editor at full width, with line numbers and without the side preview; the note edits as usual. The same key goes back to the split.", defaults: []string{"space r"}},
	{Action: ExportPDF, Section: "general", Group: GroupNote, Help: "Export note as PDF",
		Long: "Writes this note as a PDF file in the note's folder, with its blocks rendered.", defaults: []string{"space P"}},
	{Action: Themes, Section: "general", Group: GroupView, Help: "Themes",
		Long: "Opens the theme picker: the highlighted theme applies at once, Enter keeps it and Esc goes back.", defaults: []string{"space t"}},
	{Action: Graph, Section: "general", Group: GroupView, Help: "Graph of this note's links",
		Long: "Shows a map of the notes this note links to and the ones linking to it; move with h j k l and open one with Enter.", defaults: []string{"space g"}},
	{Action: GlobalGraph, Section: "general", Group: GroupView, Help: "Graph of every note",
		Long: "Shows a map of every note in the vault, grouped by top-level folder, with the links between them; h j k l move to the nearest note, n and N go round the selected note's links.", defaults: []string{"space G"}},
	{Action: Home, Section: "general", Group: GroupGo, Help: "Home screen",
		Long: "Goes back to the home screen, with recent notes and folders. Open tabs stay open.", defaults: []string{"space d"}},
	{Action: DailyNote, Section: "general", Group: GroupNote, Help: "Today's daily note",
		Long: "Opens today's note, Diario/YYYY-MM-DD.md (daily_dir in config.json names another folder). When it does not exist yet, a new note with that name starts in that folder; nothing is written until you save it. :daily does the same.", defaults: []string{"space o d"}},
	{Action: SyncNow, Section: "general", Group: GroupNote, Help: "Sync the vault with git now",
		Long: "Syncs the vault with its git remote right away, instead of waiting for the next automatic sync.", defaults: []string{"space s"}},
	{Action: PasteImage, Section: "general", Group: GroupNote, Help: "Paste an image from the clipboard",
		Long: "Saves the picture in the clipboard next to this note and links it on a line of its own. In insert mode ctrl+v does the same, and pastes text when the clipboard holds no picture.", defaults: []string{"space i", "space o p"}},
	{Action: ToggleCheckbox, Section: "general", Group: GroupNote, Help: "Toggle checkbox",
		Long: "Checks or unchecks the box of this line, or of every line of the selection: [ ] becomes [x] and [x] or [-] becomes [ ]; a line without a box gets one.", defaults: []string{"space o c"}},
	{Action: RemoveCheckbox, Section: "general", Group: GroupNote, Help: "Remove checkbox",
		Long: "Takes the box off this line, or off every line of the selection, and keeps the list item: - [x] task becomes - task.", defaults: []string{"space o C"}},
	{Action: NextLink, Section: "general", Group: GroupGo, Help: "Next [[link]]",
		Long: "Moves the cursor to the start of the next [[link]] of this note, past the last back to the first; links in code are skipped.", defaults: []string{"space o ]"}},
	{Action: PrevLink, Section: "general", Group: GroupGo, Help: "Previous [[link]]",
		Long: "Moves the cursor to the start of the previous [[link]] of this note, before the first round to the last; links in code are skipped.", defaults: []string{"space o ["}},
	{Action: Help, Section: "general", Group: GroupView, Help: "This help",
		Long: "Opens this list of every key and command. Type to filter it; Enter runs a command, Esc closes.", defaults: []string{"?", "space h", "space ?"}},
	{Action: Quit, Section: "general", Group: GroupNote, Help: "Quit (asks to save)",
		Long: "Closes Misoku. If a note has unsaved changes, it asks whether to save them first.", defaults: []string{"ctrl+c"}},

	{Action: ExplorerUp, Section: "explorer", Help: "Move up",
		Long: "In the explorer, moves the selection one row up.", defaults: []string{"up", "k"}},
	{Action: ExplorerDown, Section: "explorer", Help: "Move down",
		Long: "In the explorer, moves the selection one row down.", defaults: []string{"down", "j"}},
	{Action: ExplorerTop, Section: "explorer", Help: "First item",
		Long: "In the explorer, jumps to the first row of the tree.", defaults: []string{"g", "home"}},
	{Action: ExplorerBottom, Section: "explorer", Help: "Last item",
		Long: "In the explorer, jumps to the last row of the tree.", defaults: []string{"G", "end"}},
	{Action: ExplorerPageDown, Section: "explorer", Help: "Half page down",
		Long: "In the explorer, moves the selection half a screen down.", defaults: []string{"ctrl+d", "pgdown"}},
	{Action: ExplorerPageUp, Section: "explorer", Help: "Half page up",
		Long: "In the explorer, moves the selection half a screen up.", defaults: []string{"ctrl+u", "pgup"}},
	{Action: ExplorerOpen, Section: "explorer", Help: "Open note / expand",
		Long: "In the explorer, opens the selected note in the editor, or opens and closes the selected folder without opening any note.", defaults: []string{"enter"}},
	{Action: ExplorerPreview, Section: "explorer", Help: "Preview note (folder: open)",
		Long: "In the explorer, shows the selected note without leaving the explorer; on a folder, opens it.", defaults: []string{"p"}},
	{Action: ExplorerExpand, Section: "explorer", Help: "Expand / open",
		Long: "In the explorer, opens the selected folder, or the selected note.", defaults: []string{"l", "right"}},
	{Action: ExplorerCollapse, Section: "explorer", Help: "Collapse / go to parent",
		Long: "In the explorer, closes the selected folder, or moves to the folder that holds the selection.", defaults: []string{"h", "left"}},
	{Action: ExplorerToggleFold, Section: "explorer", Help: "Fold / unfold",
		Long: "In the explorer, opens the selected folder if it is closed and closes it if it is open.", defaults: []string{"z"}},
	{Action: ExplorerCollapseAll, Section: "explorer", Help: "Collapse all",
		Long: "In the explorer, closes every folder of the tree.", defaults: []string{"W"}},
	{Action: ExplorerNewNote, Section: "explorer", Help: "New note or folder (end with / for a folder)",
		Long: "In the explorer, asks for a name in the folder of the selection: test makes a note, test/ a folder and ideas/test both, after confirming. An empty name starts an Untitled note.", defaults: []string{"n", "%"}},
	{Action: ExplorerNewSubNote, Section: "explorer", Help: "New sub-note",
		Long: "In the explorer, starts a new note nested under the selected one.", defaults: []string{"s"}},
	{Action: ExplorerRename, Section: "explorer", Help: "Rename",
		Long: "In the explorer, renames the selected note or folder; type the new name and press Enter.", defaults: []string{"r", "e"}},
	{Action: ExplorerDelete, Section: "explorer", Help: "Delete (asks first)",
		Long: "In the explorer, deletes the selected note or folder after asking to confirm.", defaults: []string{"d", "x", "delete"}},
	{Action: ExplorerFind, Section: "explorer", Help: "Find note",
		Long: "In the explorer, opens the note finder, the same one as Find note.", defaults: []string{"\\", "f"}},
	{Action: ExplorerFileManager, Section: "explorer", Help: "Open the vault in Neovim's mini.files",
		Long: "In the explorer, hands the screen to Neovim's mini.files on the vault folder; closing it (q) returns to Misoku, which re-reads the folder at once.", defaults: []string{"M"}},
	{Action: ExplorerEditFolder, Section: "explorer", Help: "Edit the folder like text (Oil)",
		Long: "In the explorer, shows the selected folder as editable lines, like Oil: type a line to create with o/O (a trailing / makes a folder), edit a name to rename, dd or erasing the whole line with Backspace to delete, dd then p in another folder (h/l) to move, u to undo. Enter while typing only opens a new line below; nothing touches the disk until you confirm. With edits pending, Enter, =, :w or Ctrl+S (also while typing) lists the actions and applies them after Yes; No returns to the lines with the edits kept. The editor stays open on the same folder, re-read from disk, with the cursor on the new entry. With nothing pending, Enter enters a folder or opens a note. Esc or q leaves; with edits it shows the same confirm: Yes applies and closes, No drops them, Cancel keeps editing.", defaults: []string{"m"}},
	{Action: ExplorerInsert, Section: "explorer", Help: "Start typing in the folder editor",
		Long: "In the explorer, opens the folder editor (m) on the folder holding the selection, with the cursor on it, and passes it the key, as mini.files does: i/I and a/A edit the selected name, o/O type a new line below or above it.", defaults: []string{"i", "I", "a", "A", "o", "O"}},
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

// LongText returns the longer explanation of action the command palette
// shows; "" for an unknown action.
func LongText(action string) string {
	for _, a := range Actions {
		if a.Action == action {
			return a.Long
		}
	}
	return ""
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
		raw := user[name]
		if now, ok := renamed[name]; ok {
			if _, given := user[now]; given {
				continue // the new name wins over the old one
			}
			name = now
		}
		if !known[name] {
			warnings = append(warnings, fmt.Sprintf("keys: unknown action %q", name))
			continue
		}
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

// renamed maps action names of older config files to today's actions.
// reading_mode was the full-width rendered view, now the reading view;
// toggle_preview hid the side preview and left the editor alone, now the
// editor view.
var renamed = map[string]string{"reading_mode": ReadingView, "toggle_preview": EditorView}

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
	return append(append(out, m.prefixClashes()...), m.shadowed()...)
}

// prefixClashes reports a leader sequence that is the start of another one
// of the same section ("space o" and "space o c"): the shorter one runs at
// once, so the longer one can never be typed.
func (m Map) prefixClashes() []string {
	var out []string
	for _, a := range Actions {
		for _, short := range m.keys[a.Action] {
			for _, b := range Actions {
				if b.Section != a.Section {
					continue
				}
				for _, long := range m.keys[b.Action] {
					if strings.HasPrefix(long, short+" ") {
						out = append(out, fmt.Sprintf("keys: %q on %s is the start of %q on %s, so %s can never be typed; unbind one of them", short, a.Action, long, b.Action, b.Action))
					}
				}
			}
		}
	}
	return out
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
	case 2, 3:
		if token(fields[0]) != leader {
			return "", fmt.Errorf("%q: a shortcut of several keys must start with space", k)
		}
		for i := range fields {
			fields[i] = token(fields[i])
		}
		return strings.Join(fields, " "), nil
	}
	return "", fmt.Errorf("%q: use one key, or space plus one or two keys", k)
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
func (m Map) Leader(action, s string) bool { return m.LeaderSeq(action, []string{s}) }

// LeaderSeq reports whether keys, pressed in order after the leader, trigger
// action: ["o", "c"] for "space o c".
func (m Map) LeaderSeq(action string, keys []string) bool {
	want := sequence(keys)
	for _, k := range m.keys[action] {
		if k == want {
			return true
		}
	}
	return false
}

// LeaderPrefix reports whether keys, pressed after the leader, are the start
// of a longer leader sequence of one of actions, so the sequence waits for
// the next key.
func (m Map) LeaderPrefix(actions []string, keys []string) bool {
	start := sequence(keys) + " "
	for _, a := range actions {
		for _, k := range m.keys[a] {
			if strings.HasPrefix(k, start) {
				return true
			}
		}
	}
	return false
}

// sequence spells keys, pressed after the leader, as a configured key.
func sequence(keys []string) string {
	parts := []string{leader}
	for _, k := range keys {
		parts = append(parts, fromMsg(k))
	}
	return strings.Join(parts, " ")
}

// after returns the keys of k that follow the leader and prefix, or false
// when k does not start with them.
func after(k string, prefix []string) ([]string, bool) {
	fields := strings.Fields(k)
	if len(fields) <= len(prefix)+1 || fields[0] != leader {
		return nil, false
	}
	for i, p := range prefix {
		if fields[i+1] != fromMsg(p) {
			return nil, false
		}
	}
	return fields[len(prefix)+1:], true
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

// LeaderLabel renders only the keys that follow the leader for action, for
// the leader guide: "h / ?" for "space h" and "space ?". It is "" when
// action has no key of Space and one more key.
func (m Map) LeaderLabel(action string) string { return m.LeaderLabelAfter(action, nil) }

// LeaderLabelAfter is LeaderLabel for the guide's page of a group: the last
// keys of action's sequences that start with Space and prefix, "f / g" for
// "space o f" and "space o g" after ["o"].
func (m Map) LeaderLabelAfter(action string, prefix []string) string {
	var parts []string
	for _, k := range m.keys[action] {
		if rest, ok := after(k, prefix); ok && len(rest) == 1 {
			parts = append(parts, tokenLabel(rest[0], true))
		}
	}
	return strings.Join(parts, " / ")
}

// Subgroup is a key of the leader guide that leads to more keys: Key is its
// label and Name what the group holds ("o", "misoku").
type Subgroup struct{ Key, Name string }

// LeaderSubgroups lists, once each and in Actions order, the keys after
// Space and prefix that start a longer sequence of one of actions.
func (m Map) LeaderSubgroups(actions []string, prefix []string) []Subgroup {
	var out []Subgroup
	seen := map[string]bool{}
	for _, a := range actions {
		for _, k := range m.keys[a] {
			rest, ok := after(k, prefix)
			if !ok || len(rest) < 2 || seen[rest[0]] {
				continue
			}
			seen[rest[0]] = true
			name := GroupTitle(slices.Concat(prefix, rest[:1]))
			if name == "" {
				name = "more"
			}
			out = append(out, Subgroup{Key: tokenLabel(rest[0], true), Name: name})
		}
	}
	return out
}

// GroupTitle names the leader group keys lead to, for the guide's title:
// "misoku" for ["o"], "" for none.
func GroupTitle(keys []string) string {
	if len(keys) == 1 {
		return prefixNames[keys[0]]
	}
	return ""
}

// SequenceLabel renders keys typed after the leader as the guide's title
// shows them: "SPC o".
func SequenceLabel(keys []string) string { return keyLabel(sequence(keys)) }

// keyLabel renders one key or leader sequence: "SPC Shift+P", "Ctrl+E".
func keyLabel(k string) string {
	var toks []string
	for i, t := range strings.Fields(k) {
		toks = append(toks, tokenLabel(t, i > 0))
	}
	return strings.Join(toks, " ")
}

// tokenLabel renders one key. After SPC a lone capital is easy to misread as
// the lower-case key, so spell out Shift ("SPC Shift+P"). A capital on its
// own ("G") stays as is.
func tokenLabel(t string, afterLeader bool) string {
	l := label(t)
	if afterLeader && len([]rune(t)) == 1 && unicode.IsUpper([]rune(t)[0]) {
		l = "Shift+" + l
	}
	return l
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
