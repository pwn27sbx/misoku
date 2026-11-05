package preview

import (
	"strconv"
	"strings"
)

// isFenceOpen reports whether trimmed opens a ``` or ~~~ code fence.
func isFenceOpen(trimmed string) bool {
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

// fenceEnd returns the index of the line that closes the code fence opened
// at lines[open] (the same three characters), or len(lines) when none does.
func fenceEnd(lines []string, open int) int {
	fence := strings.TrimSpace(lines[open])[:3]
	i := open + 1
	for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), fence) {
		i++
	}
	return i
}

// mathEnd returns the index of the line that closes the $$ equation opened
// at lines[open] (open itself for a one-line equation), or len(lines) when
// none does.
func mathEnd(lines []string, open int) int {
	if strings.Contains(strings.TrimSpace(lines[open])[2:], "$$") {
		return open
	}
	for i := open + 1; i < len(lines); i++ {
		if strings.Contains(lines[i], "$$") {
			return i
		}
	}
	return len(lines)
}

// isTableStart reports whether lines[i] is a table header: a row with a
// "|" followed by a separator row.
func isTableStart(lines []string, i int) bool {
	return strings.Contains(lines[i], "|") && i+1 < len(lines) && tableSepRegex.MatchString(lines[i+1])
}

// tableEnd returns the index of the first line after the table whose
// header is lines[start].
func tableEnd(lines []string, start int) int {
	i := start + 2
	for i < len(lines) && strings.Contains(lines[i], "|") && strings.TrimSpace(lines[i]) != "" {
		i++
	}
	return i
}

// headingNumbers returns the outline number ("1", "1.2") of every heading
// of a note, by line index. It skips the lines compile does not draw as
// headings (code fences, $$ equations, tables, standalone images and
// embeds, block ids), so every number belongs to a heading on screen; and
// it reads the whole note up front, so folding a section never renumbers
// the headings after it.
//
// A heading's depth is one more than the open headings with fewer "#":
// skipped levels leave no zeros ("##" then "####" is 1, 1.1) and the
// shallowest heading present is top level. A lone "#" heading that comes
// first is the note's title: it takes no number and its sections start at
// 1.
func headingNumbers(lines []string) map[int]string {
	type heading struct{ line, level int }
	var headings []heading
	titles := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		switch {
		case isFenceOpen(trimmed):
			i = fenceEnd(lines, i)
			continue
		case strings.HasPrefix(trimmed, "$$"):
			i = mathEnd(lines, i)
			continue
		case isTableStart(lines, i):
			i = tableEnd(lines, i) - 1
			continue
		}
		if _, _, ok := standaloneImage(line); ok {
			continue
		}
		if _, ok := standaloneEmbed(trimmed); ok {
			continue
		}
		if m := headingRegex.FindStringSubmatch(strings.TrimSpace(stripBlockID(line))); m != nil {
			headings = append(headings, heading{i, len(m[1])})
			if len(m[1]) == 1 {
				titles++
			}
		}
	}
	if titles == 1 && len(headings) > 0 && headings[0].level == 1 {
		headings = headings[1:]
	}

	numbers := make(map[int]string, len(headings))
	var open, counts []int // levels of the open headings, counters by depth
	for _, h := range headings {
		for len(open) > 0 && open[len(open)-1] >= h.level {
			open = open[:len(open)-1]
		}
		open = append(open, h.level)
		depth := len(open)
		counts = counts[:min(len(counts), depth)]
		if len(counts) < depth {
			counts = append(counts, 0)
		}
		counts[depth-1]++
		parts := make([]string, depth)
		for j, n := range counts {
			parts[j] = strconv.Itoa(n)
		}
		numbers[h.line] = strings.Join(parts, ".")
	}
	return numbers
}
