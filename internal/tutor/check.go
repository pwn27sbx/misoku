package tutor

import (
	"regexp"
	"strings"
)

var (
	headingRe   = regexp.MustCompile(`^ {0,3}(#{1,6})[ \t]+(.*?)[ \t#]*$`)
	taskDoneRe  = regexp.MustCompile(`^\s*[-*+] \[[xX]\]\s+\S`)
	taskNoSpace = regexp.MustCompile(`^\s*[-*+] \[\]`)
	delimRe     = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)
	wikiRe      = regexp.MustCompile(`\[\[([^\[\]\n]*)\]\]`)
	blockIDRe   = regexp.MustCompile(`\s\^([A-Za-z0-9_-]+)\s*$`)
)

// hintNoSpace is the toast shown when a task is written without its space.
const hintNoSpace = "- [] no es una tarea: falta el espacio → - [ ]"

// Passed reports whether text, a lesson note's content, fulfils check.
// exists reports whether the vault has a note with that name (no ".md").
// Code blocks and inline code are ignored, so a lesson's own examples never
// pass it. An empty or unknown check never passes.
func Passed(check, text string, exists func(name string) bool) bool {
	lines := stripCode(text)
	switch check {
	case "subheading":
		h2 := false
		for _, h := range headings(lines) {
			switch h.level {
			case 2:
				h2 = true
			case 3:
				if h2 {
					return true
				}
			}
		}
	case "checkbox":
		return anyLine(lines, taskDoneRe)
	case "table":
		for _, l := range lines {
			if strings.Contains(l, "|") && delimRe.MatchString(l) {
				return true
			}
		}
	case "wikilink":
		for _, k := range links(lines) {
			if k.note != "" {
				return true
			}
		}
	case "headinglink":
		titles := map[string]bool{}
		for _, h := range headings(lines) {
			titles[strings.ToLower(strings.TrimSpace(h.text))] = true
		}
		for _, k := range links(lines) {
			if k.note == "" && k.hasHash && !strings.HasPrefix(k.sub, "^") && k.sub != "" && titles[strings.ToLower(k.sub)] {
				return true
			}
		}
	case "blockref":
		ids := map[string]bool{}
		for _, l := range lines {
			if m := blockIDRe.FindStringSubmatch(l); m != nil {
				ids[m[1]] = true
			}
		}
		for _, k := range links(lines) {
			if k.note == "" && strings.HasPrefix(k.sub, "^") && ids[k.sub[1:]] {
				return true
			}
		}
	case "newnote":
		for _, k := range links(lines) {
			if strings.EqualFold(k.note, "Mi nota") && exists != nil && exists(k.note) {
				return true
			}
		}
	}
	return false
}

// Hint returns a tip for a common mistake in a lesson note, or "".
func Hint(file, text string) string {
	if lessonFile(file) == "B02 Casillas.md" && anyLine(stripCode(text), taskNoSpace) {
		return hintNoSpace
	}
	return ""
}

func anyLine(lines []string, re *regexp.Regexp) bool {
	for _, l := range lines {
		if re.MatchString(l) {
			return true
		}
	}
	return false
}

type heading struct {
	level int
	text  string
}

func headings(lines []string) []heading {
	var hs []heading
	for _, l := range lines {
		if m := headingRe.FindStringSubmatch(l); m != nil {
			hs = append(hs, heading{level: len(m[1]), text: m[2]})
		}
	}
	return hs
}

// link is a parsed [[note#sub|alias]].
type link struct {
	note    string
	sub     string
	hasHash bool
}

func links(lines []string) []link {
	var ks []link
	for _, l := range lines {
		for _, m := range wikiRe.FindAllStringSubmatch(l, -1) {
			target, _, _ := strings.Cut(m[1], "|")
			note, sub, hash := strings.Cut(target, "#")
			ks = append(ks, link{note: strings.TrimSpace(note), sub: strings.TrimSpace(sub), hasHash: hash})
		}
	}
	return ks
}

// stripCode splits text into lines, blanking fenced code blocks (``` or ~~~)
// and removing inline code spans.
func stripCode(text string) []string {
	lines := strings.Split(text, "\n")
	fence := ""
	for i, l := range lines {
		t := strings.TrimLeft(l, " ")
		if fence != "" {
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]+" \t") == "" {
				fence = ""
			}
			lines[i] = ""
			continue
		}
		if f := fenceOpen(t); f != "" {
			fence = f
			lines[i] = ""
			continue
		}
		lines[i] = stripInline(l)
	}
	return lines
}

func fenceOpen(t string) string {
	for _, c := range []byte{'`', '~'} {
		n := 0
		for n < len(t) && t[n] == c {
			n++
		}
		if n >= 3 {
			return t[:n]
		}
	}
	return ""
}

// stripInline removes `code` spans: a run of n backticks up to the next run of
// exactly n. An unmatched run is kept as text.
func stripInline(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '`' {
			b.WriteByte(s[i])
			i++
			continue
		}
		n := run(s, i)
		end := -1
		for j := i + n; j < len(s); {
			if s[j] == '`' {
				m := run(s, j)
				if m == n {
					end = j + m
					break
				}
				j += m
				continue
			}
			j++
		}
		if end < 0 {
			b.WriteString(s[i : i+n])
			i += n
			continue
		}
		i = end
	}
	return b.String()
}

func run(s string, i int) int {
	n := 0
	for i+n < len(s) && s[i+n] == '`' {
		n++
	}
	return n
}
