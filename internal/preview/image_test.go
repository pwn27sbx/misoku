package preview_test

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

func TestLocalImagesAreDrawn(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 80, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 3), uint8(y * 6), 200, 255})
		}
	}
	_ = os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	f, _ := os.Create(filepath.Join(dir, "assets", "my pic.png"))
	_ = png.Encode(f, img)
	f.Close()

	out := preview.CompileIn("![A sunset](<assets/my pic.png>)\n\n![gone](missing.png)\n\n![web](https://x/y.png)", theme.DefaultTheme(), 40, dir)
	plain := ansi.Strip(out)
	if n := strings.Count(plain, "▀"); n != 40*10 {
		t.Fatalf("expected a 40x10 half-block picture, got %d blocks:\n%s", n, plain)
	}
	if !strings.Contains(out, "38;2;") {
		t.Fatal("expected true-colour cells")
	}
	for _, want := range []string{"A sunset", "gone (not found)", "web (remote image)"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q in:\n%s", want, plain)
		}
	}
	for _, l := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(l); w > 40 {
			t.Fatalf("line wider than pane: %d", w)
		}
	}
}

func TestCoverBannerFromFrontMatter(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	f, _ := os.Create(filepath.Join(dir, "cover.png"))
	_ = png.Encode(f, img)
	f.Close()

	src := "---\ncover: cover.png\nicon: ☁️\n---\n# GCP- ACE\n"
	out := preview.CompileIn(src, theme.DefaultTheme(), 50, dir)
	plain := ansi.Strip(out)
	if strings.Contains(plain, "cover:") || strings.Contains(plain, "---") {
		t.Fatalf("front matter must not be shown as text:\n%s", plain)
	}
	lines := strings.Split(plain, "\n")
	for i := 0; i < 8; i++ {
		if lines[i] != strings.Repeat("▀", 50) {
			t.Fatalf("banner row %d should span the pane: %q", i, lines[i])
		}
	}
	if !strings.Contains(plain, "☁️") || !strings.Contains(plain, "GCP- ACE") {
		t.Fatalf("expected icon and heading after the banner:\n%s", plain)
	}

	meta, body := preview.SplitFrontMatter("---\nnot closed\n# x")
	if meta != nil || !strings.HasPrefix(body, "---") {
		t.Fatal("unterminated front matter must be left alone")
	}
}
