package diagram

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestStateDiagramIR(t *testing.T) {
	source := strings.Join([]string{
		`stateDiagram-v2`,
		`title Checkout state`,
		`state "Idle" as idle`,
		`idle : Waiting for a request`,
		`state decision <<choice>>`,
		`state split <<fork>>`,
		`state join <<join>>`,
		`[*] --> idle`,
		`idle --> decision: begin`,
		`decision --> split: yes`,
		`decision --> idle: no`,
		`state "Processing" as processing {`,
		`direction LR`,
		`[*] --> work`,
		`state "Worker" as work`,
		`work : Runs the job`,
		`work --> [*]: done`,
		`note right of work: background job`,
		`}`,
		`split --> processing`,
		`processing --> join`,
		`join --> [*]`,
		`note left of idle: entry point`,
	}, "\n")
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	want := Diagram{
		Kind:  KindState,
		Title: "Checkout state",
		Data: StateData{
			Nodes: []StateNode{
				{ID: "idle", Label: "Idle", Description: "Waiting for a request"},
				{ID: "decision", Label: "decision", Kind: StateChoice},
				{ID: "split", Label: "split", Kind: StateFork},
				{ID: "join", Label: "join", Kind: StateJoin},
				{ID: "@start", Kind: StateStart},
				{ID: "processing", Label: "Processing"},
				{ID: "@start_2", Kind: StateStart, Parent: "processing"},
				{ID: "work", Label: "Worker", Description: "Runs the job", Parent: "processing"},
				{ID: "@end", Kind: StateEnd, Parent: "processing"},
				{ID: "@end_2", Kind: StateEnd},
			},
			Groups: []StateGroup{{ID: "processing", Label: "Processing", Direction: "LR"}},
			Transitions: []StateTransition{
				{From: "@start", To: "idle"},
				{From: "idle", To: "decision", Label: "begin"},
				{From: "decision", To: "split", Label: "yes"},
				{From: "decision", To: "idle", Label: "no"},
				{From: "@start_2", To: "work"},
				{From: "work", To: "@end", Label: "done"},
				{From: "split", To: "processing"},
				{From: "processing", To: "join"},
				{From: "join", To: "@end_2"},
			},
			Notes: []StateNote{
				{Anchor: "work", Position: "right", Text: "background job"},
				{Anchor: "idle", Position: "left", Text: "entry point"},
			},
		},
	}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("diagram IR =\n%#v\nwant\n%#v", d, want)
	}
}

func TestStateAliasesAndSpecialNodes(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		id    string
		label string
		kind  StateNodeKind
	}{
		{name: "bare state", line: `state Ready`, id: "Ready", label: "Ready", kind: StateNormal},
		{name: "description alias", line: `state "Ready now" as ready`, id: "ready", label: "Ready now", kind: StateNormal},
		{name: "id alias", line: `state ready as "Ready now"`, id: "ready", label: "Ready now", kind: StateNormal},
		{name: "choice", line: `state decide <<choice>>`, id: "decide", label: "decide", kind: StateChoice},
		{name: "fork", line: `state split <<fork>>`, id: "split", label: "split", kind: StateFork},
		{name: "join", line: `state merge <<join>>`, id: "merge", label: "merge", kind: StateJoin},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parseState([]string{tt.line})
			if err != nil {
				t.Fatal(err)
			}
			data := d.Data.(StateData)
			if len(data.Nodes) != 1 {
				t.Fatalf("nodes = %#v, want one", data.Nodes)
			}
			node := data.Nodes[0]
			if node.ID != tt.id || node.Label != tt.label || node.Kind != tt.kind {
				t.Fatalf("node = %#v, want %q/%q/%v", node, tt.id, tt.label, tt.kind)
			}
		})
	}
}

func TestStateCompositeAndNotes(t *testing.T) {
	source := strings.Join([]string{
		`stateDiagram`,
		`[*] --> outer`,
		`state "Outer" as outer {`,
		`[*] --> inner`,
		`state "Inner" as inner {`,
		`state running`,
		`inner --> running: start`,
		`running --> [*]`,
		`}`,
		`inner --> [*]`,
		`note left of inner: nested`,
		`}`,
		`outer --> [*]`,
		`note right of outer: complete`,
	}, "\n")
	d, err := parseState(splitSourceLines(source)[1:])
	if err != nil {
		t.Fatal(err)
	}
	data := d.Data.(StateData)
	wantGroups := []StateGroup{
		{ID: "outer", Label: "Outer"},
		{ID: "inner", Label: "Inner", Parent: "outer"},
	}
	if !reflect.DeepEqual(data.Groups, wantGroups) {
		t.Fatalf("groups = %#v, want %#v", data.Groups, wantGroups)
	}
	if len(data.Nodes) != 8 {
		t.Fatalf("nodes = %#v, want eight", data.Nodes)
	}
	byID := make(map[string]StateNode)
	for _, node := range data.Nodes {
		byID[node.ID] = node
	}
	if byID["outer"].Parent != "" || byID["inner"].Parent != "outer" || byID["running"].Parent != "inner" {
		t.Fatalf("composite parents = %#v", byID)
	}
	if byID["@start"].Parent != "" || byID["@start_2"].Parent != "outer" {
		t.Fatalf("nested start parents = %#v", byID)
	}
	wantNotes := []StateNote{
		{Anchor: "inner", Position: "left", Text: "nested"},
		{Anchor: "outer", Position: "right", Text: "complete"},
	}
	if !reflect.DeepEqual(data.Notes, wantNotes) {
		t.Fatalf("notes = %#v, want %#v", data.Notes, wantNotes)
	}
}

func TestStateUnknownConstruct(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		line      int
		construct string
	}{
		{name: "unknown arrow", body: "A ~~> B: bad", line: 2, construct: "A ~~> B: bad"},
		{name: "unsupported history", body: "state A <<history>>", line: 2, construct: "state A <<history>>"},
		{name: "invalid note position", body: "note above A: bad", line: 2, construct: "note above A: bad"},
		{name: "unexpected close", body: "}", line: 2, construct: "}"},
		{name: "unclosed composite", body: "state \"Open\" as open {\n[*] --> x", line: 3, construct: "unclosed composite state"},
		{name: "empty alias", body: `state "Broken" as`, line: 2, construct: `state "Broken" as`},
		{name: "duplicate state", body: "state A\nstate A", line: 3, construct: "state A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := "stateDiagram-v2\n" + tt.body
			_, err := Parse(source)
			var constructErr *ConstructError
			if !errors.As(err, &constructErr) {
				t.Fatalf("error = %v, want *ConstructError", err)
			}
			if constructErr.Kind != KindState || constructErr.Line != tt.line || constructErr.Construct != tt.construct {
				t.Fatalf("construct error = %#v, want state line %d %q", constructErr, tt.line, tt.construct)
			}
			if img := renderSource(t, source); img.Bounds().Empty() {
				t.Fatal("unknown construct produced an empty fallback")
			}
		})
	}
}

func TestStateLayoutConstructFamilies(t *testing.T) {
	source := strings.Join([]string{
		`stateDiagram-v2`,
		`title Processing state`,
		`state "Idle" as idle`,
		`idle : Waiting for work`,
		`state decide <<choice>>`,
		`state split <<fork>>`,
		`state merge <<join>>`,
		`[*] --> idle`,
		`idle --> decide: begin`,
		`decide --> split: yes`,
		`decide --> idle: no`,
		`state "Processing" as processing {`,
		`direction LR`,
		`[*] --> queued`,
		`state "Queued" as queued`,
		`queued --> running: start`,
		`state "Running" as running`,
		`running : Performs the work`,
		`running --> [*]: done`,
		`note right of running: background job`,
		`}`,
		`split --> processing`,
		`processing --> merge`,
		`merge --> [*]`,
		`note left of idle: entry point`,
	}, "\n")
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	fonts, palette := testFonts(t), testPalette()
	s, err := Layout(d, fonts, palette)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateScene(s, fonts); err != nil {
		t.Fatal(err)
	}
	if s.Width <= 0 || s.Height <= 0 || len(s.Commands) == 0 {
		t.Fatalf("scene = %dx%d with %d commands", s.Width, s.Height, len(s.Commands))
	}
	var ellipse, diamond, rect, arrow bool
	for _, command := range s.Commands {
		if command.X < 0 || command.Y < 0 || command.X2 < 0 || command.Y2 < 0 {
			t.Fatalf("negative command coordinate: %#v", command)
		}
		for _, point := range command.Points {
			if point.X < 0 || point.Y < 0 {
				t.Fatalf("negative point: %#v", command)
			}
		}
		if command.Kind == CommandText || command.Kind == CommandBadge {
			if command.X < 0 || command.Y < 0 || command.X+command.W > s.Width || command.Y+command.H > s.Height {
				t.Fatalf("text %q clips scene %dx%d", command.Text, s.Width, s.Height)
			}
		}
		ellipse = ellipse || command.Kind == CommandEllipse
		diamond = diamond || command.Kind == CommandDiamond
		rect = rect || command.Kind == CommandRect
		arrow = arrow || command.Kind == CommandArrow
	}
	if !ellipse || !diamond || !rect || !arrow {
		t.Fatalf("state primitives: ellipse=%v diamond=%v rect=%v arrow=%v", ellipse, diamond, rect, arrow)
	}
	text := sequenceTestText(s)
	for _, want := range []string{"Processing state", "Idle", "Waiting for work", "Processing", "Queued", "Running", "Performs the work", "background job", "entry point", "yes", "no", "done"} {
		if !strings.Contains(text, want) {
			t.Errorf("scene text does not contain %q", want)
		}
	}
	second, err := Layout(d, fonts, palette)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, second) {
		t.Fatal("repeated state layouts differ")
	}
}

func TestStateGoldenConstructs(t *testing.T) {
	source := strings.Join([]string{
		`stateDiagram-v2`,
		`title Order lifecycle`,
		`state "Created" as created`,
		`created : Waiting for payment`,
		`state paid <<choice>>`,
		`state split <<fork>>`,
		`state merge <<join>>`,
		`[*] --> created`,
		`created --> paid: submit`,
		`paid --> split: accepted`,
		`paid --> created: rejected`,
		`state "Fulfilment" as fulfilment {`,
		`direction LR`,
		`[*] --> queued`,
		`state "Queued" as queued`,
		`queued --> shipped: dispatch`,
		`state "Shipped" as shipped`,
		`shipped --> [*]: delivered`,
		`note right of shipped: tracking enabled`,
		`}`,
		`split --> fulfilment`,
		`fulfilment --> merge`,
		`merge --> [*]`,
		`note left of created: order entry`,
		`note right of merge: all branches complete`,
	}, "\n")
	flowchartTestValidateSource(t, source)
	assertGolden(t, "state/constructs", renderSource(t, source))
}

func BenchmarkStateParseLayout(b *testing.B) {
	source := "stateDiagram-v2\n[*] --> A\nA --> B: next\nB --> [*]"
	fonts, palette := testFonts(b), testPalette()
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
