package content

import (
	"slices"

	"github.com/pwn27sbx/misoku/internal/i18n"
)

// Command is one ex command of the editor's command line, as the command
// palette lists it.
type Command struct {
	// Usage is how it is typed, with its argument: ":w name".
	Usage string
	// Names are every spelling the command line accepts for it.
	Names []string
	// id names the command's texts in the i18n catalog
	// (i18n.CommandHelp, i18n.CommandLong).
	id string
	// Run is the line the palette runs on Enter, without ":"; "" when the
	// command needs an argument or the cursor in a table.
	Run string
	// NoNote: it works without an open note, so the palette runs it from
	// the home screen too.
	NoNote bool
}

// Commands is the catalog of ex commands, in help order. It joins
// parseVimCommand and the table commands: a name missing here is not a
// command.
var Commands = []Command{
	{id: "w", Usage: ":w", Names: []string{"w", "write"}, Run: "w"},
	{id: "saveas", Usage: ":w name", Names: []string{"sav", "saveas"}},
	{id: "wq", Usage: ":wq / :x", Names: []string{"wq", "x", "xit", "wqa", "xa"}, Run: "wq"},
	{id: "q", Usage: ":q / :q!", Names: []string{"q", "quit", "qa", "qall", "q!", "quit!", "qa!", "qall!"}, Run: "q", NoNote: true},
	{id: "bd", Usage: ":bd", Names: []string{"bd", "bdelete", "bw", "close", "bd!", "bdelete!", "bw!"}, Run: "bd"},
	{id: "enew", Usage: ":enew", Names: []string{"enew", "new", "e", "edit"}, Run: "enew", NoNote: true},
	{id: "e", Usage: ":e name", Names: []string{"e", "edit"}},
	{id: "export", Usage: ":export [file]", Names: []string{"export", "pdf"}, Run: "export"},
	{id: "graph", Usage: ":graph", Names: []string{"graph"}, Run: "graph", NoNote: true},
	{id: "graph_all", Usage: ":graph all", Names: []string{"graph"}, Run: "graph all", NoNote: true},
	{id: "daily", Usage: ":daily", Names: []string{"daily"}, Run: "daily", NoNote: true},
	{id: "history", Usage: ":history", Names: []string{"history"}, Run: "history"},
	{id: "tags", Usage: ":tags", Names: []string{"tags"}, Run: "tags", NoNote: true},
	{id: "status", Usage: ":status [value]", Names: []string{"status"}, Run: "status", NoNote: true},
	{id: "messages", Usage: ":messages", Names: []string{"messages", "mes"}, Run: "messages", NoNote: true},
	{id: "colo", Usage: ":colo [name]", Names: []string{"colo", "colorscheme", "theme"}, Run: "colo", NoNote: true},
	{id: "settings", Usage: ":settings", Names: []string{"settings"}, Run: "settings", NoNote: true},
	{id: "sync", Usage: ":sync", Names: []string{"sync"}, Run: "sync", NoNote: true},
	{id: "sync_status", Usage: ":sync status", Names: []string{"sync"}, Run: "sync status", NoNote: true},
	{id: "sync_setup", Usage: ":sync setup", Names: []string{"sync"}, Run: "sync setup", NoNote: true},
	{id: "addrow", Usage: ":addrow", Names: []string{"addrow", "tr"}},
	{id: "addcol", Usage: ":addcol", Names: []string{"addcol", "tc"}},
	{id: "delrow", Usage: ":delrow / :delcol", Names: []string{"delrow", "delcol"}},
	{id: "tablefmt", Usage: ":tablefmt", Names: []string{"tablefmt", "tf"}},
}

// HelpText is the command's one line, next to the usage, in the interface
// language.
func (c Command) HelpText() string { return i18n.T(i18n.CommandHelp(c.id)) }

// LongText explains the command at the bottom of the palette, in the
// interface language.
func (c Command) LongText() string { return i18n.T(i18n.CommandLong(c.id)) }

// isCommand reports whether name is in the catalog.
func isCommand(name string) bool {
	for _, c := range Commands {
		if slices.Contains(c.Names, name) {
			return true
		}
	}
	return false
}

// RunCommand turns a command line, without ":", into the message the editor
// sends when it is typed: nil for an empty line, a StatusMsg error for an
// unknown command. text is the note, for :w. Table commands need the
// editor's cursor and are not run here.
func RunCommand(line, text string) any { return parseVimCommand(line, text) }

// IsTableCommand reports whether name is one of the table commands, which
// the editor runs on the table under its cursor.
func IsTableCommand(name string) bool {
	_, ok := tableCommands[name]
	return ok
}
