package preview

import (
	"sort"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// The helpers below let other renderers (the PDF export) reuse the preview's
// parsing and block drawing.

// controls are the clickable bits of the views, which mean nothing on paper,
// in the interface language; longer labels come first so "+ New page" goes
// before "+ New".
func controls() []string {
	c := []string{
		i18n.T(i18n.PreviewFormAddQuestion), i18n.T(i18n.PreviewFormAddOption), i18n.T(i18n.PreviewAddValue),
		i18n.T(i18n.PreviewNewPage), calendarNav(), i18n.T(i18n.PreviewTimelineNew),
	}
	sort.SliceStable(c, func(i, j int) bool { return len(c[i]) > len(c[j]) })
	return c
}

// editingHint reports whether the plain text of a row is one of the views'
// editing hints ("(empty board: …)"), maybe cut short.
func editingHint(text string) bool {
	text = strings.TrimSuffix(text, "…")
	if text == "" {
		return false
	}
	for _, key := range []i18n.Key{i18n.PreviewBoardEmpty, i18n.PreviewChartEmpty, i18n.PreviewTimelineEmpty, i18n.PreviewFormNoQuestions} {
		if strings.HasPrefix(i18n.T(key), text) {
			return true
		}
	}
	return false
}

// PrintBlockLines draws a special fenced block (board, calendar, timeline,
// chart, form, flow) as styled terminal lines width cells wide, without its
// clickable controls or editing hints. ok is false for ordinary code.
func PrintBlockLines(lang string, body []string, th theme.Theme, width int) (lines []string, ok bool) {
	drawn, _, ok := renderBlock(lang, body, th, width, CalendarView{})
	controls := controls()
	for _, l := range drawn {
		plain := ansi.Strip(l)
		rest, found := plain, false
		for _, c := range controls {
			if strings.Contains(rest, c) {
				rest, found = strings.ReplaceAll(rest, c, ""), true
			}
		}
		hint := strings.TrimSpace(plain)
		if found && strings.Trim(rest, " │") == "" || editingHint(hint) {
			continue // a row holding only controls or an editing hint
		}
		if found {
			for _, c := range controls {
				l = strings.ReplaceAll(l, c, strings.Repeat(" ", ansi.StringWidth(c)))
			}
		}
		lines = append(lines, l)
	}
	for len(lines) > 0 && strings.TrimSpace(ansi.Strip(lines[len(lines)-1])) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, ok
}

// MathRows turns the LaTeX lines of a math block into Unicode rows, one per
// equation line.
func MathRows(body []string) []string {
	var rows []string
	for _, eq := range mathLines(body) {
		rows = append(rows, texToUnicode(eq))
	}
	return rows
}

// IsMathLang reports whether a fence name marks a math block.
func IsMathLang(lang string) bool {
	switch lang {
	case "math", "latex", "tex", "katex":
		return true
	}
	return false
}

// StandaloneImage reports whether a whole line is just an image.
func StandaloneImage(line string) (alt, src string, ok bool) { return standaloneImage(line) }

// ResolveImage turns a Markdown image source into a local file path.
func ResolveImage(src, baseDir string) (string, error) { return resolveImage(src, baseDir) }

// LinkAt parses a Markdown link "[text](url)" at the start of s, returning
// its text, target and length.
func LinkAt(s string) (text, url string, n int, ok bool) { return linkAt(s) }

// StripLayoutTags removes block HTML tags such as <div> and <br>.
func StripLayoutTags(s string) string { return stripLayoutTags(s) }
