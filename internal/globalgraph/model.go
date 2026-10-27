package globalgraph

// The model is the interactive map: a title row with the vault's counts,
// the map, and a row of buttons and key hints. Keys follow the local graph
// where they overlap (enter, + -, 0, esc, the buttons); h j k l move to the
// nearest node on that side, since this map has no tree to walk.
// / filters the map, o and u hide orphans and unresolved targets (see
// filter.go).

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
	ringAt  string   // where n or N left the selection; "" once it moved any other way

	zoom          Zoom
	width, height int
	focused       bool
	dimmed        bool
	offset        Point // map cell, at zoom, at the map's top-left
	placed        bool  // offset was set for this size and map

	dragging bool
	dragAt   Point

	filter                      string // text of the filter, "" for none
	typing                      bool   // the filter's prompt is open
	hideOrphans, hideUnresolved bool
}

// Typing reports whether the filter's prompt is open: every key is text.
func (m Model) Typing() bool { return m.typing }

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

// SetSize sets the pane size. A view the user placed at the overview keeps
// its anchor in place (see keepAnchor); otherwise the view is placed afresh
// (see viewOffset).
func (m *Model) SetSize(w, h int) {
	if w == m.width && h == m.height {
		return
	}
	if m.keepAnchor(func() { m.width, m.height = w, h }) {
		m.reveal()
	} else {
		m.placed = false
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
	// A map that grows or shrinks may change the overview's fit scale. A
	// map Update moved as a whole (see Layout.moved) moves a placed view
	// with it, so nothing jumps on screen.
	replace := func() {
		if m.placed {
			k := m.scale()
			m.offset.X += k.of(mp.l.moved.X)
			m.offset.Y += k.of(mp.l.moved.Y)
		}
		m.l, m.r = mp.l, mp.r
	}
	if mp.relaid {
		replace()
	} else {
		m.keepAnchor(replace)
	}
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
	if len(m.l.Nodes) == 0 { // an empty map takes no keys: nothing could close the prompt
		m.typing, m.filter = false, ""
	}
	m.keepSelectionShown()
}

// SetOpenNote marks the note open when the graph was opened and selects
// it. It starts a fresh history: esc only retraces moves made in the map.
func (m *Model) SetOpenNote(id string) {
	m.open, m.history = id, nil
	if m.exists(id) {
		m.sel, m.placed = id, false
		m.ringAt = ""
	}
	m.keepSelectionShown()
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
// view on the selection (with none, shows the whole map again); esc clears
// the filter or else goes back to the previous selection; / opens the
// filter's prompt, which then takes every key; o and u hide or show
// orphans and unresolved targets.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if len(m.l.Nodes) == 0 {
		return m, nil
	}
	var cmd tea.Cmd
	before := m.sel
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.typing {
			m.promptKey(msg)
			break
		}
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
		if m.filter != "" {
			m.filter = ""
			return nil
		}
		m.act("back")
	case "/":
		m.act("filter")
	case "o":
		m.act("orphans")
	case "u":
		m.act("unresolved")
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
	m.sel, m.ringAt = id, "" // cycle sets ringAt again when it was the one moving
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
		if !m.selectable(n.ID) {
			continue
		}
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
	k := m.scale()
	cx, cy := float64(m.offset.X)+float64(m.width)/2, float64(m.offset.Y)+float64(m.mapRows())/2
	best, bestD := "", math.Inf(1)
	for _, n := range m.l.Nodes {
		dx, dy := float64(k.of(n.P.X))-cx, cellAspect*(float64(k.of(n.P.Y))-cy)
		if d := dx*dx + dy*dy; d < bestD && m.selectable(n.ID) {
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
	if m.ringAt == m.sel && m.selectable(m.ring) {
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

// neighbours lists the nodes shown linked to or from id, in ID order.
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
		if id := m.l.Nodes[j].ID; m.selectable(id) {
			ids = append(ids, id)
		}
	}
	return ids
}

// act runs a button.
func (m *Model) act(action string) {
	switch action {
	case "zoom-in", "zoom-out":
		// With nothing selected and the view not moved by hand, the view
		// keeps following viewOffset at the new zoom: the whole map when it
		// fits, else the biggest galaxy, never the empty middle between
		// galaxies.
		follow := !m.placed && m.sel == ""
		z := m.zoom + 1
		if action == "zoom-out" {
			z = m.zoom - 1
		}
		m.zoomTo(z, m.anchor())
		if follow {
			m.placed = false
		}
	case "centre":
		if m.sel == "" {
			m.placed = false // the whole map when it fits, see viewOffset
		} else {
			m.offset, m.placed = m.onSelection(), true
		}
	case "back":
		// Nodes removed since are dropped; hidden ones are skipped but kept,
		// for when the toggle shows them again.
		for i := len(m.history) - 1; i >= 0; i-- {
			prev := m.history[i]
			if !m.exists(prev) {
				m.history = slices.Delete(m.history, i, i+1)
			} else if m.selectable(prev) {
				m.history = slices.Delete(m.history, i, i+1)
				m.sel = prev
				m.reveal()
				break
			}
		}
	case "reset":
		m.zoom, m.offset, m.placed = ZoomOverview, m.r.Centred(ZoomOverview, m.width, m.mapRows()), true
		m.reveal()
	case "filter":
		m.typing = true
	case "orphans", "unresolved":
		m.toggle(action)
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
	k := m.scale()
	m.zoom = z
	m.rescale(at, k)
	m.reveal()
}

// keepAnchor applies change, which may change the overview's fit scale
// (a new map or pane size). A view the user placed at the overview is
// rescaled around its anchor (the selection, or the middle) so it does not
// jump; keepAnchor reports whether it was.
func (m *Model) keepAnchor(change func()) bool {
	keep := m.placed && m.zoom == ZoomOverview && len(m.l.Nodes) > 0
	var at Point
	k := m.scale()
	if keep {
		at = m.anchor()
	}
	change()
	if keep {
		m.rescale(at, k)
	}
	return keep
}

// rescale keeps the map point under screen point at in place after the
// scale changed from k to the current one.
func (m *Model) rescale(at Point, k scale) {
	f := float64(m.scale()) / float64(k)
	m.offset = Point{int(math.Round(float64(at.X+m.offset.X)*f)) - at.X, int(math.Round(float64(at.Y+m.offset.Y)*f)) - at.Y}
}

// scale is how many screen cells one grid cell spans in the map area.
func (m Model) scale() scale { return m.r.scale(m.zoom, m.width, m.mapRows()) }

// screen is where node id's dot is on the map area of the pane.
func (m *Model) screen(id string) (x, y int) {
	n, _ := m.l.Lookup(id)
	k := m.scale()
	o := m.viewOffset()
	return k.of(n.P.X) - o.X, k.of(n.P.Y) - o.Y
}

// viewOffset is where the user panned to or, until then, the whole map
// centred when it fits the pane, otherwise the view centred on the
// selection.
func (m Model) viewOffset() Point {
	if m.placed {
		return m.offset
	}
	if k := m.scale(); k.of(m.l.W-1) < m.width && k.of(m.l.H-1) < m.mapRows() {
		return m.r.Centred(m.zoom, m.width, m.mapRows())
	}
	return m.onSelection()
}

// onSelection is the view centred on the selection or, with nothing
// selected, on the vault's sun or, on a map placed by hand, the centre of
// the biggest galaxy (most nodes, ties by key): the middle of such a map
// may be empty space between galaxies.
func (m Model) onSelection() Point {
	p, ok := Point{}, false
	if n, found := m.l.Lookup(m.sel); found {
		p, ok = n.P, true
	} else if m.l.solar {
		p, ok = m.l.Sun, true
	} else if c, found := m.biggestGalaxy(); found {
		p, ok = c.Centre, true
	}
	if !ok {
		return m.r.Centred(m.zoom, m.width, m.mapRows())
	}
	k := m.scale()
	return Point{k.of(p.X) - m.width/2, k.of(p.Y) - m.mapRows()/2}
}

// biggestGalaxy is the cluster with the most nodes; Clusters are sorted by
// key, so the first of equals wins.
func (m Model) biggestGalaxy() (Cluster, bool) {
	size := map[string]int{}
	for _, n := range m.l.Nodes {
		size[n.Cluster]++
	}
	best := -1
	for i, c := range m.l.Clusters {
		if best < 0 || size[c.Key] > size[m.l.Clusters[best].Key] {
			best = i
		}
	}
	if best < 0 {
		return Cluster{}, false
	}
	return m.l.Clusters[best], true
}

// settle fixes the view before it is moved.
func (m *Model) settle() {
	m.offset, m.placed = m.viewOffset(), true
}

// reveal scrolls the view as little as it takes to show the selection's
// dot and, when it fits, its name, on whichever side of the dot it goes.
func (m *Model) reveal() {
	n, ok := m.l.Lookup(m.sel)
	if !ok {
		return
	}
	m.settle()
	k := m.scale()
	dot, y := k.of(n.P.X), k.of(n.P.Y)
	x0, x1 := m.r.selSpan(m.frame())
	if x1-x0 >= m.width {
		x0, x1 = dot, dot
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
		if m.typing { // the row is the prompt
			break
		}
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
		Open: m.open, Focused: m.focused, Dimmed: m.dimmed, Theme: m.th,
		Filter: m.filter, HideOrphans: m.hideOrphans, HideUnresolved: m.hideUnresolved}
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

// titleLine names the selection and counts the vault, hidden nodes
// included: the toggles only say they hide them. A filter shows with its
// number of matches. In a narrow pane fitTitle decides what is left out.
func (m Model) titleLine(th theme.Theme) string {
	name := "All notes"
	if n, ok := m.l.Lookup(m.sel); ok {
		name = n.Label
	}
	orphans, unresolved := plural(m.orphans, "orphan"), fmt.Sprintf("%d unresolved", m.unresolved)
	orphansDrop, unresolvedDrop := 3, 4
	if m.hideOrphans {
		orphans, orphansDrop = orphans+" hidden", 0
	}
	if m.hideUnresolved {
		unresolved, unresolvedDrop = unresolved+" hidden", 0
	}
	parts := []titlePart{
		{text: " "},
		{text: "󰈙 " + ui.Truncate(name, 40), style: titleName, cut: true, keep: 2},
		{text: " · " + plural(m.notes, "note"), style: titleInfo, drop: 2},
		{text: " · " + plural(m.links, "link"), style: titleInfo, drop: 1},
		{text: " · " + orphans, style: titleInfo, drop: orphansDrop},
		{text: " · " + unresolved, style: titleInfo, drop: unresolvedDrop},
	}
	if m.filter != "" {
		parts = append(parts,
			titlePart{text: fmt.Sprintf(" · filter %q", m.filter), style: titleInfo, cut: true, keep: ansi.StringWidth(` · filter "`)},
			titlePart{text: ": " + plural(m.matches(), "match"), style: titleInfo})
	}
	styles := [...]lipgloss.Style{
		titlePlain: lipgloss.NewStyle(),
		titleName:  lipgloss.NewStyle().Foreground(th.Blue).Bold(true),
		titleInfo:  lipgloss.NewStyle().Foreground(th.GreyFg2),
	}
	return fitTitle(parts, m.width, styles)
}

// The title row's styles.
const (
	titlePlain = iota
	titleName
	titleInfo
)

// titlePart is one piece of the title row. Counters are whole words: the
// row may leave them out but never cuts them; only the text the user wrote
// (the note's name, the filter) may be cut.
type titlePart struct {
	text  string
	style int
	cut   bool // may be cut with a trailing "…"
	keep  int  // columns kept whole when cut: "…" goes after them
	drop  int  // left out when the row is too wide, 1 first; 0 never
}

// fitTitle renders parts in exactly w columns. When they do not fit it
// leaves out the counters by their drop order, then shortens the name
// (parts[1]), and only then cuts the row with a trailing "…" — inside the
// name or the filter, or before a counter, never through one.
func fitTitle(parts []titlePart, w int, styles [3]lipgloss.Style) string {
	width := func(ps []titlePart) int {
		n := 0
		for _, p := range ps {
			n += ansi.StringWidth(p.text)
		}
		return n
	}
	for d := 1; d <= 4 && width(parts) > w; d++ {
		parts = slices.DeleteFunc(parts, func(p titlePart) bool { return p.drop == d })
	}
	if over := width(parts) - w; over > 0 {
		name := &parts[1]
		const shortest = 4 // the icon, a space, one letter and "…"
		name.text = ui.Truncate(name.text, max(ansi.StringWidth(name.text)-over, shortest))
	}
	render := func(ps []titlePart) string {
		var b strings.Builder
		for i := 0; i < len(ps); {
			j, text := i, ""
			for ; j < len(ps) && ps[j].style == ps[i].style; j++ {
				text += ps[j].text
			}
			b.WriteString(styles[ps[i].style].Render(text))
			i = j
		}
		return b.String()
	}
	fill := styles[titlePlain]
	if width(parts) <= w {
		return ui.FitLine(render(parts), w, fill)
	}
	// The row must end in "…": find the first part that does not fit
	// before it. Text is cut inside; a counter is left out whole, marked
	// " · …"; with no room for that mark the part before it is cut instead.
	i, used := 0, 0
	for ; used+ansi.StringWidth(parts[i].text) <= w-1; i++ {
		used += ansi.StringWidth(parts[i].text)
	}
	// Columns for parts[i] cut, "…" included; it may fit whole in w when
	// a counter after it does not.
	room := min(w-used, ansi.StringWidth(parts[i].text)-1)
	for ; i >= 1; i-- {
		if p := parts[i]; p.cut && room > p.keep {
			return ui.FitLine(ui.FitLineEllipsis(render(parts[:i+1]), used+room, fill), w, fill)
		}
		if used+ansi.StringWidth(titleMore) <= w {
			return ui.FitLine(render(append(parts[:i:i], titlePart{text: titleMore, style: titleInfo})), w, fill)
		}
		if i == 2 && strings.HasSuffix(parts[1].text, "…") {
			// The shortened name already ends in "…" (a wide letter may
			// leave no room to cut it further): it ends the row.
			return ui.FitLine(render(parts[:2]), w, fill)
		}
		used -= ansi.StringWidth(parts[i-1].text)
		room = ansi.StringWidth(parts[i-1].text) - 1
	}
	return ui.FitLineEllipsis(render(parts), w, fill)
}

// titleMore ends a title row that left counters out.
const titleMore = " · …"

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	if strings.HasSuffix(word, "ch") {
		return fmt.Sprintf("%d %ses", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// matches counts the nodes shown that the filter matches.
func (m Model) matches() int {
	q, n := strings.ToLower(m.filter), 0
	for _, nd := range m.l.Nodes {
		if matchRank(nd, q) != noMatch && m.selectable(nd.ID) {
			n++
		}
	}
	return n
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
	// The filter's and the toggles' buttons come after those, only where
	// the pane has room for them as well.
	longExtra  = []string{" / filter ", " o orphans ", " u unresolved "}
	shortExtra = []string{" / ", " o ", " u "}
)

// Hint row tails, longest first.
const (
	tailFull   = "  hjkl move · n/N links · enter open · click: preview, again: open · scroll: zoom · drag: move"
	tailDrag   = "  hjkl move · n/N links · enter open · drag: move"
	tailOpen   = "  hjkl move · n/N links · enter open"
	tailKeys   = "  hjkl · n/N · enter"
	tailNarrow = "  hjkl · enter"
)

// hints are the hint row's layouts, widest first: the buttons' labels (in
// buttons() order), then the other keys and what the mouse does. The
// first that fits the pane is used; the last fits MinWidth.
var hints = []struct {
	labels [5]string
	extra  []string
	tail   string
}{
	{longLabels, longExtra, tailFull},
	{longLabels, longExtra, tailDrag},
	{longLabels, longExtra, tailOpen},
	{longLabels, shortExtra, tailDrag},
	{longLabels, shortExtra, tailOpen},
	{longLabels, shortExtra, tailKeys},
	{longLabels, nil, tailKeys},
	{shortLabels, nil, tailKeys},
	{shortLabels, nil, tailNarrow},
	{shortLabels, nil, ""},
}

func (m Model) hint() (labels []string, tail string) {
	for _, h := range hints {
		labels = append(h.labels[:], h.extra...)
		w := 1 + len(labels) - 1 + ansi.StringWidth(h.tail)
		for _, l := range labels {
			w += ansi.StringWidth(l)
		}
		if w <= m.width {
			return labels, h.tail
		}
	}
	last := hints[len(hints)-1]
	return last.labels[:], last.tail
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
	for i, action := range []string{"filter", "orphans", "unresolved"} {
		if 5+i < len(labels) {
			bs = append(bs, button{label: labels[5+i], action: action, enabled: true})
		}
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

// helpLine is the hint row: the buttons, then the other keys; or the
// filter's prompt while it is open.
func (m Model) helpLine(th theme.Theme) string {
	if m.typing {
		return ui.FitLine(lipgloss.NewStyle().Foreground(th.Fg).Render(m.promptLine()), m.width, lipgloss.NewStyle())
	}
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
