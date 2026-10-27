package globalgraph

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// vault is a small fixed vault: two projects, notes at the root, an orphan,
// an unresolved target and a CJK title.
func vault() Graph {
	return Graph{
		Nodes: []Node{
			{ID: "Home.md"}, {ID: "Inbox.md"}, {ID: "Orphan.md"},
			{ID: "Ideas", Unresolved: true},
			{ID: "work/Plan.md", Cluster: "work"}, {ID: "work/Tasks.md", Cluster: "work"},
			{ID: "work/Meeting.md", Cluster: "work"}, {ID: "work/Notes.md", Cluster: "work"},
			{ID: "study/Go.md", Cluster: "study"}, {ID: "study/Rust.md", Cluster: "study"},
			{ID: "study/Notes.md", Cluster: "study"}, {ID: "study/日本語のノート.md", Cluster: "study"},
		},
		Edges: []Edge{
			{"Home.md", "work/Plan.md"}, {"Home.md", "study/Go.md"}, {"Home.md", "Inbox.md"},
			{"work/Plan.md", "work/Tasks.md"}, {"work/Meeting.md", "work/Plan.md"},
			{"work/Tasks.md", "work/Notes.md"}, {"study/Go.md", "study/Rust.md"},
			{"study/Rust.md", "study/日本語のノート.md"}, {"study/Notes.md", "Ideas"},
			{"Inbox.md", "Ideas"}, {"study/Go.md", "work/Plan.md"}, {"Home.md", "study/Notes.md"},
		},
	}
}

func vaultFrame(r *Renderer, z Zoom, w, h int) Frame {
	return Frame{W: w, H: h, Zoom: z, Offset: r.Centred(z, w, h), Selected: "work/Plan.md", Open: "Home.md", Focused: true, Theme: theme.DefaultTheme()}
}

// trueColour renders in 24-bit colour for the rest of the test.
func trueColour(t testing.TB) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// tcell is one screen cell of a rendering, read back from its ANSI codes.
type tcell struct {
	r      rune
	fg, bg string // "r;g;b", "" for the terminal default
}

var sgrRe = regexp.MustCompile("^\x1b\\[([0-9;]*)m")

// screen reads a rendering back into cells; a wide character's right half
// is a cell with r == 0.
func screen(s string) [][]tcell {
	var rows [][]tcell
	for _, line := range strings.Split(s, "\n") {
		var row []tcell
		var fg, bg string
		for len(line) > 0 {
			if m := sgrRe.FindStringSubmatch(line); m != nil {
				fg, bg = applySGR(m[1], fg, bg)
				line = line[len(m[0]):]
				continue
			}
			r, size := utf8.DecodeRuneInString(line)
			line = line[size:]
			row = append(row, tcell{r, fg, bg})
			for range ansi.StringWidth(string(r)) - 1 {
				row = append(row, tcell{0, fg, bg})
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func applySGR(params, fg, bg string) (string, string) {
	p := strings.Split(params, ";")
	for k := 0; k < len(p); k++ {
		switch {
		case p[k] == "" || p[k] == "0":
			fg, bg = "", ""
		case (p[k] == "38" || p[k] == "48") && k+4 < len(p) && p[k+1] == "2":
			c := strings.Join(p[k+2:k+5], ";")
			if p[k] == "38" {
				fg = c
			} else {
				bg = c
			}
			k += 4
		case p[k] == "39":
			fg = ""
		case p[k] == "49":
			bg = ""
		}
	}
	return fg, bg
}

// rgb is a theme colour as the renderer writes it.
func rgb(c lipgloss.Color) string {
	cells := screen(lipgloss.NewStyle().Foreground(c).Render("x"))
	return cells[0][0].fg
}

// dotAt is the screen cell of a node's dot in frame f.
func dotAt(t *testing.T, r *Renderer, f Frame, id string) (x, y int) {
	t.Helper()
	n, ok := r.l.Lookup(id)
	if !ok {
		t.Fatalf("no node %q", id)
	}
	sx, sy := r.scale(f.Zoom, f.W, f.H)
	return n.P.X*sx - f.Offset.X, n.P.Y*sy - f.Offset.Y
}

func checkSize(t *testing.T, out string, w, h int, what string) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != h {
		t.Fatalf("%s: %d lines, want %d", what, len(lines), h)
	}
	for i, l := range lines {
		if got := ansi.StringWidth(l); got != w {
			t.Fatalf("%s: line %d is %d cells, want %d: %q", what, i, got, w, ansi.Strip(l))
		}
	}
}

func TestRenderIsExactlyThePaneSize(t *testing.T) {
	trueColour(t)
	graphs := map[string]Graph{"vault": vault(), "big": fixture(400, 6, 900, 3), "single": {Nodes: []Node{{ID: "Only.md"}}}, "empty": {}}
	sizes := [][2]int{{1, 1}, {2, 3}, {29, 8}, {30, 7}, {30, 8}, {31, 9}, {80, 24}, {200, 60}, {300, 80}}
	for name, g := range graphs {
		r := NewRenderer(Build(g), g.Edges)
		for _, s := range sizes {
			for z := range zoomLevels {
				c := r.Centred(z, s[0], s[1])
				for _, off := range []Point{c, {c.X - 7, c.Y + 3}, {-500, -500}, {500, 500}, {c.X + 1, c.Y}} {
					for _, sel := range []string{"", "work/Plan.md", "study/日本語のノート.md", "n00010.md"} {
						f := Frame{W: s[0], H: s[1], Zoom: z, Offset: off, Selected: sel, Open: "Home.md", Focused: true, Theme: theme.DefaultTheme()}
						checkSize(t, r.Render(f), s[0], s[1], name+" "+strconv.Itoa(s[0])+"x"+strconv.Itoa(s[1]))
					}
				}
			}
		}
	}
}

func TestRenderZeroSizeIsEmpty(t *testing.T) {
	r := NewRenderer(Build(vault()), vault().Edges)
	for _, s := range [][2]int{{0, 0}, {0, 10}, {10, 0}, {-1, 5}} {
		if got := r.Render(Frame{W: s[0], H: s[1], Theme: theme.DefaultTheme()}); got != "" {
			t.Fatalf("%v: got %q, want empty", s, got)
		}
	}
}

func TestRenderSmallPaneShowsMessage(t *testing.T) {
	r := NewRenderer(vaultLayout(), vault().Edges) // small enough for the smallest pane
	for _, s := range [][2]int{{29, 8}, {30, 7}, {29, 30}, {120, 7}} {
		out := ansi.Strip(r.Render(vaultFrame(r, ZoomClose, s[0], s[1])))
		if !strings.Contains(out, "Graph needs a bigger pane") {
			t.Fatalf("%v: no small-pane message:\n%s", s, out)
		}
	}
	out := ansi.Strip(r.Render(vaultFrame(r, ZoomOverview, MinWidth, MinHeight)))
	if strings.Contains(out, "bigger pane") || !strings.ContainsRune(out, glyphSelected) {
		t.Fatalf("%dx%d is big enough for the map:\n%s", MinWidth, MinHeight, out)
	}
}

func TestRenderEmptyVault(t *testing.T) {
	r := NewRenderer(Build(Graph{}), nil)
	out := ansi.Strip(r.Render(Frame{W: 40, H: 10, Theme: theme.DefaultTheme()}))
	if !strings.Contains(out, "No notes yet") {
		t.Fatalf("empty vault:\n%s", out)
	}
}

func TestRenderSingleNoteIsCentred(t *testing.T) {
	g := Graph{Nodes: []Node{{ID: "Only.md"}}}
	r := NewRenderer(Build(g), nil)
	for z := range zoomLevels {
		f := Frame{W: 41, H: 11, Zoom: z, Offset: r.Centred(z, 41, 11), Theme: theme.DefaultTheme()}
		rows := screen(r.Render(f))
		if rows[5][20].r != glyphLeaf {
			t.Fatalf("zoom %d: middle cell is %q, want the dot:\n%s", z, rows[5][20].r, ansi.Strip(r.Render(f)))
		}
	}
}

func TestRenderNoLinksShowsEveryNoteAsOrphan(t *testing.T) {
	g := vault()
	g.Edges = nil
	r := NewRenderer(Build(g), nil)
	f := vaultFrame(r, ZoomOverview, 120, 30) // the whole map in the pane
	rows := screen(r.Render(f))
	for _, n := range r.l.Nodes {
		x, y := dotAt(t, r, f, n.ID)
		want := glyphLeaf
		switch {
		case n.Unresolved:
			want = glyphUnresolved
		case n.ID == f.Selected:
			want = glyphSelected
		}
		if got := rows[y][x].r; got != want {
			t.Errorf("%s drawn as %q, want %q", n.ID, got, want)
		}
	}
	for _, row := range rows {
		for _, c := range row {
			if strings.ContainsRune("─│╱╲┼╳", c.r) {
				t.Fatalf("an edge is drawn in a vault with no links:\n%s", ansi.Strip(r.Render(f)))
			}
		}
	}
}

func TestRenderNodeGlyphsByDegree(t *testing.T) {
	g := Graph{Nodes: []Node{{ID: "Hub.md"}, {ID: "Mid.md"}, {ID: "Leaf.md"}, {ID: "Gone", Unresolved: true}}}
	g.Edges = append(g.Edges, Edge{"Mid.md", "Leaf.md"}, Edge{"Mid.md", "Gone"})
	for i := range 10 {
		id := "spoke" + strconv.Itoa(i) + ".md"
		g.Nodes = append(g.Nodes, Node{ID: id})
		g.Edges = append(g.Edges, Edge{"Hub.md", id})
	}
	r := NewRenderer(Build(g), g.Edges)
	f := Frame{W: 120, H: 30, Zoom: ZoomClose, Offset: r.Centred(ZoomClose, 120, 30), Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	for id, want := range map[string]rune{"Hub.md": glyphHub, "Mid.md": glyphLinked, "Leaf.md": glyphLeaf, "Gone": glyphUnresolved} {
		x, y := dotAt(t, r, f, id)
		if got := rows[y][x].r; got != want {
			t.Errorf("%s drawn as %q, want %q", id, got, want)
		}
	}
}

func TestRenderHighlightsSelectionNeighboursAndOpenNote(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	r := NewRenderer(Build(vault()), vault().Edges)
	f := vaultFrame(r, ZoomClose, 160, 40)
	rows := screen(r.Render(f))
	fgAt := func(id string) string {
		x, y := dotAt(t, r, f, id)
		return rows[y][x].fg
	}
	cases := map[string]lipgloss.Color{
		"work/Plan.md":    th.Yellow, // selected
		"Home.md":         th.Green,  // open (and a backlink: open wins)
		"work/Tasks.md":   th.Blue,   // link of the selection
		"work/Meeting.md": th.Purple, // backlink of the selection
		"study/Go.md":     th.Purple,
		"work/Notes.md":   th.GreyFg, // two steps away
		"study/Rust.md":   th.GreyFg,
		"Ideas":           th.GreyFg,
	}
	for id, want := range cases {
		if got := fgAt(id); got != rgb(want) {
			t.Errorf("%s dot is %s, want %s", id, got, rgb(want))
		}
	}
	x, y := dotAt(t, r, f, "work/Plan.md")
	for dx := range 2 + len("Plan") {
		if rows[y][x+dx].bg != rgb(th.OneBg) {
			t.Fatalf("focused selection cell %d has background %q, want OneBg", dx, rows[y][x+dx].bg)
		}
	}
	if rows[y][x+2].r != 'P' || rows[y][x+2].fg != rgb(th.Fg) {
		t.Fatalf("selection name starts with %q in %s, want P in Fg", rows[y][x+2].r, rows[y][x+2].fg)
	}
	f.Focused = false
	rows = screen(r.Render(f))
	if rows[y][x].bg != "" || rows[y][x].fg != rgb(th.Yellow) {
		t.Fatalf("unfocused selection: fg %s bg %q, want Yellow and no background", rows[y][x].fg, rows[y][x].bg)
	}
}

// With nothing selected each project's dots take a theme accent in turn,
// by sorted folder name; unresolved targets stay grey.
func TestRenderColoursProjectsWhenNothingIsSelected(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	r := NewRenderer(Build(vault()), vault().Edges)
	f := vaultFrame(r, ZoomClose, 160, 40)
	f.Selected, f.Open = "", ""
	rows := screen(r.Render(f))
	for id, want := range map[string]lipgloss.Color{"Home.md": th.Blue, "study/Go.md": th.Purple, "work/Plan.md": th.Teal, "Ideas": th.GreyFg} {
		x, y := dotAt(t, r, f, id)
		if got := rows[y][x].fg; got != rgb(want) {
			t.Errorf("%s dot is %s, want %s", id, got, rgb(want))
		}
	}
}

func TestRenderOverviewLabelsOnlyTheNeighbourhood(t *testing.T) {
	r := NewRenderer(Build(vault()), vault().Edges)
	out := ansi.Strip(r.Render(vaultFrame(r, ZoomOverview, 120, 30)))
	for _, name := range []string{"Plan", "Home"} {
		if !strings.Contains(out, name) {
			t.Errorf("overview misses the name %q of the selection or the open note:\n%s", name, out)
		}
	}
	if !strings.Contains(out, "Meeting") && !strings.Contains(out, "Tasks") && !strings.Contains(out, "Go") {
		t.Errorf("overview names none of the neighbours:\n%s", out)
	}
	for _, name := range []string{"Rust", "Orphan", "Ideas", "work/Notes"} {
		if strings.Contains(out, name) {
			t.Errorf("overview labels %q, outside the neighbourhood:\n%s", name, out)
		}
	}
	f := vaultFrame(r, ZoomOverview, 120, 30)
	f.Selected, f.Open = "", ""
	if out := ansi.Strip(r.Render(f)); strings.ContainsAny(out, "abcdefghijklmnopqrstuvwxyz") {
		t.Errorf("overview with no selection draws names:\n%s", out)
	}
}

func TestRenderCloseZoomLabelsFromTheTable(t *testing.T) {
	r := NewRenderer(Build(vault()), vault().Edges)
	f := vaultFrame(r, ZoomClose, 200, 40)
	f.Selected, f.Open = "", ""
	out := ansi.Strip(r.Render(f))
	for _, lb := range r.l.Labels[ZoomClose] {
		n, _ := r.l.Lookup(lb.ID)
		if !strings.Contains(out, n.Label) {
			t.Errorf("close zoom misses %q:\n%s", n.Label, out)
		}
	}
}

// Names and dots follow selection > open note > neighbours > the rest;
// edges run under every dot and name, and a lit edge beats a faint one.
func TestRenderSameCellColourPriority(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	nodes := []Placed{
		{Node: Node{ID: "Alpha.md"}, P: Point{1, 0}},
		{Node: Node{ID: "Beta.md"}, P: Point{4, 0}},
		{Node: Node{ID: "Gamma.md"}, P: Point{6, 0}},
		{Node: Node{ID: "Delta.md"}, P: Point{1, 2}},
		{Node: Node{ID: "G.md"}, P: Point{0, 1}},
		{Node: Node{ID: "H.md"}, P: Point{2, 1}},
		{Node: Node{ID: "E.md"}, P: Point{8, 2}},
		{Node: Node{ID: "F.md"}, P: Point{8, 4}},
		{Node: Node{ID: "Far.md"}, P: Point{40, 12}}, // keeps the overview at 1×1 in a 60×12 pane
	}
	// Alpha→Gamma runs over Beta; Delta→Alpha crosses the faint G-H.
	edges := []Edge{{"Alpha.md", "Gamma.md"}, {"Delta.md", "Alpha.md"}, {"G.md", "H.md"}, {"E.md", "F.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	f := Frame{W: 60, H: 12, Zoom: ZoomMedium, Selected: "Alpha.md", Open: "Alpha.md", Theme: th}
	rows := screen(r.Render(f))
	plain := ansi.Strip(r.Render(f))
	check := func(what string, x, y int, wantR rune, want lipgloss.Color) {
		t.Helper()
		if c := rows[y][x]; c.r != wantR || c.fg != rgb(want) {
			t.Errorf("%s at %d,%d is %q %s, want %q %s:\n%s", what, x, y, c.r, c.fg, wantR, rgb(want), plain)
		}
	}
	check("selected open note", 3, 0, glyphSelected, th.Yellow)
	check("lit link", 11, 0, '─', th.Blue)
	check("dot under the lit link", 12, 0, glyphLeaf, th.GreyFg)
	check("lit backlink", 3, 3, '│', th.Purple)
	check("lit backlink over a faint edge", 3, 2, '│', th.Purple)
	check("faint edge", 1, 2, '─', th.Line)
	check("faint edge", 24, 6, '│', th.Line)

	// At the overview the neighbour Alpha's name would cover the selected
	// Gamma: the selection's dot and name win, Alpha's name is left out.
	f = Frame{W: 60, H: 12, Zoom: ZoomOverview, Selected: "Gamma.md", Open: "Delta.md", Theme: th}
	rows = screen(r.Render(f))
	plain = ansi.Strip(r.Render(f))
	check("selected", 6, 0, glyphSelected, th.Yellow)
	check("selection name", 8, 0, 'G', th.Fg)
	check("backlink of the selection", 1, 0, glyphLinked, th.Purple)
	check("open note", 1, 2, glyphLeaf, th.Green)
	if strings.Contains(plain, "Alpha") {
		t.Errorf("the neighbour name overlapping the selection is drawn:\n%s", plain)
	}
	if !strings.Contains(plain, "Delta") {
		t.Errorf("the open note has no name:\n%s", plain)
	}
}

// A name of the layout's table that the selection's name would partly
// cover is left out whole, never shown in pieces.
func TestRenderNamesAreNeverShownInPieces(t *testing.T) {
	nodes := []Placed{
		{Node: Node{ID: "Selection.md"}, P: Point{0, 0}},
		{Node: Node{ID: "Xenophon.md"}, P: Point{2, 0}},
		{Node: Node{ID: "Y.md"}, P: Point{0, 3}},
		{Node: Node{ID: "Z.md"}, P: Point{8, 3}},
	}
	edges := []Edge{{"Selection.md", "Y.md"}, {"Xenophon.md", "Z.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	f := Frame{W: 60, H: 10, Zoom: ZoomMedium, Theme: theme.DefaultTheme()}
	if out := ansi.Strip(r.Render(f)); !strings.Contains(out, "Xenophon") {
		t.Fatalf("with nothing selected Xenophon is named:\n%s", out)
	}
	// The selection's name, which would hide Xenophon's dot, is cut before
	// it (there is no room on its left), and Xenophon is named whole.
	f.Selected = "Selection.md"
	out := ansi.Strip(r.Render(f))
	if !strings.Contains(out, "Se… · Xenophon") {
		t.Fatalf("Xenophon's name shows around the selection's:\n%s", out)
	}
}

// Closer in, a neighbour's name is cut before the next dot; at the
// overview, where dots are packed, it covers the dots of the rest.
func TestRenderNeighbourNamesAndDots(t *testing.T) {
	nodes := []Placed{
		{Node: Node{ID: "Neighbourhood.md"}, P: Point{0, 0}},
		{Node: Node{ID: "R.md"}, P: Point{3, 0}},
		{Node: Node{ID: "S.md"}, P: Point{0, 2}},
	}
	edges := []Edge{{"S.md", "Neighbourhood.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	f := Frame{W: 60, H: 10, Zoom: ZoomMedium, Selected: "S.md", Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	plain := ansi.Strip(r.Render(f))
	if rows[0][9].r != glyphLeaf || !strings.Contains(plain, "Neigh…") {
		t.Errorf("medium: the neighbour's name is not cut before R:\n%s", plain)
	}
	f.Zoom = ZoomOverview
	if plain := ansi.Strip(r.Render(f)); !strings.Contains(plain, "Neighbourhood") {
		t.Errorf("overview: the neighbour's name does not cover the other dots:\n%s", plain)
	}
}

// rowText is the runes of row y from x0 to x1 (inclusive), with the cells
// of dots replaced by '*' and the right halves of wide characters skipped.
func rowText(rows [][]tcell, y, x0, x1 int, dots ...int) string {
	var b strings.Builder
	for x := x0; x <= x1; x++ {
		switch {
		case slices.Contains(dots, x):
			b.WriteRune('*')
		case rows[y][x].r != 0:
			b.WriteRune(rows[y][x].r)
		}
	}
	return b.String()
}

// A neighbour right next to the selection, whose name would be cut short
// on the right, is named in full on the left of its dot.
func TestRenderNeighbourNameGoesLeftWhenTheRightIsCut(t *testing.T) {
	nodes := []Placed{
		{Node: Node{ID: "Llamas.md"}, P: Point{4, 0}},
		{Node: Node{ID: "Nube.md"}, P: Point{5, 0}},
	}
	edges := []Edge{{"Llamas.md", "Nube.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	f := Frame{W: 60, H: 10, Zoom: ZoomClose, Selected: "Nube.md", Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	plain := ansi.Strip(r.Render(f))
	// Llamas' dot is at 24, Nube's at 30.
	if got, want := rowText(rows, 0, 16, 36, 24, 30), " Llamas *─────* Nube "; got != want {
		t.Errorf("row 0 from 16 is %q, want %q:\n%s", got, want, plain)
	}
	if strings.Contains(plain, "…") {
		t.Errorf("a name is cut:\n%s", plain)
	}
}

// A left-side name goes there only when it is wholly inside the pane: at
// the pane's left edge the neighbour keeps its visible right-hand cut.
func TestRenderLeftNameStaysInsideThePane(t *testing.T) {
	nodes := []Placed{
		{Node: Node{ID: "Llamas.md"}, P: Point{4, 0}},
		{Node: Node{ID: "Nube.md"}, P: Point{5, 0}},
	}
	edges := []Edge{{"Llamas.md", "Nube.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	// Llamas' dot is at map x 24; its left blank, name and blank need 8
	// cells, so the pane must start at map x 16 or before.
	f := Frame{W: 60, H: 10, Zoom: ZoomClose, Offset: Point{16, 0}, Selected: "Nube.md", Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	if got, want := rowText(rows, 0, 0, 20, 8, 14), " Llamas *─────* Nube "; got != want {
		t.Errorf("offset 16: row 0 is %q, want %q:\n%s", got, want, ansi.Strip(r.Render(f)))
	}
	f.Offset.X = 17
	rows = screen(r.Render(f))
	if got, want := rowText(rows, 0, 0, 19, 7, 13), "       * Ll… * Nube "; got != want {
		t.Errorf("offset 17: row 0 is %q, want %q:\n%s", got, want, ansi.Strip(r.Render(f)))
	}
}

// Names placed on the left keep a blank on each side and never cover a
// dot, another name or its blanks, wide names included; a neighbour whose
// right-hand gap is taken (here by the selection's dot) is named on the
// left too.
func TestRenderLeftNamesNeverOverlap(t *testing.T) {
	nodes := []Placed{
		{Node: Node{ID: "Ab.md"}, P: Point{22, 0}},
		{Node: Node{ID: "日本.md"}, P: Point{29, 0}},
		{Node: Node{ID: "S.md"}, P: Point{30, 0}},
		{Node: Node{ID: "Cd.md"}, P: Point{40, 0}},
		{Node: Node{ID: "Far.md"}, P: Point{59, 11}}, // keeps the overview at 1×1 in a 60×12 pane
	}
	// 日本 is the busiest neighbour, so it is placed before Ab.
	edges := []Edge{{"Ab.md", "S.md"}, {"日本.md", "S.md"}, {"S.md", "Cd.md"}, {"日本.md", "Far.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	f := Frame{W: 60, H: 12, Zoom: ZoomOverview, Selected: "S.md", Theme: theme.DefaultTheme()}
	if sx, sy := r.scale(f.Zoom, f.W, f.H); sx != 1 || sy != 1 {
		t.Fatalf("setup: overview scale %dx%d, want 1x1", sx, sy)
	}
	rows := screen(r.Render(f))
	plain := ansi.Strip(r.Render(f))
	if got, want := rowText(rows, 0, 17, 45, 22, 29, 30, 40), "  Ab * 日本 ** S ──────* Cd  "; got != want {
		t.Errorf("row 0 from 17 is %q, want %q:\n%s", got, want, plain)
	}
	// Two cells further left, 日本's name fits on neither side (Ab's dot
	// is in the way on the left): it is left out, and Ab fits on the right.
	nodes[1].P = Point{27, 0}
	r = NewRenderer(placedLayout(nodes, edges), edges)
	rows = screen(r.Render(f))
	plain = ansi.Strip(r.Render(f))
	if got, want := rowText(rows, 0, 17, 45, 22, 27, 30, 40), "     * Ab *──* S ──────* Cd  "; got != want {
		t.Errorf("crowded row 0 from 17 is %q, want %q:\n%s", got, want, plain)
	}
	// Back in place, with another neighbour's dot where Ab's blank before
	// its name would go: Ab fits on neither side and is left out, and Ef
	// goes left of its own dot.
	nodes[1].P = Point{29, 0}
	nodes = append(nodes, Placed{Node: Node{ID: "Ef.md"}, P: Point{18, 0}})
	edges = append(edges, Edge{"Ef.md", "S.md"})
	r = NewRenderer(placedLayout(nodes, edges), edges)
	rows = screen(r.Render(f))
	plain = ansi.Strip(r.Render(f))
	if got, want := rowText(rows, 0, 13, 45, 18, 22, 29, 30, 40), "  Ef *───* 日本 ** S ──────* Cd  "; got != want {
		t.Errorf("row 0 from 13 with Ef is %q, want %q:\n%s", got, want, plain)
	}
}

// isDot reports whether cell c shows a node's dot.
func isDot(c tcell) bool {
	return strings.ContainsRune(string([]rune{glyphLeaf, glyphLinked, glyphHub, glyphUnresolved, glyphSelected}), c.r)
}

// The selection's name, which on the right would cover the dot of Tasks
// (its right-hand neighbour on the grid at 3×2), goes whole on the left.
func TestRenderSelectionNameGoesLeftOfACoveredDot(t *testing.T) {
	r := NewRenderer(vaultLayout(), vault().Edges)
	for _, z := range []Zoom{ZoomOverview, ZoomMedium} {
		f := vaultFrame(r, z, 100, 24)
		if sx, sy := r.scale(f.Zoom, f.W, f.H); sx != 3 || sy != 2 {
			t.Fatalf("setup: zoom %d scale %dx%d, want 3x2", z, sx, sy)
		}
		px, py := dotAt(t, r, f, "work/Plan.md")
		tx, ty := dotAt(t, r, f, "work/Tasks.md")
		if ty != py || tx != px+3 {
			t.Fatalf("setup: Tasks at (%d,%d), Plan at (%d,%d)", tx, ty, px, py)
		}
		rows := screen(r.Render(f))
		if got, want := rowText(rows, py, px-6, tx, px, tx), " Plan *──*"; got != want || !isDot(rows[ty][tx]) {
			t.Errorf("zoom %d: row %d is %q (Tasks' cell %q), want %q:\n%s", z, py, got, rows[ty][tx].r, want, ansi.Strip(r.Render(f)))
		}
		// A click on the left-side name selects the selection itself.
		if i, ok := r.nodeAt(f, px-3, py); !ok || r.l.Nodes[i].ID != "work/Plan.md" {
			t.Errorf("zoom %d: the name's cell belongs to %v %v", z, i, ok)
		}
	}
}

// When the left side is outside the pane, the selection's name is cut
// with … one blank before the first dot it would cover, or kept whole
// when that leaves fewer than minNearName cells. This is where the
// selection's name depends on the pane, like a neighbour's (see
// TestRenderLeftNameStaysInsideThePane): one column of panning switches it.
func TestRenderSelectionNameIsCutBeforeACoveredDot(t *testing.T) {
	r := NewRenderer(vaultLayout(), vault().Edges)
	for _, c := range []struct {
		zoom      Zoom
		sel, next string
		// The row from the pane's first column to the next dot with the
		// left side in the pane, and from the dot on one column further.
		left, right string
		nextShown   bool // whether the next dot is drawn one column further
	}{
		// Close: work/Notes is 6 cells right of Meeting, room for "Me…".
		{ZoomClose, "work/Meeting.md", "work/Notes.md", " Meeting *     *", "* Me… ", true},
		// Medium: Tasks is 3 cells right of Plan, too close to cut to 3.
		{ZoomMedium, "work/Plan.md", "work/Tasks.md", " Plan *──*", "* Plan ", false},
	} {
		f := vaultFrame(r, c.zoom, 100, 24)
		f.Selected = c.sel
		x, y := dotAt(t, r, f, c.sel)
		w := ansi.StringWidth(r.l.Nodes[r.l.index[c.sel]].Label)
		f.Offset.X += x - (2 + w) // the left blank at the pane's first column
		x -= x - (2 + w)
		nx, _ := dotAt(t, r, f, c.next)
		rows := screen(r.Render(f))
		if got := rowText(rows, y, 0, nx, x, nx); got != c.left {
			t.Errorf("%s with the left side in the pane: %q, want %q:\n%s", c.sel, got, c.left, ansi.Strip(r.Render(f)))
		}
		f.Offset.X++
		x, nx = x-1, nx-1
		rows = screen(r.Render(f))
		if got := rowText(rows, y, x, x+len([]rune(c.right))-1, x); got != c.right || isDot(rows[y][nx]) != c.nextShown {
			t.Errorf("%s one column further: %q (next dot shown: %v), want %q (%v):\n%s", c.sel, got, isDot(rows[y][nx]), c.right, c.nextShown, ansi.Strip(r.Render(f)))
		}
	}
}

// A pane shows the cells a bigger pane shows at the same place: what is
// drawn never depends on where the pane's edges are, except that a name
// goes left of its dot only inside the pane, so one column of panning at
// the left edge switches its side (TestRenderLeftNameStaysInsideThePane,
// TestRenderSelectionNameIsCutBeforeACoveredDot); the panes here leave
// the left edge clear of those names.
func TestRenderPanningOnlyMovesThePicture(t *testing.T) {
	trueColour(t)
	g := fixture(300, 5, 600, 9)
	g.Nodes = append(g.Nodes, vault().Nodes...)
	g.Edges = append(g.Edges, vault().Edges...)
	r := NewRenderer(Build(g), g.Edges)
	for z := range zoomLevels {
		w, h := 300, 80
		if z == ZoomOverview {
			// The overview's scale fits the map to the pane: the whole map
			// is drawn in the biggest pane with the small panes' scale.
			sx, sy := r.scale(z, 40, 10)
			if sx < maxFitX {
				w = min(w, (sx+1)*(r.l.W-1))
			}
			if sy < maxFitY {
				h = min(h, (sy+1)*(r.l.H-1))
			}
			if bx, by := r.scale(z, w, h); bx != sx || by != sy || w < 40 || h < 10 {
				t.Fatalf("no %dx%d pane draws the overview at %dx%d", w, h, sx, sy)
			}
		}
		big := Frame{W: w, H: h, Zoom: z, Offset: r.Centred(z, w, h), Selected: "work/Plan.md", Open: "Home.md", Focused: true, Theme: theme.DefaultTheme()}
		whole := screen(r.Render(big))
		clamp := func(x, y int) Point { return Point{min(max(x, 0), w-40), min(max(y, 0), h-10)} }
		spots := []Point{{0, 0}, clamp(37, 11), clamp(131, 40), clamp(260, 70), clamp(5, 60)}
		// Panes whose right edge cuts the names of the neighbourhood.
		for _, id := range []string{"work/Plan.md", "Home.md", "work/Tasks.md", "work/Meeting.md", "study/Go.md"} {
			x, y := dotAt(t, r, big, id)
			for _, dx := range []int{3, 5, 8} {
				spots = append(spots, clamp(x+dx-39, y-5))
			}
		}
		for _, at := range spots {
			f := big
			f.W, f.H = 40, 10
			f.Offset = Point{big.Offset.X + at.X, big.Offset.Y + at.Y}
			part := screen(r.Render(f))
			for y := range f.H {
				for x := 1; x < f.W-1; x++ {
					if a, b := part[y][x], whole[y+at.Y][x+at.X]; a.r != b.r || a.fg != b.fg {
						t.Fatalf("zoom %d at %v: cell %d,%d is %q %s, the whole map has %q %s", z, at, x, y, a.r, a.fg, b.r, b.fg)
					}
				}
			}
		}
	}
}

// placedLayout builds a layout with hand-placed nodes.
func placedLayout(nodes []Placed, edges []Edge) Layout {
	g := Graph{Edges: edges}
	for _, n := range nodes {
		g.Nodes = append(g.Nodes, n.Node)
	}
	norm, _ := normalize(g)
	pos := map[string]Point{}
	for _, n := range nodes {
		pos[n.ID] = n.P
	}
	w, h := 0, 0
	for i := range norm {
		norm[i].P = pos[norm[i].ID]
		w, h = max(w, norm[i].P.X+1), max(h, norm[i].P.Y+1)
	}
	return finish(norm, []Cluster{{Rect: Rect{0, 0, w, h}}})
}

func TestRenderEdgesUsePlainLineGlyphs(t *testing.T) {
	nodes := []Placed{
		{Node: Node{ID: "O.md"}, P: Point{0, 0}},
		{Node: Node{ID: "R.md"}, P: Point{3, 0}},
		{Node: Node{ID: "D.md"}, P: Point{0, 3}},
		{Node: Node{ID: "X.md"}, P: Point{2, 2}},
		{Node: Node{ID: "U.md"}, P: Point{6, 0}},
		{Node: Node{ID: "L.md"}, P: Point{4, 4}},
	}
	edges := []Edge{{"O.md", "R.md"}, {"O.md", "D.md"}, {"O.md", "X.md"}, {"L.md", "U.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	f := Frame{W: 60, H: 14, Zoom: ZoomMedium, Offset: Point{0, 0}, Selected: "O.md", Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	if c := rows[0][5].r; c != '─' {
		t.Errorf("horizontal link drawn with %q", c)
	}
	if c := rows[1][0].r; c != '│' {
		t.Errorf("vertical link drawn with %q", c)
	}
	diag := false
	for y := 1; y < 4; y++ {
		for x := 1; x < 6; x++ {
			diag = diag || rows[y][x].r == '╲'
		}
	}
	if !diag {
		t.Errorf("down-right link has no ╲:\n%s", ansi.Strip(r.Render(f)))
	}
	rising := false
	for y := 1; y < 8; y++ {
		for x := 12; x < 18; x++ {
			rising = rising || rows[y][x].r == '╱'
		}
	}
	if !rising {
		t.Errorf("up-right link has no ╱:\n%s", ansi.Strip(r.Render(f)))
	}
}

func TestRenderDimmedGreysAllText(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	r := NewRenderer(Build(vault()), vault().Edges)
	for z := range zoomLevels {
		f := vaultFrame(r, z, 120, 30)
		lit := r.Render(f)
		f.Dimmed = true
		dim := r.Render(f)
		if ansi.Strip(dim) != ansi.Strip(lit) {
			t.Fatalf("zoom %d: dimming changed the text", z)
		}
		litText := false
		litBg, dimBg := map[string]bool{}, map[string]bool{}
		for i, row := range screen(lit) {
			drow := screen(dim)[i]
			for j, c := range row {
				litText = litText || (c.fg != "" && c.fg != rgb(th.Grey) && c.fg != rgb(th.Line))
				litBg[c.bg] = true
				d := drow[j]
				dimBg[d.bg] = true
				if d.fg != "" && d.fg != rgb(th.Grey) && d.fg != rgb(th.Line) {
					t.Fatalf("zoom %d: dimmed cell %q at %d,%d drawn in %s", z, d.r, j, i, d.fg)
				}
			}
		}
		if !litText {
			t.Fatalf("zoom %d: the lit rendering is already grey", z)
		}
		if !reflect.DeepEqual(litBg, dimBg) {
			t.Fatalf("zoom %d: dimming changed the backgrounds: %v vs %v", z, dimBg, litBg)
		}
	}
}

func TestRenderFollowsTheTheme(t *testing.T) {
	trueColour(t)
	r := NewRenderer(Build(vault()), vault().Edges)
	for _, name := range theme.Names() {
		th, _ := theme.Get(name)
		f := vaultFrame(r, ZoomClose, 120, 30)
		f.Theme = th
		rows := screen(r.Render(f))
		x, y := dotAt(t, r, f, "work/Plan.md")
		if rows[y][x].fg != rgb(th.Yellow) {
			t.Fatalf("theme %s: selection drawn in %s, want its Yellow %s", name, rows[y][x].fg, rgb(th.Yellow))
		}
	}
}

// A wide character cut by either edge of the pane, or by the name's width,
// becomes blank cells; it is never split.
func TestRenderWideNamesAreNeverSplit(t *testing.T) {
	g := Graph{Nodes: []Node{{ID: "日本語のノートのとても長い名前ですね本当に.md"}, {ID: "x.md"}}, Edges: []Edge{{"x.md", "日本語のノートのとても長い名前ですね本当に.md"}}}
	// Hand-placed side by side: a map this small takes the overview's
	// biggest scale in every pane the test uses.
	r := NewRenderer(placedLayout([]Placed{{Node: g.Nodes[1], P: Point{1, 0}}, {Node: g.Nodes[0], P: Point{2, 0}}}, g.Edges), g.Edges)
	n, _ := r.l.Lookup(g.Nodes[0].ID)
	for z := range zoomLevels {
		for dx := -40; dx <= 4; dx++ {
			for _, w := range []int{30, 31, 33, 40} {
				sx, sy := r.scale(z, w, 8)
				if wx, wy := r.scale(z, w+20, 8); wx != sx || wy != sy {
					t.Fatalf("zoom %d: the wider pane below uses another scale", z)
				}
				f := Frame{W: w, H: 8, Zoom: z, Offset: Point{n.P.X*sx + 2 + dx, n.P.Y*sy - 2}, Selected: n.ID, Theme: theme.DefaultTheme()}
				out := r.Render(f)
				checkSize(t, out, w, 8, "wide")
				if strings.ContainsRune(ansi.Strip(out), '\uFFFD') {
					t.Fatalf("broken character at offset %d", dx)
				}
				// Every cell but the two at the edges, where a character
				// may be cut, is what a wider pane shows there.
				wf := f
				wf.W, wf.Offset.X = w+20, f.Offset.X-10
				small, wide := screen(out), screen(r.Render(wf))
				for y := range small {
					for x := 1; x < w-1; x++ {
						if small[y][x].r != wide[y][x+10].r {
							t.Fatalf("zoom %d offset %d width %d: cell %d,%d is %q, a wider pane has %q:\n%s\n%s", z, dx, w, x, y, small[y][x].r, wide[y][x+10].r, ansi.Strip(out), ansi.Strip(r.Render(wf)))
						}
					}
				}
				f.Offset.X = n.P.X*sx + 2 + dx - w + 1
				checkSize(t, r.Render(f), w, 8, "wide, right edge")
			}
		}
	}
	// The name is cut to whole characters within the label width.
	f := Frame{W: 60, H: 8, Zoom: ZoomClose, Offset: Point{n.P.X*6 - 1, n.P.Y*2 - 2}, Selected: n.ID, Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	line := ""
	for _, c := range rows[2] {
		if c.r != 0 {
			line += string(c.r)
		}
	}
	if !strings.Contains(line, "日本語のノートのとても…") {
		t.Fatalf("name not cut to whole characters: %q", line)
	}
}

// The cached edges are not changed by a frame: moving the selection away
// and back draws the same picture.
func TestRenderCacheDoesNotLeakBetweenFrames(t *testing.T) {
	trueColour(t)
	r := NewRenderer(Build(vault()), vault().Edges)
	for z := range zoomLevels {
		f := vaultFrame(r, z, 120, 30)
		first := r.Render(f)
		for _, sel := range []string{"study/Go.md", "Ideas", "", "Home.md"} {
			g := f
			g.Selected = sel
			r.Render(g)
		}
		if again := r.Render(f); again != first {
			t.Fatalf("zoom %d: the same frame drew differently after other frames", z)
		}
		if fresh := NewRenderer(Build(vault()), vault().Edges).Render(f); fresh != first {
			t.Fatalf("zoom %d: a cached renderer drew differently from a fresh one", z)
		}
	}
}

var goldenZooms = map[Zoom]string{ZoomOverview: "overview", ZoomMedium: "medium", ZoomClose: "close"}

// The fixed vault at each zoom, without colours. Refresh with
// UPDATE_GLOBALGRAPH_GOLDEN=1 go test ./internal/globalgraph/.
func TestRenderGolden(t *testing.T) {
	r := NewRenderer(Build(vault()), vault().Edges)
	for z, name := range goldenZooms {
		got := ansi.Strip(r.Render(vaultFrame(r, z, 100, 24))) + "\n"
		path := filepath.Join("testdata", "render-"+name+".golden")
		if os.Getenv("UPDATE_GLOBALGRAPH_GOLDEN") == "1" {
			if err := os.MkdirAll("testdata", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("%s differs from %s:\n%s", name, path, got)
		}
	}
}

// BenchmarkView5k budget: 16 ms/op for one frame of a 5k-note map on a
// 200×60 pane, after the edge layer of the zoom is cached.
func BenchmarkView5k(b *testing.B) {
	trueColour(b)
	g := fixture(5000, 20, 50000, 10)
	r := NewRenderer(Build(g), g.Edges)
	for _, z := range []Zoom{ZoomOverview, ZoomMedium, ZoomClose} {
		b.Run(goldenZooms[z], func(b *testing.B) {
			f := Frame{W: 200, H: 60, Zoom: z, Offset: r.Centred(z, 200, 60), Selected: r.l.Nodes[2500].ID, Open: r.l.Nodes[10].ID, Focused: true, Theme: theme.DefaultTheme()}
			r.Render(f)
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				f.Selected = r.l.Nodes[i%len(r.l.Nodes)].ID
				i += 97
				r.Render(f)
			}
		})
	}
	// The first frame at close zoom also rasterises every edge once.
	b.Run("close-first-frame", func(b *testing.B) {
		f := Frame{W: 200, H: 60, Zoom: ZoomClose, Offset: r.Centred(ZoomClose, 200, 60), Theme: theme.DefaultTheme()}
		for b.Loop() {
			NewRenderer(r.l, g.Edges).Render(f)
		}
	})
}

// At the overview a small map is spread over the pane, so the notes and
// the lines between them can be told apart, and the same layout in the
// same pane always gives the same picture.
func TestRenderOverviewSpreadsASmallMapOverThePane(t *testing.T) {
	const w, h = 48, 12
	draw := func() (string, *Renderer) {
		r := NewRenderer(vaultLayout(), vault().Edges) // a map much smaller than the pane
		return r.Render(Frame{W: w, H: h, Zoom: ZoomOverview, Offset: r.Centred(ZoomOverview, w, h), Theme: theme.DefaultTheme()}), r
	}
	out, r := draw()
	x0, y0, x1, y1 := w, h, -1, -1
	for y, row := range screen(out) {
		for x, c := range row {
			if strings.ContainsRune(string([]rune{glyphLeaf, glyphLinked, glyphHub, glyphUnresolved}), c.r) {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	if x1-x0+1 < w/2 || y1-y0+1 < h/2 {
		t.Fatalf("the %dx%d grid spans %dx%d cells of a %dx%d pane, want at least half:\n%s", r.l.W, r.l.H, x1-x0+1, y1-y0+1, w, h, ansi.Strip(out))
	}
	if again, _ := draw(); again != out {
		t.Fatal("the same layout in the same pane drew differently")
	}
}
