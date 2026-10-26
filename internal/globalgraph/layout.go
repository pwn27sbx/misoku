// Package globalgraph lays out and draws the map of every note.
package globalgraph

// The map is clustered by folder: every top-level folder (a project) gets a
// rectangle of the grid, rectangles are packed in shelves, biggest first,
// and inside one the busiest notes sit nearest the middle. One node takes
// one cell. Everything is sorted before it is placed, so the same notes and
// links always give the same map, whatever order they arrive in.

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

// Cluster is the area reserved for one top-level folder.
type Cluster struct {
	Key  string
	Rect Rect
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

// Scale returns how many screen cells one grid cell spans at z.
func (z Zoom) Scale() (sx, sy int) {
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
	gapX, gapY       = 3, 2  // empty cells between two clusters
	maxLabelWidth    = 24    // longer names are cut by the renderer
	mediumLabelShare = 5     // medium zoom names the busiest 1/5 of the nodes
	slackNum, slackD = 3, 2  // a new cluster reserves 3/2 of its members
	slackMin         = 2     // plus a couple of cells for small folders
	cellAspect       = 2.0   // a terminal cell is about twice as tall as wide
	noRect           = -1000 // marks a cluster with no rectangle yet
)

// Build lays out g from scratch.
func Build(g Graph) Layout {
	nodes := normalize(g)
	groups := groupByCluster(nodes)
	type shelfItem struct {
		key  string
		w, h int
	}
	items := make([]shelfItem, 0, len(groups))
	area := 0
	maxW := 0
	for _, key := range sortedKeys(groups) {
		w, h := shape(len(groups[key]))
		items = append(items, shelfItem{key, w, h})
		area += (w + gapX) * (h + gapY)
		maxW = max(maxW, w)
	}
	slices.SortStableFunc(items, func(a, b shelfItem) int { return cmp.Compare(b.w*b.h, a.w*a.h) })
	width := max(maxW, int(math.Ceil(math.Sqrt(cellAspect*float64(area)))))
	rects := map[string]Rect{}
	x, y, shelfH := 0, 0, 0
	for _, it := range items {
		if x > 0 && x+it.w > width {
			x, y, shelfH = 0, y+shelfH+gapY, 0
		}
		rects[it.key] = Rect{x, y, it.w, it.h}
		x += it.w + gapX
		shelfH = max(shelfH, it.h)
	}
	used := map[Point]bool{}
	for key, members := range groups {
		fill(nodes, members, rects[key], used)
	}
	return finish(nodes, rects)
}

// Update lays out g keeping every node of prev where it was: new nodes take
// the free cells nearest their cluster's middle, a full cluster grows into
// empty space, and a new folder gets a shelf of its own below the map. When
// a full cluster is hemmed in by others it returns a fresh Build and false.
func Update(prev Layout, g Graph) (Layout, bool) {
	nodes := normalize(g)
	rects := map[string]Rect{}
	for _, c := range prev.Clusters {
		rects[c.Key] = c.Rect
	}
	groups := groupByCluster(nodes)
	for key := range rects {
		if groups[key] == nil {
			delete(rects, key)
		}
	}
	used := map[Point]bool{}
	pending := map[string][]int{}
	for i := range nodes {
		n := &nodes[i]
		if old, ok := prev.Lookup(n.ID); ok && old.Cluster == n.Cluster {
			n.P = old.P
			used[n.P] = true
		} else {
			pending[n.Cluster] = append(pending[n.Cluster], i)
		}
	}
	for _, key := range sortedKeys(pending) {
		r, ok := rects[key]
		if !ok {
			w, h := shape(len(groups[key]))
			r = Rect{0, bottom(rects) + gapY, w, h}
			if len(rects) == 0 {
				r.Y = 0
			}
		}
		for freeCells(r, used) < len(pending[key]) {
			if r, ok = grow(key, r, rects); !ok {
				return Build(g), false
			}
		}
		rects[key] = r
		fill(nodes, pending[key], r, used)
	}
	return finish(nodes, rects), true
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
// distinct neighbours. Positions are left at noRect.
func normalize(g Graph) []Placed {
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
	for _, e := range g.Edges {
		a, okA := index[e.From]
		b, okB := index[e.To]
		if !okA || !okB || a == b {
			continue
		}
		pair := [2]int{min(a, b), max(a, b)}
		if !seen[pair] {
			seen[pair] = true
			nodes[a].Degree++
			nodes[b].Degree++
		}
	}
	return nodes
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

// shape sizes the rectangle for a cluster of n nodes with room to grow,
// about twice as wide as tall so it looks round on screen.
func shape(n int) (w, h int) {
	capacity := n*slackNum/slackD + slackMin
	w = int(math.Ceil(math.Sqrt(cellAspect * float64(capacity))))
	return w, (capacity + w - 1) / w
}

// fill gives the members, busiest first, the free cells of r nearest its
// middle. Unresolved targets go after notes; ties go by ID.
func fill(nodes []Placed, members []int, r Rect, used map[Point]bool) {
	slices.SortFunc(members, func(a, b int) int {
		na, nb := nodes[a], nodes[b]
		return cmp.Or(cmpBool(na.Unresolved, nb.Unresolved), cmp.Compare(nb.Degree, na.Degree), cmp.Compare(na.ID, nb.ID))
	})
	cells := cellsByDistance(r)
	next := 0
	for _, m := range members {
		for used[cells[next]] {
			next++
		}
		nodes[m].P = cells[next]
		used[cells[next]] = true
	}
}

// cellsByDistance lists the cells of r from its middle outwards; screen
// distance counts a row as cellAspect columns.
func cellsByDistance(r Rect) []Point {
	cx, cy := float64(r.W-1)/2, float64(r.H-1)/2
	dist := func(p Point) float64 {
		dx, dy := float64(p.X-r.X)-cx, cellAspect*(float64(p.Y-r.Y)-cy)
		return dx*dx + dy*dy
	}
	cells := make([]Point, 0, r.W*r.H)
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			cells = append(cells, Point{x, y})
		}
	}
	slices.SortStableFunc(cells, func(a, b Point) int { return cmp.Compare(dist(a), dist(b)) })
	return cells
}

func freeCells(r Rect, used map[Point]bool) int {
	free := r.W * r.H
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if used[Point{x, y}] {
				free--
			}
		}
	}
	return free
}

// bottom is the first row below every rectangle.
func bottom(rects map[string]Rect) int {
	b := 0
	for _, r := range rects {
		b = max(b, r.Y+r.H)
	}
	return b
}

// grow adds a row or column to r on the first side (down, right, left, up)
// that keeps the gap to every other cluster.
func grow(key string, r Rect, rects map[string]Rect) (Rect, bool) {
	for _, c := range []Rect{
		{r.X, r.Y, r.W, r.H + 1}, {r.X, r.Y, r.W + 1, r.H},
		{r.X - 1, r.Y, r.W + 1, r.H}, {r.X, r.Y - 1, r.W, r.H + 1},
	} {
		if c.X >= 0 && c.Y >= 0 && !crowds(key, c, rects) {
			return c, true
		}
	}
	return r, false
}

// crowds reports whether r comes within the gap of another cluster.
func crowds(key string, r Rect, rects map[string]Rect) bool {
	for k, o := range rects {
		if k != key && r.X < o.X+o.W+gapX && o.X < r.X+r.W+gapX && r.Y < o.Y+o.H+gapY && o.Y < r.Y+r.H+gapY {
			return true
		}
	}
	return false
}

// finish builds the layout from placed nodes: index, clusters, size, labels.
func finish(nodes []Placed, rects map[string]Rect) Layout {
	l := Layout{Nodes: nodes, index: make(map[string]int, len(nodes))}
	if len(nodes) == 0 {
		l.Nodes = nil
	}
	for i, n := range nodes {
		l.index[n.ID] = i
	}
	for _, key := range sortedKeys(rects) {
		r := rects[key]
		l.Clusters = append(l.Clusters, Cluster{key, r})
		l.W, l.H = max(l.W, r.X+r.W), max(l.H, r.Y+r.H)
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
	sx, sy := z.Scale()
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
