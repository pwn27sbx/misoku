package globalgraph

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// The fixture's map (vault() hand-placed by vaultLayout, grid cells):
//
//	x:  1      2           8     9
//	y0: Home   Inbox       Go    study/Notes
//	y1: Orphan Ideas       Rust  日本語のノート
//	y4: Plan   Tasks
//	y5: Meeting work/Notes
//
// The keys, mouse and filter tests need neighbours in set directions and
// distances, so they place the notes by hand instead of taking whatever
// the galaxy layout (Build) makes of them; layout_test pins Build.
var vaultCells = map[string]Point{
	"Home.md": {1, 0}, "Inbox.md": {2, 0}, "Orphan.md": {1, 1}, "Ideas": {2, 1},
	"study/Go.md": {8, 0}, "study/Notes.md": {9, 0}, "study/Rust.md": {8, 1}, "study/日本語のノート.md": {9, 1},
	"work/Plan.md": {1, 4}, "work/Tasks.md": {2, 4}, "work/Meeting.md": {1, 5}, "work/Notes.md": {2, 5},
}

// vaultLayout is vault() at vaultCells, one cluster per folder.
func vaultLayout() Layout {
	nodes, _ := normalize(vault())
	for i := range nodes {
		nodes[i].P = vaultCells[nodes[i].ID]
	}
	clusters := []Cluster{
		{Key: "", Rect: Rect{0, 0, 4, 2}, Centre: Point{1, 0}},
		{Key: "study", Rect: Rect{7, 0, 4, 2}, Centre: Point{8, 0}},
		{Key: "work", Rect: Rect{0, 4, 4, 2}, Centre: Point{1, 4}},
	}
	return finish(nodes, clusters)
}

func newModel(t testing.TB, w, h int) Model {
	t.Helper()
	m := New(theme.DefaultTheme())
	m.SetSize(w, h)
	m.SetFocused(true)
	l := vaultLayout()
	r := NewRenderer(l, vault().Edges)
	r.warm()
	m.SetMap(Map{g: vault(), l: l, r: r, relaid: true})
	m.SetOpenNote("Home.md")
	return m
}

func send(m Model, msg tea.Msg) (Model, tea.Msg) {
	m, cmd := m.Update(msg)
	if cmd == nil {
		return m, nil
	}
	return m, cmd()
}

func press(m Model, keys ...string) (Model, tea.Msg) {
	var msg tea.Msg
	for _, k := range keys {
		m, msg = send(m, keyMsg(k))
	}
	return m, msg
}

func keyMsg(k string) tea.KeyMsg {
	switch k {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func click(m Model, x, y int) (Model, tea.Msg) {
	return send(m, tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
}

func wheelAt(m Model, b tea.MouseButton, x, y int) Model {
	m, _ = send(m, tea.MouseMsg{X: x, Y: y, Button: b, Action: tea.MouseActionPress})
	return m
}

// clickOn clicks the first place text is drawn below the title row.
func clickOn(t *testing.T, m Model, text string) (Model, tea.Msg) {
	t.Helper()
	for y, l := range strings.Split(ansi.Strip(m.View()), "\n") {
		if i := strings.Index(l, text); i >= 0 && y > 0 {
			return click(m, ansi.StringWidth(l[:i]), y)
		}
	}
	t.Fatalf("%q not drawn:\n%s", text, ansi.Strip(m.View()))
	return m, nil
}

// selPane is where the selection's dot is drawn in the pane.
func selPane(m Model) (x, y int) {
	x, y = m.screen(m.sel)
	return x, y + 1
}

// checkSelVisible fails unless the selection's dot is drawn inside the map.
func checkSelVisible(t *testing.T, m Model, what string) {
	t.Helper()
	if m.sel == "" {
		return
	}
	x, y := selPane(m)
	if x < 0 || y < 1 || x >= m.width || y > m.mapRows() {
		t.Fatalf("%s: selection %q at (%d,%d) is outside the %dx%d map", what, m.sel, x, y, m.width, m.mapRows())
	}
	rows := screen(m.View())
	if got := rows[y][x].r; got != glyphSelected && got != glyphUnresolved {
		t.Fatalf("%s: cell (%d,%d) of the selection %q shows %q", what, x, y, m.sel, got)
	}
}

func TestModelArrowAndHJKLKeysMoveToTheNearestNode(t *testing.T) {
	cases := []struct {
		from, key, want string
	}{
		{"Home.md", "l", "Inbox.md"},
		{"Home.md", "right", "Inbox.md"},
		{"Home.md", "j", "Orphan.md"}, // straight below beats Ideas, below and aside
		{"Home.md", "down", "Orphan.md"},
		{"Home.md", "h", "Home.md"},      // nothing to the left
		{"Home.md", "k", "Home.md"},      // nothing above
		{"Inbox.md", "l", "study/Go.md"}, // across the gap between projects
		{"Orphan.md", "j", "work/Plan.md"},
		{"Ideas", "j", "work/Tasks.md"},
		{"study/Go.md", "j", "study/Rust.md"},
		{"study/Rust.md", "h", "Ideas"}, // same row beats Orphan, further
		{"study/Rust.md", "left", "Ideas"},
		{"work/Plan.md", "k", "Orphan.md"},
		{"work/Plan.md", "up", "Orphan.md"},
		{"work/Tasks.md", "l", "study/Rust.md"},    // the nearest within 45°
		{"work/Notes.md", "l", "study/Rust.md"},    // none within 45°: the nearest on that side
		{"study/日本語のノート.md", "j", "work/Notes.md"}, // within 45° beats the nearer Tasks
	}
	for _, c := range cases {
		t.Run(c.from+"/"+c.key, func(t *testing.T) {
			m := newModel(t, 80, 20)
			m.SetOpenNote(c.from)
			m, msg := press(m, c.key)
			if m.Selected() != c.want {
				t.Fatalf("selection = %q, want %q", m.Selected(), c.want)
			}
			var want tea.Msg
			if c.want != c.from {
				want = SelectMsg{ID: c.want}
			}
			if msg != want {
				t.Fatalf("msg = %#v, want %#v", msg, want)
			}
			checkSelVisible(t, m, c.key)
		})
	}
}

// With nothing selected and a map bigger than the pane, the view opens
// on the centre of the biggest galaxy, not on the middle of the whole map,
// which may be the empty space between galaxies.
func TestModelWithNothingSelectedCentresTheBiggestGalaxy(t *testing.T) {
	var g Graph
	add := func(key string, n int) {
		for i := range n {
			g.Nodes = append(g.Nodes, Node{ID: fmt.Sprintf("%s/n%02d.md", key, i), Cluster: key})
			if i > 0 {
				g.Edges = append(g.Edges, Edge{fmt.Sprintf("%s/n00.md", key), fmt.Sprintf("%s/n%02d.md", key, i)})
			}
		}
	}
	add("a", 3)
	add("big", 30)
	add("z", 30) // as big, but "big" sorts first
	g.Edges = append(g.Edges, Edge{"a/n01.md", "big/n01.md"}, Edge{"z/n01.md", "big/n02.md"})
	m := New(theme.DefaultTheme())
	m.SetSize(60, 20)
	m.SetFocused(true)
	m.SetGraph(g)
	for _, keys := range [][]string{nil, {"+"}, {"+", "+"}} {
		m2, _ := press(m, keys...)
		if m2.Selected() != "" {
			t.Fatalf("setup: %v selected %q", keys, m2.Selected())
		}
		sx, sy := m2.scale()
		if (m2.l.W-1)*sx < m2.width && (m2.l.H-1)*sy < m2.mapRows() {
			continue // the whole map fits: it is centred instead
		}
		x, y := m2.screen("big/n00.md")
		if x != m2.width/2 || y != m2.mapRows()/2 {
			t.Fatalf("%v: the biggest galaxy's centre is at (%d,%d), want the pane's middle (%d,%d)", keys, x, y, m2.width/2, m2.mapRows()/2)
		}
		if c := screen(m2.View())[y+1][x]; !isDot(c) {
			t.Fatalf("%v: the middle of the pane shows %q, want the hub's dot", keys, c.r)
		}
	}
	// A map that fits at the overview is centred whole there; zooming in
	// with the keys then opens on the biggest galaxy too.
	small := Graph{
		Nodes: []Node{{ID: "Home.md"}, {ID: "Orphan.md"}, {ID: "Ghost", Unresolved: true}, {ID: "work/Plan.md", Cluster: "work"}},
		Edges: []Edge{{"Home.md", "work/Plan.md"}, {"Home.md", "Ghost"}},
	}
	m = New(theme.DefaultTheme())
	m.SetSize(54, 36)
	m.SetFocused(true)
	m.SetGraph(small)
	for _, keys := range [][]string{{"+"}, {"+", "+"}} {
		m2, _ := press(m, keys...)
		if sx, sy := m2.scale(); (m2.l.W-1)*sx < m2.width && (m2.l.H-1)*sy < m2.mapRows() {
			continue // medium is the overview's scale here: + keeps the view
		}
		if x, y := m2.screen("Home.md"); x != m2.width/2 || y != m2.mapRows()/2 {
			t.Fatalf("%v: the biggest galaxy's centre Home is at (%d,%d), want the pane's middle (%d,%d)", keys, x, y, m2.width/2, m2.mapRows()/2)
		}
	}
}

func TestModelKeysWithNothingSelectedPickTheMiddle(t *testing.T) {
	m := New(theme.DefaultTheme())
	m.SetSize(80, 20)
	m.SetGraph(vault())
	if m.Selected() != "" || !strings.Contains(ansi.Strip(m.View()), "All notes") {
		t.Fatalf("no open note should mean no selection:\n%s", ansi.Strip(m.View()))
	}
	for _, k := range []string{"l", "n"} {
		m2, msg := press(m, k)
		if m2.Selected() == "" || msg != (SelectMsg{ID: m2.Selected()}) {
			t.Fatalf("%s should select a node, got %q %#v", k, m2.Selected(), msg)
		}
		checkSelVisible(t, m2, k)
	}
	if _, msg := press(m, "enter"); msg != nil {
		t.Fatalf("enter with nothing selected sent %#v", msg)
	}
}

func TestModelNCyclesThroughTheSelectionsLinks(t *testing.T) {
	m := newModel(t, 80, 20)
	// Home links to Inbox, study/Go, study/Notes and work/Plan.
	want := []string{"Inbox.md", "study/Go.md", "study/Notes.md", "work/Plan.md", "Inbox.md"}
	for i, id := range want {
		var msg tea.Msg
		m, msg = press(m, "n")
		if m.Selected() != id || msg != (SelectMsg{ID: id}) {
			t.Fatalf("n #%d: selection %q msg %#v, want %q", i+1, m.Selected(), msg, id)
		}
		checkSelVisible(t, m, "n")
	}
	m, _ = press(m, "N", "N")
	if m.Selected() != "study/Notes.md" {
		t.Fatalf("N should go back round, got %q", m.Selected())
	}

	m = newModel(t, 80, 20)
	if m, _ = press(m, "N"); m.Selected() != "work/Plan.md" {
		t.Fatalf("N from the hub should pick its last link, got %q", m.Selected())
	}
	// Moving another way starts going round the new selection's links.
	m, _ = press(m, "l", "n")
	if m.Selected() != "work/Notes.md" {
		t.Fatalf("n after l should go round Tasks' links (work/Notes, Plan), got %q", m.Selected())
	}
	if m, msg := press(newModel(t, 80, 20), "j", "n"); m.Selected() != "Orphan.md" || msg != nil {
		t.Fatalf("n on an orphan should do nothing, got %q %#v", m.Selected(), msg)
	}
}

func TestModelEnterOpensTheSelection(t *testing.T) {
	m := newModel(t, 80, 20)
	if _, msg := press(m, "enter"); msg != (OpenMsg{ID: "Home.md"}) {
		t.Fatalf("enter = %#v", msg)
	}
	m.SetOpenNote("Ideas")
	if _, msg := press(m, "enter"); msg != (OpenMsg{ID: "Ideas", Unresolved: true}) {
		t.Fatalf("enter on an unresolved target = %#v", msg)
	}
}

func TestModelZoomKeysStayWithinBounds(t *testing.T) {
	m := newModel(t, 80, 20)
	for i, want := range []Zoom{ZoomMedium, ZoomClose, ZoomClose} {
		var msg tea.Msg
		m, msg = press(m, "+")
		if m.zoom != want || msg != nil {
			t.Fatalf("+ #%d: zoom %d msg %#v, want %d", i+1, m.zoom, msg, want)
		}
		checkSelVisible(t, m, "+")
	}
	for i, want := range []Zoom{ZoomMedium, ZoomOverview, ZoomOverview} {
		m, _ = press(m, "-")
		if m.zoom != want {
			t.Fatalf("- #%d: zoom %d, want %d", i+1, m.zoom, want)
		}
		checkSelVisible(t, m, "-")
	}
}

func TestModelKeyZoomKeepsTheSelectionInPlace(t *testing.T) {
	m := newModel(t, 80, 20)
	m, _ = press(m, "l")
	x, y := selPane(m)
	m, _ = press(m, "+")
	if x2, y2 := selPane(m); x2 != x || y2 != y {
		t.Fatalf("zooming in moved the selection from (%d,%d) to (%d,%d)", x, y, x2, y2)
	}
}

func TestModelWheelZoomsAroundThePointerWithinBounds(t *testing.T) {
	m := newModel(t, 120, 30)
	gx, gy := m.screen("study/Go.md") // Home is selected; the pointer is on Go
	gy++
	for i, want := range []Zoom{ZoomMedium, ZoomClose, ZoomClose} {
		m = wheelAt(m, tea.MouseButtonWheelUp, gx, gy)
		if m.zoom != want {
			t.Fatalf("wheel up #%d: zoom %d, want %d", i+1, m.zoom, want)
		}
		checkSelVisible(t, m, "wheel up")
		if x, y := m.screen("study/Go.md"); x != gx || y+1 != gy {
			t.Fatalf("wheel up #%d under the pointer moved Go from (%d,%d) to (%d,%d)", i+1, gx, gy, x, y)
		}
	}
	for i, want := range []Zoom{ZoomMedium, ZoomOverview, ZoomOverview} {
		m = wheelAt(m, tea.MouseButtonWheelDown, 3, 3)
		if m.zoom != want {
			t.Fatalf("wheel down #%d: zoom %d, want %d", i+1, m.zoom, want)
		}
		checkSelVisible(t, m, "wheel down")
	}
}

func TestModelZeroCentresOnTheSelection(t *testing.T) {
	m := newModel(t, 80, 20)
	m, _ = send(m, tea.MouseMsg{X: 70, Y: 15, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m, _ = send(m, tea.MouseMsg{X: 60, Y: 12, Action: tea.MouseActionMotion})
	m, _ = send(m, tea.MouseMsg{X: 60, Y: 12, Action: tea.MouseActionRelease})
	if x, y := selPane(m); x == 40 && y == 10 {
		t.Fatal("the drag should have moved the selection off the middle")
	}
	m, msg := press(m, "0")
	if x, y := selPane(m); x != 40 || y != 10 || msg != nil {
		t.Fatalf("0 should put the selection mid-pane, got (%d,%d) %#v", x, y, msg)
	}
}

func TestModelEscGoesBackThroughTheSelections(t *testing.T) {
	m := newModel(t, 80, 20)
	m, _ = press(m, "l", "l", "j") // Home → Inbox → Go → Rust
	for _, want := range []string{"study/Go.md", "Inbox.md", "Home.md", "Home.md"} {
		var msg tea.Msg
		before := m.Selected()
		m, msg = press(m, "esc")
		if m.Selected() != want {
			t.Fatalf("esc: %q, want %q", m.Selected(), want)
		}
		if want != before && msg != (SelectMsg{ID: want}) || want == before && msg != nil {
			t.Fatalf("esc to %q sent %#v", want, msg)
		}
		checkSelVisible(t, m, "esc")
	}

	// Nodes removed since are skipped; reopening clears the history.
	m, _ = press(m, "l", "l", "j") // Home → Inbox → Go → Rust
	g := vault()
	g.Nodes = slicesDelete(g.Nodes, "study/Go.md")
	m.SetGraph(g)
	if m, _ = press(m, "esc"); m.Selected() != "Inbox.md" {
		t.Fatalf("esc should skip the removed Go, got %q", m.Selected())
	}
	m, _ = press(m, "l")
	m.SetOpenNote("work/Plan.md")
	if m, msg := press(m, "esc"); m.Selected() != "work/Plan.md" || msg != nil {
		t.Fatalf("reopening should clear the history, esc went to %q", m.Selected())
	}
}

func slicesDelete(nodes []Node, id string) []Node {
	var out []Node
	for _, n := range nodes {
		if n.ID != id {
			out = append(out, n)
		}
	}
	return out
}

func TestModelUnknownKeysAreIgnored(t *testing.T) {
	m := newModel(t, 80, 20)
	before := m.View()
	for _, k := range []string{"q", "x", "g", ":", "tab", "H", "L"} {
		next, cmd := m.Update(keyMsg(k))
		if cmd != nil {
			t.Fatalf("key %q returned a command", k)
		}
		if !reflect.DeepEqual(next, m) {
			t.Fatalf("key %q changed the model", k)
		}
	}
	if next, cmd := m.Update(tea.FocusMsg{}); cmd != nil || !reflect.DeepEqual(next, m) {
		t.Fatal("other messages should be ignored")
	}
	if m.View() != before {
		t.Fatal("unknown keys changed the picture")
	}
}

func TestModelEmptyMapIgnoresInput(t *testing.T) {
	m := New(theme.DefaultTheme())
	m.SetSize(80, 20)
	for _, msg := range []tea.Msg{keyMsg("l"), keyMsg("enter"), keyMsg("+"),
		tea.MouseMsg{X: 3, Y: 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}} {
		if _, cmd := m.Update(msg); cmd != nil {
			t.Fatalf("%#v on an empty map returned a command", msg)
		}
	}
	if !strings.Contains(ansi.Strip(m.View()), "No notes yet") {
		t.Fatalf("empty map:\n%s", ansi.Strip(m.View()))
	}
}

func TestModelClickSelectsThenOpens(t *testing.T) {
	// Inbox's dot is the grid cell right of the selected Home's; Home's
	// name goes left of its dot so as not to cover it.
	m := newModel(t, 80, 20)
	x, y := m.screen("Inbox.md")
	click := tea.MouseMsg{X: x, Y: y + 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
	m, msg := send(m, click)
	if m.Selected() != "Inbox.md" || msg != (SelectMsg{ID: "Inbox.md"}) {
		t.Fatalf("first click on the dot: %q %#v", m.Selected(), msg)
	}
	if _, msg = send(m, click); msg != (OpenMsg{ID: "Inbox.md"}) {
		t.Fatalf("second click on the dot = %#v", msg)
	}

	// At close zoom Inbox's name is drawn as a link of Home.
	m = newModel(t, 80, 20)
	m, _ = press(m, "+", "+")
	m, msg = clickOn(t, m, "Inbox")
	if m.Selected() != "Inbox.md" || msg != (SelectMsg{ID: "Inbox.md"}) {
		t.Fatalf("first click on the name: %q %#v", m.Selected(), msg)
	}
	if _, msg = clickOn(t, m, "Inbox"); msg != (OpenMsg{ID: "Inbox.md"}) {
		t.Fatalf("second click on the name = %#v", msg)
	}

	m.SetOpenNote("Ideas")
	x, y = selPane(m)
	if _, msg = send(m, tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}); msg != (OpenMsg{ID: "Ideas", Unresolved: true}) {
		t.Fatalf("click on the selected unresolved target = %#v", msg)
	}
}

func TestModelClickOnTheTitleRowDoesNothing(t *testing.T) {
	m := newModel(t, 80, 20)
	next, msg := click(m, 5, 0)
	if msg != nil || next.Selected() != m.Selected() || next.dragging {
		t.Fatalf("title click: %#v %q", msg, next.Selected())
	}
}

func TestModelDragPansAndResetFitsTheMap(t *testing.T) {
	m := newModel(t, 80, 20)
	m, _ = press(m, "+")
	start := m.View()
	x, y := selPane(m)
	m, _ = send(m, tea.MouseMsg{X: 70, Y: 15, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m, _ = send(m, tea.MouseMsg{X: 66, Y: 13, Action: tea.MouseActionMotion})
	m, _ = send(m, tea.MouseMsg{X: 66, Y: 13, Action: tea.MouseActionRelease})
	if x2, y2 := selPane(m); x2 != x-4 || y2 != y-2 {
		t.Fatalf("drag by (-4,-2) moved the selection from (%d,%d) to (%d,%d)", x, y, x2, y2)
	}
	if m.View() == start {
		t.Fatal("dragging should pan")
	}
	m, _ = send(m, tea.MouseMsg{X: 10, Y: 10, Action: tea.MouseActionMotion})
	if x3, y3 := selPane(m); x3 != x-4 || y3 != y-2 {
		t.Fatal("motion after release should not pan")
	}
	m, _ = clickOn(t, m, " reset ")
	want := newModel(t, 80, 20)
	want.offset, want.placed = want.r.Centred(ZoomOverview, 80, 18), true
	if m.zoom != ZoomOverview || m.View() != want.View() {
		t.Fatalf("reset should show the whole map centred:\n%s\nwant:\n%s", ansi.Strip(m.View()), ansi.Strip(want.View()))
	}
}

func TestModelHintButtonsWork(t *testing.T) {
	m := newModel(t, 120, 20)
	m, _ = clickOn(t, m, " + ")
	if m.zoom != ZoomMedium {
		t.Fatalf("+ button: zoom %d", m.zoom)
	}
	m, _ = clickOn(t, m, " − ")
	if m.zoom != ZoomOverview {
		t.Fatalf("− button: zoom %d", m.zoom)
	}
	m, _ = clickOn(t, m, " − ") // disabled at the overview
	if m.zoom != ZoomOverview {
		t.Fatalf("disabled − button zoomed to %d", m.zoom)
	}
	m, _ = press(m, "l")
	m, msg := clickOn(t, m, " back ")
	if m.Selected() != "Home.md" || msg != (SelectMsg{ID: "Home.md"}) {
		t.Fatalf("back button: %q %#v", m.Selected(), msg)
	}
	m.offset.X += 9
	m, _ = clickOn(t, m, " centre ")
	if x, y := selPane(m); x != 60 || y != 10 {
		t.Fatalf("centre button: selection at (%d,%d)", x, y)
	}
}

func TestModelHintFitsNarrowPanes(t *testing.T) {
	for _, w := range []int{MinWidth, 35, 40, 46, 52, 58, 70, 90, 140, 200} {
		m := newModel(t, w, 12)
		lines := strings.Split(ansi.Strip(m.View()), "\n")
		hint := lines[len(lines)-1]
		end := strings.TrimRight(hint, " ")
		if strings.Contains(hint, "…") || !strings.HasSuffix(end, "reset") && !strings.HasSuffix(end, "enter") &&
			!strings.HasSuffix(end, "open") && !strings.HasSuffix(end, "drag: move") {
			t.Errorf("width %d: hint cut short: %q", w, hint)
		}
		for _, s := range []string{" − ", " + ", " 0 ", " esc ", " reset "} {
			if !strings.Contains(strings.ReplaceAll(strings.ReplaceAll(hint, " centre", ""), " back", ""), s) {
				t.Errorf("width %d: hint lacks button %q: %q", w, s, hint)
			}
		}
		if w >= 40 && !strings.Contains(hint, "hjkl") {
			t.Errorf("width %d: hint should name hjkl: %q", w, hint)
		}
		if w >= 46 && !strings.Contains(hint, "n/N") {
			t.Errorf("width %d: hint should name n/N: %q", w, hint)
		}
		if w >= 58 && (!strings.Contains(hint, " 0 centre ") || !strings.Contains(hint, " esc back ")) {
			t.Errorf("width %d: buttons should keep their names: %q", w, hint)
		}
		// The buttons still work at this width.
		m, _ = clickOn(t, m, " + ")
		if m.zoom != ZoomMedium {
			t.Errorf("width %d: + button does nothing", w)
		}
	}
}

func TestModelViewIsExactlyThePaneSize(t *testing.T) {
	for _, s := range [][2]int{{1, 1}, {29, 8}, {30, 7}, {30, 8}, {31, 9}, {80, 24}, {200, 60}} {
		m := newModel(t, s[0], s[1])
		for _, z := range []string{"", "+", "+"} {
			if z != "" {
				m, _ = press(m, z)
			}
			checkSize(t, m.View(), s[0], s[1], fmt.Sprintf("%dx%d zoom %d", s[0], s[1], m.zoom))
		}
		m.SetDimmed(true)
		checkSize(t, m.View(), s[0], s[1], fmt.Sprintf("%dx%d dimmed", s[0], s[1]))
	}
	if New(theme.DefaultTheme()).View() != "" {
		t.Fatal("a pane with no size should render nothing")
	}
}

func TestModelMinimumSizeAppliesToTheWholePane(t *testing.T) {
	for _, s := range [][2]int{{MinWidth - 1, 20}, {80, MinHeight - 1}} {
		got := ansi.Strip(newModel(t, s[0], s[1]).View())
		if !strings.Contains(got, "Graph needs a bigger pane") || strings.Contains(got, "reset") {
			t.Fatalf("%dx%d should show only the message:\n%s", s[0], s[1], got)
		}
	}
	m := newModel(t, MinWidth, MinHeight)
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	if strings.Contains(m.View(), "bigger pane") || !strings.Contains(lines[0], "Home") || !strings.Contains(lines[len(lines)-1], "reset") {
		t.Fatalf("%dx%d should fit title, map and hints:\n%s", MinWidth, MinHeight, strings.Join(lines, "\n"))
	}
	checkSelVisible(t, m, "minimum pane")
	if next, cmd := newModel(t, 20, 5).Update(tea.MouseMsg{X: 1, Y: 1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}); cmd != nil || next.dragging {
		t.Fatal("the mouse should do nothing on the message")
	}
}

func TestModelTitleCountsTheVault(t *testing.T) {
	m := newModel(t, 120, 20)
	title := strings.Split(ansi.Strip(m.View()), "\n")[0]
	want := "󰈙 Home · 11 notes · 12 links · 1 orphan · 1 unresolved"
	if !strings.Contains(title, want) {
		t.Fatalf("title = %q, want %q", title, want)
	}
	m, _ = press(m, "l")
	if title := strings.Split(ansi.Strip(m.View()), "\n")[0]; !strings.HasPrefix(strings.TrimSpace(title), "󰈙 Inbox ·") {
		t.Fatalf("title should name the selection: %q", title)
	}
}

func TestModelSelectionStaysInViewOnABigMap(t *testing.T) {
	g := fixture(300, 6, 500, 3)
	m := New(theme.DefaultTheme())
	m.SetSize(40, 12)
	m.SetFocused(true)
	m.SetGraph(g)
	m.SetOpenNote(g.Nodes[0].ID)
	keys := strings.Split("l l l l l j j j j j n n n N h h h h h h k k k k k + l l l j j j + h h h h k k k - - j j l 0 esc esc esc n n", " ")
	for i, k := range keys {
		m, _ = press(m, k)
		checkSelVisible(t, m, fmt.Sprintf("key %d %q", i, k))
		checkSize(t, m.View(), 40, 12, k)
	}
	for _, b := range []tea.MouseButton{tea.MouseButtonWheelUp, tea.MouseButtonWheelUp, tea.MouseButtonWheelDown} {
		m = wheelAt(m, b, 0, 1)
		checkSelVisible(t, m, "wheel")
	}
	m.SetSize(31, 9)
	checkSelVisible(t, m, "resize")
	m, _ = clickOn(t, m, " reset ")
	checkSelVisible(t, m, "reset")
}

func TestModelIncrementalMapKeepsTheView(t *testing.T) {
	m := newModel(t, 80, 20)
	m, _ = press(m, "l", "+")
	m.offset.X += 3 // panned by hand
	before := m.viewOffset()
	g := vault()
	g.Nodes = append(g.Nodes, Node{ID: "new/One.md", Cluster: "new"})
	m.SetGraph(g)
	if m.viewOffset() != before || m.Selected() != "Inbox.md" {
		t.Fatalf("adding a note moved the view %v → %v or the selection to %q", before, m.viewOffset(), m.Selected())
	}
}

func TestModelFullRelayoutKeepsAndCentresTheSelection(t *testing.T) {
	// Grow the top-left cluster until the layout can no longer keep its
	// nodes in place, as in TestUpdateGrowsFullClustersWithoutOverlap.
	g := fixture(300, 9, 100, 6)
	m := New(theme.DefaultTheme())
	m.SetSize(60, 16)
	m.SetGraph(g)
	key := ""
	for _, c := range m.Layout().Clusters {
		if c.Rect.X == 0 && c.Rect.Y == 0 {
			key = c.Key
		}
	}
	var next Graph
	for i := range 2000 {
		next = Graph{Nodes: append(append([]Node(nil), g.Nodes...), Node{ID: fmt.Sprintf("%s/grow%04d.md", key, i), Cluster: key}), Edges: g.Edges}
		if NewMap(m.Layout(), next).relaid {
			break
		}
		m.SetGraph(next)
		g = next
	}
	fresh, sel := Build(next), ""
	for _, n := range m.Layout().Nodes {
		if f, _ := fresh.Lookup(n.ID); f.P != n.P && sel == "" {
			sel = n.ID
		}
	}
	if sel == "" {
		t.Fatal("the fixture should force a fresh layout that moves nodes")
	}
	m.SetOpenNote(sel)
	m.offset.X, m.offset.Y = m.offset.X+7, m.offset.Y+2 // panned by hand
	m, _ = press(m, "l", "esc")
	m.SetGraph(next)
	if m.Selected() != sel {
		t.Fatalf("selection moved to %q", m.Selected())
	}
	got, _ := m.Layout().Lookup(sel)
	want, _ := fresh.Lookup(sel)
	if got.P != want.P {
		t.Fatalf("the model should hold the fresh layout: %v, want %v", got.P, want.P)
	}
	if x, y := selPane(m); x != 30 || y != 8 {
		t.Fatalf("a fresh layout should centre on the selection, got (%d,%d)", x, y)
	}
	checkSelVisible(t, m, "relayout")
}

func TestModelRemovedSelectionFallsBackToTheOpenNote(t *testing.T) {
	m := newModel(t, 80, 20)
	m, _ = press(m, "l")
	g := vault()
	g.Nodes = slicesDelete(g.Nodes, "Inbox.md")
	m.SetGraph(g)
	if m.Selected() != "Home.md" {
		t.Fatalf("selection = %q, want the open note", m.Selected())
	}
}

// The overview's scale depends on the pane size, so only medium and close
// are warmed (dot cells and edge layer); the first overview frame builds
// its own, unless it shares the medium scale.
func TestNewMapWarmsTheRenderer(t *testing.T) {
	mp := NewMap(Layout{}, vault())
	for _, z := range []Zoom{ZoomMedium, ZoomClose} {
		if sx, sy := z.fixedScale(); mp.r.dots[[2]int{sx, sy}] == nil || mp.r.layers[layerKey{sx, false}] == nil {
			t.Fatalf("zoom %d not warmed", z)
		}
	}
	m := New(theme.DefaultTheme())
	m.SetSize(40, 10)
	m.SetMap(mp)
	m.View()
	sx, sy := m.scale()
	built := mp.r.dots[[2]int{sx, sy}]
	if built == nil {
		t.Fatalf("the first overview frame should cache its dot cells at %dx%d", sx, sy)
	}
	cached := len(mp.r.dots)
	m.View()
	if len(mp.r.dots) != cached || reflect.ValueOf(mp.r.dots[[2]int{sx, sy}]).UnsafePointer() != reflect.ValueOf(built).UnsafePointer() {
		t.Fatalf("a second frame at the same size should reuse the dot cells at %dx%d: %d → %d scales cached", sx, sy, cached, len(mp.r.dots))
	}
	if !mp.relaid {
		t.Fatal("a first map is a fresh layout")
	}
	if again := NewMap(mp.l, vault()); again.relaid {
		t.Fatal("the same graph again should keep the layout")
	}
}

func TestModelDimmedAndThemedTitleAndHints(t *testing.T) {
	trueColour(t)
	m := newModel(t, 80, 20)
	m.SetDimmed(true)
	th := theme.DefaultTheme().Greyed()
	rows := screen(m.View())
	for _, y := range []int{0, len(rows) - 1} {
		for x, c := range rows[y] {
			if c.r != ' ' && c.r != 0 && c.fg != "" && c.fg != rgb(th.Blue) && c.fg != rgb(th.GreyFg2) && c.fg != rgb(th.Fg) && c.fg != rgb(th.GreyFg) {
				t.Fatalf("row %d cell %d %q has colour %s, not a grey", y, x, c.r, c.fg)
			}
		}
	}
}

func TestModelFirstViewFitsTheWholeMapWhenItCan(t *testing.T) {
	m := newModel(t, 80, 20)
	if m.viewOffset() != m.r.Centred(ZoomOverview, 80, 18) {
		t.Fatalf("a map that fits should be shown whole and centred, offset %v", m.viewOffset())
	}
	m, _ = press(m, "+", "+")
	m.SetSize(40, 12) // the close map no longer fits: centre on the selection
	if x, y := selPane(m); x != 20 || y != 6 {
		t.Fatalf("a map too big should centre on the selection, got (%d,%d)", x, y)
	}
}

// n and N go round the links of the node they started from only while the
// selection is the one they reached: a selection reached any other way
// (keys, a click, the open note) has its own links gone round.
func TestModelNGoesRoundTheSelectionReachedAnotherWay(t *testing.T) {
	// Home, n → Inbox; l → Go; h → back on Inbox: n goes round Inbox's
	// links (Home, Ideas), not Home's (Inbox, study/Go, …).
	m := newModel(t, 80, 20)
	m, _ = press(m, "n")
	if m.Selected() != "Inbox.md" {
		t.Fatalf("setup: n from Home = %q", m.Selected())
	}
	m, _ = press(m, "l")
	if m.Selected() != "study/Go.md" {
		t.Fatalf("setup: l from Inbox = %q", m.Selected())
	}
	m, _ = press(m, "h")
	if m.Selected() != "Inbox.md" {
		t.Fatalf("setup: h from Go = %q", m.Selected())
	}
	if m, _ = press(m, "n"); m.Selected() != "Home.md" {
		t.Fatalf("n back on Inbox should go round Inbox's links, got %q", m.Selected())
	}

	// The same after the open note moved the selection onto where n left it.
	m = newModel(t, 80, 20)
	m, _ = press(m, "n")
	m.SetOpenNote("Inbox.md")
	if m, _ = press(m, "n"); m.Selected() != "Home.md" {
		t.Fatalf("n on the open note should go round its own links, got %q", m.Selected())
	}

	// And after esc came back to where n left the selection.
	m = newModel(t, 80, 20)
	m, _ = press(m, "n", "j", "esc")
	if m.Selected() != "Inbox.md" {
		t.Fatalf("setup: esc = %q", m.Selected())
	}
	if m, _ = press(m, "n"); m.Selected() != "Home.md" {
		t.Fatalf("n after esc should go round Inbox's links, got %q", m.Selected())
	}
}

// A wide (CJK) name takes two cells per character: a click on either half
// of any of them selects its note.
func TestModelClickOnEitherHalfOfAWideNameSelectsIt(t *testing.T) {
	const id = "study/日本語のノート.md"
	for _, half := range []int{0, 1, 2, 3} {
		// Rust sits left of it: select Rust and centre the view there.
		m := newModel(t, 80, 20)
		m, _ = press(m, "+", "+", "n", "n", "j", "0")
		if m.Selected() != "study/Rust.md" {
			t.Fatalf("setup: selection %q", m.Selected())
		}
		var x, y = -1, -1
		for row, l := range strings.Split(ansi.Strip(m.View()), "\n") {
			if i := strings.Index(l, "日本語"); i >= 0 && row > 0 {
				x, y = ansi.StringWidth(l[:i]), row
				break
			}
		}
		if x < 0 {
			t.Fatalf("the wide name is not drawn:\n%s", ansi.Strip(m.View()))
		}
		m, msg := click(m, x+half, y)
		if m.Selected() != id || msg != (SelectMsg{ID: id}) {
			t.Fatalf("click on cell %d of the name: %q %#v", half, m.Selected(), msg)
		}
	}
}

// At the overview of a small vault the dots are spread as far as the pane
// allows, up to the medium scale, and every dot is drawn (no name covers
// one, the selection's included) and a click on it selects its note.
func TestModelOverviewClickSelectsEveryDot(t *testing.T) {
	for _, c := range []struct{ w, h, sx, sy int }{{100, 30, 2, 2}, {40, 11, 1, 1}} {
		m := newModel(t, c.w, c.h)
		if sx, sy := m.scale(); sx != c.sx || sy != c.sy {
			t.Fatalf("setup: %dx%d at the overview scale %dx%d, want %dx%d", c.w, c.h, sx, sy, c.sx, c.sy)
		}
		clicked := 0
		for _, n := range m.l.Nodes {
			o := m.viewOffset()
			x, y := n.P.X*c.sx-o.X, n.P.Y*c.sy-o.Y+1
			if got := screen(m.View())[y][x].r; !strings.ContainsRune(string([]rune{glyphLeaf, glyphLinked, glyphHub, glyphUnresolved, glyphSelected}), got) {
				t.Fatalf("%dx%d: %q's dot at (%d,%d) shows %q:\n%s", c.w, c.h, n.ID, x, y, got, ansi.Strip(m.View()))
			}
			m, _ = click(m, x, y)
			if m.Selected() != n.ID {
				t.Fatalf("%dx%d: a click on %q's dot at (%d,%d) selected %q:\n%s", c.w, c.h, n.ID, x, y, m.Selected(), ansi.Strip(m.View()))
			}
			clicked++
		}
		if clicked < len(m.l.Nodes)/2 {
			t.Fatalf("%dx%d: only %d of %d dots were drawn where the overview scale puts them", c.w, c.h, clicked, len(m.l.Nodes))
		}
	}
}

// When the overview already uses the medium scale, + keeps the picture
// where it is.
func TestModelZoomInFromAMediumScaleOverviewKeepsTheView(t *testing.T) {
	m := newModel(t, 100, 24)
	before := m.viewOffset()
	m, _ = press(m, "+")
	if m.zoom != ZoomMedium || m.viewOffset() != before {
		t.Fatalf("+ moved the view %v → %v (zoom %d)", before, m.viewOffset(), m.zoom)
	}
}

// rowGap is how many rows apart Home (grid row 0) and work/Plan (grid row
// 4) are drawn.
func rowGap(m Model) int {
	_, y0 := m.screen("Home.md")
	_, y1 := m.screen("work/Plan.md")
	return y1 - y0
}

func TestModelOverviewScaleFollowsThePaneSize(t *testing.T) {
	m := newModel(t, 100, 24)
	big := rowGap(m)
	m.SetSize(40, 10)
	if small := rowGap(m); small >= big {
		t.Fatalf("a smaller pane should draw the overview smaller: rows apart %d → %d", big, small)
	}
	checkSelVisible(t, m, "smaller pane")
}

// A note added in a new cluster changes the overview's fit scale; the view
// is rescaled around the selection, so the selection does not jump.
func TestModelIncrementalMapAtTheOverviewKeepsTheSelectionInPlace(t *testing.T) {
	m := newModel(t, 64, 18)
	m, _ = press(m, "j")
	// Panned by hand, leaving room on Orphan's left for its name, which
	// goes there at 3×2 (on its right it would hide Ideas' dot).
	m.offset.X--
	x, y := selPane(m)
	sel, _ := m.l.Lookup(m.Selected())
	if sel.P.X == 0 || sel.P.Y == 0 {
		t.Fatalf("the fixture should select a node off the grid's edges, got %q at %v", sel.ID, sel.P)
	}
	gap := rowGap(m)
	g := vault()
	for i := range 12 {
		g.Nodes = append(g.Nodes, Node{ID: fmt.Sprintf("new/N%02d.md", i), Cluster: "new"})
	}
	m.SetGraph(g)
	if m.Layout().H == vaultLayout().H && m.Layout().W == vaultLayout().W {
		t.Fatal("the fixture should grow the grid")
	}
	if rowGap(m) == gap {
		t.Fatalf("the fixture should change the overview scale (rows apart %d)", gap)
	}
	if x2, y2 := selPane(m); x2 != x || y2 != y || m.Selected() != sel.ID {
		t.Fatalf("adding notes moved the selection %q from (%d,%d) to (%d,%d)", m.Selected(), x, y, x2, y2)
	}
}

// A resize changes the overview's fit scale; a view the user placed is
// rescaled around the selection, so the selection does not jump.
func TestModelResizeAtTheOverviewKeepsTheSelectionInPlace(t *testing.T) {
	m := newModel(t, 30, 8)
	m, _ = press(m, "j")
	// Panned by hand, leaving room on Orphan's left for its name, which
	// goes there at 3×2 (on its right it would hide Ideas' dot).
	m.offset.X--
	x, y := selPane(m)
	sel, _ := m.l.Lookup(m.Selected())
	if sel.P.X == 0 || sel.P.Y == 0 {
		t.Fatalf("the fixture should select a node off the grid's edges, got %q at %v", sel.ID, sel.P)
	}
	sx, sy := m.scale()
	m.SetSize(64, 18)
	if nx, ny := m.scale(); nx == sx || ny == sy {
		t.Fatalf("the fixture should change both scales: %dx%d → %dx%d", sx, sy, nx, ny)
	}
	if x2, y2 := selPane(m); x2 != x || y2 != y || m.Selected() != sel.ID {
		t.Fatalf("resizing moved the selection %q from (%d,%d) to (%d,%d)", m.Selected(), x, y, x2, y2)
	}
	checkSelVisible(t, m, "resize")
}

// A smaller pane keeps a placed overview's anchor where it was on screen,
// which may now be outside the pane: the view then scrolls the selection
// back into it.
func TestModelResizeAtTheOverviewKeepsTheSelectionShown(t *testing.T) {
	m := newModel(t, 80, 20)
	m, _ = press(m, "l", "l", "l") // to the right-hand cluster
	m.offset.X -= 10               // panned by hand: the selection nearer the right edge
	x, _ := selPane(m)
	m.SetSize(40, 10)
	if x < m.width {
		t.Fatalf("the fixture should leave the kept anchor outside the smaller pane: x=%d, width %d", x, m.width)
	}
	checkSelVisible(t, m, "smaller pane")
}

// titleModel is the vault plus two linked notes with long names, one in
// CJK, with longName or cjkName selected.
func titleModel(t *testing.T, sel, filter string, hidden bool) Model {
	t.Helper()
	g := vault()
	g.Nodes = append(g.Nodes, Node{ID: longName}, Node{ID: cjkName})
	g.Edges = append(g.Edges, Edge{"Home.md", longName}, Edge{"Home.md", cjkName})
	m := New(theme.DefaultTheme())
	m.SetSize(120, 20)
	m.SetGraph(g)
	m.SetOpenNote(sel)
	m.filter, m.hideOrphans, m.hideUnresolved = filter, hidden, hidden
	return m
}

const (
	longName = "A note whose name is much longer than any pane could hold.md"
	cjkName  = "日本語でとても長いノートの名前です.md"
)

// In a narrow pane the title row drops counters, lowest priority first
// (links, notes, orphans, unresolved; "hidden" ones never), then shortens
// the name, and only then cuts the filter. Whatever it cuts ends in "…",
// no counter is ever cut mid-word, and the row is exactly the pane wide.
func TestTitleFitsNarrowPanesWithoutCuttingWords(t *testing.T) {
	cases := []struct {
		name, sel, filter string
		hidden            bool
	}{
		{"long name", longName, "", false},
		{"CJK name", cjkName, "", false},
		{"CJK filter", longName, "日本語のノート", false},
		{"both hidden", longName, "plan", true},
		{"CJK name, CJK filter, hidden", cjkName, "日本語のノートを探す", true},
	}
	for _, c := range cases {
		m := titleModel(t, c.sel, c.filter, c.hidden)
		m.SetSize(1000, 20)
		full := strings.Split(strings.TrimRight(titleRow(m), " "), " · ")
		// full: name, notes, links, orphans, unresolved[, filter]
		if len(full) != 5+btoi(c.filter != "") {
			t.Fatalf("%s: unexpected full title %q", c.name, full)
		}
		// Below MinWidth the pane shows a message instead, but fitTitle's
		// tightest paths (the shortest name, a " · …" that no longer fits,
		// the final cut) are still checked on the title line itself.
		widths := []int{0, 1, 5, 10, 20, 29}
		for w := MinWidth; w <= 120; w++ {
			widths = append(widths, w)
		}
		for _, w := range widths {
			m.SetSize(w, 20)
			title := titleRow(m)
			if w < MinWidth {
				title = ansi.Strip(m.titleLine(m.th))
			}
			if got := ansi.StringWidth(title); got != w {
				t.Errorf("%s, width %d: title is %d wide: %q", c.name, w, got, title)
				continue
			}
			if w == 0 {
				continue
			}
			line := strings.TrimRight(title, " ")
			stopped := strings.HasSuffix(line, " · …") // a counter left out at the end
			line = strings.TrimSuffix(line, " · …")
			segs := strings.Split(line, " · ")
			shown := map[int]bool{}
			for i, s := range segs {
				switch k := slices.Index(full, s); {
				case i == 0 && k != 0 && (!strings.HasSuffix(s, "…") || !strings.HasPrefix(full[0], strings.TrimSuffix(s, "…"))):
					t.Errorf("%s, width %d: name shortened without …: %q", c.name, w, title)
				case i == 0:
				case k > 0:
					shown[k] = true
				case i == len(segs)-1 && strings.HasPrefix(s, `filter "`) && !strings.Contains(s, "\": "):
					if !strings.HasSuffix(s, "…") && !stopped {
						t.Errorf("%s, width %d: filter cut without …: %q", c.name, w, title)
					}
				default:
					t.Errorf("%s, width %d: segment %q is cut mid-word: %q", c.name, w, s, title)
				}
			}
			// " 󰈙 X…" is the shortest name.
			nameShort := ansi.StringWidth(segs[0]) <= 5 || segs[0] == full[0]
			// Drop order: links, notes, orphans, unresolved.
			for i, k := range []int{2, 1, 3, 4} {
				for _, later := range []int{2, 1, 3, 4}[i+1:] {
					if shown[k] && !shown[later] && !(c.hidden && later >= 3) {
						t.Errorf("%s, width %d: %q shown but %q dropped: %q", c.name, w, full[k], full[later], title)
					}
				}
			}
			droppable := shown[1] || shown[2] || (!c.hidden && (shown[3] || shown[4]))
			if c.filter != "" && !shown[5] && (droppable || !nameShort) {
				t.Errorf("%s, width %d: filter cut before the counters and the name: %q", c.name, w, title)
			}
			if c.hidden && (!shown[3] || !shown[4]) && (droppable || !nameShort) {
				t.Errorf("%s, width %d: a hidden counter left out before the others: %q", c.name, w, title)
			}
		}
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// A click on any cell of a neighbour's name drawn left of its dot selects
// that neighbour.
func TestModelClickOnALeftSideNameSelectsIt(t *testing.T) {
	nodes := []Placed{
		{Node: Node{ID: "Ab.md"}, P: Point{22, 0}},
		{Node: Node{ID: "日本.md"}, P: Point{29, 0}},
		{Node: Node{ID: "S.md"}, P: Point{30, 0}},
		{Node: Node{ID: "Cd.md"}, P: Point{40, 0}},
		{Node: Node{ID: "Far.md"}, P: Point{59, 11}},
	}
	edges := []Edge{{"Ab.md", "S.md"}, {"日本.md", "S.md"}, {"S.md", "Cd.md"}, {"日本.md", "Far.md"}}
	l := placedLayout(nodes, edges)
	g := Graph{Edges: edges}
	for _, n := range nodes {
		g.Nodes = append(g.Nodes, n.Node)
	}
	// The map fits the 60×12 map area at 1×1, so it is drawn from map
	// cell (0, 0), below the title row: Ab's name is at 19-20, 日本's at
	// 24-27.
	for _, c := range []struct {
		x  int
		id string
	}{{19, "Ab.md"}, {20, "Ab.md"}, {24, "日本.md"}, {25, "日本.md"}, {26, "日本.md"}, {27, "日本.md"}} {
		m := New(theme.DefaultTheme())
		m.SetSize(60, 14)
		m.SetFocused(true)
		m.SetMap(Map{g: g, l: l, r: NewRenderer(l, edges), relaid: true})
		m.SetOpenNote("S.md")
		m, msg := click(m, c.x, 1)
		if m.Selected() != c.id || msg != (SelectMsg{ID: c.id}) {
			t.Errorf("click at %d: selection %q, msg %#v, want %q:\n%s", c.x, m.Selected(), msg, c.id, ansi.Strip(m.View()))
		}
	}
}

// Moving onto a note whose name goes left of its dot scrolls that name
// into view, so it is drawn in full there instead of being cut on the
// right.
func TestModelRevealShowsALeftSideSelectionName(t *testing.T) {
	m := newModel(t, 80, 40) // Plan, 16 rows below Home at close, in view
	m, _ = press(m, "+", "+")
	x, y := m.screen("work/Plan.md")
	m, _ = click(m, x, y+1)
	// Pan Meeting's dot, right below Plan's, to the pane's third column.
	m.settle()
	mx, _ := m.screen("work/Meeting.md")
	m.offset.X += mx - 2
	m, _ = press(m, "j")
	if m.Selected() != "work/Meeting.md" {
		t.Fatalf("setup: j from Plan selected %q", m.Selected())
	}
	x, y = selPane(m)
	rows := screen(m.View())
	if x < 9 || rowText(rows, y, x-9, x, x) != " Meeting *" {
		t.Fatalf("Meeting's dot at %d, want its name whole on its left:\n%s", x, ansi.Strip(m.View()))
	}
}
