package math

import (
	"math"

	"golang.org/x/image/font/sfnt"
)

// TeX's box model (TeXbook Appendix G): every formula is a box with a width,
// a height above its baseline and a depth below it, built out of glyphs and
// rules. Layout produces the final baseline so the page can place the box.

// style is one of TeX's four maths styles; the size each one draws at follows
// from the base size.
type style int

const (
	styleDisplay style = iota
	styleText
	styleScript
	styleScriptScript
)

func (s style) script() style {
	if s < styleScript {
		return s + 1
	}
	return styleScriptScript
}

func (s style) size(base float64) float64 {
	switch s {
	case styleScript:
		return base * 0.7
	case styleScriptScript:
		return base * 0.5
	}
	return base
}

type itemKind int

const (
	itGlyph itemKind = iota
	itRule
	itTofu
	itSlash
)

// item is one drawing primitive, positioned from the box's left edge and its
// baseline (positive y is up).
type item struct {
	kind   itemKind
	x, y   float64 // glyph pen / rule or slash top-left
	r      rune
	gid    sfnt.GlyphIndex // a MATH variant drawn by id; 0 draws r
	size   float64
	sx, sy float64
	w, h   float64
}

// box is one laid-out formula, positioned on its own baseline.
type box struct {
	w, h, d float64
	items   []item
}

// layouter carries the formula's base size: style changes scale from it, not
// from the previous style's size.
type layouter struct{ base float64 }

func (b *box) add(child box, dx, dy float64) {
	b.items = append(b.items, shift(child.items, dx, dy)...)
}

func (b *box) addItem(it item, dx, dy float64) {
	it.x += dx
	it.y += dy
	b.items = append(b.items, it)
}

func shift(items []item, dx, dy float64) []item {
	out := make([]item, len(items))
	for i, it := range items {
		it.x += dx
		it.y += dy
		out[i] = it
	}
	return out
}

// spanRows adds two boxes' vertical extents.
func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func ruleThickness(size float64) float64 { return maxf(1, math.Round(0.045*size)) }
func axisHeight(size float64) float64    { return 0.25 * size }

// spacingMu is TeX's inter-atom space in multiples of 3mu (1mu = 1/18 em).
var spacingMu = [8][8]int{
	// Ord Op Bin Rel Open Close Punct Inner
	{0, 1, 2, 3, 0, 0, 0, 1}, // Ord
	{1, 1, 0, 3, 0, 0, 0, 1}, // Op
	{2, 2, 0, 0, 2, 0, 0, 2}, // Bin
	{3, 3, 0, 0, 3, 0, 0, 3}, // Rel
	{0, 0, 0, 0, 0, 0, 0, 0}, // Open
	{0, 1, 2, 3, 0, 0, 0, 1}, // Close
	{1, 1, 0, 1, 1, 1, 1, 1}, // Punct
	{1, 1, 2, 3, 1, 0, 1, 1}, // Inner
}

func spacingPx(a, b Class, size float64, st style) float64 {
	v := spacingMu[a][b]
	if st >= styleScript && v > 1 {
		return 0
	}
	mus := [4]float64{0, 3, 4, 5}[v]
	return mus / 18 * size
}

// list lays atoms out left to right, inserting TeX's inter-atom spaces.
func (l *layouter) list(nodes []Node, size float64, st style, f Font) box {
	var out box
	prev := Ord
	have := false
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if sp, ok := n.(*Space); ok {
			out.w += math.Max(0, sp.Em*size)
			continue
		}
		b, cls := l.atom(n, size, st, f)
		if b.w == 0 && len(b.items) == 0 {
			continue
		}
		if cls == Bin && (!have || prev == Bin || prev == Op || prev == Rel || prev == Open || prev == Punct) {
			cls = Ord
		}
		if have {
			p := prev
			if p == Bin && (cls == Rel || cls == Close || cls == Punct) {
				p = Ord
			}
			out.w += spacingPx(p, cls, size, st)
		}
		out.add(b, out.w, 0)
		out.w += b.w
		out.h = maxf(out.h, b.h)
		out.d = maxf(out.d, b.d)
		prev = cls
		have = true
	}
	return out
}

func (l *layouter) atom(n Node, size float64, st style, f Font) (box, Class) {
	switch t := n.(type) {
	case *Sym:
		return l.sym(t, size, st, f), t.Class
	case *Text:
		return l.text(t, size, st), Ord
	case *Group:
		return l.list(t.Body, size, st, f), Ord
	case *Fonted:
		return l.list(t.Body, size, st, t.Font), Ord
	case *Frac:
		return l.frac(t, size, st, f), Inner
	case *Sqrt:
		return l.sqrt(t, size, st, f), Inner
	case *Script:
		return l.script(t, size, st, f), Inner
	case *Fenced:
		return l.fenced(t, size, st, f), Inner
	case *BigOp:
		return l.bigop(t, size, st), Op
	case *Accent:
		return l.accent(t, size, st, f), Ord
	case *Env:
		return l.env(t, size, st, f), Inner
	case *Styled:
		return l.styled(t, size, st, f), Ord
	case *Stack:
		return l.stack(t, size, st, f), Ord
	case *XArrow:
		return l.xarrow(t, size, st, f), Rel
	case *Not:
		return l.not(t, size, st, f), Rel
	case *FnName:
		b := l.text(&Text{S: t.Name, Font: FontRoman}, size, st)
		return b, Op
	case *ClassWrap:
		b := l.list(t.Body, size, st, f)
		return b, t.Class
	case *RawText:
		b := l.text(&Text{S: t.S, Font: FontRoman}, size*0.9, st)
		return b, Ord
	case *Space:
		return box{w: t.Em * size}, Ord
	}
	return box{}, Ord
}

func (l *layouter) sym(s *Sym, size float64, st style, f Font) box {
	fs := size
	if s.Scale > 0 {
		fs = size * s.Scale
	}
	font := s.Font
	if f != 0 && (font == FontDefault || font == FontText) {
		font = f
	}
	r := glyphFont(s.R, font)
	m := metrics(r, fs)
	if !m.ok || m.advance == 0 {
		return box{w: 0.5 * fs, h: 0.65 * fs, items: []item{{kind: itTofu, size: fs, w: 0.5 * fs, h: 0.65 * fs}}}
	}
	b := box{
		w:     m.advance,
		h:     m.height,
		d:     m.depth,
		items: []item{{kind: itGlyph, r: r, size: fs, sx: 1, sy: 1}},
	}
	if s.Scale > 0 {
		// A \Big delimiter centres on the maths axis like a big operator.
		axis := axisHeight(size)
		half := (m.height + m.depth) / 2
		b.items[0].y = axis - (m.height-m.depth)/2
		b.h = maxf(0, axis+half)
		b.d = maxf(0, half-axis)
	}
	return b
}

func (l *layouter) text(t *Text, size float64, st style) box {
	var out box
	fs := t.Font
	if fs == FontText {
		fs = FontRoman
	}
	for _, r := range t.S {
		if r == ' ' {
			out.w += 0.3 * size
			continue
		}
		rr := glyphFont(r, fs)
		m := metrics(rr, size)
		if !m.ok {
			out.addItem(item{kind: itTofu, size: size, w: 0.5 * size, h: 0.65 * size}, out.w, 0)
			out.w += 0.5 * size
			out.h = maxf(out.h, 0.65*size)
			continue
		}
		out.addItem(item{kind: itGlyph, r: rr, size: size, sx: 1, sy: 1}, out.w, 0)
		out.w += m.advance
		out.h = maxf(out.h, m.height)
		out.d = maxf(out.d, m.depth)
	}
	return out
}

func (l *layouter) frac(f *Frac, size float64, st style, sup Font) box {
	if f.Style == 1 {
		st = styleDisplay
		size = l.base
	}
	if f.Style == 2 {
		st = styleText
		size = l.base
	}
	ns := st.script()
	num := l.list(f.Num, ns.size(l.base), ns, sup)
	den := l.list(f.Den, ns.size(l.base), ns, sup)
	rule := ruleThickness(size)
	axis := axisHeight(size)
	// TeX's shifts: display fractions sit further from the bar than text ones.
	upGap, downGap := 0.12*size, 0.14*size
	if st != styleDisplay {
		upGap, downGap = 0.08*size, 0.09*size
	}
	w := maxf(num.w, den.w)
	var out box
	if f.Bar {
		barTop := axis + rule/2
		numBase := barTop + upGap + num.d
		denBase := barTop - rule - downGap - den.h
		out.addItem(item{kind: itRule, w: w, h: rule}, 0, barTop)
		out.add(num, (w-num.w)/2, numBase)
		out.add(den, (w-den.w)/2, denBase)
		out.h = maxf(numBase+num.h, barTop)
		out.d = maxf(-denBase+den.d, axis)
	} else {
		// \binom: no rule, the two lines sit closer and centre on the axis.
		numBase := axis + 0.35*size + num.d
		denBase := axis - 0.35*size - den.h
		out.add(num, (w-num.w)/2, numBase)
		out.add(den, (w-den.w)/2, denBase)
		out.h = numBase + num.h
		out.d = -denBase + den.d
	}
	out.w = w
	return out
}

func (l *layouter) sqrt(s *Sqrt, size float64, st style, sup Font) box {
	body := l.list(s.Body, size, st, sup)
	rule := ruleThickness(size)
	radical := '√'
	gh := glyphHeight(radical, size)
	if gh <= 0 {
		gh = 0.8 * size
	}
	target := body.h + body.d + rule
	// The index sits in script style at the top left of the radical.
	var idx box
	if len(s.Index) > 0 {
		idx = l.list(s.Index, st.script().size(l.base), st.script(), sup)
	}
	// The font ships real radical sizes; use the smallest one that covers the
	// body, and only scale the base glyph when the table has none.
	if gid, m, _ := variantForHeight(radical, size, target); gid != 0 {
		baseline := body.h + rule - m.height
		top := body.h + rule
		var out box
		out.addItem(item{kind: itGlyph, gid: gid, size: size}, idx.w, baseline)
		vinStart := idx.w + m.advance*0.85
		out.addItem(item{kind: itRule, w: maxf(0.5*size, m.advance+body.w-(vinStart-idx.w)), h: rule}, vinStart, top)
		if idx.w > 0 {
			out.add(idx, idx.w*0.9, top)
		}
		out.add(body, idx.w+m.advance, 0)
		out.w = idx.w + m.advance + body.w
		out.h = maxf(top, body.h+rule)
		out.d = maxf(body.d, -baseline+m.depth)
		return out
	}
	// Scale the radical so its top meets the vinculum exactly: a small body
	// shrinks the glyph, a tall one stretches it.
	sy := math.Min(math.Max(target/gh, 0.5), 4)
	m := metrics(radical, size)
	rw := m.advance
	radicalBaseline := -body.d
	top := radicalBaseline + gh*sy
	var out box
	out.addItem(item{kind: itGlyph, r: radical, size: size, sx: 1, sy: sy}, idx.w, radicalBaseline)
	if idx.w > 0 {
		out.add(idx, idx.w*0.9, top)
	}
	// The vinculum starts where the radical's foot ends and runs over the body.
	vinStart := idx.w + rw*0.55
	out.addItem(item{kind: itRule, w: maxf(0.5*size, vinStart+body.w-vinStart), h: rule}, vinStart, top)
	out.add(body, idx.w+rw, 0)
	out.w = idx.w + rw + body.w
	out.h = maxf(top, body.h+rule)
	out.d = maxf(body.d, -radicalBaseline)
	return out
}

func (l *layouter) script(s *Script, size float64, st style, sup Font) box {
	base := l.list(s.Base, size, st, sup)
	ss := st.script()
	ssize := ss.size(l.base)
	var sub, supBox box
	if len(s.Sub) > 0 {
		sub = l.list(s.Sub, ssize, ss, sup)
	}
	if len(s.Sup) > 0 {
		supBox = l.list(s.Sup, ssize, ss, sup)
	}
	if s.Limits && st == styleDisplay {
		return l.limits(base, supBox, sub)
	}
	var out box
	supY := math.Max(0.45*size, base.h-0.22*size)
	if st != styleDisplay {
		supY = math.Max(0.38*size, base.h-0.2*size)
	}
	subY := -math.Max(0.2*size, base.d+0.18*size)
	if len(s.Sup) > 0 && len(s.Sub) > 0 {
		supBottom := supY - supBox.d
		need := -(supBottom - 0.08*size - sub.h)
		if need < subY {
			subY = need
		}
	}
	if len(s.Sub) > 0 {
		out.add(sub, base.w+0.05*size, subY)
		out.d = maxf(out.d, -subY+sub.d)
	}
	if len(s.Sup) > 0 {
		out.add(supBox, base.w+0.05*size, supY)
		out.h = maxf(out.h, supY+supBox.h)
	}
	out.add(base, 0, 0)
	out.w = base.w + 0.05*size + maxf(supBox.w, sub.w)
	out.h = maxf(out.h, base.h)
	out.d = maxf(out.d, base.d)
	return out
}

// limits puts scripts above and below, centred on the base: \sum_{i}^{n}.
func (l *layouter) limits(base, over, under box) box {
	var out box
	w := maxf(base.w, maxf(over.w, under.w))
	out.add(base, (w-base.w)/2, 0)
	out.h, out.d = base.h, base.d
	if len(over.items) > 0 {
		y := base.h + 0.2*l.base + over.d
		out.add(over, (w-over.w)/2, y)
		out.h = maxf(out.h, y+over.h)
	}
	if len(under.items) > 0 {
		y := -(base.d + 0.25*l.base + under.h)
		out.add(under, (w-under.w)/2, y)
		out.d = maxf(out.d, -y+under.d)
	}
	out.w = w
	return out
}

func (l *layouter) bigop(b *BigOp, size float64, st style) box {
	if st == styleDisplay {
		// Display style uses the next real size the font draws, not a scaled
		// copy of the text size.
		base := glyphHeight(b.R, size)
		if base <= 0 {
			base = 0.8 * size
		}
		if gid, m, _ := variantForHeight(b.R, size, base*1.15); gid != 0 {
			axis := axisHeight(size)
			half := (m.height + m.depth) / 2
			return box{
				w:     m.advance,
				h:     maxf(0, axis+half),
				d:     maxf(0, half-axis),
				items: []item{{kind: itGlyph, gid: gid, size: size, y: axis - (m.height-m.depth)/2}},
			}
		}
	}
	fs := size
	if st == styleDisplay {
		fs = size * 1.4
	}
	m := metrics(b.R, fs)
	if !m.ok {
		return l.sym(&Sym{R: b.R, Class: Op}, size, st, FontDefault)
	}
	axis := axisHeight(size)
	half := (m.height + m.depth) / 2
	baseline := axis - (m.height-m.depth)/2
	return box{
		w:     m.advance,
		h:     maxf(0, axis+half),
		d:     maxf(0, half-axis),
		items: []item{{kind: itGlyph, r: b.R, size: fs, sx: 1, sy: 1, y: baseline}},
	}
}

func (l *layouter) fenced(f *Fenced, size float64, st style, sup Font) box {
	body := l.list(f.Body, size, st, sup)
	axis := axisHeight(size)
	// A fence spans the content's own height and depth; a group with
	// subscripts still fits the normal parentheses, a fraction does not.
	target := body.h + body.d
	var out box
	left, leftB := l.delim(f.Open, size, axis, target)
	out.add(left, 0, leftB)
	out.add(body, left.w, 0)
	right, rightB := l.delim(f.Close, size, axis, target)
	out.add(right, left.w+body.w, rightB)
	out.w = left.w + body.w + right.w
	out.h = maxf(body.h, maxf(leftB+left.h, rightB+right.h))
	out.d = maxf(body.d, maxf(leftB-left.d, rightB-right.d))
	if out.d < 0 {
		out.d = 0
	}
	return out
}

// delim lays one delimiter. The returned box has the glyph, and baseline is
// the y its pen needs so the glyph is centred on the maths axis.
func (l *layouter) delim(d *Delim, size, axis, target float64) (box, float64) {
	if d == nil || d.R == 0 {
		return box{}, 0
	}
	fs := size
	var sy float64 = 1
	if d.Size > 0 {
		fs = size * d.Size
	} else if d.Auto {
		// A base glyph that already covers the content is what TeX uses; a
		// taller one takes the font's real size variant, and scaling the
		// base glyph is the last resort.
		baseH := glyphHeight(d.R, size)
		if baseH >= target*0.95 {
			m := metrics(d.R, size)
			if m.ok {
				return box{
					w:     m.advance,
					h:     m.height,
					d:     m.depth,
					items: []item{{kind: itGlyph, r: d.R, size: size, sx: 1, sy: 1}},
				}, axis - (m.height-m.depth)/2
			}
		}
		if gid, m, _ := variantForHeight(d.R, size, target); gid != 0 {
			b := box{
				w:     m.advance,
				h:     m.height,
				d:     m.depth,
				items: []item{{kind: itGlyph, gid: gid, size: size}},
			}
			return b, axis - (m.height-m.depth)/2
		}
		gh := glyphHeight(d.R, size)
		if gh <= 0 {
			gh = 0.8 * size
		}
		sy = math.Min(math.Max(target/gh, 1), 4)
	}
	m := metrics(d.R, fs)
	if !m.ok {
		return box{w: 0.3 * size, h: 0.7 * size, items: []item{{kind: itTofu, size: fs, w: 0.3 * size, h: 0.7 * size}}}, 0
	}
	b := box{
		w:     m.advance,
		h:     m.height * sy,
		d:     m.depth * sy,
		items: []item{{kind: itGlyph, r: d.R, size: fs, sx: 1, sy: sy}},
	}
	return b, axis - (m.height-m.depth)*sy/2
}

var accentGlyph = map[AccentKind]rune{
	AccHat: 'ˆ', AccWideHat: 'ˆ', AccTilde: '˜', AccWideTilde: '˜',
	AccVec: '→', AccDot: '˙', AccDdot: '¨', AccCheck: 'ˇ', AccBreve: '˘',
	AccAcute: '´', AccGrave: '`',
	AccOverRightArrow: '→', AccOverLeftArrow: '←', AccOverLeftRightArrow: '↔',
	AccOverBrace: '⏞', AccUnderBrace: '⏟',
}

var accentWide = map[AccentKind]bool{
	AccWideHat: true, AccWideTilde: true, AccOverRightArrow: true,
	AccOverLeftArrow: true, AccOverLeftRightArrow: true,
	AccOverBrace: true, AccUnderBrace: true,
}

func (l *layouter) accent(a *Accent, size float64, st style, sup Font) box {
	body := l.list(a.Body, size, st, sup)
	rule := ruleThickness(size)
	var out box
	out.add(body, 0, 0)
	out.w, out.h, out.d = body.w, body.h, body.d
	switch a.Kind {
	case AccBar, AccUnderline:
		if a.Kind == AccBar {
			out.addItem(item{kind: itRule, w: body.w, h: rule}, 0, body.h+0.12*size+rule)
			out.h = body.h + 0.12*size + rule
		} else {
			out.addItem(item{kind: itRule, w: body.w, h: rule}, 0, -body.d-0.12*size)
			out.d = body.d + 0.12*size + rule
		}
		return out
	}
	g, ok := accentGlyph[a.Kind]
	if !ok {
		return out
	}
	m := metrics(g, size)
	if !m.ok {
		return out
	}
	sx, sy := 1.0, 1.0
	if accentWide[a.Kind] && m.advance > 0 && body.w > m.advance {
		sx = math.Min(body.w/m.advance, 5)
	}
	if a.Kind == AccUnderBrace || a.Kind == AccUnderline {
		y := -body.d - 0.15*size
		out.addItem(item{kind: itGlyph, r: g, size: size, sx: sx, sy: sy}, (body.w-m.advance*sx)/2, y)
		out.d = maxf(out.d, body.d+0.15*size+m.depth)
		return out
	}
	y := body.h + 0.12*size + m.depth
	out.addItem(item{kind: itGlyph, r: g, size: size, sx: sx, sy: sy}, (body.w-m.advance*sx)/2, y)
	out.h = maxf(out.h, y+m.height)
	return out
}

// env lays a matrix-like environment out: cells on a grid, one baseline per
// row, then the environment's own fences.
func (l *layouter) env(e *Env, size float64, st style, sup Font) box {
	if e.Style == 1 {
		st = styleDisplay
		size = l.base
	}
	if e.Style == 2 {
		st = styleText
		size = l.base
	}
	fsize := size
	if e.Name == "smallmatrix" {
		fsize = size * 0.7
	}
	rows := e.Rows
	if len(rows) == 0 {
		return box{}
	}
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	if cols == 0 {
		return box{}
	}
	aligns := make([]ColAlign, cols)
	for i := range aligns {
		aligns[i] = AlignCenter
	}
	if len(e.Cols) > 0 {
		for i := range aligns {
			if i < len(e.Cols) {
				aligns[i] = e.Cols[i]
			}
		}
	}
	switch e.Name {
	case "cases":
		for i := range aligns {
			aligns[i] = AlignLeft
		}
	case "aligned", "align", "align*", "alignedat", "alignat", "split", "eqnarray", "eqnarray*":
		for i := range aligns {
			if i%2 == 0 {
				aligns[i] = AlignRight
			} else {
				aligns[i] = AlignLeft
			}
		}
	case "gather", "gather*", "multline", "multline*", "subarray":
		for i := range aligns {
			aligns[i] = AlignCenter
		}
	}
	// Lay every cell out once; its width decides its column's width.
	type cell struct {
		b     box
		class Class
	}
	grid := make([][]cell, len(rows))
	widths := make([]float64, cols)
	for ri, row := range rows {
		grid[ri] = make([]cell, len(row))
		for ci, cn := range row {
			cc := l.list(cn, fsize, st, sup)
			grid[ri][ci] = cell{cc, Ord}
			widths[ci] = maxf(widths[ci], cc.w)
		}
	}
	colGap := 0.6 * size
	if e.Name == "aligned" || e.Name == "align" || e.Name == "align*" || e.Name == "split" {
		colGap = 0.35 * size
	}
	var xs []float64
	x := 0.0
	for ci := 0; ci < cols; ci++ {
		xs = append(xs, x)
		x += widths[ci]
		if ci < cols-1 {
			x += colGap
		}
	}
	total := x
	rowGap := 0.3 * size
	var out box
	baseline := 0.0
	for ri := range rows {
		rh, rd := 0.0, 0.0
		for _, cc := range grid[ri] {
			rh = maxf(rh, cc.b.h)
			rd = maxf(rd, cc.b.d)
		}
		if ri > 0 {
			baseline -= rowGap + rh
		}
		for ci, cc := range grid[ri] {
			var dx float64
			switch aligns[ci] {
			case AlignLeft:
				dx = xs[ci]
			case AlignRight:
				dx = xs[ci] + widths[ci] - cc.b.w
			default:
				dx = xs[ci] + (widths[ci]-cc.b.w)/2
			}
			out.add(cc.b, dx, baseline)
		}
		out.h = maxf(out.h, baseline+rh)
		out.d = maxf(out.d, -baseline+rd)
		baseline -= rd
	}
	out.w = total
	if e.Name == "cases" {
		axis := axisHeight(size)
		open := &Delim{R: '{', Auto: true}
		left, lb := l.delim(open, size, axis, out.h+out.d)
		inner := shift(out.items, left.w+0.1*size, 0)
		out.items = append(shift(left.items, 0, lb), inner...)
		out.w += left.w + 0.1*size
		out.h = maxf(out.h, lb+left.h)
		out.d = maxf(out.d, lb-left.d)
		return out
	}
	fences := map[string][2]rune{
		"pmatrix": {'(', ')'}, "bmatrix": {'[', ']'}, "Bmatrix": {'{', '}'},
		"vmatrix": {'|', '|'}, "Vmatrix": {'‖', '‖'},
	}
	if fr, ok := fences[e.Name]; ok {
		axis := axisHeight(size)
		target := out.h + out.d
		left, lb := l.delim(&Delim{R: fr[0], Auto: true}, size, axis, target)
		right, rb := l.delim(&Delim{R: fr[1], Auto: true}, size, axis, target)
		inner := shift(out.items, left.w+0.1*size, 0)
		out.items = append(shift(left.items, 0, lb), inner...)
		out.items = append(out.items, shift(right.items, left.w+out.w+0.2*size, rb)...)
		out.w += left.w + right.w + 0.2*size
		out.h = maxf(out.h, maxf(lb+left.h, rb+right.h))
		out.d = maxf(out.d, maxf(lb-left.d, rb-right.d))
	}
	return out
}

func (l *layouter) styled(s *Styled, size float64, st style, f Font) box {
	switch s.Style {
	case 1:
		st = styleDisplay
	case 2:
		st = styleText
	case 3:
		st = styleScript
	case 4:
		st = styleScriptScript
	}
	return l.list(s.Body, st.size(l.base), st, f)
}

func (l *layouter) stack(s *Stack, size float64, st style, f Font) box {
	base := l.list(s.Base, size, st, f)
	out := base
	ss := st.script()
	ssize := ss.size(l.base)
	if len(s.Over) > 0 {
		over := l.list(s.Over, ssize, ss, f)
		y := base.h + 0.1*size + over.d
		out.add(over, (maxf(base.w, over.w)-over.w)/2, y)
		out.h = maxf(out.h, y+over.h)
		out.w = maxf(out.w, over.w)
	}
	if len(s.Under) > 0 {
		under := l.list(s.Under, ssize, ss, f)
		y := -(base.d + 0.1*size + under.h)
		out.add(under, (maxf(base.w, under.w)-under.w)/2, y)
		out.d = maxf(out.d, -y+under.d)
		out.w = maxf(out.w, under.w)
	}
	return out
}

func (l *layouter) xarrow(x *XArrow, size float64, st style, f Font) box {
	ss := st.script()
	ssize := ss.size(l.base)
	above := l.list(x.Above, ssize, ss, f)
	var below box
	if len(x.Below) > 0 {
		below = l.list(x.Below, ssize, ss, f)
	}
	glyph := '→'
	if x.Left {
		glyph = '←'
	}
	m := metrics(glyph, size)
	if !m.ok {
		m = glyphMetrics{advance: size, height: 0.5 * size, ok: true}
	}
	w := math.Max(1.0*size, above.w)
	w = math.Max(w, below.w)
	sx := 1.0
	if m.advance > 0 {
		sx = math.Min(w/m.advance, 4)
	}
	var out box
	out.addItem(item{kind: itGlyph, r: glyph, size: size, sx: sx, sy: 1}, 0, 0)
	out.w = w
	out.h = m.height
	out.d = m.depth
	if len(above.items) > 0 {
		y := m.height + 0.25*size + above.d
		out.add(above, (w-above.w)/2, y)
		out.h = maxf(out.h, y+above.h)
	}
	if len(below.items) > 0 {
		y := -(m.depth + 0.25*size + below.h)
		out.add(below, (w-below.w)/2, y)
		out.d = maxf(out.d, -y+below.d)
	}
	return out
}

func (l *layouter) not(n *Not, size float64, st style, f Font) box {
	b := l.list(n.Body, size, st, f)
	rule := ruleThickness(size)
	b.addItem(item{kind: itSlash, x: 0, y: b.h, w: b.w, h: -(b.h + b.d), size: rule}, 0, 0)
	return b
}
