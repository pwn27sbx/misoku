package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// Dim greys out the text of a rendered frame for a pane without the focus.
// It rewrites the colour codes of the visible rows only, so the pane is never
// rendered again: focus changes stay cheap. Backgrounds and the shapes that
// carry a block's colour (pictures, chart bars, callout and quote bars, board
// dots) keep their colours.
func Dim(frame string, grey lipgloss.Color) string { return dimFrame(frame, grey, false) }

// DimBackdrop greys out a whole frame that a modal box is about to be drawn
// over, with the same pass as Dim but taking every colour away: backgrounds,
// reverse video, the block shapes Dim keeps and pictures (their cells turn
// blank) all go, so the box is the only coloured thing on screen. Behind the
// text shows the frame's base paint, or the terminal's own background with
// ui.transparent. It is one pass over the frame, linear in its size.
func DimBackdrop(frame string, grey lipgloss.Color) string { return dimFrame(frame, grey, true) }

// dimFrame is Dim, or DimBackdrop when backdrop is set.
func dimFrame(frame string, grey lipgloss.Color, backdrop bool) string {
	c := lipgloss.ColorProfile().Color(string(grey))
	if c == nil {
		return frame // no grey in the theme
	}
	seq := c.Sequence(false)
	if seq == "" {
		return frame // no colour support: nothing to dim
	}
	lines := strings.Split(frame, "\n")
	for i, l := range lines {
		lines[i] = dimLine(l, seq, backdrop)
	}
	return strings.Join(lines, "\n")
}

// keepsColour reports the block-element and geometric-shape glyphs whose
// foreground is a block's colour rather than text.
func keepsColour(r rune) bool { return r >= 0x2580 && r <= 0x25ff }

// dimLine repaints the text of one line with the SGR foreground grey. Every
// other attribute and escape sequence passes through unchanged, except that a
// backdrop also loses its backgrounds, block colours and pictures.
func dimLine(line, grey string, backdrop bool) string {
	if !strings.ContainsRune(line, 0x1b) && strings.TrimSpace(line) == "" {
		return line
	}
	// A picture drawn by the terminal is a row of placeholders carrying the
	// image id in their foreground: recolouring them would lose the image. A
	// pane keeps it whole; a backdrop hides it.
	if strings.ContainsRune(line, kittyPlaceholder) {
		if !backdrop {
			return line
		}
		line = blankPictures(line)
	}
	var b strings.Builder
	fg := ""  // the line's own foreground at this point ("" for default)
	cur := "" // the foreground written so far
	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			n, params, sgr := escapeAt(line[i:])
			if !sgr {
				b.WriteString(line[i : i+n])
				i += n
				continue
			}
			var rest []string
			var reset bool
			fg, rest, reset = splitForeground(params, fg, backdrop)
			if reset {
				cur = ""
			}
			if len(rest) > 0 {
				b.WriteString("\x1b[" + strings.Join(rest, ";") + "m")
			}
			i += n
			continue
		}
		r, n := utf8.DecodeRuneInString(line[i:])
		if r != ' ' {
			want := grey
			if fg != "" && !backdrop && keepsColour(r) {
				want = fg
			}
			if want != cur {
				b.WriteString("\x1b[" + want + "m")
				cur = want
			}
		}
		b.WriteString(line[i : i+n])
		i += n
	}
	if cur != "" {
		b.WriteString("\x1b[39m")
	}
	return b.String()
}

// escapeAt measures the escape sequence at the start of s. For an SGR
// sequence ("ESC [ … m") it returns its parameters.
func escapeAt(s string) (n int, params string, sgr bool) {
	if len(s) < 2 {
		return len(s), "", false
	}
	switch s[1] {
	case '[':
		for j := 2; j < len(s); j++ {
			if s[j] >= 0x40 && s[j] <= 0x7e {
				return j + 1, s[2:j], s[j] == 'm'
			}
		}
		return len(s), "", false
	case ']': // OSC, e.g. a hyperlink: ends with BEL or ESC \
		for j := 2; j < len(s); j++ {
			if s[j] == 0x07 {
				return j + 1, "", false
			}
			if s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2, "", false
			}
		}
		return len(s), "", false
	}
	return 2, "", false
}

// splitForeground takes the foreground out of SGR parameters: it returns the
// foreground in effect afterwards, the remaining parameters, and whether
// they reset every attribute. For a backdrop the backgrounds, the underline
// colour and reverse video go too.
func splitForeground(params, fg string, backdrop bool) (string, []string, bool) {
	p := strings.Split(params, ";")
	var rest []string
	reset := false
	for k := 0; k < len(p); k++ {
		switch t := p[k]; {
		case t == "" || t == "0":
			fg, reset = "", true
			rest = append(rest, "0")
		case t == "39":
			fg = ""
		case len(t) == 2 && (t[0] == '3' || t[0] == '9') && t[1] >= '0' && t[1] <= '7':
			fg = t
		case t == "38" || t == "48" || t == "58":
			n := 1 // the colour's own parameters: "5;n" or "2;r;g;b"
			if k+1 < len(p) && p[k+1] == "5" {
				n = 3
			} else if k+1 < len(p) && p[k+1] == "2" {
				n = 5
			}
			end := min(k+n, len(p))
			if t == "38" {
				fg = strings.Join(p[k:end], ";")
			} else if !backdrop {
				rest = append(rest, p[k:end]...)
			}
			k = end - 1
		case backdrop && (t == "7" || isBackground(t)):
		default:
			rest = append(rest, t)
		}
	}
	return fg, rest, reset
}

// isBackground reports the SGR parameters that set a basic background colour.
func isBackground(t string) bool {
	switch len(t) {
	case 2:
		return t[0] == '4' && t[1] >= '0' && t[1] <= '7'
	case 3:
		return t[0] == '1' && t[1] == '0' && t[2] >= '0' && t[2] <= '7'
	}
	return false
}

// kittyPlaceholder is the rune the terminal replaces with a picture (Kitty
// graphics protocol, see internal/kitty).
const kittyPlaceholder = '\U0010EEEE'

// blankPictures turns every picture cell of line, a placeholder followed by
// its row and column diacritics, into one space, so the row keeps its width
// and the terminal draws no image there.
func blankPictures(line string) string {
	var b strings.Builder
	b.Grow(len(line))
	inPicture := false
	for _, r := range line {
		switch {
		case r == kittyPlaceholder:
			b.WriteByte(' ')
			inPicture = true
		case inPicture && unicode.Is(unicode.Mn, r):
		default:
			inPicture = false
			b.WriteRune(r)
		}
	}
	return b.String()
}
