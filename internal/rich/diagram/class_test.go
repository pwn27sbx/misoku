package diagram

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const classConstructSource = `classDiagram
    title Access model
    class "Account holder" as Person
    class Repository~T~ {
        +String name
        +int count $
        +find(id int, name String) String
        +reset()*
    }
    Person <|-- Employee
    Repository *-- Person : owns
    Person "1" o-- "0..*" Address
    Person --> Address : visits
    Person ..> Audit : records
    Person .. Audit : tracks
    Audit ..|> Repository : realizes`

func TestClassDiagramIR(t *testing.T) {
	d, err := Parse(classConstructSource)
	if err != nil {
		t.Fatal(err)
	}
	want := Diagram{Kind: KindClass, Title: "Access model", Data: ClassData{
		Classes: []Class{
			{ID: "Person", Label: "Account holder"},
			{ID: "Repository", TypeParameters: []string{"T"}, Members: []ClassMember{
				{Kind: ClassField, Visibility: "+", Type: "String", Name: "name"},
				{Kind: ClassField, Visibility: "+", Type: "int", Name: "count", Static: true},
				{Kind: ClassMethod, Visibility: "+", Name: "find", Parameters: []ClassParameter{
					{Name: "id", Type: "int"}, {Name: "name", Type: "String"},
				}, ReturnType: "String"},
				{Kind: ClassMethod, Visibility: "+", Name: "reset", Abstract: true},
			}},
			{ID: "Employee"},
			{ID: "Address"},
			{ID: "Audit"},
		},
		Relations: []ClassRelation{
			{From: "Person", To: "Employee", Kind: ClassInheritance, Style: EdgeSolid, StartMarker: ClassMarkerTriangle},
			{From: "Repository", To: "Person", Label: "owns", Kind: ClassComposition, Style: EdgeSolid, StartMarker: ClassMarkerDiamond},
			{From: "Person", To: "Address", FromMultiplicity: "1", ToMultiplicity: "0..*", Kind: ClassAggregation, Style: EdgeSolid, StartMarker: ClassMarkerHollowDiamond},
			{From: "Person", To: "Address", Label: "visits", Kind: ClassAssociation, Style: EdgeSolid, EndMarker: ClassMarkerFilledArrow},
			{From: "Person", To: "Audit", Label: "records", Kind: ClassDependency, Style: EdgeDotted, EndMarker: ClassMarkerFilledArrow},
			{From: "Person", To: "Audit", Label: "tracks", Kind: ClassAssociation, Style: EdgeDotted},
			{From: "Audit", To: "Repository", Label: "realizes", Kind: ClassRealization, Style: EdgeDotted, EndMarker: ClassMarkerTriangle},
		},
	}}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("class IR = %#v, want %#v", d, want)
	}
}

func TestClassRelationConstructFamily(t *testing.T) {
	for _, tc := range []struct {
		name       string
		connector  string
		kind       ClassRelationKind
		style      EdgeStyle
		start, end ClassMarker
	}{
		{"inheritance", "<|--", ClassInheritance, EdgeSolid, ClassMarkerTriangle, ClassMarkerNone},
		{"reverse inheritance", "--|>", ClassInheritance, EdgeSolid, ClassMarkerNone, ClassMarkerTriangle},
		{"composition", "*--", ClassComposition, EdgeSolid, ClassMarkerDiamond, ClassMarkerNone},
		{"reverse composition", "--*", ClassComposition, EdgeSolid, ClassMarkerNone, ClassMarkerDiamond},
		{"aggregation", "o--", ClassAggregation, EdgeSolid, ClassMarkerHollowDiamond, ClassMarkerNone},
		{"reverse aggregation", "--o", ClassAggregation, EdgeSolid, ClassMarkerNone, ClassMarkerHollowDiamond},
		{"directed association", "-->", ClassAssociation, EdgeSolid, ClassMarkerNone, ClassMarkerFilledArrow},
		{"reverse directed association", "<--", ClassAssociation, EdgeSolid, ClassMarkerFilledArrow, ClassMarkerNone},
		{"undirected association", "--", ClassAssociation, EdgeSolid, ClassMarkerNone, ClassMarkerNone},
		{"dependency", "..>", ClassDependency, EdgeDotted, ClassMarkerNone, ClassMarkerFilledArrow},
		{"reverse dependency", "<..", ClassDependency, EdgeDotted, ClassMarkerFilledArrow, ClassMarkerNone},
		{"undirected dotted association", "..", ClassAssociation, EdgeDotted, ClassMarkerNone, ClassMarkerNone},
		{"realization", "..|>", ClassRealization, EdgeDotted, ClassMarkerNone, ClassMarkerTriangle},
		{"reverse realization", "<|..", ClassRealization, EdgeDotted, ClassMarkerTriangle, ClassMarkerNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse("classDiagram\nA " + tc.connector + " B : relation")
			if err != nil {
				t.Fatal(err)
			}
			data := d.Data.(ClassData)
			if len(data.Classes) != 2 || len(data.Relations) != 1 {
				t.Fatalf("classes/relations = %d/%d, want 2/1", len(data.Classes), len(data.Relations))
			}
			got := data.Relations[0]
			want := ClassRelation{From: "A", To: "B", Label: "relation", Kind: tc.kind, Style: tc.style, StartMarker: tc.start, EndMarker: tc.end}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("relation = %#v, want %#v", got, want)
			}
		})
	}
}

func TestClassMultilineAndInlineMembers(t *testing.T) {
	d, err := Parse(`classDiagram
class A {
  #protected value
  ~String nickname
  +run()
}
class B { +static $ }
class A { +$int total }`)
	if err != nil {
		t.Fatal(err)
	}
	data := d.Data.(ClassData)
	if len(data.Classes) != 2 {
		t.Fatalf("classes = %#v, want two merged declarations", data.Classes)
	}
	want := []ClassMember{
		{Kind: ClassField, Visibility: "#", Type: "protected", Name: "value"},
		{Kind: ClassField, Visibility: "~", Type: "String", Name: "nickname"},
		{Kind: ClassMethod, Visibility: "+", Name: "run"},
		{Kind: ClassField, Visibility: "+", Name: "total", Type: "int", Static: true},
	}
	if !reflect.DeepEqual(data.Classes[0].Members, want) {
		t.Fatalf("members = %#v, want %#v", data.Classes[0].Members, want)
	}
	if got := data.Classes[1].Members; len(got) != 1 || got[0].Name != "static" || got[0].Static != true {
		t.Fatalf("inline static member = %#v", got)
	}
}

func TestClassUnknownConstruct(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		line   int
	}{
		{"unsupported namespace", "classDiagram\nnamespace Shapes", 2},
		{"unsupported relation", "classDiagram\nA ==> B", 2},
		{"missing block close", "classDiagram\nclass A {\n+int value", 2},
		{"invalid visibility", "classDiagram\nclass A {\n?int value\n}", 3},
		{"empty parameter", "classDiagram\nclass A {\n+run(, int value)\n}", 3},
		{"empty multiplicity", "classDiagram\nA --> \"\" B", 2},
		{"unterminated label", "classDiagram\nclass \"Broken as A", 2},
		{"empty diagram", "classDiagram\ntitle Empty", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.source)
			if err == nil {
				t.Fatalf("Parse silently accepted unsupported class syntax: %#v", d)
			}
			if !reflect.DeepEqual(d, Diagram{}) {
				t.Fatalf("Parse returned a partial diagram on error: %#v", d)
			}
			var constructErr *ConstructError
			if !errors.As(err, &constructErr) {
				t.Fatalf("error = %v, want *ConstructError", err)
			}
			lines := strings.Split(tc.source, "\n")
			basicWantConstructError(t, err, KindClass, tc.line, lines[tc.line-1])
		})
	}
}

func TestClassDiagramLayout(t *testing.T) {
	for i := 0; i < 2; i++ {
		s := basicLayout(t, classConstructSource)
		if err := validateScene(s, testFonts(t)); err != nil {
			t.Fatal(err)
		}
		counts := basicCommandCounts(s)
		if counts[CommandPolyline] != 7 {
			t.Fatalf("polyline count = %d, want one routed connector per relation", counts[CommandPolyline])
		}
		if counts[CommandPolygon] == 0 || counts[CommandLine] < 4 {
			t.Fatalf("commands = %#v, want relation markers and compartments", counts)
		}
		for _, text := range []string{
			"Access model", "Account holder", "Repository<T>", "Employee", "Address", "Audit",
			"+String name", "+int count $", "+find(id int, name String): String", "+reset()*",
			"owns", "visits", "records", "tracks", "realizes", "1", "0..*",
		} {
			if !basicHasWrappedText(s, text) {
				t.Errorf("class layout omitted %q", text)
			}
		}
	}
}

func TestClassLayoutRejectsInvalidData(t *testing.T) {
	for _, tc := range []struct {
		name string
		data any
	}{
		{"wrong payload", "class"},
		{"empty classes", ClassData{}},
		{"duplicate class", ClassData{Classes: []Class{{ID: "A"}, {ID: "A"}}}},
		{"missing endpoint", ClassData{Classes: []Class{{ID: "A"}}, Relations: []ClassRelation{{From: "A", To: "B", Kind: ClassAssociation}}}},
		{"invalid member", ClassData{Classes: []Class{{ID: "A", Members: []ClassMember{{Kind: ClassMemberKind(99), Name: "x"}}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := Layout(Diagram{Kind: KindClass, Data: tc.data}, testFonts(t), testPalette())
			if err == nil {
				t.Fatal("Layout accepted invalid class data")
			}
			var constructErr *ConstructError
			if !errors.As(err, &constructErr) || constructErr.Kind != KindClass {
				t.Fatalf("error = %v, want class *ConstructError", err)
			}
			if !reflect.DeepEqual(s, Scene{}) {
				t.Fatalf("Layout returned a partial scene: %#v", s)
			}
		})
	}
}

func TestClassConstructGolden(t *testing.T) {
	assertGolden(t, "class/constructs", renderSource(t, classConstructSource))
}
