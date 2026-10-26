package globalgraph

// The model is the interactive map: a title row with the vault's counts,
// the map, and a row of buttons and key hints. Keys follow the local graph
// where they overlap (enter, + -, 0, esc, the buttons); h j k l move to the
// nearest node on that side, since this map has no tree to walk.

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"
)

// OpenMsg asks the app to open a node picked in the map. Unresolved is set
// for a link target with no note behind it yet.
//
// It mirrors graph.OpenMsg rather than reusing it: the graph package
// imports internal/core, and the map should not, and the app has to tell
// an unresolved target from a note.
type OpenMsg struct {
	ID         string
	Unresolved bool
}

// SelectMsg reports the selected node, so the app can preview it.
type SelectMsg struct{ ID string }

// Map is a graph laid out and ready to draw. NewMap does the slow part, so
// it can run in a tea.Cmd; SetMap hands the result to the model.
type Map struct {
	g      Graph
	l      Layout
	r      *Renderer
	relaid bool // positions were not kept: the whole map was laid out anew
}

// NewMap lays g out keeping the nodes of prev where they were (see
// Update) and fills the renderer's caches. It only reads prev, so prev may
// come from a model still in use.
func NewMap(prev Layout, g Graph) Map {
	l, kept := Build(g), false
	if len(prev.Nodes) > 0 {
		l, kept = Update(prev, g)
	}
	r := NewRenderer(l, g.Edges)
	r.warm()
	return Map{g: g, l: l, r: r, relaid: !kept}
}

// Model is the global graph view.
type Model struct {
	th theme.Theme
	l  Layout
	r  *Renderer

	notes, links, orphans, unresolved int

	open    string   // the note open when the graph was opened
	sel     string   // selected node, "" for none
	history []string // earlier selections, for esc
	ring    string   // the node whose neighbours n and N go round
	ringAt  string   // where n or N left the selection last

	zoom          Zoom
	width, height int
	focused       bool
	dimmed        bool
	offset        Point // map cell, at zoom, at the map's top-left
	placed        bool  // offset was set for this size and map

	dragging bool
	dragAt   Point
}

// New makes an empty map; SetMap or SetGraph fills it.
func New(th theme.Theme) Model {
	return Model{th: th, r: NewRenderer(Layout{}, nil)}
}

// SetTheme switches colours.
func (m *Model) SetTheme(th theme.Theme) { m.th = th }

// SetFocused shows whether keys go to the map.
func (m *Model) SetFocused(f bool) { m.focused = f }

// SetDimmed greys the map out while another pane has the focus.
func (m *Model) SetDimmed(d bool) { m.dimmed = d }

// SetSize sets the pane size; the view is placed afresh (see viewOffset).
func (m *Model) SetSize(w, h int) {
	if w != m.width || h != m.height {
		m.width, m.height, m.placed = w, h, false
	}
}

// SetGraph lays g out on the Update loop; NewMap and SetMap do the same in
// two steps.
func (m *Model) SetGraph(g Graph) { m.SetMap(NewMap(m.l, g)) }

// Layout is the current layout, for NewMap.
func (m Model) Layout() Layout { return m.l }

// SetMap replaces the map. The selection stays on the same node while it
// exists (otherwise it falls back to the open note); when the whole map
// was laid out anew the view is placed afresh, on the selection unless the
// whole map fits.
func (m *Model) SetMap(mp Map) {
	m.l, m.r = mp.l, mp.r
	m.notes, m.links, m.orphans, m.unresolved = 0, 0, 0, 0
	for i, n := range m.l.Nodes {
		switch {
		case n.Unresolved:
			m.unresolved++
		case n.Degree == 0:
			m.notes++
			m.orphans++
		default:
			m.notes++
		}
		m.links += len(m.r.out[i])
	}
	if !m.exists(m.sel) {
		m.sel = ""
		if m.exists(m.open) {
			m.sel = m.open
		}
	}
	if mp.relaid {
		m.placed = false
	}
}

// SetOpenNote marks the note open when the graph was opened and selects
// it. It starts a fresh history: esc only retraces moves made in the map.
func (m *Model) SetOpenNote(id string) {
	m.open, m.history = id, nil
	if m.exists(id) {
		m.sel, m.placed = id, false
	}
}

// Selected returns the selected node's ID, "" for none.
func (m Model) Selected() string { return m.sel }

func (m Model) exists(id string) bool {
	_, ok := m.l.Lookup(id)
	return ok && id != ""
}

// mapRows is the height of the map between the title and hint rows.
func (m Model) mapRows() int { return max(m.height-2, 1) }

// tooSmall reports whether the pane shows a message instead of the map.
func (m Model) tooSmall() bool { return m.width < MinWidth || m.height < MinHeight }

// Update handles keys and mouse events (coordinates relative to the pane).
// h j k l and the arrows move to the nearest node on that side; n and N go
// round the selection's links; enter opens; + and - zoom; 0 centres the
// view on the selection; esc goes back to the previous selection.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if len(m.l.Nodes) == 0 {
		return m, nil
	}
	var cmd tea.Cmd
	before := m.sel
	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmd = m.key(msg.String())
	case tea.MouseMsg:
		if m.tooSmall() {
			return m, nil
		}
		cmd = m.mouse(msg)
	default:
		return m, nil
	}
	if cmd == nil && m.sel != before {
		id := m.sel
		cmd = func() tea.Msg { return SelectMsg{ID: id} }
	}
	return m, cmd
}

// directions maps a key to the side it moves to.
var directions = map[string]Point{
	"h": {-1, 0}, "left": {-1, 0},
	"l": {1, 0}, "right": {1, 0},
	"k": {0, -1}, "up": {0, -1},
	"j": {0, 1}, "down": {0, 1},
}

func (m *Model) key(k string) tea.Cmd {
	if d, ok := directions[k]; ok {
		m.pick(m.toward(d))
		return nil
	}
	switch k {
	case "n":
		m.cycle(1)
	case "N":
		m.cycle(-1)
	case "enter":
		if m.sel == "" {
			return nil
		}
		n, _ := m.l.Lookup(m.sel)
		return func() tea.Msg { return OpenMsg{ID: n.ID, Unresolved: n.Unresolved} }
	case "+":
		m.act("zoom-in")
	case "-":
		m.act("zoom-out")
	case "0":
		m.act("centre")
	case "esc":
		m.act("back")
	}
	return nil
}

// pick selects id, remembering the old selection for esc, and scrolls it
// into view.
func (m *Model) pick(id string) {
	if id == "" || id == m.sel {
		return
	}
	if m.sel != "" {
		m.history = append(m.history, m.sel)
	}
	m.sel = id
	m.reveal()
}

// toward is the nearest node on side d of the selection, or, with nothing
// selected, the node nearest the middle of the view. Distances are on the
// map's grid, where a row counts as cellAspect columns, so the choice does
// not change with the zoom. Nodes within 45°
// of the side come first; ties go to the smaller angle, then the busier
// node, then the ID.
func (m *Model) toward(d Point) string {
	if m.sel == "" {
		return m.nearestToMiddle()
	}
	from, _ := m.l.Lookup(m.sel)
	type cand struct {
		outside     bool
		dist, angle float64
		degree      int
		id          string
	}
	var best *cand
	for _, n := range m.l.Nodes {
		dx, dy := float64(n.P.X-from.P.X), cellAspect*float64(n.P.Y-from.P.Y)
		along, across := dx*float64(d.X)+dy*float64(d.Y), math.Abs(dx*float64(d.Y)+dy*float64(d.X))
		if along <= 0 {
			continue
		}
		c := cand{across > along, math.Hypot(dx, dy), across / along, n.Degree, n.ID}
		if best == nil || cmp.Or(cmpBool(c.outside, best.outside), cmp.Compare(c.dist, best.dist),
			cmp.Compare(c.angle, best.angle), cmp.Compare(best.degree, c.degree), cmp.Compare(c.id, best.id)) < 0 {
			best = &c
		}
	}
	if best == nil {
		return ""
	}
	return best.id
}

// nearestToMiddle is the node nearest the middle of the view; a screen row
// counts as cellAspect columns.
func (m *Model) nearestToMiddle() string {
	m.settle()
	sx, sy := m.zoom.Scale()
	cx, cy := float64(m.offset.X)+float64(m.width)/2, float64(m.offset.Y)+float64(m.mapRows())/2
	best, bestD := "", math.Inf(1)
	for _, n := range m.l.Nodes {
		dx, dy := float64(n.P.X*sx)-cx, cellAspect*(float64(n.P.Y*sy)-cy)
		if d := dx*dx + dy*dy; d < bestD {
			best, bestD = n.ID, d
		}
	}
	return best
}

// cycle selects the next (d +1) or previous (-1) neighbour of the node
// whose links are being gone round: the selection, unless the selection
// was itself reached by n or N.
func (m *Model) cycle(d int) {
	if m.sel == "" {
		m.pick(m.nearestToMiddle())
		return
	}
	hub := m.sel
	if m.ringAt == m.sel && m.exists(m.ring) {
		hub = m.ring
	}
	near := m.neighbours(hub)
	if len(near) == 0 {
		return
	}
	i := slices.Index(near, m.sel) // -1 when the hub itself is selected
	if i < 0 && d < 0 {
		i = 0
	}
	next := near[((i+d)%len(near)+len(near))%len(near)]
	m.pick(next)
	m.ring, m.ringAt = hub, m.sel
}

// neighbours lists the nodes linked to or from id, in ID order.
func (m *Model) neighbours(id string) []string {
	i := m.l.index[id]
	set := map[int]bool{}
	for _, j := range m.r.out[i] {
		set[int(j)] = true
	}
	for _, j := range m.r.in[i] {
		set[int(j)] = true
	}
	var ids []string
	for _, j := range sortedIndexes(set) {
		ids = append(ids, m.l.Nodes[j].ID)
	}
	return ids
}

// act runs a button.
func (m *Model) act(action string) {
	switch action {
	case "zoom-in":
		m.zoomTo(m.zoom+1, m.anchor())
	case "zoom-out":
		m.zoomTo(m.zoom-1, m.anchor())
	case "centre":
		m.offset, m.placed = m.onSelection(), true
	case "back":
		for len(m.history) > 0 {
			prev := m.history[len(m.history)-1]
			m.history = m.history[:len(m.history)-1]
			if m.exists(prev) { // skip nodes removed since
				m.sel = prev
				m.reveal()
				break
			}
		}
	case "reset":
		m.zoom, m.offset, m.placed = ZoomOverview, m.r.Centred(ZoomOverview, m.width, m.mapRows()), true
		m.reveal()
	}
}

// anchor is the map point that keeps its place on screen when the keys
// zoom: the selection, or the middle of the view.
func (m *Model) anchor() Point {
	m.settle()
	if m.sel != "" {
		if x, y := m.screen(m.sel); x >= 0 && y >= 0 && x < m.width && y < m.mapRows() {
			return Point{x, y}
		}
	}
	return Point{m.width / 2, m.mapRows() / 2}
}

// zoomTo changes the zoom keeping the map point under screen point at in
// place, then scrolls the selection into view.
func (m *Model) zoomTo(z Zoom, at Point) {
	if z < ZoomOverview || z >= zoomLevels || z == m.zoom {
		return
	}
	m.settle()
	sx, sy := m.zoom.Scale()
	gx, gy := float64(at.X+m.offset.X)/float64(sx), float64(at.Y+m.offset.Y)/float64(sy)
	m.zoom = z
	sx, sy = z.Scale()
	m.offset = Point{int(math.Round(gx*float64(sx))) - at.X, int(math.Round(gy*float64(sy))) - at.Y}
	m.reveal()
}

// screen is where node id's dot is on the map area of the pane.
func (m *Model) screen(id string) (x, y int) {
	n, _ := m.l.Lookup(id)
	sx, sy := m.zoom.Scale()
	o := m.viewOffset()
	return n.P.X*sx - o.X, n.P.Y*sy - o.Y
}

// viewOffset is where the user panned to or, until then, the whole map
// centred when it fits the pane, otherwise the view centred on the
// selection.
func (m Model) viewOffset() Point {
	if m.placed {
		return m.offset
	}
	sx, sy := m.zoom.Scale()
	if (m.l.W-1)*sx < m.width && (m.l.H-1)*sy < m.mapRows() {
		return m.r.Centred(m.zoom, m.width, m.mapRows())
	}
	return m.onSelection()
}

// onSelection is the view centred on the selection, or on the whole map
// with nothing selected.
func (m Model) onSelection() Point {
	n, ok := m.l.Lookup(m.sel)
	if !ok {
		return m.r.Centred(m.zoom, m.width, m.mapRows())
	}
	sx, sy := m.zoom.Scale()
	return Point{n.P.X*sx - m.width/2, n.P.Y*sy - m.mapRows()/2}
}

// settle fixes the view before it is moved.
func (m *Model) settle() {
	m.offset, m.placed = m.viewOffset(), true
}

// reveal scrolls the view as little as it takes to show the selection's
// dot and, when it fits, its name.
func (m *Model) reveal() {
	n, ok := m.l.Lookup(m.sel)
	if !ok {
		return
	}
	m.settle()
	sx, sy := m.zoom.Scale()
	x0, y := n.P.X*sx, n.P.Y*sy
	x1 := x0 + 1 + min(ansi.StringWidth(n.Label), maxLabelWidth)
	if x1-x0 >= m.width {
		x1 = x0
	}
	if m.offset.X > x0 {
		m.offset.X = x0
	} else if m.offset.X < x1-m.width+1 {
		m.offset.X = x1 - m.width + 1
	}
	if rows := m.mapRows(); m.offset.Y > y {
		m.offset.Y = y
	} else if m.offset.Y < y-rows+1 {
		m.offset.Y = y - rows + 1
	}
}

func (m *Model) mouse(msg tea.MouseMsg) tea.Cmd {
	at := Point{msg.X, msg.Y - 1} // on the map area
	switch {
	case msg.Button == tea.MouseButtonWheelUp:
		m.zoomTo(m.zoom+1, at)
	case msg.Button == tea.MouseButtonWheelDown:
		m.zoomTo(m.zoom-1, at)
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && msg.Y == m.height-1:
		for _, b := range m.buttons() {
			if msg.X >= b.x0 && msg.X < b.x1 && b.enabled {
				m.act(b.action)
			}
		}
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && msg.Y > 0:
		if i, ok := m.r.nodeAt(m.frame(), at.X, at.Y); ok {
			n := m.l.Nodes[i]
			if n.ID == m.sel {
				return func() tea.Msg { return OpenMsg{ID: n.ID, Unresolved: n.Unresolved} }
			}
			m.pick(n.ID)
			return nil
		}
		m.settle()
		m.dragging, m.dragAt = true, Point{msg.X, msg.Y}
	case msg.Action == tea.MouseActionMotion && m.dragging:
		m.offset.X -= msg.X - m.dragAt.X
		m.offset.Y -= msg.Y - m.dragAt.Y
		m.dragAt = Point{msg.X, msg.Y}
	case msg.Action == tea.MouseActionRelease:
		m.dragging = false
	}
	return nil
}

// frame is the map's part of the pane.
func (m Model) frame() Frame {
	return Frame{W: m.width, H: m.mapRows(), Offset: m.viewOffset(), Zoom: m.zoom, Selected: m.sel,
		Open: m.open, Focused: m.focused, Dimmed: m.dimmed, Theme: m.th}
}

// View renders the map at exactly the pane's size: title, map and hints,
// or a message when there are no notes or the pane is too small.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	if len(m.l.Nodes) == 0 || m.tooSmall() {
		return m.r.Render(Frame{W: m.width, H: m.height, Dimmed: m.dimmed, Theme: m.th})
	}
	th := m.th
	if m.dimmed {
		th = th.Greyed()
	}
	f := m.frame()
	lines := []string{m.titleLine(th), m.r.picture(f, th), m.helpLine(th)}
	return ui.Fit(strings.Join(lines, "\n"), m.width, m.height, lipgloss.NewStyle())
}

// titleLine names the selection and counts the vault.
func (m Model) titleLine(th theme.Theme) string {
	name := "All notes"
	if n, ok := m.l.Lookup(m.sel); ok {
		name = n.Label
	}
	info := " · " + plural(m.notes, "note") + " · " + plural(m.links, "link") + " · " +
		plural(m.orphans, "orphan") + " · " + fmt.Sprintf("%d unresolved", m.unresolved)
	head := lipgloss.NewStyle().Foreground(th.Blue).Bold(true).Render("󰈙 "+ui.Truncate(name, 40)) +
		lipgloss.NewStyle().Foreground(th.GreyFg2).Render(info)
	return ui.FitLine(" "+head, m.width, lipgloss.NewStyle())
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// button is a clickable control on the hint row.
type button struct {
	label, action string
	x0, x1        int
	enabled       bool
}

var (
	longLabels  = [5]string{" − ", " + ", " 0 centre ", " esc back ", " reset "}
	shortLabels = [5]string{" − ", " + ", " 0 ", " esc ", " reset "}
)

// hints are the hint row's layouts, widest first: the buttons' labels (in
// buttons() order), then the other keys and what the mouse does. The
// first that fits the pane is used; the last fits MinWidth.
var hints = []struct {
	labels [5]string
	tail   string
}{
	{longLabels, "  hjkl move · n/N links · enter open · click: preview, again: open · scroll: zoom · drag: move"},
	{longLabels, "  hjkl move · n/N links · enter open · drag: move"},
	{longLabels, "  hjkl move · n/N links · enter open"},
	{longLabels, "  hjkl · n/N · enter"},
	{shortLabels, "  hjkl · n/N · enter"},
	{shortLabels, "  hjkl · enter"},
	{shortLabels, ""},
}

func (m Model) hint() (labels [5]string, tail string) {
	for _, h := range hints {
		w := 1 + len(h.labels) - 1 + ansi.StringWidth(h.tail)
		for _, l := range h.labels {
			w += ansi.StringWidth(l)
		}
		if w <= m.width {
			return h.labels, h.tail
		}
	}
	last := hints[len(hints)-1]
	return last.labels, last.tail
}

// buttons sit at the start of the hint row.
func (m Model) buttons() []button {
	labels, _ := m.hint()
	bs := []button{
		{label: labels[0], action: "zoom-out", enabled: m.zoom > ZoomOverview},
		{label: labels[1], action: "zoom-in", enabled: m.zoom < zoomLevels-1},
		{label: labels[2], action: "centre", enabled: m.sel != ""},
		{label: labels[3], action: "back", enabled: len(m.history) > 0},
		{label: labels[4], action: "reset", enabled: true},
	}
	x := 1
	for i := range bs {
		bs[i].x0 = x
		x += ansi.StringWidth(bs[i].label)
		bs[i].x1 = x
		x++
	}
	return bs
}

// helpLine is the hint row: the buttons, then the other keys.
func (m Model) helpLine(th theme.Theme) string {
	var b strings.Builder
	b.WriteString(" ")
	for i, bt := range m.buttons() {
		st := lipgloss.NewStyle().Background(th.OneBg).Foreground(th.Fg)
		if !bt.enabled {
			st = st.Foreground(th.GreyFg)
		}
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(st.Render(bt.label))
	}
	_, tail := m.hint()
	b.WriteString(lipgloss.NewStyle().Foreground(th.GreyFg2).Render(tail))
	return ui.FitLine(b.String(), m.width, lipgloss.NewStyle())
}
