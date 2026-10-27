// Package globalgraph lays out and draws the map of every note.
package globalgraph

// The map is a set of galaxies, one per top-level folder (a project): the
// folder's busiest note sits in the centre and the others on rings around
// it, busier notes on inner rings. Rings are twice as wide as tall, so a
// galaxy looks round in terminal cells. The biggest galaxy goes first, in
// the middle; each next one is the one most linked to those already
// placed, and goes where it is nearest to them, with an empty band around
// every galaxy and a wider one between galaxies with no link. One node
// takes one cell. Everything is sorted before it is placed, so the same
// notes and links always give the same map, whatever order they arrive in.

import (
	"cmp"
	"math"
	"slices"
	"strings"

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

// Cluster is the galaxy of one top-level folder: the ellipse of RX columns
// by RY rows around Centre that its rings may use (RX = 2×RY, round on
// screen), and Rect, the box around that ellipse.
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
// overview at 1×1. The overview drawn on screen is scaled to fit the pane
// (see Renderer.scale); this fixed one only places the layout's labels,
// which the overview has none of.
func (z Zoom) fixedScale() (sx, sy int) {
	switch z {
	case ZoomMedium:
		return 3, 2
	case ZoomClose:
		return 6, 2
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
	Labels   [zoomLevels][]Label
	index    map[string]int
}

const (
	maxLabelWidth    = 24    // longer names are cut by the renderer
	mediumLabelShare = 5     // medium zoom names the busiest 1/5 of the nodes
	cellAspect       = 2     // a terminal cell is about twice as tall as wide
	ringStep         = 4     // screen columns between two rings of a galaxy; the first is half as far out
	ringMargin       = 2     // columns a galaxy reserves past its outer ring for rounding
	dotSpacing       = 2.5   // screen columns between two slots of a ring
	galaxyGap        = 8     // least screen columns between two galaxies
	looseGap         = 6     // extra columns between two galaxies with no link
	placeAngles      = 24    // directions tried around a placed galaxy
	maxAnchors       = 12    // placed galaxies a new one is tried beside
	paneAspect       = 3     // columns per row of a usual pane (200×60)
	linkPull         = 1     // weight of the distance to linked galaxies against the map's size
	noRect           = -1000 // marks a node with no place yet
)

// Build lays out g from scratch.
func Build(g Graph) Layout {
	nodes, pairs := normalize(g)
	groups := groupByCluster(nodes)
	keys := sortedKeys(groups)
	gs := make([]*galaxy, len(keys))
	for i, key := range keys {
		gs[i] = &galaxy{key: key, size: len(groups[key]), rings: ringsFor(len(groups[key]))}
	}
	links := clusterLinks(nodes, pairs, keys)
	placeAll(gs, links)
	if len(gs) > 0 {
		// Move the map so its top-left galaxy edge is at (0, 0).
		x0, y0 := math.MaxInt, math.MaxInt
		for _, gx := range gs {
			x0, y0 = min(x0, gx.c.X-gx.rx()), min(y0, gx.c.Y-gx.ry())
		}
		for _, gx := range gs {
			gx.c.X -= x0
			gx.c.Y -= y0
		}
	}
	used := map[Point]bool{}
	for _, gx := range gs {
		fill(nodes, groups[gx.key], gx, used)
	}
	return finish(nodes, clustersOf(gs))
}

// Update lays out g keeping every node of prev where it was: new nodes take
// the free ring slots nearest their galaxy's centre, a full galaxy grows a
// ring while it keeps the gap to the others, and a new folder gets a galaxy
// of its own beside those it links to. When a full galaxy is hemmed in by
// others it returns a fresh Build and false.
func Update(prev Layout, g Graph) (Layout, bool) {
	nodes, pairs := normalize(g)
	groups := groupByCluster(nodes)
	keys := sortedKeys(groups)
	gs := make([]*galaxy, len(keys))
	byKey := map[string]*galaxy{}
	for i, key := range keys {
		gs[i] = &galaxy{key: key, size: len(groups[key])}
		byKey[key] = gs[i]
	}
	for _, c := range prev.Clusters {
		if gx := byKey[c.Key]; gx != nil {
			gx.c, gx.rings, gx.placed = c.Centre, max(0, (c.RX-ringMargin+ringStep/2)/ringStep), true
		}
	}
	used := map[Point]bool{}
	pending := map[string][]int{}
	for i := range nodes {
		n := &nodes[i]
		if old, ok := prev.Lookup(n.ID); ok && old.Cluster == n.Cluster && byKey[n.Cluster].placed {
			n.P = old.P
			used[n.P] = true
		} else {
			pending[n.Cluster] = append(pending[n.Cluster], i)
		}
	}
	var links map[[2]int]int
	for _, key := range sortedKeys(pending) {
		gx := byKey[key]
		if !gx.placed {
			if links == nil {
				links = clusterLinks(nodes, pairs, keys)
			}
			gx.rings = ringsFor(gx.size)
			placeNew(gs, slices.Index(keys, key), links, true)
		}
		for freeSlots(gx, used) < len(pending[key]) {
			if !growRing(gx, gs) {
				return Build(g), false
			}
		}
		fill(nodes, pending[key], gx, used)
	}
	return finish(nodes, clustersOf(gs)), true
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

// galaxy is a cluster while it is laid out: rings of slots around c.
type galaxy struct {
	key    string
	size   int   // members
	rings  int   // rings reserved around the centre slot
	c      Point // centre cell
	placed bool
}

// rx and ry are the radii of the ellipse the galaxy reserves, in columns
// and rows: its outer ring plus a margin for rounding slots to cells.
func (gx *galaxy) rx() int { return ringRadius(gx.rings) + ringMargin }
func (gx *galaxy) ry() int { return gx.rx() / cellAspect }

func clustersOf(gs []*galaxy) []Cluster {
	cs := make([]Cluster, 0, len(gs))
	for _, gx := range gs {
		rx, ry := gx.rx(), gx.ry()
		cs = append(cs, Cluster{Key: gx.key, Rect: Rect{gx.c.X - rx, gx.c.Y - ry, 2*rx + 1, 2*ry + 1}, Centre: gx.c, RX: rx, RY: ry})
	}
	return cs
}

// ringRadius is how many screen columns ring k is from the centre: 2, 6,
// 10… so the first ring hugs the hub; the centre is ring 0.
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

// ringsFor is how many rings a new galaxy of n nodes reserves: enough for
// all of them plus a quarter, and at least two more, to grow into.
func ringsFor(n int) int {
	need := n + max(2, n/4)
	k, capacity := 0, 1
	for capacity < need {
		k++
		capacity += len(ringSlots(k))
	}
	return k
}

// fill gives the members, busiest first, the free slots of gx, inner rings
// first. A ring with more free slots than members left spreads them evenly
// round it. Unresolved targets go after notes; ties go by ID.
func fill(nodes []Placed, members []int, gx *galaxy, used map[Point]bool) {
	slices.SortFunc(members, func(a, b int) int {
		na, nb := nodes[a], nodes[b]
		return cmp.Or(cmpBool(na.Unresolved, nb.Unresolved), cmp.Compare(nb.Degree, na.Degree), cmp.Compare(na.ID, nb.ID))
	})
	next := 0
	for k := 0; k <= gx.rings && next < len(members); k++ {
		var free []Point
		for _, s := range ringSlots(k) {
			if p := (Point{gx.c.X + s.X, gx.c.Y + s.Y}); !used[p] {
				free = append(free, p)
			}
		}
		take := min(len(free), len(members)-next)
		for i := range take {
			p := free[i*len(free)/take]
			nodes[members[next]].P = p
			used[p] = true
			next++
		}
	}
}

func freeSlots(gx *galaxy, used map[Point]bool) int {
	free := 0
	for k := 0; k <= gx.rings; k++ {
		for _, s := range ringSlots(k) {
			if !used[Point{gx.c.X + s.X, gx.c.Y + s.Y}] {
				free++
			}
		}
	}
	return free
}

// growRing adds a ring to gx when the bigger galaxy keeps the gap to every
// other placed one and stays inside the map's top-left corner.
func growRing(gx *galaxy, gs []*galaxy) bool {
	gx.rings++
	if gx.c.X-gx.rx() < 0 || gx.c.Y-gx.ry() < 0 {
		gx.rings--
		return false
	}
	for _, o := range gs {
		if o != gx && o.placed && screenDist2(gx.c, o.c) < sq(gx.rx()+o.rx()+galaxyGap) {
			gx.rings--
			return false
		}
	}
	return true
}

// screenDist2 is the squared distance on screen between two cells, a row
// counting as cellAspect columns.
func screenDist2(a, b Point) int {
	dx, dy := a.X-b.X, cellAspect*(a.Y-b.Y)
	return dx*dx + dy*dy
}

func sq(n int) int { return n * n }

// clusterLinks counts the links between each two galaxies, by their index
// in keys, the lower index first.
func clusterLinks(nodes []Placed, pairs [][2]int, keys []string) map[[2]int]int {
	idx := make(map[string]int, len(keys))
	for i, k := range keys {
		idx[k] = i
	}
	links := map[[2]int]int{}
	for _, p := range pairs {
		a, b := idx[nodes[p[0]].Cluster], idx[nodes[p[1]].Cluster]
		if a != b {
			links[[2]int{min(a, b), max(a, b)}]++
		}
	}
	return links
}

func linkCount(links map[[2]int]int, a, b int) int {
	return links[[2]int{min(a, b), max(a, b)}]
}

// gapFor is the least screen columns between two galaxies with w links.
func gapFor(w int) int {
	if w > 0 {
		return galaxyGap
	}
	return galaxyGap + looseGap
}

// placeAll places every galaxy of a fresh map: the biggest at (0, 0), then
// again and again the one with most links to those placed (ties: bigger,
// then by key) where placeNew puts it.
func placeAll(gs []*galaxy, links map[[2]int]int) {
	if len(gs) == 0 {
		return
	}
	adj := make([][]int, len(gs))
	for p := range links {
		adj[p[0]] = append(adj[p[0]], p[1])
		adj[p[1]] = append(adj[p[1]], p[0])
	}
	pull := make([]int, len(gs)) // links to placed galaxies
	first := 0
	for i, gx := range gs {
		if gx.size > gs[first].size {
			first = i
		}
	}
	gs[first].placed = true
	for i := first; ; {
		for _, j := range adj[i] {
			pull[j] += linkCount(links, i, j)
		}
		i = -1
		for j, gx := range gs {
			if !gx.placed && (i < 0 || pull[j] > pull[i] || pull[j] == pull[i] && gx.size > gs[i].size) {
				i = j
			}
		}
		if i < 0 {
			return
		}
		placeNew(gs, i, links, false)
	}
}

// placeNew puts galaxy i next to the placed galaxies: tried in placeAngles
// directions just past each of its anchors (the placed galaxies it links
// to, else the last ones placed), it takes the free spot that keeps the
// map smallest and shaped like a pane, and nearest its linked galaxies,
// weighted by links. With keepCorner (Update), spots must keep the map's
// top-left corner where it is, so no placed node has to move.
func placeNew(gs []*galaxy, i int, links map[[2]int]int, keepCorner bool) {
	x := gs[i]
	var placed, anchors []int
	w := make([]int, len(gs)) // links from i to each galaxy
	for j, gx := range gs {
		if gx.placed {
			placed = append(placed, j)
			if w[j] = linkCount(links, i, j); w[j] > 0 {
				anchors = append(anchors, j)
			}
		}
	}
	defer func() { x.placed = true }()
	if len(placed) == 0 {
		x.c = Point{x.rx(), x.ry()}
		return
	}
	linked := len(anchors) > 0
	if linked {
		slices.SortStableFunc(anchors, func(a, b int) int { return cmp.Compare(w[b], w[a]) })
	} else {
		anchors = placed[max(0, len(placed)-maxAnchors):]
	}
	anchors = anchors[:min(len(anchors), maxAnchors)]
	var mid Point
	for _, j := range placed {
		mid.X += gs[j].c.X
		mid.Y += gs[j].c.Y
	}
	mid = Point{mid.X / len(placed), mid.Y / len(placed)}
	fits := func(c Point) bool {
		if keepCorner && (c.X-x.rx() < 0 || c.Y-x.ry() < 0) {
			return false
		}
		for _, j := range placed {
			if screenDist2(c, gs[j].c) < sq(x.rx()+gs[j].rx()+gapFor(w[j])) {
				return false
			}
		}
		return true
	}
	// box is the area the placed galaxies span; a spot that widens it costs
	// its new width plus paneAspect times its new height, so the map grows
	// about as wide as a terminal pane.
	x0, y0, x1, y1 := math.MaxInt, math.MaxInt, math.MinInt, math.MinInt
	for _, j := range placed {
		g := gs[j]
		x0, y0 = min(x0, g.c.X-g.rx()), min(y0, g.c.Y-g.ry())
		x1, y1 = max(x1, g.c.X+g.rx()), max(y1, g.c.Y+g.ry())
	}
	cost := func(c Point) (float64, int) {
		bw := max(x1, c.X+x.rx()) - min(x0, c.X-x.rx())
		bh := max(y1, c.Y+x.ry()) - min(y0, c.Y-x.ry())
		f := float64(bw + paneAspect*bh + max(bw, paneAspect*bh))
		if linked {
			sum, total := 0.0, 0
			for _, j := range anchors {
				sum += float64(w[j]) * math.Sqrt(float64(screenDist2(c, gs[j].c)))
				total += w[j]
			}
			f += linkPull * sum / float64(total)
		}
		return f, screenDist2(c, mid)
	}
	around := func(o Point, d float64, s int) Point {
		a := 2 * math.Pi * float64(s) / placeAngles
		return Point{o.X + int(math.Round(d*math.Cos(a))), o.Y + int(math.Round(d*math.Sin(a)/cellAspect))}
	}
	found, bestW, bestD := false, 0.0, 0
	try := func(c Point) {
		if !fits(c) {
			return
		}
		w, d := cost(c)
		if !found || w < bestW || w == bestW && d < bestD {
			found, bestW, bestD, x.c = true, w, d, c
		}
	}
	for _, j := range anchors {
		d := float64(gs[j].rx()+x.rx()+gapFor(w[j])) + 1
		for s := range placeAngles {
			try(around(gs[j].c, d, s))
		}
	}
	// Hemmed in everywhere: circle out from the middle until a spot fits.
	for d := float64(ringStep); !found; d += ringStep {
		for s := range placeAngles {
			try(around(mid, d, s))
		}
	}
}

// finish builds the layout from placed nodes: index, clusters, size, labels.
func finish(nodes []Placed, clusters []Cluster) Layout {
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
	setLabels(nodes)
	l.Labels[ZoomMedium] = placeLabels(nodes, ZoomMedium, mediumMinDegree(nodes))
	l.Labels[ZoomClose] = placeLabels(nodes, ZoomClose, 0)
	return l
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
// where the name, the space before it and the space after it cover no dot
// and no other name.
func placeLabels(nodes []Placed, z Zoom, minDegree int) []Label {
	sx, sy := z.fixedScale()
	taken := make(map[Point]bool, len(nodes)*4)
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
