package diagram

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/rich"
)

func TestCoreLabelSizeUsesCellsNotPixels(t *testing.T) {
	fonts := testFonts(t)
	_, width, _ := labelSize("Hello", fonts, 60, fonts.BodyPx, false)
	if width > 10 {
		t.Fatalf("label width = %d cells for a short word; pixels were treated as cells", width)
	}
	if width*fonts.CellW < int(fonts.Measure("Hello", fonts.BodyPx, false)/64) {
		t.Fatalf("label width = %d cells does not cover the measured glyphs", width)
	}
}

func TestCoreStatementSplittingPreservesLabels(t *testing.T) {
	got := splitStatements(`A["semi;colon"] --> B; C --> D`)
	want := []string{`A["semi;colon"] --> B`, `C --> D`}
	if len(got) != len(want) {
		t.Fatalf("split statements = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("statement %d = %q, want %q", i, got[i], want[i])
		}
	}
	lines := prepareLines("flowchart LR\n  A[kept %% comment] --> B\n%% full comment\nC --> D")
	if lines[1] != "  A[kept" || lines[2] != "" || lines[3] != "C --> D" {
		t.Fatalf("prepared lines = %q", lines)
	}
}

func TestCoreBoxIntegratesWithRichLayout(t *testing.T) {
	fonts := testFonts(t)
	palette := testPalette()
	renderer := New(fonts, palette)
	doc := rich.Document{Blocks: []rich.Block{{
		Kind: rich.KindDiagram,
		Lang: "mermaid",
		Code: []string{"flowchart LR", "A[Start] --> B[Finish]"},
	}}}
	page := rich.Layout(doc, rich.LayoutOptions{Width: 80, Fonts: fonts, Palette: palette, Diagram: renderer.Box})
	var found bool
	for _, item := range page.Items {
		if item.Kind == rich.ItemBox && item.Box != nil {
			found = true
		}
	}
	if !found {
		t.Fatal("rich.Layout did not place the diagram Box")
	}
	tile := (&rich.Raster{Page: page, Fonts: fonts, Palette: palette}).Tile(0, page.Height)
	if tile == nil || tile.Bounds().Empty() {
		t.Fatal("rich.Raster produced no diagram tile")
	}
}

func TestCoreNilRendererAndEmptyBlock(t *testing.T) {
	var nilRenderer *Renderer
	if nilRenderer.Box(rich.Block{}) != nil {
		t.Fatal("nil renderer returned a box")
	}
	if New(nil, testPalette()).Box(rich.Block{}) != nil {
		t.Fatal("renderer without fonts returned a box")
	}
	if box := New(testFonts(t), testPalette()).Box(rich.Block{}); box == nil || box.Draw == nil {
		t.Fatal("empty block did not use the source fallback")
	}
}

func TestCoreRoundedRectangleIsVerticallySymmetric(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	fill := color.RGBA{R: 20, G: 40, B: 60, A: 255}
	stroke := color.RGBA{R: 220, G: 220, B: 220, A: 255}
	drawRounded(img, 2, 2, 9, 5, 2, fill, stroke)
	for y := 0; y < 2; y++ {
		for x := 0; x < 9; x++ {
			top := img.RGBAAt(2+x, 2+y)
			bottom := img.RGBAAt(2+x, 2+4-y)
			if top != bottom {
				t.Fatalf("rounded rectangle is not symmetric at column %d: top %v, bottom %v", x, top, bottom)
			}
		}
	}
}

func TestCoreScenePrimitivesAreDeterministic(t *testing.T) {
	fonts := testFonts(t)
	p := testPalette()
	s := Scene{Width: 40, Height: 20}
	s.Rect(1, 1, 6, 4, p.CodeBg, p.Rule)
	s.Rounded(8, 1, 6, 4, 1, p.CodeBg, p.Rule)
	s.Ellipse(15, 1, 5, 4, p.CodeBg, p.Rule)
	s.Diamond(21, 1, 6, 4, p.CodeBg, p.Rule)
	s.Hexagon(28, 1, 6, 4, p.CodeBg, p.Rule)
	s.Parallelogram(35, 1, 5, 4, 1, p.CodeBg, p.Rule)
	s.Arrow(1, 7, 12, 7, p.Rule, EdgeSolid, ArrowFilled, ArrowNone)
	s.Arrow(14, 7, 25, 7, p.Rule, EdgeDotted, ArrowOpen, ArrowCircle)
	s.Arrow(27, 7, 38, 7, p.Rule, EdgeThick, ArrowCross, ArrowCross)
	s.Sector(1, 10, 8, 8, 0, math.Pi*1.5, p.Heading[0])
	s.Polyline([]Point{{X: 12, Y: 18}, {X: 20, Y: 10}, {X: 28, Y: 18}}, p.Rule, false, false)
	s.Badge("badge", 31, 11, 8, 3, fonts.BodyPx, p.Heading[1], p.Fg)
	first, second := renderScene(s, fonts), renderScene(s, fonts)
	if first == nil || second == nil || !bytes.Equal(first.Pix, second.Pix) {
		t.Fatal("scene rendering is nil or non-deterministic")
	}
}

func TestCoreRendererCacheHitAndByteBound(t *testing.T) {
	const kind Kind = "test-core-cache"
	t.Cleanup(func() { unregister(kind) })
	register(kind, func(lines []string) (Diagram, error) {
		return Diagram{Kind: kind, Title: lines[0]}, nil
	}, func(Diagram, *rich.Fonts, rich.Palette) (Scene, error) {
		s := Scene{Width: 80, Height: 80}
		s.Rect(0, 0, 80, 80, color.RGBA{R: 10, G: 20, B: 30, A: 255}, color.RGBA{})
		return s, nil
	})
	fonts := testFonts(t)
	r := New(fonts, testPalette())
	source := func(label string) string { return "test-core-cache\n" + label }
	first := r.render(source("first"))
	if second := r.render(source("first")); second != first {
		t.Fatal("cache hit returned a different bitmap")
	}
	for i := 0; i < maxCachedDiagrams; i++ {
		r.render(source(fmt.Sprintf("entry-%d", i)))
	}
	if len(r.cache) > maxCachedDiagrams {
		t.Fatalf("cache entries = %d, want <= %d", len(r.cache), maxCachedDiagrams)
	}
	if r.cacheBytes > maxCachedBitmapBytes {
		t.Fatalf("cache bytes = %d, want <= %d", r.cacheBytes, maxCachedBitmapBytes)
	}
	if _, ok := r.cache[source("first")]; ok {
		t.Fatal("oldest cache entry was not evicted")
	}
}

func TestCoreFallbackBoundsLongSource(t *testing.T) {
	fonts := testFonts(t)
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = fmt.Sprintf("source line %d", i)
	}
	s := fallbackScene(strings.Join(lines, "\n"), errors.New("unsupported construct"), fonts, testPalette())
	if s.Height > maxFallbackSourceLines+20 {
		t.Fatalf("fallback height = %d, want bounded source panel", s.Height)
	}
	foundNote := false
	for _, cmd := range s.Commands {
		if cmd.Kind == CommandText && strings.Contains(cmd.Text, "truncated") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatal("bounded fallback does not say that source was truncated")
	}
}

func TestCoreFallbackContainsCauseAndSource(t *testing.T) {
	fonts := testFonts(t)
	s := fallbackScene("flowchart LR\nA --> B", errors.New("unsupported construct \"edge\""), fonts, testPalette())
	var text strings.Builder
	for _, cmd := range s.Commands {
		if cmd.Kind == CommandText {
			text.WriteString(cmd.Text)
			text.WriteByte('\n')
		}
	}
	got := text.String()
	for _, want := range []string{"unsupported construct \"edge\"", "flowchart LR", "A --> B"} {
		if !strings.Contains(got, want) {
			t.Fatalf("fallback text does not contain %q:\n%s", want, got)
		}
	}
}

func TestCoreBoxDrawHonorsOriginAndClip(t *testing.T) {
	const kind Kind = "test-core-clip"
	t.Cleanup(func() { unregister(kind) })
	register(kind, func([]string) (Diagram, error) { return Diagram{Kind: kind}, nil }, func(Diagram, *rich.Fonts, rich.Palette) (Scene, error) {
		s := Scene{Width: 4, Height: 2}
		s.Rect(0, 0, 4, 2, color.RGBA{R: 20, G: 80, B: 160, A: 255}, color.RGBA{})
		return s, nil
	})
	fonts := testFonts(t)
	box := New(fonts, testPalette()).Box(rich.Block{Code: []string{"test-core-clip"}})
	box.W = 2
	img := image.NewRGBA(image.Rect(0, 0, 8*fonts.CellW, 4*fonts.CellH))
	box.Draw(img, 2*fonts.CellW, fonts.CellH)
	if got := img.RGBAAt(2*fonts.CellW, fonts.CellH); got != (color.RGBA{R: 20, G: 80, B: 160, A: 255}) {
		t.Fatalf("pixel at draw origin = %v", got)
	}
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			inside := x >= 2*fonts.CellW && x < 4*fonts.CellW && y >= fonts.CellH && y < 3*fonts.CellH
			if !inside && img.RGBAAt(x, y).A != 0 {
				t.Fatalf("paint escaped clip at (%d,%d)", x, y)
			}
		}
	}
}

func TestCoreSceneBudgetRejectsOversizeLayout(t *testing.T) {
	if err := validateScene(Scene{Width: maxSceneCells + 1, Height: 1}, testFonts(t)); err == nil {
		t.Fatal("validateScene accepted a scene over the cell budget")
	}
}

func TestCoreConstructErrorLineIncludesLeadingLines(t *testing.T) {
	const kind Kind = "test-core-lines"
	t.Cleanup(func() { unregister(kind) })
	register(kind, func(lines []string) (Diagram, error) {
		return Diagram{}, unknown(kind, 2, "bad body")
	}, func(Diagram, *rich.Fonts, rich.Palette) (Scene, error) { return Scene{}, nil })
	_, err := Parse("\n%% comment\ntest-core-lines\nbad body")
	var constructErr *ConstructError
	if !errors.As(err, &constructErr) {
		t.Fatalf("error = %v, want *ConstructError", err)
	}
	if constructErr.Line != 4 {
		t.Fatalf("error line = %d, want 4", constructErr.Line)
	}
}
