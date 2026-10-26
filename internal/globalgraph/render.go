package globalgraph

// The map is drawn with plain glyphs, one node per cell: a dot sized by how
// many notes it links, a hollow ring for a link target with no note, and
// edges as straight runs of ─ │ ╲ ╱ (the same box-drawing family as the
// local graph). At medium and close zoom every edge is drawn faintly; the
// selection's own edges are lit at every zoom. Edges always run under dots
// and names, so moving the selection never hides a node.

import (
	"slices"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/theme"
	"github.com/pwn27sbx/misoku/internal/ui"
)

// MinWidth and MinHeight are the smallest pane the map is drawn in; a
// smaller one shows a message instead.
const MinWidth, MinHeight = 30, 8

// Node glyphs. A dot grows with the node's degree: up to leafDegree links,
// fewer than hubDegree, and more.
const (
	glyphLeaf       = '·'
	glyphLinked     = '•'
	glyphHub        = '●'
	glyphUnresolved = '○'
	glyphSelected   = '●'
	leafDegree      = 1
	hubDegree       = 10
)

// Frame is what one picture of the map depends on besides the layout.
type Frame struct {
	W, H     int         // pane size in cells
	Offset   Point       // map cell, at Zoom, drawn at the pane's top-left
	Zoom     Zoom        // level of detail
	Selected string      // selected node ID; "" for none
	Open     string      // the note open when the graph was opened
	Focused  bool        // the pane has the focus: the selection gets a background
	Dimmed   bool        // draw every text colour grey, like other unfocused panes
	Theme    theme.Theme // colours
}

// Renderer draws one layout. It caches the faint edges and the dot cells of
// each zoom level, so it is meant for one goroutine (the Bubble Tea View).
type Renderer struct {
	l       Layout
	out, in [][]int32  // linked and linking node indexes, per node
	pairs   [][2]int32 // each linked pair once
	faint   [zoomLevels]*edgeLayer
	dots    [zoomLevels]map[Point]int32
}

// NewRenderer prepares l and its edges for drawing. Edges to unknown nodes,
// self-links and repeats are ignored.
func NewRenderer(l Layout, edges []Edge) *Renderer {
	r := &Renderer{l: l, out: make([][]int32, len(l.Nodes)), in: make([][]int32, len(l.Nodes))}
	seen, paired := map[[2]int32]bool{}, map[[2]int32]bool{}
	for _, e := range edges {
		a, okA := l.index[e.From]
		b, okB := l.index[e.To]
		if !okA || !okB || a == b {
			continue
		}
		dir := [2]int32{int32(a), int32(b)}
		if seen[dir] {
			continue
		}
		seen[dir] = true
		r.out[a] = append(r.out[a], int32(b))
		r.in[b] = append(r.in[b], int32(a))
		if pair := [2]int32{int32(min(a, b)), int32(max(a, b))}; !paired[pair] {
			paired[pair] = true
			r.pairs = append(r.pairs, pair)
		}
	}
	return r
}

// Centred is the offset that puts the middle of the map at the middle of a
// w×h pane at zoom z.
func (r *Renderer) Centred(z Zoom, w, h int) Point {
	sx, sy := z.Scale()
	mw, mh := max((r.l.W-1)*sx+1, 1), max((r.l.H-1)*sy+1, 1)
	return Point{floorHalf(mw - w), floorHalf(mh - h)}
}

func floorHalf(n int) int {
	if n < 0 {
		return -((-n + 1) / 2)
	}
	return n / 2
}

// Cell priorities, lowest first: faint edges, then the selection's lit
// edges, under the names and dots of the rest of the map, the selection's
// neighbours, the open note and the selection.
const (
	prioNone = iota
	prioFaint
	prioLit
	prioRestName
	prioRestDot
	prioNearName
	prioNearDot
	prioOpenName
	prioOpenDot
	prioSelName
	prioSelDot
)

// class ranks how a node is highlighted.
type class int

const (
	classRest class = iota
	classNear
	classOpen
	classSel
)

// Render draws f: exactly f.H lines of exactly f.W cells, or "" for an
// empty pane.
func (r *Renderer) Render(f Frame) string {
	if f.W <= 0 || f.H <= 0 {
		return ""
	}
	th := f.Theme
	if f.Dimmed {
		th = th.Greyed()
	}
	switch {
	case len(r.l.Nodes) == 0:
		return message("No notes yet", f.W, f.H, th)
	case f.W < MinWidth || f.H < MinHeight:
		return message("Graph needs a bigger pane", f.W, f.H, th)
	}
	d := r.newDrawing(f, th)
	d.faintEdges()
	d.litEdges()
	d.nodes()
	d.names()
	lines := make([]string, f.H)
	for y := range lines {
		lines[y] = d.cv.row(y)
	}
	return ui.Fit(strings.Join(lines, "\n"), f.W, f.H, lipgloss.NewStyle())
}

// message is text centred in a w×h pane.
func message(text string, w, h int, th theme.Theme) string {
	msg := lipgloss.NewStyle().Foreground(th.GreyFg).Render(ui.Truncate(text, w))
	return ui.Fit(lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, msg), w, h, lipgloss.NewStyle())
}

// drawing is one frame being drawn.
type drawing struct {
	r      *Renderer
	f      Frame
	th     theme.Theme
	sx, sy int
	cv     *canvas
	sel    int          // selected node index, -1 for none
	open   int          // open note index, -1 for none
	near   map[int]bool // neighbours of the selection; true when linked from it
	groups []lipgloss.Color
}

func (r *Renderer) newDrawing(f Frame, th theme.Theme) *drawing {
	d := &drawing{r: r, f: f, th: th, cv: newCanvas(f.W, f.H), sel: -1, open: -1, near: map[int]bool{}}
	d.sx, d.sy = f.Zoom.Scale()
	if i, ok := r.l.index[f.Selected]; ok {
		d.sel = i
		for _, j := range r.in[i] {
			d.near[int(j)] = false
		}
		for _, j := range r.out[i] {
			d.near[int(j)] = true
		}
	}
	if i, ok := r.l.index[f.Open]; ok {
		d.open = i
	}
	d.groups = []lipgloss.Color{th.Blue, th.Purple, th.Teal, th.Orange, th.Cyan, th.Pink, th.Red, th.NordBlue}
	return d
}

// screen is where node i's dot is in the pane.
func (d *drawing) screen(i int) (x, y int) {
	p := d.r.l.Nodes[i].P
	return p.X*d.sx - d.f.Offset.X, p.Y*d.sy - d.f.Offset.Y
}

func (d *drawing) class(i int) class {
	switch {
	case i == d.sel:
		return classSel
	case i == d.open:
		return classOpen
	}
	if _, ok := d.near[i]; ok {
		return classNear
	}
	return classRest
}

// faintEdges copies the visible part of the zoom's cached edge layer.
func (d *drawing) faintEdges() {
	z := d.f.Zoom
	if z == ZoomOverview {
		return
	}
	if d.r.faint[z] == nil {
		d.r.faint[z] = d.r.buildLayer(d.sx, d.sy)
	}
	layer := d.r.faint[z]
	st := style{fg: d.th.Line}
	for y := range d.f.H {
		ly := y + d.f.Offset.Y
		if ly < 0 || ly >= layer.h {
			continue
		}
		for x := max(0, -d.f.Offset.X); x < d.f.W && x+d.f.Offset.X < layer.w; x++ {
			if m := layer.mask[ly*layer.w+x+d.f.Offset.X]; m != 0 {
				d.cv.set(x, y, maskGlyph(m), st, prioFaint)
			}
		}
	}
}

// litEdges draws the selection's links (Blue) and backlinks (Purple).
func (d *drawing) litEdges() {
	if d.sel < 0 {
		return
	}
	x0, y0 := d.screen(d.sel)
	for _, j := range sortedIndexes(d.near) {
		fg := d.th.Purple
		if d.near[j] {
			fg = d.th.Blue
		}
		x1, y1 := d.screen(j)
		st := style{fg: fg}
		line(x0, y0, x1, y1, func(x, y int, m uint8) {
			d.cv.set(x, y, maskGlyph(m), st, prioLit)
		})
	}
}

// sortedIndexes lists m's node indexes in order, so drawing never depends
// on map order.
func sortedIndexes(m map[int]bool) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// nodes draws every visible dot.
func (d *drawing) nodes() {
	for i, n := range d.r.l.Nodes {
		x, y := d.screen(i)
		if x < 0 || y < 0 || x >= d.f.W || y >= d.f.H {
			continue
		}
		c := d.class(i)
		glyph := dotGlyph(n)
		st := style{fg: d.th.GreyFg}
		prio := prioRestDot
		switch c {
		case classSel:
			st.fg, prio = d.th.Yellow, prioSelDot
			if !n.Unresolved {
				glyph = glyphSelected
			}
			if d.f.Focused {
				st.bg = d.th.OneBg
			}
		case classOpen:
			st.fg, prio = d.th.Green, prioOpenDot
		case classNear:
			st.fg, prio = d.th.Purple, prioNearDot
			if d.near[i] {
				st.fg = d.th.Blue
			}
		default:
			if d.sel < 0 && !n.Unresolved {
				st.fg = d.groupColour(n.Cluster)
			}
		}
		d.cv.set(x, y, glyph, st, prio)
	}
}

func dotGlyph(n Placed) rune {
	switch {
	case n.Unresolved:
		return glyphUnresolved
	case n.Degree <= leafDegree:
		return glyphLeaf
	case n.Degree < hubDegree:
		return glyphLinked
	}
	return glyphHub
}

// groupColour is a project's colour while nothing is selected: the theme's
// accents in turn, by the clusters' sorted order.
func (d *drawing) groupColour(key string) lipgloss.Color {
	i, _ := slices.BinarySearchFunc(d.r.l.Clusters, key, func(c Cluster, k string) int { return strings.Compare(c.Key, k) })
	return d.groups[i%len(d.groups)]
}

// minNearName is the fewest cells a neighbour's or the open note's name
// is cut to; with less room it is left out.
const minNearName = 3

// names draws the selection's name in full, then the open note's and the
// neighbours' (busiest first), each cut short before the first dot or name
// in its way (at the overview, only their dots count); then the layout's
// names for the zoom (none at the overview) that cover none of those.
// Every name keeps a blank cell on each side, so edges never run into it.
// Names are placed on the whole map, not the pane, so panning never
// changes which are shown.
func (d *drawing) names() {
	n := &namer{d: d, dots: d.r.dotCells(d.f.Zoom), names: map[Point]bool{}}
	drawn := map[int]bool{}
	if d.sel >= 0 {
		n.draw(d.sel, min(ansi.StringWidth(d.r.l.Nodes[d.sel].Label), maxLabelWidth))
		drawn[d.sel] = true
	}
	var hot []int
	if d.open >= 0 && d.open != d.sel {
		hot = append(hot, d.open)
	}
	near := sortedIndexes(d.near)
	slices.SortStableFunc(near, func(a, b int) int { return d.r.l.Nodes[b].Degree - d.r.l.Nodes[a].Degree })
	for _, i := range near {
		if i != d.open {
			hot = append(hot, i)
		}
	}
	for _, i := range hot {
		p := d.mapCell(i)
		w := min(ansi.StringWidth(d.r.l.Nodes[i].Label), maxLabelWidth)
		room := 0
		for room < w && !n.taken(Point{p.X + 2 + room, p.Y}) {
			room++
		}
		if room < w || n.taken(Point{p.X + 2 + w, p.Y}) {
			room-- // keep a blank before what is in the way
		}
		if n.taken(Point{p.X + 1, p.Y}) || room < min(w, minNearName) {
			continue
		}
		n.draw(i, room)
		drawn[i] = true
	}
	st := style{fg: d.th.GreyFg2}
	if d.sel >= 0 {
		st.fg = d.th.GreyFg
	}
	for _, lb := range d.r.l.Labels[d.f.Zoom] {
		i := d.r.l.index[lb.ID]
		x, y := lb.X-d.f.Offset.X, lb.Y-d.f.Offset.Y
		if drawn[i] || d.class(i) != classRest || y < 0 || y >= d.f.H || x+lb.W < 0 || x >= d.f.W {
			continue
		}
		if n.anyName(lb.X-1, lb.X+lb.W, lb.Y) {
			continue
		}
		d.cv.set(x-1, y, ' ', style{}, prioRestName)
		d.cv.set(x+lb.W, y, ' ', style{}, prioRestName)
		d.cv.text(x, y, lb.W, ui.Truncate(d.r.l.Nodes[i].Label, lb.W), st, prioRestName)
	}
}

// mapCell is where node i's dot is on the map at the frame's zoom.
func (d *drawing) mapCell(i int) Point {
	p := d.r.l.Nodes[i].P
	return Point{p.X * d.sx, p.Y * d.sy}
}

// dotCells maps each node's cell at zoom z to the node, built once per zoom.
func (r *Renderer) dotCells(z Zoom) map[Point]int32 {
	if r.dots[z] == nil {
		sx, sy := z.Scale()
		r.dots[z] = make(map[Point]int32, len(r.l.Nodes))
		for i, n := range r.l.Nodes {
			r.dots[z][Point{n.P.X * sx, n.P.Y * sy}] = int32(i)
		}
	}
	return r.dots[z]
}

// namer places the neighbourhood's names on the map.
type namer struct {
	d     *drawing
	dots  map[Point]int32
	names map[Point]bool // cells of the names placed so far, with their blanks
}

// taken reports whether a name may not use map cell p.
func (n *namer) taken(p Point) bool {
	if n.names[p] {
		return true
	}
	i, ok := n.dots[p]
	return ok && (n.d.f.Zoom != ZoomOverview || n.d.class(int(i)) != classRest)
}

// anyName reports whether a cell [x0, x1] of map row y holds a name.
func (n *namer) anyName(x0, x1, y int) bool {
	for x := x0; x <= x1; x++ {
		if n.names[Point{x, y}] {
			return true
		}
	}
	return false
}

// draw draws node i's name, cut to w cells, two cells right of its dot,
// with a blank cell on each side that is not taken, and marks those cells.
// The selection's dot, gap and name share a background when the pane has
// the focus.
func (n *namer) draw(i, w int) {
	d := n.d
	st, prio := style{fg: d.th.Fg}, prioNearName
	switch d.class(i) {
	case classSel:
		st.bold, prio = true, prioSelName
		if d.f.Focused {
			st.bg = d.th.OneBg
		}
	case classOpen:
		st.bold, prio = true, prioOpenName
	}
	name := ui.Truncate(d.r.l.Nodes[i].Label, w)
	nw := ansi.StringWidth(name)
	p := d.mapCell(i)
	x, y := p.X-d.f.Offset.X, p.Y-d.f.Offset.Y
	if !n.taken(Point{p.X + 1, p.Y}) {
		d.cv.set(x+1, y, ' ', style{bg: st.bg}, prio)
	}
	if !n.taken(Point{p.X + 2 + nw, p.Y}) {
		d.cv.set(x+2+nw, y, ' ', style{}, prio)
	}
	d.cv.text(x+2, y, nw, name, st, prio)
	for c := p.X + 1; c <= p.X+2+nw; c++ {
		n.names[Point{c, p.Y}] = true
	}
}

// style is how a cell is coloured.
type style struct {
	fg, bg lipgloss.Color
	bold   bool
}

// cell is one character of the pane.
type cell struct {
	r    rune
	st   style
	prio int8
	wide bool // left half of a wide character
	tail bool // right half of a wide character
}

type canvas struct {
	cells []cell
	w, h  int
}

func newCanvas(w, h int) *canvas { return &canvas{cells: make([]cell, w*h), w: w, h: h} }

// set draws r at (x, y) unless a cell of higher priority is there. A wide
// character it covers half of is blanked.
func (c *canvas) set(x, y int, r rune, st style, prio int) {
	if x < 0 || y < 0 || x >= c.w || y >= c.h {
		return
	}
	i := y*c.w + x
	if int(c.cells[i].prio) > prio {
		return
	}
	c.blankPartner(x, y)
	c.cells[i] = cell{r: r, st: st, prio: int8(prio)}
}

// blankPartner turns the other half of a wide character at (x, y) into a
// blank cell of the same style.
func (c *canvas) blankPartner(x, y int) {
	i := y*c.w + x
	switch {
	case c.cells[i].wide && x+1 < c.w:
		c.cells[i+1] = cell{r: ' ', st: c.cells[i+1].st, prio: c.cells[i+1].prio}
	case c.cells[i].tail && x > 0:
		c.cells[i-1] = cell{r: ' ', st: c.cells[i-1].st, prio: c.cells[i-1].prio}
	}
}

// text writes s from x, at most width cells. A wide character goes in
// whole or, cut by the pane's edge or a cell of higher priority, not at all;
// the part of it inside the pane is left blank.
func (c *canvas) text(x, y, width int, s string, st style, prio int) {
	end := x + width
	for _, r := range s {
		w := ansi.StringWidth(string(r))
		if w == 0 || x+w > end {
			break
		}
		if w == 1 {
			c.set(x, y, r, st, prio)
			x++
			continue
		}
		whole := x >= 0 && x+1 < c.w
		for k := range w {
			if p := x + k; p >= 0 && p < c.w && y >= 0 && y < c.h && int(c.cells[y*c.w+p].prio) > prio {
				whole = false
			}
		}
		if whole && y >= 0 && y < c.h {
			c.set(x, y, r, st, prio)
			c.set(x+1, y, 0, st, prio)
			c.cells[y*c.w+x].wide, c.cells[y*c.w+x+1].tail = true, true
		} else {
			for k := range w {
				c.set(x+k, y, ' ', st, prio)
			}
		}
		x += w
	}
}

// row renders row y, styling runs of cells that look the same.
func (c *canvas) row(y int) string {
	var b, run strings.Builder
	var cur style
	flush := func() {
		if run.Len() == 0 {
			return
		}
		if cur == (style{}) {
			b.WriteString(run.String())
		} else {
			st := lipgloss.NewStyle().Bold(cur.bold)
			if cur.fg != "" {
				st = st.Foreground(cur.fg)
			}
			if cur.bg != "" {
				st = st.Background(cur.bg)
			}
			b.WriteString(st.Render(run.String()))
		}
		run.Reset()
	}
	for _, cl := range c.cells[y*c.w : (y+1)*c.w] {
		if cl.tail {
			continue
		}
		r := cl.r
		if r == 0 {
			r = ' '
		}
		st := cl.st
		if r == ' ' {
			st.fg, st.bold = "", false // a blank shows only its background
		}
		if st != cur {
			flush()
			cur = st
		}
		run.WriteRune(r)
	}
	flush()
	return b.String()
}

// Edge directions crossing a cell.
const (
	maskH uint8 = 1 << iota
	maskV
	maskDown // ╲
	maskUp   // ╱
)

func maskGlyph(m uint8) rune {
	switch {
	case m&maskH != 0 && m&maskV != 0:
		return '┼'
	case m&maskDown != 0 && m&maskUp != 0:
		return '╳'
	case m&maskH != 0:
		return '─'
	case m&maskV != 0:
		return '│'
	case m&maskDown != 0:
		return '╲'
	}
	return '╱'
}

// line calls plot for each cell strictly between (x0, y0) and (x1, y1) on
// a straight line, with the direction of the step that reached it.
func line(x0, y0, x1, y1 int, plot func(x, y int, m uint8)) {
	dx, dy := x1-x0, y1-y0
	n := max(abs(dx), abs(dy))
	px, py := x0, y0
	for i := 1; i < n; i++ {
		x, y := x0+roundDiv(dx*i, n), y0+roundDiv(dy*i, n)
		var m uint8
		switch {
		case y == py:
			m = maskH
		case x == px:
			m = maskV
		case (x-px)*(y-py) > 0:
			m = maskDown
		default:
			m = maskUp
		}
		plot(x, y, m)
		px, py = x, y
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// roundDiv is a/b rounded half away from zero, for b > 0.
func roundDiv(a, b int) int {
	if a < 0 {
		return -((-a*2 + b) / (2 * b))
	}
	return (a*2 + b) / (2 * b)
}

// edgeLayer is every edge of the map rasterised at one zoom level.
type edgeLayer struct {
	mask []uint8
	w, h int
}

func (r *Renderer) buildLayer(sx, sy int) *edgeLayer {
	l := &edgeLayer{w: max((r.l.W-1)*sx+1, 0), h: max((r.l.H-1)*sy+1, 0)}
	l.mask = make([]uint8, l.w*l.h)
	for _, p := range r.pairs {
		a, b := r.l.Nodes[p[0]].P, r.l.Nodes[p[1]].P
		line(a.X*sx, a.Y*sy, b.X*sx, b.Y*sy, func(x, y int, m uint8) {
			l.mask[y*l.w+x] |= m
		})
	}
	return l
}
