package diagram

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const erConstructSource = `erDiagram
    title Commerce
    CUSTOMER {
        string id PK "Primary identifier"
        string email UK
        int region_id FK
    }
    ORDER {
        int id PK
        string note "Free text"
    }
    CUSTOMER ||--o{ ORDER : places
    ORDER }|..|{ INVOICE : contains
    PRODUCT ||--|{ LINE : includes`

func TestERDiagramIR(t *testing.T) {
	d, err := Parse(erConstructSource)
	if err != nil {
		t.Fatal(err)
	}
	want := Diagram{Kind: KindER, Title: "Commerce", Data: ERData{
		Entities: []EREntity{
			{ID: "CUSTOMER", Attributes: []ERAttribute{
				{Type: "string", Name: "id", PrimaryKey: true, Comment: "Primary identifier"},
				{Type: "string", Name: "email", UniqueKey: true},
				{Type: "int", Name: "region_id", ForeignKey: true},
			}},
			{ID: "ORDER", Attributes: []ERAttribute{
				{Type: "int", Name: "id", PrimaryKey: true},
				{Type: "string", Name: "note", Comment: "Free text"},
			}},
			{ID: "INVOICE"},
			{ID: "PRODUCT"},
			{ID: "LINE"},
		},
		Relationships: []ERRelationship{
			{From: "CUSTOMER", To: "ORDER", Label: "places", FromCardinality: ERCardinalityExactlyOne, ToCardinality: ERCardinalityZeroOrMore, Identifying: true},
			{From: "ORDER", To: "INVOICE", Label: "contains", FromCardinality: ERCardinalityOneOrMore, ToCardinality: ERCardinalityOneOrMore},
			{From: "PRODUCT", To: "LINE", Label: "includes", FromCardinality: ERCardinalityExactlyOne, ToCardinality: ERCardinalityOneOrMore, Identifying: true},
		},
	}}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("ER IR = %#v, want %#v", d, want)
	}
}

func TestERCardinalityConstructFamily(t *testing.T) {
	for _, tc := range []struct {
		name        string
		connector   string
		from, to    ERCardinality
		identifying bool
	}{
		{"zero or one", "|o--||", ERCardinalityZeroOrOne, ERCardinalityExactlyOne, true},
		{"exactly one", "||--||", ERCardinalityExactlyOne, ERCardinalityExactlyOne, true},
		{"zero or more", "}o--|{", ERCardinalityZeroOrMore, ERCardinalityOneOrMore, true},
		{"one or more", "}|--o{", ERCardinalityOneOrMore, ERCardinalityZeroOrMore, true},
		{"non-identifying", "|o..o{", ERCardinalityZeroOrOne, ERCardinalityZeroOrMore, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse("erDiagram\nA " + tc.connector + " B : relates")
			if err != nil {
				t.Fatal(err)
			}
			data := d.Data.(ERData)
			if len(data.Entities) != 2 || len(data.Relationships) != 1 {
				t.Fatalf("entities/relationships = %d/%d, want 2/1", len(data.Entities), len(data.Relationships))
			}
			got := data.Relationships[0]
			want := ERRelationship{From: "A", To: "B", Label: "relates", FromCardinality: tc.from, ToCardinality: tc.to, Identifying: tc.identifying}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("relationship = %#v, want %#v", got, want)
			}
		})
	}
}

func TestERAttributeConstructFamily(t *testing.T) {
	d, err := Parse(`erDiagram
ACCOUNT {
  uuid id PK FK UK "All keys"
  string display_name "A quoted: comment"
  decimal amount
}`)
	if err != nil {
		t.Fatal(err)
	}
	data := d.Data.(ERData)
	want := []ERAttribute{
		{Type: "uuid", Name: "id", PrimaryKey: true, ForeignKey: true, UniqueKey: true, Comment: "All keys"},
		{Type: "string", Name: "display_name", Comment: "A quoted: comment"},
		{Type: "decimal", Name: "amount"},
	}
	if !reflect.DeepEqual(data.Entities[0].Attributes, want) {
		t.Fatalf("attributes = %#v, want %#v", data.Entities[0].Attributes, want)
	}
}

func TestERUnknownConstruct(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		line   int
	}{
		{"invalid cardinality", "erDiagram\nA ||--X{ B", 2},
		{"missing entity", "erDiagram\n||--o{ ORDER", 2},
		{"unknown key", "erDiagram\nA {\nstring name XX\n}", 3},
		{"duplicate key", "erDiagram\nA {\nstring name PK PK\n}", 3},
		{"missing attribute type", "erDiagram\nA {\nname PK\n}", 3},
		{"trailing attribute token", "erDiagram\nA {\nstring name PK extra\n}", 3},
		{"unterminated comment", "erDiagram\nA {\nstring name \"broken\n}", 3},
		{"missing block close", "erDiagram\nA {\nstring name", 2},
		{"empty diagram", "erDiagram\ntitle Empty", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.source)
			if err == nil {
				t.Fatal("Parse silently accepted unsupported ER syntax")
			}
			if !reflect.DeepEqual(d, Diagram{}) {
				t.Fatalf("Parse returned a partial diagram on error: %#v", d)
			}
			var constructErr *ConstructError
			if !errors.As(err, &constructErr) {
				t.Fatalf("error = %v, want *ConstructError", err)
			}
			lines := strings.Split(tc.source, "\n")
			basicWantConstructError(t, err, KindER, tc.line, lines[tc.line-1])
		})
	}
}

func TestERDiagramLayout(t *testing.T) {
	for i := 0; i < 2; i++ {
		s := basicLayout(t, erConstructSource)
		if err := validateScene(s, testFonts(t)); err != nil {
			t.Fatal(err)
		}
		counts := basicCommandCounts(s)
		if counts[CommandPolyline] != 3 {
			t.Fatalf("polyline count = %d, want one connector per relationship", counts[CommandPolyline])
		}
		if counts[CommandLine] < 12 || counts[CommandEllipse] == 0 {
			t.Fatalf("commands = %#v, want cardinality markers and entity compartments", counts)
		}
		for _, text := range []string{
			"Commerce", "CUSTOMER", "ORDER", "INVOICE", "PRODUCT", "LINE",
			"PK string id", "UK string email", "FK int region_id", `"Primary identifier"`, `"Free text"`,
			"places", "contains", "includes",
		} {
			if !basicHasWrappedText(s, text) {
				t.Errorf("ER layout omitted %q", text)
			}
		}
	}
}

func TestERLayoutRejectsInvalidData(t *testing.T) {
	for _, tc := range []struct {
		name string
		data any
	}{
		{"wrong payload", "er"},
		{"empty entities", ERData{}},
		{"duplicate entity", ERData{Entities: []EREntity{{ID: "A"}, {ID: "A"}}}},
		{"missing endpoint", ERData{Entities: []EREntity{{ID: "A"}}, Relationships: []ERRelationship{{From: "A", To: "B", FromCardinality: ERCardinalityExactlyOne, ToCardinality: ERCardinalityZeroOrMore}}}},
		{"invalid cardinality", ERData{Entities: []EREntity{{ID: "A"}, {ID: "B"}}, Relationships: []ERRelationship{{From: "A", To: "B", FromCardinality: ERCardinality(99), ToCardinality: ERCardinalityExactlyOne}}}},
		{"empty attribute", ERData{Entities: []EREntity{{ID: "A", Attributes: []ERAttribute{{Type: "string"}}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := Layout(Diagram{Kind: KindER, Data: tc.data}, testFonts(t), testPalette())
			if err == nil {
				t.Fatal("Layout accepted invalid ER data")
			}
			var constructErr *ConstructError
			if !errors.As(err, &constructErr) || constructErr.Kind != KindER {
				t.Fatalf("error = %v, want ER *ConstructError", err)
			}
			if !reflect.DeepEqual(s, Scene{}) {
				t.Fatalf("Layout returned a partial scene: %#v", s)
			}
		})
	}
}

func TestERConstructGolden(t *testing.T) {
	assertGolden(t, "er/constructs", renderSource(t, erConstructSource))
}

func TestERLayoutDebug(t *testing.T) {
	d, _ := Parse(erConstructSource)
	data := d.Data.(ERData)
	fonts := testFonts(t)
	for _, entity := range data.Entities {
		measured := erMeasureBox(entity, fonts)
		t.Logf("entity %s measured=%#v", entity.ID, measured)
	}
	items := make([]diagramGridItem, len(data.Entities))
	for i, entity := range data.Entities {
		measured := erMeasureBox(entity, fonts)
		items[i] = diagramGridItem{w: measured.w, h: measured.h}
	}
	boxes, gridWidth, gridHeight := diagramGridPlace(items, 3, 28, 18, 10, 12)
	t.Logf("boxes=%#v grid=%dx%d", boxes, gridWidth, gridHeight)
	for i := range data.Relationships {
		path, ok := diagramRoute(boxes[i], boxes[(i+1)%len(boxes)], i, 140, 52, boxes)
		t.Logf("route %d ok=%v path=%#v", i, ok, path)
	}
	s := basicLayout(t, erConstructSource)
	t.Logf("scene=%dx%d", s.Width, s.Height)
	for i, command := range s.Commands {
		if command.Y > 60 || command.Y2 > 60 || command.Y+command.H > 60 {
			t.Logf("command %d: %#v", i, command)
		}
	}
}
