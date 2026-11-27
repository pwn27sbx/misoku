package diagram

import (
	"strings"
	"testing"
)

func TestDiagramIR(t *testing.T) {
	fonts := testFonts(t)
	palette := testPalette()
	wantKinds := []Kind{
		KindFlowchart, KindSequence, KindClass, KindState, KindER, KindPie,
		KindXY, KindQuadrant, KindGantt, KindTimeline, KindJourney, KindMindmap, KindGit,
	}
	if len(benchmarkSources) != len(wantKinds) {
		t.Fatalf("benchmark corpus has %d types, want %d", len(benchmarkSources), len(wantKinds))
	}
	for i, tc := range benchmarkSources {
		t.Run(tc.name, func(t *testing.T) {
			d, err := Parse(tc.source)
			if err != nil {
				t.Fatalf("Parse: %v\nsource:\n%s", err, tc.source)
			}
			if d.Kind != wantKinds[i] {
				t.Fatalf("kind = %q, want %q", d.Kind, wantKinds[i])
			}
			scene, err := Layout(d, fonts, palette)
			if err != nil {
				t.Fatalf("Layout: %v", err)
			}
			if scene.Width <= 0 || scene.Height <= 0 || len(scene.Commands) == 0 {
				t.Fatalf("scene = %dx%d with %d commands, want a non-empty layout", scene.Width, scene.Height, len(scene.Commands))
			}
			if err := validateScene(scene, fonts); err != nil {
				t.Fatal(err)
			}
			if img := renderScene(scene, fonts); img == nil || img.Bounds().Empty() {
				t.Fatal("renderScene returned no image")
			}
		})
	}
}

func TestDiagramUnknownConstruct(t *testing.T) {
	cases := []string{
		"sequenceDiagram\nparticipant A\nA ?!> B: unsupported arrow",
		"flowchart LR\nA -->|ok| B\nB ~~> C: unsupported edge",
		"pie title Bad\n\"A\" : not-a-number",
	}
	for _, source := range cases {
		_, err := Parse(source)
		if err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Errorf("Parse(%q) error = %v, want unsupported construct", source, err)
		}
	}
}
