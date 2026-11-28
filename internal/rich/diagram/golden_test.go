package diagram

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/rich"
	"github.com/pwn27sbx/misoku/internal/rich/testgolden"
)

var updateGoldens = flag.Bool("update", false, "update diagram golden PNGs")

func testPalette() rich.Palette {
	return rich.Palette{
		Fg:     color.RGBA{R: 230, G: 230, B: 230, A: 255},
		Muted:  color.RGBA{R: 150, G: 150, B: 150, A: 255},
		Link:   color.RGBA{R: 90, G: 160, B: 240, A: 255},
		CodeBg: color.RGBA{R: 28, G: 30, B: 36, A: 255},
		Rule:   color.RGBA{R: 90, G: 95, B: 105, A: 255},
		Heading: [6]color.RGBA{
			{R: 90, G: 160, B: 240, A: 255},
			{R: 180, G: 120, B: 220, A: 255},
			{R: 100, G: 190, B: 130, A: 255},
			{R: 230, G: 190, B: 90, A: 255},
			{R: 230, G: 140, B: 90, A: 255},
			{R: 230, G: 100, B: 110, A: 255},
		},
	}
}

func renderSource(t testing.TB, source string) *image.RGBA {
	t.Helper()
	fonts := testFonts(t)
	box := New(fonts, testPalette()).Box(rich.Block{Kind: rich.KindDiagram, Lang: "mermaid", Code: splitSourceLines(source)})
	if box == nil || box.W <= 0 || box.H <= 0 || box.Draw == nil {
		t.Fatal("renderer returned no drawable box")
	}
	img := image.NewRGBA(image.Rect(0, 0, box.W*fonts.CellW, box.H*fonts.CellH))
	box.Draw(img, 0, 0)
	return img
}

func assertGolden(t testing.TB, name string, img *image.RGBA) {
	t.Helper()
	path := filepath.Join("testdata", filepath.FromSlash(name)+".png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if *updateGoldens {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create it)", path, err)
	}
	if err := testgolden.Close(want, buf.Bytes()); err != nil {
		t.Fatalf("golden %s changed (%v); run go test ./internal/rich/diagram -update and inspect the diff", path, err)
	}
}

func splitSourceLines(source string) []string {
	var lines []string
	for _, line := range bytes.Split([]byte(source), []byte("\n")) {
		lines = append(lines, string(line))
	}
	return lines
}
