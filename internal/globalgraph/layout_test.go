package globalgraph

import (
	"fmt"
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
	rects := map[string]Rect{}
	for _, c := range l.Clusters {
		rects[c.Key] = c.Rect
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
		r, ok := rects[n.Cluster]
		if !ok || n.P.X < r.X || n.P.X >= r.X+r.W || n.P.Y < r.Y || n.P.Y >= r.Y+r.H {
			t.Fatalf("%q at %v outside its cluster %q %v", n.ID, n.P, n.Cluster, r)
		}
		if n.P.X < 0 || n.P.Y < 0 || n.P.X >= l.W || n.P.Y >= l.H {
			t.Fatalf("%q at %v outside the %dx%d map", n.ID, n.P, l.W, l.H)
		}
	}
	for i, a := range l.Clusters {
		for _, b := range l.Clusters[i+1:] {
			// One empty cell at least between two clusters.
			if a.Rect.X < b.Rect.X+b.Rect.W+1 && b.Rect.X < a.Rect.X+a.Rect.W+1 &&
				a.Rect.Y < b.Rect.Y+b.Rect.H+1 && b.Rect.Y < a.Rect.Y+a.Rect.H+1 {
				t.Fatalf("clusters %q %v and %q %v overlap or touch", a.Key, a.Rect, b.Key, b.Rect)
			}
		}
	}
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

func TestBuildPutsHubsNearClusterCentre(t *testing.T) {
	g := fixture(300, 1, 0, 3) // one root cluster
	hub := g.Nodes[17].ID
	for _, n := range g.Nodes[:120] {
		if n.ID != hub {
			g.Edges = append(g.Edges, Edge{hub, n.ID})
		}
	}
	l := Build(g)
	c := l.Clusters[0].Rect
	dist := func(p Point) float64 {
		dx := float64(p.X) - float64(c.X) - float64(c.W-1)/2
		dy := 2 * (float64(p.Y) - float64(c.Y) - float64(c.H-1)/2)
		return dx*dx + dy*dy
	}
	h, _ := l.Lookup(hub)
	for _, n := range l.Nodes {
		if n.ID != hub && dist(n.P) < dist(h.P) {
			t.Fatalf("%q (degree %d) is nearer the centre than the hub", n.ID, n.Degree)
		}
	}
	// Linked notes come before orphans and unresolved targets after notes.
	for _, n := range l.Nodes {
		for _, m := range l.Nodes {
			if n.Degree > m.Degree && n.Unresolved == m.Unresolved && dist(n.P) > dist(m.P) {
				t.Fatalf("%q (degree %d) is farther out than %q (degree %d)", n.ID, n.Degree, m.ID, m.Degree)
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
	// The top-left cluster has a neighbour on its right and a shelf below,
	// so it can only grow until it reaches them.
	key := ""
	for _, c := range l.Clusters {
		if c.Rect.X == 0 && c.Rect.Y == 0 {
			key = c.Key
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
