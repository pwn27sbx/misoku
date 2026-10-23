package preview

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// dim greys out the text of a rendered frame for a pane without the focus.
// It rewrites the colour codes of the visible rows only, so the compiled
// Markdown is never rendered again: focus changes stay cheap. Backgrounds
// and the shapes that carry a block's colour (pictures, chart bars, callout
// and quote bars, board dots) keep their colours.
func dim(frame string, grey lipgloss.Color) string {
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
		lines[i] = dimLine(l, seq)
	}
	return strings.Join(lines, "\n")
}

// keepsColour reports the block-element and geometric-shape glyphs whose
// foreground is a block's colour rather than text.
func keepsColour(r rune) bool { return r >= 0x2580 && r <= 0x25ff }

// dimLine repaints the text of one line with the SGR foreground grey. Every
// other attribute and escape sequence passes through unchanged.
func dimLine(line, grey string) string {
	if !strings.ContainsRune(line, 0x1b) && strings.TrimSpace(line) == "" {
		return line
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
			fg, rest, reset = splitForeground(params, fg)
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
			if fg != "" && keepsColour(r) {
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
// they reset every attribute.
func splitForeground(params, fg string) (string, []string, bool) {
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
			} else {
				rest = append(rest, p[k:end]...)
			}
			k = end - 1
		default:
			rest = append(rest, t)
		}
	}
	return fg, rest, reset
}
