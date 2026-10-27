package globalgraph

// The map is drawn as a solar system, one node per cell: a dot sized by how
// many notes it links, a hollow ring for a link target with no note, each
// in its galaxy's colour. A solar map (one Build or Update laid out) also
// shows its frame: the vault's sun in a halo with the vault's name, each
// galaxy's sun with the folder's name and count above, every galaxy joined
// to the vault's sun by a gently curved line, every note joined to its sun
// by a spoke and dotted rings through the notes, all dimmer than the dots.
// Lines are braille (2×4 dots per cell, so several share a cell) at every
// zoom. The real links run over that frame, brighter and gently curved:
// in the galaxy's colour inside one, a light grey between two; the
// selection's own links are lit over them. The overview spreads the map
// over most of the pane, at a scale that may be a fraction of a cell. Lines always run under names,
// dots and suns, so moving the selection never hides a node, and with a
// selection the rest of the map fades instead of greying.

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
// fewer than hubDegree, and more. On a solar map no note is smaller than
// glyphLinked, and each galaxy's busiest linked note is a glyphHub.
const (
	glyphLeaf       = '·'
	glyphLinked     = '•'
	glyphHub        = '●'
	glyphUnresolved = '○'
	glyphSelected   = '●'
	glyphSun        = '✦' // a folder's sun
	glyphVaultSun   = '◉' // the vault's sun
	leafDegree      = 1
	hubDegree       = 10
	topNotes        = 1 // notes of each galaxy the overview names, past its hub
)

// halo is the ring round the vault's sun: the braille dots of each cell,
// by cell from the sun, of a circle round the middle of its cell.
var halo = func() map[Point]uint8 {
	const rad = 3.5 // braille dots from the middle of the sun's cell (0.5, 1.5)
	cells := map[Point]uint8{}
	n := 44 // points round it, more than its dots
	for j := range n {
		a := 2 * math.Pi * float64(j) / float64(n)
		px, py := int(math.Round(0.5+rad*math.Cos(a))), int(math.Round(1.5+rad*math.Sin(a)))
		cells[Point{px >> 1, py >> 2}] |= brailleBits[py&3][px&1]
	}
	delete(cells, Point{})
	return cells
}()

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

// Renderer draws one layout. It caches the lines of each zoom level and
// the dot cells of each scale, so it is meant for one goroutine (the
// Bubble Tea View).
type Renderer struct {
	l       Layout
	out, in [][]int32  // linked and linking node indexes, per node
	pairs   [][2]int32 // each linked pair once
	galaxy  []int32    // per node, its index in l.Clusters
	hubs    []int32    // per galaxy, its busiest linked note, or -1
	tops    [][]int32  // per galaxy, its topNotes busiest notes, the overview names
	notes   []int      // per galaxy, its notes (not counting missing ones)
	outer   []int      // per galaxy, its outermost ring that holds a node
	layers  map[layerKey]*edgeLayer
	frames  map[layerKey]*frameLayer
	dots    map[scale]map[Point]int32
}

// layerKey picks a cached layer of lines: the scale, and whether the lines
// to unresolved targets and to orphans are left out.
type layerKey struct {
	k             scale
	known, linked bool
}

// NewRenderer prepares l and its edges for drawing. Edges to unknown nodes,
// self-links and repeats are ignored.
func NewRenderer(l Layout, edges []Edge) *Renderer {
	r := &Renderer{l: l, out: make([][]int32, len(l.Nodes)), in: make([][]int32, len(l.Nodes)),
		layers: map[layerKey]*edgeLayer{}, frames: map[layerKey]*frameLayer{}, dots: map[scale]map[Point]int32{}}
	r.galaxy = make([]int32, len(l.Nodes))
	r.hubs = make([]int32, len(l.Clusters))
	for c := range r.hubs {
		r.hubs[c] = -1
	}
	r.tops, r.notes, r.outer = make([][]int32, len(l.Clusters)), make([]int, len(l.Clusters)), make([]int, len(l.Clusters))
	for i, n := range l.Nodes {
		c, _ := slices.BinarySearchFunc(l.Clusters, n.Cluster, func(c Cluster, k string) int { return strings.Compare(c.Key, k) })
		c = min(c, max(len(l.Clusters)-1, 0))
		r.galaxy[i] = int32(c)
		if l.solar {
			r.outer[c] = max(r.outer[c], ringOf(l.Clusters[c].Centre, n.P))
		}
		if !n.Unresolved {
			r.notes[c]++
			r.tops[c] = append(r.tops[c], int32(i))
		}
		if h := r.hubs[c]; n.Degree > 0 && !n.Unresolved && (h < 0 || n.Degree > l.Nodes[h].Degree) {
			r.hubs[c] = int32(i) // nodes are sorted by ID: ties keep the first
		}
	}
	for c, top := range r.tops {
		// Nodes are sorted by ID, so the stable sort breaks ties by ID.
		slices.SortStableFunc(top, func(a, b int32) int { return l.Nodes[b].Degree - l.Nodes[a].Degree })
		r.tops[c] = top[:min(len(top), topNotes)]
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

// scale is how many screen cells one grid cell spans, in 64ths of a cell,
// the same across and down so galaxies stay round. A grid cell lands on
// the screen cell nearest its place; from one cell per grid cell up, two
// grid cells never land on one screen cell.
type scale int

const scaleUnit = 64

// whole is the scale of n screen cells per grid cell.
func whole(n int) scale { return scale(n * scaleUnit) }

// of is grid coordinate v on the screen.
func (k scale) of(v int) int { return roundDiv(v*int(k), scaleUnit) }

// at is grid cell p on the screen.
func (k scale) at(p Point) Point { return Point{k.of(p.X), k.of(p.Y)} }

// maxFit caps the overview's scale at the medium one, so zooming in never
// draws the map smaller.
var maxFit = func() scale { s, _ := ZoomMedium.fixedScale(); return whole(s) }()

// fitShare is how much of the pane the overview fills on the axis the map
// fills first: a margin is left round it.
const fitShare = 0.9

// scale is how many screen cells one grid cell spans at zoom z in a w×h
// pane. Medium and close have fixed whole scales; the overview takes the
// scale, fractions of a cell included, that spreads the map over most of
// the pane (see fitShare), from one cell per grid cell up to the medium
// scale, so a small map is spread out instead of packed in the middle.
func (r *Renderer) scale(z Zoom, w, h int) scale {
	if z != ZoomOverview {
		s, _ := z.fixedScale()
		return whole(s)
	}
	fit := func(pane, grid int) float64 { return fitShare * float64(pane-1) / float64(max(grid-1, 1)) }
	k := scale(min(fit(w, r.l.W), fit(h, r.l.H)) * scaleUnit)
	return min(max(k, whole(1)), maxFit)
}

// Centred is the offset that puts the middle of the map at the middle of a
// w×h pane at zoom z.
func (r *Renderer) Centred(z Zoom, w, h int) Point {
	k := r.scale(z, w, h)
	mw, mh := max(k.of(r.l.W-1)+1, 1), max(k.of(r.l.H-1)+1, 1)
	return Point{floorHalf(mw - w), floorHalf(mh - h)}
}

func floorHalf(n int) int {
	if n < 0 {
		return -((-n + 1) / 2)
	}
	return n / 2
}

// Cell priorities, lowest first: the solar frame (rings, spokes, the
// lines from the galaxies to the vault's sun), the real links, the
// selection's lit links, under the names and dots of the rest of the map,
// the selection's neighbours, the open note and the selection, and the
// suns over all.
const (
	prioNone = iota
	prioRing
	prioSpoke
	prioTrunk
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
	prioSun
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
	d.frame()
	d.faintEdges()
	d.litEdges()
	d.nodes()
	d.suns()
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
		s, _ := z.fixedScale()
		k := whole(s)
		r.dotCells(k)
		r.layer(k, false)
		r.frameAt(layerKey{k: k})
	}
}

// maxCached is how many scales a cache keeps before it starts again: the
// overview's scale follows the pane, so resizing would otherwise keep one
// set of layers per size the pane went through.
const maxCached = 12

// cached is m[key], built on first use.
func cached[K comparable, V any](m map[K]V, key K, build func() V) V {
	v, ok := m[key]
	if !ok {
		if len(m) >= maxCached {
			clear(m)
		}
		v = build()
		m[key] = v
	}
	return v
}

// layer is the cached edge layer at scale k.
func (r *Renderer) layer(k scale, known bool) *edgeLayer {
	return cached(r.layers, layerKey{k: k, known: known}, func() *edgeLayer { return r.buildLayer(k, known) })
}

// frameAt is the cached solar frame for key.
func (r *Renderer) frameAt(key layerKey) *frameLayer {
	return cached(r.frames, key, func() *frameLayer { return r.buildFrame(key) })
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
	k      scale
	cv     *canvas
	sel    int              // selected node index, -1 for none
	open   int              // open note index, -1 for none
	near   map[int]bool     // neighbours of the selection; true when linked from it
	hue    []lipgloss.Color // per galaxy: its colour
	faded  []lipgloss.Color // per galaxy: its colour faded, for the rest of the map
	muted  []lipgloss.Color // per galaxy: its colour a little faded, for the overview's names
	link   []lipgloss.Color // per galaxy: the colour of the links inside it
	spoke  []lipgloss.Color // per galaxy: its spokes
	ring   []lipgloss.Color // per galaxy: its rings
	cross  lipgloss.Color   // links between galaxies
	trunk  lipgloss.Color   // the lines from the galaxies to the vault's sun
	halo   lipgloss.Color   // the ring round the vault's sun
	hidden []bool           // per node, hidden by a toggle; nil when none is on
	dim    []bool           // per node, not matched by the filter; nil with no filter
}

func (r *Renderer) newDrawing(f Frame, th theme.Theme) *drawing {
	d := &drawing{r: r, f: f, th: th, cv: newCanvas(f.W, f.H), sel: -1, open: -1, near: map[int]bool{}}
	d.k = r.scale(f.Zoom, f.W, f.H)
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
	d.hue, d.faded, d.link = make([]lipgloss.Color, n), make([]lipgloss.Color, n), make([]lipgloss.Color, n)
	d.muted, d.spoke, d.ring = make([]lipgloss.Color, n), make([]lipgloss.Color, n), make([]lipgloss.Color, n)
	d.cross, d.trunk, d.halo = th.LightGrey, d.fade(th.Fg, trunkFade, th.Line), d.fade(th.Fg, haloFade, th.GreyFg)
	if d.sel >= 0 {
		d.cross = d.fade(th.LightGrey, restFade, th.GreyFg)
	}
	for c := range n {
		d.hue[c] = palette[c%len(palette)]
		d.faded[c] = d.fade(d.hue[c], restFade, th.GreyFg)
		d.muted[c] = d.fade(d.hue[c], nameFade, th.GreyFg2)
		d.link[c] = d.hue[c]
		if d.sel >= 0 {
			d.link[c] = d.faded[c]
		}
		d.spoke[c] = d.fade(d.hue[c], spokeFade, th.Line)
		d.ring[c] = d.fade(d.hue[c], ringFade, th.Line)
	}
	return d
}

// How far toward the background a colour fades: the dots, hub names and
// links of the rest of the map while a note is selected (and links to
// missing notes); the overview's note names, a little; the solar frame
// always: the vault's sun's halo, the lines to it, the spokes and,
// faintest, the rings.
const (
	restFade  = 0.55
	nameFade  = 0.3
	haloFade  = 0.4
	trunkFade = 0.55
	spokeFade = 0.5
	ringFade  = 0.62
)

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
	return d.k.of(p.X) - d.f.Offset.X, d.k.of(p.Y) - d.f.Offset.Y
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

// frame copies the visible part of the scale's cached solar frame, each
// cell in the colour of the strongest line through it: a ring, a spoke (in
// their galaxy's colours) or a line to the vault's sun. A map placed by
// hand has none.
func (d *drawing) frame() {
	if !d.r.l.solar {
		return
	}
	layer := d.r.frameAt(layerKey{k: d.k, known: d.f.HideUnresolved, linked: d.f.HideOrphans})
	for y := range d.f.H {
		ly := y + d.f.Offset.Y
		if ly < 0 || ly >= layer.h {
			continue
		}
		for x := max(0, -d.f.Offset.X); x < d.f.W && x+d.f.Offset.X < layer.w; x++ {
			i := ly*layer.w + x + d.f.Offset.X
			m := layer.bits[i]
			if m == 0 {
				continue
			}
			st, prio := style{fg: d.trunk}, prioTrunk
			switch g := layer.galaxy[i]; layer.kind[i] {
			case prioSpoke:
				st, prio = style{fg: d.spoke[g]}, prioSpoke
			case prioRing:
				st, prio = style{fg: d.ring[g]}, prioRing
			}
			d.cv.set(x, y, brailleBlank+rune(m), st, prio)
		}
	}
}

// suns draws the vault's sun, with its halo, and each folder's, over
// everything: they are no nodes, so no click or key picks them. The halo
// goes over the frame only.
func (d *drawing) suns() {
	if !d.r.l.solar {
		return
	}
	put := func(p Point, r rune, st style) {
		d.cv.set(d.k.of(p.X)-d.f.Offset.X, d.k.of(p.Y)-d.f.Offset.Y, r, st, prioSun)
	}
	sun := d.k.at(d.r.l.Sun)
	for at, bits := range halo {
		d.cv.set(sun.X+at.X-d.f.Offset.X, sun.Y+at.Y-d.f.Offset.Y, brailleBlank+rune(bits), style{fg: d.halo}, prioTrunk)
	}
	put(d.r.l.Sun, glyphVaultSun, style{fg: d.th.Fg, bold: true})
	for g, c := range d.r.l.Clusters {
		if c.Key != "" {
			put(c.Centre, glyphSun, style{fg: d.hue[g], bold: true})
		}
	}
}

// faintEdges copies the visible part of the scale's cached edge layer, the
// real links, over the frame: a cell inside one galaxy in that galaxy's
// link colour, any other in light grey.
func (d *drawing) faintEdges() {
	layer := d.r.layer(d.k, d.f.HideUnresolved)
	between := style{fg: d.cross}
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
					st = style{fg: d.link[g-1]}
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
		d.r.link(a, b, func(x, y int, bit uint8) {
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
		glyph := d.r.glyph(i)
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

// glyph is node i's dot.
func (r *Renderer) glyph(i int) rune {
	n := r.l.Nodes[i]
	switch {
	case n.Unresolved:
		return glyphUnresolved
	case r.l.solar && r.hubs[r.galaxy[i]] == int32(i):
		return glyphHub
	case n.Degree <= leafDegree && !r.l.solar:
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
	n := newNamer(d)
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
		named := d.hubs()
		if d.r.l.solar {
			for rank := range topNotes {
				for _, top := range d.r.tops {
					if rank < len(top) {
						named = append(named, int(top[rank]))
					}
				}
			}
		}
		for _, i := range named {
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

// hubs lists each galaxy's busiest linked note, busiest first, ties by ID.
func (d *drawing) hubs() []int {
	var hubs []int
	for _, i := range d.r.hubs {
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
// hub's in its galaxy's colour (faded like its dot), and on a solar map
// any galaxy's top note's too, a little faded (see nameFade); any other
// in grey, a little darker while a note is selected; with a filter, grey
// when it does not match and the text colour when it does.
func (d *drawing) restNameStyle(i int) style {
	g := d.r.galaxy[i]
	switch {
	case d.isDim(i):
		return style{fg: d.th.Grey}
	case d.dim != nil:
		return style{fg: d.th.Fg}
	case d.r.l.solar && d.sel < 0 && (d.r.hubs[g] == int32(i) || slices.Contains(d.r.tops[g], int32(i))):
		return style{fg: d.muted[g]}
	case d.r.hubs[g] == int32(i):
		return style{fg: d.restColour(i)}
	case d.sel >= 0:
		return style{fg: d.th.GreyFg}
	}
	return style{fg: d.th.GreyFg2}
}

// headings writes each galaxy's name and number of notes over its sun,
// the name bold in its colour and the number fainter, and the vault's name
// over the vault's sun in the text colour, on the row just over its halo
// when that is free: centred on the row just over the galaxy's outermost
// ring that holds a note, else on the top row of its box, which no ring
// reaches, else on the row above it, else on the row below the box; with
// no room on any of them it is left out. Each
// galaxy's box keeps room for its heading, so the overview, which fits the
// map in the pane, shows every heading whole; it is cut with … only when
// the pane is narrower.
func (n *namer) headings() {
	d := n.d
	if l := d.r.l; l.solar && l.Name != "" {
		top, bottom := l.Sun.Y-1, l.Sun.Y
		if g, ok := slices.BinarySearchFunc(l.Clusters, "", func(c Cluster, k string) int { return strings.Compare(c.Key, k) }); ok {
			top, bottom = l.Clusters[g].Rect.Y, l.Clusters[g].Rect.Y+l.Clusters[g].Rect.H-1
		}
		rows := []int{d.k.of(l.Sun.Y) - 2, d.k.of(top), d.k.of(top) - 1, d.k.of(bottom) + 1}
		n.heading(l.Name, "", l.Sun.X, rows, style{fg: d.th.Fg, bold: true}, style{})
	}
	for g, c := range d.r.l.Clusters {
		if c.Key != "" {
			top, bottom := c.Rect.Y, c.Rect.Y+c.Rect.H-1
			hug := d.k.of(c.Centre.Y-(ringRadius(d.r.outer[g])+1)/cellAspect) - 1 // over the ring's top row
			n.heading(c.Key, fmt.Sprintf(" · %d", d.r.notes[g]), c.Centre.X, []int{max(hug, d.k.of(top)), d.k.of(top), d.k.of(top) - 1, d.k.of(bottom) + 1},
				style{fg: d.hue[g], bold: true}, style{fg: d.muted[g]})
		}
	}
}

// heading writes name, then tail, centred on grid column cx, on the first
// free one of the map rows (see headings), with a blank cell on each side,
// and marks those cells. When both do not fit in the pane, the tail is
// left out first.
func (n *namer) heading(name, tail string, cx int, rows []int, st, tailSt style) {
	d := n.d
	full, tw := ansi.StringWidth(name), ansi.StringWidth(tail)
	if full+tw > d.f.W-2 {
		tail, tw = "", 0
	}
	w := min(full, d.f.W-2)
	if w < min(full, minNearName) {
		return
	}
	all := w + tw
	x0 := d.k.of(cx) - all/2
	for _, y := range rows {
		free := true
		for x := x0 - 1; x <= x0+all && free; x++ {
			free = !n.taken(Point{x, y})
		}
		if !free {
			continue
		}
		for x := x0 - 1; x <= x0+all; x++ {
			n.names[Point{x, y}] = true
		}
		sx, sy := x0-d.f.Offset.X, y-d.f.Offset.Y
		d.cv.set(sx-1, sy, ' ', style{}, prioRestName)
		d.cv.set(sx+all, sy, ' ', style{}, prioRestName)
		d.cv.text(sx, sy, w, ui.Truncate(name, w), st, prioRestName)
		if tw > 0 {
			d.cv.text(sx+w, sy, tw, tail, tailSt, prioRestName)
		}
		return
	}
}

// mapCell is where node i's dot is on the map at the frame's zoom.
func (d *drawing) mapCell(i int) Point {
	p := d.r.l.Nodes[i].P
	return d.k.at(p)
}

// dotCells maps each node's cell at scale k to the node.
func (r *Renderer) dotCells(k scale) map[Point]int32 {
	return cached(r.dots, k, func() map[Point]int32 {
		cells := make(map[Point]int32, len(r.l.Nodes))
		for i, n := range r.l.Nodes {
			cells[k.at(n.P)] = int32(i)
		}
		return cells
	})
}

// namer places the neighbourhood's names on the map.
type namer struct {
	d     *drawing
	dots  map[Point]int32
	suns  map[Point]bool
	names map[Point]bool // cells of the names placed so far, with their blanks
}

func newNamer(d *drawing) *namer {
	n := &namer{d: d, dots: d.r.dotCells(d.k), suns: map[Point]bool{}, names: map[Point]bool{}}
	for _, p := range d.r.l.suns() {
		n.suns[d.k.at(p)] = true
	}
	if d.r.l.solar {
		sun := d.k.at(d.r.l.Sun)
		for at := range halo {
			n.suns[Point{sun.X + at.X, sun.Y + at.Y}] = true
		}
	}
	return n
}

// taken reports whether a name may not use map cell p: another name, its
// blanks, a sun or a shown dot are there.
func (n *namer) taken(p Point) bool {
	if n.names[p] || n.suns[p] {
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

// firstDot is the first map column in [x0, x1] of row y holding a sun or
// a dot that is not hidden, or -1: no name covers either.
func (n *namer) firstDot(x0, x1, y int) int {
	for x := x0; x <= x1; x++ {
		if i, ok := n.dots[Point{x, y}]; ok && !n.d.isHidden(int(i)) || n.suns[Point{x, y}] {
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
	n := newNamer(d)
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

// link plots the line of a real link between screen cells a and b: on a
// solar map a curve bent by linkBend, on a map placed by hand straight.
func (r *Renderer) link(a, b Point, plot func(x, y int, bit uint8)) {
	if r.l.solar {
		brailleCurve(a.X, a.Y, b.X, b.Y, linkBend, plot)
	} else {
		brailleLine(a.X, a.Y, b.X, b.Y, plot)
	}
}

// brailleCurve is brailleLine bent into a quadratic Bézier curve: its
// control point sits bend × the line's length off the line's middle, to
// the left going from (x0, y0) to (x1, y1). Braille dots are as far apart
// across as down, so the bend is the same whichever way the line runs.
func brailleCurve(x0, y0, x1, y1 int, bend float64, plot func(x, y int, bit uint8)) {
	ax, ay, bx, by := 2*x0, 4*y0+1, 2*x1, 4*y1+1
	fax, fay, fbx, fby := float64(ax), float64(ay), float64(bx), float64(by)
	cx, cy := (fax+fbx)/2+bend*(fby-fay), (fay+fby)/2-bend*(fbx-fax)
	// The curve moves at most 2×max(|C-A|, |B-C|) along an axis over the
	// whole of t, so that many steps never skip a dot.
	reach := max(math.Abs(cx-fax), math.Abs(fbx-cx), math.Abs(cy-fay), math.Abs(fby-cy))
	n := int(2*reach) + 1
	// Forward differences: B(t) = A + 2(C−A)t + (A−2C+B)t², stepped by 1/n.
	h := 1 / float64(n)
	x, y := fax, fay
	dx, dy := 2*(cx-fax)*h+(fax-2*cx+fbx)*h*h, 2*(cy-fay)*h+(fay-2*cy+fby)*h*h
	ddx, ddy := 2*(fax-2*cx+fbx)*h*h, 2*(fay-2*cy+fby)*h*h
	px, py := ax, ay
	for i := 1; i < n; i++ {
		x, y = x+dx, y+dy
		dx, dy = dx+ddx, dy+ddy
		qx, qy := int(math.Floor(x+0.5)), int(math.Floor(y+0.5))
		if qx == px && qy == py {
			continue
		}
		px, py = qx, qy
		x, y := qx>>1, qy>>2
		if (x == x0 && y == y0) || (x == x1 && y == y1) {
			continue
		}
		plot(x, y, brailleBits[qy&3][qx&1])
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
func (r *Renderer) buildLayer(k scale, known bool) *edgeLayer {
	l := &edgeLayer{w: max(k.of(r.l.W-1)+1, 0), h: max(k.of(r.l.H-1)+1, 0)}
	l.bits, l.galaxy = make([]uint8, l.w*l.h), make([]int32, l.w*l.h)
	for _, p := range r.pairs {
		if known && (r.l.Nodes[p[0]].Unresolved || r.l.Nodes[p[1]].Unresolved) {
			continue
		}
		g := int32(-1)
		if r.galaxy[p[0]] == r.galaxy[p[1]] {
			g = r.galaxy[p[0]] + 1
		}
		a, b := k.at(r.l.Nodes[p[0]].P), k.at(r.l.Nodes[p[1]].P)
		r.link(a, b, func(x, y int, bit uint8) {
			if x < 0 || y < 0 || x >= l.w || y >= l.h {
				return
			}
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

// frameLayer is the solar frame rasterised at one scale: the braille dots
// of each cell, the strongest kind of line through it (prioRing, prioSpoke
// or prioTrunk) and the galaxy of that line.
type frameLayer struct {
	bits   []uint8
	kind   []int8
	galaxy []int32
	w, h   int
}

// The rings and spokes are dotted, so they never read as links: a dot
// every ringDots braille dots round a ring, every lineDots along a spoke.
// The lines to the vault's sun are dotted too, more closely.
const ringDots, lineDots, trunkDots = 3, 2, 2

// How far lines bend (see brailleCurve): the lines to the vault's sun all
// the same way round, like the arms of a galaxy; the real links a little.
const trunkBend, linkBend = 0.16, 0.1

// buildFrame rasterises the frame at scale key.k: a line from each folder's
// galaxy to the vault's (from outside the rings of one to outside the
// rings of the other, so the suns stay clear), a spoke from each shown
// node to its galaxy's sun, and a dotted ring through the nodes of each
// ring that holds one. Nodes key leaves out (unresolved targets, orphans)
// get no spoke and no ring.
func (r *Renderer) buildFrame(key layerKey) *frameLayer {
	k := key.k
	f := &frameLayer{w: max(k.of(r.l.W-1)+1, 0), h: max(k.of(r.l.H-1)+1, 0)}
	f.bits, f.kind, f.galaxy = make([]uint8, f.w*f.h), make([]int8, f.w*f.h), make([]int32, f.w*f.h)
	plot := func(kind int8, g int32, every int) func(x, y int, bit uint8) {
		step := 0
		return func(x, y int, bit uint8) {
			if step++; (step-1)%every != 0 || x < 0 || y < 0 || x >= f.w || y >= f.h {
				return
			}
			i := y*f.w + x
			f.bits[i] |= bit
			if kind >= f.kind[i] {
				f.kind[i], f.galaxy[i] = kind, g
			}
		}
	}
	sun := r.l.Sun
	centre := func(g int32) Point {
		if r.l.Clusters[g].Key == "" {
			return sun
		}
		return r.l.Clusters[g].Centre
	}
	rings := make([]map[int]bool, len(r.l.Clusters))
	outer := make([]int, len(r.l.Clusters)) // outermost ring drawn, per galaxy
	for i, n := range r.l.Nodes {
		if key.known && n.Unresolved || key.linked && n.Degree == 0 {
			continue
		}
		g := r.galaxy[i]
		c := centre(g)
		a, b := k.at(c), k.at(n.P)
		brailleLine(a.X, a.Y, b.X, b.Y, plot(prioSpoke, g, lineDots))
		if rings[g] == nil {
			rings[g] = map[int]bool{}
		}
		ring := ringOf(c, n.P)
		rings[g][ring] = true
		outer[g] = max(outer[g], ring)
	}
	rootOuter := 0
	for g, c := range r.l.Clusters {
		if c.Key == "" {
			rootOuter = outer[g]
		}
	}
	for g, c := range r.l.Clusters {
		for ring := range rings[g] {
			p := centre(int32(g))
			at := k.at(p)
			dottedRing(at.X, at.Y, k.of(ringRadius(ring)), plot(prioRing, int32(g), 1))
		}
		if c.Key == "" {
			continue
		}
		// Clear of both galaxies' rings, a cell past the outer one.
		inner, far := k.of(ringRadius(rootOuter)+1), k.of(ringRadius(outer[g])+1)
		trunk := plot(prioTrunk, int32(g), trunkDots)
		from, to := k.at(sun), k.at(c.Centre)
		brailleCurve(from.X, from.Y, to.X, to.Y, trunkBend, func(x, y int, bit uint8) {
			if screenDist2(Point{x, y}, from) > inner*inner && screenDist2(Point{x, y}, to) > far*far {
				trunk(x, y, bit)
			}
		})
	}
	return f
}

// ringOf is the ring around sun c that cell p is on.
func ringOf(c, p Point) int {
	d := math.Sqrt(float64(screenDist2(p, c)))
	return max(1, int(math.Round((d+ringStep/2)/ringStep)))
}

// dottedRing plots a dot every ringDots braille dots round the circle of
// r screen columns about the dot of cell (cx, cy): r columns across and
// r/2 rows down are both 2r braille dots, so it is a circle in dots.
func dottedRing(cx, cy, r int, plot func(x, y int, bit uint8)) {
	if r <= 0 {
		return
	}
	ax, ay, rad := 2*cx, 4*cy+1, float64(2*r)
	n := max(8, int(2*math.Pi*rad/ringDots))
	for j := range n {
		a := 2 * math.Pi * float64(j) / float64(n)
		px, py := ax+int(math.Round(rad*math.Cos(a))), ay+int(math.Round(rad*math.Sin(a)))
		plot(px>>1, py>>2, brailleBits[py&3][px&1])
	}
}
