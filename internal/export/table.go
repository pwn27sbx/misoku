package export

import (
	"math"
	"strings"
	"unicode/utf8"
)

// Table type sizes in points and spacing in millimetres.
const (
	tableFont     = 9.5 // the size every table starts at
	tableFontMin  = 7.0 // the smallest it shrinks to before turning sideways
	tableFontStep = 0.5
	tableLine     = 4.8  // line height at tableFont
	cellPad       = 3.0  // horizontal padding inside a cell, both sides together
	colMin        = 14.0 // narrowest column a table asks for before sharing out the width
	colMaxShare   = 0.6  // widest column, as a share of the table width
)

// tableLayout is how a table fits the page: its type size, column widths
// and whether it needs sideways pages.
type tableLayout struct {
	size      float64
	widths    []float64
	landscape bool
}

// tableLayout fits rows to the page: upright at the largest type size whose
// columns all hold their longest word, then sideways the same way. A table
// that fits nowhere prints sideways at the smallest size, and only its
// words wider than their column break.
func (d *doc) tableLayout(rows [][]string, cols int) tableLayout {
	pw, ph := d.pdf.GetPageSize()
	l, _, r, _ := d.pdf.GetMargins()
	upright, sideways := min(pw, ph)-l-r, max(pw, ph)-l-r
	for _, landscape := range []bool{false, true} {
		width := upright
		if landscape {
			width = sideways
		}
		for size := tableFont; size >= tableFontMin; size -= tableFontStep {
			if widths, ok := d.columnWidths(rows, cols, size, width); ok {
				return tableLayout{size: size, widths: widths, landscape: landscape}
			}
		}
	}
	widths, _ := d.columnWidths(rows, cols, tableFontMin, sideways)
	return tableLayout{size: tableFontMin, widths: widths, landscape: true}
}

// columnWidths shares width between the columns at size points. Each column
// first asks for its longest cell, as tables always have; if that leaves a
// column narrower than its longest word, every column gets its longest word
// and the rest goes to those that asked for more. It reports false, with
// widths scaled down to fit, when the longest words alone are too wide.
func (d *doc) columnWidths(rows [][]string, cols int, size, width float64) ([]float64, bool) {
	asked := make([]float64, cols) // longest cell
	least := make([]float64, cols) // longest word
	for ri, r := range rows {
		style := rowStyle(ri)
		for c, cell := range r {
			text := plain(cell)
			asked[c] = max(asked[c], d.measure("sans", "", size, text)+4)
			for _, w := range wordPieces(text) {
				least[c] = max(least[c], d.measure("sans", style, size, w)+cellPad)
			}
		}
	}
	total := 0.0
	for c := range asked {
		asked[c] = min(max(asked[c], colMin), width*colMaxShare)
		total += asked[c]
	}
	widths := make([]float64, cols)
	roomy := true
	for c := range asked {
		widths[c] = asked[c] * width / total
		roomy = roomy && widths[c] >= least[c]
	}
	if roomy {
		return widths, true
	}
	need, more := 0.0, 0.0
	for c := range least {
		need += least[c]
		more += max(asked[c]-least[c], 0)
	}
	if need > width {
		for c := range least {
			widths[c] = least[c] * width / need
		}
		return widths, false
	}
	for c := range least {
		widths[c] = least[c]
		if more > 0 {
			widths[c] += (width - need) * max(asked[c]-least[c], 0) / more
		} else {
			widths[c] += (width - need) / float64(cols)
		}
	}
	return widths, true
}

// rowStyle is the font style of row ri: the header row is bold.
func rowStyle(ri int) string {
	if ri == 0 {
		return "B"
	}
	return ""
}

// wordPieces splits text into the pieces a line may break between: at
// spaces, and after "-" and "/" inside a word.
func wordPieces(text string) []string {
	var out []string
	for _, word := range strings.Fields(text) {
		out = append(out, splitAfterBreaks(word)...)
	}
	return out
}

// splitAfterBreaks cuts word after each "-" and "/".
func splitAfterBreaks(word string) []string {
	var out []string
	for word != "" {
		i := strings.IndexAny(word, "-/")
		if i < 0 {
			return append(out, word)
		}
		out = append(out, word[:i+1])
		word = word[i+1:]
	}
	return out
}

// fits reports whether s, in the current font, is no wider than w. It
// measures in font units exactly as fpdf's SplitText does, so tables that
// fitted before wrap the same way.
func (d *doc) fits(s string, w float64) bool {
	_, size := d.pdf.GetFontSize()
	return float64(d.pdf.GetStringSymbolWidth(s)) <= math.Ceil(w*1000/size)
}

// wrapWords breaks text into lines no wider than w in the current font. A
// line breaks at a space; a word too wide for a line of its own also breaks
// after its "-" and "/", and a piece still too wide breaks between letters.
func (d *doc) wrapWords(text string, w float64) []string {
	var lines []string
	line := ""
	put := func(piece, sep string) {
		if line != "" && d.fits(line+sep+piece, w) {
			line += sep + piece
			return
		}
		if line != "" {
			lines = append(lines, line)
		}
		for !d.fits(piece, w) && utf8.RuneCountInString(piece) > 1 {
			cut := d.fitPrefix(piece, w)
			lines = append(lines, piece[:cut])
			piece = piece[cut:]
		}
		line = piece
	}
	for _, word := range strings.Fields(text) {
		if d.fits(word, w) {
			put(word, " ")
			continue
		}
		for i, piece := range splitAfterBreaks(word) {
			sep := ""
			if i == 0 {
				sep = " "
			}
			put(piece, sep)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

// fitPrefix is the byte length of the longest start of s, at least one
// letter, that fits in w.
func (d *doc) fitPrefix(s string, w float64) int {
	_, first := utf8.DecodeRuneInString(s)
	cut := first
	for i, r := range s {
		end := i + utf8.RuneLen(r)
		if i > 0 && !d.fits(s[:end], w) {
			break
		}
		cut = end
	}
	return cut
}

// startsTable reports whether a Markdown table starts at lines[i]: a row
// with "|" followed by a separator row.
func startsTable(lines []string, i int) bool {
	return i+1 < len(lines) && strings.Contains(lines[i], "|") && tableSep.MatchString(lines[i+1])
}

// tableRows parses the table starting at lines[i] and returns its rows and
// the index of its last line.
func tableRows(lines []string, i int) ([][]string, int) {
	rows := [][]string{splitRow(strings.TrimSpace(lines[i]))}
	for i += 2; i < len(lines) && strings.Contains(lines[i], "|") && strings.TrimSpace(lines[i]) != ""; i++ {
		rows = append(rows, splitRow(lines[i]))
	}
	return rows, i - 1
}

// tableCols is the number of columns of rows: the longest row's.
func tableCols(rows [][]string) int {
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	return cols
}

// wideTableAt reports whether a table starts at lines[i] that only fits on
// a sideways page.
func (d *doc) wideTableAt(lines []string, i int) bool {
	if i >= len(lines) || !startsTable(lines, i) {
		return false
	}
	rows, _ := tableRows(lines, i)
	cols := tableCols(rows)
	return cols > 0 && d.tableLayout(rows, cols).landscape
}

// table draws rows as a grid, the first row as its header. A table too wide
// for an upright page gets sideways pages of its own; the header repeats on
// every page the table spans.
func (d *doc) table(rows [][]string) {
	cols := tableCols(rows)
	if cols == 0 {
		return
	}
	lay := d.tableLayout(rows, cols)
	if lay.landscape {
		d.sideways() // blocks has turned the page already; nested blocks may not
	}
	d.gap(2)
	// The body font is set first, as tables always have, so the font
	// commands of a table that fits stay byte for byte the same.
	d.font("sans", "", lay.size)
	lineH := tableLine * lay.size / tableFont
	d.pdf.SetDrawColor(inkRule.r, inkRule.g, inkRule.b)
	d.pdf.SetLineWidth(0.25)
	var header [][]string
	var headerH float64
	for ri, r := range rows {
		cells, h := d.tableRow(r, cols, ri, lay, lineH)
		if ri == 0 {
			header, headerH = cells, h
			d.ensure(h)
		} else if _, ph := d.pdf.GetPageSize(); d.pdf.GetY()+h > ph-margin {
			d.ensure(h)
			d.font("sans", rowStyle(0), lay.size)
			d.drawRow(header, headerH, true, lay.widths, lineH)
			d.font("sans", rowStyle(ri), lay.size)
		}
		d.drawRow(cells, h, ri == 0, lay.widths, lineH)
	}
	d.pdf.Ln(3)
}

// tableRow sets the font for row ri of the table and wraps its cells,
// returning their lines and the row height.
func (d *doc) tableRow(r []string, cols, ri int, lay tableLayout, lineH float64) ([][]string, float64) {
	d.font("sans", rowStyle(ri), lay.size)
	cells := make([][]string, cols)
	h := lineH
	for c := 0; c < cols; c++ {
		text := ""
		if c < len(r) {
			text = d.clean(plain(r[c]))
		}
		cells[c] = d.wrapWords(text, lay.widths[c]-cellPad)
		if len(cells[c]) == 0 {
			cells[c] = []string{""}
		}
		h = max(h, float64(len(cells[c]))*lineH)
	}
	return cells, h + 2
}

// drawRow draws one row of wrapped cells at the current line, shaded when it
// is the header, and moves below it.
func (d *doc) drawRow(cells [][]string, h float64, header bool, widths []float64, lineH float64) {
	l, _, _, _ := d.pdf.GetMargins()
	y, x := d.pdf.GetY(), l
	for c, lines := range cells {
		if header {
			d.fill(paperCode)
			d.pdf.Rect(x, y, widths[c], h, "FD")
		} else {
			d.pdf.Rect(x, y, widths[c], h, "D")
		}
		d.color(inkText)
		for li, line := range lines {
			d.pdf.SetXY(x+1.5, y+1+float64(li)*lineH)
			d.pdf.CellFormat(widths[c]-cellPad, lineH, line, "", 0, "L", false, 0, "")
		}
		x += widths[c]
	}
	d.pdf.SetXY(l, y+h)
}
