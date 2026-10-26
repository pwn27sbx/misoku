package content

import "slices"

// Command is one ex command of the editor's command line, as the command
// palette lists it.
type Command struct {
	// Usage is how it is typed, with its argument: ":w name".
	Usage string
	// Names are every spelling the command line accepts for it.
	Names []string
	// Help is one line, next to the usage; Long explains it at the bottom
	// of the palette.
	Help, Long string
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
	{Usage: ":w", Names: []string{"w", "write"}, Help: "Save", Run: "w",
		Long: "Writes the note to disk. A note that was never saved asks which folder it goes in."},
	{Usage: ":w name", Names: []string{"sav", "saveas"}, Help: "Save as name.md",
		Long: "Saves the note under a new name, name.md; a path such as folder/name puts it in that folder of the vault. :sav and :saveas do the same."},
	{Usage: ":wq / :x", Names: []string{"wq", "x", "xit", "wqa", "xa"}, Help: "Save and quit", Run: "wq",
		Long: "Saves this note, then closes Misoku. Another note with unsaved changes still asks first."},
	{Usage: ":q / :q!", Names: []string{"q", "quit", "qa", "qall", "q!", "quit!", "qa!", "qall!"}, Help: "Quit / discard", Run: "q", NoNote: true,
		Long: "Closes Misoku. :q asks before losing unsaved changes; :q! throws them away."},
	{Usage: ":bd", Names: []string{"bd", "bdelete", "bw", "close", "bd!", "bdelete!", "bw!"}, Help: "Close tab", Run: "bd",
		Long: "Closes this tab. With unsaved changes it asks first; :bd! closes it without saving."},
	{Usage: ":enew", Names: []string{"enew", "new", "e", "edit"}, Help: "New note", Run: "enew", NoNote: true,
		Long: "Opens an empty tab for a new note, as the new note key does."},
	{Usage: ":export [file]", Names: []string{"export", "pdf"}, Help: "Export to a file", Run: "export",
		Long: "Writes this note as a PDF. Without a file it goes in the note's folder, named after the note; a file or folder picks where. :pdf does the same."},
	{Usage: ":graph", Names: []string{"graph"}, Help: "Graph of this note's links", Run: "graph", NoNote: true,
		Long: "Shows the map of the notes this note links to and the ones linking to it."},
	{Usage: ":graph all", Names: []string{"graph"}, Help: "Graph of every note", Run: "graph all", NoNote: true,
		Long: "Shows the map of every note in the vault and the links between them, notes without links and links to missing notes included."},
	{Usage: ":messages", Names: []string{"messages", "mes"}, Help: "Last messages of the bar", Run: "messages", NoNote: true,
		Long: "Lists the last 100 messages the status bar showed, with their time, so an error that went away can be read again."},
	{Usage: ":colo [name]", Names: []string{"colo", "colorscheme", "theme"}, Help: "Switch theme", Run: "colo", NoNote: true,
		Long: "With a name, applies and saves that theme; without one, opens the theme picker."},
	{Usage: ":settings", Names: []string{"settings"}, Help: "Settings menu", Run: "settings", NoNote: true,
		Long: "Opens the settings menu: accordion folders, the explorer's hide width, heading icons and bullets, and the dialog border. A change applies and is saved at once."},
	{Usage: ":sync", Names: []string{"sync"}, Help: "Sync the vault with git", Run: "sync", NoNote: true,
		Long: "Syncs the vault with its git remote now, like the sync key."},
	{Usage: ":sync status", Names: []string{"sync"}, Help: "Sync state, conflict copies", Run: "sync status", NoNote: true,
		Long: "Shows the state of the sync and any conflict copies left to look at."},
	{Usage: ":sync setup", Names: []string{"sync"}, Help: "Connect the vault to GitHub", Run: "sync setup", NoNote: true,
		Long: "Walks through connecting this vault to a GitHub repository, so it syncs on its own."},
	{Usage: ":addrow", Names: []string{"addrow", "tr"}, Help: "Table row",
		Long: "With the cursor in a Markdown table, adds a row below the cursor. :tr is the short form."},
	{Usage: ":addcol", Names: []string{"addcol", "tc"}, Help: "Table column",
		Long: "With the cursor in a Markdown table, adds a column after the cursor. :tc is the short form."},
	{Usage: ":delrow / :delcol", Names: []string{"delrow", "delcol"}, Help: "Delete table row / column",
		Long: "With the cursor in a Markdown table, deletes the row or the column under the cursor."},
	{Usage: ":tablefmt", Names: []string{"tablefmt", "tf"}, Help: "Align the table",
		Long: "With the cursor in a Markdown table, lines up its columns. :tf is the short form."},
}

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
