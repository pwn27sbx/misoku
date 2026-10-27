package globalgraph

// The map is drawn as galaxies, one node per cell: a dot sized by how many
// notes it links, a hollow ring for a link target with no note, each in its
// galaxy's colour, and the galaxy's folder name above it. Edges are thin
// braille lines (2×4 dots per cell, so several edges share a cell) at every
// zoom: inside a galaxy in its colour faded toward the background, between
// galaxies in grey; the selection's own edges are lit over them. Edges
// always run under dots and names, so moving the selection never hides a
// node, and with a selection the rest of the map fades instead of greying.

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strconv"
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
	galaxy  []int32    // per node, its index in l.Clusters
	centres []int32    // per galaxy, the node at its centre, or -1
	layers  map[layerKey]*edgeLayer
	dots    map[[2]int]map[Point]int32 // by scale: sx, sy
}

// layerKey picks a cached edge layer: the scale, and whether the edges to
// unresolved targets are left out.
type layerKey struct {
	k     int
	known bool
}

// NewRenderer prepares l and its edges for drawing. Edges to unknown nodes,
// self-links and repeats are ignored.
func NewRenderer(l Layout, edges []Edge) *Renderer {
	r := &Renderer{l: l, out: make([][]int32, len(l.Nodes)), in: make([][]int32, len(l.Nodes)), layers: map[layerKey]*edgeLayer{}, dots: map[[2]int]map[Point]int32{}}
	r.galaxy = make([]int32, len(l.Nodes))
	r.centres = make([]int32, len(l.Clusters))
	for c := range r.centres {
		r.centres[c] = -1
	}
	for i, n := range l.Nodes {
		c, _ := slices.BinarySearchFunc(l.Clusters, n.Cluster, func(c Cluster, k string) int { return strings.Compare(c.Key, k) })
		r.galaxy[i] = int32(min(c, max(len(l.Clusters)-1, 0)))
		if c < len(l.Clusters) && l.Clusters[c].Key == n.Cluster && l.Clusters[c].Centre == n.P {
			r.centres[c] = int32(i)
		}
	}
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

// maxFit caps the overview's scale at the medium one, so zooming in never
// draws the map smaller.
var maxFit, _ = ZoomMedium.fixedScale()

// scale is how many screen cells one grid cell spans at zoom z in a w×h
// pane, the same across and down (sx == sy) so galaxies stay round. Medium
// and close have fixed scales; the overview takes the largest whole scale,
// up to the medium one, that fits the map in the pane, so a small map is
// spread out instead of packed in the middle.
func (r *Renderer) scale(z Zoom, w, h int) (sx, sy int) {
	if z != ZoomOverview {
		return z.fixedScale()
	}
	k := min((w-1)/max(r.l.W-1, 1), (h-1)/max(r.l.H-1, 1))
	k = min(max(k, 1), maxFit)
	return k, k
}

// centreOf is the node at the centre of galaxy c.
func (r *Renderer) centreOf(c Cluster) (int, bool) {
	i, ok := slices.BinarySearchFunc(r.l.Clusters, c.Key, func(c Cluster, k string) int { return strings.Compare(c.Key, k) })
	if !ok || r.centres[i] < 0 {
		return 0, false
	}
	return int(r.centres[i]), true
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
// is shared. The overview's scale depends on the pane size, unknown here,
// so its first frame builds what it needs unless it shares the medium
// scale.
func (r *Renderer) warm() {
	for _, z := range []Zoom{ZoomMedium, ZoomClose} {
		k, _ := z.fixedScale()
		r.dotCells(k, k)
		r.layer(k, false)
	}
}

// layer is the cached edge layer at scale k, built on first use.
func (r *Renderer) layer(k int, known bool) *edgeLayer {
	key := layerKey{k, known}
	if r.layers[key] == nil {
		r.layers[key] = r.buildLayer(k, known)
	}
	return r.layers[key]
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
	sel    int              // selected node index, -1 for none
	open   int              // open note index, -1 for none
	near   map[int]bool     // neighbours of the selection; true when linked from it
	hue    []lipgloss.Color // per galaxy: its colour
	faded  []lipgloss.Color // per galaxy: its colour faded, for the rest of the map
	edge   []lipgloss.Color // per galaxy: the colour of the edges inside it
	hidden []bool           // per node, hidden by a toggle; nil when none is on
	dim    []bool           // per node, not matched by the filter; nil with no filter
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
	palette := []lipgloss.Color{th.Blue, th.Purple, th.Teal, th.Orange, th.Cyan, th.Pink, th.Red, th.NordBlue}
	n := len(r.l.Clusters)
	d.hue, d.faded, d.edge = make([]lipgloss.Color, n), make([]lipgloss.Color, n), make([]lipgloss.Color, n)
	for c := range n {
		d.hue[c] = palette[c%len(palette)]
		d.faded[c] = d.fade(d.hue[c], restFade, th.GreyFg)
		d.edge[c] = d.fade(d.hue[c], edgeFade, th.Line)
	}
	return d
}

// How far toward the background a colour fades: the dots and hub names of
// the rest of the map while a note is selected (and links to missing
// notes), and the edges inside a galaxy.
const restFade, edgeFade = 0.55, 0.62

// fade is c faded toward the background by t, or c itself when the pane
// is greyed out (every colour there is already the same grey).
func (d *drawing) fade(c lipgloss.Color, t float64, fallback lipgloss.Color) lipgloss.Color {
	if d.f.Dimmed {
		return c
	}
	return fade(c, d.th.Bg, t, fallback)
}

// fade blends colour a toward b by t (0 is a, 1 is b). Both must be
// "#rrggbb"; otherwise (a terminal palette index, say) it is fallback.
func fade(a, b lipgloss.Color, t float64, fallback lipgloss.Color) lipgloss.Color {
	ra, okA := hexRGB(string(a))
	rb, okB := hexRGB(string(b))
	if !okA || !okB {
		return fallback
	}
	var out [3]int
	for k := range out {
		out[k] = int(math.Round(float64(ra[k]) + (float64(rb[k])-float64(ra[k]))*t))
	}
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", out[0], out[1], out[2]))
}

func hexRGB(s string) ([3]int, bool) {
	var c [3]int
	if len(s) != 7 || s[0] != '#' {
		return c, false
	}
	for k := range c {
		v, err := strconv.ParseUint(s[1+2*k:3+2*k], 16, 8)
		if err != nil {
			return c, false
		}
		c[k] = int(v)
	}
	return c, true
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

// faintEdges copies the visible part of the scale's cached edge layer:
// a cell inside one galaxy in that galaxy's edge colour, any other in grey.
func (d *drawing) faintEdges() {
	layer := d.r.layer(d.sx, d.f.HideUnresolved)
	between := style{fg: d.th.GreyFg}
	for y := range d.f.H {
		ly := y + d.f.Offset.Y
		if ly < 0 || ly >= layer.h {
			continue
		}
		for x := max(0, -d.f.Offset.X); x < d.f.W && x+d.f.Offset.X < layer.w; x++ {
			i := ly*layer.w + x + d.f.Offset.X
			if m := layer.bits[i]; m != 0 {
				st := between
				if g := layer.galaxy[i]; g > 0 {
					st = style{fg: d.edge[g-1]}
				}
				d.cv.set(x, y, brailleBlank+rune(m), st, prioFaint)
			}
		}
	}
}

// litEdges draws the selection's links (Blue) and backlinks (Purple) over
// the faint ones, on the same dots: a lit edge's cell shows only lit dots.
func (d *drawing) litEdges() {
	if d.sel < 0 {
		return
	}
	for _, j := range sortedIndexes(d.near) {
		fg := d.th.Purple
		if d.near[j] {
			fg = d.th.Blue
		}
		st := style{fg: fg}
		a, b := d.mapCell(min(d.sel, j)), d.mapCell(max(d.sel, j)) // the faint layer's direction
		brailleLine(a.X, a.Y, b.X, b.Y, func(x, y int, bit uint8) {
			x, y = x-d.f.Offset.X, y-d.f.Offset.Y
			if x < 0 || y < 0 || x >= d.f.W || y >= d.f.H {
				return
			}
			r := brailleBlank + rune(bit)
			if c := d.cv.cells[y*d.f.W+x]; c.prio == prioLit {
				r |= c.r
			}
			d.cv.set(x, y, r, st, prioLit)
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
			st.fg = d.restColour(i)
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

// restColour is the colour of node i outside the highlighted few: its
// galaxy's colour (the theme's accents in turn, by the galaxies' sorted
// order), faded while another note is selected and for a link to a
// missing note. A filter instead greys what it does not match (see isDim)
// and shows a matching missing note in the text colour.
func (d *drawing) restColour(i int) lipgloss.Color {
	g, unresolved := d.r.galaxy[i], d.r.l.Nodes[i].Unresolved
	switch {
	case d.dim != nil && unresolved:
		return d.th.Fg
	case d.dim != nil:
		return d.hue[g]
	case d.sel >= 0 || unresolved:
		return d.faded[g]
	}
	return d.hue[g]
}

// minNearName is the fewest cells a neighbour's or the open note's name
// is cut to; with less room it is left out.
const minNearName = 3

// names draws the selection's name (see selName), then the open note's
// and the neighbours' (busiest first), then each galaxy's heading, then
// the hubs' names: at the overview each galaxy's centre note, closer in
// the layout's names for the zoom. A highlighted note's or a hub's name
// goes in full on the right of its dot or, when that side is in the way,
// in full on the left; failing both, cut short on the right before the
// first dot or name in its way (see place). A name of the layout's table
// that covers any of those is left out. No name covers another note's dot
// (the selection's only when nothing else fits, see selName), and every
// name keeps a blank cell on each side, so edges never run into it.
// Names are placed on the whole map, not the pane, so panning does not
// change which are shown, with one exception: a name goes left of its dot
// only when it fits inside the pane, so near the pane's left edge a
// neighbour or the selection may fall back to the right-hand cut.
func (d *drawing) names() {
	n := &namer{d: d, dots: d.r.dotCells(d.sx, d.sy), names: map[Point]bool{}}
	drawn := map[int]bool{}
	if d.sel >= 0 {
		w, at := n.selName(d.mapCell(d.sel), min(ansi.StringWidth(d.r.l.Nodes[d.sel].Label), maxLabelWidth))
		n.draw(d.sel, w, at)
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
		drawn[i] = n.place(i)
	}
	n.headings()
	if d.f.Zoom == ZoomOverview {
		for _, i := range d.hubs() {
			if !drawn[i] && d.class(i) == classRest && !d.isHidden(i) {
				drawn[i] = n.place(i)
			}
		}
	}
	for _, lb := range d.r.l.Labels[d.f.Zoom] {
		i := d.r.l.index[lb.ID]
		x, y := lb.X-d.f.Offset.X, lb.Y-d.f.Offset.Y
		if drawn[i] || d.class(i) != classRest || d.isHidden(i) || y < 0 || y >= d.f.H || x+lb.W < 0 || x >= d.f.W {
			continue
		}
		if n.anyName(lb.X-1, lb.X+lb.W, lb.Y) {
			continue
		}
		d.cv.owner = int32(i) + 1
		d.cv.set(x-1, y, ' ', style{}, prioRestName)
		d.cv.set(x+lb.W, y, ' ', style{}, prioRestName)
		d.cv.text(x, y, lb.W, ui.Truncate(d.r.l.Nodes[i].Label, lb.W), d.restNameStyle(i), prioRestName)
	}
	d.cv.owner = 0
}

// hubs lists the galaxies' centre notes, busiest first, ties by ID.
func (d *drawing) hubs() []int {
	var hubs []int
	for _, i := range d.r.centres {
		if i >= 0 {
			hubs = append(hubs, int(i))
		}
	}
	slices.SortFunc(hubs, func(a, b int) int {
		return cmp.Or(d.r.l.Nodes[b].Degree-d.r.l.Nodes[a].Degree, strings.Compare(d.r.l.Nodes[a].ID, d.r.l.Nodes[b].ID))
	})
	return hubs
}

// restNameStyle is the colour of a name outside the highlighted few: a
// hub's in its galaxy's colour (faded like its dot), any other in grey,
// a little darker while a note is selected; with a filter, grey when it
// does not match and the text colour when it does.
func (d *drawing) restNameStyle(i int) style {
	switch {
	case d.isDim(i):
		return style{fg: d.th.Grey}
	case d.dim != nil:
		return style{fg: d.th.Fg}
	case d.r.centres[d.r.galaxy[i]] == int32(i):
		return style{fg: d.restColour(i)}
	case d.sel >= 0:
		return style{fg: d.th.GreyFg}
	}
	return style{fg: d.th.GreyFg2}
}

// headings names each galaxy (the notes at the vault root have no name),
// bold in its colour, centred over it on the top row of its box, which no
// ring reaches, or the row above when a name or dot is there; with no room
// on either, it is left out. A heading wider than its galaxy's box is cut,
// or left out when that leaves fewer than minNearName cells.
func (n *namer) headings() {
	d := n.d
	for g, c := range d.r.l.Clusters {
		full := ansi.StringWidth(c.Key)
		w := min(full, maxLabelWidth, c.Rect.W*d.sx-2)
		if c.Key == "" || w < min(full, minNearName) {
			continue
		}
		x0 := c.Centre.X*d.sx - w/2
		for _, y := range []int{c.Rect.Y * d.sy, c.Rect.Y*d.sy - 1} {
			free := true
			for x := x0 - 1; x <= x0+w && free; x++ {
				free = !n.taken(Point{x, y})
			}
			if !free {
				continue
			}
			for x := x0 - 1; x <= x0+w; x++ {
				n.names[Point{x, y}] = true
			}
			sx, sy := x0-d.f.Offset.X, y-d.f.Offset.Y
			d.cv.set(sx-1, sy, ' ', style{}, prioRestName)
			d.cv.set(sx+w, sy, ' ', style{}, prioRestName)
			d.cv.text(sx, sy, w, ui.Truncate(c.Key, w), style{fg: d.hue[g], bold: true}, prioRestName)
			break
		}
	}
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

// taken reports whether a name may not use map cell p: another name, its
// blanks or a shown dot are there.
func (n *namer) taken(p Point) bool {
	if n.names[p] {
		return true
	}
	i, ok := n.dots[p]
	return ok && !n.d.isHidden(int(i))
}

// place draws node i's name in full on the right of its dot or, when that
// side is in the way, in full on the left (see fitsLeft); failing both,
// cut short on the right one blank before what is in its way, unless that
// leaves fewer than minNearName cells or the gap next to the dot is taken:
// then it is left out. It reports whether the name was drawn.
func (n *namer) place(i int) bool {
	p := n.d.mapCell(i)
	w := min(ansi.StringWidth(n.d.r.l.Nodes[i].Label), maxLabelWidth)
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
		return false
	default:
		n.draw(i, room, sideRight)
	}
	return true
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

// selName is how wide the selection's name, w cells in full, is drawn
// beside its dot at map cell p, and on which side, so that it hides no
// other visible note's dot (at the overview too): in full on the right
// when that hides none; else in full on the left when it fits there (see
// fitsLeft) and hides none; else on the right, cut with … one blank
// before the first dot it would hide, or in full when that leaves fewer
// than minNearName cells. It is the first name placed: only dots are in
// its way.
func (n *namer) selName(p Point, w int) (int, side) {
	x0, x1 := p.X+2, p.X+1+w // the name; draw blanks a free cell each side
	if !n.taken(Point{p.X + 1, p.Y}) {
		x0--
	}
	if !n.taken(Point{p.X + 2 + w, p.Y}) {
		x1++
	}
	dot := n.firstDot(x0, x1, p.Y)
	switch {
	case dot < 0:
		return w, sideRight
	case n.fitsLeft(p, w) && n.firstDot(p.X-2-w, p.X-1, p.Y) < 0:
		return w, sideLeft
	case dot-p.X-3 >= minNearName:
		return dot - p.X - 3, sideRight
	}
	return w, sideRight
}

// firstDot is the first map column in [x0, x1] of row y holding a dot
// that is not hidden, or -1.
func (n *namer) firstDot(x0, x1, y int) int {
	for x := x0; x <= x1; x++ {
		if i, ok := n.dots[Point{x, y}]; ok && !n.d.isHidden(int(i)) {
			return x
		}
	}
	return -1
}

// selSpan is the map columns from the selection's dot to the far end of
// its name in frame f, as names places them when the pane leaves room on
// the dot's left (see selName); (0, 0) when f selects no note.
func (r *Renderer) selSpan(f Frame) (x0, x1 int) {
	d := r.newDrawing(f, f.Theme)
	if d.sel < 0 {
		return 0, 0
	}
	p := d.mapCell(d.sel)
	w := min(ansi.StringWidth(r.l.Nodes[d.sel].Label), maxLabelWidth)
	d.f.Offset.X = p.X - 2 - w
	n := &namer{d: d, dots: r.dotCells(d.sx, d.sy), names: map[Point]bool{}}
	nw, at := n.selName(p, w)
	if at == sideLeft {
		return p.X - 2 - nw, p.X
	}
	return p.X, p.X + 1 + nw
}

// draw draws node i's name, cut to w cells, two cells right (or, on the
// left side, left) of its dot, with a blank cell on each side that is not
// taken, and marks those cells. The selection's dot, gap and name share a
// background when the pane has the focus.
func (n *namer) draw(i, w int, at side) {
	d := n.d
	st, prio := style{fg: d.th.Fg}, prioNearName
	switch d.class(i) {
	case classRest:
		st, prio = d.restNameStyle(i), prioRestName
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

// brailleBlank is the empty braille pattern; a pattern's dots are bits
// added to it, brailleBits[row][column] in a cell of 2 columns by 4 rows.
const brailleBlank = '\u2800'

var brailleBits = [4][2]uint8{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

// brailleLine calls plot with the cell and braille dot of each step of a
// straight line between the dots of cells (x0, y0) and (x1, y1), at braille
// resolution, outside those two cells. Every line leaves a dot from its
// cell's middle dot row (the second of four) and left column, so lines
// from one dot meet there.
func brailleLine(x0, y0, x1, y1 int, plot func(x, y int, bit uint8)) {
	ax, ay, bx, by := 2*x0, 4*y0+1, 2*x1, 4*y1+1
	dx, dy := bx-ax, by-ay
	n := max(abs(dx), abs(dy))
	for i := 1; i < n; i++ {
		px, py := ax+roundDiv(dx*i, n), ay+roundDiv(dy*i, n)
		x, y := px>>1, py>>2 // floor, also below zero
		if (x == x0 && y == y0) || (x == x1 && y == y1) {
			continue
		}
		plot(x, y, brailleBits[py&3][px&1])
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

// edgeLayer is every edge of the map rasterised at one scale: the braille
// dots of each cell, and the galaxy (index + 1) whose inner edges alone
// cross it, or -1 for a cell an edge between galaxies, or edges of
// several, cross.
type edgeLayer struct {
	bits   []uint8
	galaxy []int32
	w, h   int
}

// buildLayer rasterises the edges at scale k; known leaves out those to
// unresolved targets.
func (r *Renderer) buildLayer(k int, known bool) *edgeLayer {
	l := &edgeLayer{w: max((r.l.W-1)*k+1, 0), h: max((r.l.H-1)*k+1, 0)}
	l.bits, l.galaxy = make([]uint8, l.w*l.h), make([]int32, l.w*l.h)
	for _, p := range r.pairs {
		if known && (r.l.Nodes[p[0]].Unresolved || r.l.Nodes[p[1]].Unresolved) {
			continue
		}
		g := int32(-1)
		if r.galaxy[p[0]] == r.galaxy[p[1]] {
			g = r.galaxy[p[0]] + 1
		}
		a, b := r.l.Nodes[p[0]].P, r.l.Nodes[p[1]].P
		brailleLine(a.X*k, a.Y*k, b.X*k, b.Y*k, func(x, y int, bit uint8) {
			i := y*l.w + x
			l.bits[i] |= bit
			if l.galaxy[i] == 0 {
				l.galaxy[i] = g
			} else if l.galaxy[i] != g {
				l.galaxy[i] = -1
			}
		})
	}
	return l
}
