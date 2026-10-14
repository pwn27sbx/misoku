package dashboard_test

import (
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardComponent(t *testing.T) {
	th := theme.DefaultTheme()
	db := dashboard.New(th)
	db.SetSize(100, 40)

	now := time.Now()
	pages := []core.Page{
		{ID: "p1.md", Title: "Intro Document", UpdatedAt: now.Add(-time.Hour)},
		{ID: "work/p2.md", Title: "Todo List", UpdatedAt: now},
		{ID: "work", Title: "work", IsFolder: true},
	}
	db.SetRecentPages(pages)

	view := db.View()
	for _, want := range []string{"New Note", "Find Note", "Intro Document.md", "Todo List.md", "2 notes"} {
		if !strings.Contains(view, want) {
			t.Errorf("expected view to contain %q, got %q", want, view)
		}
	}
	if got := strings.Count(view, "\n") + 1; got != 40 {
		t.Errorf("expected exactly 40 rows, got %d", got)
	}

	// Most recently edited note comes first.
	if recent := db.RecentPages(); len(recent) != 2 || recent[0].ID != "work/p2.md" {
		t.Errorf("expected newest note first, got %v", recent)
	}

	if a, _ := db.SelectedAction(); a != dashboard.ActionNewPage {
		t.Errorf("expected default action ActionNewPage, got %v", a)
	}
	db, _ = db.Update(tea.KeyMsg{Type: tea.KeyDown})
	if a, _ := db.SelectedAction(); a != dashboard.ActionFind {
		t.Errorf("expected ActionFind after down, got %v", a)
	}
	db, _ = db.Update(tea.KeyMsg{Type: tea.KeyUp})
	db, _ = db.Update(tea.KeyMsg{Type: tea.KeyUp})
	if a, id := db.SelectedAction(); a != dashboard.ActionOpenRecent || id != "p1.md" {
		t.Errorf("expected wrap-around to last recent note, got %v %q", a, id)
	}
}

func TestDashboardClick(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 40)
	db.SetRecentPages(nil)

	lines := strings.Split(db.View(), "\n")
	for y, l := range lines {
		if x := strings.Index(l, "Quit"); x >= 0 {
			a, _, ok := db.Click(len([]rune(l[:x])), y)
			if !ok || a != dashboard.ActionQuit {
				t.Fatalf("clicking Quit gave %v, %v", a, ok)
			}
			return
		}
	}
	t.Fatal("Quit button not rendered")
}

func TestDashboardZeroSize(t *testing.T) {
	th := theme.DefaultTheme()
	db := dashboard.New(th)
	db.SetSize(0, 0)

	if db.View() != "" {
		t.Errorf("expected empty string for zero size, got %q", db.View())
	}
}
