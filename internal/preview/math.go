package preview

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// renderMath draws a math block: each line (or "\\"-separated row) of LaTeX
// is turned into a line of Unicode, indented under the text around it.
func renderMath(body []string, th theme.Theme, _ int, _ *[]Hit) []string {
	var src []string
	for _, eq := range mathLines(body) {
		src = append(src, texToUnicode(eq))
	}
	if len(src) == 0 {
		return []string{lipgloss.NewStyle().Foreground(th.GreyFg).Render("Empty equation: write LaTeX like  E = mc^2")}
	}
	style := lipgloss.NewStyle().Foreground(th.Fg)
	var out []string
	for _, row := range strings.Split(strings.Join(src, "\n"), "\n") {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		out = append(out, style.Render("  "+row))
	}
	return out
}

// mathLines splits a block into equations, one per line; a line that leaves
// a { open carries on into the next.
func mathLines(body []string) []string {
	var eqs []string
	cur, depth := "", 0
	for _, l := range body {
		l = strings.TrimSpace(l)
		if l == "" && depth == 0 {
			continue
		}
		cur = strings.TrimSpace(cur + " " + l)
		depth += strings.Count(l, "{") - strings.Count(l, "\\{") - strings.Count(l, "}") + strings.Count(l, "\\}")
		if depth <= 0 {
			eqs, cur, depth = append(eqs, cur), "", 0
		}
	}
	if cur != "" {
		eqs = append(eqs, cur)
	}
	return eqs
}

// texToUnicode converts a LaTeX math expression to one line of Unicode per
// "\\" row: Greek letters and symbols become their characters, ^ and _
// become super- and subscripts where Unicode has them, \frac{a}{b} becomes
// a/b and \sqrt{x} becomes √x. Unknown commands keep their name.
func texToUnicode(s string) string {
	p := &texParser{src: []rune(s)}
	rows := strings.Split(p.until(0), "\n")
	for i, row := range rows {
		rows[i] = strings.Join(strings.Fields(row), " ")
	}
	return strings.Join(rows, "\n")
}

type texParser struct {
	src []rune
	pos int
}

// until parses atoms until the closing rune (0 = end of input) and returns
// them as text.
func (p *texParser) until(closing rune) string {
	out, base := "", ""
	for p.pos < len(p.src) {
		r := p.src[p.pos]
		if closing != 0 && r == closing {
			p.pos++
			return out
		}
		switch r {
		case '^', '_':
			p.pos++
			// Scripts hug their base; after the limits of ∑ or lim the
			// expression carries on after a space.
			out = strings.TrimRight(out, " ") + script(p.arg(), r == '^')
			if bigOperator(base) {
				out += " "
			}
		case '&':
			p.pos++
			out += "  "
		default:
			base = p.atom()
			out += base
		}
	}
	return out
}

// bigOperator reports whether an atom takes limits (∑, ∫, lim …).
func bigOperator(atom string) bool {
	atom = strings.TrimSpace(atom)
	return strings.Contains("∑∏∐∫∬∭∮⋃⋂⨁⨂", atom) && atom != "" || texFunctions[atom]
}

// arg reads one argument: a {group}, a \command or a single rune.
func (p *texParser) arg() string {
	p.skipSpace()
	if p.pos >= len(p.src) {
		return ""
	}
	if p.src[p.pos] == '{' {
		p.pos++
		return p.until('}')
	}
	return p.atom()
}

// atom reads one rune, group or command.
func (p *texParser) atom() string {
	r := p.src[p.pos]
	switch {
	case r == '{':
		p.pos++
		return p.until('}')
	case r == '}':
		p.pos++ // stray closing brace
		return ""
	case r == '\\':
		return p.command()
	case r == '~':
		p.pos++
		return " "
	case unicode.IsSpace(r):
		p.pos++
		return ""
	case strings.ContainsRune("=<>+", r):
		p.pos++
		return " " + string(r) + " "
	case r == '-':
		p.pos++
		if p.unary() {
			return "−"
		}
		return " − "
	case r == ',' || r == ':':
		p.pos++
		return string(r) + " "
	}
	p.pos++
	return string(r)
}

// unary reports whether the "-" just read is a sign rather than a minus:
// it starts the expression or follows an operator or opening bracket.
func (p *texParser) unary() bool {
	for i := p.pos - 2; i >= 0; i-- {
		if r := p.src[i]; !unicode.IsSpace(r) {
			return strings.ContainsRune("{([=<>+-,:&^_", r)
		}
	}
	return true
}

func (p *texParser) skipSpace() {
	for p.pos < len(p.src) && unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
}

// command reads a \command and its arguments.
func (p *texParser) command() string {
	p.pos++ // backslash
	if p.pos >= len(p.src) {
		return ""
	}
	start := p.pos
	for p.pos < len(p.src) && unicode.IsLetter(p.src[p.pos]) {
		p.pos++
	}
	if p.pos == start { // a symbol command: \\ \, \{ …
		r := p.src[p.pos]
		p.pos++
		switch r {
		case '\\':
			return "\n"
		case ',', ':', ';', ' ':
			return " "
		case '!':
			return ""
		}
		return string(r)
	}
	name := string(p.src[start:p.pos])
	switch name {
	case "frac", "dfrac", "tfrac", "cfrac":
		num, den := p.arg(), p.arg()
		return group(num) + "/" + group(den)
	case "sqrt":
		root := "√"
		p.skipSpace()
		if p.pos < len(p.src) && p.src[p.pos] == '[' {
			p.pos++
			n := p.until(']')
			switch n {
			case "3":
				root = "∛"
			case "4":
				root = "∜"
			default:
				root = script(n, true) + "√"
			}
		}
		return root + group(p.arg())
	case "text", "textrm", "mathrm", "mathit", "mathbf", "mathsf", "mathtt", "boldsymbol", "operatorname", "mbox":
		return p.arg()
	case "mathbb":
		return mapRunes(p.arg(), doubleStruck)
	case "hat", "widehat":
		return combine(p.arg(), '̂')
	case "bar", "overline":
		return combine(p.arg(), '̄')
	case "vec":
		return combine(p.arg(), '⃗')
	case "dot":
		return combine(p.arg(), '̇')
	case "ddot":
		return combine(p.arg(), '̈')
	case "tilde", "widetilde":
		return combine(p.arg(), '̃')
	case "left", "right", "big", "Big", "bigg", "Bigg", "displaystyle", "limits", "nolimits":
		return ""
	case "begin", "end":
		p.arg() // environment name: matrix, aligned, cases …
		return ""
	}
	if sym, ok := texCommands[name]; ok {
		return sym
	}
	if texFunctions[name] {
		return name + " "
	}
	return name
}

// group wraps a fraction part in brackets unless it is a single number or
// symbol, so 1/2a reads as 1/(2a).
func group(s string) string {
	s = strings.TrimSpace(s)
	base := strings.Map(func(r rune) rune {
		if isScript(r) {
			return -1
		}
		return r
	}, s)
	if len([]rune(base)) <= 1 || strings.Trim(base, "0123456789.") == "" || bracketed(strings.TrimLeft(base, "√∛∜")) {
		return s
	}
	return "(" + s + ")"
}

// bracketed reports whether s is one (…) group from end to end.
func bracketed(s string) bool {
	if !strings.HasPrefix(s, "(") || !strings.HasSuffix(s, ")") {
		return false
	}
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 && i < len(s)-1 {
				return false
			}
		}
	}
	return true
}

// isScript reports whether r is one of the super- or subscript runes.
func isScript(r rune) bool {
	for _, t := range []map[rune]rune{superscripts, subscripts} {
		for k, v := range t {
			if v == r && v != k {
				return true
			}
		}
	}
	return false
}

// script writes s as superscript or subscript, or as ^(s) / _(s) when a
// rune has no Unicode form.
func script(s string, sup bool) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	if s == "" {
		return ""
	}
	table := subscripts
	mark := "_"
	if sup {
		table, mark = superscripts, "^"
	}
	var b strings.Builder
	for _, r := range s {
		m, ok := table[r]
		if !ok {
			if len([]rune(s)) == 1 {
				return mark + s
			}
			return mark + "(" + s + ")"
		}
		b.WriteRune(m)
	}
	return b.String()
}

// combine puts an accent (a combining mark) over the last rune of s.
func combine(s string, mark rune) string {
	return strings.TrimSpace(s) + string(mark)
}

func mapRunes(s string, table map[rune]rune) string {
	return strings.Map(func(r rune) rune {
		if m, ok := table[r]; ok {
			return m
		}
		return r
	}, s)
}

var superscripts = map[rune]rune{
	'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴', '5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
	'+': '⁺', '-': '⁻', '−': '⁻', '=': '⁼', '(': '⁽', ')': '⁾',
	'a': 'ᵃ', 'b': 'ᵇ', 'c': 'ᶜ', 'd': 'ᵈ', 'e': 'ᵉ', 'f': 'ᶠ', 'g': 'ᵍ', 'h': 'ʰ', 'i': 'ⁱ', 'j': 'ʲ',
	'k': 'ᵏ', 'l': 'ˡ', 'm': 'ᵐ', 'n': 'ⁿ', 'o': 'ᵒ', 'p': 'ᵖ', 'r': 'ʳ', 's': 'ˢ', 't': 'ᵗ', 'u': 'ᵘ',
	'v': 'ᵛ', 'w': 'ʷ', 'x': 'ˣ', 'y': 'ʸ', 'z': 'ᶻ',
	'A': 'ᴬ', 'B': 'ᴮ', 'D': 'ᴰ', 'E': 'ᴱ', 'G': 'ᴳ', 'H': 'ᴴ', 'I': 'ᴵ', 'J': 'ᴶ', 'K': 'ᴷ', 'L': 'ᴸ',
	'M': 'ᴹ', 'N': 'ᴺ', 'O': 'ᴼ', 'P': 'ᴾ', 'R': 'ᴿ', 'T': 'ᵀ', 'U': 'ᵁ', 'V': 'ⱽ', 'W': 'ᵂ',
	'α': 'ᵅ', 'β': 'ᵝ', 'γ': 'ᵞ', 'δ': 'ᵟ', 'θ': 'ᶿ', 'φ': 'ᵠ', 'χ': 'ᵡ',
	'′': '′', '∗': '*', '*': '*',
}

var subscripts = map[rune]rune{
	'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄', '5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
	'+': '₊', '-': '₋', '−': '₋', '=': '₌', '(': '₍', ')': '₎',
	'a': 'ₐ', 'e': 'ₑ', 'h': 'ₕ', 'i': 'ᵢ', 'j': 'ⱼ', 'k': 'ₖ', 'l': 'ₗ', 'm': 'ₘ', 'n': 'ₙ', 'o': 'ₒ',
	'p': 'ₚ', 'r': 'ᵣ', 's': 'ₛ', 't': 'ₜ', 'u': 'ᵤ', 'v': 'ᵥ', 'x': 'ₓ',
	'β': 'ᵦ', 'γ': 'ᵧ', 'ρ': 'ᵨ', 'φ': 'ᵩ', 'χ': 'ᵪ',
}

var doubleStruck = map[rune]rune{
	'N': 'ℕ', 'Z': 'ℤ', 'Q': 'ℚ', 'R': 'ℝ', 'C': 'ℂ', 'P': 'ℙ', 'H': 'ℍ', 'E': '𝔼', '1': '𝟙',
}

// texFunctions are upright operator names written as words.
var texFunctions = map[string]bool{
	"sin": true, "cos": true, "tan": true, "cot": true, "sec": true, "csc": true,
	"arcsin": true, "arccos": true, "arctan": true, "sinh": true, "cosh": true, "tanh": true,
	"log": true, "ln": true, "lg": true, "exp": true, "lim": true, "max": true, "min": true,
	"sup": true, "inf": true, "det": true, "gcd": true, "deg": true, "dim": true, "ker": true,
	"arg": true, "Pr": true, "mod": true, "bmod": true,
}

var texCommands = map[string]string{
	// Greek
	"alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ", "epsilon": "ϵ", "varepsilon": "ε",
	"zeta": "ζ", "eta": "η", "theta": "θ", "vartheta": "ϑ", "iota": "ι", "kappa": "κ",
	"lambda": "λ", "mu": "μ", "nu": "ν", "xi": "ξ", "omicron": "ο", "pi": "π", "varpi": "ϖ",
	"rho": "ρ", "varrho": "ϱ", "sigma": "σ", "varsigma": "ς", "tau": "τ", "upsilon": "υ",
	"phi": "ϕ", "varphi": "φ", "chi": "χ", "psi": "ψ", "omega": "ω",
	"Gamma": "Γ", "Delta": "Δ", "Theta": "Θ", "Lambda": "Λ", "Xi": "Ξ", "Pi": "Π",
	"Sigma": "Σ", "Upsilon": "Υ", "Phi": "Φ", "Psi": "Ψ", "Omega": "Ω",
	// Big operators
	"sum": "∑", "prod": "∏", "coprod": "∐", "int": "∫", "iint": "∬", "iiint": "∭", "oint": "∮",
	"bigcup": "⋃", "bigcap": "⋂", "bigoplus": "⨁", "bigotimes": "⨂",
	// Binary operators and relations
	"pm": " ± ", "mp": " ∓ ", "times": " × ", "cdot": " · ", "div": " ÷ ", "ast": "∗", "star": "⋆",
	"circ": "∘", "bullet": "∙", "oplus": " ⊕ ", "otimes": " ⊗ ", "wedge": " ∧ ", "vee": " ∨ ",
	"land": " ∧ ", "lor": " ∨ ", "neg": "¬", "lnot": "¬", "cup": " ∪ ", "cap": " ∩ ", "setminus": " ∖ ",
	"leq": " ≤ ", "le": " ≤ ", "geq": " ≥ ", "ge": " ≥ ", "neq": " ≠ ", "ne": " ≠ ", "ll": " ≪ ", "gg": " ≫ ",
	"approx": " ≈ ", "equiv": " ≡ ", "sim": " ∼ ", "simeq": " ≃ ", "cong": " ≅ ", "propto": " ∝ ",
	"in": " ∈ ", "notin": " ∉ ", "ni": " ∋ ", "subset": " ⊂ ", "subseteq": " ⊆ ", "supset": " ⊃ ",
	"supseteq": " ⊇ ", "perp": " ⊥ ", "parallel": " ∥ ", "mid": " ∣ ", "models": " ⊨ ", "vdash": " ⊢ ",
	// Arrows
	"to": " → ", "rightarrow": " → ", "leftarrow": " ← ", "gets": " ← ", "leftrightarrow": " ↔ ",
	"Rightarrow": " ⇒ ", "Leftarrow": " ⇐ ", "Leftrightarrow": " ⇔ ", "implies": " ⟹ ", "iff": " ⟺ ",
	"mapsto": " ↦ ", "uparrow": "↑", "downarrow": "↓", "longrightarrow": " ⟶ ", "longleftarrow": " ⟵ ",
	// Misc symbols
	"infty": "∞", "partial": "∂", "nabla": "∇", "forall": "∀", "exists": "∃", "nexists": "∄",
	"emptyset": "∅", "varnothing": "∅", "angle": "∠", "triangle": "△", "therefore": "∴", "because": "∵",
	"hbar": "ℏ", "ell": "ℓ", "Re": "ℜ", "Im": "ℑ", "aleph": "ℵ", "wp": "℘", "prime": "′", "degree": "°",
	"ldots": "…", "cdots": "⋯", "vdots": "⋮", "ddots": "⋱", "dots": "…",
	"langle": "⟨", "rangle": "⟩", "lfloor": "⌊", "rfloor": "⌋", "lceil": "⌈", "rceil": "⌉",
	"lvert": "|", "rvert": "|", "vert": "|", "Vert": "‖", "lVert": "‖", "rVert": "‖", "|": "‖",
	"quad": "  ", "qquad": "    ", "space": " ",
}
