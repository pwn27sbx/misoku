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
	// The last item is now the "work" project, listed after the recent notes.
	if a, id := db.SelectedAction(); a != dashboard.ActionOpenProject || id != "work" {
		t.Errorf("expected wrap-around to the last item, the work project, got %v %q", a, id)
	}
	db, _ = db.Update(tea.KeyMsg{Type: tea.KeyUp})
	if a, id := db.SelectedAction(); a != dashboard.ActionOpenRecent || id != "p1.md" {
		t.Errorf("expected the last recent note above the projects, got %v %q", a, id)
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

// "Sync between computers" is offered on s only while the folder does not
// sync; it opens the setup assistant.
func TestDashboardOffersSyncOnlyWhenAsked(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 40)
	db.SetRecentPages(nil)
	if strings.Contains(db.View(), "Sync between computers") {
		t.Fatal("the sync entry must not show unless offered")
	}
	db.SetSyncSetup(true)
	view := db.View()
	if !strings.Contains(view, "Sync between computers") {
		t.Fatalf("the sync entry must show once offered:\n%s", view)
	}
	for _, l := range strings.Split(view, "\n") {
		if strings.Contains(l, "Sync between computers") && !strings.HasSuffix(strings.TrimRight(l, " "), "s") {
			t.Errorf("the sync entry must show its key, s: %q", l)
		}
	}
	found := false
	for i := 0; i < 10; i++ {
		if a, _ := db.SelectedAction(); a == dashboard.ActionSyncSetup {
			found = true
			break
		}
		db, _ = db.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if !found {
		t.Error("the sync entry must be selectable, with ActionSyncSetup")
	}
	db.SetSyncSetup(false)
	if strings.Contains(db.View(), "Sync between computers") {
		t.Error("the sync entry must go once it is no longer offered")
	}
	if a, _ := db.SelectedAction(); a == dashboard.ActionNone {
		t.Error("removing the entry must leave the cursor on an entry")
	}
	db.SetSyncSetup(true)
	db.SetSyncSetup(true)
	if n := strings.Count(db.View(), "Sync between computers"); n != 1 {
		t.Errorf("offering it twice must show it once, not %d times", n)
	}
}

// Adding or removing the sync entry keeps the cursor on the item it was
// on, not on the one that moved into its row.
func TestDashboardSyncEntryKeepsTheCursorOnItsItem(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 40)
	db.SetRecentPages([]core.Page{{ID: "a.md", Title: "a", UpdatedAt: time.Now()}})
	db.SetSyncSetup(true)
	for i := 0; i < 5; i++ { // New Note, Find, Explorer, Themes, Sync → Keymaps
		db, _ = db.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if a, _ := db.SelectedAction(); a != dashboard.ActionKeymap {
		t.Fatalf("setup: cursor on %v, want Keymaps", a)
	}
	db.SetSyncSetup(false)
	if a, _ := db.SelectedAction(); a != dashboard.ActionKeymap {
		t.Errorf("removing the sync entry moved the cursor from Keymaps to %v", a)
	}
	db.SetSyncSetup(true)
	if a, _ := db.SelectedAction(); a != dashboard.ActionKeymap {
		t.Errorf("adding the sync entry moved the cursor from Keymaps to %v", a)
	}
	for i := 0; i < 2; i++ { // Quit → the recent note
		db, _ = db.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	db.SetSyncSetup(false)
	if a, id := db.SelectedAction(); a != dashboard.ActionOpenRecent || id != "a.md" {
		t.Errorf("removing the sync entry moved the cursor from the recent note to %v %q", a, id)
	}
}
