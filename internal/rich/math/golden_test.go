package math

import (
	"bytes"
	"flag"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/rich"
)

var update = flag.Bool("update", false, "rewrite the golden images")

// goldenGroups pin what each construct family draws. The maths font is
// bundled, so the pixels are identical on Linux, macOS and Windows; a change
// here is a visible change and must be looked at. Regenerate with the Go
// toolchain go.mod names (GOTOOLCHAIN=go1.26.6), or the pixels differ from
// CI's and every golden fails there: a newer local toolchain changes the
// rasteriser's rounding.
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
			if !bytes.Equal(got, want) {
				t.Fatalf("%s: pixels changed; look at the new image, then run -update if it is right", path)
			}
		})
	}
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
