package math

import (
	"image"
	"image/color"
	"math"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/rich"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// constructCorpus is the acceptance list from the plan (§3.6.2): one case per
// construct a viewer has to support, written as a reader would write it.
var constructCorpus = []string{
	// Fractions and binomials.
	`\frac{a}{b}`, `\dfrac{a}{b}`, `\tfrac{a}{b}`, `\cfrac{a}{b+c}`,
	`\binom{n}{k}`, `\dbinom{n}{k}`, `\tbinom{n}{k}`,
	// Roots.
	`\sqrt{x}`, `\sqrt[3]{x+1}`, `\sqrt{\frac{a}{b}}`,
	// Text and operator names.
	`\text{if } x>0`, `\mbox{plain}`, `\textrm{roman}`, `\textnormal{normal}`,
	`\textup{up}`, `\textit{italic}`, `\textbf{bold}`,
	`\operatorname{softmax}(x)`, `\operatorname*{arg\,max}_x f(x)`,
	// Alphabets.
	`\mathrm{d}`, `\mathbf{A}`, `\boldsymbol{\beta}`, `\bm{x}`,
	`\mathbb{R}`, `\mathcal{L}`, `\mathscr{M}`, `\mathit{f}`,
	`\mathfrak{g}`, `\mathsf{T}`, `\mathtt{code}`,
	// Delimiters and fences.
	`\left( \frac{a}{b} \right)`, `\left[ x \right]`, `\left\{ x \right\}`,
	`\left. \frac{df}{dx} \right|_{x=0}`, `\left\langle x \right\rangle`,
	`\left\lVert x \right\rVert`, `\left\lfloor x \right\rfloor`,
	`\left\lceil x \right\rceil`, `\Big|`, `\big( x \big)`, `\Bigg[ y \Bigg]`,
	// Environments.
	`\begin{array}{lcr} a & b & c \\ d & e & f \end{array}`,
	`\begin{subarray}{c} a \\ b \end{subarray}`,
	`\begin{aligned} a &= b \\ c &= d \end{aligned}`,
	`\begin{align} a &= b \\ c &= d \end{align}`,
	`\begin{align*} a &= b \\ c &= d \end{align*}`,
	`\begin{alignedat}{2} a &= b & c &= d \end{alignedat}`,
	`\begin{split} a &= b \\ c &= d \end{split}`,
	`\begin{eqnarray} a &=& b \end{eqnarray}`,
	`\begin{cases} 0 & \text{if } x \\ 1 & \text{else} \end{cases}`,
	`\begin{matrix} a & b \\ c & d \end{matrix}`,
	`\begin{pmatrix} a & b \\ c & d \end{pmatrix}`,
	`\begin{bmatrix} a & b \\ c & d \end{bmatrix}`,
	`\begin{Bmatrix} a & b \\ c & d \end{Bmatrix}`,
	`\begin{vmatrix} a & b \\ c & d \end{vmatrix}`,
	`\begin{Vmatrix} a & b \\ c & d \end{Vmatrix}`,
	// Accents.
	`\hat{x}`, `\widehat{xyz}`, `\tilde{x}`, `\widetilde{xyz}`, `\vec{v}`,
	`\dot{x}`, `\ddot{x}`, `\bar{x}`, `\overline{xyz}`, `\underline{xyz}`,
	`\overrightarrow{AB}`, `\overleftarrow{AB}`, `\overleftrightarrow{AB}`,
	`\overbrace{a+b}^{n}`, `\underbrace{a+b}_{n}`, `\check{x}`, `\breve{x}`,
	`\acute{x}`, `\grave{x}`,
	// Big operators and their limits.
	`\sum_{i=1}^{n} i`, `\prod_{i=1}^{n} i`, `\coprod_{i} i`,
	`\bigcup_{i} A_i`, `\bigcap_{i} A_i`, `\bigoplus_{i} A_i`,
	`\bigotimes_{i} A_i`, `\int_0^1 f(x)\,dx`, `\iint_D f`,
	`\iiint_V f`, `\oint_C f`,
	// Spacing.
	`a\,b`, `a\;b`, `a\:b`, `a\!b`, `a\quad b`, `a\qquad b`, `a\ b`, `a\space b`,
	// Class overrides.
	`\mathbin{x}`, `\mathrel{x}`, `\mathord{x}`, `\mathop{x}`,
	`\mathopen{x}`, `\mathclose{x}`, `\mathpunct{x}`,
	// Over/under stacks and arrows.
	`\overset{a}{b}`, `\stackrel{a}{=}`, `\underset{a}{b}`,
	`\xrightarrow{a}`, `\xrightarrow[below]{above}`, `\xleftarrow{a}`,
	// Styles.
	`\displaystyle \frac{a}{b}`, `\textstyle \frac{a}{b}`,
	`\scriptstyle \frac{a}{b}`, `\scriptscriptstyle \frac{a}{b}`,
	// Functions and symbols.
	`\sin x`, `\cos x`, `\tan x`, `\log x`, `\ln x`, `\exp x`,
	`\lim_{x \to 0} f(x)`, `\max_{i} x_i`, `\min_{i} x_i`,
	`\det A`, `\gcd(a,b)`, `\sup_{n} x_n`, `\inf_{n} x_n`,
	`\mod n`, `\bmod n`, `f \colon A \to B`,
	`\not=`, `\not\in`, `\forall x \in \mathbb{R}`, `\exists y`,
	`\nabla f`, `\partial_t u = \Delta u`, `\infty`, `\emptyset`,
	`\alpha\beta\gamma\delta\epsilon\varepsilon\zeta\eta\theta\vartheta`,
	`\Gamma\Delta\Theta\Lambda\Xi\Pi\Sigma\Upsilon\Phi\Psi\Omega`,
	`x \in A \cup B \cap C`, `A \subseteq B \subsetneq C`,
	`a \leq b \geq c \neq d \approx e \equiv f`,
	`a \ll b \gg c \sim d \simeq e \propto f \perp g \parallel h`,
	`A \to B \leftarrow C \Rightarrow D \Leftrightarrow E \mapsto F`,
	`\longrightarrow \longleftarrow \Longrightarrow \Longleftrightarrow`,
	`\otimes \oplus \odot \times \div \cdot \pm \mp`,
	`\lfloor x \rfloor \lceil y \rceil \langle z \rangle`,
	`\dots \cdots \vdots \ddots \ldots`,
}

func TestMathCorpus(t *testing.T) {
	cfg := testConfig()
	for _, src := range constructCorpus {
		src := src
		t.Run(mustName(src), func(t *testing.T) {
			e, err := Parse(src)
			if err != nil {
				t.Fatalf("%s: %v", src, err)
			}
			if len(e.Unknown) > 0 {
				t.Fatalf("%s: unknown commands %v", src, e.Unknown)
			}
			l := &layouter{base: cfg.SizePx}
			b := l.list(e.Nodes, cfg.SizePx, styleDisplay, 0)
			if b.w <= 0 || b.h+b.d <= 0 {
				t.Fatalf("%s: empty box %v x %v", src, b.w, b.h+b.d)
			}
			if math.IsNaN(b.w) || math.IsInf(b.w, 0) {
				t.Fatalf("%s: width %v", src, b.w)
			}
			if box := Box([]string{src}, 600, cfg); box == nil {
				t.Fatalf("%s: Box returned nil", src)
			}
		})
	}
}

// TestMathCorpusTable renders every symbol command on its own: if a name maps
// to a glyph with the wrong class or size, this is where it shows.
func TestMathCorpusTable(t *testing.T) {
	cfg := testConfig()
	names := make([]string, 0, len(symbols))
	for name := range symbols {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		src := `\` + name
		if _, err := Parse(src); err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		if b := Box([]string{src}, 400, cfg); b == nil {
			t.Fatalf("%s: no box", src)
		}
	}
}

func TestMathInlineBaseline(t *testing.T) {
	ink, err := LayoutInline(`x^2+\frac{1}{2}`, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if ink.Width <= 0 || ink.Ascent <= 0 || ink.Depth <= 0 {
		t.Fatalf("ink = %v x %v (+%v), want a real box", ink.Width, ink.Ascent, ink.Depth)
	}
	const baseline = 40
	img := image.NewRGBA(image.Rect(0, 0, int(ink.Width)+20, 80))
	ink.Draw(img, 10, baseline)
	minY, maxY := 80, 0
	for y := 0; y < 80; y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0 {
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if minY == 80 {
		t.Fatal("nothing was drawn")
	}
	// One pixel of tolerance for rounding in the rasteriser.
	if float64(baseline-minY) > ink.Ascent+1 {
		t.Fatalf("ink rises %d px, ascent is %v", baseline-minY, ink.Ascent)
	}
	if float64(maxY-baseline) > ink.Depth+1 {
		t.Fatalf("ink descends %d px, depth is %v", maxY-baseline, ink.Depth)
	}
}

func TestMathForms(t *testing.T) {
	src := []byte("# Title\n\nInline $x^2$ does not break the page.\n\n$$\\frac{a}{b}$$\n\n```math\n\\sum_{i=1}^n i\n```\n")
	doc := rich.Parse(src)
	var mathBlocks int
	for _, b := range doc.Blocks {
		if b.Kind == rich.KindMath {
			mathBlocks++
		}
	}
	if mathBlocks != 2 {
		t.Fatalf("parsed %d maths blocks, want 2 ($$ and ```math)", mathBlocks)
	}
	cfg := testConfig()
	page := rich.Layout(doc, rich.LayoutOptions{
		Width: 80, Fonts: testFonts(t), Palette: testPalette(),
		Math: func(b rich.Block) *rich.Box {
			return FromBlock(b, 900, cfg)
		},
	})
	var boxes int
	for _, it := range page.Items {
		if it.Kind == rich.ItemBox && it.Box != nil {
			boxes++
			if it.Box.W <= 0 || it.Box.H <= 0 || it.Box.Draw == nil {
				t.Fatalf("broken box: %+v", it.Box)
			}
		}
	}
	if boxes != 2 {
		t.Fatalf("page has %d maths boxes, want 2", boxes)
	}
	// The source mapping still points at the maths lines.
	if line := page.SourceLine(page.RowOf(4)); line < 3 || line > 6 {
		t.Fatalf("source line for the $$ block = %d", line)
	}
}

func TestMathUnknownRendersLiteral(t *testing.T) {
	cfg := testConfig()
	if b := Box([]string{`\zzz{x}`}, 300, cfg); b == nil {
		t.Fatal("unknown command must still draw literal text")
	}
}

// TestMathBoxWithinBudget is the plan's layout target: a formula must not
// cost more than 15 ms to lay out and rasterise.
func TestMathBoxWithinBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("timing test")
	}
	cfg := testConfig()
	src := []string{`\frac{\partial L}{\partial W_j} = -\frac{1}{n}\sum_{i=0}^{n-1} X_{ij}\,\left(Y_i - X_i W\right)`}
	const runs = 20
	var samples []time.Duration
	for i := 0; i < runs; i++ {
		ResetCache()
		start := time.Now()
		if b := Box(src, 900, cfg); b == nil {
			t.Fatal("no box")
		}
		samples = append(samples, time.Since(start))
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	median := samples[len(samples)/2]
	if median > 15*time.Millisecond {
		t.Fatalf("median layout %v, budget 15ms", median)
	}
	t.Logf("median layout %v", median)
}

func BenchmarkMathLayout(b *testing.B) {
	cfg := testConfig()
	src := []string{`\frac{\partial L}{\partial W_j} = -\frac{1}{n}\sum_{i=0}^{n-1} X_{ij}\,\left(Y_i - X_i W\right)`}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResetCache()
		Box(src, 900, cfg)
	}
}

func BenchmarkMathDraw(b *testing.B) {
	cfg := testConfig()
	src := []string{`\frac{\partial L}{\partial W_j} = -\frac{1}{n}\sum_{i=0}^{n-1} X_{ij}\,\left(Y_i - X_i W\right)`}
	box := Box(src, 900, cfg)
	dst := image.NewRGBA(image.Rect(0, 0, 900, 200))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		box.Draw(dst, 0, 0)
	}
}

func mustName(src string) string {
	r := strings.NewReplacer("/", "_", " ", "_", "\\", "", "{", "", "}", "", "(", "", ")", "", "&", "", "^", "sup", "_", "sub")
	name := r.Replace(src)
	if len(name) > 60 {
		name = name[:60]
	}
	return name
}

// testFonts and testPalette keep the integration test on the same test-only
// faces the rich package pins its goldens with.
func testFonts(t *testing.T) *rich.Fonts {
	t.Helper()
	f, err := rich.NewFonts(goregular.TTF, gomono.TTF, rich.FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func testPalette() rich.Palette {
	return rich.Palette{
		Fg: color.RGBA{200, 200, 200, 255}, Muted: color.RGBA{120, 120, 120, 255},
		Link: color.RGBA{80, 140, 255, 255}, CodeBg: color.RGBA{30, 30, 30, 255},
		Rule: color.RGBA{70, 70, 70, 255},
	}
}
