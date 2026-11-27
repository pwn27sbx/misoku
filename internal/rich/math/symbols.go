package math

// The symbol tables: command name to glyph, atom class and default alphabet.
// LaTeX's names are used as written in documents; a missing command renders
// literally instead of vanishing.

type symDef struct {
	r     rune
	class Class
	font  Font
}

func ord(r rune) symDef    { return symDef{r, Ord, FontDefault} }
func bin(r rune) symDef    { return symDef{r, Bin, FontDefault} }
func rel(r rune) symDef    { return symDef{r, Rel, FontDefault} }
func op(r rune) symDef     { return symDef{r, Op, FontDefault} }
func open(r rune) symDef   { return symDef{r, Open, FontDefault} }
func closeg(r rune) symDef { return symDef{r, Close, FontDefault} }
func punct(r rune) symDef  { return symDef{r, Punct, FontDefault} }

var symbols = map[string]symDef{
	// Greek, lower case.
	"alpha":      ord('α'),
	"beta":       ord('β'),
	"gamma":      ord('γ'),
	"delta":      ord('δ'),
	"epsilon":    ord('ϵ'),
	"varepsilon": ord('ε'),
	"zeta":       ord('ζ'),
	"eta":        ord('η'),
	"theta":      ord('θ'),
	"vartheta":   ord('ϑ'),
	"iota":       ord('ι'),
	"kappa":      ord('κ'),
	"varkappa":   ord('ϰ'),
	"lambda":     ord('λ'),
	"mu":         ord('μ'),
	"nu":         ord('ν'),
	"xi":         ord('ξ'),
	"omicron":    ord('ο'),
	"pi":         ord('π'),
	"varpi":      ord('ϖ'),
	"rho":        ord('ρ'),
	"varrho":     ord('ϱ'),
	"sigma":      ord('σ'),
	"varsigma":   ord('ς'),
	"tau":        ord('τ'),
	"upsilon":    ord('υ'),
	"phi":        ord('ϕ'),
	"varphi":     ord('φ'),
	"chi":        ord('χ'),
	"psi":        ord('ψ'),
	"omega":      ord('ω'),
	// Greek, upper case (TeX draws these upright).
	"Gamma":   ord('Γ'),
	"Delta":   ord('Δ'),
	"Theta":   ord('Θ'),
	"Lambda":  ord('Λ'),
	"Xi":      ord('Ξ'),
	"Pi":      ord('Π'),
	"Sigma":   ord('Σ'),
	"Upsilon": ord('Υ'),
	"Phi":     ord('Φ'),
	"Psi":     ord('Ψ'),
	"Omega":   ord('Ω'),

	// Binary operators.
	"pm":            bin('±'),
	"mp":            bin('∓'),
	"times":         bin('×'),
	"div":           bin('÷'),
	"cdot":          bin('⋅'),
	"cdotp":         punct('·'),
	"ast":           bin('∗'),
	"star":          bin('⋆'),
	"circ":          bin('∘'),
	"bullet":        bin('∙'),
	"oplus":         bin('⊕'),
	"ominus":        bin('⊖'),
	"otimes":        bin('⊗'),
	"oslash":        bin('⊘'),
	"odot":          bin('⊙'),
	"bigcirc":       bin('◯'),
	"dagger":        bin('†'),
	"ddagger":       bin('‡'),
	"cap":           bin('∩'),
	"cup":           bin('∪'),
	"sqcap":         bin('⊓'),
	"sqcup":         bin('⊔'),
	"wedge":         bin('∧'),
	"land":          bin('∧'),
	"vee":           bin('∨'),
	"lor":           bin('∨'),
	"setminus":      bin('∖'),
	"smallsetminus": bin('∖'),
	"wr":            bin('≀'),
	"amalg":         bin('⨿'),
	"triangleleft":  bin('◁'),
	"triangleright": bin('▷'),
	"lhd":           bin('⊲'),
	"rhd":           bin('⊳'),
	"unlhd":         bin('⊴'),
	"unrhd":         bin('⊵'),
	"dotplus":       bin('∔'),
	"ltimes":        bin('⋉'),
	"rtimes":        bin('⋊'),
	"boxplus":       bin('⊞'),
	"boxminus":      bin('⊟'),
	"boxtimes":      bin('⊠'),
	"boxdot":        bin('⊡'),
	"curlywedge":    bin('⋏'),
	"curlyvee":      bin('⋎'),

	// Relations.
	"leq":             rel('≤'),
	"le":              rel('≤'),
	"geq":             rel('≥'),
	"ge":              rel('≥'),
	"neq":             rel('≠'),
	"ne":              rel('≠'),
	"equiv":           rel('≡'),
	"sim":             rel('∼'),
	"simeq":           rel('≃'),
	"approx":          rel('≈'),
	"cong":            rel('≅'),
	"propto":          rel('∝'),
	"in":              rel('∈'),
	"notin":           rel('∉'),
	"ni":              rel('∋'),
	"owns":            rel('∋'),
	"subset":          rel('⊂'),
	"supset":          rel('⊃'),
	"subseteq":        rel('⊆'),
	"supseteq":        rel('⊇'),
	"nsubseteq":       rel('⊈'),
	"nsupseteq":       rel('⊉'),
	"sqsubseteq":      rel('⊑'),
	"sqsupseteq":      rel('⊒'),
	"prec":            rel('≺'),
	"succ":            rel('≻'),
	"preceq":          rel('⪯'),
	"succeq":          rel('⪰'),
	"ll":              rel('≪'),
	"gg":              rel('≫'),
	"perp":            rel('⊥'),
	"mid":             rel('∣'),
	"parallel":        rel('∥'),
	"nmid":            rel('∤'),
	"nparallel":       rel('∦'),
	"models":          rel('⊨'),
	"vdash":           rel('⊢'),
	"dashv":           rel('⊣'),
	"doteq":           rel('≐'),
	"asymp":           rel('≍'),
	"bowtie":          rel('⋈'),
	"smile":           rel('⌣'),
	"frown":           rel('⌢'),
	"trianglelefteq":  rel('⊴'),
	"trianglerighteq": rel('⊵'),
	"therefore":       rel('∴'),
	"because":         rel('∵'),
	"lesssim":         rel('≲'),
	"gtrsim":          rel('≳'),
	"leqslant":        rel('⩽'),
	"geqslant":        rel('⩾'),
	"subsetneq":       rel('⊊'),
	"supsetneq":       rel('⊋'),
	"approxeq":        rel('≊'),
	"coloneqq":        rel('≔'),
	"eqqcolon":        rel('≕'),

	// Arrows.
	"to":                 rel('→'),
	"rightarrow":         rel('→'),
	"leftarrow":          rel('←'),
	"gets":               rel('←'),
	"uparrow":            rel('↑'),
	"downarrow":          rel('↓'),
	"updownarrow":        rel('↕'),
	"leftrightarrow":     rel('↔'),
	"Rightarrow":         rel('⇒'),
	"Leftarrow":          rel('⇐'),
	"Leftrightarrow":     rel('⇔'),
	"longrightarrow":     rel('⟶'),
	"longleftarrow":      rel('⟵'),
	"longleftrightarrow": rel('⟷'),
	"Longrightarrow":     rel('⟹'),
	"Longleftarrow":      rel('⟸'),
	"Longleftrightarrow": rel('⟺'),
	"mapsto":             rel('↦'),
	"longmapsto":         rel('⟼'),
	"hookrightarrow":     rel('↪'),
	"hookleftarrow":      rel('↩'),
	"rightharpoonup":     rel('⇀'),
	"rightharpoondown":   rel('⇁'),
	"leftharpoonup":      rel('↼'),
	"leftharpoondown":    rel('↽'),
	"rightleftharpoons":  rel('⇌'),
	"nearrow":            rel('↗'),
	"searrow":            rel('↘'),
	"swarrow":            rel('↙'),
	"nwarrow":            rel('↖'),
	"implies":            rel('⟹'),
	"impliedby":          rel('⟸'),
	"iff":                rel('⟺'),
	"leadsto":            rel('⇝'),

	// Ordinary symbols.
	"infty":         ord('∞'),
	"partial":       ord('∂'),
	"nabla":         ord('∇'),
	"forall":        ord('∀'),
	"exists":        ord('∃'),
	"nexists":       ord('∄'),
	"emptyset":      ord('∅'),
	"varnothing":    ord('∅'),
	"hbar":          ord('ℏ'),
	"hslash":        ord('ℏ'),
	"ell":           ord('ℓ'),
	"Re":            ord('ℜ'),
	"Im":            ord('ℑ'),
	"aleph":         ord('ℵ'),
	"beth":          ord('ℶ'),
	"gimel":         ord('ℷ'),
	"daleth":        ord('ℸ'),
	"wp":            ord('℘'),
	"prime":         ord('′'),
	"angle":         ord('∠'),
	"measuredangle": ord('∡'),
	"triangle":      ord('△'),
	"square":        ord('□'),
	"Box":           ord('□'),
	"blacksquare":   ord('■'),
	"lozenge":       ord('◊'),
	"clubsuit":      ord('♣'),
	"diamondsuit":   ord('♢'),
	"heartsuit":     ord('♡'),
	"spadesuit":     ord('♠'),
	"top":           ord('⊤'),
	"bot":           ord('⊥'),
	"flat":          ord('♭'),
	"natural":       ord('♮'),
	"sharp":         ord('♯'),
	"checkmark":     ord('✓'),
	"degree":        ord('°'),
	"dots":          ord('…'),
	"ldots":         ord('…'),
	"cdots":         ord('⋯'),
	"vdots":         ord('⋮'),
	"ddots":         ord('⋱'),
	"colon":         punct(':'),
	"vert":          ord('|'),
	"Vert":          ord('‖'),
	"backslash":     ord('∖'),
	"surd":          ord('√'),
	"copyright":     ord('©'),
	"pounds":        ord('£'),
	"euro":          ord('€'),
	"dag":           bin('†'),
	"ddag":          bin('‡'),
	"P":             ord('¶'),
	"S":             ord('§'),

	// Delimiters. Their class is Open/Close only when they stand as a pair;
	// a lone one is an ordinary glyph, which is why \left…\right exists.
	"langle": open('⟨'),
	"rangle": closeg('⟩'),
	"lVert":  open('‖'),
	"rVert":  closeg('‖'),
	"lvert":  open('|'),
	"rvert":  closeg('|'),
	"lfloor": open('⌊'),
	"rfloor": closeg('⌋'),
	"lceil":  open('⌈'),
	"rceil":  closeg('⌉'),
	"lbrace": open('{'),
	"rbrace": closeg('}'),
	"lbrack": open('['),
	"rbrack": closeg(']'),

	// Large operators. The layout decides whether their scripts sit beside or
	// above/below; the parser only records which glyph they are.
	"sum":       op('∑'),
	"prod":      op('∏'),
	"coprod":    op('∐'),
	"bigcup":    op('⋃'),
	"bigcap":    op('⋂'),
	"bigoplus":  op('⨁'),
	"bigotimes": op('⨂'),
	"bigodot":   op('⨀'),
	"biguplus":  op('⨄'),
	"int":       op('∫'),
	"iint":      op('∬'),
	"iiint":     op('∭'),
	"oint":      op('∮'),
	"oiint":     op('∯'),

	// Punctuation and escapes.
	"%": ord('%'),
	"&": ord('&'),
	"#": ord('#'),
	"_": ord('_'),
	"$": ord('$'),
	"{": ord('{'),
	"}": ord('}'),
}

// limitOps take their scripts above and below in display style. Integrals do
// not; TeX keeps their limits at the side.
var limitOps = map[rune]bool{
	'∑': true, '∏': true, '∐': true, '⋃': true, '⋂': true,
	'⨁': true, '⨂': true, '⨀': true, '⨄': true,
}

// functionNames are the operator names TeX typesets upright. Limits marks the
// ones whose scripts go above and below in display style.
type fnDef struct {
	name   string
	limits bool
}

var functionNames = map[string]fnDef{
	"sin": {"sin", false}, "cos": {"cos", false}, "tan": {"tan", false},
	"cot": {"cot", false}, "sec": {"sec", false}, "csc": {"csc", false},
	"arcsin": {"arcsin", false}, "arccos": {"arccos", false}, "arctan": {"arctan", false},
	"sinh": {"sinh", false}, "cosh": {"cosh", false}, "tanh": {"tanh", false},
	"coth": {"coth", false}, "log": {"log", false}, "ln": {"ln", false},
	"lg": {"lg", false}, "exp": {"exp", false}, "deg": {"deg", false},
	"dim": {"dim", false}, "ker": {"ker", false}, "hom": {"hom", false},
	"arg": {"arg", false}, "det": {"det", true}, "gcd": {"gcd", true},
	"inf": {"inf", true}, "sup": {"sup", true}, "lim": {"lim", true},
	"liminf": {"lim inf", true}, "limsup": {"lim sup", true},
	"max": {"max", true}, "min": {"min", true}, "Pr": {"Pr", true},
	"mod": {"mod", false}, "bmod": {"mod", false},
}

// runeClass is the atom class of a raw character, following TeX's tables.
func runeClass(r rune) Class {
	switch r {
	case '+', '-', '*', '/', '±', '∓', '×', '÷', '·', '⋅':
		return Bin
	case '=', '<', '>', ':', '≤', '≥', '≠', '≈', '≡', '∼', '→', '←', '↔', '⇒', '⇐', '⇔', '∈', '∉', '⊂', '⊃', '⊆', '⊇', '∞':
		return Rel
	case ',', ';':
		return Punct
	case '(', '[', '{', '⟨', '⌊', '⌈':
		return Open
	case ')', ']', '}', '⟩', '⌋', '⌉', '!', '?':
		return Close
	default:
		return Ord
	}
}

// mathAlnum maps a base letter to the Unicode maths alphanumeric block for a
// font. ASCII holes (script B, fraktur C, double-struck R…) are spelled out.
func mathAlnum(r rune, f Font) rune {
	if f == FontDefault || f == FontRoman || f == FontText {
		return r
	}
	type block struct {
		upper, lower, digit rune
	}
	blocks := map[Font]block{
		FontBold:       {0x1D400, 0x1D41A, 0x1D7CE},
		FontItalic:     {0x1D434, 0x1D44E, 0},
		FontBoldItalic: {0x1D468, 0x1D482, 0},
		FontScript:     {0x1D49C, 0x1D4B6, 0},
		FontFraktur:    {0x1D504, 0x1D51E, 0},
		FontDouble:     {0x1D538, 0x1D552, 0x1D7D8},
		FontSans:       {0x1D5A0, 0x1D5BA, 0x1D7E2},
		FontSansBold:   {0x1D5D4, 0x1D5EE, 0x1D7EC},
		FontMono:       {0x1D670, 0x1D68A, 0x1D7F6},
	}
	b, ok := blocks[f]
	if !ok {
		return r
	}
	switch {
	case r >= 'A' && r <= 'Z':
		return hole(b.upper, r-'A', f, false)
	case r >= 'a' && r <= 'z':
		return hole(b.lower, r-'a', f, true)
	case r >= '0' && r <= '9' && b.digit != 0:
		return b.digit + (r - '0')
	}
	return mathGreek(r, f)
}

// hole fills the punctuation-letter slots Unicode left out of the blocks.
func hole(start rune, off rune, f Font, lower bool) rune {
	switch f {
	case FontScript:
		if !lower {
			switch off {
			case 1:
				return 'ℬ'
			case 4:
				return 'ℰ'
			case 5:
				return 'ℱ'
			case 7:
				return 'ℋ'
			case 8:
				return 'ℐ'
			case 11:
				return 'ℒ'
			case 12:
				return 'ℳ'
			case 17:
				return 'ℛ'
			}
			break
		}
		switch off {
		case 4:
			return 'ℯ'
		case 6:
			return 'ℊ'
		case 14:
			return 'ℴ'
		}
	case FontFraktur:
		if !lower {
			switch off {
			case 2:
				return 'ℭ'
			case 7:
				return 'ℌ'
			case 8:
				return 'ℑ'
			case 17:
				return 'ℜ'
			case 25:
				return 'ℨ'
			}
		}
	case FontDouble:
		if !lower {
			switch off {
			case 2:
				return 'ℂ'
			case 7:
				return 'ℍ'
			case 13:
				return 'ℕ'
			case 15:
				return 'ℙ'
			case 16:
				return 'ℚ'
			case 17:
				return 'ℝ'
			case 25:
				return 'ℤ'
			}
		}
	case FontItalic:
		if lower && off == 7 {
			return 'ℎ'
		}
	}
	return start + off
}

// mathGreek maps Greek letters to the styled block, which is ordered
// Alpha…Omega then alpha…omega, with the final sigma in the alphabet.
func mathGreek(r rune, f Font) rune {
	upper := []rune("ΑΒΓΔΕΖΗΘΙΚΛΜΝΞΟΠΡϴΣΤΥΦΧΨΩ")
	lower := []rune("αβγδεζηθικλμνξοπρςστυφχψω")
	upperStart := map[Font]rune{FontBold: 0x1D6A8, FontItalic: 0x1D6E2, FontBoldItalic: 0x1D71C}[f]
	lowerStart := map[Font]rune{FontBold: 0x1D6C2, FontItalic: 0x1D6FC, FontBoldItalic: 0x1D736}[f]
	if upperStart == 0 || lowerStart == 0 {
		return r
	}
	for i, g := range upper {
		if g == r {
			return upperStart + rune(i)
		}
	}
	for i, g := range lower {
		if g == r {
			return lowerStart + rune(i)
		}
	}
	return r
}

// isLetter reports whether r is drawn italic by default in maths.
func isLetter(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
		(r >= 0x03B1 && r <= 0x03C9) // lower-case Greek
}
