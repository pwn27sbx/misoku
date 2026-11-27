package diagram

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/rich"
)

func TestFlowParseDirections(t *testing.T) {
	tests := []struct {
		name      string
		header    string
		direction string
	}{
		{name: "graph default", header: "graph", direction: "TD"},
		{name: "flowchart default", header: "flowchart", direction: "TD"},
		{name: "top down", header: "flowchart TD", direction: "TD"},
		{name: "top bottom alias", header: "graph TB", direction: "TB"},
		{name: "left right", header: "flowchart LR", direction: "LR"},
		{name: "right left", header: "graph RL", direction: "RL"},
		{name: "bottom top", header: "flowchart BT", direction: "BT"},
		{name: "case insensitive", header: "FlowChart lr", direction: "LR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := Parse(tt.header + "\nA --> B")
			if err != nil {
				t.Fatal(err)
			}
			if d.Kind != KindFlowchart || d.Direction != tt.direction {
				t.Fatalf("diagram = kind %q direction %q, want flowchart %q", d.Kind, d.Direction, tt.direction)
			}
		})
	}
}

func TestFlowParseShapes(t *testing.T) {
	tests := []struct {
		name   string
		source string
		label  string
		shape  NodeShape
	}{
		{name: "rectangle", source: `A[Rectangle]`, label: "Rectangle", shape: ShapeRect},
		{name: "rounded", source: `A(Rounded)`, label: "Rounded", shape: ShapeRounded},
		{name: "stadium", source: `A([Stadium])`, label: "Stadium", shape: ShapeStadium},
		{name: "subroutine", source: `A[[Subroutine]]`, label: "Subroutine", shape: ShapeSubroutine},
		{name: "cylinder", source: `A[(Cylinder)]`, label: "Cylinder", shape: ShapeCylinder},
		{name: "circle", source: `A((Circle))`, label: "Circle", shape: ShapeCircle},
		{name: "diamond", source: `A{Decision?}`, label: "Decision?", shape: ShapeDiamond},
		{name: "hexagon", source: `A{{Prepare}}`, label: "Prepare", shape: ShapeHexagon},
		{name: "parallelogram", source: `A[/Input/]`, label: "Input", shape: ShapeParallelogram},
		{name: "opposite parallelogram", source: `A[\Output\]`, label: "Output", shape: ShapeParallelogramAlt},
		{name: "double circle", source: `A(((Double)))`, label: "Double", shape: ShapeDoubleCircle},
		{name: "asymmetric", source: `A>Asymmetric]`, label: "Asymmetric", shape: ShapeAsymmetric},
		{name: "quoted punctuation", source: `A["semi; [brackets] (and) {braces}"]`, label: "semi; [brackets] (and) {braces}", shape: ShapeRect},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parseFlow("flowchart TD", []string{tt.source})
			if err != nil {
				t.Fatal(err)
			}
			if len(d.Nodes) != 1 {
				t.Fatalf("nodes = %#v, want one", d.Nodes)
			}
			n := d.Nodes[0]
			if n.ID != "A" || n.Label != tt.label || n.Shape != tt.shape {
				t.Fatalf("node = %#v, want A/%q/%v", n, tt.label, tt.shape)
			}
		})
	}
}

func TestFlowParseLaterLabelUpdates(t *testing.T) {
	d, err := parseFlow("flowchart TD", []string{`A["first; [label]"]`, `A("updated")`, `B`, `A`})
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Nodes) != 2 || d.Nodes[0].Label != "updated" || d.Nodes[0].Shape != ShapeRounded || d.Nodes[1].Label != "B" {
		t.Fatalf("nodes = %#v, want updated A and bare B", d.Nodes)
	}
}

func TestFlowParseEdges(t *testing.T) {
	tests := []struct {
		name   string
		source string
		label  string
		style  EdgeStyle
		start  Arrowhead
		end    Arrowhead
	}{
		{name: "solid open", source: `A --> B`, style: EdgeSolid, end: ArrowOpen},
		{name: "solid none", source: `A --- B`, style: EdgeSolid},
		{name: "dotted open", source: `A -.-> B`, style: EdgeDotted, end: ArrowOpen},
		{name: "dotted none", source: `A -.- B`, style: EdgeDotted},
		{name: "thick filled", source: `A ==> B`, style: EdgeThick, end: ArrowFilled},
		{name: "thick none", source: `A === B`, style: EdgeThick},
		{name: "invisible", source: `A ~~~ B`, style: EdgeInvisible},
		{name: "circle", source: `A --o B`, style: EdgeSolid, end: ArrowCircle},
		{name: "cross", source: `A --x B`, style: EdgeSolid, end: ArrowCross},
		{name: "bidirectional open", source: `A <--> B`, style: EdgeSolid, start: ArrowOpen, end: ArrowOpen},
		{name: "bidirectional circle", source: `A o--o B`, style: EdgeSolid, start: ArrowCircle, end: ArrowCircle},
		{name: "bidirectional cross", source: `A x--x B`, style: EdgeSolid, start: ArrowCross, end: ArrowCross},
		{name: "pipe label", source: `A -->|go| B`, label: "go", style: EdgeSolid, end: ArrowOpen},
		{name: "quoted pipe label", source: `A -.->|"left | right"| B`, label: "left | right", style: EdgeDotted, end: ArrowOpen},
		{name: "solid text label", source: `A -- go --> B`, label: "go", style: EdgeSolid, end: ArrowOpen},
		{name: "dotted text label", source: `A -. go .-> B`, label: "go", style: EdgeDotted, end: ArrowOpen},
		{name: "thick text label", source: `A == go ==> B`, label: "go", style: EdgeThick, end: ArrowFilled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parseFlow("flowchart LR", []string{tt.source})
			if err != nil {
				t.Fatal(err)
			}
			if len(d.Edges) != 1 {
				t.Fatalf("edges = %#v, want one", d.Edges)
			}
			e := d.Edges[0]
			if e.From != "A" || e.To != "B" || e.Label != tt.label || e.Style != tt.style || e.Start != tt.start || e.End != tt.end {
				t.Fatalf("edge = %#v, want A->B label %q style %v start %v end %v", e, tt.label, tt.style, tt.start, tt.end)
			}
		})
	}
}

func TestFlowParseChainsAndSemicolons(t *testing.T) {
	d, err := parseFlow("flowchart TD", []string{`A-->B-->C; B-- no -->D`, `C --- E`})
	if err != nil {
		t.Fatal(err)
	}
	wantEdges := []Edge{
		{From: "A", To: "B", Style: EdgeSolid, End: ArrowOpen},
		{From: "B", To: "C", Style: EdgeSolid, End: ArrowOpen},
		{From: "B", To: "D", Label: "no", Style: EdgeSolid, End: ArrowOpen},
		{From: "C", To: "E", Style: EdgeSolid},
	}
	if !reflect.DeepEqual(d.Edges, wantEdges) {
		t.Fatalf("edges = %#v, want %#v", d.Edges, wantEdges)
	}
	if len(d.Nodes) != 5 {
		t.Fatalf("nodes = %#v, want five chain nodes", d.Nodes)
	}
}

func TestFlowParseSubgraphsAndDirectives(t *testing.T) {
	source := []string{
		`title Deployment`,
		`subgraph outer [Outer]`,
		`direction LR`,
		`subgraph inner [Inner]`,
		`a[(Database)] --> b[[Worker]]`,
		`end`,
		`end`,
		`classDef accent fill:#f00,stroke:#0f0`,
		`class a accent`,
		`style b stroke:red`,
		`linkStyle 0 stroke:blue`,
		`click a "https://example.com"`,
	}
	d, err := parseFlow("flowchart TD", source)
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Deployment" || d.Direction != "TD" {
		t.Fatalf("title/direction = %q/%q", d.Title, d.Direction)
	}
	wantGroups := []Group{
		{ID: "outer", Label: "Outer", Direction: "LR", Nodes: []string{"a", "b"}},
		{ID: "inner", Label: "Inner", Nodes: []string{"a", "b"}},
	}
	if !reflect.DeepEqual(d.Groups, wantGroups) {
		t.Fatalf("groups = %#v, want %#v", d.Groups, wantGroups)
	}
	for _, n := range d.Nodes {
		if n.Group != "inner" {
			t.Fatalf("node %q group = %q, want inner", n.ID, n.Group)
		}
	}
}

func TestFlowDiagramIR(t *testing.T) {
	source := "flowchart LR\n" + strings.Join([]string{
		`subgraph api [API]`,
		`a([Start]) -->|yes| b{Choose?}`,
		`end`,
		`b -. next .-> c[(Store)]`,
		`c ==> d((Done))`,
	}, "\n")
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	want := Diagram{
		Kind:      KindFlowchart,
		Direction: "LR",
		Nodes: []Node{
			{ID: "a", Label: "Start", Shape: ShapeStadium, Group: "api"},
			{ID: "b", Label: "Choose?", Shape: ShapeDiamond, Group: "api"},
			{ID: "c", Label: "Store", Shape: ShapeCylinder},
			{ID: "d", Label: "Done", Shape: ShapeCircle},
		},
		Edges: []Edge{
			{From: "a", To: "b", Label: "yes", Style: EdgeSolid, End: ArrowOpen},
			{From: "b", To: "c", Label: "next", Style: EdgeDotted, End: ArrowOpen},
			{From: "c", To: "d", Style: EdgeThick, End: ArrowFilled},
		},
		Groups: []Group{{ID: "api", Label: "API", Nodes: []string{"a", "b"}}},
	}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("diagram IR =\n%#v\nwant\n%#v", d, want)
	}
}

func TestFlowParseErrors(t *testing.T) {
	tests := []struct {
		name   string
		header string
		lines  []string
		line   int
	}{
		{name: "invalid root direction", header: "flowchart XX", line: 1},
		{name: "missing target", header: "flowchart TD", lines: []string{`A -->`}, line: 2},
		{name: "unclosed subgraph", header: "flowchart TD", lines: []string{`subgraph s [S]`, `A --> B`}, line: 3},
		{name: "unexpected end", header: "flowchart TD", lines: []string{`end`}, line: 2},
		{name: "invalid nested direction", header: "flowchart TD", lines: []string{`subgraph s [S]`, `direction XX`}, line: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseFlow(tt.header, tt.lines)
			var constructErr *ConstructError
			if !errors.As(err, &constructErr) {
				t.Fatalf("error = %v, want *ConstructError", err)
			}
			if constructErr.Kind != KindFlowchart || constructErr.Line != tt.line {
				t.Fatalf("construct error = %#v, want flowchart line %d", constructErr, tt.line)
			}
		})
	}
}

func TestFlowUnknownConstruct(t *testing.T) {
	source := "flowchart TD\nA --> B\nmystery thing"
	_, err := Parse(source)
	var constructErr *ConstructError
	if !errors.As(err, &constructErr) {
		t.Fatalf("error = %v, want *ConstructError", err)
	}
	if constructErr.Line != 3 || constructErr.Construct != "mystery thing" {
		t.Fatalf("construct error = %#v, want line 3 mystery thing", constructErr)
	}
	img := renderSource(t, source)
	if img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
		t.Fatal("unknown construct produced an empty fallback")
	}
}

func TestFlowLayoutDirections(t *testing.T) {
	fonts := testFonts(t)
	palette := testPalette()
	tests := []struct {
		direction string
		check     func(a, b Command) bool
	}{
		{direction: "TD", check: func(a, b Command) bool { return a.Y < b.Y && a.X+a.W/2 == b.X+b.W/2 }},
		{direction: "TB", check: func(a, b Command) bool { return a.Y < b.Y && a.X+a.W/2 == b.X+b.W/2 }},
		{direction: "LR", check: func(a, b Command) bool { return a.X < b.X && a.Y+a.H/2 == b.Y+b.H/2 }},
		{direction: "RL", check: func(a, b Command) bool { return a.X > b.X && a.Y+a.H/2 == b.Y+b.H/2 }},
		{direction: "BT", check: func(a, b Command) bool { return a.Y > b.Y && a.X+a.W/2 == b.X+b.W/2 }},
	}
	for _, tt := range tests {
		t.Run(tt.direction, func(t *testing.T) {
			d, err := Parse("flowchart " + tt.direction + "\nA[Start] --> B[Finish]")
			if err != nil {
				t.Fatal(err)
			}
			s, err := Layout(d, fonts, palette)
			if err != nil {
				t.Fatal(err)
			}
			rects := flowchartTestNodeRects(s, palette)
			if len(rects) != 2 {
				t.Fatalf("node rectangles = %#v, want two", rects)
			}
			if !tt.check(rects[0], rects[1]) {
				t.Fatalf("%s node rectangles = %#v, %#v", tt.direction, rects[0], rects[1])
			}
		})
	}
}

func TestFlowLayoutConstructFamilies(t *testing.T) {
	source := strings.Join([]string{
		`flowchart LR`,
		`subgraph shapes [All shapes]`,
		`a[Rectangle] --> b(Rounded) --> c([Stadium])`,
		`d[[Subroutine]] --> e[(Cylinder)] --> f((Circle))`,
		`end`,
		`g{Decision} --> h{{Hexagon}} --> i[/Input/]`,
		`j[\Output\] --> k(((Double))) --> l>Flag]`,
		`m --o n`,
		`n --x o`,
		`n -. label .-> o`,
		`o ==> p`,
		`p ~~~ q`,
		`q --> q`,
		`q -. back .-> m`,
		`classDef muted fill:#222`,
		`class a muted`,
		`style b stroke:red`,
		`linkStyle 0 stroke:blue`,
		`click c "https://example.com"`,
	}, "\n")
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	fonts := testFonts(t)
	s, err := Layout(d, fonts, testPalette())
	if err != nil {
		t.Fatal(err)
	}
	if s.Width <= 0 || s.Height <= 0 {
		t.Fatalf("scene size = %dx%d, want positive", s.Width, s.Height)
	}
	kinds := make(map[CommandKind]bool)
	heads := make(map[Arrowhead]bool)
	tails := make(map[Arrowhead]bool)
	dashed, thick, polyline := false, false, false
	for _, c := range s.Commands {
		kinds[c.Kind] = true
		if c.Kind == CommandArrow {
			heads[c.Head] = true
			tails[c.Tail] = true
			dashed = dashed || c.Dashed
			thick = thick || c.Thick
		}
		polyline = polyline || c.Kind == CommandPolyline
	}
	for _, kind := range []CommandKind{CommandRect, CommandRounded, CommandEllipse, CommandDiamond, CommandHexagon, CommandParallelogram, CommandPolygon, CommandArrow, CommandPolyline, CommandText} {
		if !kinds[kind] {
			t.Errorf("scene has no command kind %v", kind)
		}
	}
	for _, head := range []Arrowhead{ArrowOpen, ArrowFilled, ArrowCircle, ArrowCross} {
		if !heads[head] {
			t.Errorf("scene has no arrowhead %v", head)
		}
	}
	if !tails[ArrowNone] || !dashed || !thick || !polyline {
		t.Fatalf("edge commands: tails=%v dashed=%v thick=%v polyline=%v", tails, dashed, thick, polyline)
	}
	if !flowchartTestHasText(s, "All shapes") || !flowchartTestHasText(s, "back") {
		t.Fatal("scene lost a subgraph title or back-edge label")
	}
}

func TestFlowLayoutSubgraphBounds(t *testing.T) {
	source := `flowchart TD
subgraph outer [Outer]
  subgraph inner [Inner]
    a[One] --> b[Two]
  end
end`
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Layout(d, testFonts(t), testPalette())
	if err != nil {
		t.Fatal(err)
	}
	palette := testPalette()
	nodes := flowchartTestNodeRects(s, palette)
	if len(nodes) != 2 {
		t.Fatalf("node rectangles = %#v, want two", nodes)
	}
	bounds := flowchartTestGroupBounds(s)
	if len(bounds) != 2 {
		t.Fatalf("group bounds = %#v, want outer and inner", bounds)
	}
	for _, bound := range bounds {
		if bound.X < 0 || bound.Y < 0 {
			t.Fatalf("group bound %#v starts outside the scene", bound)
		}
	}
	sort.Slice(bounds, func(i, j int) bool { return bounds[i].W*bounds[i].H < bounds[j].W*bounds[j].H })
	inner, outer := bounds[0], bounds[1]
	if !flowchartTestContains(inner, nodes[0]) && !flowchartTestContains(inner, nodes[1]) {
		t.Fatalf("inner bound %#v contains neither node %#v", inner, nodes)
	}
	for _, node := range nodes {
		if !flowchartTestContains(outer, node) {
			t.Fatalf("outer bound %#v does not contain node %#v", outer, node)
		}
	}
	if !flowchartTestHasText(s, "Outer") || !flowchartTestHasText(s, "Inner") {
		t.Fatal("subgraph titles are not visible")
	}
}

func TestFlowLayoutNestedDirection(t *testing.T) {
	source := `flowchart TD
subgraph side [Side]
  direction LR
  a[First] --> b[Second]
end`
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if d.Groups[0].Direction != "LR" {
		t.Fatalf("group direction = %q, want LR", d.Groups[0].Direction)
	}
	s, err := Layout(d, testFonts(t), testPalette())
	if err != nil {
		t.Fatal(err)
	}
	rects := flowchartTestNodeRects(s, testPalette())
	if len(rects) != 2 || !(rects[0].X < rects[1].X) {
		t.Fatalf("nested LR rectangles = %#v, want first left of second", rects)
	}
}

func TestFlowDirectivesDoNotAlterGeometry(t *testing.T) {
	fonts := testFonts(t)
	palette := testPalette()
	base, err := Parse("flowchart LR\nA[One] --> B[Two]")
	if err != nil {
		t.Fatal(err)
	}
	decorated, err := Parse("flowchart LR\nA[One] --> B[Two]\nclassDef x fill:#f00\nclass A x\nstyle B stroke:red\nlinkStyle 0 stroke:blue\nclick A \"https://example.com\"")
	if err != nil {
		t.Fatal(err)
	}
	a, err := Layout(base, fonts, palette)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Layout(decorated, fonts, palette)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("benign styling directives altered flowchart geometry")
	}
}

func TestFlowLayoutDeterministic(t *testing.T) {
	d, err := Parse("flowchart BT\nA --> B\nB --> C\nC -. back .-> A\nA --> A")
	if err != nil {
		t.Fatal(err)
	}
	fonts := testFonts(t)
	palette := testPalette()
	first, err := Layout(d, fonts, palette)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Layout(d, fonts, palette)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated flowchart layouts differ")
	}
}

func TestFlowGoldenConstructs(t *testing.T) {
	source := strings.Join([]string{
		`flowchart LR`,
		`subgraph shapes [Shapes]`,
		`a[Rectangle] --> b(Rounded) --> c([Stadium])`,
		`d[[Subroutine]] --> e[(Cylinder)] --> f((Circle))`,
		`end`,
		`g{Decision} --> h{{Hexagon}} --> i[/Input/]`,
		`j[\Output\] --> k(((Double))) --> l>Flag]`,
		`m --o n`,
		`n --x o`,
		`n -. label .-> o`,
		`o ==> p`,
		`p ~~~ q`,
		`q --> q`,
		`q -. back .-> m`,
	}, "\n")
	flowchartTestValidateSource(t, source)
	assertGolden(t, "flow/constructs", renderSource(t, source))
}

func TestFlowGoldenDirections(t *testing.T) {
	source := `flowchart RL
subgraph review [Review]
  A[Start] -->|check| B{Ready?}
  B -- no --> C[Fix]
  C -. retry .-> B
end
B -- yes --> D([Done])`
	flowchartTestValidateSource(t, source)
	assertGolden(t, "flow/directions", renderSource(t, source))
}

func BenchmarkFlowParseLayout(b *testing.B) {
	source := "flowchart LR\n" + strings.Join([]string{
		`subgraph service [Service]`,
		`a([Request]) --> b{Auth?}`,
		`b -- yes --> c[(Store)]`,
		`b -- no --> d[Deny]`,
		`c ==> e([Response])`,
		`d -. retry .-> a`,
		`end`,
	}, "\n")
	fonts := testFonts(b)
	palette := testPalette()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d, err := Parse(source)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := Layout(d, fonts, palette); err != nil {
			b.Fatal(err)
		}
	}
}

func flowchartTestValidateSource(t testing.TB, source string) {
	t.Helper()
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	fonts := testFonts(t)
	s, err := Layout(d, fonts, testPalette())
	if err != nil {
		t.Fatal(err)
	}
	if err := validateScene(s, fonts); err != nil {
		t.Fatalf("scene %dx%d is not renderable: %v", s.Width, s.Height, err)
	}
	for _, command := range s.Commands {
		if command.Kind != CommandText && command.Kind != CommandBadge {
			continue
		}
		if command.X < 0 || command.Y < 0 || command.X+command.W > s.Width || command.Y+command.H > s.Height {
			t.Fatalf("text %q at (%d,%d,%d,%d) clips scene %dx%d", command.Text, command.X, command.Y, command.W, command.H, s.Width, s.Height)
		}
	}
}

func flowchartTestNodeRects(s Scene, palette rich.Palette) []Command {
	var out []Command
	for _, c := range s.Commands {
		if c.Kind == CommandRect && c.Fill == palette.CodeBg && c.Stroke.A != 0 {
			out = append(out, c)
		}
	}
	return out
}

func flowchartTestGroupBounds(s Scene) []Command {
	var out []Command
	for _, c := range s.Commands {
		if (c.Kind == CommandRounded || c.Kind == CommandRect) && c.Fill.A == 0 && c.Stroke.A != 0 {
			out = append(out, c)
		}
	}
	return out
}

func flowchartTestContains(bounds, node Command) bool {
	return bounds.X <= node.X && bounds.Y <= node.Y && bounds.X+bounds.W >= node.X+node.W && bounds.Y+bounds.H >= node.Y+node.H
}

func flowchartTestHasText(s Scene, text string) bool {
	for _, c := range s.Commands {
		if c.Kind == CommandText && c.Text == text {
			return true
		}
	}
	return false
}
