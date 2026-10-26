package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// editorPane and previewPane split a frame row at the dividers.
func editorPane(v string) string {
	var out []string
	for _, l := range strings.Split(v, "\n") {
		parts := strings.Split(l, "│")
		if len(parts) >= 3 {
			out = append(out, parts[len(parts)-2])
		}
	}
	return strings.Join(out, "\n")
}

func previewPane(v string) string {
	var out []string
	for _, l := range strings.Split(v, "\n") {
		if i := strings.LastIndex(l, "│"); i >= 0 {
			out = append(out, l[i+len("│"):])
		}
	}
	return strings.Join(out, "\n")
}

func graphWorkspace(t *testing.T) *harness {
	t.Helper()
	store := newTestStore(t)
	for _, n := range [][2]string{
		{"Untitled-1", "first [[Untitled-3]]"},
		{"Untitled-2", "[Untitled-1](Untitled-1.md)"},
		{"Untitled-3", "third"},
		{"Other", "[[Untitled-2]]"},
	} {
		if _, err := store.SaveAs("", n[0], n[1]); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("untitled-2")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.keys(":graph")
	h.key(tea.KeyEnter)
	return h
}

// clickGraph clicks text drawn in the editor pane (where the graph is).
func (h *harness) clickGraph(text string) {
	h.t.Helper()
	for y, l := range strings.Split(h.view(), "\n") {
		parts := strings.Split(l, "│")
		if len(parts) < 3 {
			continue
		}
		ed := parts[len(parts)-2]
		if i := strings.Index(ed, text); i >= 0 {
			x := ansi.StringWidth(strings.Join(parts[:len(parts)-2], "│")) + 1 + ansi.StringWidth(ed[:i])
			h.click(x, y)
			return
		}
	}
	h.t.Fatalf("%q not in the graph:\n%s", text, h.view())
}

func TestGraphTabWithPreviewOfSelection(t *testing.T) {
	h := graphWorkspace(t)
	v := h.view()
	ed := editorPane(v)
	if !strings.Contains(ed, "Untitled-2 · 1 link · 1 backlink · 1 step") || !strings.Contains(ed, "Other ●") || !strings.Contains(ed, "● Untitled-1") {
		t.Fatalf("expected the graph in the editor pane:\n%s", v)
	}
	if strings.Contains(ed, "Untitled-3") {
		t.Fatalf("Untitled-3 is two steps away:\n%s", ed)
	}
	if !strings.Contains(previewPane(v), "Untitled-1") {
		t.Fatalf("the preview should show Untitled-2:\n%s", v)
	}

	h.keys("cq") // keys the graph doesn't use do nothing
	if editorPane(h.view()) != ed {
		t.Fatalf("unbound keys should not change the graph:\n%s", h.view())
	}
	h.keys("l") // the graph's own keys reach it: select, and the preview follows
	if !strings.Contains(previewPane(h.view()), "first Untitled-3") {
		t.Fatalf("l should select Untitled-1 and preview it:\n%s", h.view())
	}
	h.keys("h")

	h.clickGraph("Untitled-1") // select: the preview follows
	if !strings.Contains(previewPane(h.view()), "first Untitled-3") {
		t.Fatalf("the preview should show the selected note:\n%s", h.view())
	}

	h.clickGraph(" − ") // zoom out: links of links
	if ed := editorPane(h.view()); !strings.Contains(ed, "2 steps") {
		t.Fatalf("− should zoom out:\n%s", ed)
	}
	h.send(tea.MouseMsg{X: 60, Y: 10, Button: tea.MouseButtonWheelUp})
	if ed := editorPane(h.view()); !strings.Contains(ed, "1 step") {
		t.Fatalf("the wheel should zoom back in:\n%s", ed)
	}

	h.clickGraph(" centre ")
	if ed := editorPane(h.view()); !strings.Contains(ed, "Untitled-1 · 1 link · 1 backlink") {
		t.Fatalf("centre should re-centre on Untitled-1:\n%s", ed)
	}
	h.clickGraph(" back ")
	if ed := editorPane(h.view()); !strings.Contains(ed, "Untitled-2 · 1 link") {
		t.Fatalf("back should return to Untitled-2:\n%s", ed)
	}
	h.clickGraph(" back ") // nothing further back: stays on Untitled-2
	if ed := editorPane(h.view()); !strings.Contains(ed, "Untitled-2 · 1 link") {
		t.Fatalf("back past the start should do nothing:\n%s", ed)
	}

	h.clickGraph("Untitled-1")
	h.clickGraph("Untitled-1") // a second click opens it; the graph tab stays
	v = h.view()
	if !strings.Contains(v, "1 first [[Untitled-3]]") || !strings.Contains(v, "Graph") {
		t.Fatalf("a second click should open Untitled-1 next to the graph tab:\n%s", v)
	}

	h.keys(" g") // back to the graph, now centred on Untitled-1
	if ed := editorPane(h.view()); !strings.Contains(ed, "Untitled-1 · 1 link · 1 backlink") {
		t.Fatalf("the graph should centre on the note it was opened from:\n%s", ed)
	}
}

func TestGraphPansAndClicks(t *testing.T) {
	h := graphWorkspace(t)
	before := editorPane(h.view())
	x := 45 // a blank spot in the graph, above the map
	h.send(tea.MouseMsg{X: x, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	h.send(tea.MouseMsg{X: x + 8, Y: 5, Action: tea.MouseActionMotion})
	h.send(tea.MouseMsg{X: x + 8, Y: 5, Action: tea.MouseActionRelease})
	if editorPane(h.view()) == before {
		t.Fatal("dragging should pan the map")
	}
	h.clickGraph(" reset ")
	if editorPane(h.view()) != before {
		t.Fatal("reset should put the map back")
	}
	h.clickGraph("Other")
	if !strings.Contains(previewPane(h.view()), "Untitled-2") {
		t.Fatalf("clicking Other should preview it:\n%s", h.view())
	}
	h.clickGraph("Other")
	if v := h.view(); !strings.Contains(v, "1 [[Untitled-2]]") {
		t.Fatalf("a second click should open Other:\n%s", v)
	}
}

// Regression: the graph tab showed the "new" marker of unsaved notes, and
// following a link in the preview from the graph tab claimed the graph had
// unsaved changes.
func TestGraphTabIsNotAnUnsavedNote(t *testing.T) {
	h := graphWorkspace(t)
	if v := h.view(); strings.Contains(v, "Graph new") || strings.Contains(v, "unsaved") || strings.Contains(v, "0 words") {
		t.Fatalf("the graph tab should not look like an unsaved note:\n%s", v)
	}
	h.clickPreview("Untitled-1") // the preview of Untitled-2 links to it
	v := h.view()
	if strings.Contains(v, "unsaved changes") || !strings.Contains(v, "1 first [[Untitled-3]]") {
		t.Fatalf("following a link from the graph tab should just open the note:\n%s", v)
	}
}

// editOutside rewrites a note the way another program would, with a
// modification time well past any save the test made, so even a filesystem
// with coarse timestamps sees the change.
func editOutside(t *testing.T, root, rel, content string) {
	t.Helper()
	writeOutside(t, root, rel, content)
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(root, filepath.FromSlash(rel)), later, later); err != nil {
		t.Fatal(err)
	}
}

// A link added by another program to a note that has no tab reaches the
// graph on the next workspace check, though the tree did not change.
func TestGraphShowsLinkAddedOutsideToAClosedNote(t *testing.T) {
	h := graphWorkspace(t)
	if ed := editorPane(h.view()); !strings.Contains(ed, "Untitled-2 · 1 link · 1 backlink ·") {
		t.Fatalf("unexpected graph:\n%s", ed)
	}
	editOutside(t, h.store.Root(), "Untitled-3.md", "third [[Untitled-2]]")
	h.send(app.CheckWorkspaceMsg{})
	if ed := editorPane(h.view()); !strings.Contains(ed, "Untitled-2 · 1 link · 2 backlinks ·") {
		t.Fatalf("the link added outside should show after the check:\n%s", ed)
	}
}

// A link added by another program to a note open in a tab reloads the tab
// and reaches the graph on the same check.
func TestGraphShowsLinkAddedOutsideToAnOpenNote(t *testing.T) {
	h := graphWorkspace(t) // Untitled-2 is open in a tab, without edits
	h.keys("l")            // select Untitled-1
	editOutside(t, h.store.Root(), "Untitled-2.md", "[Untitled-1](Untitled-1.md) [[Untitled-3]]")
	h.send(app.CheckWorkspaceMsg{})
	if ed := editorPane(h.view()); !strings.Contains(ed, "Untitled-2 · 2 links · 1 backlink ·") {
		t.Fatalf("the open note's new link should show after the check:\n%s", ed)
	}
	if !strings.Contains(previewPane(h.view()), "first Untitled-3") {
		t.Fatalf("the selection's preview should stay:\n%s", h.view())
	}
}

// Saving a note refreshes the links of the whole workspace: the note it
// now links to lists it under its backlinks.
func TestSaveRefreshesBacklinksOfLinkedNote(t *testing.T) {
	h := graphWorkspace(t)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("untitled-3")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.typeAtTop("see [o](Other.md) ")
	h.ex("w")
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("other")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Linked from 1 note") {
		t.Fatalf("Other should list the note just saved under its backlinks:\n%s", v)
	}
}

// A closed note that starts linking to the open one, changed by another
// program, shows up in the open note's backlinks after the check.
func TestPreviewBacklinksFollowLinkAddedOutside(t *testing.T) {
	h := graphWorkspace(t)
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("untitled-3")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	if v := h.view(); !strings.Contains(v, "Linked from 1 note") {
		t.Fatalf("Untitled-3 should have one backlink:\n%s", v)
	}
	editOutside(t, h.store.Root(), "Other.md", "[[Untitled-2]] and [[Untitled-3]]")
	h.send(app.CheckWorkspaceMsg{})
	if v := h.view(); !strings.Contains(v, "Linked from 2 notes") {
		t.Fatalf("the backlink added outside should show after the check:\n%s", v)
	}
}
