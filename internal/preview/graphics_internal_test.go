package preview

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/pwn27sbx/misoku/internal/kitty"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// collectSink records what the graphics path sends.
type collectSink struct{ seqs []string }

func (c *collectSink) Send(seqs ...string) error {
	c.seqs = append(c.seqs, seqs...)
	return nil
}

func writePNG(t *testing.T, dir string, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 128, 255})
		}
	}
	path := filepath.Join(dir, "big.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

// A picture over the transmit cap must not stall the pane: it falls back to the
// half-block drawing, which is exactly today's output.
func TestGraphicsFallsBackWhenThePictureIsTooLargeToSend(t *testing.T) {
	old := maxGraphicsBytes
	maxGraphicsBytes = 16
	defer func() { maxGraphicsBytes = old }()

	s := &collectSink{}
	SetImageMode(kitty.Graphics)
	SetImageSink(s)
	defer func() {
		SetImageMode(kitty.Blocks)
		SetImageSink(nil)
		ClearImages()
	}()

	dir := t.TempDir()
	writePNG(t, dir, 40, 20)
	lines, err := renderImage(filepath.Join(dir, "big.png"), 20, theme.DefaultTheme().Bg)
	if err != nil {
		t.Fatalf("renderImage: %v", err)
	}
	if len(s.seqs) != 0 {
		t.Fatalf("an oversized picture was sent anyway: %q", s.seqs)
	}
	if !strings.Contains(strings.Join(lines, "\n"), "▀") {
		t.Fatal("expected half blocks for an oversized picture")
	}
}

// Dimming greys the text of a pane without the focus; a picture keeps its
// colours, so its rows are left exactly as they are.
func TestDimLeavesPicturesAlone(t *testing.T) {
	line := kitty.Placeholders(9, 3, 1)[0]
	if got := dim(line, lipgloss.Color("#808080")); got != line {
		t.Fatalf("dim rewrote a placeholder row:\n%q\n%q", line, got)
	}
}
