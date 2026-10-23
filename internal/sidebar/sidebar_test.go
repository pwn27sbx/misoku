package sidebar_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/keymap"
	"github.com/pwn27sbx/misoku/internal/sidebar"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSidebarSettersAndAccessors(t *testing.T) {
	th := theme.DefaultTheme()
	sb := sidebar.New(th)
	sb.SetSize(28, 25)
	sb.SetPages([]core.Page{
		{ID: "work/a.md", Title: "a", ParentID: "work"},
		{ID: "work", Title: "work", IsFolder: true},
	})
	sb.SetTheme(th)
	sb.SetWorkspaceName("notes")
	sb.SetActiveID("work/a.md")
	sb.SetModified(map[string]bool{"work/a.md": true})
	sb.SetFocused(false)
	if sb.IsRenaming() || sb.IsBusy() {
		t.Error("expected not renaming/busy")
	}
	if p, ok := sb.SelectedPage(); ok && p.ID == "" {
		t.Error("SelectedPage returned empty page")
	}
	_ = sb.ContextParentID()
	sb.Scroll(1)
	sb.View()
}

func TestSidebarComponent(t *testing.T) {
	th := theme.DefaultTheme()
	sb := sidebar.New(th)
	sb.SetSize(28, 25)

	pages := []core.Page{
		{ID: "p1", Title: "Code Turtle"},
		{ID: "p2", Title: "Tasks (1)", ParentID: "p1"},
		{ID: "p3", Title: "Project Manager"},
	}
	sb.SetPages(pages)
	sb.SetFocused(true)

	// Folders start collapsed, like nvim-tree; selecting a nested page
	// reveals it.
	for _, it := range sb.VisibleItems() {
		if it.Page.Title == "Tasks (1)" {
			t.Fatalf("expected child hidden while parent is collapsed")
		}
	}
	sb.SetSelectedID("p2")

	view := sb.View()
	if !strings.Contains(view, "Code Turtle") {
		t.Errorf("expected view to contain parent 'Code Turtle', got %q", view)
	}
	if !strings.Contains(view, "Tasks (1)") {
		t.Errorf("expected view to contain child 'Tasks (1)', got %q", view)
	}
	if strings.Contains(view, "RECENTS") {
		t.Errorf("expected view NOT to contain RECENTS header, got %q", view)
	}
	if !strings.Contains(view, "Find note") {
		t.Errorf("expected view to contain the find button, got %q", view)
	}

	// Test collapsing folder
	sb.SetSelectedID("p1")
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	itemsAfterCollapse := sb.VisibleItems()
	for _, it := range itemsAfterCollapse {
		if it.Page.Title == "Tasks (1)" {
			t.Errorf("expected collapsed folder to hide child item from visible tree, but found %v", it)
		}
	}

	// Test expanding folder again
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	itemsAfterExpand := sb.VisibleItems()
	found := false
	for _, it := range itemsAfterExpand {
		if it.Page.Title == "Tasks (1)" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected re-expanded folder to show child 'Tasks (1)' in visible tree, but not found")
	}
}

func TestSidebarEditableElements(t *testing.T) {
	th := theme.DefaultTheme()
	sb := sidebar.New(th)
	sb.SetSize(28, 25)

	pages := []core.Page{
		{ID: "p1", Title: "Original Title"},
	}
	sb.SetPages(pages)

	// Press 'r' to edit/rename selected element
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !sb.IsRenaming() {
		t.Fatal("expected sidebar to be in renaming mode after 'r'")
	}

	// Simulate pressing 'enter' or typing and saving
	sb, cmd := sb.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if sb.IsRenaming() {
		t.Errorf("expected renaming to finish after enter")
	}
	if cmd == nil {
		t.Errorf("expected cmd emitting PageUpdatedMsg on rename commit")
	}
	msg := cmd()
	updatedMsg, ok := msg.(core.PageUpdatedMsg)
	if !ok || updatedMsg.Page.Title != "Original Title" {
		t.Errorf("expected PageUpdatedMsg with title 'Original Title', got %v", msg)
	}
}

func TestSidebarBackslashOpensFinderAndClickKeepsFocus(t *testing.T) {
	sb := sidebar.New(theme.DefaultTheme())
	sb.SetSize(28, 25)
	sb.SetPages([]core.Page{
		{ID: "a.md", Title: "a"},
		{ID: "b.md", Title: "b"},
	})
	sb.SetFocused(true)

	_, cmd := sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	if cmd == nil {
		t.Fatal("expected a command from '\\'")
	}
	if _, ok := cmd().(core.FindRequestMsg); !ok {
		t.Fatal("expected '\\' to request the global finder")
	}

	// Rows start at y=4 (title, spacer, find button, spacer). Click b.md.
	sb, cmd = sb.Update(tea.MouseMsg{X: 5, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if cmd == nil {
		t.Fatal("expected click on a note to open it")
	}
	sel, ok := cmd().(core.PageSelectedMsg)
	if !ok || sel.ID != "b.md" || !sel.KeepFocus {
		t.Fatalf("expected quiet open of b.md, got %#v", sel)
	}
	if p, _ := sb.SelectedPage(); p.ID != "b.md" {
		t.Fatalf("cursor should follow the click, got %q", p.ID)
	}
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p, _ := sb.SelectedPage(); p.ID != "a.md" {
		t.Fatalf("keyboard should keep working after a click, got %q", p.ID)
	}
}

func TestFindHintShowsTheEffectiveKey(t *testing.T) {
	sb := sidebar.New(theme.DefaultTheme())
	sb.SetSize(28, 10)
	if v := sb.View(); !strings.Contains(v, `\ `) {
		t.Fatalf("default find hint should be \\:\n%s", v)
	}

	km, _ := keymap.New(map[string][]string{"explorer.find": {"ctrl+f"}})
	sb.SetKeys(km)
	if v := sb.View(); !strings.Contains(v, "Ctrl+F") || strings.Contains(v, `\`) {
		t.Fatalf("find hint should follow explorer.find:\n%s", v)
	}
}
