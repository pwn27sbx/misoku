package content_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

func keys(c content.Model, s string) content.Model {
	for _, r := range s {
		c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return c
}

func TestContentComponent(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "notes/Notes.md", Title: "Notes", Content: "Hello world"})

	view := c.View()
	if strings.Contains(view, "Notes.md") || !strings.Contains(view, "Hello world") {
		t.Errorf("expected only the text (no file name row), got %q", view)
	}
	if c.ModeString() != "NORMAL" {
		t.Errorf("expected NORMAL mode, got %q", c.ModeString())
	}
	if got := strings.Count(view, "\n") + 1; got != 20 {
		t.Errorf("expected exactly 20 rows, got %d", got)
	}
}

func TestVimEdits(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "a.md", Content: "one\ntwo\nthree"})

	c = keys(c, "jdd")
	if c.Value() != "one\nthree" {
		t.Fatalf("dd: got %q", c.Value())
	}
	c = keys(c, "x")
	if c.Value() != "one\nhree" {
		t.Fatalf("x: got %q", c.Value())
	}
	c = keys(c, "ozero")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if c.Value() != "one\nhree\nzero" || c.Mode() != content.ModeNormal {
		t.Fatalf("o: got %q mode %v", c.Value(), c.Mode())
	}
	c = keys(c, "ggA!")
	if !strings.HasPrefix(c.Value(), "one!") {
		t.Fatalf("gg A: got %q", c.Value())
	}
}

func TestWheelScrollsWithoutFocus(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 10)
	var b strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&b, "row %d\n", i)
	}
	c.SetPage(core.Page{ID: "a.md", Content: b.String()})
	for i := 0; i < 10; i++ {
		c, _ = c.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	}
	v := c.View()
	if strings.Contains(v, "row 1\n") || !strings.Contains(v, "row 31") {
		t.Fatalf("expected view scrolled by 30 lines, got:\n%s", v)
	}
	if line, _ := c.CursorPosition(); line != 31 {
		t.Fatalf("cursor should be dragged to the first visible line, got %d", line)
	}
}

func TestCommandParsing(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 10)
	c.SetPage(core.Page{ID: "a.md", Content: "x"})
	c = keys(c, ":w notes/new")
	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	var found bool
	for _, msg := range collect(cmd) {
		if s, ok := msg.(core.VimSaveMsg); ok && s.Path == "notes/new" && s.Content == "x" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected VimSaveMsg with path from ':w notes/new'")
	}
}

func TestExportCommand(t *testing.T) {
	for in, want := range map[string]string{":export": "", ":export out/report.pdf": "out/report.pdf", ":pdf notes": "notes"} {
		c := content.New(theme.DefaultTheme())
		c.SetSize(80, 10)
		c.SetPage(core.Page{ID: "a.md", Content: "x"})
		c = keys(c, in)
		_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
		found := false
		for _, msg := range collect(cmd) {
			if e, ok := msg.(core.ExportMsg); ok && e.Path == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: expected ExportMsg{Path: %q}", in, want)
		}
	}
}

func collect(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, collect(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func TestSlashMenuInsertsBlocks(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "a.md", Content: ""})
	c.SetFocused(true)
	c = keys(c, "i/")
	if !c.SlashOpen() {
		t.Fatal("expected '/' at line start to open the block menu")
	}
	if box, _, _, ok := c.SlashView(); !ok || !strings.Contains(box, "Heading 1") || !strings.Contains(box, "Basic blocks") {
		t.Fatalf("expected menu with sections, got %q", box)
	}
	c = keys(c, "todo")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	c = keys(c, "buy milk")
	if c.Value() != "- [ ] buy milk" || c.SlashOpen() {
		t.Fatalf("expected to-do inserted, got %q", c.Value())
	}

	// Code block puts the cursor inside the fence.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	c = keys(c, "/code")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	c = keys(c, "x := 1")
	if !strings.HasSuffix(c.Value(), "```\nx := 1\n```") {
		t.Fatalf("expected cursor inside code fence, got %q", c.Value())
	}

	// A slash inside a word is just a slash; Esc closes without changes.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	c = keys(c, "Goa/b")
	if c.SlashOpen() {
		t.Fatal("slash inside a word must not open the menu")
	}
	c = keys(c, " /")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if c.SlashOpen() || c.Mode() != content.ModeInsert {
		t.Fatal("Esc should only close the menu")
	}
}

func TestTabIndents(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 10)
	c.SetPage(core.Page{ID: "a.md", Content: ""})
	c.SetFocused(true)
	c = keys(c, "i- item")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	if c.Value() != "  - item" {
		t.Fatalf("Tab on a list line should nest it, got %q", c.Value())
	}
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if c.Value() != "- item" {
		t.Fatalf("Shift+Tab should outdent, got %q", c.Value())
	}
	c = keys(c, " x")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	c = keys(c, "a")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	c = keys(c, "b")
	if !strings.HasSuffix(c.Value(), "\na  b") {
		t.Fatalf("Tab in text should insert two spaces, got %q", c.Value())
	}
}

func TestWideRunesNeverPanic(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetFocused(true)
	line := strings.Repeat("綴り • Terminal 綴 ", 6)
	for w := 20; w <= 60; w++ {
		c.SetSize(w, 10)
		c.SetPage(core.Page{ID: "a.md", Content: line + "\n" + line})
		for i := 0; i < len([]rune(line))+2; i++ {
			_ = c.View()
			c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		}
		c = keys(c, "A")
		_ = c.View()
		c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	}
}

func TestClickPastLineEndNeverPanics(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 10)
	c.SetPage(core.Page{ID: "a.md", Content: "```\n#include <stdio.h>\n\n```\n\n![x](a.png)"})
	for y := 0; y < 8; y++ {
		for _, x := range []int{0, 6, 9, 10, 20, 79} {
			c, _ = c.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
			_ = c.View()
		}
	}
	for i := 0; i < 5; i++ {
		c, _ = c.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
		_ = c.View()
	}
}

func TestModeStringAndFocus(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetPage(core.Page{ID: "a.md", Content: "x"})
	if got := c.ModeString(); got != "NORMAL" {
		t.Errorf("ModeString = %q", got)
	}
	c.EnterInsert()
	if got := c.ModeString(); got != "INSERT" {
		t.Errorf("ModeString = %q", got)
	}
	c.ExitInsert()
	if got := c.ModeString(); got != "NORMAL" {
		t.Errorf("ModeString = %q", got)
	}
	if !c.HasPage() || c.LineCount() != 1 {
		t.Errorf("HasPage=%v LineCount=%d", c.HasPage(), c.LineCount())
	}
	c.CommandView(20, "4", "")
	c.SetTheme(theme.DefaultTheme())
	c.Focus()
	c.Blur()
	c.Init()
}

func TestBufferCommands(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetPage(core.Page{ID: "a.md", Content: "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight"})
	c.GotoLine(3)
	if l, _ := c.CursorPosition(); l != 3 {
		t.Errorf("line = %d", l)
	}
	c.ReplaceText("a\nb")
	if c.Value() != "a\nb" {
		t.Errorf("Value = %q", c.Value())
	}
	c.SetPage(core.Page{ID: "a.md", Content: strings.Join([]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}, "\n")})
	c.SetSize(40, 4)
	c.ScrollBy(3)
}

func TestSetFrontMatterAndInsertText(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetPage(core.Page{ID: "a.md", Content: "body text"})
	c.SetFrontMatter("title", "Hello")
	if got := c.Value(); !strings.HasPrefix(got, "---\ntitle: Hello\n---\n") {
		t.Errorf("Value = %q", got)
	}
	c.SetFrontMatter("title", "Updated")
	if got := c.Value(); !strings.Contains(got, "title: Updated") {
		t.Errorf("Value = %q", got)
	}
	c.InsertText("hi ")
	if !strings.Contains(c.Value(), "hi ") {
		t.Errorf("Value = %q", c.Value())
	}
}

func TestTableOpAtAndSetTableCell(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetPage(core.Page{ID: "a.md", Content: "| a | b |\n| --- | --- |\n| 1 | 2 |"})
	if !c.SetTableCell(2, 0, "99") {
		t.Error("SetTableCell should succeed on the cell row")
	}
	if !strings.Contains(c.Value(), "99") {
		t.Errorf("Value = %q", c.Value())
	}
	_ = c.TableOpAt(2, 0, "addCol")
}

func TestTableEditing(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "a.md", Content: "| a | b |\n| --- | --- |\n| 1 | 2 |"})
	c.SetFocused(true)

	// Cursor on the "1" row; add a column to the right of the first cell.
	c = keys(c, "jj")
	c = keys(c, ":addcol")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	want := "| a   | Column | b   |\n| --- | ------ | --- |\n| 1   |        | 2   |"
	if c.Value() != want {
		t.Fatalf("addcol:\n%s\nwant\n%s", c.Value(), want)
	}

	// Tab moves through cells and adds a row after the last one.
	c = keys(c, "i")
	// From the new "Column" cell: Tab -> "2", Tab -> new row, first cell.
	for i := 0; i < 2; i++ {
		c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	c = keys(c, "x")
	if lines := strings.Split(c.Value(), "\n"); len(lines) != 4 || !strings.HasPrefix(lines[3], "| x") {
		t.Fatalf("Tab past the last cell should add a row and type into it:\n%s", c.Value())
	}

	// /delete row from the slash menu.
	c = keys(c, " /delete row")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if strings.Count(c.Value(), "\n") != 2 {
		t.Fatalf("expected the new row deleted:\n%s", c.Value())
	}

	// Outside a table the table actions are hidden and :addrow complains.
	c.SetPage(core.Page{ID: "b.md", Content: "plain"})
	c = keys(c, "A /")
	if box, _, _, _ := c.SlashView(); strings.Contains(box, "Add row") {
		t.Fatal("table actions should only show inside a table")
	}
}

// enter presses Enter and returns what it asked the app for.
func enter(c content.Model) (content.Model, []tea.Msg) {
	c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return c, collect(cmd)
}

// followed returns the link an Enter asked the app to follow, if any.
func followed(msgs []tea.Msg) (core.WikiLink, bool) {
	for _, msg := range msgs {
		if f, ok := msg.(content.FollowWikiLinkMsg); ok {
			return f.Link, true
		}
	}
	return core.WikiLink{}, false
}

// S5 (a): Enter on a [[link]] outside code follows it and leaves the text
// and the cursor's line as they are.
func TestEnterFollowsWikiLink(t *testing.T) {
	cases := []struct {
		text, keys, target string
	}{
		{"see [[Other#Sec|alias]] here", "llll", "Other"}, // on the first "["
		{"see [[Other#Sec|alias]] here", "lllllllllllllllllllll", "Other"},
		{"`x` [[After]]", "llllll", "After"},     // code before the link
		{"- [ ] [[Boxed]]", "llllllll", "Boxed"}, // a link beats the box
		{"```\nx\n```\n[[Closed]]", "jjj", "Closed"},
		{"![[pic.png]]", "", "pic.png"}, // embeds too, as nvim's link_at
	}
	for _, tc := range cases {
		c := keys(newNote(tc.text), tc.keys)
		row, _ := c.CursorPosition()
		c, msgs := enter(c)
		l, ok := followed(msgs)
		if !ok || l.Target != tc.target {
			t.Errorf("%q + %q: followed %v %q, want %q", tc.text, tc.keys, ok, l.Target, tc.target)
		}
		if c.Value() != tc.text {
			t.Errorf("%q: following must not edit, got %q", tc.text, c.Value())
		}
		if r, _ := c.CursorPosition(); r != row {
			t.Errorf("%q: following must not move the cursor, row %d -> %d", tc.text, row, r)
		}
	}
	c, msgs := enter(keys(newNote("see [[Other#Sec]]"), "llll"))
	if l, _ := followed(msgs); l.Heading != "Sec" {
		t.Errorf("the heading travels with the link, got %+v (%q)", l, c.Value())
	}
}

// S5: a link inside fenced or inline code is text: Enter moves down.
func TestEnterIgnoresLinksInCode(t *testing.T) {
	cases := []struct {
		text, keys string
	}{
		{"```\n[[A]]\n```", "j"},
		{"~~~\n[[A]]\n~~~", "j"},
		{"````\n```\n[[A]]\n````", "jj"}, // a shorter run does not close
		{"```go\n[[A]]\n```", "j"},
		{"```[[A]]\nnext", ""}, // the opening fence line itself
		{"`[[A]]` x\nnext", "ll"},
		{"``a ` [[A]]``\nnext", "llllll"},
		{"see [[A]] x\nnext", "llllllllll"}, // past the link
	}
	for _, tc := range cases {
		c := keys(newNote(tc.text), tc.keys)
		row, _ := c.CursorPosition()
		c, msgs := enter(c)
		if _, ok := followed(msgs); ok {
			t.Errorf("%q + %q: a link in code was followed", tc.text, tc.keys)
		}
		if r, _ := c.CursorPosition(); r != row+1 {
			t.Errorf("%q + %q: Enter should move down from row %d, got %d", tc.text, tc.keys, row, r)
		}
		if c.Value() != tc.text {
			t.Errorf("%q: got %q", tc.text, c.Value())
		}
	}
}

// S5 (b): Enter on a box line toggles it, inside a fence too, as one change.
func TestEnterTogglesCheckbox(t *testing.T) {
	c, msgs := enter(keys(newNote("- [ ] task\nnext"), "ll"))
	if c.Value() != "- [x] task\nnext" || len(msgs) != 0 {
		t.Fatalf("Enter on a box: got %q %v", c.Value(), msgs)
	}
	if r, col := c.CursorPosition(); r != 1 || col != 3 {
		t.Fatalf("the cursor stays put, got %d:%d", r, col)
	}
	c, _ = undo(c)
	if c.Value() != "- [ ] task\nnext" {
		t.Fatalf("u: got %q", c.Value())
	}
	c, _ = enter(keys(newNote("```\n- [x] a\n```"), "j"))
	if c.Value() != "```\n- [ ] a\n```" {
		t.Fatalf("Enter on a box inside a fence: got %q", c.Value())
	}
}

// S5 (c), prior behaviour: Enter anywhere else moves down and edits nothing.
func TestEnterMovesDownElsewhere(t *testing.T) {
	for _, text := range []string{"plain\nnext", "- item\nnext", "- [?] a\nnext", "[t](n.md)\nnext", "https://x.y\nnext"} {
		c, msgs := enter(newNote(text))
		if r, _ := c.CursorPosition(); r != 2 || c.Value() != text || len(msgs) != 0 {
			t.Errorf("%q: Enter should only move down, got row %d %q %v", text, r, c.Value(), msgs)
		}
	}
}

// selection returns the text the selection holds, through y.
func selection(t *testing.T, c content.Model) string {
	t.Helper()
	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	for _, msg := range collect(cmd) {
		if cp, ok := msg.(core.CopyMsg); ok {
			return cp.Text
		}
	}
	return ""
}

// S3: in VISUAL and V-LINE the toggle reaches every line from the
// selection's first row to its last, keeps the selection, and is one u.
func TestToggleCheckboxesRange(t *testing.T) {
	const text = "a\n- [ ] b\n- [x] c\nd"
	for _, start := range []string{"V", "lv"} {
		c := keys(newNote(text), start+"jj")
		if !c.ToggleCheckboxes() {
			t.Fatalf("%s: ToggleCheckboxes reported no change", start)
		}
		if want := "- [ ] a\n- [x] b\n- [ ] c\nd"; c.Value() != want {
			t.Fatalf("%s: got %q, want %q", start, c.Value(), want)
		}
		if c.Mode() == content.ModeNormal {
			t.Fatalf("%s: the selection should stay", start)
		}
		if got := selection(t, c); start == "V" && got != "- [ ] a\n- [x] b\n- [ ] c" {
			t.Fatalf("V: the selection should cover the same rows, got %q", got)
		}
		c = esc(c) // u does nothing in VISUAL
		c, _ = undo(c)
		if c.Value() != text {
			t.Fatalf("%s: one u should restore every line, got %q", start, c.Value())
		}
		if _, msg := undo(c); msg != "Already at oldest change" {
			t.Fatalf("%s: the range should be one change, got %q", start, msg)
		}
	}
	// Upwards: the cursor above the anchor.
	c := keys(newNote(text), "jjVk")
	c.ToggleCheckboxes()
	if c.Value() != "a\n- [x] b\n- [ ] c\nd" {
		t.Fatalf("upward range: got %q", c.Value())
	}
}

// S2/S4: in NORMAL only the cursor's line; each call is a change of its own,
// recorded at once.
func TestCheckboxEditsAreOneChangeEach(t *testing.T) {
	c := keys(newNote("a\nb"), "j")
	c.ToggleCheckboxes()
	c.ToggleCheckboxes()
	if c.Value() != "a\n- [x] b" {
		t.Fatalf("got %q", c.Value())
	}
	c, _ = undo(c)
	if c.Value() != "a\n- [ ] b" {
		t.Fatalf("each toggle is one change, got %q", c.Value())
	}
	c = keys(newNote("- [x] a\n- [ ] b"), "Vj")
	if !c.RemoveCheckboxes() || c.Value() != "- a\n- b" {
		t.Fatalf("RemoveCheckboxes on a range: got %q", c.Value())
	}
	c = newNote("plain")
	if c.RemoveCheckboxes() {
		t.Fatal("removing from a line without a box changes nothing")
	}
	if _, msg := undo(c); msg != "Already at oldest change" {
		t.Fatalf("no change, nothing to undo: %q", msg)
	}
}

// S6: JumpLink moves to the start of the next (dir 1) or previous (-1)
// [[link]], wrapping, and skips links in fenced or inline code.
func TestJumpLink(t *testing.T) {
	const text = "a [[One]] b\n```\n[[Fenced]]\n```\n`[[Inline]]` ñ ![[Two]]\n[[Three]]"
	cases := []struct {
		name  string
		moves []int
		line  int
		col   int
	}{
		{"next from the top", []int{1}, 1, 3},
		{"next skips fence and inline code, lands on !", []int{1, 1}, 5, 16},
		{"next after the last wraps to the first", []int{1, 1, 1, 1}, 1, 3},
		{"previous from the top wraps to the last", []int{-1}, 6, 1},
		{"previous skips back over code", []int{-1, -1, -1}, 1, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := content.New(theme.DefaultTheme())
			c.SetSize(80, 20)
			c.SetPage(core.Page{ID: "n.md", Title: "n", Content: text})
			for _, d := range tc.moves {
				if !c.JumpLink(d) {
					t.Fatalf("JumpLink(%d) found no link", d)
				}
			}
			if line, col := c.CursorPosition(); line != tc.line || col != tc.col {
				t.Fatalf("cursor at %d:%d, want %d:%d", line, col, tc.line, tc.col)
			}
		})
	}
}

// S6: with no link outside code JumpLink reports false and the cursor stays.
func TestJumpLinkWithoutLinks(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "n.md", Title: "n", Content: "plain\n```\n[[Fenced]]\n```\n`[[Inline]]`"})
	c = keys(c, "jl")
	for _, d := range []int{1, -1} {
		if c.JumpLink(d) {
			t.Fatalf("JumpLink(%d) should find no link", d)
		}
		if line, col := c.CursorPosition(); line != 2 || col != 2 {
			t.Fatalf("the cursor moved to %d:%d", line, col)
		}
	}
}
