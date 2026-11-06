package dashboard_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestHomeRecentNoteShowsItsStatusGlyph(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	th := theme.DefaultTheme()
	plain := dashboard.New(th)
	plain.SetSize(100, 50)
	plain.SetRecentPages(workspacePages())
	before := plain.View()

	db := dashboard.New(th)
	db.SetSize(100, 50)
	db.SetRecentPages(workspacePages())
	db.SetStatuses(map[string]core.NoteStatus{"Misoku/Plan.md": core.StatusCompleted})
	raw := db.View()

	row := rowWith(raw, "Plan.md")
	text := ansi.Strip(row)
	if !strings.Contains(text, "✓") || strings.Contains(text, "") {
		t.Fatalf("the recent note should show ✓ in place of its icon, row %q", text)
	}
	i := strings.Index(row, "✓")
	if pre := row[:i]; !strings.Contains(pre[strings.LastIndex(pre, "\x1b["):], fgCode(th.Green)) {
		t.Fatalf("✓ should be green (%s), row %q", th.Green, row)
	}
	old := ansi.Strip(rowWith(before, "Plan.md"))
	if strings.Replace(old, "", "✓", 1) != text {
		t.Fatalf("only the icon should change:\n got %q\nwant %q", text, old)
	}
	if a, b := rowWith(raw, "suelta.md"), rowWith(before, "suelta.md"); a != b {
		t.Fatalf("a recent note without status should be drawn as before:\n got %q\nwant %q", a, b)
	}
	if raw == before {
		t.Fatal("setting a status should change the home screen")
	}
	db.SetStatuses(nil)
	if db.View() != before {
		t.Fatal("clearing the statuses should bring back the plain icons")
	}
}
