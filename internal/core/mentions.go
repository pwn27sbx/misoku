package core

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// UnlinkedMention is a note that names another note's title in its text
// without linking to it.
type UnlinkedMention struct {
	ID, Title string
	// Line is the 1-based line of the first mention, Text that line trimmed.
	Line int
	Text string
	// Count is how many mentions the note holds.
	Count int
	// Content is the note's text as it was scanned: linking the mentions
	// rewrites it, and only while the file still holds it (ApplyRelinks).
	Content string
}

// mentionMatcher finds one title in notes; build it once per title.
type mentionMatcher struct{ re *regexp.Regexp }

// newMentionMatcher matches title in any letter case. A title a wiki link
// can't hold ("[", "]", "|", "#" or a line break) matches nothing: its
// mentions could never become links.
func newMentionMatcher(title string) mentionMatcher {
	title = strings.TrimSpace(title)
	if title == "" || strings.ContainsAny(title, "[]|#\n\r") {
		return mentionMatcher{}
	}
	return mentionMatcher{re: regexp.MustCompile(`(?i)` + regexp.QuoteMeta(title))}
}

// Mentions returns the byte ranges in text where title is written as a
// whole word, in any letter case. Word edges are Unicode letters, digits
// and combining marks, so "Diseño" is not found in "rediseño". The front
// matter, fenced and inline code, wiki links and embeds, Markdown links,
// URLs, HTML comments and inline tags hold no mentions; nor do the parts
// of an email, a domain or a path ("juan@zentro.com", "www.zentro.com",
// "Proyectos/Diseño.md"), a reference link's text ("[Diseño][1]", a
// "[Diseño]: …" definition) or a block id ("^abc"): linking them would
// break them.
func Mentions(text, title string) [][2]int {
	return newMentionMatcher(title).find(text)
}

func (mm mentionMatcher) find(text string) [][2]int {
	if mm.re == nil {
		return nil
	}
	var out [][2]int
	lines := strings.Split(text, "\n")
	first := FrontMatterEnd(lines) + 1 // 0 without front matter
	fence, inComment, off := "", false, 0
	for i, line := range lines {
		start := off
		off += len(line) + 1
		if i < first {
			continue
		}
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
		// The same masking as tags: code spans, comments, links and URLs
		// are blanked, keeping byte offsets; then the tags themselves.
		var masked string
		masked, inComment = maskTagLine(line, inComment)
		if spans := tagSpans(masked); len(spans) > 0 {
			b := []byte(masked)
			for _, sp := range spans {
				for j := sp[0]; j < sp[1]; j++ {
					b[j] = ' '
				}
			}
			masked = string(b)
		}
		for _, r := range mm.words(masked) {
			out = append(out, [2]int{start + r[0], start + r[1]})
		}
	}
	return out
}

// words finds the matches in s that stand as whole words. A match that
// fails the word edges is retried one rune later, so a candidate glued to
// a word never hides a real mention that overlaps it.
func (mm mentionMatcher) words(s string) [][2]int {
	var out [][2]int
	for i := 0; i < len(s); {
		loc := mm.re.FindStringIndex(s[i:])
		if loc == nil {
			break
		}
		a, b := i+loc[0], i+loc[1]
		if mentionEdges(s, a, b) {
			out = append(out, [2]int{a, b})
			i = b
			continue
		}
		_, size := utf8.DecodeRuneInString(s[a:])
		i = a + size
	}
	return out
}

// mentionEdges reports whether s[a:b] stands alone: no word goes on at
// either edge, and it is not part of an email, a domain or a path, a tag
// or a block id, nor the text of a reference link.
func mentionEdges(s string, a, b int) bool {
	before, _ := utf8.DecodeLastRuneInString(s[:a])
	after, size := utf8.DecodeRuneInString(s[b:])
	if a > 0 && (isMentionRune(before) || strings.ContainsRune("@./#^", before)) {
		return false
	}
	if b == len(s) {
		return true
	}
	if isMentionRune(after) {
		return false
	}
	// "zentro.com", "juan@x", "Diseño/notas": joined to what follows. A
	// "." or "/" that ends the sentence or the line leaves the word alone.
	if strings.ContainsRune(".@/", after) {
		if next, _ := utf8.DecodeRuneInString(s[b+size:]); isMentionRune(next) {
			return false
		}
	}
	// "[Diseño][1]", "[Diseño](…)" and "[Diseño]: …" are link syntax.
	if before == '[' && after == ']' && b+1 < len(s) && strings.ContainsRune("[(:", rune(s[b+1])) {
		return false
	}
	return true
}

// isMentionRune reports whether r continues a word: a letter, a digit or a
// combining mark (the accent of a decomposed "ñ").
func isMentionRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r)
}

// LinkMentions turns every mention of title in text (see Mentions) into a
// wiki link to target: "[[target]]" when the mention is written as target,
// otherwise "[[target|mention]]", so the text reads the same. It returns
// the new text and the number of links made.
func LinkMentions(text, title, target string) (string, int) {
	ms := Mentions(text, title)
	if len(ms) == 0 {
		return text, 0
	}
	var b strings.Builder
	last := 0
	for _, r := range ms {
		b.WriteString(text[last:r[0]])
		orig := text[r[0]:r[1]]
		if orig == target {
			b.WriteString("[[" + target + "]]")
		} else {
			b.WriteString("[[" + target + "|" + orig + "]]")
		}
		last = r[1]
	}
	b.WriteString(text[last:])
	return b.String(), len(ms)
}

// FindUnlinkedMentions lists the notes among notes that mention target's
// title (see Mentions) and don't link to target, sorted by title. Folders,
// target itself, notes skip reports (such as templates) and notes read
// fails on are left out. read returns a note's text and may be called
// from any goroutine the caller allows.
func FindUnlinkedMentions(notes []Page, target Page, read func(id string) (string, error), skip func(id string) bool) []UnlinkedMention {
	mm := newMentionMatcher(target.Title)
	if mm.re == nil {
		return nil
	}
	var out []UnlinkedMention
	for _, p := range notes {
		if p.IsFolder || p.ID == target.ID || (skip != nil && skip(p.ID)) {
			continue
		}
		text, err := read(p.ID)
		if err != nil {
			continue
		}
		ms := mm.find(text)
		if len(ms) == 0 || linksTo(notes, Page{ID: p.ID, Title: p.Title, Content: text}, target.ID) {
			continue
		}
		lineStart := strings.LastIndexByte(text[:ms[0][0]], '\n') + 1
		lineEnd := len(text)
		if j := strings.IndexByte(text[ms[0][0]:], '\n'); j >= 0 {
			lineEnd = ms[0][0] + j
		}
		out = append(out, UnlinkedMention{
			ID:      p.ID,
			Title:   p.Title,
			Line:    strings.Count(text[:ms[0][0]], "\n") + 1,
			Text:    strings.TrimSpace(text[lineStart:lineEnd]),
			Count:   len(ms),
			Content: text,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title) })
	return out
}

// linksTo reports whether from links to the note id, as Backlinks counts.
func linksTo(notes []Page, from Page, id string) bool {
	for _, to := range OutLinks(notes, from) {
		if to == id {
			return true
		}
	}
	return false
}
