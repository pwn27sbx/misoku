package content

import (
	"regexp"
	"unicode/utf8"
)

// hangMarker matches what starts a list item or a quote: leading
// indentation, then quote marks (">" with or without one space, repeated)
// optionally followed by a list marker, or a list marker alone ("- ", "* ",
// "+ ", "1. ", "1) "), each list marker optionally followed by a task box.
var hangMarker = regexp.MustCompile(
	`^[ \t]*(?:(?:>[ \t]?)+[ \t]*(?:(?:[-*+]|[0-9]+[.)])[ \t]+(?:\[[ xX-]\][ \t]+)?)?|(?:[-*+]|[0-9]+[.)])[ \t]+(?:\[[ xX-]\][ \t]+)?)`)

// hangIndent returns the hanging indent of a line for the editor's soft wrap:
// the rune count of its list or quote marker, so the continuation rows start
// under the item's text. A tab counts as one, as the wrap draws it. Lines
// that are neither get 0. The check is per line, without context, so a list
// line inside a code fence hangs too; it is display only.
func hangIndent(line []rune) int {
	return utf8.RuneCountInString(hangMarker.FindString(string(line)))
}
