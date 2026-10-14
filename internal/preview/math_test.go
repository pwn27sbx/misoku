package preview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/theme"
)

func TestTexToUnicode(t *testing.T) {
	cases := map[string]string{
		`E = mc^2`:                             "E = mc²",
		`\alpha + \beta \leq \gamma`:           "α + β ≤ γ",
		`x_1, x_2, \ldots, x_n`:                "x₁, x₂, …, xₙ",
		`\sum_{i=1}^{n} x_i^2`:                 "∑ᵢ₌₁ⁿ xᵢ²",
		`\frac{a+b}{2}`:                        "(a + b)/2",
		`\frac{1}{\sqrt{2\pi}}`:                "1/√(2π)",
		`\sqrt{x+1}`:                           "√(x + 1)",
		`\sqrt[3]{8} = 2`:                      "∛8 = 2",
		`e^{i\pi} + 1 = 0`:                     "e^(iπ) + 1 = 0",
		`e^{2i} + 1 = 0`:                       "e²ⁱ + 1 = 0",
		`\int_0^1 x\,dx`:                       "∫₀¹ x dx",
		`x = \frac{-b \pm \sqrt{b^2-4ac}}{2a}`: "x = (−b ± √(b² − 4ac))/(2a)",
		`\frac{\rho}{\epsilon_0}`:              "ρ/ϵ₀",
		`f: X \to Y`:                           "f: X → Y",
		`a - b`:                                "a − b",
		`x^{-1}`:                               "x⁻¹",
		`f(x) = x^{\text{big}}`:                "f(x) = xᵇⁱᵍ",
		`a^{Q}`:                                "a^Q",
		`a_{xyz!}`:                             "a_(xyz!)",
		`\forall x \in \mathbb{R}`:             "∀x ∈ ℝ",
		`\lim_{x \to 0} \frac{\sin x}{x}`:      "lim_(x→0) (sin x)/x",
		`\vec{v}`:                              "v⃗",
		`a \\ b`:                               "a\nb",
		`\unknown{x}`:                          "unknownx",
		`\frac{1}`:                             "1/",
		`x^`:                                   "x",
		`\`:                                    "",
		`{{{`:                                  "",
	}
	for in, want := range cases {
		if got := texToUnicode(in); got != want {
			t.Errorf("texToUnicode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMathBlocks(t *testing.T) {
	th := theme.DefaultTheme()
	for _, md := range []string{
		"```math\nE = mc^2\n```",
		"$$E = mc^2$$",
		"$$\nE = mc^2\n$$",
		"$$ E = mc^2\n$$",
	} {
		out := ansi.Strip(Compile(md, th, 60))
		if !strings.Contains(out, "E = mc²") {
			t.Errorf("%q: equation not rendered:\n%s", md, out)
		}
		if strings.Contains(out, "$$") || strings.Contains(out, "mc^2") {
			t.Errorf("%q: raw LaTeX leaked into preview:\n%s", md, out)
		}
	}
	// Each line is its own equation; an open brace joins lines.
	out := ansi.Strip(Compile("```math\n- A^2+b^2 = 2AB\n- E=mc^2\n\\frac{a}{\n2}\n```", th, 60))
	rows := map[string]bool{}
	for _, l := range strings.Split(out, "\n") {
		rows[strings.TrimSpace(l)] = true
	}
	for _, want := range []string{"−A² + b² = 2AB", "−E = mc²", "a/2"} {
		if !rows[want] {
			t.Errorf("math block rows: missing %q in\n%s", want, out)
		}
	}
	// An unclosed $$ swallows the rest of the note without panicking.
	_ = Compile("$$\nx^2\n", th, 60)
}
