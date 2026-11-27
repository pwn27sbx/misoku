package math

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/rich"
)

var update = flag.Bool("update", false, "rewrite the golden images")

// goldenGroups pin what each construct family draws. The maths font is
// bundled, so the pictures agree on Linux, macOS and Windows; a change here is
// a visible change and must be looked at. Regenerate with the Go toolchain
// go.mod names (GOTOOLCHAIN=go1.26.6), or the pixels differ from CI's: a newer
// toolchain changes the rasteriser's rounding.
var goldenGroups = map[string]string{
	"fractions": `\frac{a}{b} \quad \dfrac{a+b}{c+d} \quad \binom{n}{k} \quad \cfrac{1}{1+\frac{1}{x}}`,
	"roots":     `\sqrt{x} \quad \sqrt[3]{x+1} \quad \sqrt{\frac{a}{b}}`,
	"scripts":   `x^2 \quad x_i \quad x_i^2 \quad x_{i+1}^{n-1} \quad \sum_{i=1}^{n} i \quad \int_0^1 f(x)\,dx`,
	"fences":    `\left(\frac{a}{b}\right) \quad \left[\frac{a}{b}\right] \quad \left\{\frac{a}{b}\right\} \quad \Big| \quad \lVert x \rVert`,
	"matrices":  `\begin{pmatrix} a & b \\ c & d \end{pmatrix} \quad \begin{cases} 0 & \text{if } x \\ 1 & \text{else} \end{cases}`,
	"accents":   `\hat{x} \quad \tilde{x} \quad \vec{AB} \quad \overline{xyz} \quad \overrightarrow{AB} \quad \underbrace{a+b}_{n}`,
	"alphabets": `\mathbb{R} \quad \mathcal{L} \quad \mathfrak{g} \quad \mathsf{T} \quad \mathbf{A} \quad \boldsymbol{\beta} \quad \mathtt{code}`,
	"aligned":   `f(x)=\begin{aligned} a &= b \\ c &= d \end{aligned} \quad \text{with } x>0`,
	"arrows":    `\xrightarrow[below]{above} \quad \overset{a}{=} \quad \not= \quad \xleftarrow{n}`,
	"symbols":   `\forall x \in \mathbb{R}: \nabla f(x) = 0 \quad \alpha \leq \beta \quad A \cup B \subseteq C \quad \infty`,
	"document":  `\frac{\partial L}{\partial W_j} = -\frac{1}{n}\sum_{i=0}^{n-1} X_{ij}\,\left(Y_i - X_i W\right)`,
}

func TestMathGolden(t *testing.T) {
	dir := filepath.Join("testdata", "golden")
	if *update {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, src := range goldenGroups {
		name, src := name, src
		t.Run(name, func(t *testing.T) {
			ResetCache()
			box := Box([]string{src}, 900, testConfig())
			if box == nil {
				t.Fatalf("%s: no box", src)
			}
			got := renderBox(t, box)
			path := filepath.Join(dir, name+".png")
			if *update {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("wrote %s (%d bytes)", path, len(got))
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (run go test -run TestMathGolden -update)", err)
			}
			if err := goldenClose(want, got); err != nil {
				t.Fatalf("%s: %v; look at the new image, then run -update if it is right", path, err)
			}
		})
	}
}

// goldenClose compares two formula PNGs. Exact bytes cannot hold across
// architectures: arm64 and amd64 round the rasteriser's antialiasing a little
// differently, so a few edge pixels differ by one or two levels. The pictures
// must still be the same size, cover the same box, carry the same amount of
// ink, and no pixel may differ by more than a rounding step.
func goldenClose(want, got []byte) error {
	w, err := png.Decode(bytes.NewReader(want))
	if err != nil {
		return fmt.Errorf("golden is not a PNG: %w", err)
	}
	g, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		return fmt.Errorf("new image is not a PNG: %w", err)
	}
	if !w.Bounds().Eq(g.Bounds()) {
		return fmt.Errorf("size %v, want %v", g.Bounds().Size(), w.Bounds().Size())
	}
	bw, iw := inkBox(w), inkBox(g)
	if bw != iw {
		return fmt.Errorf("ink covers %v, want %v", iw, bw)
	}
	bounds := w.Bounds()
	total := bounds.Dx() * bounds.Dy()
	wantInk, gotInk, differing := 0, 0, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			wr, wg, wb, wa := w.At(x, y).RGBA()
			gr, gg, gb, ga := g.At(x, y).RGBA()
			if wa|wg|wb != 0 {
				wantInk++
			}
			if ga|gg|gb != 0 {
				gotInk++
			}
			delta := 0
			for _, d := range []int{absDiff(wr, gr), absDiff(wg, gg), absDiff(wb, gb), absDiff(wa, ga)} {
				if d > delta {
					delta = d
				}
			}
			if delta == 0 {
				continue
			}
			// RGBA() is 16 bits per channel.
			if delta > 32*257 {
				return fmt.Errorf("pixel %d,%d differs by %d levels", x, y, delta/257)
			}
			differing++
		}
	}
	if margin := max(wantInk/50, 40); gotInk < wantInk-margin || gotInk > wantInk+margin {
		return fmt.Errorf("ink %d pixels, want %d", gotInk, wantInk)
	}
	if differing > max(total/20, 200) {
		return fmt.Errorf("%d pixels differ, more than antialiasing explains", differing)
	}
	return nil
}

// inkBox is the bounding box of every non-transparent pixel, or the empty
// rectangle for a blank image.
func inkBox(img image.Image) image.Rectangle {
	b := img.Bounds()
	first := image.Point{X: -1, Y: -1}
	last := image.Point{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a == 0 {
				continue
			}
			if first.X < 0 {
				first = image.Point{X: x, Y: y}
			}
			last = image.Point{X: x, Y: y}
		}
	}
	if first.X < 0 {
		return image.Rectangle{}
	}
	return image.Rect(first.X, first.Y, last.X+1, last.Y+1)
}

func absDiff(a, b uint32) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// renderBox draws a box the way the page's raster does, so the golden covers
// the same bytes the terminal receives.
func renderBox(t *testing.T, box *rich.Box) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, box.W*12, box.H*24))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{16, 16, 16, 255}), image.Point{}, draw.Src)
	box.Draw(img, 0, 0)
	return rich.EncodePNG(img)
}

func TestMathBoxCache(t *testing.T) {
	cfg := testConfig()
	src := []string{`\frac{a}{b}`}
	ResetCache()
	first := Box(src, 400, cfg)
	second := Box(src, 400, cfg)
	if first == nil || second == nil {
		t.Fatal("no box")
	}
	if first != second {
		t.Fatal("same formula, same size: Box should reuse the cached image")
	}
	other := Box(src, 200, cfg)
	if other == first {
		t.Fatal("a different width must not reuse the cached layout")
	}
}

// The comparison itself is pinned: identical pictures pass, antialiasing
// rounding passes, and a real change (a missing glyph, a shift, a size) fails.
func TestGoldenClose(t *testing.T) {
	// NRGBA keeps the values the PNG really stores, so a perturbation survives
	// the round trip instead of being re-premultiplied.
	base := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	draw.Draw(base, image.Rect(5, 4, 20, 16), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	encode := func(img image.Image) []byte {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	want := encode(base)
	if err := goldenClose(want, want); err != nil {
		t.Fatalf("the same picture failed: %v", err)
	}

	// One antialiased edge pixel a level off.
	near := cloneNRGBA(base)
	near.SetNRGBA(5, 8, color.NRGBA{255, 255, 255, 253})
	near.SetNRGBA(19, 4, color.NRGBA{253, 255, 255, 255})
	if err := goldenClose(want, encode(near)); err != nil {
		t.Fatalf("a rounding difference failed: %v", err)
	}

	// A glyph gone: less ink, a different box.
	less := image.NewNRGBA(base.Bounds())
	draw.Draw(less, image.Rect(12, 4, 20, 16), image.NewUniform(color.NRGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	if err := goldenClose(want, encode(less)); err == nil {
		t.Fatal("a missing glyph passed")
	}
	// A different size.
	if err := goldenClose(want, encode(image.NewNRGBA(image.Rect(0, 0, 41, 20)))); err == nil {
		t.Fatal("a different size passed")
	}
	// The whole picture shifted one pixel.
	shifted := image.NewNRGBA(base.Bounds())
	draw.Draw(shifted, shifted.Bounds(), base, image.Pt(-1, 0), draw.Src)
	if err := goldenClose(want, encode(shifted)); err == nil {
		t.Fatal("a shifted picture passed")
	}
}

func cloneNRGBA(src *image.NRGBA) *image.NRGBA {
	dst := image.NewNRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}
