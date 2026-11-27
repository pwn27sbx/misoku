// Package math lays TeX maths out and draws it for the rich reading view.
//
// It is written from scratch: a tokeniser, a parser for a TeX subset, TeX's
// box model (Appendix G of the TeXbook) and a renderer that draws with the
// bundled Latin Modern Math face. mddog's engine was read only as a reference
// for which constructs a viewer must support (MIT, see docs/rich-math.md); no
// code was translated.
//
// The engine is pure Go. The page reaches it through Box and FromBlock
// (math.go), which hand it a sized rich.Box; Parse and the node types are
// exported so tests can pin the tree, and everything about drawing stays
// internal so the page never learns how a formula is drawn.
package math

// Class is TeX's atom class: it decides the space between two neighbours.
type Class int

const (
	Ord Class = iota
	Op
	Bin
	Rel
	Open
	Close
	Punct
	Inner
)

// String is for test failures only.
func (c Class) String() string {
	return [...]string{"ord", "op", "bin", "rel", "open", "close", "punct", "inner"}[c]
}

// Font selects the alphabet a letter is drawn in. FontDefault follows TeX:
// Latin and lowercase Greek letters italic, digits and uppercase Greek upright.
type Font int

const (
	FontDefault Font = iota
	FontRoman
	FontBold
	FontItalic
	FontBoldItalic
	FontDouble   // \mathbb
	FontScript   // \mathcal, \mathscr
	FontFraktur  // \mathfrak
	FontSans     // \mathsf
	FontSansBold // \mathbf{\mathsf} is not a TeX construct; kept for parity
	FontMono     // \mathtt
	FontText     // \text, \textrm: upright words, spaces kept
)

// Node is one element of a formula. The tree is what the parser produces and
// the layout walks.
type Node interface{ node() }

// Sym is one glyph: a letter, a digit, an operator or a symbol command.
type Sym struct {
	R     rune
	Class Class
	Font  Font
	Scale float64 // 0 means 1; \big and friends scale a delimiter up
}

// Space is explicit horizontal space (\, \quad, \  …), in ems.
type Space struct{ Em float64 }

// Text is a word drawn upright with its spaces, from \text and \mbox.
type Text struct {
	S    string
	Font Font
}

// Fonted draws its body in one alphabet: \mathbf{x}, \mathbb{R}, …
type Fonted struct {
	Font Font
	Body []Node
}

// Group is {…}: it exists for scoping and for scripts, and draws as its body.
type Group struct{ Body []Node }

// Frac is \frac and its variants. Bar=false is \binom, which keeps the
// parentheses and drops the rule.
type Frac struct {
	Num, Den []Node
	Bar      bool
	Style    int // 0 keep, 1 display, 2 text
}

// Sqrt is \sqrt{x} and \sqrt[n]{x}.
type Sqrt struct{ Index, Body []Node }

// Script is a base with a subscript and/or a superscript. Limits puts them
// above and below instead of to the side (display style, \sum, \lim).
type Script struct {
	Base, Sub, Sup []Node
	Limits         bool
}

// Delim is one delimiter glyph. Size 0 is a plain delimiter at the current
// size; a positive Size is a fixed multiple (\Big); Auto stretches to the
// content (\left … \right).
type Delim struct {
	R    rune
	Size float64
	Auto bool
}

// Fenced is \left( … \right) and the matrix environments' brackets.
type Fenced struct {
	Open, Close *Delim
	Body        []Node
}

// BigOp is a large operator: ∑, ∫ and their relatives.
type BigOp struct{ R rune }

// AccentKind is which accent to draw over (or under) a body.
type AccentKind int

const (
	AccHat AccentKind = iota
	AccWideHat
	AccTilde
	AccWideTilde
	AccBar
	AccVec
	AccDot
	AccDdot
	AccCheck
	AccBreve
	AccAcute
	AccGrave
	AccOverRightArrow
	AccOverLeftArrow
	AccOverLeftRightArrow
	AccOverBrace
	AccUnderBrace
	AccUnderline
)

// Accent draws an accent above its body, or a line/brace below for the
// underline family.
type Accent struct {
	Kind AccentKind
	Body []Node
}

// ColAlign is one column's alignment in an array or matrix.
type ColAlign int

const (
	AlignCenter ColAlign = iota
	AlignLeft
	AlignRight
)

// Env is a matrix-like environment: array, matrix, cases, aligned, split…
type Env struct {
	Name  string
	Rows  [][][]Node // rows → cells → nodes
	Cols  []ColAlign
	Style int // 0 keep, 1 display, 2 text
}

// Styled is \displaystyle, \textstyle, \scriptstyle, \scriptscriptstyle.
type Styled struct {
	Style int // 1 display, 2 text, 3 script, 4 scriptscript
	Body  []Node
}

// Stack is \overset, \underset and \stackrel.
type Stack struct {
	Base, Over, Under []Node
}

// XArrow is \xrightarrow[below]{above} and \xleftarrow.
type XArrow struct {
	Left  bool
	Above []Node
	Below []Node
}

// Not is \not: a slash drawn over the next relation.
type Not struct{ Body []Node }

// FnName is a function name (\sin, \lim, \operatorname{…}). Limits means its
// scripts go above and below in display style.
type FnName struct {
	Name   string
	Limits bool
}

// ClassWrap forces an atom class, from \mathbin, \mathrel and friends.
type ClassWrap struct {
	Class Class
	Body  []Node
}

// RawText is a construct the engine does not know: it is drawn literally so a
// formula never disappears or crashes the page.
type RawText struct{ S string }

func (*Sym) node()       {}
func (*Space) node()     {}
func (*Text) node()      {}
func (*Fonted) node()    {}
func (*Group) node()     {}
func (*Frac) node()      {}
func (*Sqrt) node()      {}
func (*Script) node()    {}
func (*Fenced) node()    {}
func (*BigOp) node()     {}
func (*Accent) node()    {}
func (*Env) node()       {}
func (*Styled) node()    {}
func (*Stack) node()     {}
func (*XArrow) node()    {}
func (*Not) node()       {}
func (*FnName) node()    {}
func (*ClassWrap) node() {}
func (*RawText) node()   {}

// Expr is a parsed formula. Unknown keeps every command the parser did not
// know, so a test can fail on a gap instead of the engine hiding it.
type Expr struct {
	Nodes   []Node
	Unknown []string
}
