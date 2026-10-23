package dashboard_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/pwn27sbx/misoku/internal/dashboard"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// rowWith returns the first stripped row of v that contains s.
func rowWith(v, s string) string {
	for _, l := range strings.Split(v, "\n") {
		if strings.Contains(ansi.Strip(l), s) {
			return l
		}
	}
	return ""
}

func TestHomeListsGraphOnKeyG(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 50)
	db.SetRecentPages(workspacePages())
	v := ansi.Strip(db.View())

	row := strings.TrimRight(rowWith(v, "Graph"), " ")
	if row == "" || !strings.HasSuffix(row, " g") {
		t.Fatalf("the home screen should list Graph on key g, row %q:\n%s", row, v)
	}
	// Graph comes right after Keymaps.
	if strings.Index(v, "Keymaps") > strings.Index(v, "Graph") || strings.Index(v, "Graph") > strings.Index(v, "Quit") {
		t.Fatalf("Graph should sit between Keymaps and Quit:\n%s", v)
	}
	for i := 0; i < 5; i++ {
		db, _ = db.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if a, _ := db.SelectedAction(); a != dashboard.ActionGraph {
		t.Fatalf("the sixth entry should be Graph, got %v", a)
	}
}

func TestHomeFolderSectionIsCalledFolders(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 50)
	db.SetRecentPages(workspacePages())
	v := ansi.Strip(db.View())
	if rowWith(v, "FOLDERS") == "" {
		t.Fatalf("the folder section should be titled FOLDERS:\n%s", v)
	}
	for _, l := range strings.Split(v, "\n") {
		if strings.TrimSpace(l) == "Projects" || strings.HasSuffix(strings.TrimSpace(l), " Projects") {
			t.Fatalf("no section is called Projects any more, row %q:\n%s", l, v)
		}
	}
}

// fgCode is the true-colour SGR parameter for a foreground hex colour.
func fgCode(c lipgloss.Color) string {
	h := strings.TrimPrefix(string(c), "#")
	r, _ := strconv.ParseUint(h[0:2], 16, 8)
	g, _ := strconv.ParseUint(h[2:4], 16, 8)
	b, _ := strconv.ParseUint(h[4:6], 16, 8)
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

func TestHomeSectionTitlesAreGreySmallCapsWithoutIcon(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	th := theme.DefaultTheme()
	db := dashboard.New(th)
	db.SetSize(100, 50)
	db.SetRecentPages(workspacePages())
	raw := db.View()

	for _, title := range []string{"RECENT", "FOLDERS"} {
		row := rowWith(raw, title)
		if got := strings.TrimSpace(ansi.Strip(row)); got != title {
			t.Fatalf("section title should be just %q, without an icon, got %q", title, got)
		}
		if !strings.Contains(row, fgCode(th.GreyFg)) {
			t.Fatalf("section title %s should be grey (%s), row %q", title, th.GreyFg, row)
		}
	}
	if v := ansi.Strip(raw); strings.Contains(v, "Recent") {
		t.Fatalf("the old Recent title is still there:\n%s", v)
	}
}

func TestHomeBottomLinePointsToSpaceAndHelp(t *testing.T) {
	db := dashboard.New(theme.DefaultTheme())
	db.SetSize(100, 50)
	db.SetRecentPages(workspacePages())
	v := ansi.Strip(db.View())

	const hint = "SPC opens every other command · ? shows them all"
	at := strings.Index(v, hint)
	if at < 0 {
		t.Fatalf("the home screen should end with the %q line:\n%s", hint, v)
	}
	if footer := strings.Index(v, "notes  ·"); footer < 0 || footer > at {
		t.Fatalf("the hint goes below the notes footer:\n%s", v)
	}
}
