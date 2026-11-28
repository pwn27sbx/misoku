package diagram

import (
	"image/color"
	"math"
	"strings"
	"testing"
)

func TestFadeProducesValidPremultipliedRGBA(t *testing.T) {
	c := color.RGBA{R: 230, G: 140, B: 90, A: 255}
	got := fade(c, 40)
	if got.A != 40 {
		t.Fatalf("fade alpha = %d, want 40", got.A)
	}
	if got.R > got.A || got.G > got.A || got.B > got.A {
		t.Fatalf("fade(%+v, 40) = %+v: invalid premultiplied RGBA (channels exceed alpha)", c, got)
	}
}

func TestPieIsCircularInPixels(t *testing.T) {
	img := renderSource(t, basicPieSource)
	// Restrict to the chart area on the left side (before legend).
	b := img.Bounds()
	minX, minY, maxX, maxY := b.Max.X, b.Max.Y, b.Min.X, b.Min.Y
	chartMaxX := b.Dx() * 6 / 10
	// Skip the title row at the top (first 28 px).
	for y := 28; y < b.Max.Y; y++ {
		for x := b.Min.X; x < chartMaxX; x++ {
			if img.RGBAAt(x, y).A > 0 {
				minX = min(minX, x)
				minY = min(minY, y)
				maxX = max(maxX, x)
				maxY = max(maxY, y)
			}
		}
	}
	w, h := maxX-minX+1, maxY-minY+1
	if w <= 0 || h <= 0 {
		t.Fatalf("pie pixels not found")
	}
	ratio := float64(w) / float64(h)
	if math.Abs(ratio-1.0) > 0.1 {
		t.Fatalf("pie aspect ratio = %.2f (%dx%d px), want ~1.0 circle", ratio, w, h)
	}
}

func TestSequenceDeactivatesSenderAndHidesRectCSS(t *testing.T) {
	source := strings.Join([]string{
		"sequenceDiagram",
		"participant Client",
		"participant API",
		"Client->>+API: request",
		"rect rgba(0,0,0,0)",
		"API->>-Client: done",
		"end",
	}, "\n")
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	data := d.Data.(SequenceData)
	var deact string
	for _, ev := range data.Events {
		if ev.Kind == SequenceMessage && ev.Deactivate != "" {
			deact = ev.Deactivate
		}
	}
	if deact != "API" {
		t.Fatalf("Deactivate = %q, want sender %q", deact, "API")
	}
	s, err := Layout(d, testFonts(t), testPalette())
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range s.Commands {
		if (cmd.Kind == CommandText || cmd.Kind == CommandBadge) && strings.Contains(cmd.Text, "rgba(") {
			t.Fatalf("rect CSS colour rendered as text label: %q", cmd.Text)
		}
	}
}

func TestStateHorizontalCompositeIsCompact(t *testing.T) {
	source := strings.Join([]string{
		`stateDiagram-v2`,
		`state "Fulfilment" as fulfilment {`,
		`direction LR`,
		`[*] --> queued`,
		`state "Queued" as queued`,
		`queued --> shipped: dispatch`,
		`state "Shipped" as shipped`,
		`shipped --> [*]: delivered`,
		`}`,
	}, "\n")
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Layout(d, testFonts(t), testPalette())
	if err != nil {
		t.Fatal(err)
	}
	if s.Height > 18 {
		t.Fatalf("horizontal composite state height = %d cells, want <= 18", s.Height)
	}
}

func TestFlowchartEdgeLabelMidpointNotOnTarget(t *testing.T) {
	source := "flowchart LR\nA[Start] -->|check| B[End]"
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Layout(d, testFonts(t), testPalette())
	if err != nil {
		t.Fatal(err)
	}
	var target Command
	var label Command
	for _, cmd := range s.Commands {
		if cmd.Kind == CommandRect && cmd.Stroke.A != 0 && cmd.X > target.X {
			target = cmd
		}
		if cmd.Kind == CommandText && cmd.Text == "check" {
			label = cmd
		}
	}
	if label.Text == "" {
		t.Fatal("missing edge label")
	}
	if label.X+label.W > target.X {
		t.Fatalf("edge label (%d..%d) overlaps target box starting at %d", label.X, label.X+label.W, target.X)
	}
}

func TestERPortStaysWithinBoxHeight(t *testing.T) {
	box := diagramGridBox{x: 10, y: 20, w: 24, h: 4}
	for i := 0; i < 6; i++ {
		for _, side := range []diagramSide{diagramSideLeft, diagramSideRight} {
			p := diagramPort(box, side, i)
			if p.Y < box.y || p.Y >= box.y+box.h {
				t.Fatalf("diagramPort(side=%v, idx=%d) Y=%d outside box [%d, %d)", side, i, p.Y, box.y, box.y+box.h)
			}
		}
	}
}
