package globalgraph

import (
	"cmp"
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// fixture builds a vault-like graph: notes spread over clusters of uneven
// size (some at the root), a few unresolved targets and random links.
func fixture(notes, clusters, edges int, seed uint64) Graph {
	r := rand.New(rand.NewPCG(seed, seed^0x9e37))
	var g Graph
	for i := range notes {
		c := r.IntN(clusters) * r.IntN(clusters) / clusters // skewed sizes
		n := Node{ID: fmt.Sprintf("n%05d.md", i)}
		if c > 0 {
			n.Cluster = fmt.Sprintf("c%02d", c)
			n.ID = fmt.Sprintf("c%02d/sub/n%05d.md", c, i)
		}
		g.Nodes = append(g.Nodes, n)
	}
	for i := range notes / 20 {
		g.Nodes = append(g.Nodes, Node{ID: fmt.Sprintf("missing %d", i), Unresolved: true})
	}
	for range edges {
		a, b := g.Nodes[r.IntN(len(g.Nodes))], g.Nodes[r.IntN(len(g.Nodes))]
		g.Edges = append(g.Edges, Edge{a.ID, b.ID})
	}
	return g
}

func shuffled(g Graph, seed uint64) Graph {
	r := rand.New(rand.NewPCG(seed, 7))
	out := Graph{Nodes: append([]Node(nil), g.Nodes...), Edges: append([]Edge(nil), g.Edges...)}
	r.Shuffle(len(out.Nodes), func(i, j int) { out.Nodes[i], out.Nodes[j] = out.Nodes[j], out.Nodes[i] })
	r.Shuffle(len(out.Edges), func(i, j int) { out.Edges[i], out.Edges[j] = out.Edges[j], out.Edges[i] })
	for i, e := range out.Edges {
		if i%2 == 0 {
			out.Edges[i] = Edge{e.To, e.From}
		}
	}
	return out
}

// checkInvariants pins the geometry every layout must keep.
func checkInvariants(t *testing.T, l Layout, g Graph) {
	t.Helper()
	want := map[string]bool{}
	for _, n := range g.Nodes {
		want[n.ID] = true
	}
	if len(l.Nodes) != len(want) {
		t.Fatalf("placed %d nodes, want %d", len(l.Nodes), len(want))
	}
	galaxies := map[string]Cluster{}
	for _, c := range l.Clusters {
		galaxies[c.Key] = c
		if c.RX != cellAspect*c.RY || c.Rect != (Rect{c.Centre.X - c.RX, c.Centre.Y - c.RY, 2*c.RX + 1, 2*c.RY + 1}) {
			t.Fatalf("galaxy %q: rect %v is not the box of centre %v and radii %d×%d", c.Key, c.Rect, c.Centre, c.RX, c.RY)
		}
		if c.Rect.X < 0 || c.Rect.Y < 0 || c.Rect.X+c.Rect.W > l.W || c.Rect.Y+c.Rect.H > l.H {
			t.Fatalf("galaxy %q %v outside the %dx%d map", c.Key, c.Rect, l.W, l.H)
		}
	}
	cells := map[Point]string{}
	for _, n := range l.Nodes {
		if !want[n.ID] {
			t.Fatalf("unknown node %q placed", n.ID)
		}
		if other, ok := cells[n.P]; ok {
			t.Fatalf("%q and %q share cell %v", n.ID, other, n.P)
		}
		cells[n.P] = n.ID
		c, ok := galaxies[n.Cluster]
		if !ok || screenDist2(n.P, c.Centre) > c.RX*c.RX {
			t.Fatalf("%q at %v outside its galaxy %q %+v", n.ID, n.P, n.Cluster, c)
		}
		if n.P.X < 0 || n.P.Y < 0 || n.P.X >= l.W || n.P.Y >= l.H {
			t.Fatalf("%q at %v outside the %dx%d map", n.ID, n.P, l.W, l.H)
		}
	}
	for i, a := range l.Clusters {
		for _, b := range l.Clusters[i+1:] {
			// Round galaxies on screen, with an empty band between two.
			if d := a.RX + b.RX + galaxyGap; screenDist2(a.Centre, b.Centre) < d*d {
				t.Fatalf("galaxies %q %+v and %q %+v are closer than the gap", a.Key, a, b.Key, b)
			}
		}
	}
}

// ring is the ring of galaxy c that cell p sits nearest, 0 being the
// centre.
func ring(c Cluster, p Point) int {
	d := math.Sqrt(float64(screenDist2(p, c.Centre)))
	best := 0
	for k := 1; float64(ringRadius(k-1)) <= d; k++ {
		if math.Abs(d-float64(ringRadius(k))) < math.Abs(d-float64(ringRadius(best))) {
			best = k
		}
	}
	return best
}

func cluster(l Layout, key string) Cluster {
	for _, c := range l.Clusters {
		if c.Key == key {
			return c
		}
	}
	return Cluster{}
}

func positions(l Layout) map[string]Point {
	m := map[string]Point{}
	for _, n := range l.Nodes {
		m[n.ID] = n.P
	}
	return m
}

func TestBuildEmpty(t *testing.T) {
	l := Build(Graph{})
	if len(l.Nodes) != 0 || len(l.Clusters) != 0 || l.W != 0 || l.H != 0 {
		t.Fatalf("empty graph laid out as %+v", l)
	}
	for z := range zoomLevels {
		if len(l.Labels[z]) != 0 {
			t.Fatalf("zoom %d has labels %v", z, l.Labels[z])
		}
	}
	if _, ok := l.Lookup("x"); ok {
		t.Fatal("lookup in an empty layout found a node")
	}
}

func TestBuildSingleNode(t *testing.T) {
	g := Graph{Nodes: []Node{{ID: "Only.md"}}}
	l := Build(g)
	checkInvariants(t, l, g)
	n, ok := l.Lookup("Only.md")
	if !ok || n.Label != "Only" || n.Degree != 0 {
		t.Fatalf("Lookup = %+v, %v", n, ok)
	}
	if got := l.Labels[ZoomClose]; len(got) != 1 || got[0].ID != "Only.md" {
		t.Fatalf("close labels = %v, want the only note", got)
	}
}

func TestBuildIsDeterministicAndOrderIndependent(t *testing.T) {
	g := fixture(600, 8, 1200, 1)
	want := Build(g)
	for seed := range uint64(5) {
		if got := Build(shuffled(g, seed)); !reflect.DeepEqual(got, want) {
			t.Fatalf("shuffled input (seed %d) gave a different layout", seed)
		}
	}
	if got := Build(g); !reflect.DeepEqual(got, want) {
		t.Fatal("same input gave a different layout")
	}
}

func TestBuildKeepsGeometryInvariants(t *testing.T) {
	for _, tc := range []struct{ notes, clusters, edges int }{
		{2, 1, 1}, {10, 3, 0}, {108, 6, 150}, {700, 12, 2000}, {2000, 30, 100},
	} {
		t.Run(fmt.Sprintf("%d_notes", tc.notes), func(t *testing.T) {
			g := fixture(tc.notes, tc.clusters, tc.edges, 2)
			checkInvariants(t, Build(g), g)
		})
	}
}

func TestBuildPlacesOrphansAndUnresolved(t *testing.T) {
	g := Graph{
		Nodes: []Node{
			{ID: "a/Hub.md", Cluster: "a"}, {ID: "a/Orphan.md", Cluster: "a"},
			{ID: "Ghost", Unresolved: true}, {ID: "Root.md"},
		},
		Edges: []Edge{{"a/Hub.md", "Ghost"}, {"Root.md", "a/Hub.md"}},
	}
	l := Build(g)
	checkInvariants(t, l, g)
	for id, deg := range map[string]int{"a/Hub.md": 2, "a/Orphan.md": 0, "Ghost": 1, "Root.md": 1} {
		n, ok := l.Lookup(id)
		if !ok || n.Degree != deg {
			t.Fatalf("Lookup(%q) = %+v, %v; want degree %d", id, n, ok, deg)
		}
	}
	if n, _ := l.Lookup("Ghost"); !n.Unresolved {
		t.Fatal("unresolved flag lost")
	}
}

func TestDegreeIgnoresSelfDuplicateAndUnknownEdges(t *testing.T) {
	g := Graph{
		Nodes: []Node{{ID: "A.md"}, {ID: "B.md"}},
		Edges: []Edge{{"A.md", "B.md"}, {"B.md", "A.md"}, {"A.md", "A.md"}, {"A.md", "Nope.md"}},
	}
	l := Build(g)
	for _, id := range []string{"A.md", "B.md"} {
		if n, _ := l.Lookup(id); n.Degree != 1 {
			t.Fatalf("%s degree = %d, want 1", id, n.Degree)
		}
	}
}

func TestBuildPutsTheHubAtTheGalaxyCentre(t *testing.T) {
	g := fixture(300, 1, 0, 3) // one root cluster
	hub := g.Nodes[17].ID
	for _, n := range g.Nodes[:120] {
		if n.ID != hub {
			g.Edges = append(g.Edges, Edge{hub, n.ID})
		}
	}
	l := Build(g)
	checkInvariants(t, l, g)
	if h, _ := l.Lookup(hub); h.P != l.Clusters[0].Centre {
		t.Fatalf("hub at %v, galaxy centre at %v", h.P, l.Clusters[0].Centre)
	}
}

// busiest is the node a galaxy puts in its centre: the note with the most
// links, ties by ID, or an unresolved target when the galaxy has no note.
func busiest(nodes []Placed) Placed {
	best := nodes[0]
	for _, n := range nodes[1:] {
		if cmp.Or(cmpBool(n.Unresolved, best.Unresolved), cmp.Compare(best.Degree, n.Degree), cmp.Compare(n.ID, best.ID)) < 0 {
			best = n
		}
	}
	return best
}

func members(l Layout) map[string][]Placed {
	m := map[string][]Placed{}
	for _, n := range l.Nodes {
		m[n.Cluster] = append(m[n.Cluster], n)
	}
	return m
}

func TestGalaxyCentreIsTheBusiestNote(t *testing.T) {
	g := fixture(700, 8, 2500, 11)
	l := Build(g)
	checkInvariants(t, l, g)
	for key, ns := range members(l) {
		c := cluster(l, key)
		if b := busiest(ns); b.P != c.Centre {
			t.Errorf("galaxy %q: busiest %q (degree %d) at %v, centre %v", key, b.ID, b.Degree, b.P, c.Centre)
		}
	}
}

func TestGalaxyRingsFollowDegree(t *testing.T) {
	g := fixture(700, 8, 2500, 12)
	l := Build(g)
	for key, ns := range members(l) {
		c := cluster(l, key)
		for _, a := range ns {
			if ra := ring(c, a.P); ra == 0 && a.P != c.Centre {
				t.Fatalf("galaxy %q: %q at %v rounds onto the centre", key, a.ID, a.P)
			}
			for _, b := range ns {
				if a.Unresolved == b.Unresolved && a.Degree > b.Degree && ring(c, a.P) > ring(c, b.P) {
					t.Fatalf("galaxy %q: %q (degree %d) on ring %d, outside %q (degree %d) on ring %d",
						key, a.ID, a.Degree, ring(c, a.P), b.ID, b.Degree, ring(c, b.P))
				}
			}
		}
		// Unresolved targets go on rings outside the notes.
		for _, a := range ns {
			for _, b := range ns {
				if !a.Unresolved && b.Unresolved && ring(c, a.P) > ring(c, b.P) {
					t.Fatalf("galaxy %q: note %q outside unresolved %q", key, a.ID, b.ID)
				}
			}
		}
	}
}

func TestGalaxiesLookRoundOnScreen(t *testing.T) {
	g := fixture(1200, 6, 3000, 13)
	l := Build(g)
	checked := 0
	for key, ns := range members(l) {
		if len(ns) < 40 {
			continue
		}
		c := cluster(l, key)
		// The outer ring may be part filled; the full rings inside it
		// give the galaxy its shape.
		outer := 0
		for _, n := range ns {
			outer = max(outer, ring(c, n.P))
		}
		dx, dy := 0, 0
		for _, n := range ns {
			if ring(c, n.P) < outer {
				dx, dy = max(dx, abs(n.P.X-c.Centre.X)), max(dy, abs(n.P.Y-c.Centre.Y))
			}
		}
		// Twice as many columns as rows looks round in cells twice as
		// tall as wide.
		if ratio := float64(dx) / float64(cellAspect*dy); ratio < 0.8 || ratio > 1.25 {
			t.Errorf("galaxy %q of %d notes spans %d columns by %d rows from its centre: not round", key, len(ns), dx, dy)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no galaxy big enough to check")
	}
}

func TestLinkedGalaxiesSitCloser(t *testing.T) {
	var g Graph
	add := func(key string, n int) {
		for i := range n {
			g.Nodes = append(g.Nodes, Node{ID: fmt.Sprintf("%s/n%02d.md", key, i), Cluster: key})
		}
	}
	link := func(a, b string, n int) {
		for i := range n {
			g.Edges = append(g.Edges, Edge{fmt.Sprintf("%s/n%02d.md", a, i%20), fmt.Sprintf("%s/n%02d.md", b, (i*7)%20)})
		}
	}
	// The stranger sorts first and is as big as the others, so only the
	// links can push it away.
	add("main", 60)
	add("a-stranger", 20)
	add("m-acquaintance", 20)
	add("z-friend", 20)
	link("main", "z-friend", 30)
	link("main", "m-acquaintance", 2)
	link("a-stranger", "a-stranger", 10)
	for _, in := range []Graph{g, shuffled(g, 3)} {
		l := Build(in)
		checkInvariants(t, l, in)
		main := cluster(l, "main").Centre
		d := func(key string) int { return screenDist2(main, cluster(l, key).Centre) }
		if d("z-friend") > d("m-acquaintance") || d("m-acquaintance") >= d("a-stranger") {
			t.Fatalf("distance² to main: friend %d, acquaintance %d, stranger %d; want friend ≤ acquaintance < stranger",
				d("z-friend"), d("m-acquaintance"), d("a-stranger"))
		}
	}
}

func TestUpdatePutsANewNoteOnTheInnermostFreeSlot(t *testing.T) {
	base := fixture(400, 6, 800, 4)
	prev := Build(base)
	key := prev.Clusters[1].Key
	g := Graph{Nodes: append(append([]Node(nil), base.Nodes...), Node{ID: key + "/new.md", Cluster: key}), Edges: base.Edges}
	l, kept := Update(prev, g)
	if !kept {
		t.Fatal("Update fell back to a full layout")
	}
	checkInvariants(t, l, g)
	c := cluster(l, key)
	n, _ := l.Lookup(key + "/new.md")
	// Every slot of the rings inside the new note's ring is taken.
	taken := map[Point]bool{}
	for _, m := range l.Nodes {
		taken[m.P] = true
	}
	for k := range ring(c, n.P) {
		for _, s := range ringSlots(k) {
			if p := (Point{c.Centre.X + s.X, c.Centre.Y + s.Y}); !taken[p] {
				t.Fatalf("new note at %v on ring %d, but %v on ring %d is free", n.P, ring(c, n.P), p, k)
			}
		}
	}
}

func TestUpdateAddingOneNoteMovesNoOtherNode(t *testing.T) {
	base := fixture(400, 6, 800, 4)
	prev := Build(base)
	before := positions(prev)
	cases := map[string]Node{
		"existing cluster": {ID: "c01/sub/new.md", Cluster: "c01"},
		"root":             {ID: "new.md"},
		"new cluster":      {ID: "zz/new.md", Cluster: "zz"},
		"unresolved":       {ID: "Nowhere", Unresolved: true},
	}
	for name, add := range cases {
		t.Run(name, func(t *testing.T) {
			g := Graph{Nodes: append(append([]Node(nil), base.Nodes...), add), Edges: append([]Edge(nil), base.Edges...)}
			// Links to the busiest notes change degrees but must not move them.
			for _, n := range base.Nodes[:10] {
				g.Edges = append(g.Edges, Edge{add.ID, n.ID})
			}
			l, kept := Update(prev, g)
			if !kept {
				t.Fatal("Update fell back to a full layout")
			}
			checkInvariants(t, l, g)
			after := positions(l)
			for id, p := range before {
				if q := after[id]; q != p {
					t.Fatalf("%q moved from %v to %v", id, p, q)
				}
			}
			if l2, _ := Update(prev, shuffled(g, 9)); !reflect.DeepEqual(l2, l) {
				t.Fatal("Update depends on input order")
			}
		})
	}
}

func TestUpdateRemovingNotesMovesNoOtherNode(t *testing.T) {
	base := fixture(200, 4, 300, 5)
	prev := Build(base)
	g := Graph{Nodes: base.Nodes[5:], Edges: base.Edges}
	l, kept := Update(prev, g)
	if !kept {
		t.Fatal("Update fell back to a full layout")
	}
	checkInvariants(t, l, g)
	after := positions(l)
	for id, p := range positions(prev) {
		if q, ok := after[id]; ok && q != p {
			t.Fatalf("%q moved from %v to %v", id, p, q)
		}
	}
}

func TestUpdateGrowsFullClustersWithoutOverlap(t *testing.T) {
	g := fixture(300, 9, 100, 6)
	l := Build(g)
	// The biggest galaxy is laid out first, in the middle, with the
	// others around it, so it can only grow until it reaches them.
	key, most := "", 0
	for k, ns := range members(l) {
		if len(ns) > most || len(ns) == most && k < key {
			key, most = k, len(ns)
		}
	}
	for i := range 2000 {
		before := positions(l)
		g.Nodes = append(g.Nodes, Node{ID: fmt.Sprintf("%s/grow%04d.md", key, i), Cluster: key})
		next, kept := Update(l, g)
		checkInvariants(t, next, g)
		if !kept {
			if !reflect.DeepEqual(next, Build(g)) {
				t.Fatal("fallback is not a fresh layout")
			}
			return // the cluster was hemmed in: a full layout is the way out
		}
		after := positions(next)
		for id, p := range before {
			if q := after[id]; q != p {
				t.Fatalf("step %d: %q moved from %v to %v", i, id, p, q)
			}
		}
		l = next
	}
	t.Fatalf("cluster %q never ran out of room: the fallback is untested", key)
}

func TestLabelsDisambiguateSameNames(t *testing.T) {
	g := Graph{Nodes: []Node{
		{ID: "a/Note.md", Cluster: "a"}, {ID: "b/Note.md", Cluster: "b"},
		{ID: "x/deep/Same.md", Cluster: "x"}, {ID: "y/deep/Same.md", Cluster: "y"},
		{ID: "Other.md"}, {ID: "Ghost", Unresolved: true},
	}}
	want := map[string]string{
		"a/Note.md": "a/Note", "b/Note.md": "b/Note",
		"x/deep/Same.md": "x/deep/Same", "y/deep/Same.md": "y/deep/Same",
		"Other.md": "Other", "Ghost": "Ghost",
	}
	l := Build(g)
	for id, label := range want {
		if n, _ := l.Lookup(id); n.Label != label {
			t.Errorf("label of %q = %q, want %q", id, n.Label, label)
		}
	}
}

// labelCells is every screen cell a label claims: the space after its dot,
// its text and one space after it.
func labelCells(lb Label) []Point {
	var ps []Point
	for x := lb.X - 1; x <= lb.X+lb.W; x++ {
		ps = append(ps, Point{x, lb.Y})
	}
	return ps
}

func TestLabelsAreCollisionFreeAtEveryZoom(t *testing.T) {
	g := fixture(1500, 10, 3000, 7)
	l := Build(g)
	if len(l.Labels[ZoomOverview]) != 0 {
		t.Fatalf("overview shows %d labels, want none", len(l.Labels[ZoomOverview]))
	}
	if len(l.Labels[ZoomMedium]) == 0 || len(l.Labels[ZoomClose]) <= len(l.Labels[ZoomMedium]) {
		t.Fatalf("medium %d labels, close %d: close must show more", len(l.Labels[ZoomMedium]), len(l.Labels[ZoomClose]))
	}
	for _, z := range []Zoom{ZoomMedium, ZoomClose} {
		sx, sy := z.fixedScale()
		dots := map[Point]bool{}
		for _, n := range l.Nodes {
			dots[Point{n.P.X * sx, n.P.Y * sy}] = true
		}
		used := map[Point]string{}
		for _, lb := range l.Labels[z] {
			n, _ := l.Lookup(lb.ID)
			if lb.X != n.P.X*sx+2 || lb.Y != n.P.Y*sy || lb.W != min(ansi.StringWidth(n.Label), maxLabelWidth) {
				t.Fatalf("zoom %d: label %+v not beside %q at %v", z, lb, n.ID, n.P)
			}
			for _, c := range labelCells(lb) {
				if dots[c] {
					t.Fatalf("zoom %d: label of %q covers a node at %v", z, lb.ID, c)
				}
				if other, ok := used[c]; ok {
					t.Fatalf("zoom %d: labels of %q and %q overlap at %v", z, lb.ID, other, c)
				}
				used[c] = lb.ID
			}
		}
	}
}

func TestCloseZoomLabelsEveryNodeThatFits(t *testing.T) {
	g := fixture(800, 6, 1500, 8)
	l := Build(g)
	sx, sy := ZoomClose.fixedScale()
	taken := map[Point]bool{}
	for _, n := range l.Nodes {
		taken[Point{n.P.X * sx, n.P.Y * sy}] = true
	}
	shown := map[string]bool{}
	for _, lb := range l.Labels[ZoomClose] {
		shown[lb.ID] = true
		for _, c := range labelCells(lb) {
			taken[c] = true
		}
	}
	for _, n := range l.Nodes {
		if shown[n.ID] {
			continue
		}
		lb := Label{ID: n.ID, X: n.P.X*sx + 2, Y: n.P.Y * sy, W: min(ansi.StringWidth(n.Label), maxLabelWidth)}
		free := true
		for _, c := range labelCells(lb) {
			free = free && !taken[c]
		}
		if free {
			t.Fatalf("label of %q fits at close zoom but is not shown", n.ID)
		}
	}
}

func TestMediumZoomLabelsOnlyBusyNodes(t *testing.T) {
	g := fixture(500, 5, 900, 9)
	l := Build(g)
	degrees := make([]int, 0, len(l.Nodes))
	for _, n := range l.Nodes {
		degrees = append(degrees, n.Degree)
	}
	for _, lb := range l.Labels[ZoomMedium] {
		n, _ := l.Lookup(lb.ID)
		busier := 0
		for _, d := range degrees {
			if d > n.Degree {
				busier++
			}
		}
		if n.Degree == 0 || busier >= len(degrees)/mediumLabelShare {
			t.Fatalf("medium zoom labels %q with degree %d (%d nodes busier)", n.ID, n.Degree, busier)
		}
	}
}

// BenchmarkLayout5k budget: 50 ms/op. Build runs off the Update loop, once
// per index change; the plan's frame budget (16 ms) is for the renderer.
func BenchmarkLayout5k(b *testing.B) {
	g := fixture(5000, 20, 10000, 10)
	b.ReportAllocs()
	for b.Loop() {
		Build(g)
	}
}

// BenchmarkLayoutManyFolders: placing galaxies costs about the square of
// the folders, so a vault of many top-level folders must stay in budget too.
func BenchmarkLayoutManyFolders(b *testing.B) {
	g := fixture(5000, 400, 20000, 14)
	b.ReportAllocs()
	for b.Loop() {
		Build(g)
	}
}
