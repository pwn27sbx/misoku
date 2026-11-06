package core

import (
	"regexp"
	"strings"
)

// frontMatterKey matches a "key: value" line at column 0, the only field
// shape the front matter reader understands.
var frontMatterKey = regexp.MustCompile(`^([A-Za-z_][\w-]*)\s*:\s*(.*)$`)

// frontMatterItem matches a YAML "- item" line under a key with no value.
var frontMatterItem = regexp.MustCompile(`^\s*-\s+(.*)$`)

// Field is one "key: value" entry of a note's front matter.
type Field struct {
	// Key is the key as the note spells it.
	Key string
	// Value is the scalar after the colon, trimmed of spaces and quotes.
	Value string
	// List holds the items of a "[a, b]" value or of "- a" lines below the
	// key; nil for a plain scalar.
	List []string
	// Line and EndLine are the field's first and last line in the text
	// (they differ only for a "- a" list).
	Line, EndLine int
}

// Items returns the field as a list: its List when it has one, otherwise
// its Value split on commas, with empty items dropped.
func (f Field) Items() []string {
	if f.List != nil {
		return f.List
	}
	return splitItems(f.Value)
}

// FrontMatter is the leading block of a note, opened by a "---" first line
// and closed by the next "---" or "..." line.
type FrontMatter struct {
	// Fields are the block's entries in order.
	Fields []Field
	// Closer is the closing line ("---" or "..."), "" when there is no block.
	Closer string
	// StartLine and EndLine are the opening and closing lines, -1 when
	// there is no block.
	StartLine, EndLine int
	// Body is the text after the block, or the whole text without one.
	Body string
}

// Get returns the field whose key matches key regardless of case; with
// repeated keys the last one wins, as in a map.
func (fm FrontMatter) Get(key string) (Field, bool) {
	for i := len(fm.Fields) - 1; i >= 0; i-- {
		if strings.EqualFold(fm.Fields[i].Key, key) {
			return fm.Fields[i], true
		}
	}
	return Field{}, false
}

// IsFrontMatterOpen reports whether line, as the first line of a note,
// opens a front matter block.
func IsFrontMatterOpen(line string) bool {
	return strings.TrimRight(line, "\r") == "---"
}

// IsFrontMatterClose reports whether line closes an open front matter block.
func IsFrontMatterClose(line string) bool {
	t := strings.TrimRight(line, "\r")
	return t == "---" || t == "..."
}

// FrontMatterEnd returns the line that closes the front matter opened by
// the first line, or -1 when there is none: without a closer the first
// "---" is an ordinary rule.
func FrontMatterEnd(lines []string) int {
	if len(lines) > 1 && IsFrontMatterOpen(lines[0]) {
		for end := 1; end < len(lines); end++ {
			if IsFrontMatterClose(lines[end]) {
				return end
			}
		}
	}
	return -1
}

// ParseFrontMatter reads the front matter of text. Lines that are not
// "key: value" fields or "- item" lines of a field are ignored.
func ParseFrontMatter(text string) FrontMatter {
	lines := strings.Split(text, "\n")
	end := FrontMatterEnd(lines)
	if end < 0 {
		return FrontMatter{StartLine: -1, EndLine: -1, Body: text}
	}
	fm := FrontMatter{
		Closer:    strings.TrimRight(lines[end], "\r"),
		StartLine: 0,
		EndLine:   end,
		Body:      strings.Join(lines[end+1:], "\n"),
	}
	for i := 1; i < end; i++ {
		l := strings.TrimRight(lines[i], "\r")
		if m := frontMatterKey.FindStringSubmatch(l); m != nil {
			raw := strings.TrimSpace(m[2])
			f := Field{Key: m[1], Value: unquote(raw), Line: i, EndLine: i}
			if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
				f.List = splitItems(raw[1 : len(raw)-1])
			}
			fm.Fields = append(fm.Fields, f)
			continue
		}
		last := len(fm.Fields) - 1
		if m := frontMatterItem.FindStringSubmatch(l); m != nil && last >= 0 && fm.Fields[last].Value == "" && fm.Fields[last].EndLine == i-1 {
			fm.Fields[last].List = append(fm.Fields[last].List, unquote(strings.TrimSpace(m[1])))
			fm.Fields[last].EndLine = i
		}
	}
	return fm
}

// SetFrontMatterField sets key to value in text's front matter: it replaces
// the field with that key (any case, keeping the note's spelling), or adds
// it before the block's closer, or creates a block when there is none.
// Values YAML would misread are quoted.
func SetFrontMatterField(text, key, value string) string {
	fm := ParseFrontMatter(text)
	lines := strings.Split(text, "\n")
	// Match the note's line endings: the block's own, or the text's when
	// a new block is created.
	cr := ""
	if fm.EndLine >= 0 && strings.HasSuffix(lines[0], "\r") || fm.EndLine < 0 && strings.Contains(text, "\r\n") {
		cr = "\r"
	}
	eol := cr + "\n"
	entry := func(k string) string {
		if v := quote(value); v != "" {
			return k + ": " + v
		}
		return k + ":"
	}
	if fm.EndLine < 0 {
		return "---" + eol + entry(key) + eol + "---" + eol + text
	}
	if f, ok := fm.Get(key); ok {
		out := append([]string{}, lines[:f.Line]...)
		out = append(out, entry(f.Key)+cr)
		return strings.Join(append(out, lines[f.EndLine+1:]...), "\n")
	}
	out := append([]string{}, lines[:fm.EndLine]...)
	out = append(out, entry(key)+cr)
	return strings.Join(append(out, lines[fm.EndLine:]...), "\n")
}

// quote wraps value in quotes when YAML would read it as something else: a
// mapping (": "), a comment ("#"), a flow list or map, a quoted string, or
// a value with edge spaces. It uses single quotes when value holds '"'.
func quote(value string) string {
	needs := strings.Contains(value, ": ") || strings.TrimSpace(value) != value ||
		value != "" && strings.ContainsRune("#[{'\"", rune(value[0]))
	if !needs {
		return value
	}
	if strings.Contains(value, `"`) {
		return "'" + value + "'"
	}
	return `"` + value + `"`
}

// unquote trims the quotes around a raw value, as the preview always has.
func unquote(raw string) string {
	return strings.Trim(raw, `"'`)
}

// splitItems splits a comma-separated list, trimming spaces and quotes from
// each item and dropping empty ones.
func splitItems(s string) []string {
	var items []string
	for _, it := range strings.Split(s, ",") {
		if it = unquote(strings.TrimSpace(it)); it != "" {
			items = append(items, it)
		}
	}
	return items
}
