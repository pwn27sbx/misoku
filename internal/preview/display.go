package preview

import (
	"sort"
	"strings"
	"sync/atomic"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

// DisplayBlock is one source range of the note and the compiled rows it owns.
type DisplayBlock struct {
	Line, End int      // source lines [Line, End)
	Row       int      // first compiled row
	Rows      []string // compiled rows it owns
	Folded    bool     // a folded section: lines after the first are hidden
}

// Display is the last compile cut by source line, for a view that draws the
// compiled rows of some lines and the Markdown of others. Blocks are sorted,
// disjoint, cover every source line and their Rows add up to the compiled body.
type Display struct {
	Generation uint64 // global counter, bumped by every recompile
	PageID     string
	Failed     bool // the compile panicked
	Blocks     []DisplayBlock
	Footer     []string // backlinks and mentions
	FooterRow  int
}

// generations numbers the displays of every preview, so a display can be told
// from any other one.
var generations atomic.Uint64

// Display returns the partition of the last compile.
func (m Model) Display() Display { return m.display }

// HitAtRow returns the view element at column x of compiled row row. The row
// is absolute, whatever the viewport's scroll, and x counts from the row's
// first column: the display's rows carry no page margin.
func (m Model) HitAtRow(row, x int) (Hit, bool) {
	for i := len(m.hits) - 1; i >= 0; i-- {
		h := m.hits[i]
		if row >= h.Row && row < h.Row+h.H && x >= h.X0 && x < h.X1 {
			return h, true
		}
	}
	return Hit{}, false
}

// PressRow does what a click does at column x of compiled row row, without
// the drag handles. It reports false when nothing is there.
func (m *Model) PressRow(row, x int) (tea.Cmd, bool) {
	h, ok := m.HitAtRow(row, x)
	if !ok {
		return nil, false
	}
	return m.press(h), true
}

// newDisplay cuts a compile's rows by source line. all holds the body's rows
// followed by the footer's.
func newDisplay(raw, pageID string, failed bool, all []string, bodyRows int, blocks []Block) Display {
	d := Display{
		Generation: generations.Add(1),
		PageID:     pageID,
		Failed:     failed,
		Footer:     all[bodyRows:],
		FooterRow:  bodyRows,
	}
	if !failed {
		d.Blocks = partition(strings.Split(raw, "\n"), frontMatterEnd(raw), all[:bodyRows], blocks)
	}
	return d
}

// compiledRows splits compiled output into rows; no output has no rows.
func compiledRows(compiled string) []string {
	if compiled == "" {
		return nil
	}
	return strings.Split(compiled, "\n")
}

// frontMatterEnd is the line after the note's front matter, 0 without one.
func frontMatterEnd(raw string) int {
	if meta, _ := SplitFrontMatter(raw); meta != nil {
		return core.ParseFrontMatter(raw).EndLine + 1
	}
	return 0
}

// span is a source range on its way to becoming a DisplayBlock.
type span struct {
	line, end   int
	row, rowEnd int
	folded      bool
	gap         bool // lines no block drew: its rows are worked out from its neighbours
}

// partition gives every source line and every body row to exactly one block.
func partition(lines []string, frontEnd int, rows []string, blocks []Block) []DisplayBlock {
	spans := mergeBlocks(lines, blocks)
	if frontEnd > 0 {
		firstRow := len(rows)
		if len(spans) > 0 {
			firstRow = spans[0].row
		}
		spans = append([]span{{end: frontEnd, rowEnd: firstRow}}, spans...)
	}
	spans = fillGaps(lines, spans)

	// Backwards, so each block's rows run up to where the next one starts.
	out := make([]DisplayBlock, len(spans))
	next := len(rows)
	for k := len(spans) - 1; k >= 0; k-- {
		s, start := spans[k], next
		switch {
		case k == 0:
			start = 0
		case !s.gap:
			start = s.row
		case !spans[k-1].gap:
			start = spans[k-1].rowEnd // the first run of a gap owns its spacing
		}
		out[k] = DisplayBlock{Line: s.line, End: s.end, Row: start, Rows: rows[start:next], Folded: s.folded}
		next = start
	}
	return out
}

// mergeBlocks turns the compiler's blocks, which nest and overlap, into
// disjoint spans sorted by line. A callout's quote lines become one span.
func mergeBlocks(lines []string, blocks []Block) []span {
	sorted := append([]Block(nil), blocks...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].Line < sorted[b].Line })

	var out []span
	inCallout := false
	for _, b := range sorted {
		if n := len(out); n > 0 {
			last := &out[n-1]
			overlaps := b.Line < last.end || b.Row < last.rowEnd
			if overlaps || (inCallout && b.Line == last.end && calloutContinues(lines[b.Line])) {
				last.end = max(last.end, b.End)
				last.row = min(last.row, b.Row)
				last.rowEnd = max(last.rowEnd, b.Row+b.H)
				last.folded = last.folded || b.Folded
				continue
			}
		}
		out = append(out, span{line: b.Line, end: b.End, row: b.Row, rowEnd: b.Row + b.H, folded: b.Folded})
		inCallout = isCalloutHead(lines[b.Line])
	}
	return out
}

func isCalloutHead(line string) bool {
	m := blockquoteRegex.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	_, _, ok := callout(m[1])
	return ok
}

// calloutContinues reports whether line is a quote line that does not start
// another callout.
func calloutContinues(line string) bool {
	return blockquoteRegex.MatchString(line) && !isCalloutHead(line)
}

// fillGaps adds the lines no block drew. A run of blank lines is its own span;
// the stripped lines (comments, tag-only, "^id") join whatever follows them.
func fillGaps(lines []string, spans []span) []span {
	var out []span
	at := 0
	for _, s := range spans {
		out, s.line = appendGap(out, lines, at, s.line)
		out = append(out, s)
		at = s.end
	}
	out, rest := appendGap(out, lines, at, len(lines))
	switch {
	case rest == len(lines):
	case len(out) > 0:
		out[len(out)-1].end = len(lines) // nothing follows: join what came before
	default:
		out = append(out, span{end: len(lines), gap: true})
	}
	return out
}

// appendGap appends the spans of the uncovered lines [lo, hi), each ending at
// a run of blank lines, and returns where the stripped lines left over begin.
func appendGap(out []span, lines []string, lo, hi int) ([]span, int) {
	start := lo
	for i := lo; i < hi; i++ {
		if isBlank(lines[i]) && (i+1 == hi || !isBlank(lines[i+1])) {
			out = append(out, span{line: start, end: i + 1, gap: true})
			start = i + 1
		}
	}
	return out, start
}

func isBlank(line string) bool { return strings.TrimSpace(line) == "" }
