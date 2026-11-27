package math

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type tokKind int

const (
	tokEOF tokKind = iota
	tokChar
	tokCmd
	tokSpace
	tokLBrace
	tokRBrace
	tokSup
	tokSub
	tokAmp
	tokRow
)

type token struct {
	kind     tokKind
	r        rune
	s        string
	pos, end int
}

func isASCIILetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// lex turns the source into tokens. Positions are kept so an argument that
// must keep its spaces (\text) can be cut from the original string.
func lex(src string) []token {
	var out []token
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\\':
			start := i
			i++
			if i >= len(src) {
				out = append(out, token{kind: tokCmd, pos: start, end: i})
				continue
			}
			if isASCIILetter(src[i]) {
				for i < len(src) && isASCIILetter(src[i]) {
					i++
				}
				out = append(out, token{kind: tokCmd, s: src[start+1 : i], pos: start, end: i})
				continue
			}
			r, size := utf8.DecodeRuneInString(src[i:])
			i += size
			if r == '\\' {
				out = append(out, token{kind: tokRow, s: `\\`, pos: start, end: i})
				continue
			}
			out = append(out, token{kind: tokCmd, r: r, s: string(r), pos: start, end: i})
		case c == '{':
			out = append(out, token{kind: tokLBrace, r: '{', pos: i, end: i + 1})
			i++
		case c == '}':
			out = append(out, token{kind: tokRBrace, r: '}', pos: i, end: i + 1})
			i++
		case c == '^':
			out = append(out, token{kind: tokSup, r: '^', pos: i, end: i + 1})
			i++
		case c == '_':
			out = append(out, token{kind: tokSub, r: '_', pos: i, end: i + 1})
			i++
		case c == '&':
			out = append(out, token{kind: tokAmp, r: '&', pos: i, end: i + 1})
			i++
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			start := i
			for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\n' || src[i] == '\r') {
				i++
			}
			out = append(out, token{kind: tokSpace, r: ' ', pos: start, end: i})
		default:
			start := i
			r, size := utf8.DecodeRuneInString(src[i:])
			i += size
			out = append(out, token{kind: tokChar, r: r, pos: start, end: i})
		}
	}
	out = append(out, token{kind: tokEOF, pos: len(src), end: len(src)})
	return out
}

// stopSet says what ends a list. A closing brace is accepted only when
// brace is set, so a stray "}" is an error instead of a silent truncation.
type stopSet struct {
	brace bool
	amp   bool
	row   bool
	brack bool
	cmds  map[string]bool
}

type parser struct {
	src     string
	toks    []token
	i       int
	unknown []string
}

// Parse turns one formula's source into an expression tree. Malformed input
// (an unclosed group) returns an error; anything merely unknown becomes
// literal text and is listed in Expr.Unknown.
func Parse(src string) (*Expr, error) {
	p := &parser{src: src, toks: lex(src)}
	var lines [][]Node
	for {
		p.skipSpace()
		if p.peek().kind == tokEOF {
			break
		}
		body, err := p.parseList(stopSet{row: true})
		if err != nil {
			return nil, err
		}
		lines = append(lines, body)
		if p.peek().kind == tokRow {
			p.next()
			continue
		}
		break
	}
	e := &Expr{Unknown: p.unknown}
	switch len(lines) {
	case 0:
	case 1:
		e.Nodes = lines[0]
	default:
		rows := make([][][]Node, len(lines))
		for i, line := range lines {
			rows[i] = [][]Node{line}
		}
		e.Nodes = []Node{&Env{Name: "gather", Rows: rows, Style: 1}}
	}
	return e, nil
}

func (p *parser) peek() token { return p.toks[p.i] }

func (p *parser) next() token {
	t := p.toks[p.i]
	if t.kind != tokEOF {
		p.i++
	}
	return t
}

func (p *parser) skipSpace() {
	for p.peek().kind == tokSpace {
		p.i++
	}
}

func (p *parser) peekCmd(name string) bool {
	t := p.peek()
	return t.kind == tokCmd && t.s == name
}

func (p *parser) parseList(st stopSet) ([]Node, error) {
	var nodes []Node
	for {
		p.skipSpace()
		t := p.peek()
		if t.kind == tokEOF {
			return nodes, nil
		}
		if t.kind == tokRBrace {
			if st.brace {
				return nodes, nil
			}
			return nil, fmt.Errorf("math: unexpected } at byte %d", t.pos)
		}
		if st.amp && t.kind == tokAmp {
			return nodes, nil
		}
		if st.row && t.kind == tokRow {
			return nodes, nil
		}
		if st.brack && t.kind == tokChar && t.r == ']' {
			return nodes, nil
		}
		if t.kind == tokCmd && st.cmds != nil && st.cmds[t.s] {
			return nodes, nil
		}
		if t.kind == tokCmd && styleLevel(t.s) > 0 {
			p.next()
			rest, err := p.parseList(st)
			if err != nil {
				return nil, err
			}
			if len(rest) > 0 {
				nodes = append(nodes, &Styled{Style: styleLevel(t.s), Body: rest})
			}
			return nodes, nil
		}
		if t.kind == tokSup || t.kind == tokSub {
			atom, err := p.scripts(nil)
			if err != nil {
				return nil, err
			}
			if atom == nil {
				continue
			}
			nodes = append(nodes, atom)
			continue
		}
		if t.kind == tokRow {
			// A row break that no environment claimed ends this list.
			return nodes, nil
		}
		atom, err := p.atom()
		if err != nil {
			return nil, err
		}
		atom, err = p.scripts(atom)
		if err != nil {
			return nil, err
		}
		if atom != nil {
			nodes = append(nodes, atom)
		}
	}
}

// scripts attaches ^ and _ to base, and honours \limits on big operators.
func (p *parser) scripts(base Node) (Node, error) {
	var sup, sub []Node
	limits := false
	if b, ok := base.(*BigOp); ok && limitOps[b.R] {
		limits = true
	}
	if f, ok := base.(*FnName); ok && f.Limits {
		limits = true
	}
	if a, ok := base.(*Accent); ok && (a.Kind == AccOverBrace || a.Kind == AccUnderBrace) {
		limits = true
	}
	for {
		p.skipSpace()
		t := p.peek()
		if t.kind == tokCmd && (t.s == "limits" || t.s == "nolimits") {
			p.next()
			limits = t.s == "limits"
			continue
		}
		switch t.kind {
		case tokSup:
			p.next()
			arg, err := p.arg()
			if err != nil {
				return nil, err
			}
			sup = arg
		case tokSub:
			p.next()
			arg, err := p.arg()
			if err != nil {
				return nil, err
			}
			sub = arg
		default:
			if base == nil {
				if sup == nil && sub == nil {
					return nil, nil
				}
				return &Script{Sup: sup, Sub: sub}, nil
			}
			if sup == nil && sub == nil {
				return base, nil
			}
			if s, ok := base.(*Script); ok {
				if s.Sup == nil {
					s.Sup = sup
				}
				if s.Sub == nil {
					s.Sub = sub
				}
				s.Limits = s.Limits || limits
				return s, nil
			}
			return &Script{Base: []Node{base}, Sup: sup, Sub: sub, Limits: limits}, nil
		}
	}
}

func (p *parser) atom() (Node, error) {
	p.skipSpace()
	t := p.peek()
	switch t.kind {
	case tokLBrace:
		p.next()
		body, err := p.parseList(stopSet{brace: true})
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRBrace {
			return nil, fmt.Errorf("math: unclosed group at byte %d", t.pos)
		}
		p.next()
		return &Group{Body: body}, nil
	case tokChar:
		p.next()
		return charNode(t.r), nil
	case tokCmd:
		p.next()
		return p.command(t)
	case tokEOF:
		return nil, nil
	}
	return nil, fmt.Errorf("math: unexpected %q", string(t.r))
}

// arg reads one argument: a group, or the single atom TeX allows (\frac12).
func (p *parser) arg() ([]Node, error) {
	p.skipSpace()
	if p.peek().kind == tokLBrace {
		p.next()
		body, err := p.parseList(stopSet{brace: true})
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tokRBrace {
			return nil, fmt.Errorf("math: unclosed argument")
		}
		p.next()
		return body, nil
	}
	atom, err := p.atom()
	if err != nil {
		return nil, err
	}
	if atom == nil {
		return nil, nil
	}
	return []Node{atom}, nil
}

// rawArg cuts a brace argument out of the source, spaces and all. \text needs
// its own spaces, which the maths tokeniser ignores everywhere else.
func (p *parser) rawArg() (string, error) {
	p.skipSpace()
	if p.peek().kind != tokLBrace {
		return "", fmt.Errorf("math: expected { after command")
	}
	open := p.next()
	depth := 1
	start := open.end
	for {
		t := p.next()
		switch t.kind {
		case tokEOF:
			return "", fmt.Errorf("math: unclosed argument")
		case tokLBrace:
			depth++
		case tokRBrace:
			depth--
			if depth == 0 {
				return cleanText(p.src[start:t.pos]), nil
			}
		}
	}
}

// nameArg reads \operatorname's argument: a group, or TeX's single token.
func (p *parser) nameArg() (string, error) {
	p.skipSpace()
	if p.peek().kind == tokLBrace {
		return p.rawArg()
	}
	t := p.next()
	if t.kind == tokCmd {
		if d, ok := symbols[t.s]; ok {
			return string(d.r), nil
		}
		return t.s, nil
	}
	if t.kind == tokEOF {
		return "", nil
	}
	return string(t.r), nil
}

// cleanText removes the escapes a text argument may carry.
func cleanText(s string) string {
	r := strings.NewReplacer(`\\`, " ", `\{`, "{", `\}`, "}", `\$`, "$", `\%`, "%", `\&`, "&", `\_`, "_")
	return r.Replace(s)
}

func charNode(r rune) Node {
	switch r {
	case '\'':
		return &Sym{R: '′', Class: Ord}
	case '~':
		return &Space{Em: interwordSpace}
	}
	if r >= '0' && r <= '9' || isLetter(r) || r >= 0x80 {
		f := FontDefault
		if !isLetter(r) {
			f = FontRoman
		}
		return &Sym{R: r, Class: runeClass(r), Font: f}
	}
	return &Sym{R: r, Class: runeClass(r), Font: FontRoman}
}

// TeX's spaces, in ems. 1mu is 1/18 em; the named forms keep the three
// values apart at the call sites.
const (
	thinSpace        = 3.0 / 18
	mediumSpace      = 4.0 / 18
	thickSpace       = 5.0 / 18
	interwordSpace   = 1.0 / 3
	italicCorrection = 0.2
)

func styleLevel(name string) int {
	switch name {
	case "displaystyle":
		return 1
	case "textstyle":
		return 2
	case "scriptstyle":
		return 3
	case "scriptscriptstyle":
		return 4
	}
	return 0
}

func fontFor(name string) (Font, bool) {
	switch name {
	case "mathrm", "mathup":
		return FontRoman, true
	case "mathbf":
		return FontBold, true
	case "mathit":
		return FontItalic, true
	case "boldsymbol", "bm":
		return FontBoldItalic, true
	case "mathbb":
		return FontDouble, true
	case "mathcal", "mathscr":
		return FontScript, true
	case "mathfrak":
		return FontFraktur, true
	case "mathsf":
		return FontSans, true
	case "mathtt":
		return FontMono, true
	}
	return 0, false
}

// fixedDelimScale is the size \middle uses until it can match \left…\right.
const fixedDelimScale = 1.6

// bigSizes are the fixed delimiter sizes \big…\Bigg ask for, as multiples of
// the base size.
var bigSizes = map[string]float64{
	"big": 1.2, "bigl": 1.2, "bigr": 1.2, "bigm": 1.2,
	"Big": 1.6, "Bigl": 1.6, "Bigr": 1.6, "Bigm": 1.6,
	"bigg": 2.2, "biggl": 2.2, "biggr": 2.2, "biggm": 2.2,
	"Bigg": 2.8, "Biggl": 2.8, "Biggr": 2.8, "Biggm": 2.8,
}

func (p *parser) command(t token) (Node, error) {
	name := t.s
	if d, ok := symbols[name]; ok {
		// Large operators are their own node so a plain \sum takes limits
		// above and below in display style, like \lim does.
		if limitOps[d.r] {
			return &BigOp{R: d.r}, nil
		}
		return &Sym{R: d.r, Class: d.class, Font: d.font}, nil
	}
	if f, ok := fontFor(name); ok {
		body, err := p.arg()
		if err != nil {
			return nil, err
		}
		return &Fonted{Font: f, Body: body}, nil
	}
	if size, ok := bigSizes[name]; ok {
		d, err := p.delim()
		if err != nil {
			return nil, err
		}
		class := Ord
		switch name[len(name)-1] {
		case 'l':
			class = Open
		case 'r':
			class = Close
		case 'm':
			class = Rel
		}
		if d.R == 0 {
			return &Space{}, nil
		}
		return &Sym{R: d.R, Class: class, Scale: size}, nil
	}
	if fn, ok := functionNames[name]; ok {
		return &FnName{Name: fn.name, Limits: fn.limits}, nil
	}
	switch name {
	case "frac", "dfrac", "tfrac", "cfrac":
		num, err := p.arg()
		if err != nil {
			return nil, err
		}
		den, err := p.arg()
		if err != nil {
			return nil, err
		}
		f := &Frac{Num: num, Den: den, Bar: true}
		if name == "dfrac" {
			f.Style = 1
		}
		if name == "tfrac" {
			f.Style = 2
		}
		return f, nil
	case "binom", "dbinom", "tbinom":
		num, err := p.arg()
		if err != nil {
			return nil, err
		}
		den, err := p.arg()
		if err != nil {
			return nil, err
		}
		f := &Frac{Num: num, Den: den}
		if name == "dbinom" {
			f.Style = 1
		}
		if name == "tbinom" {
			f.Style = 2
		}
		return &Fenced{
			Open:  &Delim{R: '(', Auto: true},
			Close: &Delim{R: ')', Auto: true},
			Body:  []Node{f},
		}, nil
	case "sqrt":
		var index []Node
		if p.peek().kind == tokChar && p.peek().r == '[' {
			p.next()
			idx, err := p.parseList(stopSet{brack: true})
			if err != nil {
				return nil, err
			}
			if p.peek().kind == tokChar && p.peek().r == ']' {
				p.next()
			}
			index = idx
		}
		body, err := p.arg()
		if err != nil {
			return nil, err
		}
		return &Sqrt{Index: index, Body: body}, nil
	case "text", "mbox", "textrm", "textnormal", "textup", "textit", "textbf":
		s, err := p.rawArg()
		if err != nil {
			return nil, err
		}
		f := FontText
		switch name {
		case "textbf":
			f = FontBold
		case "textit":
			f = FontItalic
		}
		return &Text{S: s, Font: f}, nil
	case "operatorname":
		limits := false
		p.skipSpace()
		if t := p.peek(); t.kind == tokChar && t.r == '*' {
			p.next()
			limits = true
		}
		s, err := p.nameArg()
		if err != nil {
			return nil, err
		}
		return &FnName{Name: strings.TrimSpace(s), Limits: limits}, nil
	case "middle":
		d, err := p.delim()
		if err != nil {
			return nil, err
		}
		return &Sym{R: d.R, Class: Ord, Scale: fixedDelimScale}, nil
	case "left":
		open, err := p.delim()
		if err != nil {
			return nil, err
		}
		body, err := p.parseList(stopSet{cmds: map[string]bool{"right": true}})
		if err != nil {
			return nil, err
		}
		if !p.peekCmd("right") {
			return nil, fmt.Errorf("math: \\left without \\right")
		}
		p.next()
		closeg, err := p.delim()
		if err != nil {
			return nil, err
		}
		return &Fenced{Open: open, Close: closeg, Body: body}, nil
	case "right":
		return nil, fmt.Errorf("math: \\right without \\left")
	case "begin":
		return p.environment()
	case "end":
		return nil, fmt.Errorf("math: \\end without \\begin")
	case "overset", "stackrel":
		over, err := p.arg()
		if err != nil {
			return nil, err
		}
		base, err := p.arg()
		if err != nil {
			return nil, err
		}
		return &Stack{Base: base, Over: over}, nil
	case "underset":
		under, err := p.arg()
		if err != nil {
			return nil, err
		}
		base, err := p.arg()
		if err != nil {
			return nil, err
		}
		return &Stack{Base: base, Under: under}, nil
	case "xrightarrow", "xleftarrow":
		var below []Node
		if p.peek().kind == tokChar && p.peek().r == '[' {
			p.next()
			b, err := p.parseList(stopSet{brack: true})
			if err != nil {
				return nil, err
			}
			if p.peek().kind == tokChar && p.peek().r == ']' {
				p.next()
			}
			below = b
		}
		above, err := p.arg()
		if err != nil {
			return nil, err
		}
		return &XArrow{Left: name == "xleftarrow", Above: above, Below: below}, nil
	case "not":
		body, err := p.arg()
		if err != nil {
			return nil, err
		}
		return &Not{Body: body}, nil
	case "mathbin", "mathrel", "mathord", "mathop", "mathopen", "mathclose", "mathpunct":
		body, err := p.arg()
		if err != nil {
			return nil, err
		}
		return &ClassWrap{Class: classFor(name), Body: body}, nil
	case "quad":
		return &Space{Em: 1}, nil
	case "qquad":
		return &Space{Em: 2}, nil
	case "space":
		return &Space{Em: interwordSpace}, nil
	case "enspace":
		return &Space{Em: 0.5}, nil
	case ",":
		return &Space{Em: thinSpace}, nil
	case ":":
		return &Space{Em: mediumSpace}, nil
	case ">":
		return &Space{Em: mediumSpace}, nil
	case ";":
		return &Space{Em: thickSpace}, nil
	case "!":
		return &Space{Em: -thinSpace}, nil
	case " ":
		return &Space{Em: interwordSpace}, nil
	case "/":
		return &Space{Em: italicCorrection}, nil
	case "limits", "nolimits":
		return nil, nil
	case "label", "tag", "nonumber", "notag":
		// Labels number equations in LaTeX; the page draws none of them.
		if name == "label" || name == "tag" {
			_, _ = p.rawArg()
		}
		return nil, nil
	}
	if acc, ok := accents[name]; ok {
		body, err := p.arg()
		if err != nil {
			return nil, err
		}
		return &Accent{Kind: acc, Body: body}, nil
	}
	p.unknown = append(p.unknown, name)
	return &RawText{S: "\\" + name}, nil
}

// classFor maps the \math* class commands.
func classFor(name string) Class {
	switch name {
	case "mathbin":
		return Bin
	case "mathrel":
		return Rel
	case "mathop":
		return Op
	case "mathopen":
		return Open
	case "mathclose":
		return Close
	case "mathpunct":
		return Punct
	}
	return Ord
}

var accents = map[string]AccentKind{
	"hat": AccHat, "widehat": AccWideHat,
	"tilde": AccTilde, "widetilde": AccWideTilde,
	"bar": AccBar, "vec": AccVec, "dot": AccDot, "ddot": AccDdot,
	"check": AccCheck, "breve": AccBreve, "acute": AccAcute, "grave": AccGrave,
	"overline": AccBar, "underline": AccUnderline,
	"overrightarrow": AccOverRightArrow, "overleftarrow": AccOverLeftArrow,
	"overleftrightarrow": AccOverLeftRightArrow,
	"overbrace":          AccOverBrace, "underbrace": AccUnderBrace,
}

// delim reads the delimiter argument of \left, \right and \big.
func (p *parser) delim() (*Delim, error) {
	p.skipSpace()
	t := p.peek()
	switch t.kind {
	case tokChar:
		p.next()
		return &Delim{R: t.r}, nil
	case tokCmd:
		p.next()
		if t.s == "." {
			return &Delim{R: 0}, nil
		}
		if d, ok := symbols[t.s]; ok {
			return &Delim{R: d.r}, nil
		}
		if t.r != 0 {
			return &Delim{R: t.r}, nil
		}
		return nil, fmt.Errorf("math: %q is not a delimiter", "\\"+t.s)
	}
	return nil, fmt.Errorf("math: expected a delimiter")
}

// environment parses \begin{name} … \end{name}; a mismatched or missing
// closing name is an error, never a silently accepted grid.
func (p *parser) environment() (Node, error) {
	name, err := p.rawArg()
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	cols := []ColAlign(nil)
	switch name {
	case "array", "subarray", "alignedat":
		if p.peek().kind == tokLBrace {
			spec, err := p.rawArg()
			if err != nil {
				return nil, err
			}
			cols = parseColSpec(spec)
		}
	}
	closeEnv := func() error {
		p.next() // \end
		endName, err := p.rawArg()
		if err != nil {
			return err
		}
		if got := strings.TrimSpace(endName); got != name {
			return fmt.Errorf("math: \\begin{%s} closed by \\end{%s}", name, got)
		}
		return nil
	}
	var rows [][][]Node
	var cells [][]Node
	for {
		p.skipSpace()
		if p.peek().kind == tokEOF {
			return nil, fmt.Errorf("math: \\begin{%s} without \\end", name)
		}
		if p.peekCmd("end") {
			if err := closeEnv(); err != nil {
				return nil, err
			}
			break
		}
		cell, err := p.parseList(stopSet{amp: true, row: true, cmds: map[string]bool{"end": true}})
		if err != nil {
			return nil, err
		}
		cells = append(cells, cell)
		p.skipSpace()
		switch {
		case p.peekCmd("end"):
			if err := closeEnv(); err != nil {
				return nil, err
			}
			goto done
		case p.peek().kind == tokAmp:
			p.next()
			p.skipSpace()
			// "a &" ends the row with an empty second cell; keep it so the
			// column still exists for alignment.
			if k := p.peek().kind; k == tokEOF || k == tokRow || p.peekCmd("end") {
				cells = append(cells, nil)
			}
		case p.peek().kind == tokRow:
			p.next()
			p.skipSpace()
			// An optional row length follows (\\[2pt]); it never changes the
			// grid, so it is consumed and dropped.
			if p.peek().kind == tokChar && p.peek().r == '[' {
				p.next()
				for p.peek().kind != tokEOF && !(p.peek().kind == tokChar && p.peek().r == ']') {
					p.next()
				}
				if p.peek().kind == tokChar {
					p.next()
				}
			}
			rows = append(rows, cells)
			cells = nil
		default:
			return nil, fmt.Errorf("math: unexpected token in \\begin{%s}", name)
		}
	}
done:
	// A cell terminated by & just before \end still belongs to the grid.
	if len(cells) > 0 {
		rows = append(rows, cells)
	}
	return &Env{Name: name, Rows: rows, Cols: cols}, nil
}

// parseColSpec turns an array's {lcr|…} spec into per-column alignments.
func parseColSpec(spec string) []ColAlign {
	var out []ColAlign
	skip := 0
	for _, r := range spec {
		if skip > 0 {
			if r == '{' {
				skip++
			}
			if r == '}' {
				skip--
			}
			continue
		}
		switch r {
		case 'l':
			out = append(out, AlignLeft)
		case 'r':
			out = append(out, AlignRight)
		case 'c':
			out = append(out, AlignCenter)
		case 'p', 'm', 'b':
			out = append(out, AlignLeft)
			skip = 1 // skip the {width}
		}
	}
	return out
}
