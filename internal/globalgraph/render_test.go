package globalgraph

import (
	"fmt"
	"math"
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

// realisticVault is shaped like a real vault: 110 notes in twelve folders
// of 1 to 15 notes and eleven at the root, long names, and only 5 links,
// so almost every note is an orphan.
func realisticVault() Graph {
	folders := []struct {
		key string
		n   int
	}{
		{"zentro-tickets", 15}, {"misoku", 14}, {"stow-portfolio", 12}, {"gastos", 11},
		{"lecturas", 10}, {"recetas", 9}, {"viajes", 8}, {"trabajo-2026", 7},
		{"ideas", 6}, {"diario", 4}, {"cursos-online", 2}, {"finanzas-personales", 1},
	}
	names := []string{
		"2026-10-25-rom-galaxy-notes", "1179-02 Detalle de campaña poco entendible", "Plan",
		"reunion semanal", "2026-09-14 retro", "pendientes", "Ideas sueltas para el portfolio",
		"lista", "README", "notas de la llamada con el cliente",
	}
	g := Graph{Name: "notas"}
	add := func(dir string, i int) string {
		k := len(g.Nodes)
		id := fmt.Sprintf("%s %03d.md", names[k%len(names)], k)
		if dir != "" {
			id = dir + "/" + id
		}
		g.Nodes = append(g.Nodes, Node{ID: id, Cluster: dir})
		return id
	}
	for i := range 11 {
		add("", i)
	}
	for _, f := range folders {
		for i := range f.n {
			add(f.key, i)
		}
	}
	id := func(k int) string { return g.Nodes[k].ID }
	g.Edges = []Edge{
		{id(0), id(11 + 15)}, // root → misoku
		{id(11), id(12)},     // inside zentro-tickets
		{id(11), id(13)},
		{id(26), id(40)}, // misoku → stow-portfolio
		{id(1), id(2)},   // root → root
	}
	return g
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
	k := r.scale(f.Zoom, f.W, f.H)
	return k.of(n.P.X) - f.Offset.X, k.of(n.P.Y) - f.Offset.Y
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

// S21 with S7: a single note orbits the vault's sun, which sits in the
// middle of the pane (give or take a cell of rounding), the note's dot on
// the innermost ring that holds notes, at every zoom.
func TestRenderSingleNoteIsCentred(t *testing.T) {
	g := Graph{Nodes: []Node{{ID: "Only.md"}}}
	r := NewRenderer(Build(g), nil)
	for z := range zoomLevels {
		w, h := 81, 21
		f := Frame{W: w, H: h, Zoom: z, Offset: r.Centred(z, w, h), Theme: theme.DefaultTheme()}
		rows := screen(r.Render(f))
		sx, sy := at(r, f, r.l.Sun)
		x, y := dotAt(t, r, f, "Only.md")
		k := r.scale(z, w, h)
		near := k.of(ringRadius(rootFirstRing)) + 1
		if abs(sx-w/2) > 1 || abs(sy-h/2) > 1 || rows[sy][sx].r != glyphVaultSun || rows[y][x].r != glyphLinked || abs(x-sx) > near || abs(y-sy) > near/cellAspect {
			t.Fatalf("zoom %d: the sun at (%d,%d) is %q, the note at (%d,%d) is %q; want the sun in the middle, the dot beside it:\n%s",
				z, sx, sy, rows[sy][sx].r, x, y, rows[y][x].r, ansi.Strip(r.Render(f)))
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
		want := glyphLinked // a solar map draws no note smaller
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
	f := Frame{W: 120, H: 30, Zoom: ZoomOverview, Offset: r.Centred(ZoomOverview, 120, 30), Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	// A solar map draws no note smaller than •; a map placed by hand draws
	// a note with one link or none as ·.
	for id, want := range map[string]rune{"Hub.md": glyphHub, "Mid.md": glyphLinked, "Leaf.md": glyphLinked, "Gone": glyphUnresolved} {
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
	f := vaultFrame(r, ZoomClose, 320, 100) // the whole map
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
		// Two steps away: the galaxy's colour, faded (a missing note too).
		"work/Notes.md": fade(th.Teal, th.Bg, restFade, th.GreyFg),
		"study/Rust.md": fade(th.Purple, th.Bg, restFade, th.GreyFg),
		"Ideas":         fade(th.Blue, th.Bg, restFade, th.GreyFg),
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
// by sorted folder name; unresolved targets take it faded.
func TestRenderColoursProjectsWhenNothingIsSelected(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	r := NewRenderer(Build(vault()), vault().Edges)
	f := vaultFrame(r, ZoomClose, 320, 100)
	f.Selected, f.Open = "", ""
	rows := screen(r.Render(f))
	for id, want := range map[string]lipgloss.Color{"Home.md": th.Blue, "study/Go.md": th.Purple, "work/Plan.md": th.Teal, "Ideas": fade(th.Blue, th.Bg, restFade, th.GreyFg)} {
		x, y := dotAt(t, r, f, id)
		if got := rows[y][x].fg; got != rgb(want) {
			t.Errorf("%s dot is %s, want %s", id, got, rgb(want))
		}
	}
}

// The overview names the selection, the open note, the neighbours, each
// galaxy and each galaxy's hub and top notes (see topNotes), and nothing
// else.
func TestRenderOverviewLabelsTheNeighbourhoodAndTheHubs(t *testing.T) {
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
	out = ansi.Strip(r.Render(f))
	allowed := []string{"work", "study", "Plan", "Go", "Home"}
	for _, top := range r.tops {
		for _, i := range top {
			allowed = append(allowed, strings.FieldsFunc(r.l.Nodes[i].Label, func(c rune) bool { return c < 'A' || c > 'z' })...)
		}
	}
	words := strings.FieldsFunc(out, func(c rune) bool { return c < 'A' || c > 'z' })
	for _, w := range words {
		if !slices.Contains(allowed, w) {
			t.Errorf("overview with no selection names %q, neither a galaxy nor a hub:\n%s", w, out)
		}
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
		c := rows[y][x]
		if (wantR == anyBraille && !isBraille(c.r)) || (wantR != anyBraille && c.r != wantR) || c.fg != rgb(want) {
			t.Errorf("%s at %d,%d is %q %s, want %q %s:\n%s", what, x, y, c.r, c.fg, wantR, rgb(want), plain)
		}
	}
	// Medium zoom: every grid cell is 2×2 screen cells.
	faded, edge := fade(th.Blue, th.Bg, restFade, th.GreyFg), fade(th.Blue, th.Bg, restFade, th.GreyFg)
	check("selected open note", 2, 0, glyphSelected, th.Yellow)
	check("lit link", 10, 0, anyBraille, th.Blue)
	check("dot under the lit link", 8, 0, glyphLeaf, faded)
	check("lit backlink", 2, 3, anyBraille, th.Purple)
	check("lit backlink over a faint edge", 2, 2, anyBraille, th.Purple)
	check("faint edge", 1, 2, anyBraille, edge)
	check("faint edge", 16, 6, anyBraille, edge)

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
	// The open note's name is cut before E's dot: no name covers a dot.
	if !strings.Contains(plain, "Del… ·") {
		t.Errorf("the open note's name is not cut before E's dot:\n%s", plain)
	}
}

// A name of the layout's table that the selection's name would partly
// cover is left out whole, never shown in pieces.
func TestRenderNamesAreNeverShownInPieces(t *testing.T) {
	nodes := []Placed{
		{Node: Node{ID: "Selection.md"}, P: Point{0, 0}},
		{Node: Node{ID: "Xenophon.md"}, P: Point{3, 0}},
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

// A neighbour's name is cut before the next dot, at the overview too: no
// name covers a dot.
func TestRenderNeighbourNamesNeverCoverDots(t *testing.T) {
	nodes := []Placed{
		{Node: Node{ID: "Neighbourhood.md"}, P: Point{0, 0}},
		{Node: Node{ID: "R.md"}, P: Point{3, 0}},
		{Node: Node{ID: "S.md"}, P: Point{0, 2}},
	}
	edges := []Edge{{"S.md", "Neighbourhood.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	for _, z := range []Zoom{ZoomOverview, ZoomMedium} {
		f := Frame{W: 60, H: 10, Zoom: z, Selected: "S.md", Theme: theme.DefaultTheme()}
		rows := screen(r.Render(f))
		plain := ansi.Strip(r.Render(f))
		if x, y := dotAt(t, r, f, "R.md"); rows[y][x].r != glyphLeaf || !strings.Contains(plain, "· Ne… ·") {
			t.Errorf("zoom %d: the neighbour's name is not cut before R:\n%s", z, plain)
		}
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
		{Node: Node{ID: "Nube.md"}, P: Point{6, 0}},
	}
	edges := []Edge{{"Llamas.md", "Nube.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	f := Frame{W: 60, H: 10, Zoom: ZoomClose, Selected: "Nube.md", Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	plain := ansi.Strip(r.Render(f))
	// Llamas' dot is at 16, Nube's at 24.
	if got, want := rowText(rows, 0, 8, 30, 16, 24), " Llamas *⠒⠒⠒⠒⠒⠒⠒* Nube "; got != want {
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
		{Node: Node{ID: "Nube.md"}, P: Point{6, 0}},
	}
	edges := []Edge{{"Llamas.md", "Nube.md"}}
	r := NewRenderer(placedLayout(nodes, edges), edges)
	// Llamas' dot is at map x 16; its left blank, name and blank need 8
	// cells, so the pane must start at map x 8 or before.
	f := Frame{W: 60, H: 10, Zoom: ZoomClose, Offset: Point{8, 0}, Selected: "Nube.md", Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	if got, want := rowText(rows, 0, 0, 22, 8, 16), " Llamas *⠒⠒⠒⠒⠒⠒⠒* Nube "; got != want {
		t.Errorf("offset 8: row 0 is %q, want %q:\n%s", got, want, ansi.Strip(r.Render(f)))
	}
	f.Offset.X = 9
	rows = screen(r.Render(f))
	if got, want := rowText(rows, 0, 0, 21, 7, 15), "       * Llam… * Nube "; got != want {
		t.Errorf("offset 9: row 0 is %q, want %q:\n%s", got, want, ansi.Strip(r.Render(f)))
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
	if k := r.scale(f.Zoom, f.W, f.H); k != whole(1) {
		t.Fatalf("setup: overview scale %d/%d, want 1", k, scaleUnit)
	}
	rows := screen(r.Render(f))
	plain := ansi.Strip(r.Render(f))
	if got, want := rowText(rows, 0, 17, 45, 22, 29, 30, 40), "  Ab * 日本 ** S ⠒⠒⠒⠒⠒⠒* Cd  "; got != want {
		t.Errorf("row 0 from 17 is %q, want %q:\n%s", got, want, plain)
	}
	// Two cells further left, 日本's name fits on neither side (Ab's dot
	// is in the way on the left): it is left out, and Ab fits on the right.
	nodes[1].P = Point{27, 0}
	r = NewRenderer(placedLayout(nodes, edges), edges)
	rows = screen(r.Render(f))
	plain = ansi.Strip(r.Render(f))
	if got, want := rowText(rows, 0, 17, 45, 22, 27, 30, 40), "     * Ab *⠒⠒* S ⠒⠒⠒⠒⠒⠒* Cd  "; got != want {
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
	if got, want := rowText(rows, 0, 13, 45, 18, 22, 29, 30, 40), "  Ef *⠒⠒⠒* 日本 ** S ⠒⠒⠒⠒⠒⠒* Cd  "; got != want {
		t.Errorf("row 0 from 13 with Ef is %q, want %q:\n%s", got, want, plain)
	}
}

// isDot reports whether cell c shows a node's dot.
func isDot(c tcell) bool {
	return strings.ContainsRune(string([]rune{glyphLeaf, glyphLinked, glyphHub, glyphUnresolved, glyphSelected}), c.r)
}

// The selection's name, which on the right would cover the dot of Tasks
// (its right-hand neighbour on the grid at 2×2), goes whole on the left.
func TestRenderSelectionNameGoesLeftOfACoveredDot(t *testing.T) {
	r := NewRenderer(vaultLayout(), vault().Edges)
	for _, z := range []Zoom{ZoomOverview, ZoomMedium} {
		f := vaultFrame(r, z, 100, 24)
		if k := r.scale(f.Zoom, f.W, f.H); k != whole(2) {
			t.Fatalf("setup: zoom %d scale %d/%d, want 2", z, k, scaleUnit)
		}
		px, py := dotAt(t, r, f, "work/Plan.md")
		tx, ty := dotAt(t, r, f, "work/Tasks.md")
		if ty != py || tx != px+2 {
			t.Fatalf("setup: Tasks at (%d,%d), Plan at (%d,%d)", tx, ty, px, py)
		}
		rows := screen(r.Render(f))
		if got, want := rowText(rows, py, px-6, tx, px, tx), " Plan *⠒*"; got != want || !isDot(rows[ty][tx]) {
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
	// Two grid cells apart, with no link between them.
	apart := []Placed{
		{Node: Node{ID: "work/Meeting.md", Cluster: "work"}, P: Point{1, 0}},
		{Node: Node{ID: "work/Notes.md", Cluster: "work"}, P: Point{3, 0}},
		{Node: Node{ID: "Far.md"}, P: Point{20, 10}},
	}
	for _, c := range []struct {
		r         *Renderer
		zoom      Zoom
		sel, next string
		// The row from the pane's first column to the next dot with the
		// left side in the pane, and from the dot on one column further.
		left, right string
		nextShown   bool // whether the next dot is drawn one column further
	}{
		// Close: work/Notes is 8 cells right of Meeting, room for "Meet…".
		{NewRenderer(placedLayout(apart, nil), nil), ZoomClose, "work/Meeting.md", "work/Notes.md", " Meeting *       *", "* Meet… ", true},
		// Medium: Tasks is 2 cells right of Plan, too close to cut to 3.
		{NewRenderer(vaultLayout(), vault().Edges), ZoomMedium, "work/Plan.md", "work/Tasks.md", " Plan *⠒*", "* Plan ", false},
	} {
		r := c.r
		f := vaultFrame(r, c.zoom, 100, 24)
		f.Selected = c.sel
		x, y := dotAt(t, r, f, c.sel)
		f.Offset.Y += y - 10 // the selection's row in the pane
		y = 10
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
// the left edge clear of those names (the spots that do not are skipped).
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
			k := r.scale(z, 40, 10)
			for r.scale(z, w, h) != k && (w > 40 || h > 10) {
				w, h = max(w-1, 40), max(h-1, 10)
			}
			if r.scale(z, w, h) != k {
				t.Fatalf("no %dx%d pane draws the overview at %d/%d", w, h, k, scaleUnit)
			}
		}
		big := Frame{W: w, H: h, Zoom: z, Offset: r.Centred(z, w, h), Selected: "work/Plan.md", Open: "Home.md", Focused: true, Theme: theme.DefaultTheme()}
		whole := screen(r.Render(big))
		// The names drawn left of their dots, as spans of the big pane from
		// the name's left blank to where it would end on the right: a small
		// pane whose left edge falls there may draw it on the right.
		var left [][3]int
		for i, n := range r.l.Nodes {
			x, y := dotAt(t, r, big, n.ID)
			if own, ok := r.nodeAt(big, x-2, y); ok && own == i {
				w := min(ansi.StringWidth(n.Label), maxLabelWidth)
				left = append(left, [3]int{x - 3 - w, x + 3 + w, y})
			}
		}
		cutsLeftName := func(at Point) bool {
			for _, s := range left {
				if at.X > s[0] && at.X <= s[1] && s[2] >= at.Y && s[2] < at.Y+10 {
					return true
				}
			}
			return false
		}
		clamp := func(x, y int) Point { return Point{min(max(x, 0), w-40), min(max(y, 0), h-10)} }
		spots := []Point{{0, 0}, clamp(37, 11), clamp(131, 40), clamp(260, 70), clamp(5, 60)}
		// Panes whose right edge cuts the names of the neighbourhood.
		for _, id := range []string{"work/Plan.md", "Home.md", "work/Tasks.md", "work/Meeting.md", "study/Go.md"} {
			x, y := dotAt(t, r, big, id)
			for _, dx := range []int{3, 5, 8} {
				spots = append(spots, clamp(x+dx-39, y-5))
			}
		}
		compared := 0
		for _, at := range spots {
			if cutsLeftName(at) {
				continue
			}
			compared++
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
		if compared < len(spots)/2 {
			t.Fatalf("zoom %d: only %d of %d panes compared", z, compared, len(spots))
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

// Edges run every way as braille: a straight run keeps to one row (or
// column) of braille dots; a slope goes down or up across its cells.
func TestRenderEdgesRunEveryWay(t *testing.T) {
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
	f := Frame{W: 60, H: 20, Zoom: ZoomClose, Offset: Point{0, 0}, Theme: theme.DefaultTheme()}
	rows := screen(r.Render(f))
	plain := ansi.Strip(r.Render(f))
	if c := rows[0][8].r; c != brailleBlank+0x12 {
		t.Errorf("horizontal link drawn with %q, want ⠒:\n%s", c, plain)
	}
	if c := rows[2][0].r; c != brailleBlank+0x47 {
		t.Errorf("vertical link drawn with %q, want ⡇:\n%s", c, plain)
	}
	// A slope's braille cells move right going down, or going up.
	slope := func(x0, x1, y0, y1 int, down bool) bool {
		top, bottom := Point{-1, -1}, Point{-1, -1}
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				if isBraille(rows[y][x].r) {
					if top.Y < 0 {
						top = Point{x, y}
					}
					bottom = Point{x, y}
				}
			}
		}
		if top.Y < 0 || top.Y == bottom.Y {
			return false
		}
		return (down && bottom.X > top.X) || (!down && top.X > bottom.X)
	}
	if !slope(1, 7, 1, 7, true) {
		t.Errorf("down-right link O-X has no falling braille:\n%s", plain)
	}
	if !slope(17, 23, 1, 15, false) {
		t.Errorf("up-right link L-U has no rising braille:\n%s", plain)
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
		x, y := dotAt(t, r, f, "work/Plan.md")
		f.Offset.X, f.Offset.Y = f.Offset.X+x-60, f.Offset.Y+y-15 // the selection in the middle
		x, y = 60, 15
		rows := screen(r.Render(f))
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
				k := r.scale(z, w, 8)
				if r.scale(z, w+20, 8) != k {
					t.Fatalf("zoom %d: the wider pane below uses another scale", z)
				}
				f := Frame{W: w, H: 8, Zoom: z, Offset: Point{k.of(n.P.X) + 2 + dx, k.of(n.P.Y) - 2}, Selected: n.ID, Theme: theme.DefaultTheme()}
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
				f.Offset.X = k.of(n.P.X) + 2 + dx - w + 1
				checkSize(t, r.Render(f), w, 8, "wide, right edge")
			}
		}
	}
	// The name is cut to whole characters within the label width.
	k, _ := ZoomClose.fixedScale()
	f := Frame{W: 60, H: 8, Zoom: ZoomClose, Offset: Point{n.P.X*k - 1, n.P.Y*k - 2}, Selected: n.ID, Theme: theme.DefaultTheme()}
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
	// The scale is the same across and down, so the map fills one way.
	if x1-x0+1 < w/2 && y1-y0+1 < h/2 {
		t.Fatalf("the %dx%d grid spans %dx%d cells of a %dx%d pane, want at least half one way:\n%s", r.l.W, r.l.H, x1-x0+1, y1-y0+1, w, h, ansi.Strip(out))
	}
	if again, _ := draw(); again != out {
		t.Fatal("the same layout in the same pane drew differently")
	}
}

// isBraille reports whether r is a braille pattern with at least one dot.
func isBraille(r rune) bool { return r > 0x2800 && r <= 0x28FF }

// anyBraille stands for any braille cell in a wanted rune.
const anyBraille rune = -1

// The map keeps galaxies round on screen: one grid cell spans as many
// screen columns as rows at every zoom (the layout's rings already are
// twice as wide as tall, like a terminal cell is twice as tall as wide).
func TestRenderScaleIsUniform(t *testing.T) {
	r := NewRenderer(Build(vault()), vault().Edges)
	big := NewRenderer(Build(fixture(1500, 10, 3000, 7)), nil)
	for _, rr := range []*Renderer{r, big} {
		for z := range zoomLevels {
			for _, p := range [][2]int{{40, 10}, {100, 24}, {300, 80}, {300, 12}} {
				if k := rr.scale(z, p[0], p[1]); k < whole(1) {
					t.Errorf("zoom %d pane %dx%d: scale %d/%d, want at least a cell per grid cell", z, p[0], p[1], k, scaleUnit)
				}
			}
		}
		for _, c := range rr.l.Clusters {
			if c.RX != 2*c.RY {
				t.Fatalf("galaxy %q is %dx%d, want twice as wide as tall", c.Key, c.RX, c.RY)
			}
		}
	}
	for z := ZoomOverview; z < ZoomClose; z++ {
		a := r.scale(z, 300, 80)
		b := r.scale(z+1, 300, 80)
		if b < a {
			t.Errorf("zooming in from %d draws the map smaller: %d → %d", z, a, b)
		}
	}
}

// Every note takes its galaxy's colour; with a selection, notes outside
// its neighbourhood keep that hue, faded toward the background, instead
// of all turning the same grey.
func TestRenderNodesKeepTheirGalaxyHue(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	r := NewRenderer(Build(vault()), vault().Edges)
	for z := range zoomLevels {
		f := vaultFrame(r, z, 200, 60)
		rows := screen(r.Render(f))
		fg := func(id string) string {
			x, y := dotAt(t, r, f, id)
			return rows[y][x].fg
		}
		rust, notes := fg("study/Rust.md"), fg("work/Notes.md")
		if rust == rgb(th.GreyFg) || notes == rgb(th.GreyFg) || rust == notes {
			t.Errorf("zoom %d: far notes are %s (study) and %s (work), want each its galaxy's faded hue", z, rust, notes)
		}
		if want := rgb(fade(th.Purple, th.Bg, restFade, th.GreyFg)); rust != want {
			t.Errorf("zoom %d: study/Rust is %s, want study's colour faded %s", z, rust, want)
		}
		if want := rgb(fade(th.Teal, th.Bg, restFade, th.GreyFg)); notes != want {
			t.Errorf("zoom %d: work/Notes is %s, want work's colour faded %s", z, notes, want)
		}
	}
}

func TestFadeBlendsHexColours(t *testing.T) {
	for _, c := range []struct {
		a, b lipgloss.Color
		t    float64
		want lipgloss.Color
	}{
		{"#ff0000", "#000000", 0.5, "#800000"},
		{"#ff8000", "#ffffff", 0, "#ff8000"},
		{"#102030", "#506070", 1, "#506070"},
		{"#FFFFFF", "#000000", 0.25, "#bfbfbf"},
		{"12", "#000000", 0.5, "#777777"}, // not hex: the fallback
		{"#ff0000", "", 0.5, "#777777"},
	} {
		if got := fade(c.a, c.b, c.t, "#777777"); got != c.want {
			t.Errorf("fade(%s, %s, %v) = %s, want %s", c.a, c.b, c.t, got, c.want)
		}
	}
}

// headingAt finds the galaxy heading of cluster c in rows: its row and
// first column, or ok false.
func headingAt(rows [][]tcell, name string) (x, y int, ok bool) {
	for y, row := range rows {
		var b strings.Builder
		for _, c := range row {
			if c.r == 0 {
				b.WriteRune('\uFFFF') // a wide character's right half: count every cell
			} else {
				b.WriteRune(c.r)
			}
		}
		if i := strings.Index(b.String(), " "+name+" "); i >= 0 {
			return len([]rune(b.String()[:i])) + 1, y, true
		}
	}
	return 0, 0, false
}

// Each named galaxy shows its folder's name and its number of notes over
// its sun, inside its box, the name in its colour and bold, on no dot; the
// notes at the vault root have no name.
func TestRenderGalaxiesAreNamed(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	r := NewRenderer(Build(vault()), vault().Edges)
	colours := map[string]lipgloss.Color{"study": th.Purple, "work": th.Teal}
	for z := range zoomLevels {
		for _, sel := range []string{"", "work/Plan.md"} {
			f := vaultFrame(r, z, 200, 60)
			f.Selected = sel
			out := r.Render(f)
			rows := screen(out)
			for _, c := range r.l.Clusters {
				if c.Key == "" {
					continue
				}
				x, y, ok := headingAt(rows, c.Key)
				if !ok {
					t.Fatalf("zoom %d sel %q: no heading %q:\n%s", z, sel, c.Key, ansi.Strip(out))
				}
				k := r.scale(z, f.W, f.H)
				top := k.of(c.Rect.Y) - f.Offset.Y
				cx := k.of(c.Centre.X) - f.Offset.X
				tail := fmt.Sprintf(" · %d", r.notes[slices.IndexFunc(r.l.Clusters, func(o Cluster) bool { return o.Key == c.Key })])
				if !strings.Contains(rowText(rows, y, x, f.W-1), c.Key+tail) {
					t.Errorf("zoom %d: heading %q has no count %q after it", z, c.Key, tail)
				}
				sun := k.of(c.Centre.Y) - f.Offset.Y
				if y >= sun || y < top-1 || x > cx || x+len(c.Key)+len([]rune(tail)) <= cx {
					t.Errorf("zoom %d: heading %q at %d,%d, want over its sun from row %d (or the one above) to %d, across column %d", z, c.Key, x, y, top, sun-1, cx)
				}
				for k := range len(c.Key) {
					if rows[y][x+k].fg != rgb(colours[c.Key]) {
						t.Errorf("zoom %d: heading %q letter %d in %s, want %s", z, c.Key, k, rows[y][x+k].fg, rgb(colours[c.Key]))
					}
				}
				if !strings.Contains(out, lipgloss.NewStyle().Bold(true).Foreground(colours[c.Key]).Render(c.Key)) {
					t.Errorf("zoom %d: heading %q is not bold", z, c.Key)
				}
			}
			for _, n := range r.l.Nodes {
				x, y := dotAt(t, r, f, n.ID)
				if x >= 0 && y >= 0 && x < f.W && y < f.H && !isDot(rows[y][x]) {
					t.Errorf("zoom %d sel %q: %s's dot is covered by %q:\n%s", z, sel, n.ID, rows[y][x].r, ansi.Strip(out))
				}
			}
		}
	}
}

// Edges are thin braille lines at every zoom, the overview included, from
// one dot to the other; dots and names are drawn over them.
func TestRenderEdgesAreBrailleAtEveryZoom(t *testing.T) {
	r := NewRenderer(Build(vault()), vault().Edges)
	for z := range zoomLevels {
		f := vaultFrame(r, z, 200, 60)
		f.Selected, f.Open = "", ""
		out := r.Render(f)
		rows := screen(out)
		braille := 0
		for _, row := range rows {
			for _, c := range row {
				if isBraille(c.r) {
					braille++
				}
				if strings.ContainsRune("─│╱╲┼╳", c.r) {
					t.Fatalf("zoom %d: box-drawing edge %q:\n%s", z, c.r, ansi.Strip(out))
				}
			}
		}
		if braille == 0 {
			t.Fatalf("zoom %d: no braille edge:\n%s", z, ansi.Strip(out))
		}
		for _, n := range r.l.Nodes {
			x, y := dotAt(t, r, f, n.ID)
			if !isDot(rows[y][x]) {
				t.Errorf("zoom %d: %s's dot shows %q", z, n.ID, rows[y][x].r)
			}
		}
	}
	// Hand-placed: each edge leaves its dot from the very next cell.
	nodes := []Placed{
		{Node: Node{ID: "O.md"}, P: Point{0, 0}},
		{Node: Node{ID: "R.md"}, P: Point{6, 0}},
		{Node: Node{ID: "D.md"}, P: Point{0, 6}},
		{Node: Node{ID: "X.md"}, P: Point{6, 6}},
	}
	edges := []Edge{{"O.md", "R.md"}, {"O.md", "D.md"}, {"O.md", "X.md"}}
	r = NewRenderer(placedLayout(nodes, edges), edges)
	for z := range zoomLevels {
		f := Frame{W: 60, H: 30, Zoom: z, Theme: theme.DefaultTheme()}
		rows := screen(r.Render(f))
		k := r.scale(z, f.W, f.H).of(1) // a whole scale: the map is small
		plain := ansi.Strip(r.Render(f))
		for _, c := range []struct {
			what string
			x, y int
		}{
			{"left of R", 6*k - 1, 0}, // right of O is O's name
			{"below O", 0, 1}, {"above D", 0, 6*k - 1},
			{"down-right of O", 1, 1}, {"up-left of X", 6*k - 1, 6*k - 1},
			{"middle of O-R", 3 * k, 0}, {"middle of O-D", 0, 3 * k}, {"middle of O-X", 3 * k, 3 * k},
		} {
			if !isBraille(rows[c.y][c.x].r) {
				t.Errorf("zoom %d: %s (%d,%d) is %q, want a braille edge:\n%s", z, c.what, c.x, c.y, rows[c.y][c.x].r, plain)
			}
		}
		// A straight line uses one row (or column) of braille dots.
		h, v := rows[0][3*k].r-0x2800, rows[3*k][0].r-0x2800
		if h != 0x12 && h != 0x24 && h != 0x09 && h != 0xC0 {
			t.Errorf("zoom %d: horizontal edge cell is %q, want one row of dots", z, rows[0][3*k].r)
		}
		if v != 0x47 && v != 0xB8 {
			t.Errorf("zoom %d: vertical edge cell is %q, want one column of dots", z, rows[3*k][0].r)
		}
	}
}

// S7: real links stand out over the frame: inside a galaxy in its full
// colour, between galaxies in light grey; the selection's links stay lit
// over them.
func TestRenderEdgeColours(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	r := NewRenderer(Build(vault()), vault().Edges)
	f := vaultFrame(r, ZoomClose, 300, 80)
	f.Selected, f.Open = "", ""
	rows := screen(r.Render(f))
	// coloured reports whether a braille cell along the link from a to b
	// (where no other edge crosses it) is drawn in want.
	coloured := func(a, b string, want lipgloss.Color) bool {
		for _, p := range linkPath(r, f, a, b) {
			if c := rows[p.Y][p.X]; isBraille(c.r) && c.fg == rgb(want) {
				return true
			}
		}
		return false
	}
	plain := ansi.Strip(r.Render(f))
	if !coloured("study/Rust.md", "study/日本語のノート.md", th.Purple) {
		t.Errorf("the link inside study is not in study's full colour:\n%s", plain)
	}
	if !coloured("study/Go.md", "work/Plan.md", th.LightGrey) {
		t.Errorf("the link between galaxies is not LightGrey:\n%s", plain)
	}
	f.Selected = "work/Plan.md"
	rows = screen(r.Render(f))
	if !coloured("study/Go.md", "work/Plan.md", th.Purple) {
		t.Errorf("the selection's backlink is not Purple")
	}
	f.Selected = "Home.md" // its link to Plan runs clear of the names
	rows = screen(r.Render(f))
	if !coloured("Home.md", "work/Plan.md", th.Blue) {
		t.Errorf("the selection's link is not Blue")
	}
}

// At the overview each galaxy's hub is named, with or without a
// selection, and no name ever covers a dot.
func TestRenderOverviewNamesTheHubs(t *testing.T) {
	g := fixture(300, 5, 600, 9)
	r := NewRenderer(Build(g), g.Edges)
	for _, sel := range []string{"", r.l.Nodes[40].ID} {
		f := Frame{W: 200, H: 60, Zoom: ZoomOverview, Offset: r.Centred(ZoomOverview, 200, 60), Selected: sel, Theme: theme.DefaultTheme()}
		out := r.Render(f)
		rows := screen(out)
		named := 0
		for _, h := range r.hubs {
			if h < 0 {
				continue
			}
			i := int(h)
			x, y := dotAt(t, r, f, r.l.Nodes[i].ID)
			if x < 0 || y < 0 || x >= f.W || y >= f.H {
				continue
			}
			if own, ok := r.nodeAt(f, x+2, y); ok && own == i {
				named++
			} else if own, ok := r.nodeAt(f, x-2, y); ok && own == i {
				named++
			}
		}
		if named == 0 {
			t.Fatalf("sel %q: no hub is named at the overview:\n%s", sel, ansi.Strip(out))
		}
		// The selection's own name may stay whole over its neighbours'
		// dots when no cut leaves minNearName cells (polish S4).
		selX, selY := -1, -1
		if sel != "" {
			selX, selY = dotAt(t, r, f, sel)
		}
		for _, n := range r.l.Nodes {
			x, y := dotAt(t, r, f, n.ID)
			if y == selY && x > selX && x <= selX+2+len(sel) {
				continue
			}
			if x >= 0 && y >= 0 && x < f.W && y < f.H && !isDot(rows[y][x]) {
				t.Fatalf("sel %q: %s's dot is covered by %q:\n%s", sel, n.ID, rows[y][x].r, ansi.Strip(out))
			}
		}
	}
	// The vault: both projects' hubs carry their names.
	r = NewRenderer(Build(vault()), vault().Edges)
	f := vaultFrame(r, ZoomOverview, 100, 24)
	f.Selected, f.Open = "", ""
	out := ansi.Strip(r.Render(f))
	for _, name := range []string{"Plan", "Go"} {
		if !strings.Contains(out, " "+name+" ") {
			t.Errorf("the hub %q is not named at the overview:\n%s", name, out)
		}
	}
}

// As before the galaxies: with a filter, a match outside the selection's
// neighbourhood keeps its galaxy's full colour (a missing note the text
// colour) and the rest turns grey; the highlighted few keep theirs.
func TestRenderFilterGreysTheRestInsteadOfFading(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	r := NewRenderer(Build(vault()), vault().Edges)
	for filter, want := range map[string]map[string]lipgloss.Color{
		"Rust": {"study/Rust.md": th.Purple, "work/Notes.md": th.Grey, "work/Tasks.md": th.Grey, "work/Plan.md": th.Yellow},
		"Idea": {"Ideas": th.Fg, "study/Rust.md": th.Grey, "Home.md": th.Grey},
		"Home": {"Home.md": th.Green, "Orphan.md": th.Grey, "work/Meeting.md": th.Grey},
	} {
		f := vaultFrame(r, ZoomClose, 200, 60)
		f.Filter = filter
		rows := screen(r.Render(f))
		for id, c := range want {
			x, y := dotAt(t, r, f, id)
			if got := rows[y][x].fg; got != rgb(c) {
				t.Errorf("filter %q: %s is %s, want %s", filter, id, got, rgb(c))
			}
		}
	}
}

// solarFrame shows the realistic vault whole at zoom z (the overview at
// 1×1) in a pane big enough for it, nothing selected.
func solarFrame(t *testing.T, z Zoom) (*Renderer, Frame) {
	t.Helper()
	g := realisticVault()
	r := NewRenderer(Build(g), g.Edges)
	k := []int{1, 2, 4}[z]
	w, h := (r.l.W+2)*k, (r.l.H+2)*k
	if z == ZoomOverview {
		w, h = r.l.W+2, r.l.H+2
	}
	return r, Frame{W: w, H: h, Zoom: z, Offset: r.Centred(z, w, h), Theme: theme.DefaultTheme()}
}

// at is where grid cell p is drawn in f.
func at(r *Renderer, f Frame, p Point) (x, y int) {
	k := r.scale(f.Zoom, f.W, f.H)
	return k.of(p.X) - f.Offset.X, k.of(p.Y) - f.Offset.Y
}

// hueOf is galaxy g's colour, as the renderer picks it.
func hueOf(th theme.Theme, g int) lipgloss.Color {
	palette := []lipgloss.Color{th.Blue, th.Purple, th.Teal, th.Orange, th.Cyan, th.Pink, th.Red, th.NordBlue}
	return palette[g%len(palette)]
}

// S7: the vault's sun sits in the middle of the map with the vault's name
// over its galaxy, at every zoom.
func TestSolarVaultSunInTheMiddleWithTheVaultName(t *testing.T) {
	for z := range zoomLevels {
		r, f := solarFrame(t, z)
		if dx, dy := r.l.Sun.X-r.l.W/2, r.l.Sun.Y-r.l.H/2; abs(dx) > r.l.W/8 || abs(dy) > r.l.H/8 {
			t.Fatalf("the vault's sun %v is off the middle of the %dx%d map", r.l.Sun, r.l.W, r.l.H)
		}
		rows := screen(r.Render(f))
		x, y := at(r, f, r.l.Sun)
		if rows[y][x].r != glyphVaultSun {
			t.Fatalf("zoom %d: the vault's sun cell shows %q:\n%s", z, rows[y][x].r, ansi.Strip(r.Render(f)))
		}
		found := false
		k := r.scale(f.Zoom, f.W, f.H)
		for dy := 1; dy <= k.of(r.l.Clusters[0].RY+1)+2 && !found; dy++ {
			found = y-dy >= 0 && strings.Contains(rowText(rows, y-dy, max(0, x-10), min(f.W-1, x+10)), "notas")
		}
		if !found {
			t.Fatalf("zoom %d: no vault name over the sun:\n%s", z, ansi.Strip(r.Render(f)))
		}
	}
}

// S7: every galaxy is joined to the vault's sun by a dim dotted curve.
func TestSolarEveryGalaxyIsJoinedToTheVaultSun(t *testing.T) {
	trueColour(t)
	r, f := solarFrame(t, ZoomMedium)
	th := f.Theme
	trunk := rgb(fade(th.Fg, th.Bg, trunkFade, th.Line))
	rows := screen(r.Render(f))
	k := r.scale(f.Zoom, f.W, f.H)
	sun := k.at(r.l.Sun)
	for _, c := range r.l.Clusters {
		if c.Key == "" {
			continue
		}
		var path []Point
		end := k.at(c.Centre)
		brailleCurve(sun.X, sun.Y, end.X, end.Y, trunkBend, func(x, y int, _ uint8) {
			if p := (Point{x - f.Offset.X, y - f.Offset.Y}); len(path) == 0 || path[len(path)-1] != p {
				path = append(path, p)
			}
		})
		hits := 0
		for _, p := range path[len(path)/5 : len(path)*4/5] { // the middle of the line, clear of both galaxies
			if cl := rows[p.Y][p.X]; isBraille(cl.r) && cl.fg == trunk {
				hits++
			}
		}
		if hits < 6 {
			t.Errorf("galaxy %q: %d trunk cells between it and the vault's sun", c.Key, hits)
		}
	}
}

// S7: each galaxy's sun is drawn at its centre in its colour, with the
// folder's full name, bold, over it.
func TestSolarGalaxySunAndFullFolderName(t *testing.T) {
	trueColour(t)
	r, f := solarFrame(t, ZoomOverview)
	rows := screen(r.Render(f))
	for g, c := range r.l.Clusters {
		if c.Key == "" {
			continue
		}
		x, y := at(r, f, c.Centre)
		if cl := rows[y][x]; cl.r != glyphSun || cl.fg != rgb(hueOf(f.Theme, g)) {
			t.Errorf("galaxy %q: centre shows %q %s, want its sun in %s", c.Key, cl.r, cl.fg, rgb(hueOf(f.Theme, g)))
		}
		hx, hy, ok := headingAt(rows, c.Key)
		all := len([]rune(fmt.Sprintf("%s · %d", c.Key, r.notes[g])))
		if !ok || hy >= y || hy < y-c.RY-2 || abs(hx+all/2-x) > 1 {
			t.Errorf("galaxy %q: full name at (%d,%d) %v, want centred over its sun at (%d,%d):\n%s", c.Key, hx, hy, ok, x, y, ansi.Strip(r.Render(f)))
		}
	}
}

// S7: every ring that holds a note is drawn as a dotted circle in its
// galaxy's faintest colour.
func TestSolarOrbitRingsAreDrawn(t *testing.T) {
	trueColour(t)
	r, f := solarFrame(t, ZoomClose)
	rows := screen(r.Render(f))
	k := r.scale(f.Zoom, f.W, f.H)
	for g, c := range r.l.Clusters {
		ring := rgb(fade(hueOf(f.Theme, g), f.Theme.Bg, ringFade, f.Theme.Line))
		used := map[int]bool{}
		for _, n := range r.l.Nodes {
			if n.Cluster == c.Key {
				used[ringOf(c.Centre, n.P)] = true
			}
		}
		for ringK := range used {
			cx, cy := at(r, f, c.Centre)
			rad := float64(k.of(ringRadius(ringK)))
			hits, tries := 0, 0
			for a := 0.0; a < 2*math.Pi; a += math.Pi / 16 {
				x, y := cx+int(math.Round(rad*math.Cos(a))), cy+int(math.Round(rad/cellAspect*math.Sin(a)))
				tries++
				for _, dx := range []int{0, -1, 1} { // the dots are ringDots apart
					if cl := rows[y][x+dx]; isBraille(cl.r) && cl.fg == ring {
						hits++
						break
					}
				}
			}
			if hits < tries/2 {
				t.Errorf("galaxy %q ring %d: %d of %d cells on it are ring dots", c.Key, ringK, hits, tries)
			}
		}
	}
}

// S7: every note is joined to its galaxy's sun (the vault's sun for the
// notes at the root) by a spoke in the galaxy's faded colour.
func TestSolarEveryNoteHasASpokeToItsSun(t *testing.T) {
	trueColour(t)
	r, f := solarFrame(t, ZoomClose)
	rows := screen(r.Render(f))
	for _, n := range r.l.Nodes {
		g := int(r.galaxy[r.l.index[n.ID]])
		c := r.l.Clusters[g].Centre
		if r.l.Clusters[g].Key == "" {
			c = r.l.Sun
		}
		spoke := rgb(fade(hueOf(f.Theme, g), f.Theme.Bg, spokeFade, f.Theme.Line))
		cx, cy := at(r, f, c)
		nx, ny := dotAt(t, r, f, n.ID)
		hits := 0
		for i := 1; i < 10; i++ {
			x, y := cx+(nx-cx)*i/10, cy+(ny-cy)*i/10
			if cl := rows[y][x]; isBraille(cl.r) && cl.fg == spoke {
				hits++
			}
		}
		if hits == 0 {
			t.Errorf("%s has no spoke to its sun at (%d,%d)", n.ID, cx, cy)
		}
	}
}

// S7: the real links run over the frame, brighter: every cell of a link
// that no dot, name or sun takes is drawn in the link's colour (light grey
// between galaxies, the galaxy's full colour inside one), none in a
// frame colour.
func TestSolarRealLinksAreBrighterOverTheFrame(t *testing.T) {
	trueColour(t)
	r, f := solarFrame(t, ZoomMedium)
	th := f.Theme
	rows := screen(r.Render(f))
	k := r.scale(f.Zoom, f.W, f.H)
	for _, p := range r.pairs {
		a, b := r.l.Nodes[p[0]], r.l.Nodes[p[1]]
		want := rgb(th.LightGrey)
		if a.Cluster == b.Cluster {
			want = rgb(hueOf(th, int(r.galaxy[p[0]])))
		}
		drawn := 0
		pa, pb := k.at(a.P), k.at(b.P)
		brailleCurve(pa.X, pa.Y, pb.X, pb.Y, linkBend, func(x, y int, _ uint8) {
			x, y = x-f.Offset.X, y-f.Offset.Y
			cl := rows[y][x]
			if !isBraille(cl.r) {
				return // a dot, a name or a sun
			}
			if cl.fg != want {
				t.Errorf("link %s→%s: cell (%d,%d) is %s, want %s", a.ID, b.ID, x, y, cl.fg, want)
			}
			drawn++
		})
		if drawn == 0 {
			t.Errorf("link %s→%s is not drawn", a.ID, b.ID)
		}
	}
}

// S7: dots, suns and names are never covered by the lines, at any zoom.
func TestSolarDotsSunsAndNamesAreNeverCovered(t *testing.T) {
	for z := range zoomLevels {
		r, f := solarFrame(t, z)
		rows := screen(r.Render(f))
		plain := ansi.Strip(r.Render(f))
		for _, n := range r.l.Nodes {
			if x, y := dotAt(t, r, f, n.ID); !isDot(rows[y][x]) {
				t.Fatalf("zoom %d: %s's dot shows %q:\n%s", z, n.ID, rows[y][x].r, plain)
			}
		}
		for _, c := range r.l.Clusters {
			if c.Key == "" {
				continue
			}
			if x, y := at(r, f, c.Centre); rows[y][x].r != glyphSun {
				t.Fatalf("zoom %d: %q's sun shows %q", z, c.Key, rows[y][x].r)
			}
			if !strings.Contains(plain, " "+c.Key+" ") {
				t.Fatalf("zoom %d: %q's name is not whole:\n%s", z, c.Key, plain)
			}
		}
	}
}

// S7: suns are no nodes: a click on one selects nothing and opens nothing,
// and the keys only ever move between notes.
func TestSolarSunsAreNotSelectable(t *testing.T) {
	g := realisticVault()
	m := New(theme.DefaultTheme())
	m.SetSize(120, 40)
	m.SetFocused(true)
	m.SetGraph(g)
	suns := []Point{m.l.Sun}
	for _, c := range m.l.Clusters {
		suns = append(suns, c.Centre)
	}
	k := m.scale()
	for _, p := range suns {
		o := m.viewOffset()
		m2, msg := click(m, k.of(p.X)-o.X, k.of(p.Y)-o.Y+1)
		if m2.Selected() != "" || msg != nil {
			t.Fatalf("a click on the sun at %v selected %q / sent %#v", p, m2.Selected(), msg)
		}
	}
	m, _ = press(m, "l")
	for i, k := range strings.Split("l l j j h h h k k l j h k j l", " ") {
		m, _ = press(m, k)
		n, ok := m.l.Lookup(m.Selected())
		if !ok {
			t.Fatalf("key %d %q selected %q, which is no note", i, k, m.Selected())
		}
		for _, p := range suns {
			if n.P == p {
				t.Fatalf("key %d %q selected a node on a sun", i, k)
			}
		}
	}
}

// S4 and the filter still hold on a solar map: the selection is Yellow on
// OneBg, its links' ends Blue and Purple, the rest keeps its galaxy's
// colour faded; a filter greys what it does not match instead.
func TestSolarSelectionAndFilterColoursStillHold(t *testing.T) {
	trueColour(t)
	r, f := solarFrame(t, ZoomMedium)
	th := f.Theme
	g := realisticVault()
	sel := g.Edges[3].From // misoku → stow-portfolio
	f.Selected, f.Focused = sel, true
	rows := screen(r.Render(f))
	check := func(id string, fg, bg lipgloss.Color) {
		t.Helper()
		x, y := dotAt(t, r, f, id)
		if c := rows[y][x]; c.fg != rgb(fg) || bg != "" && c.bg != rgb(bg) {
			t.Errorf("%s is %s on %s, want %s on %s", id, c.fg, c.bg, rgb(fg), rgb(bg))
		}
	}
	check(sel, th.Yellow, th.OneBg)
	check(g.Edges[3].To, th.Blue, "")
	check(g.Edges[0].From, th.Purple, "") // the root note linking to it
	rest := g.Nodes[60].ID
	gi := int(r.galaxy[r.l.index[rest]])
	check(rest, fade(hueOf(th, gi), th.Bg, restFade, th.GreyFg), "")
	f.Selected, f.Filter = "", "detalle"
	rows = screen(r.Render(f))
	for _, n := range r.l.Nodes {
		x, y := dotAt(t, r, f, n.ID)
		matched := strings.Contains(strings.ToLower(n.ID), "detalle")
		if c := rows[y][x]; !matched && c.fg != rgb(th.Grey) || matched && c.fg == rgb(th.Grey) {
			t.Fatalf("filter: %s (match %v) is %s", n.ID, matched, c.fg)
		}
	}
}

// S7: no name runs over a sun. The note straight left of a galaxy's sun
// (on its third ring) is selected: its name, which would reach the sun on
// the right, stops before it or goes left, in one piece.
func TestSolarNamesStopBeforeASun(t *testing.T) {
	var g Graph
	for i := range 46 { // rings 1 to 3 full
		g.Nodes = append(g.Nodes, Node{ID: fmt.Sprintf("big/a-rather-long-note-name-%02d.md", i), Cluster: "big"})
	}
	r := NewRenderer(Build(g), nil)
	c := r.l.Clusters[0]
	sel := ""
	for _, n := range r.l.Nodes {
		if n.P == (Point{c.Centre.X - ringRadius(3), c.Centre.Y}) {
			sel = n.ID
		}
	}
	if sel == "" {
		t.Fatal("setup: no note straight left of the sun")
	}
	for z := range zoomLevels {
		f := Frame{W: 160, H: 50, Zoom: z, Selected: sel, Theme: theme.DefaultTheme()}
		f.Offset = r.Centred(z, f.W, f.H)
		_, y := dotAt(t, r, f, sel)
		sx, _ := at(r, f, c.Centre)
		var own []int
		for x := range f.W {
			if i, ok := r.nodeAt(f, x, y); ok && r.l.Nodes[i].ID == sel {
				own = append(own, x)
			}
		}
		if len(own) == 0 || own[len(own)-1]-own[0]+1 != len(own) || own[0] <= sx && sx <= own[len(own)-1] {
			t.Fatalf("zoom %d: the selection's dot and name take columns %v, the sun is at %d:\n%s", z, own, sx, ansi.Strip(r.Render(f)))
		}
	}
}

// linkPath is the cells, in f, of the line drawn for the link between a
// and b.
func linkPath(r *Renderer, f Frame, a, b string) []Point {
	ia, ib := r.l.index[a], r.l.index[b]
	k := r.scale(f.Zoom, f.W, f.H)
	from, to := k.at(r.l.Nodes[min(ia, ib)].P), k.at(r.l.Nodes[max(ia, ib)].P)
	var path []Point
	r.link(from, to, func(x, y int, _ uint8) {
		if x -= f.Offset.X; x >= 0 && x < f.W {
			if y -= f.Offset.Y; y >= 0 && y < f.H {
				path = append(path, Point{x, y})
			}
		}
	})
	return path
}

// paneFrame is the realistic vault's overview in a pane the size of the
// user's: 180 columns by 56 map rows.
func paneFrame(t *testing.T) (*Renderer, Frame) {
	t.Helper()
	g := realisticVault()
	r := NewRenderer(Build(g), g.Edges)
	w, h := 180, 56
	return r, Frame{W: w, H: h, Zoom: ZoomOverview, Offset: r.Centred(ZoomOverview, w, h), Theme: theme.DefaultTheme()}
}

// S7 (look): the overview fills the pane, at least 85% of its width or of
// its height, and fits in it.
func TestSolarOverviewFillsThePane(t *testing.T) {
	r, f := paneFrame(t)
	rows := screen(r.Render(f))
	x0, y0, x1, y1 := f.W, f.H, -1, -1
	for y, row := range rows {
		for x, c := range row {
			if c.r != ' ' && c.r != 0 {
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			}
		}
	}
	w, h := x1-x0+1, y1-y0+1
	if w*100 < 85*f.W && h*100 < 85*f.H {
		t.Fatalf("the map takes %dx%d of the %dx%d pane, under 85%% of both:\n%s", w, h, f.W, f.H, ansi.Strip(r.Render(f)))
	}
	if x0 < 0 || y0 < 0 || x1 >= f.W || y1 >= f.H {
		t.Fatalf("the map spills out of the pane: %d,%d to %d,%d", x0, y0, x1, y1)
	}
}

// S7 (look): each folder's heading is its name and its number of notes,
// in full, when the pane is wide enough.
func TestSolarHeadingsCarryTheCountInFull(t *testing.T) {
	r, f := paneFrame(t)
	rows := screen(r.Render(f))
	count := map[string]int{}
	for _, n := range r.l.Nodes {
		count[n.Cluster]++
	}
	for _, c := range r.l.Clusters {
		if c.Key == "" {
			continue
		}
		if _, _, ok := headingAt(rows, fmt.Sprintf("%s · %d", c.Key, count[c.Key])); !ok {
			t.Errorf("no full heading %q · %d", c.Key, count[c.Key])
		}
	}
	if t.Failed() {
		t.Log("\n" + ansi.Strip(r.Render(f)))
	}
}

// S7 (look): every note is a clear dot in its galaxy's full colour when
// nothing is selected: • for a note, ● for its galaxy's busiest, never
// the small ·.
func TestSolarNotesAreFullColourDots(t *testing.T) {
	trueColour(t)
	r, f := paneFrame(t)
	rows := screen(r.Render(f))
	for i, n := range r.l.Nodes {
		x, y := at(r, f, n.P)
		c := rows[y][x]
		if c.r != glyphLinked && c.r != glyphHub {
			t.Fatalf("%s is drawn %q, want • or ●", n.ID, c.r)
		}
		if want := rgb(hueOf(f.Theme, int(r.galaxy[i]))); c.fg != want {
			t.Fatalf("%s is in %s, want its galaxy's %s", n.ID, c.fg, want)
		}
	}
}

// S7 (look): the vault's sun has a halo, a tight braille ring round it,
// even with no note at the root.
func TestSolarVaultSunHasAHalo(t *testing.T) {
	g := Graph{Name: "v"}
	for _, f := range []string{"a", "b", "c"} {
		for i := range 5 {
			g.Nodes = append(g.Nodes, Node{ID: fmt.Sprintf("%s/n%d.md", f, i), Cluster: f})
		}
	}
	r := NewRenderer(Build(g), nil)
	for z := range zoomLevels {
		w, h := 120, 40
		f := Frame{W: w, H: h, Zoom: z, Offset: Point{}, Theme: theme.DefaultTheme()}
		x, y := at(r, f, r.l.Sun)
		f.Offset = Point{x - w/2, y - h/2}
		rows := screen(r.Render(f))
		halo := 0
		for _, d := range []Point{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
			if isBraille(rows[h/2+d.Y][w/2+d.X].r) {
				halo++
			}
		}
		if halo < 6 {
			t.Fatalf("zoom %d: %d of the 8 cells round the vault's sun hold a halo:\n%s", z, halo, ansi.Strip(r.Render(f)))
		}
	}
}

// S7 (look): the overview names, beside the folders' headings, a note or
// two of most galaxies in a muted tone.
func TestSolarOverviewNamesTopNotesOfMostGalaxies(t *testing.T) {
	r, f := paneFrame(t)
	rows := screen(r.Render(f))
	named := map[int32]bool{}
	for i, n := range r.l.Nodes {
		x, y := at(r, f, n.P)
		cut := min(len([]rune(n.Label)), 6)
		label := string([]rune(n.Label)[:cut])
		if strings.Contains(rowText(rows, y, max(0, x-maxLabelWidth-2), min(f.W-1, x+maxLabelWidth+2)), label) {
			named[r.galaxy[i]] = true
		}
	}
	if len(named)*3 < len(r.l.Clusters)*2 {
		t.Fatalf("only %d of %d galaxies have a note named:\n%s", len(named), len(r.l.Clusters), ansi.Strip(r.Render(f)))
	}
}

// S7 (look): the lines from the vault's sun to the galaxies are gentle
// curves, not straight rays: the middle of each long one is off the
// straight line.
func TestSolarTrunksAreCurved(t *testing.T) {
	g := realisticVault()
	r := NewRenderer(Build(g), g.Edges)
	k := whole(2)
	fl := r.frameAt(layerKey{k: k})
	sun := k.at(r.l.Sun)
	for gi, c := range r.l.Clusters {
		if c.Key == "" {
			continue
		}
		end := k.at(c.Centre)
		dx, dy := float64(end.X-sun.X), 2*float64(end.Y-sun.Y) // screen columns
		length := math.Hypot(dx, dy)
		if length < 40 {
			continue
		}
		far := 0.0
		for i, kind := range fl.kind {
			if kind != prioTrunk || fl.galaxy[i] != int32(gi) {
				continue
			}
			px, py := float64(i%fl.w-sun.X), 2*float64(i/fl.w-sun.Y)
			far = max(far, math.Abs(dx*py-dy*px)/length)
		}
		if far < 2.5 {
			t.Errorf("the line to %s strays at most %.1f columns from straight", c.Key, far)
		}
	}
}
