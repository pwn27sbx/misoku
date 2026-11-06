package dashboard_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// S3: the home screen lists Learn under Graph on key l, with the levels
// passed, and opening it is an action of its own.
func TestHomeListsLearnUnderGraphOnKeyL(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 50)
	db.SetRecentPages(nil)
	db.SetLearn("Learn 3/7")
	v := db.View()

	row := strings.TrimRight(rowWith(v, "Learn"), " ")
	if row == "" || !strings.HasSuffix(row, " l") {
		t.Fatalf("the home screen should list Learn on key l, row %q:\n%s", row, v)
	}
	if strings.Index(v, "Graph") > strings.Index(v, "Learn") || strings.Index(v, "Learn") > strings.Index(v, "Settings") {
		t.Fatalf("Learn should sit between Graph and Settings:\n%s", v)
	}
	for i := 0; i < 6; i++ { // New Note, Find, Explorer, Themes, Keymaps, Graph → Learn
		db, _ = db.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if a, _ := db.SelectedAction(); a != dashboard.ActionLearn {
		t.Fatalf("the seventh entry should be Learn, got %v", a)
	}
}

// S3: the entry says how many levels pass, and a check once all of them do.
func TestLearnEntryShowsTheProgress(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 50)
	db.SetRecentPages(nil)
	for _, label := range []string{"Learn 0/7", "Learn 3/7", "Learn 7/7 ✓"} {
		db.SetLearn(label)
		if v := db.View(); rowWith(v, label) == "" {
			t.Errorf("the entry should read %q:\n%s", label, v)
		}
	}
	db.SetLearn("")
	if v := db.View(); strings.Contains(v, "Learn") {
		t.Errorf("an empty label hides the entry:\n%s", v)
	}
}

// S3: 13 rows are enough to see the entry; Home does not scroll.
func TestLearnEntryIsVisibleFromThirteenRows(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 13)
	db.SetRecentPages(nil)
	db.SetLearn("Learn 3/7")
	if v := db.View(); rowWith(v, "Learn 3/7") == "" {
		t.Fatalf("the Learn entry should show at 13 rows:\n%s", v)
	}
}

// Adding or removing the entry keeps the cursor on the item it was on.
func TestLearnEntryKeepsTheCursorOnItsItem(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 50)
	db.SetRecentPages(nil)
	for i := 0; i < 5; i++ { // → Graph
		db, _ = db.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if a, _ := db.SelectedAction(); a != dashboard.ActionGraph {
		t.Fatalf("setup: cursor on %v, want Graph", a)
	}
	db.SetLearn("Learn 1/7")
	if a, _ := db.SelectedAction(); a != dashboard.ActionGraph {
		t.Errorf("adding Learn moved the cursor from Graph to %v", a)
	}
	db.SetLearn("")
	if a, _ := db.SelectedAction(); a != dashboard.ActionGraph {
		t.Errorf("removing Learn moved the cursor from Graph to %v", a)
	}
	// The entry below the removed one keeps its place too.
	db.SetLearn("Learn 1/7")
	for i := 0; i < 2; i++ { // Learn → Settings
		db, _ = db.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if a, _ := db.SelectedAction(); a != dashboard.ActionSettings {
		t.Fatalf("setup: cursor on %v, want Settings", a)
	}
	db.SetLearn("")
	if a, _ := db.SelectedAction(); a != dashboard.ActionSettings {
		t.Errorf("removing Learn moved the cursor from Settings to %v", a)
	}
}
