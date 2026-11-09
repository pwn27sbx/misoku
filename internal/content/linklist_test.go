package content_test

import (
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// linkSource is the vault the inline list reads: Beta first (the newest),
// then the note being edited, A.
func linkSource() []core.LinkNote {
	at := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	return []core.LinkNote{
		{ID: "beta.md", Title: "Beta", Text: "# Beta\n\n## Deploy\n\nsteps ^deploy", UpdatedAt: at},
		{ID: "a.md", Title: "A", Text: "# Top\n\n## Deploy\n\nbody ^first", UpdatedAt: at.Add(-time.Hour)},
	}
}

// linkListNote is a note with the inline list open after a typed "["[, at
// the end of the note so the text under it stays as written.
func linkListNote(t *testing.T, text string, src []core.LinkNote) content.Model {
	t.Helper()
	c := linkListNoteAt(t, text)
	c.GotoLine(c.LineCount())
	c = keys(c, "[[")
	c.OpenLinkList(src, c.RequestLinkList())
	return c
}

// linkListNoteAt is a note in INSERT mode, no link typed yet.
func linkListNoteAt(t *testing.T, text string) content.Model {
	t.Helper()
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 20)
	c.SetPage(core.Page{ID: "a.md", Title: "A", Content: text})
	c.SetFocused(true)
	c.EnterInsert()
	return c
}

// linkText is a note with headings and a block marker.
func linkText() string { return "# Top\n\n## Deploy\n\nbody ^first\n" }

// S3: "[[" opens the small list under the cursor, not the finder.
func TestLinkListOpensAfterDoubleBracket(t *testing.T) {
	c := linkListNote(t, "", linkSource())
	if !c.LinkListOpen() {
		t.Fatal("[[]] should open the inline link list")
	}
	box, _, _, ok := c.LinkListView()
	if !ok || !strings.Contains(box, "Beta") {
		t.Fatalf("the list should offer Beta, got %q", box)
	}
	if strings.Contains(c.View(), "Link to Note") {
		t.Fatal("no finder should open")
	}
}

// S3: typing filters the list; Enter inserts the completion and closes it.
func TestLinkListFiltersAndEnterInserts(t *testing.T) {
	c := linkListNote(t, "", linkSource())
	c = keys(c, "bet")
	if box, _, _, _ := c.LinkListView(); !strings.Contains(box, "Beta") || strings.Contains(box, "A\n") {
		t.Fatalf("typing should narrow the list, got %q", box)
	}
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if c.Value() != "[[Beta]]" {
		t.Fatalf("Enter should complete the link, got %q", c.Value())
	}
	if c.LinkListOpen() {
		t.Fatal("inserting should close the list")
	}
}

// S3: Tab inserts too.
func TestLinkListTabInserts(t *testing.T) {
	c := linkListNote(t, "", linkSource())
	c = keys(c, "bet")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	if c.Value() != "[[Beta]]" || c.LinkListOpen() {
		t.Fatalf("Tab should complete the link, got %q (open %v)", c.Value(), c.LinkListOpen())
	}
}

// S3: Esc closes the list only: the text and INSERT mode stay.
func TestLinkListEscClosesAndLeavesTheText(t *testing.T) {
	c := linkListNote(t, "", linkSource())
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if c.LinkListOpen() || c.Value() != "[[" || c.Mode() != content.ModeInsert {
		t.Fatalf("Esc should only close the list: %q, %v, %v", c.Value(), c.LinkListOpen(), c.Mode())
	}
}

// S3: typing the closing "]]" closes the list.
func TestLinkListClosesOnClosedBrackets(t *testing.T) {
	c := linkListNote(t, "", linkSource())
	c = keys(c, "Be]]")
	if c.LinkListOpen() {
		t.Fatal("]] should close the list")
	}
}

// S3: leaving INSERT closes the list.
func TestLinkListClosesLeavingInsert(t *testing.T) {
	c := linkListNote(t, "", linkSource())
	c.ExitInsert()
	if c.LinkListOpen() {
		t.Fatal("leaving INSERT should close the list")
	}
}

// S3: deleting the "[[" closes the list.
func TestLinkListClosesWhenTheBracketsGo(t *testing.T) {
	c := linkListNote(t, "", linkSource())
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if c.LinkListOpen() {
		t.Fatal("deleting [[]] should close the list")
	}
}

// S3: moving the cursor out of the link closes the list.
func TestLinkListClosesMovingTheCursorOut(t *testing.T) {
	c := linkListNote(t, "", linkSource())
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyLeft})
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if c.LinkListOpen() {
		t.Fatal("the cursor left the link, the list should close")
	}
}

// S1/S3: "#" lists the headings of the note, filtered as typed.
func TestLinkListHeadings(t *testing.T) {
	c := linkListNote(t, linkText(), linkSource())
	c = keys(c, "#")
	if box, _, _, _ := c.LinkListView(); !strings.Contains(box, "Top") || !strings.Contains(box, "Deploy") {
		t.Fatalf("# should list the headings, got %q", box)
	}
	c = keys(c, "dep")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if c.Value() != linkText()+"[[#Deploy]]" {
		t.Fatalf("Enter on a heading should insert it, got %q", c.Value())
	}
}

// S1/S3: "#^" lists block ids.
func TestLinkListBlocks(t *testing.T) {
	c := linkListNote(t, linkText(), linkSource())
	c = keys(c, "#^")
	if box, _, _, _ := c.LinkListView(); !strings.Contains(box, "^first") {
		t.Fatalf("#^ should list the blocks, got %q", box)
	}
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if c.Value() != linkText()+"[[#^first]]" {
		t.Fatalf("Enter on a block should insert it, got %q", c.Value())
	}
}

// S3/A7: a typed name with no match completes as a link to a new note.
func TestLinkListCompletesAnUnknownName(t *testing.T) {
	c := linkListNote(t, "", nil)
	c = keys(c, "Brand New")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if c.Value() != "[[Brand New]]" {
		t.Fatalf("Enter should complete the typed name, got %q", c.Value())
	}
}

// A7: Enter with an empty query inserts nothing and is not passed through.
func TestLinkListEnterOnAnEmptyQueryInsertsNothing(t *testing.T) {
	c := linkListNote(t, "", nil)
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if c.Value() != "[[" || c.LinkListOpen() {
		t.Fatalf("Enter on an empty query should insert nothing, got %q", c.Value())
	}
}

// S3: a click on an item inserts it.
func TestLinkListMouseClickInserts(t *testing.T) {
	c := linkListNote(t, "", linkSource())
	box, x, y, ok := c.LinkListView()
	if !ok || !strings.Contains(box, "Beta") {
		t.Fatalf("list not drawn: %q", box)
	}
	c, _ = c.Update(tea.MouseMsg{X: x + 2, Y: y + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if c.Value() != "[[Beta]]" {
		t.Fatalf("a click on the first item should insert it, got %q", c.Value())
	}
}

// S3/A8: a "[[" typed inside a table cell opens the list and completes.
func TestLinkListInATableCell(t *testing.T) {
	c := linkListNoteAt(t, "| Name | Value |\n| --- | --- |\n| a | b |\n")
	c.GotoLine(3)
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnd})
	c = keys(c, " [[")
	c.OpenLinkList(linkSource(), c.RequestLinkList())
	if !c.LinkListOpen() {
		t.Fatal("[[]] inside a table cell should open the inline list")
	}
	c = keys(c, "bet")
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !strings.Contains(c.Value(), "| a | b | [[Beta]]") {
		t.Fatalf("the link should land in the cell, got %q", c.Value())
	}
}

// S3: with no room below, the list opens above the cursor.
func TestLinkListOpensAboveWhenItDoesNotFit(t *testing.T) {
	c := content.New(theme.DefaultTheme())
	c.SetSize(80, 8)
	c.SetPage(core.Page{ID: "a.md", Title: "A", Content: strings.Repeat("line\n", 20)})
	c.SetFocused(true)
	c.GotoLine(c.LineCount())
	c.EnterInsert()
	c = keys(c, "A[[")
	c.OpenLinkList(linkSource(), c.RequestLinkList())
	_, cy, ok := c.CursorScreen()
	if !ok {
		t.Fatal("no cursor")
	}
	_, _, y, drawn := c.LinkListView()
	if !drawn || y >= cy {
		t.Fatalf("the list should open above the cursor: y=%d, cursor=%d", y, cy)
	}
}

// The app reads the notes away from Update: a read that arrives after Esc
// must not bring the list back. Only a new "[[" opens it again.
func TestLinkListStaysClosedAfterALateSource(t *testing.T) {
	c := linkListNoteAt(t, "")
	c.GotoLine(c.LineCount())
	c = keys(c, "[[#step")
	req := c.RequestLinkList()
	c.OpenLinkList(linkSource(), req)
	if !c.LinkListOpen() {
		t.Fatal("the trigger should open the list")
	}
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if c.LinkListOpen() {
		t.Fatal("Esc should close the list")
	}
	c.OpenLinkList(linkSource(), req) // the read was still in flight
	if c.LinkListOpen() {
		t.Fatal("a late source must not reopen the list")
	}

	// A new "[[" asks again; the old read still does not open it.
	c.EnterInsert()
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnd})
	c = keys(c, " [[")
	again := c.RequestLinkList()
	if again == 0 || again == req {
		t.Fatalf("a new trigger should ask with a new token, got %d after %d", again, req)
	}
	c.OpenLinkList(linkSource(), req)
	if c.LinkListOpen() {
		t.Fatal("the read of an older trigger must not open the list")
	}
	c.OpenLinkList(linkSource(), again)
	if !c.LinkListOpen() {
		t.Fatal("a new trigger should open the list")
	}
}

// Every way of closing the list also drops the trigger its read waits for.
func TestLinkListDropsALateSourceAfterEveryClose(t *testing.T) {
	closes := []struct {
		name  string
		close func(content.Model) content.Model
	}{
		{"]]", func(c content.Model) content.Model { return keys(c, "]]") }},
		{"leaving INSERT", func(c content.Model) content.Model {
			c.ExitInsert()
			return c
		}},
		{"the cursor leaving the link", func(c content.Model) content.Model {
			for i := 0; i < 8; i++ {
				c, _ = c.Update(tea.KeyMsg{Type: tea.KeyLeft})
			}
			return c
		}},
		{"deleting [[", func(c content.Model) content.Model {
			for i := 0; i < 5; i++ { // back to just after the "[["; "#step" is 5 runes
				c, _ = c.Update(tea.KeyMsg{Type: tea.KeyLeft})
			}
			c, _ = c.Update(tea.KeyMsg{Type: tea.KeyBackspace})
			c, _ = c.Update(tea.KeyMsg{Type: tea.KeyBackspace})
			return c
		}},
	}
	for _, tc := range closes {
		t.Run(tc.name, func(t *testing.T) {
			c := linkListNoteAt(t, "")
			c.GotoLine(c.LineCount())
			c = keys(c, "[[#step")
			req := c.RequestLinkList()
			c.OpenLinkList(linkSource(), req)
			if !c.LinkListOpen() {
				t.Fatal("the trigger should open the list")
			}
			c = tc.close(c)
			if c.LinkListOpen() {
				t.Fatal("the close should hide the list")
			}
			c.OpenLinkList(linkSource(), req)
			if c.LinkListOpen() {
				t.Fatal("a late source must not reopen the list")
			}
		})
	}
}
