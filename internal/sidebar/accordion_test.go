package sidebar_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/sidebar"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// accordionPages is a tree with two sibling folders at the root (A, B), a
// subfolder A1 inside A, and a loose root note.
func accordionPages() []core.Page {
	return []core.Page{
		{ID: "A", Title: "A", IsFolder: true},
		{ID: "A/a.md", Title: "a", ParentID: "A"},
		{ID: "A/A1", Title: "A1", IsFolder: true, ParentID: "A"},
		{ID: "A/A1/x.md", Title: "x", ParentID: "A/A1"},
		{ID: "B", Title: "B", IsFolder: true},
		{ID: "B/b.md", Title: "b", ParentID: "B"},
		{ID: "c.md", Title: "c"},
	}
}

func newAccordionSidebar() sidebar.Model {
	sb := sidebar.New(theme.DefaultTheme())
	sb.SetSize(28, 25)
	sb.SetPages(accordionPages())
	sb.SetFocused(true)
	return sb
}

// rowOf returns the visible row index of id, or -1.
func rowOf(sb sidebar.Model, id string) int {
	for i, it := range sb.VisibleItems() {
		if it.Page.ID == id {
			return i
		}
	}
	return -1
}

func isOpen(sb sidebar.Model, id string) bool {
	for _, it := range sb.VisibleItems() {
		if it.Page.ID == id {
			return it.Expanded
		}
	}
	return false
}

// moveTo walks the cursor to id with the keyboard, from the top.
func moveTo(t *testing.T, sb sidebar.Model, id string) sidebar.Model {
	t.Helper()
	sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	for range sb.VisibleItems() {
		if p, ok := sb.SelectedPage(); ok && p.ID == id {
			return sb
		}
		sb, _ = sb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}
	if p, ok := sb.SelectedPage(); ok && p.ID == id {
		return sb
	}
	t.Fatalf("row %q is not visible", id)
	return sb
}

func keyOpener(msg tea.KeyMsg) func(*testing.T, sidebar.Model, string) sidebar.Model {
	return func(t *testing.T, sb sidebar.Model, id string) sidebar.Model {
		sb = moveTo(t, sb, id)
		sb, _ = sb.Update(msg)
		return sb
	}
}

func runeKey(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

// openers lists every way the explorer can open a folder.
var openers = []struct {
	name string
	open func(*testing.T, sidebar.Model, string) sidebar.Model
}{
	{"key l", keyOpener(runeKey('l'))},
	{"key enter", keyOpener(tea.KeyMsg{Type: tea.KeyEnter})},
	{"key z", keyOpener(runeKey('z'))},
	{"key o", keyOpener(runeKey('o'))},
	{"key a (new sub-note)", keyOpener(runeKey('a'))},
	{"click", func(t *testing.T, sb sidebar.Model, id string) sidebar.Model {
		row := rowOf(sb, id)
		if row < 0 {
			t.Fatalf("row %q is not visible", id)
		}
		// Rows start at y=4 (title, spacer, find button, spacer).
		sb, _ = sb.Update(tea.MouseMsg{X: 5, Y: 4 + row, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		return sb
	}},
	{"Expand", func(_ *testing.T, sb sidebar.Model, id string) sidebar.Model {
		sb.Expand(id)
		return sb
	}},
	{"SetSelectedID", func(_ *testing.T, sb sidebar.Model, id string) sidebar.Model {
		// Revealing a note opens the folder that holds it.
		sb.SetSelectedID(map[string]string{"A": "A/a.md", "B": "B/b.md", "A/A1": "A/A1/x.md"}[id])
		return sb
	}},
}

func TestAccordionOpeningFolderClosesSiblings(t *testing.T) {
	for _, op := range openers {
		t.Run(op.name, func(t *testing.T) {
			sb := op.open(t, newAccordionSidebar(), "A")
			if !isOpen(sb, "A") {
				t.Fatal("A should open")
			}
			sb = op.open(t, sb, "B")
			if !isOpen(sb, "B") {
				t.Fatal("B should open")
			}
			if isOpen(sb, "A") {
				t.Error("opening B should close its sibling A")
			}
		})
	}
}

func TestAccordionOpeningSubfolderKeepsParentOpen(t *testing.T) {
	for _, op := range openers {
		t.Run(op.name, func(t *testing.T) {
			sb := op.open(t, newAccordionSidebar(), "A")
			sb = op.open(t, sb, "A/A1")
			if !isOpen(sb, "A/A1") {
				t.Fatal("A1 should open")
			}
			if !isOpen(sb, "A") {
				t.Error("opening subfolder A1 should keep its parent A open")
			}
		})
	}
}

func TestAccordionOffKeepsSeveralFoldersOpen(t *testing.T) {
	for _, op := range openers {
		t.Run(op.name, func(t *testing.T) {
			sb := newAccordionSidebar()
			sb.SetAccordion(false)
			sb = op.open(t, sb, "A")
			sb = op.open(t, sb, "B")
			if !isOpen(sb, "A") || !isOpen(sb, "B") {
				t.Errorf("with the accordion off both folders stay open: A=%v B=%v", isOpen(sb, "A"), isOpen(sb, "B"))
			}
		})
	}
}

func TestAccordionRevealingNoteLeavesOnlyItsAncestorChainOpen(t *testing.T) {
	sb := newAccordionSidebar()
	sb.Expand("B")
	sb.SetSelectedID("A/A1/x.md")
	if !isOpen(sb, "A") || !isOpen(sb, "A/A1") {
		t.Fatalf("the note's folder chain should open: A=%v A1=%v", isOpen(sb, "A"), isOpen(sb, "A/A1"))
	}
	if isOpen(sb, "B") {
		t.Error("revealing a note inside A should close sibling B")
	}
	if p, _ := sb.SelectedPage(); p.ID != "A/A1/x.md" {
		t.Errorf("cursor should land on the revealed note, got %q", p.ID)
	}
}

func TestAccordionClosingFolderStillWorks(t *testing.T) {
	sb := newAccordionSidebar()
	sb = keyOpener(runeKey('z'))(t, sb, "A")
	sb, _ = sb.Update(runeKey('z'))
	if isOpen(sb, "A") {
		t.Error("z on an open folder should close it")
	}
}

// Closing an open sibling above the cursor removes rows above it; the cursor
// must stay on the folder the user opened, not on whatever row slid into its
// old index.
func TestAccordionCursorStaysOnOpenedFolder(t *testing.T) {
	for _, op := range openers {
		if op.name == "SetSelectedID" {
			continue // it selects the revealed note, not the folder
		}
		t.Run(op.name, func(t *testing.T) {
			sb := op.open(t, newAccordionSidebar(), "A")
			sb = moveTo(t, sb, "B")
			sb = op.open(t, sb, "B")
			if p, _ := sb.SelectedPage(); p.ID != "B" {
				t.Errorf("cursor should stay on the opened folder B, got %q", p.ID)
			}
		})
	}
}
