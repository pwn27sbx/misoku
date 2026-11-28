package rich

import (
	"os"
	"testing"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// TestRenderSamplePNG renders the defect sample to the path in RICH_SAMPLE_PNG
// so the fix can be looked at. It is a development aid, not part of the suite.
func TestRenderSamplePNG(t *testing.T) {
	out := os.Getenv("RICH_SAMPLE_PNG")
	if out == "" {
		t.Skip("set RICH_SAMPLE_PNG to render")
	}
	f, err := LoadFonts(FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
	if err != nil {
		t.Fatal(err)
	}
	src := "# Heading one\n\nSome **bold text** and a [[wiki link]] in a paragraph that wraps across the page width, then *italic* and `code`.\n\n## Second heading\n\n- item one\n- item two\n\n```go\nx := 10\n```\n"
	doc := Parse([]byte(src))
	pal := PaletteFromTheme(theme.DefaultTheme())
	page := Layout(doc, LayoutOptions{Width: 36, Fonts: f, Palette: pal})
	img := (&Raster{Page: page, Fonts: f, Palette: pal}).Tile(0, page.Height)
	if err := os.WriteFile(out, EncodePNG(img), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%dx%d, %d rows)", out, img.Bounds().Dx(), img.Bounds().Dy(), page.Height)
}
