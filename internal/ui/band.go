package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Band paints the cursor-line band bg behind a rendered row and pads the row
// to width with it, the way the editor bands the cursor's raw line. Like Dim
// it rewrites colour codes only: the band is put back after every reset,
// cells that carry a background of their own (code, bands, a calendar's day,
// reverse video) keep it, and picture cells are left exactly as they were.
// Without colour support the row comes back unchanged.
func Band(row string, bg lipgloss.Color, width int) string {
	c := lipgloss.ColorProfile().Color(string(bg))
	if c == nil {
		return row
	}
	seq := c.Sequence(true)
	if seq == "" {
		return row
	}
	band := "\x1b[" + seq + "m"
	var b strings.Builder
	b.Grow(len(row) + 4*len(band))
	own, reverse := false, false // the row's own background, reverse video
	on := false                  // the band is in effect
	inPicture := false
	for i := 0; i < len(row); {
		if row[i] == 0x1b {
			n, params, sgr := escapeAt(row[i:])
			b.WriteString(row[i : i+n])
			i += n
			if sgr {
				var touched bool
				own, reverse, touched = backgroundAfter(params, own, reverse)
				if touched {
					on = false
				}
			}
			continue
		}
		r, n := utf8.DecodeRuneInString(row[i:])
		switch {
		case r == kittyPlaceholder:
			inPicture = true
		case inPicture && unicode.Is(unicode.Mn, r):
		default:
			inPicture = false
		}
		switch {
		case inPicture && on:
			b.WriteString("\x1b[49m")
			on = false
		case !inPicture && !on && !own && !reverse:
			b.WriteString(band)
			on = true
		}
		b.WriteString(row[i : i+n])
		i += n
	}
	if pad := width - ansi.StringWidth(row); pad > 0 {
		b.WriteString("\x1b[0m" + band + strings.Repeat(" ", pad) + "\x1b[0m")
	} else if on {
		b.WriteString("\x1b[49m")
	}
	return b.String()
}

// backgroundAfter reads SGR parameters for what Band needs: whether a
// background of the row's own and reverse video are in effect afterwards,
// and whether the parameters touched the background at all (a reset, a
// colour, reverse video), which ends the band until it is written again.
func backgroundAfter(params string, own, reverse bool) (bool, bool, bool) {
	p := strings.Split(params, ";")
	touched := false
	for k := 0; k < len(p); k++ {
		switch t := p[k]; {
		case t == "" || t == "0":
			own, reverse, touched = false, false, true
		case t == "49":
			own, touched = false, true
		case t == "7":
			reverse, touched = true, true
		case t == "27":
			reverse, touched = false, true
		case isBackground(t):
			own, touched = true, true
		case t == "38" || t == "48" || t == "58":
			n := 1 // the colour's own parameters: "5;n" or "2;r;g;b"
			if k+1 < len(p) && p[k+1] == "5" {
				n = 3
			} else if k+1 < len(p) && p[k+1] == "2" {
				n = 5
			}
			if t == "48" {
				own, touched = true, true
			}
			k = min(k+n, len(p)) - 1
		}
	}
	return own, reverse, touched
}
