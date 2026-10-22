// Package testnote builds long Markdown notes for tests and benchmarks.
//
// The notes are deterministic: the same line count always gives the same
// text, so benchmark runs before and after a change measure the same input.
package testnote

import (
	"fmt"
	"strings"
)

// Note returns a Markdown note of exactly n lines (no trailing newline).
// It cycles through headings, paragraphs, long lines that soft-wrap, lists,
// task lists, tables, fenced code blocks and quotes with links. Sections are
// only added whole, so code fences are always balanced; the last few lines
// are filled with plain paragraphs. n <= 0 gives an empty note.
func Note(n int) string {
	if n <= 0 {
		return ""
	}
	lines := make([]string, 0, n)
	for i := 0; len(lines) < n; i++ {
		sec := section(i)
		if len(lines)+len(sec) > n {
			for j := 0; len(lines) < n; j++ {
				lines = append(lines, fmt.Sprintf("Filler line %d keeps the note at its exact length.", j+1))
			}
			break
		}
		lines = append(lines, sec...)
	}
	return strings.Join(lines, "\n")
}

// section returns the i-th section of the cycle, ending with a blank line.
func section(i int) []string {
	n := i + 1
	switch i % 7 {
	case 0:
		return []string{
			fmt.Sprintf("# Chapter %d", n),
			"",
			fmt.Sprintf("Opening paragraph %d with **bold**, _italic_ and `inline code`.", n),
			"",
		}
	case 1:
		return []string{
			fmt.Sprintf("## Section %d", n),
			"",
			// A long line: it soft-wraps over several rows in any pane.
			strings.Repeat(fmt.Sprintf("long line %d wraps across the editor width ", n), 8),
			"",
		}
	case 2:
		return []string{
			fmt.Sprintf("- item %d", n),
			"- [ ] open task",
			"- [x] done task",
			"  - nested item",
			fmt.Sprintf("1. ordered step %d", n),
			"",
		}
	case 3:
		return []string{
			"| name | count | note |",
			"| --- | ---: | :-- |",
			fmt.Sprintf("| alpha | %d | first row |", n),
			fmt.Sprintf("| beta | %d | second row |", n*2),
			fmt.Sprintf("| gamma | %d | third row |", n*3),
			"",
		}
	case 4:
		return []string{
			"```go",
			fmt.Sprintf("func example%d() int {", n),
			"\t// a comment inside a code block",
			fmt.Sprintf("\treturn %d", n),
			"}",
			"```",
			"",
		}
	case 5:
		return []string{
			fmt.Sprintf("### Notes %d", n),
			fmt.Sprintf("> A quote that links to [[note-%d]] and [a site](https://example.com/%d).", n, n),
			"",
		}
	default:
		return []string{
			fmt.Sprintf("Paragraph %d: plain prose that fits on one row.", n),
			"Another plain line right after it.",
			"",
		}
	}
}
