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

	// The filter and the toggles (see filter.go): nodes the filter doesn't
	// match are greyed out; hidden ones and their edges are not drawn.
	Filter                      string
	HideOrphans, HideUnresolved bool
}

// Renderer draws one layout. It caches the faint edges of each zoom level
// and the dot cells of each scale, so it is meant for one goroutine (the
// Bubble Tea View).
type Renderer struct {
	l       Layout
	out, in [][]int32  // linked and linking node indexes, per node
	pairs   [][2]int32 // each linked pair once
	faint   [zoomLevels]*edgeLayer
	known   [zoomLevels]*edgeLayer     // faint without the edges to unresolved targets
	dots    map[[2]int]map[Point]int32 // by scale: sx, sy
}

// NewRenderer prepares l and its edges for drawing. Edges to unknown nodes,
// self-links and repeats are ignored.
func NewRenderer(l Layout, edges []Edge) *Renderer {
	r := &Renderer{l: l, out: make([][]int32, len(l.Nodes)), in: make([][]int32, len(l.Nodes)), dots: map[[2]int]map[Point]int32{}}
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

// maxFitX and maxFitY cap the overview's scale at the medium one, so
// zooming in never draws the map smaller.
const maxFitX, maxFitY = 3, 2

// scale is how many screen cells one grid cell spans at zoom z in a w×h
// pane. Medium and close have fixed scales; the overview takes the largest
// whole scale, up to the medium one, that fits the map in the pane, so a
// small map is spread out instead of packed in the middle.
func (r *Renderer) scale(z Zoom, w, h int) (sx, sy int) {
	if z != ZoomOverview {
		return z.fixedScale()
	}
	sx = min(max((w-1)/max(r.l.W-1, 1), 1), maxFitX)
	sy = min(max((h-1)/max(r.l.H-1, 1), 1), maxFitY)
	return sx, sy
}

// Centred is the offset that puts the middle of the map at the middle of a
// w×h pane at zoom z.
func (r *Renderer) Centred(z Zoom, w, h int) Point {
	sx, sy := r.scale(z, w, h)
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
	return r.picture(f, th)
}

// picture draws the map of f with no size check, for a model that checks
// the size of its whole pane instead.
func (r *Renderer) picture(f Frame, th theme.Theme) string {
	d := r.draw(f, th)
	lines := make([]string, f.H)
	for y := range lines {
		lines[y] = d.cv.row(y)
	}
	return ui.Fit(strings.Join(lines, "\n"), f.W, f.H, lipgloss.NewStyle())
}

func (r *Renderer) draw(f Frame, th theme.Theme) *drawing {
	d := r.newDrawing(f, th)
	d.faintEdges()
	d.litEdges()
	d.nodes()
	d.names()
	return d
}

// nodeAt is the node whose dot or name is drawn at pane cell (x, y) of f.
func (r *Renderer) nodeAt(f Frame, x, y int) (int, bool) {
	if x < 0 || y < 0 || x >= f.W || y >= f.H || len(r.l.Nodes) == 0 {
		return 0, false
	}
	own := r.draw(f, f.Theme).cv.cells[y*f.W+x].own
	return int(own) - 1, own > 0
}

// warm builds the caches of the medium and close zoom levels, so the first
// frames after a new map need not; it is meant to run before the renderer
// is shared. The overview's dot cells depend on the pane size, unknown
// here, so its first frame builds them; it draws no faint edges.
func (r *Renderer) warm() {
	for _, z := range []Zoom{ZoomMedium, ZoomClose} {
		sx, sy := z.fixedScale()
		r.dotCells(sx, sy)
		if r.faint[z] == nil {
			r.faint[z] = r.buildLayer(sx, sy, false)
		}
	}
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
	hidden []bool // per node, hidden by a toggle; nil when none is on
	dim    []bool // per node, not matched by the filter; nil with no filter
}

func (r *Renderer) newDrawing(f Frame, th theme.Theme) *drawing {
	d := &drawing{r: r, f: f, th: th, cv: newCanvas(f.W, f.H), sel: -1, open: -1, near: map[int]bool{}}
	d.sx, d.sy = r.scale(f.Zoom, f.W, f.H)
	if f.HideOrphans || f.HideUnresolved {
		d.hidden = make([]bool, len(r.l.Nodes))
		for i, n := range r.l.Nodes {
			d.hidden[i] = hides(n, f.HideOrphans, f.HideUnresolved)
		}
	}
	if f.Filter != "" {
		q := strings.ToLower(f.Filter)
		d.dim = make([]bool, len(r.l.Nodes))
		for i, n := range r.l.Nodes {
			d.dim[i] = matchRank(n, q) == noMatch
		}
	}
	if i, ok := r.l.index[f.Selected]; ok {
		d.sel = i
		for _, j := range r.in[i] {
			if !d.isHidden(int(j)) {
				d.near[int(j)] = false
			}
		}
		for _, j := range r.out[i] {
			if !d.isHidden(int(j)) {
				d.near[int(j)] = true
			}
		}
	}
	if i, ok := r.l.index[f.Open]; ok {
		d.open = i
	}
	d.groups = []lipgloss.Color{th.Blue, th.Purple, th.Teal, th.Orange, th.Cyan, th.Pink, th.Red, th.NordBlue}
	return d
}

func (d *drawing) isHidden(i int) bool { return d.hidden != nil && d.hidden[i] }

// isDim reports whether node i is greyed out by the filter: the selection
// never is.
func (d *drawing) isDim(i int) bool { return d.dim != nil && d.dim[i] && i != d.sel }

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
	layers := &d.r.faint
	if d.f.HideUnresolved {
		layers = &d.r.known
	}
	if layers[z] == nil {
		layers[z] = d.r.buildLayer(d.sx, d.sy, d.f.HideUnresolved)
	}
	layer := layers[z]
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
		if x < 0 || y < 0 || x >= d.f.W || y >= d.f.H || d.isHidden(i) {
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
			switch {
			case d.dim != nil && !d.dim[i] && n.Unresolved:
				st.fg = d.th.Fg // a match stands out of the greyed rest
			case (d.sel < 0 || d.dim != nil) && !n.Unresolved:
				st.fg = d.groupColour(n.Cluster)
			}
		}
		if d.isDim(i) {
			st.fg = d.th.Grey
		}
		d.cv.owner = int32(i) + 1
		d.cv.set(x, y, glyph, st, prio)
	}
	d.cv.owner = 0
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
// neighbours' (busiest first): each in full on the right of its dot or,
// when that side is in the way, in full on the left; failing both, cut
// short on the right before the first dot or name in its way (at the
// overview, only their dots count); then the layout's
// names for the zoom (none at the overview) that cover none of those.
// Every name keeps a blank cell on each side, so edges never run into it.
// Names are placed on the whole map, not the pane, so panning does not
// change which are shown, with one exception: a name goes left of its dot
// only when it fits inside the pane, so near the pane's left edge a
// neighbour may fall back to the right-hand cut.
func (d *drawing) names() {
	n := &namer{d: d, dots: d.r.dotCells(d.sx, d.sy), names: map[Point]bool{}}
	drawn := map[int]bool{}
	if d.sel >= 0 {
		n.draw(d.sel, min(ansi.StringWidth(d.r.l.Nodes[d.sel].Label), maxLabelWidth), sideRight)
		drawn[d.sel] = true
	}
	var hot []int
	if d.open >= 0 && d.open != d.sel && !d.isHidden(d.open) {
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
		gap := n.taken(Point{p.X + 1, p.Y})
		switch {
		case !gap && room >= w:
			n.draw(i, w, sideRight)
		case n.fitsLeft(p, w):
			n.draw(i, w, sideLeft)
		case gap || room < min(w, minNearName):
			continue
		default:
			n.draw(i, room, sideRight)
		}
		drawn[i] = true
	}
	st := style{fg: d.th.GreyFg2}
	if d.sel >= 0 {
		st.fg = d.th.GreyFg
	}
	for _, lb := range d.r.l.Labels[d.f.Zoom] {
		i := d.r.l.index[lb.ID]
		x, y := lb.X-d.f.Offset.X, lb.Y-d.f.Offset.Y
		if drawn[i] || d.class(i) != classRest || d.isHidden(i) || y < 0 || y >= d.f.H || x+lb.W < 0 || x >= d.f.W {
			continue
		}
		st := st
		switch {
		case d.isDim(i):
			st.fg = d.th.Grey
		case d.dim != nil:
			st.fg = d.th.Fg
		}
		if n.anyName(lb.X-1, lb.X+lb.W, lb.Y) {
			continue
		}
		d.cv.owner = int32(i) + 1
		d.cv.set(x-1, y, ' ', style{}, prioRestName)
		d.cv.set(x+lb.W, y, ' ', style{}, prioRestName)
		d.cv.text(x, y, lb.W, ui.Truncate(d.r.l.Nodes[i].Label, lb.W), st, prioRestName)
	}
	d.cv.owner = 0
}

// mapCell is where node i's dot is on the map at the frame's zoom.
func (d *drawing) mapCell(i int) Point {
	p := d.r.l.Nodes[i].P
	return Point{p.X * d.sx, p.Y * d.sy}
}

// dotCells maps each node's cell at scale sx×sy to the node, built once
// per scale (at most a handful: the overview's fits and the fixed ones).
func (r *Renderer) dotCells(sx, sy int) map[Point]int32 {
	key := [2]int{sx, sy}
	if r.dots[key] == nil {
		cells := make(map[Point]int32, len(r.l.Nodes))
		for i, n := range r.l.Nodes {
			cells[Point{n.P.X * sx, n.P.Y * sy}] = int32(i)
		}
		r.dots[key] = cells
	}
	return r.dots[key]
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
	return ok && !n.d.isHidden(int(i)) && (n.d.f.Zoom != ZoomOverview || n.d.class(int(i)) != classRest)
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

// side is where a name goes beside its dot.
type side int

const (
	sideRight side = iota
	sideLeft
)

// fitsLeft reports whether a name w cells wide fits left of the dot at map
// cell p: a blank next to the dot, the name, and a blank before it, none
// of them taken and all inside the pane. A name cut off by the pane's left
// edge would show less than the right-hand cut does.
func (n *namer) fitsLeft(p Point, w int) bool {
	if p.X-2-w-n.d.f.Offset.X < 0 {
		return false
	}
	for x := p.X - 2 - w; x <= p.X-1; x++ {
		if n.taken(Point{x, p.Y}) {
			return false
		}
	}
	return true
}

// draw draws node i's name, cut to w cells, two cells right (or, on the
// left side, left) of its dot, with a blank cell on each side that is not
// taken, and marks those cells. The selection's dot, gap and name share a
// background when the pane has the focus.
func (n *namer) draw(i, w int, at side) {
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
	if d.isDim(i) {
		st.fg = d.th.Grey
	}
	name := ui.Truncate(d.r.l.Nodes[i].Label, w)
	nw := ansi.StringWidth(name)
	p := d.mapCell(i)
	x, y := p.X-d.f.Offset.X, p.Y-d.f.Offset.Y
	d.cv.owner = int32(i) + 1
	defer func() { d.cv.owner = 0 }()
	if at == sideLeft { // fitsLeft checked every cell
		d.cv.set(x-2-nw, y, ' ', style{}, prio)
		d.cv.text(x-1-nw, y, nw, name, st, prio)
		d.cv.set(x-1, y, ' ', style{}, prio)
		for c := p.X - 2 - nw; c <= p.X-1; c++ {
			n.names[Point{c, p.Y}] = true
		}
		return
	}
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
	wide bool  // left half of a wide character
	tail bool  // right half of a wide character
	own  int32 // node index + 1 of the dot or name drawn here; 0 for none
}

type canvas struct {
	cells []cell
	w, h  int
	owner int32 // own of the cells set next
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
	c.cells[i] = cell{r: r, st: st, prio: int8(prio), own: c.owner}
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

// buildLayer rasterises the edges at scale sx×sy; known leaves out those
// to unresolved targets.
func (r *Renderer) buildLayer(sx, sy int, known bool) *edgeLayer {
	l := &edgeLayer{w: max((r.l.W-1)*sx+1, 0), h: max((r.l.H-1)*sy+1, 0)}
	l.mask = make([]uint8, l.w*l.h)
	for _, p := range r.pairs {
		if known && (r.l.Nodes[p[0]].Unresolved || r.l.Nodes[p[1]].Unresolved) {
			continue
		}
		a, b := r.l.Nodes[p[0]].P, r.l.Nodes[p[1]].P
		line(a.X*sx, a.Y*sy, b.X*sx, b.Y*sy, func(x, y int, m uint8) {
			l.mask[y*l.w+x] |= m
		})
	}
	return l
}
