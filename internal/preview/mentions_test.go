package preview_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// mentionPreview is Docker.md with one backlink.
func mentionPreview() preview.Model {
	m := preview.New(theme.DefaultTheme())
	m.SetSize(80, 30)
	m.SetPage(core.Page{ID: "Docker.md", Title: "Docker", Content: "body"})
	m.SetBacklinks([]core.Backlink{{ID: "Study.md", Title: "Study", Line: 3, Text: "Learn [[Docker]]"}})
	return m
}

// hitOn returns the hit under the first occurrence of text on screen.
func hitOn(t *testing.T, m preview.Model, text string) preview.Hit {
	t.Helper()
	v := ansi.Strip(m.View())
	for y, l := range strings.Split(v, "\n") {
		if i := strings.Index(l, text); i >= 0 {
			h, ok := m.HitAt(ansi.StringWidth(l[:i]), y)
			if !ok {
				t.Fatalf("no hit on %q", text)
			}
			return h
		}
	}
	t.Fatalf("%q not on screen:\n%s", text, v)
	return preview.Hit{}
}

// S8: the "Unlinked mentions" section follows the backlinks; a row opens
// its note at the mention, and its link button asks to link it.
func TestUnlinkedMentionsSection(t *testing.T) {
	m := mentionPreview()
	m.SetMentions("Docker.md", []core.UnlinkedMention{{ID: "Ops.md", Title: "Ops", Line: 2, Text: "we run docker", Count: 2}})
	v := ansi.Strip(m.View())
	back, ment := strings.Index(v, "Linked from 1 note"), strings.Index(v, "Unlinked mentions in 1 note")
	if back < 0 || ment < back {
		t.Fatalf("expected the mentions section after the backlinks:\n%s", v)
	}
	if !strings.Contains(v, "Ops · 2 mentions") || !strings.Contains(v, "we run docker") {
		t.Fatalf("expected the row and its snippet:\n%s", v)
	}
	if h := hitOn(t, m, "Ops"); h.Kind != preview.HitNoteLink || h.Arg != "Ops.md" || h.Index != 2 {
		t.Fatalf("row hit = %+v", h)
	}
	if h := hitOn(t, m, "link ]"); h.Kind != preview.HitMentionLink || h.Arg != "Ops.md" {
		t.Fatalf("link button hit = %+v", h)
	}
}

// S8: mentions found for another note (the one open before) are not shown.
func TestUnlinkedMentionsOfAnotherNoteAreNotShown(t *testing.T) {
	m := mentionPreview()
	m.SetMentions("Other.md", []core.UnlinkedMention{{ID: "Ops.md", Title: "Ops", Line: 1, Text: "x", Count: 1}})
	if v := ansi.Strip(m.View()); strings.Contains(v, "Unlinked mentions") {
		t.Fatalf("mentions of another note shown:\n%s", v)
	}
}

// Existing behaviour: a note without mentions renders exactly as before.
func TestNoMentionsRendersAsBefore(t *testing.T) {
	before := mentionPreview().View()
	m := mentionPreview()
	m.SetMentions("Docker.md", nil)
	if got := m.View(); got != before {
		t.Fatalf("a note without mentions changed:\n%s\nwant\n%s", got, before)
	}
}
