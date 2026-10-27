// Package globalgraph lays out and draws the map of every note.
package globalgraph

// The map is a solar system of folders. The vault's sun sits in the middle
// with the notes at the vault root on rings around it, past a first ring
// left to its halo. Each top-level folder (a project) is a galaxy: a sun
// of its own, the folder's notes on rings around it (busier notes on
// inner rings, ring k holding at most ringShare×k, so a galaxy's size
// follows its notes), and its name and count above. The galaxies sit on
// one or more big elliptical orbits around the vault's sun, big and small
// ones taking turns round it and the most linked side by side. Rings are
// twice as wide as tall, so they look round in terminal cells; orbits are
// flatter, the shape of a pane. Every galaxy keeps an empty band around
// its box. One node takes one cell; suns are not nodes. Everything is sorted before it
// is placed, so the same notes and links always give the same map,
// whatever order they arrive in.

import (
	"cmp"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
)

// Node is one note (or unresolved link target) of the global map.
type Node struct {
	ID         string // note ID, a slash-separated path such as "proj/Note.md"
	Cluster    string // top-level folder; "" for notes at the vault root
	Unresolved bool   // a link target with no note behind it
}

// Edge links two nodes by ID; direction does not matter for the layout.
type Edge struct{ From, To string }

// Graph is the input of the layout.
type Graph struct {
	Nodes []Node
	Edges []Edge
	Name  string // the vault's name, written over its sun
}

// Point is a cell of the virtual grid.
type Point struct{ X, Y int }

// Rect is an area of the virtual grid.
type Rect struct{ X, Y, W, H int }

// Placed is a node with its place on the grid.
type Placed struct {
	Node
	Degree int    // distinct linked nodes, self-links and unknown IDs ignored
	Label  string // name to draw: the shortest path suffix no other node shares
	P      Point
}

// Cluster is the galaxy of one top-level folder: its sun at Centre (for
// the notes at the root, the vault's sun), the ellipse of RX columns by RY
// rows around it that its rings may use (RX = 2×RY, round on screen), and
// Rect, the box around that ellipse.
type Cluster struct {
	Key    string
	Rect   Rect
	Centre Point
	RX, RY int
}

// Zoom is a level of detail of the map.
type Zoom int

// Zoom levels, from the whole map to readable names.
const (
	ZoomOverview Zoom = iota
	ZoomMedium
	ZoomClose
	zoomLevels
)

// fixedScale is how many screen cells one grid cell spans at z, with the
// overview at 1×1. The grid already counts columns and rows the way the
// screen does (a galaxy is twice as wide as tall), so both axes take the
// same scale and galaxies stay round at every zoom. The overview drawn on
// screen is scaled to fit the pane (see Renderer.scale); this fixed one
// only places the layout's labels, which the overview has none of.
func (z Zoom) fixedScale() (sx, sy int) {
	switch z {
	case ZoomMedium:
		return 2, 2
	case ZoomClose:
		return 4, 4
	}
	return 1, 1
}

// Label is a name drawn on screen at one zoom level: W cells from (X, Y),
// two cells right of its node's dot.
type Label struct {
	ID      string
	X, Y, W int
}

// Layout is the placed map.
type Layout struct {
	Nodes    []Placed  // sorted by ID
	Clusters []Cluster // sorted by Key
	W, H     int       // grid size
	Sun      Point     // the vault's sun, in the middle of the orbits
	Name     string    // the vault's name
	Labels   [zoomLevels][]Label
	index    map[string]int
	solar    bool  // laid out by Build or Update, so it has suns, rings and orbits; a map placed by hand has none
	moved    Point // how far Update moved the whole map to keep it inside the grid
}

const (
	maxLabelWidth    = 24  // longer names are cut by the renderer
	mediumLabelShare = 5   // medium zoom names the busiest 1/5 of the nodes
	cellAspect       = 2   // a terminal cell is about twice as tall as wide
	ringStep         = 4   // screen columns between two rings of a galaxy; the first is half as far out
	ringMargin       = 2   // columns a galaxy reserves past its outer ring for rounding
	dotSpacing       = 2.5 // screen columns between two slots of a ring
	ringShare        = 4   // ring k holds at most ringShare×k notes, so a galaxy grows with its notes,
	sparseRings      = 3   // on its first sparseRings rings
	galaxyGap        = 3   // least empty columns between two galaxies' boxes
	galaxyRowGap     = 1   // least empty rows between them
	orbitFlat        = 3.2 // columns per row across an orbit: flatter than a ring, like a pane
	orbitRound       = 2.0 // the orbit's superellipse power: 2 is an ellipse, more is squarer
	firstOrbit       = 14  // galaxies on the first orbit; each next one holds orbitGrowth more
	orbitGrowth      = 8
	newAngles        = 48 // directions Update tries for a new galaxy
	orbitSearch      = 24 // columns past the first orbit that fits that Update still tries
	linkPull         = 1  // weight of the distance to linked galaxies against the orbit's size
	maxHeading       = 40 // a galaxy keeps room for this much of its name
	headCount        = 6  // and for its count of notes: " · 999"
	rootFirstRing    = 2  // the vault's notes start on ring 2, clear of its sun's halo
	goldenTurn       = 0.6180339887498949
	noRect           = -1000 // marks a node with no place yet
)

// Build lays out g from scratch.
func Build(g Graph) Layout {
	nodes, pairs := normalize(g)
	if len(nodes) == 0 {
		return finish(nil, nil) // an empty vault has no map, not even a sun
	}
	groups := groupByCluster(nodes)
	root := &galaxy{root: true, size: len(groups[""]), head: ansi.StringWidth(g.Name), placed: true}
	root.rings = ringsFor(root.size, rootFirstRing)
	var gs []*galaxy // the folders
	for _, key := range sortedKeys(groups) {
		if key != "" {
			gs = append(gs, &galaxy{key: key, size: len(groups[key]), rings: ringsFor(len(groups[key]), 1), head: ansi.StringWidth(key) + headCount})
		}
	}
	placeOrbits(root, gs, clusterLinks(nodes, pairs, gs))
	all := append([]*galaxy{root}, gs...)
	b := bounds(all)
	shift(all, Point{-b.X, -b.Y})
	used := map[Point]bool{}
	for _, gx := range all {
		fill(nodes, groups[gx.key], gx, used)
	}
	return assemble(nodes, all, g.Name)
}

// Update lays out g keeping every node of prev where it is in its galaxy:
// new nodes take the free ring slots nearest their galaxy's sun, a full
// galaxy grows a ring and pushes the galaxies in its way outward (their
// notes move with them, the rest stay), and a new folder gets a galaxy on
// the orbits, beside those it links to. When the whole map would leave the
// grid's top-left corner, every node moves by the same step (see moved).
// It returns a fresh Build and false only when prev was not laid out by
// Build or Update or when pushing goes round in circles.
func Update(prev Layout, g Graph) (Layout, bool) {
	if !prev.solar {
		return Build(g), false
	}
	nodes, pairs := normalize(g)
	groups := groupByCluster(nodes)
	root := &galaxy{root: true, size: len(groups[""]), head: ansi.StringWidth(g.Name), c: prev.Sun, placed: true}
	all := []*galaxy{root}
	byKey := map[string]*galaxy{"": root}
	for _, key := range sortedKeys(groups) {
		if key != "" {
			gx := &galaxy{key: key, size: len(groups[key]), head: ansi.StringWidth(key) + headCount}
			all, byKey[key] = append(all, gx), gx
		}
	}
	for _, c := range prev.Clusters {
		if gx := byKey[c.Key]; gx != nil {
			gx.c, gx.rings, gx.placed = c.Centre, max(0, (c.RX-ringMargin+ringStep/2)/ringStep), true
		}
	}
	// Kept nodes hold their offset from their sun while galaxies move.
	taken := map[*galaxy]map[Point]bool{}
	pending := map[string][]int{}
	for i := range nodes {
		n, gx := &nodes[i], byKey[nodes[i].Cluster]
		if old, ok := prev.Lookup(n.ID); ok && old.Cluster == n.Cluster && gx.placed {
			if taken[gx] == nil {
				taken[gx] = map[Point]bool{}
			}
			n.P = Point{old.P.X - gx.c.X, old.P.Y - gx.c.Y}
			taken[gx][n.P] = true
		} else {
			pending[n.Cluster] = append(pending[n.Cluster], i)
		}
	}
	var links map[[2]int]int
	folders := all[1:]
	for _, key := range sortedKeys(pending) {
		gx := byKey[key]
		if !gx.placed {
			if links == nil {
				links = clusterLinks(nodes, pairs, folders)
			}
			gx.rings = ringsFor(gx.size, 1)
			placeNew(gx, all, root.c, links)
		}
		for freeSlots(gx, taken[gx]) < len(pending[key]) {
			gx.rings++
			if !makeRoom(gx, all, root.c) {
				return Build(g), false
			}
		}
	}
	b := bounds(all)
	moved := Point{max(0, -b.X), max(0, -b.Y)}
	shift(all, moved)
	used := map[Point]bool{}
	for i := range nodes {
		if n := &nodes[i]; n.P.X != noRect {
			gx := byKey[n.Cluster]
			n.P = Point{n.P.X + gx.c.X, n.P.Y + gx.c.Y}
			used[n.P] = true
		}
	}
	for _, key := range sortedKeys(pending) {
		fill(nodes, pending[key], byKey[key], used)
	}
	l := assemble(nodes, all, g.Name)
	l.moved = moved
	return l, true
}

// Lookup returns the placed node with id.
func (l Layout) Lookup(id string) (Placed, bool) {
	i, ok := l.index[id]
	if !ok {
		return Placed{}, false
	}
	return l.Nodes[i], true
}

// normalize sorts nodes by ID, drops repeated IDs and counts each node's
// distinct neighbours, which it also returns as index pairs. Positions are
// left at noRect.
func normalize(g Graph) ([]Placed, [][2]int) {
	nodes := make([]Placed, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		nodes = append(nodes, Placed{Node: n, P: Point{noRect, noRect}})
	}
	slices.SortFunc(nodes, func(a, b Placed) int {
		return cmp.Or(cmp.Compare(a.ID, b.ID), cmp.Compare(a.Cluster, b.Cluster), cmpBool(a.Unresolved, b.Unresolved))
	})
	nodes = slices.CompactFunc(nodes, func(a, b Placed) bool { return a.ID == b.ID })
	index := make(map[string]int, len(nodes))
	for i, n := range nodes {
		index[n.ID] = i
	}
	seen := map[[2]int]bool{}
	var pairs [][2]int
	for _, e := range g.Edges {
		a, okA := index[e.From]
		b, okB := index[e.To]
		if !okA || !okB || a == b {
			continue
		}
		pair := [2]int{min(a, b), max(a, b)}
		if !seen[pair] {
			seen[pair] = true
			pairs = append(pairs, pair)
			nodes[a].Degree++
			nodes[b].Degree++
		}
	}
	return nodes, pairs
}

func cmpBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	}
	return -1
}

// groupByCluster lists node indexes per cluster key.
func groupByCluster(nodes []Placed) map[string][]int {
	groups := map[string][]int{}
	for i, n := range nodes {
		groups[n.Cluster] = append(groups[n.Cluster], i)
	}
	return groups
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// galaxy is a cluster while it is laid out: a sun at c with rings of slots
// around it.
type galaxy struct {
	key    string
	size   int   // members
	rings  int   // rings reserved around the sun
	head   int   // width of its name: the folder's, or the vault's for the root
	c      Point // the sun's cell
	placed bool
	root   bool // the vault's sun and the notes at the root
}

// firstRing is the innermost ring that holds notes: round the vault's sun
// the first ring is left to its halo.
func (gx *galaxy) firstRing() int {
	if gx.root {
		return rootFirstRing
	}
	return 1
}

// rx and ry are the radii of the ellipse the galaxy reserves, in columns
// and rows: its outer ring plus a margin for rounding slots to cells.
func (gx *galaxy) rx() int { return ringRadius(gx.rings) + ringMargin }
func (gx *galaxy) ry() int { return gx.rx() / cellAspect }

// box is what a galaxy keeps to itself: its rings, and its name centred
// on the row above them (with a blank each side), the top row of its
// ellipse; the ellipse's bottom row is left to the gap.
func (gx *galaxy) box() Rect {
	hw := gx.rx()
	if gx.head > 0 {
		hw = max(hw, min(gx.head, maxHeading)/2+1)
	}
	return Rect{gx.c.X - hw, gx.c.Y - gx.ry(), 2*hw + 1, 2 * gx.ry()}
}

// apart reports whether two boxes keep the gap between galaxies.
func apart(a, b Rect) bool {
	return a.X+a.W+galaxyGap <= b.X || b.X+b.W+galaxyGap <= a.X || a.Y+a.H+galaxyRowGap <= b.Y || b.Y+b.H+galaxyRowGap <= a.Y
}

// fitsAmong reports whether gx keeps the gap to every other placed galaxy.
func fitsAmong(gx *galaxy, all []*galaxy) bool {
	b := gx.box()
	for _, o := range all {
		if o != gx && o.placed && !apart(b, o.box()) {
			return false
		}
	}
	return true
}

// bounds is the box around every placed galaxy's box.
func bounds(all []*galaxy) Rect {
	x0, y0, x1, y1 := math.MaxInt, math.MaxInt, math.MinInt, math.MinInt
	for _, gx := range all {
		b := gx.box()
		x0, y0 = min(x0, b.X), min(y0, b.Y)
		x1, y1 = max(x1, b.X+b.W), max(y1, b.Y+b.H)
	}
	return Rect{x0, y0, x1 - x0, y1 - y0}
}

func shift(all []*galaxy, d Point) {
	for _, gx := range all {
		gx.c.X += d.X
		gx.c.Y += d.Y
	}
}

// assemble builds the layout of a solar map: the galaxies with notes as
// clusters (the root's only when it has notes), the grid wide enough for
// every galaxy's box and name.
func assemble(nodes []Placed, all []*galaxy, name string) Layout {
	cs := make([]Cluster, 0, len(all))
	for _, gx := range all {
		if gx.size == 0 {
			continue
		}
		rx, ry := gx.rx(), gx.ry()
		cs = append(cs, Cluster{Key: gx.key, Rect: Rect{gx.c.X - rx, gx.c.Y - ry, 2*rx + 1, 2*ry + 1}, Centre: gx.c, RX: rx, RY: ry})
	}
	sun := all[0].c
	l := build(nodes, cs, &sun)
	l.Name = name
	for _, gx := range all {
		b := gx.box()
		l.W, l.H = max(l.W, b.X+b.W), max(l.H, b.Y+b.H)
	}
	return l
}

// ringRadius is how many screen columns ring k is from the sun: 2, 6,
// 10… so the first ring hugs it; the sun itself is ring 0.
func ringRadius(k int) int {
	if k == 0 {
		return 0
	}
	return k*ringStep - ringStep/2
}

// ringSlots lists the cells of ring k around (0, 0), by angle: an ellipse
// of ringRadius(k) columns by half as many rows, a slot every dotSpacing
// screen columns, odd rings turned half a slot so their dots fall between
// those of even ones.
func ringSlots(k int) []Point {
	if k == 0 {
		return []Point{{0, 0}}
	}
	r := float64(ringRadius(k))
	n := max(6, int(2*math.Pi*r/dotSpacing))
	turn := 0.0
	if k%2 == 1 {
		turn = math.Pi / float64(n)
	}
	slots := make([]Point, 0, n)
	for j := range n {
		a := turn + 2*math.Pi*float64(j)/float64(n)
		p := Point{cellRound(r * math.Cos(a)), cellRound(r * math.Sin(a) / cellAspect)}
		if !slices.Contains(slots, p) {
			slots = append(slots, p)
		}
	}
	return slots
}

// cellRound rounds v to the nearest cell, halves away from zero, without
// letting float error turn sin(30°)·2 = 0.99999… into the wrong side of a
// half.
func cellRound(v float64) int { return int(math.Round(math.Round(v*1e6) / 1e6)) }

// ringCapacity is how many nodes ring k holds: on the first sparseRings
// rings ringShare×k (or fewer when it has fewer slots), past them every
// slot. Inner rings are left part empty, so the size of a small galaxy
// follows its notes instead of jumping a whole ring at a time; a big one
// stays as compact as before.
func ringCapacity(k int) int {
	if k > sparseRings {
		return len(ringSlots(k))
	}
	return min(len(ringSlots(k)), ringShare*k)
}

// ringsFor is how many rings a new galaxy of n nodes reserves, its notes
// from ring first out: enough for all of them plus an eighth to grow
// into. The sun takes no slot.
func ringsFor(n, first int) int {
	need := n + n/8
	k, capacity := first-1, 0
	for capacity < need {
		k++
		capacity += ringCapacity(k)
	}
	return k
}

// fill gives the members, busiest first, the free slots of gx, inner rings
// first, each ring up to its capacity. A ring with more room than members
// left spreads them evenly round it. Unresolved targets go after notes;
// ties go by ID.
func fill(nodes []Placed, members []int, gx *galaxy, used map[Point]bool) {
	slices.SortFunc(members, func(a, b int) int {
		na, nb := nodes[a], nodes[b]
		return cmp.Or(cmpBool(na.Unresolved, nb.Unresolved), cmp.Compare(nb.Degree, na.Degree), cmp.Compare(na.ID, nb.ID))
	})
	next := 0
	for k := gx.firstRing(); k <= gx.rings && next < len(members); k++ {
		var free []Point
		slots := ringSlots(k)
		for _, s := range slots {
			if p := (Point{gx.c.X + s.X, gx.c.Y + s.Y}); !used[p] {
				free = append(free, p)
			}
		}
		room := ringCapacity(k) - (len(slots) - len(free))
		take := max(0, min(len(free), room, len(members)-next))
		for i := range take {
			p := free[i*len(free)/take]
			nodes[members[next]].P = p
			used[p] = true
			next++
		}
	}
}

// freeSlots counts the nodes the rings of gx still have room for, past
// those kept; taken holds offsets from the sun.
func freeSlots(gx *galaxy, taken map[Point]bool) int {
	free := 0
	for k := gx.firstRing(); k <= gx.rings; k++ {
		held := 0
		for _, s := range ringSlots(k) {
			if taken[s] {
				held++
			}
		}
		free += max(0, ringCapacity(k)-held)
	}
	return free
}

// makeRoom moves galaxies until gx, grown a ring, keeps the gap to every
// other: of two galaxies too close, the one farther from the vault's sun
// moves straight away from it (never the root's), and is checked again in
// turn. It reports false when that takes more moves than the map could
// need, which only a loop would.
func makeRoom(gx *galaxy, all []*galaxy, sun Point) bool {
	queue := []*galaxy{gx}
	for moves := 0; len(queue) > 0; {
		q := queue[0]
		queue = queue[1:]
		for _, o := range all {
			if o == q || !o.placed || apart(q.box(), o.box()) {
				continue
			}
			mv, still := o, q
			if o.root || !q.root && screenDist2(q.c, sun) > screenDist2(o.c, sun) {
				mv, still = q, o
			}
			if moves++; mv.root || moves > 4*len(all)*len(all) {
				return false
			}
			pushOut(mv, still, sun)
			queue = append(queue, mv)
			if mv == q {
				break // q moved: it is checked again from the queue
			}
		}
	}
	return true
}

// pushOut moves mv straight away from the vault's sun, a column at a time
// on screen, until it keeps the gap to still.
func pushOut(mv, still *galaxy, sun Point) {
	dx, dy := float64(mv.c.X-sun.X), float64(cellAspect*(mv.c.Y-sun.Y))
	d := math.Hypot(dx, dy)
	if d == 0 {
		dx, d = 1, 1
	}
	for t := d + 1; !apart(mv.box(), still.box()); t++ {
		mv.c = Point{sun.X + cellRound(dx/d*t), sun.Y + cellRound(dy/d*t/cellAspect)}
	}
}

// screenDist2 is the squared distance on screen between two cells, a row
// counting as cellAspect columns.
func screenDist2(a, b Point) int {
	dx, dy := a.X-b.X, cellAspect*(a.Y-b.Y)
	return dx*dx + dy*dy
}

// clusterLinks counts the links between each two folders' galaxies, by
// their index in gs, the lower index first. Links to the root's notes
// don't count.
func clusterLinks(nodes []Placed, pairs [][2]int, gs []*galaxy) map[[2]int]int {
	idx := make(map[string]int, len(gs))
	for i, gx := range gs {
		idx[gx.key] = i
	}
	links := map[[2]int]int{}
	for _, p := range pairs {
		a, okA := idx[nodes[p[0]].Cluster]
		b, okB := idx[nodes[p[1]].Cluster]
		if okA && okB && a != b {
			links[[2]int{min(a, b), max(a, b)}]++
		}
	}
	return links
}

func linkCount(links map[[2]int]int, a, b int) int {
	return links[[2]int{min(a, b), max(a, b)}]
}

// pullOrder lists the galaxies the way they go on the orbits: the biggest
// first, then again and again the one with most links to those listed
// (ties: bigger, then by key).
func pullOrder(gs []*galaxy, links map[[2]int]int) []int {
	if len(gs) == 0 {
		return nil
	}
	adj := make([][]int, len(gs))
	for p := range links {
		adj[p[0]] = append(adj[p[0]], p[1])
		adj[p[1]] = append(adj[p[1]], p[0])
	}
	first := 0
	for i, gx := range gs {
		if gx.size > gs[first].size {
			first = i
		}
	}
	pull, listed := make([]int, len(gs)), make([]bool, len(gs))
	order := []int{first}
	listed[first] = true
	for i := first; len(order) < len(gs); {
		for _, j := range adj[i] {
			pull[j] += linkCount(links, i, j)
		}
		i = -1
		for j, gx := range gs {
			if !listed[j] && (i < 0 || pull[j] > pull[i] || pull[j] == pull[i] && gx.size > gs[i].size) {
				i = j
			}
		}
		listed[i] = true
		order = append(order, i)
	}
	return order
}

// placeOrbits puts the root's sun at (0, 0) and the galaxies on orbits
// around it, in pullOrder: firstOrbit of them on the first, orbitGrowth
// more on each next one out.
func placeOrbits(root *galaxy, gs []*galaxy, links map[[2]int]int) {
	order := pullOrder(gs, links)
	placed := []*galaxy{root}
	rx := 1
	for k, start := 0, 0; start < len(order); k++ {
		n := min(firstOrbit+k*orbitGrowth, len(order)-start)
		on := orbitOrder(order[start:start+n], gs, links)
		rx = fitOrbit(on, placed, rx)
		for _, gx := range on {
			gx.placed = true
		}
		placed = append(placed, on...)
		start += n
	}
}

// orbitOrder lists one orbit's galaxies clockwise from the top. A galaxy
// that links to one already listed goes beside it, on the side of the
// neighbour it links to more; any other goes where the golden turn of its
// rank in ids (the biggest first) falls round the orbit, but never between
// two linked galaxies, so big and small galaxies take turns all the way
// round instead of the small ones piling up together.
func orbitOrder(ids []int, gs []*galaxy, links map[[2]int]int) []*galaxy {
	type seat struct {
		id int
		at float64 // the turn it is listed at
	}
	var round []seat
	for rank, j := range ids {
		best, most := -1, 0
		for s, o := range round {
			if w := linkCount(links, o.id, j); w > most {
				best, most = s, w
			}
		}
		if best >= 0 {
			pos := best + 1
			if n := len(round); n > 1 && linkCount(links, round[(best+n-1)%n].id, j) > linkCount(links, round[(best+1)%n].id, j) {
				pos = best
			}
			round = slices.Insert(round, pos, seat{j, round[best].at})
			continue
		}
		at := math.Mod(float64(rank)*goldenTurn, 1)
		pos := sort.Search(len(round), func(s int) bool { return round[s].at > at })
		for tries, n := 0, len(round); n > 1 && tries < n; tries++ { // never between two linked galaxies
			if linkCount(links, round[(pos+n-1)%n].id, round[pos%n].id) == 0 {
				break
			}
			pos = pos%n + 1
		}
		round = slices.Insert(round, pos, seat{j, at})
	}
	on := make([]*galaxy, len(round))
	for i, st := range round {
		on[i] = gs[st.id]
	}
	return on
}

// orbitSteps is how finely an orbit is walked round.
const orbitSteps = 720

// onOrbit is the cell at turn t (0 is the top, then clockwise, 1 a whole
// turn, measured along the orbit's length on screen) of the orbit of rx
// columns round sun: an ellipse (see orbitRound) rx/orbitFlat rows tall.
func onOrbit(sun Point, rx, t float64) Point {
	t -= math.Floor(t)
	lengths := orbitLengths()
	j := min(max(sort.SearchFloat64s(lengths, t), 1), len(lengths)-1)
	f := (t - lengths[j-1]) / (lengths[j] - lengths[j-1])
	x, y := orbitAt(float64(j-1) + f)
	return Point{sun.X + cellRound(rx*x), sun.Y + cellRound(rx/orbitFlat*y)}
}

// orbitAt is the point of the unit orbit at step s of orbitSteps, from
// the top clockwise.
func orbitAt(s float64) (x, y float64) {
	a := 2*math.Pi*s/orbitSteps - math.Pi/2
	curve := func(v float64) float64 { return math.Copysign(math.Pow(math.Abs(v), 2/orbitRound), v) }
	return curve(math.Cos(a)), curve(math.Sin(a))
}

// orbitLengths is, per step of the unit orbit, the share of its length on
// screen (a row counting as cellAspect columns) up to that step.
var orbitLengths = sync.OnceValue(func() []float64 {
	lengths := make([]float64, orbitSteps+1)
	px, py := orbitAt(0)
	for j := 1; j <= orbitSteps; j++ {
		x, y := orbitAt(float64(j))
		lengths[j] = lengths[j-1] + math.Hypot(x-px, cellAspect*(y-py)/orbitFlat)
		px, py = x, y
	}
	for j := range lengths {
		lengths[j] /= lengths[orbitSteps]
	}
	return lengths
})

// fitOrbit puts the galaxies of on, in that order, round the smallest
// orbit, from rx columns out, where they keep the gap to each other and to
// those placed, and returns its size. On each orbit tried the first goes
// on top (with one galaxy, on the right; with two, on the left), each next
// one at the first turn past the one before where it keeps the gap to it,
// and the room left over before the first comes round again is shared out
// evenly between them.
func fitOrbit(on []*galaxy, placed []*galaxy, rx int) int {
	start := 0.0
	switch len(on) {
	case 1:
		start = 0.25
	case 2:
		start = 0.75
	}
	sun := placed[0].c
	turns := make([]float64, len(on))
	// next is the first turn from t where gx keeps the gap to prev, or
	// one past the end of the orbit.
	next := func(gx, prev *galaxy, r, t float64) float64 {
		for ; t <= start+1; t += 1.0 / orbitSteps {
			if gx.c = onOrbit(sun, r, t); apart(gx.box(), prev.box()) {
				return t
			}
		}
		return start + 2
	}
	for ; ; rx += max(1, rx/32) {
		r := float64(rx)
		turns[0] = start
		on[0].c = onOrbit(sun, r, start)
		end := start
		for i := 1; i < len(on) && end <= start+1; i++ {
			turns[i] = next(on[i], on[i-1], r, turns[i-1])
			end = turns[i]
		}
		if len(on) > 1 && end <= start+1 {
			end = next(on[0], on[len(on)-1], r, end) // where the first may come round again
		}
		if end > start+1 {
			continue
		}
		slack := 0.0
		if len(on) > 1 {
			slack = (start + 1 - end) / float64(len(on))
		}
		for _, share := range []float64{slack, 0} {
			for i, gx := range on {
				gx.c = onOrbit(sun, r, turns[i]+share*float64(i))
			}
			if orbitFits(on, placed) {
				return rx
			}
		}
	}
}

func orbitFits(on, placed []*galaxy) bool {
	for i, gx := range on {
		b := gx.box()
		for _, o := range on[i+1:] {
			if !apart(b, o.box()) {
				return false
			}
		}
		for _, o := range placed {
			if !apart(b, o.box()) {
				return false
			}
		}
	}
	return true
}

// placeNew puts a new folder's galaxy x on an orbit round the vault's
// sun, tried in newAngles directions from the smallest orbit where it
// fits out to orbitSearch columns further: the spot nearest the galaxies
// it links to (weighted by links), else the smallest orbit, nearest the
// top. A spot that would push the map past the grid's top-left corner,
// moving every node, comes last.
func placeNew(x *galaxy, all []*galaxy, sun Point, links map[[2]int]int) {
	folders, self := all[1:], slices.Index(all[1:], x)
	type spot struct {
		corner bool
		cost   float64
		c      Point
	}
	var best *spot
	first := -1
	for rx := 1; first < 0 || rx <= first+orbitSearch; rx++ {
		for s := range newAngles {
			x.c = onOrbit(sun, float64(rx), float64(s)/newAngles)
			if !fitsAmong(x, all) {
				continue
			}
			if first < 0 {
				first = rx
			}
			b := x.box()
			sp := spot{corner: b.X < 0 || b.Y < 0, cost: float64(rx), c: x.c}
			for j, o := range folders {
				if w := linkCount(links, self, j); w > 0 && o.placed {
					sp.cost += linkPull * float64(w) * math.Sqrt(float64(screenDist2(x.c, o.c)))
				}
			}
			if best == nil || cmpBool(sp.corner, best.corner) < 0 || sp.corner == best.corner && sp.cost < best.cost {
				best = &sp
			}
		}
	}
	x.c, x.placed = best.c, true
}

// finish builds a layout from nodes placed by hand, with no suns.
func finish(nodes []Placed, clusters []Cluster) Layout { return build(nodes, clusters, nil) }

// build makes the layout of placed nodes: index, clusters, size, labels.
// With a sun the map is solar: the vault's sun there, each folder's in
// its cluster's centre, and no name covers one.
func build(nodes []Placed, clusters []Cluster, sun *Point) Layout {
	l := Layout{Nodes: nodes, index: make(map[string]int, len(nodes))}
	if len(nodes) == 0 {
		l.Nodes = nil
	}
	for i, n := range nodes {
		l.index[n.ID] = i
	}
	slices.SortFunc(clusters, func(a, b Cluster) int { return strings.Compare(a.Key, b.Key) })
	for _, c := range clusters {
		l.Clusters = append(l.Clusters, c)
		l.W, l.H = max(l.W, c.Rect.X+c.Rect.W), max(l.H, c.Rect.Y+c.Rect.H)
	}
	if sun != nil {
		l.Sun, l.solar = *sun, true
	}
	setLabels(nodes)
	suns := l.suns()
	l.Labels[ZoomMedium] = placeLabels(nodes, suns, ZoomMedium, mediumMinDegree(nodes))
	l.Labels[ZoomClose] = placeLabels(nodes, suns, ZoomClose, 0)
	return l
}

// suns lists the grid cells of the suns: the vault's, then each folder's.
func (l Layout) suns() []Point {
	if !l.solar {
		return nil
	}
	ps := []Point{l.Sun}
	for _, c := range l.Clusters {
		if c.Key != "" {
			ps = append(ps, c.Centre)
		}
	}
	return ps
}

// setLabels names each node by the shortest path suffix, without ".md",
// that no other node shares: "Note" alone, "a/Note" beside "b/Note".
func setLabels(nodes []Placed) {
	parts := make([][]string, len(nodes))
	for i, n := range nodes {
		parts[i] = strings.Split(strings.TrimSuffix(n.ID, ".md"), "/")
	}
	todo := make([]int, len(nodes))
	for i := range todo {
		todo[i] = i
	}
	for k := 1; len(todo) > 0; k++ {
		suffix := func(i int) string { return strings.Join(parts[i][max(0, len(parts[i])-k):], "/") }
		count := map[string]int{}
		for i := range nodes {
			count[suffix(i)]++
		}
		var left []int
		for _, i := range todo {
			if count[suffix(i)] == 1 || k >= len(parts[i]) {
				nodes[i].Label = suffix(i)
			} else {
				left = append(left, i)
			}
		}
		todo = left
	}
}

// mediumMinDegree is the degree a node needs for a name at medium zoom:
// that of the last node in the busiest 1/mediumLabelShare, and at least 1.
func mediumMinDegree(nodes []Placed) int {
	if len(nodes) == 0 {
		return 1
	}
	degrees := make([]int, len(nodes))
	for i, n := range nodes {
		degrees[i] = n.Degree
	}
	slices.Sort(degrees)
	slices.Reverse(degrees)
	return max(1, degrees[max(0, len(nodes)/mediumLabelShare-1)])
}

// placeLabels names nodes with at least minDegree links, busiest first,
// where the name, the space before it and the space after it cover no dot,
// no sun and no other name.
func placeLabels(nodes []Placed, suns []Point, z Zoom, minDegree int) []Label {
	sx, sy := z.fixedScale()
	taken := make(map[Point]bool, len(nodes)*4)
	for _, p := range suns {
		taken[Point{p.X * sx, p.Y * sy}] = true
	}
	order := make([]int, 0, len(nodes))
	for i, n := range nodes {
		taken[Point{n.P.X * sx, n.P.Y * sy}] = true
		if n.Degree >= minDegree {
			order = append(order, i)
		}
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Or(cmp.Compare(nodes[b].Degree, nodes[a].Degree), cmpBool(nodes[a].Unresolved, nodes[b].Unresolved))
	})
	var labels []Label
	for _, i := range order {
		n := nodes[i]
		lb := Label{ID: n.ID, X: n.P.X*sx + 2, Y: n.P.Y * sy, W: min(ansi.StringWidth(n.Label), maxLabelWidth)}
		if fits(lb, taken) {
			for x := lb.X - 1; x <= lb.X+lb.W; x++ {
				taken[Point{x, lb.Y}] = true
			}
			labels = append(labels, lb)
		}
	}
	return labels
}

func fits(lb Label, taken map[Point]bool) bool {
	for x := lb.X - 1; x <= lb.X+lb.W; x++ {
		if taken[Point{x, lb.Y}] {
			return false
		}
	}
	return true
}
