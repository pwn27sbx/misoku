package highlight

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// Highlighter remembers the last highlighted text so the next call only
// re-colours what an edit changed. Its result is always identical to the
// package Markdown function on the same text and theme.
//
// Beside the colours it keeps, per line, the state the scanner was in when
// the line began: whether a segment (one Markdown line, or a whole fenced
// block) starts there outside front matter and code, and whether an HTML
// comment, whose tags don't count, is open there. After an edit it rescans
// from the segment holding the first changed line and stops at the first
// line, past the last changed one, where both the new scan and the saved
// state agree that a segment starts with the same comment state: from there
// on the text and the state are the same as before, so are the colours. Opening or closing a
// fence can therefore recolour to the end of the note, which is correct.
//
// The zero value is ready to use. A Highlighter is not safe for concurrent
// use, and callers must not modify the returned colours (later results share
// the unchanged lines).
type Highlighter struct {
	valid     bool
	th        theme.Theme
	lines     []string
	cols      Colors
	starts    []bool // starts[i]: a segment starts on line i
	comment   []bool // comment[i]: an HTML comment is open where line i starts
	fm        int    // core.FrontMatterEnd of lines
	recolored int
}

// Markdown returns the colours of text, equal to Markdown(text, th).
func (h *Highlighter) Markdown(text string, th theme.Theme) Colors {
	lines := strings.Split(text, "\n")
	fm := core.FrontMatterEnd(lines)
	if !h.valid || h.th != th {
		return h.save(lines, fm, th, 0, nil, nil, nil, len(lines), 0)
	}
	old := h.lines
	p := 0 // first changed line
	for p < len(lines) && p < len(old) && lines[p] == old[p] {
		p++
	}
	if p == len(lines) && p == len(old) {
		h.recolored = 0
		return h.cols
	}
	k := 0 // unchanged lines at the end
	for k < len(lines)-p && k < len(old)-p && lines[len(lines)-1-k] == old[len(old)-1-k] {
		k++
	}
	newEnd, delta := len(lines)-k, len(lines)-len(old)

	// Restart at the segment holding the first changed line; when the text
	// grew at the end, that is the old last line's segment, which an
	// unclosed fence extends. An edit inside the old front matter walks back
	// to line 0 (no segment starts there); one that closes a new front
	// matter (p <= fm) restarts at the top too.
	s := 0
	if p > fm {
		for s = min(p, len(old)-1); s > 0 && !h.starts[s]; s-- {
		}
	}
	return h.save(lines, fm, th, s, h.cols[:s], h.starts[:s], h.comment[:s+1], newEnd, delta)
}

// save rescans lines from s (keeping the colours and states before it, and
// the comment state at s) until it reaches a segment start at or past newEnd
// that the previous scan also started a segment on with the same comment
// state (that line moved by delta), then reuses the rest.
func (h *Highlighter) save(lines []string, fm int, th theme.Theme, s int, keepCols Colors, keepStarts, keepComment []bool, newEnd, delta int) Colors {
	cols := make(Colors, len(lines))
	starts := make([]bool, len(lines))
	comment := make([]bool, len(lines))
	copy(cols, keepCols)
	copy(starts, keepStarts)
	copy(comment, keepComment)
	i := s
	if i <= fm {
		i = colorFrontMatter(lines, fm, cols, th)
	}
	inComment := i == s && i < len(comment) && comment[i]
	for i < len(lines) {
		if i >= newEnd && h.starts[i-delta] && h.comment[i-delta] == inComment {
			copy(cols[i:], h.cols[i-delta:])
			copy(starts[i:], h.starts[i-delta:])
			copy(comment[i:], h.comment[i-delta:])
			break
		}
		starts[i], comment[i] = true, inComment
		i, inComment = segment(lines, i, inComment, cols, th)
	}
	h.recolored = min(i, len(lines)) - s
	h.valid, h.th, h.lines, h.cols, h.starts, h.comment, h.fm = true, th, lines, cols, starts, comment, fm
	return cols
}

// Recolored reports how many lines the last Markdown call coloured afresh.
func (h *Highlighter) Recolored() int { return h.recolored }
