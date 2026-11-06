package core

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// tagURLRe matches a bare URL, whose "#fragment" is not a tag.
	tagURLRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^\s<>]+`)
	// tagMDLinkRe matches a Markdown link or image, text and target both.
	tagMDLinkRe = regexp.MustCompile(`!?\[[^\]]*\]\([^)]*\)`)
)

// TagSpans returns the byte ranges of the inline tags in one line, "#"
// included. A tag is "#" and a letter, then letters, digits, "_", "-" or
// "/" (a trailing "/" is not part of it); it doesn't count after a letter,
// digit, "_", "/", "#" or "&" ("a#b", "##x", "&#123;"), nor inside code
// spans, wiki links, embeds, Markdown links, URLs or an HTML comment that
// opens on the line. "# Heading" is not a tag. Fences, front matter and
// comments spanning lines need the whole note: see Tags.
func TagSpans(line string) [][2]int {
	masked, _ := maskTagLine(line, false)
	return tagSpans(masked)
}

// Tags lists a note's tags: the front matter's "tags:" (as "a, b", "[a, b]"
// or "- a" lines, a leading "#" dropped) and then the inline tags of the
// body, outside fenced code and HTML comments. Tags are lowercased and
// listed once, in order of first appearance.
func Tags(text string) []string {
	return tagsOf(ParseFrontMatter(text))
}

// tagsOf lists the tags of a parsed note, as Tags does. The tags are fresh
// strings, so keeping them doesn't keep the note's text alive.
func tagsOf(fm FrontMatter) []string {
	var out []string
	seen := map[string]bool{}
	add := func(tag string) {
		tag = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tag), "#"))
		if tag != "" && !seen[tag] {
			seen[tag] = true
			out = append(out, strings.Clone(tag))
		}
	}
	if f, ok := fm.Get("tags"); ok {
		for _, it := range f.Items() {
			add(it)
		}
	}
	fence, inComment := "", false
	for _, line := range strings.Split(fm.Body, "\n") {
		if !inComment {
			t := strings.TrimSpace(line)
			if fence != "" {
				if strings.HasPrefix(t, fence) {
					fence = ""
				}
				continue
			}
			if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
				fence = t[:3]
				continue
			}
		}
		var masked string
		masked, inComment = maskTagLine(line, inComment)
		for _, s := range tagSpans(masked) {
			add(line[s[0]+1 : s[1]])
		}
	}
	return out
}

// maskTagLine blanks the parts of line that hold no tags (code spans, HTML
// comments, links and URLs), keeping byte offsets. inComment tells whether
// the line starts inside an HTML comment; the result tells whether one is
// still open at its end.
func maskTagLine(line string, inComment bool) (string, bool) {
	start := 0
	if inComment {
		end := strings.Index(line, "-->")
		if end < 0 {
			return strings.Repeat(" ", len(line)), true
		}
		start = end + 3
	}
	b := []byte(strings.Repeat(" ", start) + stripInlineCode(line[start:]))
	blank := func(i, j int) {
		for ; i < j; i++ {
			b[i] = ' '
		}
	}
	for i := start; ; {
		open := strings.Index(string(b[i:]), "<!--")
		if open < 0 {
			break
		}
		open += i
		end := strings.Index(string(b[open+4:]), "-->")
		if end < 0 {
			blank(open, len(b))
			return string(b), true
		}
		i = open + 4 + end + 3
		blank(open, i)
	}
	for _, re := range []*regexp.Regexp{wikiLinkRe, tagMDLinkRe, tagURLRe} {
		for _, m := range re.FindAllIndex(b, -1) {
			blank(m[0], m[1])
		}
	}
	return string(b), false
}

// tagSpans applies the tag rules to a line already masked by maskTagLine.
func tagSpans(s string) [][2]int {
	var out [][2]int
	for i := 0; i < len(s); i++ {
		if s[i] != '#' {
			continue
		}
		if prev, _ := utf8.DecodeLastRuneInString(s[:i]); i > 0 && (isWordRune(prev) || strings.ContainsRune("/#&", prev)) {
			continue
		}
		if first, _ := utf8.DecodeRuneInString(s[i+1:]); !unicode.IsLetter(first) {
			continue
		}
		j := i + 1
		for j < len(s) {
			r, size := utf8.DecodeRuneInString(s[j:])
			if !isWordRune(r) && !unicode.IsMark(r) && r != '-' && r != '/' {
				break
			}
			j += size
		}
		end := j
		for s[end-1] == '/' {
			end--
		}
		out = append(out, [2]int{i, end})
		i = j - 1
	}
	return out
}

// isWordRune reports whether r is a letter, a digit or "_".
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
