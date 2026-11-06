package content_test

import (
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// templateDay is the day the template tests insert on.
var templateDay = time.Date(2026, 11, 6, 9, 30, 0, 0, time.Local)

// S6: "/plantilla" and "/template" both pick the Template item, which asks
// the app for the template picker.
func TestSlashTemplateAsksTheAppForTheTemplatePicker(t *testing.T) {
	for _, typed := range []string{"plantilla", "template"} {
		c := keys(newNote(""), "i/"+typed)
		c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Fatalf("/%s: Enter returned no command", typed)
		}
		msg, ok := cmd().(core.SlashActionMsg)
		if !ok || msg.Action != "template" {
			t.Fatalf("/%s: Enter = %#v, want SlashActionMsg{template}", typed, cmd())
		}
		if c.Value() != "" {
			t.Fatalf("/%s: the typed query should be gone, the note is %q", typed, c.Value())
		}
	}
}

// S6: the placeholders take today's date and the note's title, in both
// languages, and the body lands at the cursor.
func TestInsertTemplateReplacesThePlaceholdersAtTheCursor(t *testing.T) {
	c := newNote("before\nafter")
	c = keys(c, "j") // the cursor on "after", column 0
	c.InsertTemplate("# {{titulo}} / {{title}}\n{{fecha}} {{date}}\n", "Plan", templateDay)
	want := "before\n# Plan / Plan\n2026-11-06 2026-11-06\nafter"
	if c.Value() != want {
		t.Fatalf("after the insert the note is %q, want %q", c.Value(), want)
	}
}

// S6: the template's front matter joins the note's: its new keys are added,
// the note's own keys win, and there is never a second block.
func TestInsertTemplateMergesTheFrontMatterIntoTheNotes(t *testing.T) {
	c := newNote("---\nstatus: active\n---\nbody")
	c = keys(c, "G$a") // INSERT after "body"
	c.InsertTemplate("---\nstatus: onhold\ntags: [daily, '{{fecha}}']\ncreated: {{fecha}}\n---\n\n## {{titulo}}", "Diary", templateDay)
	want := "---\nstatus: active\ntags: [daily, '2026-11-06']\ncreated: 2026-11-06\n---\nbody\n## Diary"
	if c.Value() != want {
		t.Fatalf("after the insert the note is %q, want %q", c.Value(), want)
	}
	c = esc(c)
	c, _ = undo(c)
	if c.Value() != "---\nstatus: active\n---\nbody" {
		t.Fatalf("one u should undo the whole insert, the note is %q", c.Value())
	}
}

// S6: a note without front matter gets the template's block on top, and
// the body still lands where the cursor was.
func TestInsertTemplateGivesANoteWithoutFrontMatterTheTemplatesBlock(t *testing.T) {
	c := newNote("first\n")
	c = keys(c, "G") // the empty last line
	c.InsertTemplate("---\ntype: meeting\n---\nAgenda", "Sync", templateDay)
	if want := "---\ntype: meeting\n---\nfirst\nAgenda"; c.Value() != want {
		t.Fatalf("after the insert the note is %q, want %q", c.Value(), want)
	}
}

// S6: with the cursor inside the note's front matter the body goes right
// after the block, never into it.
func TestInsertTemplateWithTheCursorInTheFrontMatterGoesBelowIt(t *testing.T) {
	c := newNote("---\nstatus: active\n---\nbody")
	c = keys(c, "j") // on "status: active"
	c.InsertTemplate("Hello", "x", templateDay)
	if want := "---\nstatus: active\n---\nHellobody"; c.Value() != want {
		t.Fatalf("after the insert the note is %q, want %q", c.Value(), want)
	}
}

// S6: the insert is one change of its own, so u brings the note back even
// when it lands in an INSERT session.
func TestInsertTemplateIsOneChangeInNormalMode(t *testing.T) {
	c := newNote("x")
	c.InsertTemplate("---\na: 1\n---\nT", "n", templateDay)
	c, _ = undo(c)
	if c.Value() != "x" {
		t.Fatalf("u after the insert = %q, want %q", c.Value(), "x")
	}
}
