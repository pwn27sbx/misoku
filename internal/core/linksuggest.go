package core

import (
	"path"
	"sort"
	"strings"
	"time"
)

// Kinds of link suggestions, as LinkSuggestion.Kind holds them.
const (
	LinkKindNote    = "note"
	LinkKindHeading = "heading"
	LinkKindBlock   = "block"
)

// LinkNote is a note the "[[" link suggestions can offer: its ID, title and
// text, the text being where its headings and block ids are read.
type LinkNote struct {
	ID        string
	Title     string
	Text      string
	UpdatedAt time.Time
}

// LinkSuggestion is one completion of a "[[" link. Insert is what goes
// between "[[" and "]]", written so ParseWikiLink reads it back.
type LinkSuggestion struct {
	Insert string // "Docker", "#S3 Storage", "AWS Notes#^s3-basics"
	Label  string // the note name, heading text or block id the list shows
	Detail string // the folder, note or line text that tells it apart
	Kind   string // LinkKindNote, LinkKindHeading or LinkKindBlock
	NoteID string // the note it belongs to
}

// BlockID is a block marker of a note: its id and the line it marks.
type BlockID struct {
	ID   string
	Text string // the marked line's text, trimmed
}

// LinkSuggestions reads the text typed after "[[" and returns the links it
// can complete, following the link grammar ParseWikiLink reads:
//
//	name…      notes, by name
//	#text      headings of the note being edited, all levels
//	Note#text  headings of that note
//	#^text     block ids of the note being edited
//	Note#^text block ids of that note
//
// Notes are resolved with ResolveNote, headings and blocks with the same
// rules LinkLine follows. current is the ID of the note being edited and
// currentText its text as it stands in the editor, which wins over the text
// the notes list holds for it.
func LinkSuggestions(notes []LinkNote, current, currentText, query string) []LinkSuggestion {
	target, frag, hasHash := strings.Cut(query, "#")
	if !hasHash {
		return noteSuggestions(notes, strings.TrimSpace(query))
	}
	target = strings.TrimSpace(target)
	note, text, prefix, ok := linkTarget(notes, target, current, currentText)
	if !ok {
		return nil
	}
	if id, isBlock := strings.CutPrefix(frag, "^"); isBlock {
		return blockSuggestions(text, prefix, note.ID, id)
	}
	return headingSuggestions(text, prefix, note.ID, frag)
}

// linkTarget finds the note a "#" follows: the note being edited when the
// target is empty, another one otherwise. It returns that note, its text and
// the name a link to it writes before the "#".
func linkTarget(notes []LinkNote, target, current, currentText string) (LinkNote, string, string, bool) {
	if target == "" {
		return LinkNote{ID: current}, currentText, "", true
	}
	p, ok := ResolveNote(linkPages(notes), target, current)
	if !ok {
		return LinkNote{}, "", "", false
	}
	if p.ID == current {
		return LinkNote{ID: p.ID, Title: p.Title}, currentText, "", true
	}
	for _, n := range notes {
		if n.ID == p.ID {
			return n, n.Text, linkInsertName(notes, n), true
		}
	}
	return LinkNote{}, "", "", false
}

// linkPages is the notes as Page values, for ResolveNote.
func linkPages(notes []LinkNote) []Page {
	pages := make([]Page, len(notes))
	for i, n := range notes {
		pages[i] = Page{ID: n.ID, Title: n.Title}
	}
	return pages
}

// linkInsertName is the name a link to n writes: its title, or its path when
// another note has the same title, as the finder's link picker does.
func linkInsertName(notes []LinkNote, n LinkNote) string {
	return linkInsertNameCounted(n, titleCounts(notes))
}

// titleCounts counts the notes behind each title, ignoring letter case, so a
// link to one of several notes with the same title writes its path.
func titleCounts(notes []LinkNote) map[string]int {
	counts := make(map[string]int, len(notes))
	for _, n := range notes {
		counts[strings.ToLower(n.Title)]++
	}
	return counts
}

// linkInsertNameCounted is linkInsertName with the titles already counted.
func linkInsertNameCounted(n LinkNote, counts map[string]int) string {
	if n.Title == "" || counts[strings.ToLower(n.Title)] > 1 {
		return strings.TrimSuffix(n.ID, mdExt)
	}
	return n.Title
}

// noteSuggestions lists the notes whose name matches q: an exact title, then
// a title that starts with it, then one that contains it, then a path that
// contains it, then a fuzzy match. Newest first among equals.
func noteSuggestions(notes []LinkNote, q string) []LinkSuggestion {
	type hit struct {
		suggestion LinkSuggestion
		score      int
		updated    time.Time
	}
	items := make([]hit, 0, len(notes))
	counts := titleCounts(notes)
	for _, n := range notes {
		score, ok := noteScore(q, n)
		if !ok {
			continue
		}
		items = append(items, hit{suggestion: LinkSuggestion{
			Insert: linkInsertNameCounted(n, counts),
			Label:  noteTitle(n),
			Detail: path.Dir(n.ID),
			Kind:   LinkKindNote,
			NoteID: n.ID,
		}, score: score, updated: n.UpdatedAt})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score < items[j].score
		}
		if !items[i].updated.Equal(items[j].updated) {
			return items[i].updated.After(items[j].updated)
		}
		return items[i].suggestion.NoteID < items[j].suggestion.NoteID
	})
	out := make([]LinkSuggestion, len(items))
	for i := range items {
		s := items[i].suggestion
		if s.Detail == "." {
			s.Detail = ""
		}
		out[i] = s
	}
	return out
}

// noteScore ranks a note against a typed name, lower being better.
func noteScore(q string, n LinkNote) (int, bool) {
	if q == "" {
		return 0, true
	}
	want := strings.ToLower(q)
	title := strings.ToLower(noteTitle(n))
	id := strings.ToLower(strings.TrimSuffix(n.ID, mdExt))
	switch {
	case title == want:
		return 0, true
	case strings.HasPrefix(title, want):
		return 1, true
	case strings.Contains(title, want):
		return 2, true
	case strings.Contains(id, want):
		return 3, true
	case isSubsequence(want, title):
		return 4, true
	}
	return 0, false
}

// isSubsequence reports whether every rune of want appears in have, in order.
func isSubsequence(want, have string) bool {
	rest := []rune(have)
	i := 0
	for _, r := range want {
		for i < len(rest) && rest[i] != r {
			i++
		}
		if i == len(rest) {
			return false
		}
		i++
	}
	return true
}

// noteTitle is the name a note shows: its title, or its file name.
func noteTitle(n LinkNote) string {
	if n.Title != "" {
		return n.Title
	}
	return path.Base(strings.TrimSuffix(n.ID, mdExt))
}

// headingSuggestions lists the headings of text that contain filter, in the
// note, each once: two headings that write the same text complete the same
// link.
func headingSuggestions(text, prefix, noteID, filter string) []LinkSuggestion {
	var out []LinkSuggestion
	seen := map[string]bool{}
	for _, h := range Headings(text) {
		if !containsFold(h, filter) {
			continue
		}
		heading := linkHeading(h)
		insert := prefix + "#" + heading
		if heading == "" || seen[insert] {
			continue
		}
		seen[insert] = true
		out = append(out, LinkSuggestion{
			Insert: insert, Label: h, Detail: prefix, Kind: LinkKindHeading, NoteID: noteID,
		})
	}
	return out
}

// blockSuggestions lists the block ids of text that contain filter, in the
// note, each shown with the line it marks.
func blockSuggestions(text, prefix, noteID, filter string) []LinkSuggestion {
	var out []LinkSuggestion
	seen := map[string]bool{}
	for _, b := range BlockIDs(text) {
		if seen[b.ID] || !containsFold(b.ID, filter) {
			continue
		}
		seen[b.ID] = true
		out = append(out, LinkSuggestion{
			Insert: prefix + "#^" + b.ID, Label: "^" + b.ID, Detail: b.Text,
			Kind: LinkKindBlock, NoteID: noteID,
		})
	}
	return out
}

// linkHeading writes heading text so a link reads it back: the characters
// that would end the link ("[", "]") or split it ("|") are dropped, and so is
// a leading "^", which would make it a block id. Dropping them leaves
// headingKey matching the heading as written.
func linkHeading(h string) string {
	h = strings.TrimSpace(h)
	h = strings.TrimLeft(h, "^")
	h = strings.NewReplacer("[", "", "]", "", "|", "").Replace(h)
	return strings.TrimSpace(h)
}

// containsFold reports whether s holds sub, ignoring letter case.
func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// LinkCompletion is what Enter inserts when no suggestion matched: a plain
// typed name becomes a link to a note that does not exist yet, which
// following starts. An empty query, or one naming a heading or a block,
// inserts nothing: no note holds it.
func LinkCompletion(query string) string {
	q := strings.TrimSpace(query)
	if q == "" || strings.ContainsAny(q, "#\n") {
		return ""
	}
	return q
}

// Headings returns the text of every heading in text, in order and as
// written, without its "#" marks; fenced code blocks are skipped.
func Headings(text string) []string {
	var out []string
	for _, line := range markdownLines(text) {
		if m := headingRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			out = append(out, m[2])
		}
	}
	return out
}

// BlockIDs returns every block marker of text, in order; fenced code blocks
// are skipped. A marker alone on its line marks the block above it, whose
// text is reported.
func BlockIDs(text string) []BlockID {
	lines := markdownLines(text)
	var out []BlockID
	for i, line := range lines {
		t := strings.TrimSpace(line)
		m := BlockIDRe.FindStringSubmatchIndex(t)
		if m == nil {
			continue
		}
		marked := strings.TrimSpace(t[:m[0]])
		if marked == "" {
			// The marker is alone on its line: it marks the block above,
			// whose last text line is what the list shows.
			for j := i - 1; j >= 0; j-- {
				if prev := strings.TrimSpace(lines[j]); prev != "" {
					marked = prev
					break
				}
			}
		}
		out = append(out, BlockID{ID: t[m[4]:m[5]], Text: marked})
	}
	return out
}

// markdownLines returns the lines of text outside fenced code blocks, the
// fences included.
func markdownLines(text string) []string {
	var out []string
	fence := ""
	for _, line := range strings.Split(text, "\n") {
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
		out = append(out, line)
	}
	return out
}
