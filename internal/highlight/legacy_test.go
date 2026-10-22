package highlight

import (
	"strings"

	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

// legacyMarkdown is Markdown exactly as it was before the incremental
// highlighter split it into segments. The tests compare both on every input
// so the refactor cannot change a single colour.
func legacyMarkdown(text string, th theme.Theme) Colors {
	lines := strings.Split(text, "\n")
	out := make(Colors, len(lines))
	first := 0
	if len(lines) > 1 && strings.TrimRight(lines[0], "\r") == "---" {
		for end := 1; end < len(lines); end++ {
			if t := strings.TrimRight(lines[end], "\r"); t == "---" || t == "..." {
				for j := 0; j <= end; j++ {
					out[j] = frontMatterColors(lines[j], j == 0 || j == end, th)
				}
				first = end + 1
				break
			}
		}
	}
	for i := first; i < len(lines); i++ {
		line := lines[i]
		cols := make([]lipgloss.Color, len([]rune(line)))
		out[i] = cols
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence := trimmed[:3]
			lang := strings.TrimSpace(trimmed[3:])
			fill(cols, 0, len(cols), th.GreyFg2)
			start := i + 1
			end := start
			for end < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[end]), fence) {
				end++
			}
			code := Code(lang, strings.Join(lines[start:end], "\n"), th)
			for j := start; j < end; j++ {
				out[j] = code[j-start]
			}
			if end < len(lines) {
				closing := make([]lipgloss.Color, len([]rune(lines[end])))
				fill(closing, 0, len(closing), th.GreyFg2)
				out[end] = closing
			}
			i = end
			continue
		}

		switch {
		case headingRe.MatchString(line):
			level := len(headingRe.FindStringSubmatch(line)[1])
			fill(cols, 0, len(cols), headingColor(level, th))
			continue
		case ruleRe.MatchString(line):
			fill(cols, 0, len(cols), th.GreyFg)
			continue
		case quoteRe.MatchString(line):
			idx := len([]rune(line[:strings.Index(line, ">")]))
			fill(cols, idx, idx+1, th.Purple)
		}
		if m := listRe.FindStringSubmatchIndex(line); m != nil {
			fill(cols, runeIdx(line, m[4]), runeIdx(line, m[5]), th.Red)
			if m[8] >= 0 {
				box := th.GreyFg2
				if strings.ContainsAny(line[m[8]:m[9]], "xX") {
					box = th.Green
				}
				fill(cols, runeIdx(line, m[8]), runeIdx(line, m[9]), box)
			}
		}
		inline(line, cols, th)
	}
	return out
}
