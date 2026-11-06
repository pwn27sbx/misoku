package sidebar_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/sidebar"
	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// statusPages is a flat workspace with one note per status and a plain one.
func statusPages() []core.Page {
	return []core.Page{
		{ID: "act.md", Title: "act"},
		{ID: "hold.md", Title: "hold"},
		{ID: "done.md", Title: "done"},
		{ID: "drop.md", Title: "drop"},
		{ID: "plain.md", Title: "plain"},
	}
}

// statusRow returns the raw (styled) row of v whose text contains name.
func statusRow(t *testing.T, v, name string) string {
	t.Helper()
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(ansi.Strip(l), name) {
			return l
		}
	}
	t.Fatalf("no row with %q in:\n%s", name, ansi.Strip(v))
	return ""
}

// fgBefore reports whether the last escape sequence before glyph in row sets
// the foreground to c.
func fgBefore(row, glyph string, c lipgloss.Color) bool {
	i := strings.Index(row, glyph)
	if i < 0 {
		return false
	}
	pre := row[:i]
	j := strings.LastIndex(pre, "\x1b[")
	if j < 0 {
		return false
	}
	return strings.Contains(pre[j:], termenv.TrueColor.Color(string(c)).Sequence(false))
}

func TestExplorerShowsNoteStatusGlyphInIconCell(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	th := theme.DefaultTheme()
	plainSB := sidebar.New(th)
	plainSB.SetSize(30, 20)
	plainSB.SetPages(statusPages())
	before := plainSB.View()

	sb := sidebar.New(th)
	sb.SetSize(30, 20)
	sb.SetPages(statusPages())
	sb.SetStatuses(map[string]core.NoteStatus{
		"act.md":  core.StatusActive,
		"hold.md": core.StatusOnHold,
		"done.md": core.StatusCompleted,
		"drop.md": core.StatusDropped,
	})
	v := sb.View()

	plainRow := ansi.Strip(statusRow(t, v, "plain"))
	nameCol := ansi.StringWidth(plainRow[:strings.Index(plainRow, "plain")])
	cases := []struct {
		name, glyph string
		color       lipgloss.Color
	}{
		{"act", "●", th.Blue},
		{"hold", "◐", th.Yellow},
		{"done", "✓", th.Green},
		{"drop", "✗", th.MutedFg},
	}
	for _, c := range cases {
		row := statusRow(t, v, c.name)
		text := ansi.Strip(row)
		if !strings.Contains(text, c.glyph) {
			t.Errorf("%s: expected glyph %q in %q", c.name, c.glyph, text)
		}
		if strings.Contains(text, "\ue609") {
			t.Errorf("%s: the status glyph should replace the note icon: %q", c.name, text)
		}
		if !fgBefore(row, c.glyph, c.color) {
			t.Errorf("%s: glyph %q should be drawn in %s: %q", c.name, c.glyph, c.color, row)
		}
		if got := ansi.StringWidth(text[:strings.Index(text, c.name)]); got != nameCol {
			t.Errorf("%s: name starts at column %d, a plain note's at %d", c.name, got, nameCol)
		}
		if w := ansi.StringWidth(text); w != 30 {
			t.Errorf("%s: row is %d cells wide, want 30", c.name, w)
		}
	}
	if got, want := statusRow(t, v, "plain"), statusRow(t, before, "plain"); got != want {
		t.Errorf("a note without status should be drawn as before:\n got %q\nwant %q", got, want)
	}
}

func TestExplorerWithoutStatusesIsUnchanged(t *testing.T) {
	th := theme.DefaultTheme()
	a := sidebar.New(th)
	a.SetSize(30, 20)
	a.SetPages(statusPages())
	b := sidebar.New(th)
	b.SetSize(30, 20)
	b.SetPages(statusPages())
	b.SetStatuses(nil)
	b.SetStatuses(map[string]core.NoteStatus{"missing.md": core.StatusActive})
	if a.View() != b.View() {
		t.Fatalf("statuses of notes not in the tree should change nothing")
	}
}
