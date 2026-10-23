package graph

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// sample: Hub links to Docker (which links back) and AWS; Study links to
// Hub; Index links to Study; AWS links to EC2 and 日本語; Lonely is alone.
func sample() ([]core.Page, []core.Edge) {
	var notes []core.Page
	for _, n := range []string{"Hub", "Docker", "AWS", "Study", "Index", "EC2", "日本語", "Lonely"} {
		notes = append(notes, core.Page{ID: n + ".md", Title: n})
	}
	e := func(a, b string) core.Edge { return core.Edge{From: a + ".md", To: b + ".md"} }
	edges := []core.Edge{
		e("Hub", "Docker"), e("Docker", "Hub"), e("Hub", "AWS"), e("Study", "Hub"),
		e("Index", "Study"), e("AWS", "EC2"), e("AWS", "日本語"), e("Hub", "Missing"),
	}
	return notes, edges
}

func newSample(t *testing.T, w, h int) Model {
	t.Helper()
	notes, edges := sample()
	m := New(theme.DefaultTheme())
	m.SetSize(w, h)
	m.SetFocused(true)
	m.SetLinks(notes, edges)
	m.SetCentre("Hub.md")
	return m
}

func update(m Model, msg tea.Msg) (Model, tea.Msg) {
	m, cmd := m.Update(msg)
	if cmd == nil {
		return m, nil
	}
	return m, cmd()
}

func clickAt(m Model, x, y int) (Model, tea.Msg) {
	return update(m, tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
}

// clickText clicks the first place text is drawn.
func clickText(t *testing.T, m Model, text string) (Model, tea.Msg) {
	t.Helper()
	for y, l := range strings.Split(drawn(m), "\n") {
		if i := strings.Index(l, text); i >= 0 {
			return clickAt(m, ansi.StringWidth(l[:i]), y)
		}
	}
	t.Fatalf("%q not drawn:\n%s", text, drawn(m))
	return m, nil
}

func wheel(m Model, b tea.MouseButton) Model {
	m, _ = update(m, tea.MouseMsg{X: 10, Y: 5, Button: b})
	return m
}

func drawn(m Model) string { return ansi.Strip(m.View()) }

func assertSize(t *testing.T, m Model, w, h int) {
	t.Helper()
	lines := strings.Split(m.View(), "\n")
	if len(lines) != h {
		t.Fatalf("rows = %d, want %d", len(lines), h)
	}
	for i, l := range lines {
		if got := ansi.StringWidth(l); got != w {
			t.Fatalf("row %d is %d wide, want %d: %q", i, got, w, ansi.Strip(l))
		}
	}
}

func TestOneStepShowsLinksAndBacklinks(t *testing.T) {
	m := newSample(t, 90, 20)
	assertSize(t, m, 90, 20)
	v := drawn(m)
	if !strings.Contains(v, "Hub · 2 links · 2 backlinks · 1 step") {
		t.Fatalf("title:\n%s", v)
	}
	if !strings.Contains(v, "Study ●─") || !strings.Contains(v, "● Docker") || !strings.Contains(v, "● AWS") {
		t.Fatalf("expected Study left, Docker and AWS right:\n%s", v)
	}
	for _, s := range []string{"Index", "EC2", "Lonely"} {
		if strings.Contains(v, s) {
			t.Errorf("%s is not directly linked:\n%s", s, v)
		}
	}
	if strings.Count(v, "Docker") != 1 {
		t.Fatalf("a note linked both ways appears once:\n%s", v)
	}
	if !strings.ContainsRune(v, '╭') || !strings.ContainsRune(v, '╰') {
		t.Fatalf("expected box-drawing joins:\n%s", v)
	}
}

func TestZoomOutReachesFurther(t *testing.T) {
	m := newSample(t, 120, 24)
	m = wheel(m, tea.MouseButtonWheelDown)
	v := drawn(m)
	for _, s := range []string{"Index ●", "● EC2", "● 日本語", "2 steps"} {
		if !strings.Contains(v, s) {
			t.Errorf("missing %q at 2 steps:\n%s", s, v)
		}
	}
	if strings.Contains(v, "Lonely") {
		t.Errorf("Lonely is never linked:\n%s", v)
	}
	// EC2 hangs off AWS: same row band, one column right.
	tr := m.tree()
	aws, ec2 := tr.nodes[tr.index["AWS.md"]], tr.nodes[tr.index["EC2.md"]]
	if ec2.parent != tr.index["AWS.md"] || ec2.col() != aws.col()+1 {
		t.Fatalf("EC2 should be AWS's child: %+v %+v", aws, ec2)
	}
	for i := 0; i < 5; i++ {
		m = wheel(m, tea.MouseButtonWheelUp)
	}
	if !strings.Contains(drawn(m), "1 step") {
		t.Fatal("zooming in stops at one step")
	}
	m, _ = clickText(t, m, " − ")
	if !strings.Contains(drawn(m), "2 steps") {
		t.Fatal("the − button zooms out")
	}
	m, _ = clickText(t, m, " + ")
	if !strings.Contains(drawn(m), "1 step") {
		t.Fatal("the + button zooms in")
	}
}

func key(m Model, k string) (Model, tea.Msg) {
	switch k {
	case "enter":
		return update(m, tea.KeyMsg{Type: tea.KeyEnter})
	case "esc":
		return update(m, tea.KeyMsg{Type: tea.KeyEsc})
	}
	return update(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
}

// keys presses each key in turn and returns the last message sent.
func keys(m Model, ks ...string) (Model, tea.Msg) {
	var msg tea.Msg
	for _, k := range ks {
		m, msg = key(m, k)
	}
	return m, msg
}

func TestOtherKeysDoNothing(t *testing.T) {
	m := newSample(t, 90, 20)
	before := drawn(m)
	for _, k := range []string{"q", "c", "x", "g"} {
		var msg tea.Msg
		m, msg = key(m, k)
		if msg != nil {
			t.Fatalf("key %q sent %#v", k, msg)
		}
	}
	if drawn(m) != before {
		t.Fatal("unbound keys should not change the graph")
	}
}

func TestKeysHLMoveBetweenParentAndChildren(t *testing.T) {
	m := newSample(t, 120, 24)
	m, msg := key(m, "l")
	if m.Selected() != "Docker.md" || msg != (SelectMsg{ID: "Docker.md"}) {
		t.Fatalf("l from the centre should select its first link, got %q %#v", m.Selected(), msg)
	}
	if m, _ = key(m, "h"); m.Selected() != "Hub.md" {
		t.Fatalf("h from a link should go back to its parent, got %q", m.Selected())
	}
	if m, _ = key(m, "h"); m.Selected() != "Study.md" {
		t.Fatalf("h from the centre should select its first backlink, got %q", m.Selected())
	}
	if m, _ = key(m, "l"); m.Selected() != "Hub.md" {
		t.Fatalf("l from a backlink should go back to its parent, got %q", m.Selected())
	}

	m, _ = keys(m, "-", "h", "h") // two steps: Study's backlink is Index
	if m.Selected() != "Index.md" {
		t.Fatalf("h from a backlink should go to its own backlink, got %q", m.Selected())
	}
	m, msg = key(m, "h") // a leaf: nowhere further
	if m.Selected() != "Index.md" || msg != nil {
		t.Fatalf("h from a leaf should stay, got %q %#v", m.Selected(), msg)
	}
	m, _ = keys(m, "l", "l", "l", "j", "l")
	if m.Selected() != "EC2.md" {
		t.Fatalf("l from AWS should reach EC2, got %q", m.Selected())
	}
}

func TestKeysJKMoveBetweenSiblings(t *testing.T) {
	m := newSample(t, 120, 24)
	m, msg := key(m, "j")
	if m.Selected() != "Hub.md" || msg != nil {
		t.Fatalf("the centre has no siblings, got %q %#v", m.Selected(), msg)
	}
	m, _ = keys(m, "l", "j")
	if m.Selected() != "AWS.md" {
		t.Fatalf("j should move to the next sibling, got %q", m.Selected())
	}
	if m, msg = key(m, "j"); m.Selected() != "AWS.md" || msg != nil {
		t.Fatalf("j on the last sibling should stay, got %q %#v", m.Selected(), msg)
	}
	if m, msg = key(m, "k"); m.Selected() != "Docker.md" || msg != (SelectMsg{ID: "Docker.md"}) {
		t.Fatalf("k should move to the previous sibling, got %q %#v", m.Selected(), msg)
	}
	if m, _ = key(m, "k"); m.Selected() != "Docker.md" {
		t.Fatalf("k on the first sibling should stay, got %q", m.Selected())
	}
}

func TestKeyEnterOpensSelected(t *testing.T) {
	m := newSample(t, 90, 20)
	if _, msg := key(m, "enter"); msg != (OpenMsg{ID: "Hub.md"}) {
		t.Fatalf("enter on the centre = %#v", msg)
	}
	if _, msg := keys(m, "l", "j", "enter"); msg != (OpenMsg{ID: "AWS.md"}) {
		t.Fatalf("enter should open the selected note, got %#v", msg)
	}
}

func TestKeysPlusMinusZoom(t *testing.T) {
	m := newSample(t, 120, 24)
	m, _ = key(m, "-")
	if !strings.Contains(drawn(m), "2 steps") || !strings.Contains(drawn(m), "EC2") {
		t.Fatalf("- should zoom out:\n%s", drawn(m))
	}
	m, _ = key(m, "+")
	if !strings.Contains(drawn(m), "1 step") || strings.Contains(drawn(m), "EC2") {
		t.Fatalf("+ should zoom in:\n%s", drawn(m))
	}
}

func TestKeyZeroCentresAndEscGoesBack(t *testing.T) {
	m := newSample(t, 90, 20)
	m, _ = keys(m, "l", "j", "0")
	if m.Centre() != "AWS.md" || !strings.Contains(drawn(m), "AWS · 2 links · 1 backlink") {
		t.Fatalf("0 should re-centre on the selected note:\n%s", drawn(m))
	}
	m, _ = key(m, "esc")
	if m.Centre() != "Hub.md" {
		t.Fatalf("esc should go back to Hub, got %q", m.Centre())
	}
	m, _ = key(m, "esc") // nothing left to go back to
	if m.Centre() != "Hub.md" {
		t.Fatalf("esc with no history moved to %q", m.Centre())
	}
}

// The bottom row names the keys and still fits a 40-column pane.
func TestKeyHintFitsThePane(t *testing.T) {
	for _, w := range []int{40, 50, 52, 60, 90, 140} {
		m := newSample(t, w, 20)
		assertSize(t, m, w, 20)
		lines := strings.Split(drawn(m), "\n")
		hint := lines[len(lines)-1]
		if strings.Contains(hint, "…") {
			t.Errorf("width %d: hint cut short: %q", w, hint)
		}
		for _, s := range []string{"hjkl", "enter", "−", "+", "0", "esc"} {
			if !strings.Contains(hint, s) {
				t.Errorf("width %d: hint lacks %q: %q", w, s, hint)
			}
		}
		if w >= 52 && (!strings.Contains(hint, " 0 centre ") || !strings.Contains(hint, " esc back ")) {
			t.Errorf("width %d: buttons should keep their names: %q", w, hint)
		}
		// The buttons still work from the hint row.
		m, _ = keys(m, "l", "j")
		m, _ = clickText(t, m, "0")
		if m.Centre() != "AWS.md" {
			t.Errorf("width %d: the centre button no longer works", w)
		}
	}
}

func TestCentreAndBackButtons(t *testing.T) {
	m := newSample(t, 90, 20)
	m, msg := clickText(t, m, "AWS")
	if msg != (SelectMsg{ID: "AWS.md"}) {
		t.Fatalf("click = %#v", msg)
	}
	m, _ = clickText(t, m, " centre ")
	if m.Centre() != "AWS.md" || !strings.Contains(drawn(m), "AWS · 2 links · 1 backlink") {
		t.Fatalf("centre should re-centre on AWS:\n%s", drawn(m))
	}
	m, _ = clickText(t, m, " back ")
	if m.Centre() != "Hub.md" {
		t.Fatalf("back should return to Hub, got %q", m.Centre())
	}
	m, _ = clickText(t, m, " back ") // nothing left to go back to
	if m.Centre() != "Hub.md" {
		t.Fatalf("back with no history moved to %q", m.Centre())
	}
}

func TestPanAndDrag(t *testing.T) {
	m := newSample(t, 90, 20)
	start := drawn(m)
	m, _ = m.Update(tea.MouseMsg{X: 5, Y: 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m, _ = m.Update(tea.MouseMsg{X: 15, Y: 4, Action: tea.MouseActionMotion})
	m, _ = m.Update(tea.MouseMsg{X: 15, Y: 4, Action: tea.MouseActionRelease})
	if drawn(m) == start {
		t.Fatal("dragging empty space should pan")
	}
	assertSize(t, m, 90, 20)
	m, _ = clickText(t, m, " reset ")
	if drawn(m) != start {
		t.Fatal("reset should put the map back")
	}
}

func TestClickSelectsThenOpens(t *testing.T) {
	m := newSample(t, 90, 20)
	for y, l := range strings.Split(drawn(m), "\n") {
		if i := strings.Index(l, "AWS"); i >= 0 {
			click := tea.MouseMsg{X: ansi.StringWidth(l[:i]), Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
			var cmd tea.Cmd
			m, cmd = m.Update(click)
			if m.Selected() != "AWS.md" || cmd == nil || cmd() != (SelectMsg{ID: "AWS.md"}) {
				t.Fatalf("first click should select AWS, got %q", m.Selected())
			}
			_, cmd = m.Update(click)
			if cmd == nil || cmd() != (OpenMsg{ID: "AWS.md"}) {
				t.Fatal("second click should open AWS")
			}
			return
		}
	}
	t.Fatal("AWS not drawn")
}

func TestBigMapsStayExactSize(t *testing.T) {
	notes := []core.Page{{ID: "c.md", Title: "Centre"}}
	var edges []core.Edge
	for i := 0; i < 60; i++ {
		id := strings.Repeat("n", i/26+1) + string(rune('a'+i%26)) + ".md"
		notes = append(notes, core.Page{ID: id, Title: "Note " + id})
		if i%2 == 0 {
			edges = append(edges, core.Edge{From: "c.md", To: id})
		} else {
			edges = append(edges, core.Edge{From: id, To: "c.md"})
		}
	}
	m := New(theme.DefaultTheme())
	m.SetSize(50, 15)
	m.SetFocused(true)
	m.SetLinks(notes, edges)
	m.SetCentre("c.md")
	assertSize(t, m, 50, 15)
	for i := 0; i < 3; i++ {
		m = wheel(m, tea.MouseButtonWheelDown)
		assertSize(t, m, 50, 15)
	}
	m.SetSize(20, 6)
	assertSize(t, m, 20, 6)
}

// Regression: opening the graph from a note that isn't the most linked one
// used to leave the most linked note in the history, so "back" jumped there.
func TestBackOnlyRetracesMovesInTheGraph(t *testing.T) {
	notes, edges := sample()
	m := New(theme.DefaultTheme())
	m.SetSize(90, 20)
	m.SetLinks(notes, edges) // Hub, the most linked note, fills in as centre
	m.SetCentre("AWS.md")
	if len(m.history) != 0 {
		t.Fatalf("opening the graph should start with no history, got %v", m.history)
	}
	m, _ = clickText(t, m, " back ")
	if m.Centre() != "AWS.md" {
		t.Fatalf("back with nothing to go back to moved to %q", m.Centre())
	}

	m, _ = clickText(t, m, "EC2")
	m, _ = clickText(t, m, " centre ")
	m, _ = clickText(t, m, " back ")
	if m.Centre() != "AWS.md" {
		t.Fatalf("back should return to AWS, got %q", m.Centre())
	}

	m, _ = clickText(t, m, "EC2")
	m, _ = clickText(t, m, " centre ")
	m.SetCentre("Docker.md") // the graph is opened again from another note
	m, _ = clickText(t, m, " back ")
	if m.Centre() != "Docker.md" {
		t.Fatalf("reopening should clear the history; back went to %q", m.Centre())
	}
}

// Moving with the keys scrolls the map to keep the selected note on screen.
func TestKeysKeepTheSelectionInView(t *testing.T) {
	notes := []core.Page{{ID: "c.md", Title: "Centre"}}
	var edges []core.Edge
	for i := 0; i < 30; i++ {
		id := string(rune('a'+i%26)) + strings.Repeat("z", i/26) + ".md"
		notes = append(notes, core.Page{ID: id, Title: "Note " + id})
		edges = append(edges, core.Edge{From: "c.md", To: id})
	}
	m := New(theme.DefaultTheme())
	m.SetSize(50, 12)
	m.SetFocused(true)
	m.SetLinks(notes, edges)
	m.SetCentre("c.md")
	m, _ = key(m, "l")
	for i := 0; i < 29; i++ {
		m, _ = key(m, "j")
		if title := "Note " + m.Selected(); !strings.Contains(drawn(m), title) {
			t.Fatalf("after %d× j, %q is off screen:\n%s", i+1, title, drawn(m))
		}
	}
	assertSize(t, m, 50, 12)
	for i := 0; i < 29; i++ {
		m, _ = key(m, "k")
	}
	if !strings.Contains(drawn(m), "Note a.md") {
		t.Fatalf("k should scroll back up:\n%s", drawn(m))
	}
}
