package diagram

import (
	"image"
	"testing"

	"github.com/pwn27sbx/misoku/internal/rich"
)

var benchmarkSources = []struct {
	name   string
	source string
}{
	{"flowchart", "flowchart LR\nA[Start] --> B{Choice}\nB -->|yes| C[(Done)]"},
	{"sequence", "sequenceDiagram\nparticipant A\nparticipant B\nA->>B: hello\nB-->>A: reply"},
	{"class", "classDiagram\nclass A\nclass B\nA <|-- B"},
	{"state", "stateDiagram-v2\n[*] --> A\nA --> B: next\nB --> [*]"},
	{"er", "erDiagram\nCUSTOMER ||--o{ ORDER : places"},
	{"pie", "pie\ntitle Share\n\"A\" : 60\n\"B\" : 40"},
	{"xychart", "xychart-beta\ntitle Series\nx-axis [min --> max]\nbar [1, 2, 3]"},
	{"quadrant", "quadrantChart\ntitle Map\nx-axis Low --> High\ny-axis Low --> High\nA: [0.7, 0.8]"},
	{"gantt", "gantt\ntitle Plan\ndateFormat YYYY-MM-DD\nsection Work\nTask: 1d"},
	{"timeline", "timeline\ntitle Events\nsection 2026\nLaunch: done"},
	{"journey", "journey\ntitle Day\nsection Work\nTask: 4: Alice"},
	{"mindmap", "mindmap\nroot((Root))\n  Child"},
	{"gitGraph", "gitGraph\ncommit id: \"initial\"\nbranch dev\ncommit id: \"feature\""},
}

func BenchmarkDiagramParseLayout(b *testing.B) {
	fonts := testFonts(b)
	palette := testPalette()
	for _, tc := range benchmarkSources {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				d, err := Parse(tc.source)
				if err != nil {
					b.Fatal(err)
				}
				if _, err := Layout(d, fonts, palette); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDiagramCachedDraw(b *testing.B) {
	fonts := testFonts(b)
	for _, tc := range benchmarkSources {
		b.Run(tc.name, func(b *testing.B) {
			box := New(fonts, testPalette()).Box(rich.Block{Code: splitSourceLines(tc.source)})
			if box == nil {
				b.Fatal("no box")
			}
			img := image.NewRGBA(image.Rect(0, 0, box.W*fonts.CellW, box.H*fonts.CellH))
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				box.Draw(img, 0, 0)
			}
		})
	}
}
