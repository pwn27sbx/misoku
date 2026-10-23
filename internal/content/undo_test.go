package content_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// newNote is an editor showing one note in NORMAL mode, with its own undo
// history as the app gives every tab one.
func newNote(text string) content.Model {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetHistory(content.NewHistory())
	c.SetPage(core.Page{ID: "a.md", Content: text})
	return c
}

// status runs cmd and returns the status bar text it asks for, if any.
func status(cmd tea.Cmd) string {
	if cmd == nil {
		return ""
	}
	switch msg := cmd().(type) {
	case core.StatusMsg:
		return msg.Text
	case tea.BatchMsg:
		for _, c := range msg {
			if s := status(c); s != "" {
				return s
			}
		}
	}
	return ""
}

func undo(c content.Model) (content.Model, string) {
	c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	return c, status(cmd)
}

func redo(c content.Model) (content.Model, string) {
	c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	return c, status(cmd)
}

func esc(c content.Model) content.Model {
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	return c
}

// S7: u undoes the last change and says which one, like vim.
func TestUndoRestoresTheTextBeforeTheLastChange(t *testing.T) {
	c := newNote("hello")
	c = esc(keys(c, "A world"))
	c, msg := undo(c)
	if c.Value() != "hello" {
		t.Fatalf("u should bring back the text before the change, got %q", c.Value())
	}
	if msg != "1 change; before #1" {
		t.Fatalf("status = %q, want %q", msg, "1 change; before #1")
	}
	c, msg = undo(c)
	if c.Value() != "hello" || msg != "Already at oldest change" {
		t.Fatalf("u with nothing to undo: text %q, status %q", c.Value(), msg)
	}
}

// S7: Ctrl+R redoes what u undid.
func TestCtrlRRedoesTheUndoneChange(t *testing.T) {
	c := newNote("hello")
	c = esc(keys(c, "A world"))
	c = esc(keys(c, "A!"))
	c, _ = undo(c)
	c, _ = undo(c)
	c, msg := redo(c)
	if c.Value() != "hello world" || msg != "1 change; after #1" {
		t.Fatalf("first Ctrl+R: text %q, status %q", c.Value(), msg)
	}
	c, msg = redo(c)
	if c.Value() != "hello world!" || msg != "1 change; after #2" {
		t.Fatalf("second Ctrl+R: text %q, status %q", c.Value(), msg)
	}
	c, msg = redo(c)
	if c.Value() != "hello world!" || msg != "Already at newest change" {
		t.Fatalf("Ctrl+R with nothing to redo: text %q, status %q", c.Value(), msg)
	}
	// A new change after an undo drops what could be redone.
	c, _ = undo(c)
	c = keys(c, "0x")
	c, msg = redo(c)
	if c.Value() != "ello world" || msg != "Already at newest change" {
		t.Fatalf("redo after a new change: text %q, status %q", c.Value(), msg)
	}
}

// S7: the bar text counts changes from the first one, as «before #3».
func TestUndoNamesTheChangeNumber(t *testing.T) {
	c := newNote("abc")
	c = keys(c, "xxx")
	_, msg := undo(c)
	if msg != "1 change; before #3" {
		t.Fatalf("status = %q, want %q", msg, "1 change; before #3")
	}
}

// S7: everything typed in one INSERT session is one change, whatever it
// typed or deleted, and however the session started (o opens a line first).
func TestOneInsertSessionIsOneChange(t *testing.T) {
	c := newNote("one")
	c = keys(c, "otwo")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	c = esc(keys(c, "o three"))
	if c.Value() != "one\ntwo three" {
		t.Fatalf("setup: got %q", c.Value())
	}
	c, _ = undo(c)
	if c.Value() != "one" {
		t.Fatalf("one u should undo the whole INSERT session, got %q", c.Value())
	}
}

// S7: dd, x and p are one change each.
func TestNormalCommandsAreOneChangeEach(t *testing.T) {
	restore := content.SetClipboard(func() (string, error) { return "P", nil })
	defer restore()
	c := newNote("one\ntwo\nthree")
	c = keys(c, "dd")
	c = keys(c, "x")
	c = keys(c, "p")
	if c.Value() != "wPo\nthree" {
		t.Fatalf("setup: got %q", c.Value())
	}
	for _, want := range []string{"wo\nthree", "two\nthree", "one\ntwo\nthree"} {
		c, _ = undo(c)
		if c.Value() != want {
			t.Fatalf("u: got %q, want %q", c.Value(), want)
		}
	}
}

// S7: a visual-mode cut and a ":" table command are a change each, and an
// edit u has not seen yet (still unrecorded) is never lost by undoing.
func TestVisualCutAndTableCommandUndo(t *testing.T) {
	c := newNote("| a | b |\n| --- | --- |\n| 1 | 2 |\nend")
	c = keys(c, ":tr")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if strings.Count(c.Value(), "\n") != 4 {
		t.Fatalf(":tr should add a row, got %q", c.Value())
	}
	c, _ = undo(c)
	if c.Value() != "| a | b |\n| --- | --- |\n| 1 | 2 |\nend" {
		t.Fatalf("u after :tr: got %q", c.Value())
	}
	c = keys(c, "GVd")
	if strings.Contains(c.Value(), "end") {
		t.Fatalf("V d should cut the line, got %q", c.Value())
	}
	c, _ = undo(c)
	if !strings.HasSuffix(c.Value(), "\nend") {
		t.Fatalf("u after v d: got %q", c.Value())
	}
}

// S7: the edits the app makes for the preview, the "/" menu and media are
// undoable one by one, and loading a note is not a change.
func TestAppEditsAreUndoableAndLoadingIsNot(t *testing.T) {
	c := newNote("| a | b |\n| --- | --- |\n| 1 | 2 |")
	c.SetTableCell(2, 0, "x")
	c.TableOpAt(2, 0, "addrow")
	c.SetFrontMatter("cover", "c.png")
	c.InsertText("link")
	c.ReplaceText("all new")
	steps := 0
	for range 20 {
		var msg string
		c, msg = undo(c)
		if msg == "Already at oldest change" {
			break
		}
		steps++
	}
	if steps != 5 {
		t.Fatalf("five app edits should be five changes, got %d", steps)
	}
	if c.Value() != "| a | b |\n| --- | --- |\n| 1 | 2 |" {
		t.Fatalf("undoing every app edit should give the loaded text, got %q", c.Value())
	}

	// Prior behaviour: an edit that does not apply says so and records
	// nothing.
	if !c.SetTableCell(99, 0, "no") && !c.TableOpAt(99, 0, "addrow") {
		if _, msg := undo(c); msg != "Already at oldest change" {
			t.Fatalf("a failed edit is not a change: %q", msg)
		}
	} else {
		t.Fatal("SetTableCell and TableOpAt should report a row outside the note")
	}

	// SetTableCell re-aligns the table too: still one change.
	c.SetTableCell(2, 1, "y")
	c, _ = undo(c)
	if c.Value() != "| a | b |\n| --- | --- |\n| 1 | 2 |" {
		t.Fatalf("SetTableCell should be one change, got %q", c.Value())
	}
}

// S7: each app edit is recorded as soon as it is made, so loading the same
// text again (a tab saved under a new name is reloaded into the editor)
// keeps it undoable.
func TestAppEditIsRecordedAtOnce(t *testing.T) {
	const table = "| a | b |\n| --- | --- |\n| 1 | 2 |"
	edits := map[string]func(c *content.Model){
		"ReplaceText":    func(c *content.Model) { c.ReplaceText("new") },
		"SetFrontMatter": func(c *content.Model) { c.SetFrontMatter("cover", "c.png") },
		"InsertText":     func(c *content.Model) { c.InsertText("link") },
		"TableOpAt":      func(c *content.Model) { c.TableOpAt(2, 0, "addrow") },
		"SetTableCell":   func(c *content.Model) { c.SetTableCell(2, 0, "x") },
	}
	for name, edit := range edits {
		c := newNote(table)
		edit(&c)
		c.SetPage(core.Page{ID: "a.md", Content: c.Value()})
		c, _ = undo(c)
		if c.Value() != table {
			t.Errorf("%s: u after reloading the same text: got %q", name, c.Value())
		}
	}
}

// S7: "/" menu edits are part of the INSERT session that typed them.
func TestSlashMenuEditIsUndone(t *testing.T) {
	c := newNote("x")
	c = keys(c, "o/h1")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	c = esc(keys(c, "Title"))
	if c.Value() != "x\n# Title" {
		t.Fatalf("setup: got %q", c.Value())
	}
	c, _ = undo(c)
	if c.Value() != "x" {
		t.Fatalf("u should undo the slash insert, got %q", c.Value())
	}
}

// S7: SetBuffer is not a change, and a history attached to another note is
// started over instead of restoring that note's text into this one.
func TestLoadingIsNotAChangeAndForeignHistoryIsDropped(t *testing.T) {
	h := content.NewHistory()
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetHistory(h)
	c.SetPage(core.Page{ID: "a.md", Content: "alpha"})
	c = keys(c, "x")

	// Same history, another note (a tab reused for another file).
	c.SetPage(core.Page{ID: "b.md", Content: "lpha"})
	c, msg := undo(c)
	if c.Value() != "lpha" || msg != "Already at oldest change" {
		t.Fatalf("another note must not undo into the first: %q, %q", c.Value(), msg)
	}

	// Coming back to the same note with its history keeps it.
	h2 := content.NewHistory()
	c.SetHistory(h2)
	c.SetPage(core.Page{ID: "c.md", Content: "gamma"})
	c = keys(c, "x")
	other := content.NewHistory()
	c.SetHistory(other)
	c.SetPage(core.Page{ID: "d.md", Content: "delta"})
	c.SetHistory(h2)
	c.SetPage(core.Page{ID: "c.md", Content: "amma"})
	c, _ = undo(c)
	if c.Value() != "gamma" {
		t.Fatalf("a tab's history should survive showing another tab, got %q", c.Value())
	}
}

// S7: a history whose current text is not the text loaded under the same
// note (changed behind its back) starts over instead of undoing into it.
func TestHistoryOfOtherTextIsDropped(t *testing.T) {
	h := content.NewHistory()
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetHistory(h)
	c.SetPage(core.Page{ID: "a.md", Content: "alpha"})
	c = keys(c, "x")
	c.SetPage(core.Page{ID: "a.md", Content: "something else"})
	c, msg := undo(c)
	if c.Value() != "something else" || msg != "Already at oldest change" {
		t.Fatalf("other text must not be undone into: %q, %q", c.Value(), msg)
	}
}

// S7: an INSERT session the app ends (focus left the editor) is one change
// of its own, and u never loses an edit it has not recorded yet: redo
// brings it back.
func TestUnrecordedEditIsNeverLostByUndo(t *testing.T) {
	c := newNote("hello")
	c = keys(c, "x")
	c = keys(c, "A1")
	c.ExitInsert()
	c.ReplaceText("replaced")
	c, _ = undo(c)
	if c.Value() != "ello1" {
		t.Fatalf("u after a preview edit: got %q", c.Value())
	}
	c, _ = undo(c)
	if c.Value() != "ello" {
		t.Fatalf("the INSERT session is its own change: got %q", c.Value())
	}

	c = newNote("hello")
	c = keys(c, "x")
	c = keys(c, "A1")
	c.ExitInsert()
	c, _ = undo(c)
	if c.Value() != "ello" {
		t.Fatalf("u undoes the session the app ended: got %q", c.Value())
	}
	c, _ = redo(c)
	if c.Value() != "ello1" {
		t.Fatalf("Ctrl+R brings the session back: got %q", c.Value())
	}
}

// S7: Rename moves a history to the note's new ID (rename, Save As).
func TestRenamedHistoryIsKept(t *testing.T) {
	h := content.NewHistory()
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetHistory(h)
	c.SetPage(core.Page{ID: "a.md", Content: "alpha"})
	c = keys(c, "x")
	h.Rename("b.md")
	c.SetPage(core.Page{ID: "b.md", Content: "lpha"})
	c, _ = undo(c)
	if c.Value() != "alpha" {
		t.Fatalf("a renamed note keeps its history, got %q", c.Value())
	}
}

// S7 / D7: ResetHistory drops every change, so a reload from disk cannot
// be undone into the text it replaced.
func TestResetHistoryDropsEveryChange(t *testing.T) {
	c := newNote("hello")
	c = keys(c, "x")
	c.ReplaceText("from disk")
	c.ResetHistory()
	c, msg := undo(c)
	if c.Value() != "from disk" || msg != "Already at oldest change" {
		t.Fatalf("after a reset: text %q, status %q", c.Value(), msg)
	}
}

// D5: at most 200 changes are kept; the oldest go first and the numbers
// keep counting from the first change.
func TestHistoryKeepsTwoHundredChanges(t *testing.T) {
	c := newNote(strings.Repeat("x", 250))
	for range 250 {
		c = keys(c, "x")
	}
	n := 0
	var first string
	for range 300 {
		var msg string
		c, msg = undo(c)
		if msg == "Already at oldest change" {
			break
		}
		if n == 0 {
			first = msg
		}
		n++
	}
	if n != 200 {
		t.Fatalf("expected 200 undoable changes, got %d", n)
	}
	if first != "1 change; before #250" {
		t.Fatalf("numbers should keep counting, got %q", first)
	}
	if c.Value() != strings.Repeat("x", 200) {
		t.Fatalf("oldest kept state: got %d runes", len(c.Value()))
	}
}

// D5: at most 8 MB of snapshots per tab; the oldest go first.
func TestHistoryKeepsAtMostEightMegabytes(t *testing.T) {
	const mb = 1 << 20
	// 8000 lines of 400 bytes: the textarea holds at most 10000 lines.
	c := newNote(strings.Repeat(strings.Repeat("a", 399)+"\n", 3*mb/400))
	for range 4 {
		c = keys(c, "x")
	}
	// Each snapshot is about 3 MB: two fit in 8 MB, so one change is left.
	n := 0
	for range 10 {
		var msg string
		c, msg = undo(c)
		if msg == "Already at oldest change" {
			break
		}
		n++
	}
	if n != 1 {
		t.Fatalf("expected 1 undoable change under 8 MB, got %d", n)
	}
}

// S7 (D5): a note so large that no change fits in 8 MB says why u has
// nothing to undo.
func TestUndoExplainsANoteTooLargeForHistory(t *testing.T) {
	const mb = 1 << 20
	// 5 MB in 500-byte lines: two copies exceed 8 MB.
	c := newNote(strings.Repeat(strings.Repeat("a", 499)+"\n", 5*mb/500))
	c = keys(c, "x")
	c, msg := undo(c)
	if msg != "Note too large to keep undo history (limit 8 MB)" {
		t.Fatalf("u on a note too large for history said %q", msg)
	}
}

// Prior behaviour: moving the cursor or opening INSERT and leaving it
// without typing records nothing.
func TestMovesAndEmptyInsertAreNotChanges(t *testing.T) {
	c := newNote("one\ntwo")
	c = esc(keys(c, "jlhki"))
	c, msg := undo(c)
	if msg != "Already at oldest change" || c.Value() != "one\ntwo" {
		t.Fatalf("no change was made: text %q, status %q", c.Value(), msg)
	}
}
