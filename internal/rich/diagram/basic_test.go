package diagram

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

const basicPieSource = `pie
    title Pets adopted
    showData
    "Dogs" : 386
    "Cats" : 85
    "Rats" : 15`

const basicQuadrantSource = `quadrantChart
    title Campaign reach
    x-axis Low reach --> High reach
    y-axis Low engagement --> High engagement
    quadrant-1 High reach, high engagement
    quadrant-2 Low reach, high engagement
    quadrant-3 Low reach, low engagement
    quadrant-4 High reach, low engagement
    "Campaign: A" : [0.3, 0.6]
    Campaign B : [0.8, 0.2]
    Campaign C : [0.5, 0.5]`

const basicTimelineSource = `timeline
    title Product history
    section Foundations
    2020 - 2021 : Research
    2022 : Prototype
    section Launch
    2023 : Release
    2024 - 2025 : Growth`

const basicJourneySource = `journey
    title Working day
    section Morning
    Make tea: 5: Me
    Go upstairs: 3: Me, Cat
    section Work
    "Write code: core": 4: Me
    Review: 2: Me, Cat`

func TestBasicPieParser(t *testing.T) {
	t.Run("parses quoted labels and values", func(t *testing.T) {
		d, err := Parse("pie\nshowData\ntitle Share\n\"A \\\"quoted\\\" label\" : 0.5\nB rejected")
		if err == nil {
			t.Fatal("Parse accepted an unquoted slice label")
		}
		basicWantConstructError(t, err, KindPie, 5, "B rejected")

		d, err = Parse("pie\nshowData\ntitle Share\n\"A \\\"quoted\\\" label\" : 0.5\n'B' : 1.5e1")
		if err != nil {
			t.Fatal(err)
		}
		want := PieData{ShowData: true, Slices: []PieSlice{
			{Label: `A "quoted" label`, Value: 0.5},
			{Label: "B", Value: 15},
		}}
		if d.Kind != KindPie || d.Title != "Share" || !reflect.DeepEqual(d.Data, want) {
			t.Fatalf("pie = %#v, want title Share and %#v", d, want)
		}
	})

	t.Run("canonical header shows data", func(t *testing.T) {
		d, err := Parse("pie showData\ntitle Share\n\"Dogs\" : 3\n\"Cats\" : 1")
		if err != nil {
			t.Fatal(err)
		}
		want := PieData{ShowData: true, Slices: []PieSlice{
			{Label: "Dogs", Value: 3},
			{Label: "Cats", Value: 1},
		}}
		if d.Kind != KindPie || d.Title != "Share" || !reflect.DeepEqual(d.Data, want) {
			t.Fatalf("pie = %#v, want canonical showData header and %#v", d, want)
		}
	})

	t.Run("showData defaults off", func(t *testing.T) {
		d, err := Parse("pie\n\"Only\" : 2")
		if err != nil {
			t.Fatal(err)
		}
		if got := d.Data.(PieData).ShowData; got {
			t.Fatal("showData = true, want false")
		}
	})

	for _, tc := range []struct {
		name   string
		source string
		line   int
	}{
		{"negative value", "pie\n\"Dogs\" : -1", 2},
		{"non-finite value", "pie\n\"Dogs\" : NaN", 2},
		{"zero total", "pie\n\"Dogs\" : 0\n\"Cats\" : 0", 2},
		{"missing value", "pie\n\"Dogs\" :", 2},
		{"extra field", "pie\n\"Dogs\" : 1 : extra", 2},
		{"unescaped label quote", "pie\n\"Dogs\" \"Cats\" : 1", 2},
		{"duplicate directive", "pie\nshowData\nshowData\n\"Dogs\" : 1", 3},
		{"empty chart", "pie\ntitle Empty", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.source)
			if err == nil {
				t.Fatal("Parse accepted invalid pie syntax")
			}
			if !reflect.DeepEqual(d, Diagram{}) {
				t.Fatalf("Parse returned a partial diagram on error: %#v", d)
			}
			lines := strings.Split(tc.source, "\n")
			basicWantConstructError(t, err, KindPie, tc.line, lines[tc.line-1])
		})
	}
}

func TestBasicQuadrantParser(t *testing.T) {
	d, err := Parse(`quadrantChart
title Map
quadrant-2 High engagement
"A: point": [0, 1]
x-axis "Left side" --> "Right side"
quadrant-1 High reach
y-axis Bottom --> Top
B: [1.0, 0]
quadrant-3 Low engagement
quadrant-4 Low reach`)
	if err != nil {
		t.Fatal(err)
	}
	want := QuadrantData{
		XAxis: QuadrantAxis{From: "Left side", To: "Right side"},
		YAxis: QuadrantAxis{From: "Bottom", To: "Top"},
		Labels: [4]string{
			"High reach", "High engagement", "Low engagement", "Low reach",
		},
		Points: []QuadrantPoint{
			{Label: "A: point", X: 0, Y: 1},
			{Label: "B", X: 1, Y: 0},
		},
	}
	if d.Kind != KindQuadrant || d.Title != "Map" || !reflect.DeepEqual(d.Data, want) {
		t.Fatalf("quadrant = %#v, want title Map and %#v", d, want)
	}

	for _, tc := range []struct {
		name   string
		source string
		line   int
	}{
		{"missing x-axis", "quadrantChart\ny-axis Low --> High", 2},
		{"missing y-axis", "quadrantChart\nx-axis Low --> High", 2},
		{"malformed axis", "quadrantChart\nx-axis Low\n y-axis High", 2},
		{"point outside range", "quadrantChart\nx-axis Low --> High\ny-axis Low --> High\nA: [1.1, 0]", 4},
		{"negative coordinate", "quadrantChart\nx-axis Low --> High\ny-axis Low --> High\nA: [0, -0.1]", 4},
		{"wrong coordinate count", "quadrantChart\nx-axis Low --> High\ny-axis Low --> High\nA: [0.2]", 4},
		{"non-numeric coordinate", "quadrantChart\nx-axis Low --> High\ny-axis Low --> High\nA: [0.2, nope]", 4},
		{"unknown quadrant", "quadrantChart\nx-axis Low --> High\ny-axis Low --> High\nquadrant-5 Other", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.source)
			if err == nil {
				t.Fatal("Parse accepted invalid quadrant syntax")
			}
			if !reflect.DeepEqual(d, Diagram{}) {
				t.Fatalf("Parse returned a partial diagram on error: %#v", d)
			}
			lines := strings.Split(tc.source, "\n")
			basicWantConstructError(t, err, KindQuadrant, tc.line, lines[tc.line-1])
		})
	}
}

func TestBasicTimelineParser(t *testing.T) {
	d, err := Parse(`timeline
title History
2020 - 2022 : Research
section Launch
2023 : "Ship: beta"
2024 - 2025 : Grow`)
	if err != nil {
		t.Fatal(err)
	}
	want := TimelineData{Sections: []TimelineSection{
		{Events: []TimelineEvent{{Period: "2020 - 2022", Label: "Research"}}},
		{Title: "Launch", Events: []TimelineEvent{
			{Period: "2023", Label: "Ship: beta"},
			{Period: "2024 - 2025", Label: "Grow"},
		}},
	}}
	if d.Kind != KindTimeline || d.Title != "History" || !reflect.DeepEqual(d.Data, want) {
		t.Fatalf("timeline = %#v, want title History and %#v", d, want)
	}

	for _, tc := range []struct {
		name   string
		source string
		line   int
	}{
		{"missing separator", "timeline\n2024", 2},
		{"missing period", "timeline\n: Launch", 2},
		{"missing event", "timeline\n2024 :", 2},
		{"empty section", "timeline\nsection", 2},
		{"empty later section", "timeline\nsection First\n2020 : Launch\nsection Empty", 4},
		{"empty timeline", "timeline\ntitle Empty", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.source)
			if err == nil {
				t.Fatal("Parse accepted invalid timeline syntax")
			}
			if !reflect.DeepEqual(d, Diagram{}) {
				t.Fatalf("Parse returned a partial diagram on error: %#v", d)
			}
			lines := strings.Split(tc.source, "\n")
			basicWantConstructError(t, err, KindTimeline, tc.line, lines[tc.line-1])
		})
	}
}

func TestBasicJourneyParser(t *testing.T) {
	d, err := Parse(`journey
title Day
Make tea: 5: Me
section Work
"Write: code": 1: Me, Cat, "Dog"
Review: 3: Cat`)
	if err != nil {
		t.Fatal(err)
	}
	want := JourneyData{Sections: []JourneySection{
		{Tasks: []JourneyTask{{Label: "Make tea", Score: 5, Actors: []string{"Me"}}}},
		{Title: "Work", Tasks: []JourneyTask{
			{Label: "Write: code", Score: 1, Actors: []string{"Me", "Cat", "Dog"}},
			{Label: "Review", Score: 3, Actors: []string{"Cat"}},
		}},
	}}
	if d.Kind != KindJourney || d.Title != "Day" || !reflect.DeepEqual(d.Data, want) {
		t.Fatalf("journey = %#v, want title Day and %#v", d, want)
	}

	for _, tc := range []struct {
		name   string
		source string
		line   int
	}{
		{"score below range", "journey\nTask: 0: Me", 2},
		{"score above range", "journey\nTask: 6: Me", 2},
		{"non-integer score", "journey\nTask: 2.5: Me", 2},
		{"missing actors", "journey\nTask: 3:", 2},
		{"empty actor", "journey\nTask: 3: Me,", 2},
		{"duplicate actor", "journey\nTask: 3: Me, Me", 2},
		{"too many fields", "journey\nTask: 3: Me: extra", 2},
		{"empty later section", "journey\nsection First\nTask: 3: Me\nsection Empty", 4},
		{"empty journey", "journey\ntitle Empty", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.source)
			if err == nil {
				t.Fatal("Parse accepted invalid journey syntax")
			}
			if !reflect.DeepEqual(d, Diagram{}) {
				t.Fatalf("Parse returned a partial diagram on error: %#v", d)
			}
			lines := strings.Split(tc.source, "\n")
			basicWantConstructError(t, err, KindJourney, tc.line, lines[tc.line-1])
		})
	}
}

func TestBasicDiagramIR(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   Diagram
	}{
		{
			name:   "pie",
			source: basicPieSource,
			want: Diagram{Kind: KindPie, Title: "Pets adopted", Data: PieData{
				ShowData: true,
				Slices: []PieSlice{
					{Label: "Dogs", Value: 386},
					{Label: "Cats", Value: 85},
					{Label: "Rats", Value: 15},
				},
			}},
		},
		{
			name:   "quadrant",
			source: basicQuadrantSource,
			want: Diagram{Kind: KindQuadrant, Title: "Campaign reach", Data: QuadrantData{
				XAxis: QuadrantAxis{From: "Low reach", To: "High reach"},
				YAxis: QuadrantAxis{From: "Low engagement", To: "High engagement"},
				Labels: [4]string{
					"High reach, high engagement", "Low reach, high engagement",
					"Low reach, low engagement", "High reach, low engagement",
				},
				Points: []QuadrantPoint{
					{Label: "Campaign: A", X: 0.3, Y: 0.6},
					{Label: "Campaign B", X: 0.8, Y: 0.2},
					{Label: "Campaign C", X: 0.5, Y: 0.5},
				},
			}},
		},
		{
			name:   "timeline",
			source: basicTimelineSource,
			want: Diagram{Kind: KindTimeline, Title: "Product history", Data: TimelineData{
				Sections: []TimelineSection{
					{Title: "Foundations", Events: []TimelineEvent{
						{Period: "2020 - 2021", Label: "Research"},
						{Period: "2022", Label: "Prototype"},
					}},
					{Title: "Launch", Events: []TimelineEvent{
						{Period: "2023", Label: "Release"},
						{Period: "2024 - 2025", Label: "Growth"},
					}},
				},
			}},
		},
		{
			name:   "journey",
			source: basicJourneySource,
			want: Diagram{Kind: KindJourney, Title: "Working day", Data: JourneyData{
				Sections: []JourneySection{
					{Title: "Morning", Tasks: []JourneyTask{
						{Label: "Make tea", Score: 5, Actors: []string{"Me"}},
						{Label: "Go upstairs", Score: 3, Actors: []string{"Me", "Cat"}},
					}},
					{Title: "Work", Tasks: []JourneyTask{
						{Label: "Write code: core", Score: 4, Actors: []string{"Me"}},
						{Label: "Review", Score: 2, Actors: []string{"Me", "Cat"}},
					}},
				},
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("IR = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestBasicUnknownConstruct(t *testing.T) {
	for _, tc := range []struct {
		name      string
		source    string
		kind      Kind
		line      int
		construct string
	}{
		{"pie extra field", "pie\n\"Dogs\" : 1 : extra", KindPie, 2, `"Dogs" : 1 : extra`},
		{"quadrant bad coordinate", "quadrantChart\nx-axis Low --> High\ny-axis Low --> High\nA: [0.2, nope]", KindQuadrant, 4, "A: [0.2, nope]"},
		{"timeline missing event", "timeline\n2024", KindTimeline, 2, "2024"},
		{"journey bad score", "journey\nTask: 9: Me", KindJourney, 2, "Task: 9: Me"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.source)
			if err == nil {
				t.Fatal("Parse silently accepted an unsupported construct")
			}
			if !reflect.DeepEqual(d, Diagram{}) {
				t.Fatalf("Parse returned a partial diagram on error: %#v", d)
			}
			basicWantConstructError(t, err, tc.kind, tc.line, tc.construct)
		})
	}
}

func TestBasicPieLayout(t *testing.T) {
	for _, tc := range []struct {
		name      string
		directive string
		wantValue bool
	}{
		{"values shown", "showData\n", true},
		{"values hidden", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "pie\ntitle Share\n" + tc.directive + "\"Dogs\" : 3\n\"Cats\" : 1"
			s := basicLayout(t, source)
			basicAssertScene(t, s)
			var sectors []Command
			for _, cmd := range s.Commands {
				if cmd.Kind == CommandSector {
					sectors = append(sectors, cmd)
				}
			}
			if len(sectors) != 2 {
				t.Fatalf("sector count = %d, want 2", len(sectors))
			}
			if math.Abs(sectors[0].Start+math.Pi/2) > 1e-9 || math.Abs(sectors[1].End-3*math.Pi/2) > 1e-9 {
				t.Fatalf("sector angles = [%g, %g], [%g, %g]; want a complete clockwise pie", sectors[0].Start, sectors[0].End, sectors[1].Start, sectors[1].End)
			}
			if math.Abs((sectors[0].End-sectors[0].Start)-1.5*math.Pi) > 1e-9 {
				t.Fatalf("first sector angle = %g, want 3/4 of a circle", sectors[0].End-sectors[0].Start)
			}
			if !basicHasWrappedText(s, "Dogs") || !basicHasWrappedText(s, "Cats") || !basicHasText(s, "Share") {
				t.Fatal("pie layout omitted its title or legend labels")
			}
			if got := basicHasText(s, "Dogs: 3"); got != tc.wantValue {
				t.Fatalf("value label present = %v, want %v", got, tc.wantValue)
			}
		})
	}
}

func TestBasicPieLayoutZeroSlice(t *testing.T) {
	s := basicLayout(t, "pie\n\"Empty\" : 0\n\"Full\" : 1")
	basicAssertScene(t, s)
	sectors := 0
	for _, cmd := range s.Commands {
		if cmd.Kind == CommandSector {
			sectors++
		}
	}
	if sectors != 1 {
		t.Fatalf("sector count = %d, want one sector for the non-zero slice", sectors)
	}
	if !basicHasText(s, "Empty") || !basicHasText(s, "Full") {
		t.Fatal("zero-value slice disappeared from the legend")
	}
}

func TestBasicQuadrantLayout(t *testing.T) {
	s := basicLayout(t, basicQuadrantSource)
	basicAssertScene(t, s)
	counts := basicCommandCounts(s)
	if counts[CommandRect] < 4 || counts[CommandLine] < 2 || counts[CommandEllipse] != 3 {
		t.Fatalf("quadrant commands = %#v, want four quadrants, two axes and three points", counts)
	}
	for _, text := range []string{
		"Campaign reach", "Low reach", "High reach", "Low engagement", "High engagement",
		"High reach, high engagement", "Low reach, high engagement",
		"Low reach, low engagement", "High reach, low engagement",
		"Campaign: A", "Campaign B", "Campaign C",
	} {
		if !basicHasWrappedText(s, text) {
			t.Errorf("quadrant layout omitted %q", text)
		}
	}
}

func TestBasicTimelineLayout(t *testing.T) {
	s := basicLayout(t, basicTimelineSource)
	basicAssertScene(t, s)
	counts := basicCommandCounts(s)
	if counts[CommandLine] < 4 || counts[CommandBadge] != 4 {
		t.Fatalf("timeline commands = %#v, want at least four event connectors and four period badges", counts)
	}
	for _, text := range []string{
		"Product history", "Foundations", "Launch", "2020 - 2021", "2022", "2023", "2024 - 2025",
		"Research", "Prototype", "Release", "Growth",
	} {
		if !basicHasText(s, text) {
			t.Errorf("timeline layout omitted %q", text)
		}
	}
	rows := basicTextRows(s, "Research", "Prototype", "Release", "Growth")
	for i := 1; i < len(rows); i++ {
		if rows[i] == rows[i-1] {
			t.Fatalf("event rows = %v, want alternating rows", rows)
		}
	}
}

func TestBasicJourneyLayout(t *testing.T) {
	s := basicLayout(t, basicJourneySource)
	basicAssertScene(t, s)
	counts := basicCommandCounts(s)
	if counts[CommandEllipse] != 8 {
		t.Fatalf("ellipse count = %d, want one score cell for each of two actors and four tasks", counts[CommandEllipse])
	}
	for _, text := range []string{
		"Working day", "Morning", "Work", "Me", "Cat", "Make tea", "Go upstairs", "Write code: core", "Review",
	} {
		if !basicHasWrappedText(s, text) {
			t.Errorf("journey layout omitted %q", text)
		}
	}
	for _, score := range []string{"1", "2", "3", "4", "5"} {
		if !basicHasText(s, score) {
			t.Errorf("journey layout omitted score %s", score)
		}
	}
}

func TestBasicLayoutsDeterministic(t *testing.T) {
	for _, source := range []string{basicPieSource, basicQuadrantSource, basicTimelineSource, basicJourneySource} {
		d, err := Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		first, err := Layout(d, testFonts(t), testPalette())
		if err != nil {
			t.Fatal(err)
		}
		second, err := Layout(d, testFonts(t), testPalette())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("%s layout is not deterministic", d.Kind)
		}
	}
}

func TestBasicLayoutRejectsInvalidData(t *testing.T) {
	for _, tc := range []struct {
		name string
		d    Diagram
	}{
		{"pie wrong payload", Diagram{Kind: KindPie, Data: "pie"}},
		{"pie empty values", Diagram{Kind: KindPie, Data: PieData{Slices: []PieSlice{{Label: "A", Value: 0}}}}},
		{"quadrant wrong payload", Diagram{Kind: KindQuadrant, Data: "quadrant"}},
		{"quadrant invalid coordinate", Diagram{Kind: KindQuadrant, Data: QuadrantData{Points: []QuadrantPoint{{Label: "A", X: 2}}}}},
		{"timeline wrong payload", Diagram{Kind: KindTimeline, Data: "timeline"}},
		{"timeline empty section", Diagram{Kind: KindTimeline, Data: TimelineData{Sections: []TimelineSection{{Title: "Empty"}}}}},
		{"journey wrong payload", Diagram{Kind: KindJourney, Data: "journey"}},
		{"journey invalid score", Diagram{Kind: KindJourney, Data: JourneyData{Sections: []JourneySection{{Tasks: []JourneyTask{{Label: "Task", Score: 6, Actors: []string{"Me"}}}}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := Layout(tc.d, testFonts(t), testPalette())
			if err == nil {
				t.Fatal("Layout accepted invalid typed data")
			}
			var constructErr *ConstructError
			if !errors.As(err, &constructErr) || constructErr.Kind != tc.d.Kind {
				t.Fatalf("Layout error = %v, want *ConstructError for %s", err, tc.d.Kind)
			}
			if !reflect.DeepEqual(s, Scene{}) {
				t.Fatalf("Layout returned a partial scene on error: %#v", s)
			}
		})
	}
}

func TestBasicRenderedTitleWidth(t *testing.T) {
	fonts := testFonts(t)
	for _, tc := range []struct {
		name   string
		source string
		title  string
	}{
		{"pie", basicPieSource, "Pets adopted"},
		{"quadrant", basicQuadrantSource, "Campaign reach"},
		{"timeline", basicTimelineSource, "Product history"},
		{"journey", basicJourneySource, "Working day"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			img := renderSource(t, tc.source)
			minX, maxX := img.Bounds().Dx(), -1
			for y := 0; y < fonts.CellH && y < img.Bounds().Dy(); y++ {
				for x := 0; x < img.Bounds().Dx(); x++ {
					if img.RGBAAt(x, y).A == 0 {
						continue
					}
					minX = min(minX, x)
					maxX = max(maxX, x)
				}
			}
			if maxX < minX {
				t.Fatal("rendered title painted no pixels")
			}
			span := maxX - minX + 1
			want := int(math.Ceil(float64(fonts.Measure(tc.title, fonts.BodyPx, false)) / 64))
			if span > want+2 || span < want-2 {
				t.Fatalf("rendered title spans %d px, measured width is %d px; glyphs are clipped or spaced incorrectly", span, want)
			}
		})
	}
}

func TestBasicPieGolden(t *testing.T) {
	assertGolden(t, "basic/pie", renderSource(t, basicPieSource))
}

func TestBasicQuadrantGolden(t *testing.T) {
	assertGolden(t, "basic/quadrant", renderSource(t, basicQuadrantSource))
}

func TestBasicTimelineGolden(t *testing.T) {
	assertGolden(t, "basic/timeline", renderSource(t, basicTimelineSource))
}

func TestBasicJourneyGolden(t *testing.T) {
	assertGolden(t, "basic/journey", renderSource(t, basicJourneySource))
}

func basicWantConstructError(t *testing.T, err error, kind Kind, line int, construct string) {
	t.Helper()
	var constructErr *ConstructError
	if !errors.As(err, &constructErr) {
		t.Fatalf("error = %v, want *ConstructError", err)
	}
	if constructErr.Kind != kind || constructErr.Line != line || constructErr.Construct != construct {
		t.Fatalf("ConstructError = %#v, want kind %s line %d construct %q", constructErr, kind, line, construct)
	}
	if !strings.Contains(err.Error(), "unsupported construct") {
		t.Fatalf("error %q does not explain the unsupported construct", err)
	}
}

func basicLayout(t *testing.T, source string) Scene {
	t.Helper()
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Layout(d, testFonts(t), testPalette())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func basicAssertScene(t *testing.T, s Scene) {
	t.Helper()
	if s.Width <= 0 || s.Height <= 0 || len(s.Commands) == 0 {
		t.Fatalf("scene = %#v, want positive dimensions and commands", s)
	}
	for i, cmd := range s.Commands {
		if cmd.X < 0 || cmd.Y < 0 || cmd.X2 < 0 || cmd.Y2 < 0 {
			t.Fatalf("command %d has a negative coordinate: %#v", i, cmd)
		}
		if cmd.Kind != CommandLine && cmd.Kind != CommandArrow && cmd.Kind != CommandPolyline && (cmd.W <= 0 || cmd.H <= 0) {
			t.Fatalf("command %d has non-positive dimensions: %#v", i, cmd)
		}
		if cmd.Kind == CommandLine || cmd.Kind == CommandArrow {
			if cmd.X2 > s.Width || cmd.Y2 > s.Height {
				t.Fatalf("command %d ends outside the scene: %#v", i, cmd)
			}
		} else if cmd.X+cmd.W > s.Width || cmd.Y+cmd.H > s.Height {
			t.Fatalf("command %d extends outside the scene: %#v", i, cmd)
		}
	}
}

func basicCommandCounts(s Scene) map[CommandKind]int {
	counts := make(map[CommandKind]int)
	for _, cmd := range s.Commands {
		counts[cmd.Kind]++
	}
	return counts
}

func basicHasText(s Scene, want string) bool {
	for _, cmd := range s.Commands {
		if (cmd.Kind == CommandText || cmd.Kind == CommandBadge) && cmd.Text == want {
			return true
		}
	}
	return false
}

func basicHasWrappedText(s Scene, want string) bool {
	var texts []string
	for _, cmd := range s.Commands {
		if cmd.Kind == CommandText || cmd.Kind == CommandBadge {
			texts = append(texts, cmd.Text)
		}
	}
	return strings.Contains(strings.Join(texts, " "), want)
}

func basicTextRows(s Scene, labels ...string) []int {
	var rows []int
	for _, label := range labels {
		found := false
		for _, cmd := range s.Commands {
			if cmd.Kind == CommandText && cmd.Text == label {
				rows = append(rows, cmd.Y)
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return rows
}

// Mermaid writes the title on the header line as often as on its own:
// "pie title Pets" is valid and must draw, with or without showData.
func TestBasicPieInlineTitle(t *testing.T) {
	for _, src := range []string{
		"pie title Mascotas\n\"Perros\" : 60\n\"Gatos\" : 40",
		"pie showData title Mascotas\n\"Perros\" : 60\n\"Gatos\" : 40",
	} {
		d, err := Parse(src)
		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		if d.Kind != KindPie || d.Title != "Mascotas" {
			t.Fatalf("%q: kind %q title %q", src, d.Kind, d.Title)
		}
		data, ok := d.Data.(PieData)
		if !ok || len(data.Slices) != 2 {
			t.Fatalf("%q: data %+v", src, d.Data)
		}
	}
	if _, err := Parse("pie title\n\"A\" : 1"); err == nil {
		t.Fatal("an empty inline title was accepted")
	}
	if _, err := Parse("pie nonsense\n\"A\" : 1"); err == nil {
		t.Fatal("a bogus header was accepted")
	}
}
