package content_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
)

// notACommand reports whether msg is the editor's "Not an editor command".
func notACommand(msg any) bool {
	s, ok := msg.(core.StatusMsg)
	return ok && strings.HasPrefix(s.Text, "E492")
}

// The catalog is what the command palette lists: every entry is documented
// and every name it gives is one the command line accepts.
func TestCommandCatalogNamesAreAcceptedByTheCommandLine(t *testing.T) {
	if len(content.Commands) == 0 {
		t.Fatal("the catalog is empty")
	}
	for _, c := range content.Commands {
		if !strings.HasPrefix(c.Usage, ":") || c.Help == "" || c.Long == "" || len(c.Names) == 0 {
			t.Errorf("%q: needs a :usage, Help, Long and names", c.Usage)
		}
		for _, n := range c.Names {
			if content.IsTableCommand(n) {
				continue
			}
			if msg := content.RunCommand(n, ""); notACommand(msg) {
				t.Errorf("%q lists %q, but the command line rejects it: %v", c.Usage, n, msg)
			}
		}
	}
}

// The catalog joins parseVimCommand's switch and the table commands: a
// name missing from it is not a command, and every table command is in it.
func TestCommandCatalogCoversTheTableCommands(t *testing.T) {
	listed := map[string]bool{}
	for _, c := range content.Commands {
		for _, n := range c.Names {
			listed[n] = true
		}
	}
	for _, n := range []string{"addrow", "tr", "addcol", "tc", "delrow", "delcol", "tablefmt", "tf"} {
		if !content.IsTableCommand(n) || !listed[n] {
			t.Errorf("table command %q must be in the catalog", n)
		}
	}
	for _, n := range []string{"w", "q", "export", "graph", "messages", "sync", "colo", "settings", "enew", "bd"} {
		if !listed[n] {
			t.Errorf("%q must be in the catalog", n)
		}
	}
	if !notACommand(content.RunCommand("nosuch", "")) {
		t.Error("a name the catalog does not list is not a command")
	}
}

// Run is the line the palette runs on Enter: it gives the command's message,
// never an error. Commands that need an argument or a table have none.
func TestCommandCatalogRunLinesGiveTheirMessage(t *testing.T) {
	runs := map[string]any{}
	for _, c := range content.Commands {
		if c.Run == "" {
			continue
		}
		msg := content.RunCommand(c.Run, "text")
		if msg == nil || notACommand(msg) {
			t.Errorf("%q: Run %q gives %v", c.Usage, c.Run, msg)
		}
		if s, ok := msg.(core.StatusMsg); ok && s.Error {
			t.Errorf("%q: Run %q gives an error: %v", c.Usage, c.Run, s.Text)
		}
		runs[c.Usage] = msg
	}
	if _, ok := runs[":settings"].(core.SettingsMsg); !ok {
		t.Errorf(":settings must run, got %v", runs[":settings"])
	}
	if _, ok := runs[":messages"].(core.MessagesMsg); !ok {
		t.Errorf(":messages must run, got %v", runs[":messages"])
	}
	for _, c := range content.Commands {
		if content.IsTableCommand(c.Names[0]) && c.Run != "" {
			t.Errorf("%q needs the cursor in a table: the palette cannot run it", c.Usage)
		}
	}
}

// NoNote marks what the palette runs from the home screen: only commands it
// can run at all, and never one that acts on the open note.
func TestCommandCatalogNoNoteCommandsRunAndLeaveTheNoteAlone(t *testing.T) {
	no := map[string]bool{}
	for _, c := range content.Commands {
		if c.NoNote && c.Run == "" {
			t.Errorf("%q works without a note but has no Run line", c.Usage)
		}
		no[c.Usage] = c.NoNote
	}
	for _, u := range []string{":settings", ":messages", ":q / :q!", ":colo [name]", ":graph", ":graph all", ":enew", ":sync", ":sync status", ":sync setup"} {
		if !no[u] {
			t.Errorf("%s needs no note", u)
		}
	}
	for _, u := range []string{":w", ":w name", ":wq / :x", ":bd", ":export [file]"} {
		if no[u] {
			t.Errorf("%s acts on the open note", u)
		}
	}
}

// ":graph" opens the local graph, ":graph all" the global one; any other
// argument is an error, like :sync's.
func TestGraphCommandTakesAllOrNothing(t *testing.T) {
	if _, ok := content.RunCommand("graph", "").(core.GraphMsg); !ok {
		t.Errorf(":graph = %#v, want GraphMsg", content.RunCommand("graph", ""))
	}
	if _, ok := content.RunCommand("graph all", "").(core.GlobalGraphMsg); !ok {
		t.Errorf(":graph all = %#v, want GlobalGraphMsg", content.RunCommand("graph all", ""))
	}
	want := core.StatusMsg{Text: "E488: Trailing characters: foo (use :graph or :graph all)", Error: true}
	if got := content.RunCommand("graph foo", ""); got != want {
		t.Errorf(":graph foo = %#v, want %#v", got, want)
	}
}
