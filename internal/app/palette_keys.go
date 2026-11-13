package app

import (
	"slices"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/keymap"

	tea "github.com/charmbracelet/bubbletea"
)

// paletteEntry is one row of the command palette.
type paletteEntry struct {
	group paletteGroup // the heading it is listed under
	key   string       // the whole key, or the command as typed
	title string       // one line, next to the key
	long  string       // the explanation at the bottom of the palette
	// run performs the entry on Enter; nil for a key that only works in
	// its own place (the editor, the explorer, the graph), which the
	// palette just explains.
	run func(m *Model) tea.Cmd
	// needsNote: opened from the home screen, the entry would act on a
	// note, so it is only explained, as its key does nothing there.
	needsNote bool
}

// fixedKey is a key that is not configurable: Vim editing, the preview,
// the graph and the mouse. Its words are read when the palette opens, in
// the language then in use.
type fixedKey struct {
	key         func() string
	title, long i18n.Key
}

// fixed is a key named the same in every language.
func fixed(key string) func() string { return func() string { return key } }

// Palette groups, in display order: an icon and the group's name.
type paletteGroup struct {
	icon string
	name i18n.Key
}

func (g paletteGroup) String() string { return g.icon + i18n.T(g.name) }

var (
	groupExport   = paletteGroup{"󰈦 ", i18n.AppPalGroupExport}
	groupGeneral  = paletteGroup{"󰕭 ", i18n.AppPalGroupGeneral}
	groupExplorer = paletteGroup{"󰙅 ", i18n.AppPalGroupExplorer}
	groupEditor   = paletteGroup{" ", i18n.AppPalGroupEditor}
	groupCommands = paletteGroup{" ", i18n.AppPalGroupCommands}
	groupPreview  = paletteGroup{" ", i18n.AppPalGroupPreview}
	groupGraph    = paletteGroup{"󰘚 ", i18n.AppPalGroupGraph}
	groupGlobal   = paletteGroup{"󰘚 ", i18n.AppPalGroupGlobal}
	groupMouse    = paletteGroup{"󰍽 ", i18n.AppPalGroupMouse}
)

var editorKeys = []fixedKey{
	{fixed("i a A I"), i18n.AppPal01, i18n.AppPal01Long},
	{fixed("Tab/S-Tab"), i18n.AppPal02, i18n.AppPal02Long},
	{fixed("o / O"), i18n.AppPal03, i18n.AppPal03Long},
	{fixed("h j k l"), i18n.AppPal04, i18n.AppPal04Long},
	{fixed("w / b"), i18n.AppPal05, i18n.AppPal05Long},
	{fixed("0 / $"), i18n.AppPal06, i18n.AppPal06Long},
	{fixed("gg / G"), i18n.AppPal07, i18n.AppPal07Long},
	{fixed("Ctrl+D/U"), i18n.AppPal08, i18n.AppPal08Long},
	{fixed("x / dd"), i18n.AppPal09, i18n.AppPal09Long},
	{fixed("yy / p"), i18n.AppPal10, i18n.AppPal10Long},
	{fixed("v / V"), i18n.AppPal11, i18n.AppPal11Long},
	{fixed("u"), i18n.AppPal12, i18n.AppPal12Long},
	{fixed("Ctrl+R"), i18n.AppPal13, i18n.AppPal13Long},
	{fixed("/ (insert)"), i18n.AppPal14, i18n.AppPal14Long},
	{fixed("[[ (insert)"), i18n.AppPal15, i18n.AppPal15Long},
	{fixed("gd / gx"), i18n.AppPal16, i18n.AppPal16Long},
	{fixed("Enter"), i18n.AppPal17, i18n.AppPal17Long},
	{fixed(":"), i18n.AppPal18, i18n.AppPal18Long},
	{fixed("/"), i18n.AppPal19, i18n.AppPal19Long},
	{fixed("n / N"), i18n.AppPal20, i18n.AppPal20Long},
}

var previewKeys = []fixedKey{
	{fixed("j / k"), i18n.AppPal21, i18n.AppPal21Long},
	{fixed("d / u"), i18n.AppPal22, i18n.AppPal22Long},
	{fixed("Space / PgDn"), i18n.AppPal23, i18n.AppPal23Long},
	{fixed("g / G"), i18n.AppPal24, i18n.AppPal24Long},
	{fixed("< / >"), i18n.AppPal25, i18n.AppPal25Long},
	{fixed("T"), i18n.AppPal26, i18n.AppPal26Long},
	{fixed("zM / zR"), i18n.AppPal27, i18n.AppPal27Long},
}

var graphKeys = []fixedKey{
	{fixed("h j k l"), i18n.AppPal28, i18n.AppPal28Long},
	{fixed("Enter"), i18n.AppPal29, i18n.AppPal29Long},
	{fixed("+ / -"), i18n.AppPal30, i18n.AppPal30Long},
	{fixed("0"), i18n.AppPal31, i18n.AppPal31Long},
	{fixed("Esc"), i18n.AppPal32, i18n.AppPal32Long},
}

var globalGraphKeys = []fixedKey{
	{fixed("h j k l"), i18n.AppPal33, i18n.AppPal33Long},
	{fixed("n / N"), i18n.AppPal34, i18n.AppPal34Long},
	{fixed("Enter"), i18n.AppPal35, i18n.AppPal35Long},
	{fixed("+ / -"), i18n.AppPal36, i18n.AppPal36Long},
	{fixed("0"), i18n.AppPal37, i18n.AppPal37Long},
	{fixed("Esc"), i18n.AppPal38, i18n.AppPal38Long},
	{fixed("/"), i18n.AppPal39, i18n.AppPal39Long},
	{fixed("o"), i18n.AppPal40, i18n.AppPal40Long},
	{fixed("u"), i18n.AppPal41, i18n.AppPal41Long},
}

var mouseKeys = []fixedKey{
	{func() string { return i18n.T(i18n.AppPalKeyClick) }, i18n.AppPal42, i18n.AppPal42Long},
	{func() string { return i18n.T(i18n.AppPalKeyWheel) }, i18n.AppPal43, i18n.AppPal43Long},
	{func() string { return i18n.T(i18n.AppPalKeyDrag) }, i18n.AppPal44, i18n.AppPal44Long},
}

// paletteEntries lists every row of the palette, in display order: how to
// export first (issue #4), then the configurable actions with the keys in
// use, the fixed editor keys, the editor's commands, and the keys of the
// preview, the graph, the global graph and the mouse.
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
	out = append(out, paletteEntry{group: groupExport, key: "misoku export", title: i18n.T(i18n.AppPalShell),
		long: i18n.T(i18n.AppPalShellLong)})

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
	out = append(out, fixedEntries(groupGlobal, globalGraphKeys)...)
	return append(out, fixedEntries(groupMouse, mouseKeys)...)
}

// actionEntry lists a keymap action. General actions run on Enter, as
// their key would from where the palette was opened: from home, only the
// homeActions, the others need an open note. The explorer's act on its
// selection, so they are only explained, and so is the help itself.
func actionEntry(group paletteGroup, label, action string, home bool) paletteEntry {
	info := keymap.Actions[slices.IndexFunc(keymap.Actions, func(i keymap.Info) bool { return i.Action == action })]
	e := paletteEntry{group: group, key: label, title: keymap.HelpText(action), long: keymap.LongText(action)}
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
func commandEntry(group paletteGroup, c content.Command, home bool) paletteEntry {
	e := paletteEntry{group: group, key: c.Usage, title: c.HelpText(), long: c.LongText()}
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

func fixedEntries(group paletteGroup, keys []fixedKey) []paletteEntry {
	out := make([]paletteEntry, len(keys))
	for i, k := range keys {
		out[i] = paletteEntry{group: group, key: k.key(), title: i18n.T(k.title), long: i18n.T(k.long)}
	}
	return out
}
