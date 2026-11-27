package rich

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"
)

func testPage(t *testing.T) (*Page, *Fonts) {
	t.Helper()
	f := testFonts(t)
	doc := Parse([]byte("# Title\n\nSome text that draws pixels.\n\n```go\nx := 1\n```\n"))
	p := Layout(doc, LayoutOptions{Width: 40, Fonts: f, Palette: testPalette()})
	return p, f
}

func TestRasterTileSizeAndBackground(t *testing.T) {
	p, f := testPage(t)
	r := &Raster{Page: p, Fonts: f, Palette: testPalette()}
	img := r.Tile(0, 10)
	if got := img.Bounds().Dx(); got != p.Width*f.CellW {
		t.Fatalf("tile width = %d, want %d", got, p.Width*f.CellW)
	}
	if got := img.Bounds().Dy(); got != 10*f.CellH {
		t.Fatalf("tile height = %d, want %d", got, 10*f.CellH)
	}
	bg := testPalette().Bg
	if got := img.RGBAAt(0, 0); got != bg {
		t.Fatalf("corner = %v, want the page background %v", got, bg)
	}
}

func TestRasterDrawsTextPixels(t *testing.T) {
	p, f := testPage(t)
	r := &Raster{Page: p, Fonts: f, Palette: testPalette()}
	img := r.Tile(0, p.Height)
	bg := testPalette().Bg
	changed := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if img.RGBAAt(x, y) != bg {
				changed++
			}
		}
	}
	if changed < 100 {
		t.Fatalf("only %d pixels differ from the background; the text did not draw", changed)
	}
}

func TestRasterIsDeterministic(t *testing.T) {
	p, f := testPage(t)
	r := &Raster{Page: p, Fonts: f, Palette: testPalette()}
	a := EncodePNG(r.Tile(0, p.Height))
	b := EncodePNG(r.Tile(0, p.Height))
	if !bytes.Equal(a, b) {
		t.Fatal("the same page rasterised to different bytes")
	}
	if _, err := png.Decode(bytes.NewReader(a)); err != nil {
		t.Fatalf("tile is not a PNG: %v", err)
	}
	if len(a) > 512*1024 {
		t.Fatalf("tile is %d bytes, over the 512 KB cap", len(a))
	}
}

func TestRasterSlicesThePage(t *testing.T) {
	p, f := testPage(t)
	r := &Raster{Page: p, Fonts: f, Palette: testPalette()}
	top := r.Tile(0, 4)
	bottom := r.Tile(4, 4)
	if bytes.Equal(EncodePNG(top), EncodePNG(bottom)) {
		t.Fatal("two different slices produced the same pixels")
	}
	_ = image.Rect
}

func TestRasterThinRuleIsTwoPixels(t *testing.T) {
	f := testFonts(t)
	p := Layout(Document{Blocks: []Block{{Kind: KindRule, Line: 0, End: 1}}},
		LayoutOptions{Width: 20, Fonts: f, Palette: testPalette()})
	r := &Raster{Page: p, Fonts: f, Palette: testPalette()}
	img := r.Tile(0, p.Height)
	rule := testPalette().Rule
	rows := 0
	for y := 0; y < img.Bounds().Dy(); y++ {
		if img.RGBAAt(pageMargin*f.CellW, y) == rule {
			rows++
		}
	}
	if rows != 2 {
		t.Fatalf("rule rows = %d, want 2", rows)
	}
}

// A maths or diagram engine hands the page a sized box; the raster calls its
// draw function at the tile's pixel origin, and the page never looks inside.
func TestEngineBoxDrawsAtItsOrigin(t *testing.T) {
	f := testFonts(t)
	pal := testPalette()
	doc := Document{Blocks: []Block{{Kind: KindMath, Code: []string{"x^2"}, Line: 0, End: 1}}}
	drawn := false
	p := Layout(doc, LayoutOptions{Width: 30, Fonts: f, Palette: pal,
		Math: func(b Block) *Box {
			return &Box{W: 10, H: 3, Draw: func(dst draw.Image, x, y int) {
				drawn = x > 0 && y == 0
				draw.Draw(dst, image.Rect(x, y, x+2, y+2), image.NewUniform(color.RGBA{255, 0, 0, 255}), image.Point{}, draw.Src)
			}}
		}})
	var box *Item
	for i := range p.Items {
		if p.Items[i].Kind == ItemBox {
			box = &p.Items[i]
		}
	}
	if box == nil {
		t.Fatal("the layout did not place the engine's box")
	}
	if box.W != 10 || box.H != 3 {
		t.Fatalf("box = %dx%d cells, want 10x3", box.W, box.H)
	}
	img := (&Raster{Page: p, Fonts: f, Palette: pal}).Tile(box.Y, box.H)
	if !drawn {
		t.Fatal("the raster did not call the box's draw function")
	}
	if got := img.RGBAAt(box.X*f.CellW, 0); got != (color.RGBA{255, 0, 0, 255}) {
		t.Fatalf("pixel at the box origin = %v, want red", got)
	}
}
