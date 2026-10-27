package globalgraph

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// cellOf is the screen cell of node id's dot in m's view.
func cellOf(t *testing.T, m Model, id string) tcell {
	t.Helper()
	x, y := m.screen(id)
	rows := screen(m.View())
	return rows[y+1][x]
}

// hintRow is the last row of m's view, without colours.
func hintRow(m Model) string {
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	return lines[len(lines)-1]
}

func titleRow(m Model) string { return strings.Split(ansi.Strip(m.View()), "\n")[0] }

const nodeGlyphs = string(glyphLeaf) + string(glyphLinked) + string(glyphHub) + string(glyphUnresolved)

// The filter greys out the nodes it doesn't match and leaves every node
// drawn where it was.
func TestFilterDimsNonMatchesWithoutMovingThem(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	m := newModel(t, 120, 20)
	before := map[string]tcell{}
	for _, n := range m.l.Nodes {
		before[n.ID] = cellOf(t, m, n.ID)
	}
	m, _ = press(m, "/", "PLAN")
	for _, n := range m.l.Nodes {
		if !strings.ContainsRune(nodeGlyphs, before[n.ID].r) {
			continue // under a name at the overview
		}
		got := cellOf(t, m, n.ID)
		if got.r != before[n.ID].r {
			t.Errorf("%s: dot %q, want %q where it was", n.ID, got.r, before[n.ID].r)
		}
		want := rgb(th.Grey)
		switch n.ID {
		case "work/Plan.md":
			want = rgb(th.Blue) // a link of the selection that matches keeps its colour
		case "Home.md":
			want = rgb(th.Yellow) // the selection is never greyed
		}
		if got.fg != want {
			t.Errorf("%s: dot colour %s, want %s", n.ID, got.fg, want)
		}
	}
	if !strings.Contains(titleRow(m), `filter "PLAN": 1 match`) {
		t.Fatalf("the title should name the filter: %q", titleRow(m))
	}
}

// Enter jumps to the best match: a name starting with the text, then a
// name holding it, then a path holding it; then the shortest name, then
// the ID.
func TestFilterEnterJumpsToTheBestMatch(t *testing.T) {
	cases := []struct{ query, want string }{
		{"in", "Inbox.md"},          // name prefix beats Meeting holding "in"
		{"g", "study/Go.md"},        // prefix beats Meeting
		{"o", "Orphan.md"},          // a name starting with it beats the shorter Go and Home
		{"s", "study/Rust.md"},      // a name holding it beats Go, whose path only holds it
		{"a", "work/Plan.md"},       // no prefix: the shortest of Plan, Tasks, Ideas, Orphan…
		{"notes", "study/Notes.md"}, // two prefixes of one length: the smaller ID
		{"work/", "work/Plan.md"},   // the path only: the shortest name
		{"ノート", "study/日本語のノート.md"}, // wide text
		{"IDEAS", "Ideas"},          // case-insensitive; unresolved targets match too
	}
	for _, c := range cases {
		m := newModel(t, 120, 20)
		m, msg := press(m, "/", c.query, "enter")
		if m.Selected() != c.want || msg != (SelectMsg{ID: c.want}) {
			t.Errorf("%q: selected %q (%#v), want %q", c.query, m.Selected(), msg, c.want)
		}
		if m.Typing() || m.filter != c.query {
			t.Errorf("%q: enter should close the prompt and keep the filter, typing %v filter %q", c.query, m.Typing(), m.filter)
		}
	}
	m := newModel(t, 120, 20)
	if m, msg := press(m, "/", "zzz", "enter"); m.Selected() != "Home.md" || msg != nil || m.Typing() {
		t.Fatalf("no match: the selection should stay, got %q %#v", m.Selected(), msg)
	}
}

// esc in the prompt clears the filter; out of it, esc clears the filter
// first and then goes back, as before.
func TestFilterEscClearsThenGoesBack(t *testing.T) {
	m := newModel(t, 120, 20)
	m, _ = press(m, "/", "pl", "esc")
	if m.Typing() || m.filter != "" || m.Selected() != "Home.md" {
		t.Fatalf("esc in the prompt: typing %v filter %q selected %q", m.Typing(), m.filter, m.Selected())
	}
	m, _ = press(m, "/", "plan", "enter")
	if m.Selected() != "work/Plan.md" {
		t.Fatalf("enter should jump to Plan, got %q", m.Selected())
	}
	if m, _ = press(m, "esc"); m.filter != "" || m.Selected() != "work/Plan.md" {
		t.Fatalf("the first esc clears the filter only: filter %q selected %q", m.filter, m.Selected())
	}
	if m, _ = press(m, "esc"); m.Selected() != "Home.md" {
		t.Fatalf("the next esc goes back, got %q", m.Selected())
	}
}

// While the prompt is open every letter, the space and the graph's own
// keys are text.
func TestFilterPromptTakesEveryKeyAsText(t *testing.T) {
	m := newModel(t, 120, 20)
	m, _ = press(m, "/", "o", "u", "n", "j", "+", "0", "?", "/")
	m, _ = send(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m, _ = press(m, "x", "tab")
	m, _ = send(m, tea.KeyMsg{Type: tea.KeyBackspace})
	if m.filter != "ounj+0?/ " || !m.Typing() {
		t.Fatalf("filter %q typing %v, want %q and the prompt open", m.filter, m.Typing(), "ounj+0?/ ")
	}
	if m.Selected() != "Home.md" || m.zoom != ZoomOverview || m.hideOrphans || m.hideUnresolved {
		t.Fatalf("keys typed in the prompt should not act: selected %q zoom %d", m.Selected(), m.zoom)
	}
	if !strings.Contains(hintRow(m), "/ ounj+0?/ ") {
		t.Fatalf("the prompt should show the text: %q", hintRow(m))
	}
}

// The prompt fits every pane the map is drawn in, keeping the end of a
// long text in sight.
func TestFilterPromptFitsNarrowPanes(t *testing.T) {
	long := "a filter text far longer than the pane"
	for _, w := range []int{MinWidth, 35, 50, 80} {
		m := newModel(t, w, MinHeight)
		m, _ = press(m, "/", long)
		checkSize(t, m.View(), w, MinHeight, "prompt")
		hint := hintRow(m)
		if !strings.Contains(hint, "/ ") || !strings.Contains(hint, "the pane") {
			t.Errorf("width %d: the prompt should show / and the end of the text: %q", w, hint)
		}
		m, _ = press(m, "esc")
		m, _ = press(m, "/", "pl")
		if hint := hintRow(m); !strings.Contains(hint, "/ pl") || !strings.Contains(hint, "esc") {
			t.Errorf("width %d: a short text leaves room for the keys: %q", w, hint)
		}
	}
}

// o and u hide orphans and unresolved targets and show them again; the
// other nodes stay where they were and the title keeps the vault's counts.
func TestToggleHidesAndShowsOrphansAndUnresolved(t *testing.T) {
	m := newModel(t, 120, 20)
	before := m.View()
	orphan, ideas, home := cellOf(t, m, "Orphan.md"), cellOf(t, m, "Ideas"), cellOf(t, m, "Home.md")
	if orphan.r == ' ' || ideas.r != glyphUnresolved {
		t.Fatalf("both should be drawn before the toggles: %q %q", orphan.r, ideas.r)
	}
	m, _ = press(m, "o", "u")
	for _, id := range []string{"Orphan.md", "Ideas"} {
		x, y := m.screen(id)
		if got := cellOf(t, m, id).r; strings.ContainsRune(nodeGlyphs, got) {
			t.Errorf("%s should be hidden, its cell shows %q", id, got)
		}
		if _, ok := m.r.nodeAt(m.frame(), x, y); ok {
			t.Errorf("%s should be hidden, its cell still belongs to a node", id)
		}
	}
	if got := cellOf(t, m, "Home.md").r; got != home.r {
		t.Errorf("Home should stay where it was, its cell shows %q", got)
	}
	want := "11 notes · 12 links · 1 orphan hidden · 1 unresolved hidden"
	if !strings.Contains(titleRow(m), want) {
		t.Errorf("title %q, want %q", titleRow(m), want)
	}
	m, _ = press(m, "o", "u")
	if m.View() != before {
		t.Fatal("pressing o and u again should show the map as before")
	}
}

// Hiding unresolved targets hides their links too.
func TestToggleUnresolvedHidesTheirLinks(t *testing.T) {
	m := newModel(t, 120, 20)
	m, _ = press(m, "+") // every link is drawn from the medium zoom on
	// Inbox sits right above Ideas: the cell between them is their link.
	x, y := m.screen("Inbox.md")
	between := func() rune { return screen(m.View())[y+2][x].r }
	if between() != '⡇' {
		t.Fatalf("the link Inbox → Ideas should be drawn, got %q", between())
	}
	m, _ = press(m, "u")
	if got := between(); got != ' ' {
		t.Fatalf("u should hide the link to Ideas, got %q", got)
	}
}

// Hidden nodes are skipped by h j k l, n and N and the mouse.
func TestHiddenNodesAreSkipped(t *testing.T) {
	m := newModel(t, 120, 20)
	if m, _ := press(m, "o", "j"); m.Selected() != "Ideas" {
		t.Errorf("j from Home with orphans hidden: %q, want Ideas", m.Selected())
	}
	if m, _ := press(m, "u", "l", "j"); m.Selected() != "Orphan.md" {
		t.Errorf("j from Inbox with unresolved hidden: %q, want Orphan", m.Selected())
	}
	m, _ = press(m, "u", "l") // Inbox: links Home and Ideas
	for range 3 {
		if m, _ = press(m, "n"); m.Selected() == "Ideas" {
			t.Fatal("n should skip the hidden Ideas")
		}
	}
	m = newModel(t, 120, 20)
	x, y := m.screen("Orphan.md")
	m, _ = press(m, "o")
	if m, msg := click(m, x, y+1); m.Selected() != "Home.md" || msg != nil {
		t.Fatalf("a click on the hidden Orphan should select nothing, got %q", m.Selected())
	}
}

// When a toggle hides the selection it moves to the open note, or to the
// nearest node still shown.
func TestHidingTheSelectionMovesIt(t *testing.T) {
	m := newModel(t, 120, 20)
	m, _ = press(m, "j") // Orphan
	if m, msg := press(m, "o"); m.Selected() != "Home.md" || msg != (SelectMsg{ID: "Home.md"}) {
		t.Fatalf("hiding Orphan should select the open note, got %q %#v", m.Selected(), msg)
	}
	m.SetOpenNote("Orphan.md")
	m, msg := press(m, "o")
	if m.Selected() != "Ideas" || msg != (SelectMsg{ID: "Ideas"}) {
		t.Fatalf("with the open note hidden too, the nearest node: got %q, want Ideas", m.Selected())
	}
	if m, _ = press(m, "u"); m.Selected() != "Inbox.md" {
		t.Fatalf("hiding Ideas: got %q, want the nearest, Inbox", m.Selected())
	}
	checkSelVisible(t, m, "after hiding")
}

// The hint row has buttons for the filter and the toggles where they fit.
func TestFilterAndToggleButtons(t *testing.T) {
	m := newModel(t, 200, 20)
	for _, b := range []string{" / filter ", " o orphans ", " u unresolved "} {
		if !strings.Contains(hintRow(m), b) {
			t.Fatalf("hint lacks %q: %q", b, hintRow(m))
		}
	}
	m, _ = clickOn(t, m, " o orphans ")
	m, _ = clickOn(t, m, " u unresolved ")
	if !m.hideOrphans || !m.hideUnresolved {
		t.Fatal("the toggle buttons should hide orphans and unresolved targets")
	}
	if m, _ = clickOn(t, m, " / filter "); !m.Typing() {
		t.Fatal("the filter button should open the prompt")
	}
	m = newModel(t, 70, 20)
	for _, b := range []string{" / ", " o ", " u "} {
		if !strings.Contains(hintRow(m), b) {
			t.Fatalf("width 70: hint lacks %q: %q", b, hintRow(m))
		}
	}
}

// A map that becomes empty while the prompt is open closes the prompt:
// the empty map takes no keys, so nothing else could close it, and the app
// would keep typing every key into it.
func TestEmptyMapClosesThePrompt(t *testing.T) {
	m := newModel(t, 120, 20)
	m, _ = press(m, "/", "pl")
	m.SetGraph(Graph{})
	m, _ = press(m, "esc")
	if m.Typing() {
		t.Fatal("the prompt should be closed on an empty map")
	}
	m.SetGraph(vault())
	if m.Typing() || m.filter != "" {
		t.Fatalf("the map coming back should not reopen the prompt: typing %v filter %q", m.Typing(), m.filter)
	}
}

// esc skips the selections a toggle hides without forgetting them: once
// they show again, esc goes back to them.
func TestBackSkipsHiddenSelectionsWithoutDroppingThem(t *testing.T) {
	m := newModel(t, 120, 20)
	m, _ = press(m, "j", "l") // Home → Orphan → Ideas
	if m.Selected() != "Ideas" {
		t.Fatalf("setup: selected %q, want Ideas", m.Selected())
	}
	if m, _ = press(m, "o", "esc"); m.Selected() != "Home.md" {
		t.Fatalf("esc should skip the hidden Orphan, got %q", m.Selected())
	}
	m, msg := press(m, "o", "esc")
	if m.Selected() != "Orphan.md" || msg != (SelectMsg{ID: "Orphan.md"}) {
		t.Fatalf("with orphans shown again esc should go back to Orphan, got %q %#v", m.Selected(), msg)
	}
}

// When a toggle hides every node nothing is selected; showing them again
// selects the open note.
func TestShowingNodesAgainSelectsTheOpenNote(t *testing.T) {
	m := New(theme.DefaultTheme())
	m.SetSize(80, 20)
	m.SetFocused(true)
	m.SetGraph(Graph{Nodes: []Node{{ID: "A.md"}, {ID: "B.md"}}}) // two orphans
	m.SetOpenNote("A.md")
	if m, _ = press(m, "o"); m.Selected() != "" {
		t.Fatalf("with every node hidden nothing should be selected, got %q", m.Selected())
	}
	m, msg := press(m, "o")
	if m.Selected() != "A.md" || msg != (SelectMsg{ID: "A.md"}) {
		t.Fatalf("showing the nodes again should select the open note, got %q %#v", m.Selected(), msg)
	}
}
