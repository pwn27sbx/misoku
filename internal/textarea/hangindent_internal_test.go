package textarea

import (
	"strings"
	"testing"
	"unicode"

	"github.com/rivo/uniseg"
)

// V1: the hanging indent keeps the signatures every caller and test uses.
var (
	_ func([]rune, int) [][]rune        = wrap
	_ func(Model, []rune, int) [][]rune = Model.memoizedWrap
)

// V2/V3: with any indent the segments still hold every rune of the line plus
// one trailing space, the indent never enters them, and no continuation row
// is wider than the room left beside the indent (bar its trailing space).
func TestWrapIndentKeepsRunesAndRoom(t *testing.T) {
	lines := []string{
		"- [ ] alpha beta gamma delta epsilon zeta",
		"- " + strings.Repeat("a", 45),
		"  - nested " + strings.Repeat("word ", 12),
		"> > - [x] quoted task with a few more words to wrap around",
		"",
		"-",
		"x " + strings.Repeat("bb ", 30),
	}
	for _, text := range lines {
		runes := []rune(text)
		for width := 4; width <= 30; width++ {
			for indent := 0; indent <= width/2; indent++ {
				segs := wrapIndent(runes, width, indent)
				var joined []rune
				for _, s := range segs {
					joined = append(joined, s...)
				}
				if string(joined) != text+" " {
					t.Fatalf("%q width %d indent %d: segments join to %q", text, width, indent, string(joined))
				}
				for k, s := range segs {
					room := width
					if k > 0 {
						room -= indent
					}
					w := uniseg.StringWidth(strings.TrimRightFunc(string(s), unicode.IsSpace))
					if w > room {
						t.Fatalf("%q width %d indent %d: row %d is %d wide, room %d", text, width, indent, k, w, room)
					}
				}
				if indent == 0 && len(wrap(runes, width)) != len(segs) {
					t.Fatalf("%q width %d: wrapIndent(…, 0) differs from wrap", text, width)
				}
			}
		}
	}
}
