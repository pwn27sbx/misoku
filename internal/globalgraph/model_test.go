package globalgraph

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// The fixture's map (vault(), grid cells):
//
//	x:  1      2           8     9
//	y0: Home   Inbox       Go    study/Notes
//	y1: Orphan Ideas       Rust  日本語のノート
//	y4: Plan   Tasks
//	y5: Meeting work/Notes

func newModel(t testing.TB, w, h int) Model {
	t.Helper()
	m := New(theme.DefaultTheme())
	m.SetSize(w, h)
	m.SetFocused(true)
	m.SetGraph(vault())
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
	for _, k := range []string{"q", "x", "g", ":", "/", "tab", "H", "L"} {
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
	// Inbox's dot is the cell right of the selected Home's.
	m := newModel(t, 80, 20)
	x, y := selPane(m)
	click := tea.MouseMsg{X: x + 1, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
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

func TestNewMapWarmsTheRenderer(t *testing.T) {
	mp := NewMap(Layout{}, vault())
	for z := range zoomLevels {
		if mp.r.dots[z] == nil || z != ZoomOverview && mp.r.faint[z] == nil {
			t.Fatalf("zoom %d not warmed", z)
		}
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
