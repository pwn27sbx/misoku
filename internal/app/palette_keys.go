package app

import (
	"slices"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
)

// paletteEntry is one row of the command palette.
type paletteEntry struct {
	group string // the heading it is listed under
	key   string // the whole key, or the command as typed
	title string // one line, next to the key
	long  string // the explanation at the bottom of the palette
	// run performs the entry on Enter; nil for a key that only works in
	// its own place (the editor, the explorer, the graph), which the
	// palette just explains.
	run func(m *Model) tea.Cmd
	// needsNote: opened from the home screen, the entry would act on a
	// note, so it is only explained, as its key does nothing there.
	needsNote bool
}

// fixedKey is a key that is not configurable: Vim editing, the preview,
// the graph and the mouse.
type fixedKey struct{ key, title, long string }

// Palette groups, in display order.
const (
	groupExport   = "󰈦 EXPORT"
	groupGeneral  = "󰕭 GENERAL"
	groupExplorer = "󰙅 EXPLORER"
	groupEditor   = " EDITOR · NORMAL"
	groupCommands = " COMMANDS"
	groupPreview  = " PREVIEW"
	groupGraph    = "󰘚 GRAPH"
	groupMouse    = "󰍽 MOUSE"
)

var editorKeys = []fixedKey{
	{"i a A I", "Insert mode", "Starts typing: before the cursor (i), after it (a), at the end of the line (A) or at its start (I). Esc goes back to NORMAL."},
	{"Tab/S-Tab", "Indent / outdent (insert)", "In INSERT mode, Tab indents the line or the list item and Shift+Tab takes the indent back; in a table they move between cells."},
	{"o / O", "New line below / above", "Opens a new line below (o) or above (O) the cursor and starts typing on it."},
	{"h j k l", "Move", "Moves the cursor left, down, up and right, as the arrow keys do."},
	{"w / b", "Next / previous word", "Moves the cursor to the start of the next word (w) or of the previous one (b)."},
	{"0 / $", "Line start / end", "Moves the cursor to the start (0) or the end ($) of the line."},
	{"gg / G", "Top / bottom", "Moves the cursor to the first (gg) or the last (G) line of the note."},
	{"Ctrl+D/U", "Half page down / up", "Scrolls the editor half a screen down (Ctrl+D) or up (Ctrl+U)."},
	{"x / dd", "Delete char / line", "Deletes the character under the cursor (x) or the whole line (dd), which can then be pasted with p."},
	{"yy / p", "Copy line / paste", "Copies the line (yy) and pastes what was copied or deleted below the cursor (p)."},
	{"v / V", "Select, then y copy, d cut", "Starts selecting by character (v) or by line (V); then y copies the selection and d cuts it."},
	{"u", "Undo", "Undoes the last change: a whole stay in INSERT mode, or one NORMAL command such as dd, p or x."},
	{"Ctrl+R", "Redo", "Redoes the last change that u undid."},
	{"/ (insert)", "Block menu: headings, lists, blocks", "Typed in INSERT mode at the start of a line or after a space, / opens the block menu: headings, lists, tables, callouts, media and more."},
	{"[[ (insert)", "Link to a note", "Typed in INSERT mode, [[ opens the note finder to pick the note to link to."},
	{"gd / gx", "Follow the link under the cursor", "Opens the note or the web page the link under the cursor points to."},
	{":", "Command line", "Opens the command line at the bottom of the editor for the commands below; Enter runs it, Esc leaves it."},
}

var previewKeys = []fixedKey{
	{"j / k", "Scroll a line", "With the focus on the preview, scrolls it one line down (j) or up (k)."},
	{"d / u", "Half page down / up", "With the focus on the preview, scrolls it half a screen down (d) or up (u); Ctrl+D and Ctrl+U too."},
	{"Space / PgDn", "Page down", "With the focus on the preview, scrolls it a whole screen down; PgUp goes back up."},
	{"g / G", "Top / bottom", "With the focus on the preview, jumps to its top (g) or its bottom (G)."},
	{"< / >", "Calendar month back / forward", "Moves the calendar blocks of the note one month back (<, or H) or forward (>, or L)."},
	{"T", "Calendar: this month", "Brings the calendar blocks of the note back to the current month."},
	{"zM / zR", "Fold / unfold all", "Folds (zM) or unfolds (zR) every foldable block of the preview at once."},
}

var graphKeys = []fixedKey{
	{"h j k l", "Move from note to note", "In the graph, j and k move between notes at the same level; h and l move one column left or right, to the parent or to a child."},
	{"Enter", "Open the selected note", "In the graph, opens the selected note in the editor."},
	{"+ / -", "Zoom in / out", "In the graph, shows the map closer (+) or further away (-)."},
	{"0", "Centre the graph", "In the graph, puts the selected note back in the middle of the map."},
	{"Esc", "Back", "In the graph, goes back to the note shown before, as the back button does."},
}

var mouseKeys = []fixedKey{
	{"Click", "Tabs, tree, cursor", "A click picks a tab, a row of the explorer, a place in the editor, or a checkbox or a link in the preview."},
	{"Wheel", "Scroll any pane", "The wheel scrolls the pane under the mouse, whichever has the focus."},
	{"Drag", "Resize the panes", "Dragging a divider between two panes makes one wider and the other narrower."},
}

// paletteEntries lists every row of the palette, in display order: how to
// export first (issue #4), then the configurable actions with the keys in
// use, the fixed editor keys, the editor's commands, and the keys of the
// preview, the graph and the mouse.
func (m *Model) paletteEntries() []paletteEntry {
	var out []paletteEntry
	commands := slices.Clone(content.Commands)
	home := m.viewMode == viewModeDashboard

	// Export: the key (when bound), the command and the CLI.
	if label := m.keys.Label(keymap.ExportPDF); label != "" {
		out = append(out, actionEntry(groupExport, label, keymap.ExportPDF, home))
	}
	for i, c := range commands {
		if slices.Contains(c.Names, "export") {
			out = append(out, commandEntry(groupExport, c, home))
			commands = slices.Delete(commands, i, i+1)
			break
		}
	}
	out = append(out, paletteEntry{group: groupExport, key: "misoku export", title: "From the shell",
		long: "Run misoku export note.md in a terminal to write the PDF without opening Misoku."})

	for _, a := range keymap.Actions {
		label := m.keys.Label(a.Action)
		if label == "" || a.Action == keymap.ExportPDF {
			continue
		}
		group := groupGeneral
		if a.Section == "explorer" {
			group = groupExplorer
		}
		out = append(out, actionEntry(group, label, a.Action, home))
	}
	out = append(out, fixedEntries(groupEditor, editorKeys)...)
	for _, c := range commands {
		out = append(out, commandEntry(groupCommands, c, home))
	}
	out = append(out, fixedEntries(groupPreview, previewKeys)...)
	out = append(out, fixedEntries(groupGraph, graphKeys)...)
	return append(out, fixedEntries(groupMouse, mouseKeys)...)
}

// actionEntry lists a keymap action. General actions run on Enter, as
// their key would from where the palette was opened: from home, only the
// homeActions, the others need an open note. The explorer's act on its
// selection, so they are only explained, and so is the help itself.
func actionEntry(group, label, action string, home bool) paletteEntry {
	info := keymap.Actions[slices.IndexFunc(keymap.Actions, func(i keymap.Info) bool { return i.Action == action })]
	e := paletteEntry{group: group, key: label, title: info.Help, long: info.Long}
	switch {
	case info.Section != "general" || action == keymap.Help:
	case !home:
		e.run = func(m *Model) tea.Cmd { return m.runAction(action) }
	case slices.Contains(homeActions, action):
		e.run = func(m *Model) tea.Cmd { return m.runHomeAction(action) }
	default:
		e.needsNote = true
	}
	return e
}

// commandEntry lists an ex command. One that needs no argument and no
// table runs on Enter, exactly as if it was typed on the command line;
// from home, only one that needs no note (Command.NoNote).
func commandEntry(group string, c content.Command, home bool) paletteEntry {
	e := paletteEntry{group: group, key: c.Usage, title: c.Help, long: c.Long}
	if home && c.Run != "" && !c.NoNote {
		e.needsNote = true
		return e
	}
	if c.Run != "" {
		line := c.Run
		e.run = func(m *Model) tea.Cmd {
			msg := content.RunCommand(line, m.content.Value())
			return func() tea.Msg { return msg }
		}
	}
	return e
}

func fixedEntries(group string, keys []fixedKey) []paletteEntry {
	out := make([]paletteEntry, len(keys))
	for i, k := range keys {
		out[i] = paletteEntry{group: group, key: k.key, title: k.title, long: k.long}
	}
	return out
}
