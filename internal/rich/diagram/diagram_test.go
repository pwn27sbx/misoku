package diagram

import (
	"image"
	"image/color"
	"testing"

	"github.com/pwn27sbx/misoku/internal/rich"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

func testFonts(t testing.TB) *rich.Fonts {
	t.Helper()
	f, err := rich.NewFonts(goregular.TTF, gomono.TTF, rich.FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestDiagramRegistryAndCachedDraw(t *testing.T) {
	const kind Kind = "test-cached"
	register(kind, func(lines []string) (Diagram, error) {
		return Diagram{Kind: kind, Title: "cached"}, nil
	}, func(d Diagram, fonts *rich.Fonts, palette rich.Palette) (Scene, error) {
		s := Scene{Width: 4, Height: 2}
		s.Rect(0, 0, 4, 2, palette.Fg, palette.Fg)
		return s, nil
	})

	fonts := testFonts(t)
	r := New(fonts, rich.Palette{Fg: color.RGBA{R: 20, G: 80, B: 160, A: 255}})
	box := r.Box(rich.Block{Code: []string{"test-cached"}})
	if box == nil || box.W != 4 || box.H != 2 {
		t.Fatalf("Box = %#v, want 4x2 cells", box)
	}

	// Layout may clamp the public width after the engine creates the box. Draw
	// must observe that clamp rather than painting beyond the contracted box.
	box.W = 2
	img := image.NewRGBA(image.Rect(0, 0, 4*fonts.CellW, 2*fonts.CellH))
	box.Draw(img, 0, 0)
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 2 * fonts.CellW; x < img.Bounds().Dx(); x++ {
			if img.RGBAAt(x, y).A != 0 {
				t.Fatalf("pixel (%d,%d) lies beyond the clamped box", x, y)
			}
		}
	}
}

func TestDiagramUnknownConstructFallsBackToSource(t *testing.T) {
	fonts := testFonts(t)
	r := New(fonts, rich.Palette{Fg: color.RGBA{R: 220, G: 220, B: 220, A: 255}, Rule: color.RGBA{R: 90, G: 90, B: 90, A: 255}})
	box := r.Box(rich.Block{Kind: rich.KindDiagram, Lang: "mermaid", Code: []string{
		"sequenceDiagram", "participant A", "A ?!> B: unsupported arrow",
	}})
	if box == nil || box.W <= 0 || box.H <= 0 || box.Draw == nil {
		t.Fatalf("Box = %#v, want a drawable source fallback", box)
	}
	img := image.NewRGBA(image.Rect(0, 0, box.W*fonts.CellW, box.H*fonts.CellH))
	box.Draw(img, 0, 0)
	painted := false
	for i := range img.Pix {
		if img.Pix[i] != 0 {
			painted = true
			break
		}
	}
	if !painted {
		t.Fatal("fallback drew an empty image")
	}
}

func TestDiagramRejectsUnknownType(t *testing.T) {
	_, err := Parse("notMermaid\nA --> B")
	if err == nil {
		t.Fatal("Parse accepted an unknown diagram type")
	}
}
