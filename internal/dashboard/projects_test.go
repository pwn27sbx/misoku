package dashboard_test

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/theme"
)

func workspacePages() []core.Page {
	now := time.Now()
	return []core.Page{
		{ID: "Misoku", Title: "Misoku", IsFolder: true},
		{ID: "Trabajo", Title: "Trabajo", IsFolder: true},
		{ID: "Trabajo/Clientes", Title: "Clientes", ParentID: "Trabajo", IsFolder: true},
		{ID: "Misoku/Plan.md", Title: "Plan", ParentID: "Misoku", UpdatedAt: now},
		{ID: "Trabajo/Clientes/a.md", Title: "a", ParentID: "Trabajo/Clientes", UpdatedAt: now.Add(-time.Minute)},
		{ID: "suelta.md", Title: "suelta", UpdatedAt: now.Add(-time.Hour)},
	}
}

func TestProjectsListTheTopLevelFolders(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 50)
	db.SetRecentPages(workspacePages())
	v := ansi.Strip(db.View())

	if !strings.Contains(v, "Projects") {
		t.Fatalf("the home screen should have a Projects section:\n%s", v)
	}
	got := db.Projects()
	if len(got) != 2 || got[0].ID != "Misoku" || got[1].ID != "Trabajo" {
		t.Fatalf("projects are the top-level folders, alphabetically, got %+v", got)
	}
	// Keys continue after the three recent notes: 4 and 5.
	for _, want := range [][2]string{{"Misoku", "4"}, {"Trabajo", "5"}} {
		row := ""
		for _, l := range strings.Split(v, "\n") {
			if strings.Contains(l, want[0]) && !strings.Contains(l, ".md") {
				row = l
			}
		}
		if !strings.HasSuffix(strings.TrimRight(row, " "), want[1]) {
			t.Fatalf("project %s should be on key %s, row %q", want[0], want[1], row)
		}
	}
}

func TestNoProjectsSectionWithoutFolders(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 50)
	db.SetRecentPages([]core.Page{{ID: "a.md", Title: "a", UpdatedAt: time.Now()}})
	if v := ansi.Strip(db.View()); strings.Contains(v, "Projects") {
		t.Fatalf("no folders, no Projects section:\n%s", v)
	}
}

func TestSmallScreensShowTheMisokuName(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(80, 18) // too short for the big banner
	db.SetRecentPages(workspacePages())
	v := ansi.Strip(db.View())
	if strings.Contains(v, "T S U Z U R I") || !strings.Contains(v, "M I S O K U") {
		t.Fatalf("the small-screen title should say MISOKU:\n%s", v)
	}
}
