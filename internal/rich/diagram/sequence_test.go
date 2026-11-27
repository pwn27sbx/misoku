package diagram

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSequenceDiagramIR(t *testing.T) {
	source := strings.Join([]string{
		`sequenceDiagram`,
		`title Checkout`,
		`autonumber 10 5`,
		`actor Client as Customer`,
		`participant API`,
		`box "Services"`,
		`participant DB as Database`,
		`participant Cache`,
		`end`,
		`Client->>API: request`,
		`API-->>DB: query`,
		`note right of DB: durable`,
		`activate DB`,
		`DB--xAPI: failure`,
		`deactivate DB`,
		`alt attempt`,
		`loop retry`,
		`API-.->>Cache: lookup`,
		`end`,
		`else fallback`,
		`API->>Cache: store`,
		`end`,
		`opt audit`,
		`note over API,Cache: record`,
		`end`,
		`par parallel`,
		`Client-->API: ping`,
		`and`,
		`API-->Client: pong`,
		`end`,
		`rect rgba(0,0,0,0)`,
		`Client<<->>API: sync`,
		`end`,
	}, "\n")
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	want := Diagram{
		Kind:  KindSequence,
		Title: "Checkout",
		Data: SequenceData{
			Participants: []SequenceParticipant{
				{ID: "Client", Label: "Customer", Actor: true},
				{ID: "API", Label: "API"},
				{ID: "DB", Label: "Database", Group: "box_1"},
				{ID: "Cache", Label: "Cache", Group: "box_1"},
			},
			Groups: []SequenceGroup{{ID: "box_1", Label: "Services"}},
			Frames: []SequenceFrame{
				{ID: "frame_1", Kind: "alt", Label: "attempt"},
				{ID: "frame_2", Kind: "loop", Label: "retry", Parent: "frame_1"},
				{ID: "frame_3", Kind: "opt", Label: "audit"},
				{ID: "frame_4", Kind: "par", Label: "parallel"},
				{ID: "frame_5", Kind: "rect", Label: "rgba(0,0,0,0)"},
			},
			Events: []SequenceEvent{
				{Kind: SequenceMessage, From: "Client", To: "API", Label: "request", Style: EdgeSolid, End: ArrowFilled},
				{Kind: SequenceMessage, From: "API", To: "DB", Label: "query", Style: EdgeSolid, End: ArrowFilled},
				{Kind: SequenceNoteEvent, Note: SequenceNote{Position: "right", Participants: []string{"DB"}, Text: "durable"}},
				{Kind: SequenceActivate, Participant: "DB"},
				{Kind: SequenceMessage, From: "DB", To: "API", Label: "failure", Style: EdgeSolid, End: ArrowCross},
				{Kind: SequenceDeactivate, Participant: "DB"},
				{Kind: SequenceFrameOpen, Frame: "frame_1"},
				{Kind: SequenceFrameOpen, Frame: "frame_2"},
				{Kind: SequenceMessage, From: "API", To: "Cache", Label: "lookup", Style: EdgeDotted, End: ArrowFilled},
				{Kind: SequenceFrameEnd, Frame: "frame_2"},
				{Kind: SequenceFrameElse, Frame: "frame_1", Label: "fallback"},
				{Kind: SequenceMessage, From: "API", To: "Cache", Label: "store", Style: EdgeSolid, End: ArrowFilled},
				{Kind: SequenceFrameEnd, Frame: "frame_1"},
				{Kind: SequenceFrameOpen, Frame: "frame_3"},
				{Kind: SequenceNoteEvent, Note: SequenceNote{Position: "over", Participants: []string{"API", "Cache"}, Text: "record"}},
				{Kind: SequenceFrameEnd, Frame: "frame_3"},
				{Kind: SequenceFrameOpen, Frame: "frame_4"},
				{Kind: SequenceMessage, From: "Client", To: "API", Label: "ping", Style: EdgeSolid, End: ArrowOpen},
				{Kind: SequenceFrameElse, Frame: "frame_4"},
				{Kind: SequenceMessage, From: "API", To: "Client", Label: "pong", Style: EdgeSolid, End: ArrowOpen},
				{Kind: SequenceFrameEnd, Frame: "frame_4"},
				{Kind: SequenceFrameOpen, Frame: "frame_5"},
				{Kind: SequenceMessage, From: "Client", To: "API", Label: "sync", Style: EdgeSolid, Start: ArrowFilled, End: ArrowFilled},
				{Kind: SequenceFrameEnd, Frame: "frame_5"},
			},
			Autonumber:      true,
			AutonumberStart: 10,
			AutonumberStep:  5,
		},
	}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("diagram IR =\n%#v\nwant\n%#v", d, want)
	}
}

func TestSequenceMessages(t *testing.T) {
	tests := []struct {
		name  string
		arrow string
		style EdgeStyle
		start Arrowhead
		end   Arrowhead
		label string
	}{
		{name: "solid line", arrow: `->`, style: EdgeSolid},
		{name: "solid open", arrow: `-->`, style: EdgeSolid, end: ArrowOpen, label: "open"},
		{name: "solid filled", arrow: `->>`, style: EdgeSolid, end: ArrowFilled, label: "filled"},
		{name: "solid async", arrow: `-->>`, style: EdgeSolid, end: ArrowFilled},
		{name: "solid cross", arrow: `--x`, style: EdgeSolid, end: ArrowCross},
		{name: "dotted open", arrow: `-.->`, style: EdgeDotted, end: ArrowOpen},
		{name: "dotted filled", arrow: `-.->>`, style: EdgeDotted, end: ArrowFilled},
		{name: "dotted cross", arrow: `-.x`, style: EdgeDotted, end: ArrowCross},
		{name: "bidirectional open", arrow: `<-->`, style: EdgeSolid, start: ArrowOpen, end: ArrowOpen},
		{name: "bidirectional filled", arrow: `<<->>`, style: EdgeSolid, start: ArrowFilled, end: ArrowFilled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := `A` + tt.arrow + `B`
			if tt.label != "" {
				line += ": " + tt.label
			}
			d, err := parseSequence([]string{`participant A`, `participant B`, line})
			if err != nil {
				t.Fatal(err)
			}
			data := d.Data.(SequenceData)
			if len(data.Events) != 1 {
				t.Fatalf("events = %#v, want one message", data.Events)
			}
			event := data.Events[0]
			if event.Kind != SequenceMessage || event.From != "A" || event.To != "B" || event.Label != tt.label || event.Style != tt.style || event.Start != tt.start || event.End != tt.end {
				t.Fatalf("message = %#v, want A->B label %q style %v start %v end %v", event, tt.label, tt.style, tt.start, tt.end)
			}
		})
	}
}

func TestSequenceActivationShorthand(t *testing.T) {
	d, err := parseSequence([]string{`A->>+B: start`, `B->>-A: stop`})
	if err != nil {
		t.Fatal(err)
	}
	data := d.Data.(SequenceData)
	want := []SequenceEvent{
		{Kind: SequenceMessage, From: "A", To: "B", Label: "start", Style: EdgeSolid, End: ArrowFilled, Activate: "B"},
		{Kind: SequenceMessage, From: "B", To: "A", Label: "stop", Style: EdgeSolid, End: ArrowFilled, Deactivate: "B"},
	}
	if !reflect.DeepEqual(data.Events, want) {
		t.Fatalf("events = %#v, want %#v", data.Events, want)
	}
}

func TestSequenceFramesAndNotes(t *testing.T) {
	tests := []struct {
		name   string
		lines  []string
		frames []SequenceFrame
		events []SequenceEvent
	}{
		{
			name:   "alternative branches",
			lines:  []string{`alt yes`, `A->>B: ok`, `else no`, `A--xB: bad`, `end`},
			frames: []SequenceFrame{{ID: "frame_1", Kind: "alt", Label: "yes"}},
			events: []SequenceEvent{
				{Kind: SequenceFrameOpen, Frame: "frame_1"},
				{Kind: SequenceMessage, From: "A", To: "B", Label: "ok", Style: EdgeSolid, End: ArrowFilled},
				{Kind: SequenceFrameElse, Frame: "frame_1", Label: "no"},
				{Kind: SequenceMessage, From: "A", To: "B", Label: "bad", Style: EdgeSolid, End: ArrowCross},
				{Kind: SequenceFrameEnd, Frame: "frame_1"},
			},
		},
		{
			name:   "parallel branches",
			lines:  []string{`par work`, `A->>B: one`, `and other`, `A->>C: two`, `end`},
			frames: []SequenceFrame{{ID: "frame_1", Kind: "par", Label: "work"}},
			events: []SequenceEvent{
				{Kind: SequenceFrameOpen, Frame: "frame_1"},
				{Kind: SequenceMessage, From: "A", To: "B", Label: "one", Style: EdgeSolid, End: ArrowFilled},
				{Kind: SequenceFrameElse, Frame: "frame_1", Label: "other"},
				{Kind: SequenceMessage, From: "A", To: "C", Label: "two", Style: EdgeSolid, End: ArrowFilled},
				{Kind: SequenceFrameEnd, Frame: "frame_1"},
			},
		},
		{
			name:  "note positions",
			lines: []string{`note left of A: left`, `note right of B: right`, `note over A,B: both`},
			events: []SequenceEvent{
				{Kind: SequenceNoteEvent, Note: SequenceNote{Position: "left", Participants: []string{"A"}, Text: "left"}},
				{Kind: SequenceNoteEvent, Note: SequenceNote{Position: "right", Participants: []string{"B"}, Text: "right"}},
				{Kind: SequenceNoteEvent, Note: SequenceNote{Position: "over", Participants: []string{"A", "B"}, Text: "both"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parseSequence(tt.lines)
			if err != nil {
				t.Fatal(err)
			}
			data := d.Data.(SequenceData)
			if !reflect.DeepEqual(data.Frames, tt.frames) || !reflect.DeepEqual(data.Events, tt.events) {
				t.Fatalf("frames/events = %#v / %#v, want %#v / %#v", data.Frames, data.Events, tt.frames, tt.events)
			}
		})
	}
}

func TestSequenceUnknownConstruct(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		line      int
		construct string
	}{
		{name: "unknown arrow", body: "A ?!> B: bad", line: 2, construct: "A ?!> B: bad"},
		{name: "unsupported frame", body: "critical failure", line: 2, construct: "critical failure"},
		{name: "unexpected end", body: "end", line: 2, construct: "end"},
		{name: "unclosed frame", body: "loop retry\nA->>B: work", line: 3, construct: "unclosed frame"},
		{name: "invalid note position", body: "note above A: bad", line: 2, construct: "note above A: bad"},
		{name: "invalid autonumber", body: "autonumber nope", line: 2, construct: "autonumber nope"},
		{name: "unclosed box", body: "box \"Service\"\nparticipant A", line: 3, construct: "unclosed box"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := "sequenceDiagram\n" + tt.body
			_, err := Parse(source)
			var constructErr *ConstructError
			if !errors.As(err, &constructErr) {
				t.Fatalf("error = %v, want *ConstructError", err)
			}
			if constructErr.Kind != KindSequence || constructErr.Line != tt.line || constructErr.Construct != tt.construct {
				t.Fatalf("construct error = %#v, want sequence line %d %q", constructErr, tt.line, tt.construct)
			}
			if img := renderSource(t, source); img.Bounds().Empty() {
				t.Fatal("unknown construct produced an empty fallback")
			}
		})
	}
}

func TestSequenceLayoutConstructFamilies(t *testing.T) {
	source := strings.Join([]string{
		`sequenceDiagram`,
		`title Request flow`,
		`autonumber`,
		`actor Client as Customer`,
		`participant API`,
		`box "Services"`,
		`participant DB as Database`,
		`participant Cache`,
		`end`,
		`Client->>+API: request`,
		`note left of Client: entry`,
		`API-->>DB: query`,
		`activate DB`,
		`DB--xAPI: failure`,
		`deactivate DB`,
		`alt cached`,
		`API-.->>Cache: lookup`,
		`else fresh`,
		`API->>Cache: store`,
		`end`,
		`note over API,Cache: shared`,
		`opt audit`,
		`Client-->API: ping`,
		`end`,
		`par parallel`,
		`Client-->API: one`,
		`and`,
		`API-->Client: two`,
		`end`,
		`rect rgba(0,0,0,0)`,
		`Client<<->>API: sync`,
		`API->>-Client: done`,
		`end`,
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
	var dashed, filled, cross, bidirectional bool
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
		if command.Kind == CommandArrow {
			dashed = dashed || command.Dashed
			filled = filled || command.Head == ArrowFilled
			cross = cross || command.Head == ArrowCross
			bidirectional = bidirectional || (command.Head == ArrowFilled && command.Tail == ArrowFilled)
		}
	}
	if !dashed || !filled || !cross || !bidirectional {
		t.Fatalf("message decorations: dashed=%v filled=%v cross=%v bidirectional=%v", dashed, filled, cross, bidirectional)
	}
	text := sequenceTestText(s)
	for _, want := range []string{"Request flow", "Customer", "Services", "entry", "shared", "1. request", "alt cached", "else fresh", "and"} {
		if !strings.Contains(text, want) {
			t.Errorf("scene text does not contain %q", want)
		}
	}
	second, err := Layout(d, fonts, palette)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, second) {
		t.Fatal("repeated sequence layouts differ")
	}
}

func TestSequenceGoldenConstructs(t *testing.T) {
	source := strings.Join([]string{
		`sequenceDiagram`,
		`title Checkout sequence`,
		`autonumber 1 1`,
		`actor Client as Customer`,
		`participant API`,
		`box "Services"`,
		`participant DB as Database`,
		`participant Cache`,
		`end`,
		`Client->>+API: request`,
		`note left of Client: entry point`,
		`API-->>DB: query`,
		`activate DB`,
		`DB--xAPI: failure`,
		`deactivate DB`,
		`alt cached`,
		`API-.->>Cache: lookup`,
		`else fresh`,
		`API->>Cache: store`,
		`end`,
		`note over API,Cache: shared cache`,
		`opt audit`,
		`Client-->API: ping`,
		`end`,
		`par parallel`,
		`Client-->API: one`,
		`and`,
		`API-->Client: two`,
		`end`,
		`rect rgba(0,0,0,0)`,
		`Client<<->>API: sync`,
		`API->>-Client: done`,
		`end`,
	}, "\n")
	flowchartTestValidateSource(t, source)
	assertGolden(t, "sequence/constructs", renderSource(t, source))
}

func BenchmarkSequenceParseLayout(b *testing.B) {
	source := "sequenceDiagram\nparticipant A\nparticipant B\nA->>B: hello\nB-->>A: reply"
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

func sequenceTestText(s Scene) string {
	var out strings.Builder
	for _, command := range s.Commands {
		if command.Kind == CommandText || command.Kind == CommandBadge {
			out.WriteString(command.Text)
			out.WriteByte('\n')
		}
	}
	return out.String()
}
