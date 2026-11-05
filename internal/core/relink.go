package core

import (
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Move renames or moves a note or a folder. From and To are IDs from the
// workspace root ("Projects/AWS.md", "Projects"). A note moves with its
// sub-note folder ("Projects/AWS/"), and a folder with everything in it.
type Move struct{ From, To string }

// Relink is the rewrite of one note's links after a set of moves.
type Relink struct {
	ID       string // the note's ID after the moves
	From, To string // its text before and after the rewrite
	Links    int    // links rewritten
}

// movePath maps an ID or file path through moves; paths no move touches
// come back unchanged.
func movePath(moves []Move, p string) string {
	for _, m := range moves {
		if p == m.From {
			return m.To
		}
		from, to := strings.TrimSuffix(m.From, mdExt), strings.TrimSuffix(m.To, mdExt)
		if rest, ok := strings.CutPrefix(p, from+"/"); ok {
			return to + "/" + rest
		}
	}
	return p
}

// ExpandMoves lists the move of every note in before that moves: the notes
// moves name, their sub-notes and the notes in moved folders, in list order.
func ExpandMoves(before []Page, moves []Move) []Move {
	var out []Move
	for _, p := range before {
		if to := movePath(moves, p.ID); !p.IsFolder && to != p.ID {
			out = append(out, Move{p.ID, to})
		}
	}
	return out
}

// PlanRelinks rewrites the links of every note whose text is in content
// (keyed by its ID before the moves) so that each link that resolved to a
// note before the moves resolves to that same note, moved, afterwards:
// links to moved notes, a moved note's own relative links, links another
// note's new name would capture, and Markdown links to files (images,
// attachments) that moved or that a moved note left behind. Wiki links get
// the bare note name when it resolves, else the path from the root;
// Markdown links get the path relative to the linking note. Aliases,
// headings, block ids, "!", ".md", "<…>" and %-escaping stay as written;
// links in code and links that resolve to nothing are left alone. Only
// notes with a rewritten link are returned, in list order.
func PlanRelinks(before []Page, moves []Move, content map[string]string) []Relink {
	var notes, after []Page
	for _, p := range before {
		if !p.IsFolder {
			notes = append(notes, p)
			after = append(after, Page{ID: movePath(moves, p.ID)})
		}
	}
	r := relinker{moves: moves, before: lookup(notes), after: lookup(after)}
	var out []Relink
	for _, p := range notes {
		text, ok := content[p.ID]
		if !ok {
			continue
		}
		newID := movePath(moves, p.ID)
		if to, n := r.note(text, p.ID, newID); n > 0 {
			out = append(out, Relink{ID: newID, From: text, To: to, Links: n})
		}
	}
	return out
}

// lookup indexes notes for resolving wiki links like LinkIndex does.
func lookup(notes []Page) *LinkIndex {
	x := &LinkIndex{notes: notes}
	x.buildLookup()
	return x
}

type relinker struct {
	moves         []Move
	before, after *LinkIndex
}

// fileLinkRe matches any Markdown link target; fileLink filters it.
var fileLinkRe = regexp.MustCompile(`\]\(<?([^)>]+?)>?(#[^)]*)?\)`)

type edit struct {
	start, end int
	text       string
}

// note rewrites one note's links, skipping code like parseLinks, and
// returns the new text and how many links changed.
func (r relinker) note(text, oldID, newID string) (string, int) {
	lines := strings.Split(text, "\n")
	count := 0
	fence := ""
	for i, line := range lines {
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
		code := stripInlineCode(line)
		var edits []edit
		add := func(start, end int, s string, ok bool) {
			if ok {
				edits = append(edits, edit{start, end, s})
			}
		}
		for _, m := range wikiLinkRe.FindAllStringSubmatchIndex(code, -1) {
			raw, _, _ := strings.Cut(code[m[4]:m[5]], "|")
			raw, _, _ = strings.Cut(raw, "#")
			target := strings.TrimSpace(raw)
			if target == "" {
				continue
			}
			start := m[4] + strings.Index(raw, target)
			s, ok := r.wiki(target, oldID, newID)
			add(start, start+len(target), s, ok)
		}
		noteLink := map[int]bool{}
		for _, m := range mdLinkRe.FindAllStringSubmatchIndex(code, -1) {
			noteLink[m[0]] = true
			s, ok := r.markdown(code[m[2]:m[3]], oldID, newID, true)
			add(m[2], m[3], s, ok)
		}
		for _, m := range fileLinkRe.FindAllStringSubmatchIndex(code, -1) {
			if !noteLink[m[0]] {
				s, ok := r.markdown(code[m[2]:m[3]], oldID, newID, false)
				add(m[2], m[3], s, ok)
			}
		}
		if len(edits) == 0 {
			continue
		}
		sort.Slice(edits, func(a, b int) bool { return edits[a].start < edits[b].start })
		var b strings.Builder
		last := 0
		for _, e := range edits {
			b.WriteString(line[last:e.start])
			b.WriteString(e.text)
			last = e.end
		}
		b.WriteString(line[last:])
		lines[i] = b.String()
		count += len(edits)
	}
	return strings.Join(lines, "\n"), count
}

// wiki returns the new target of a wiki link, or false to keep it.
func (r relinker) wiki(target, oldID, newID string) (string, bool) {
	was, ok := r.before.resolveWiki(target, oldID)
	if !ok {
		return "", false
	}
	want := movePath(r.moves, was)
	if now, ok := r.after.resolveWiki(target, newID); ok && now == want {
		return "", false
	}
	ext := ""
	if strings.HasSuffix(strings.ToLower(target), mdExt) {
		ext = target[len(target)-len(mdExt):]
	}
	full := strings.TrimSuffix(want, mdExt)
	for _, c := range []string{path.Base(full), full} {
		if now, ok := r.after.resolveWiki(c, newID); ok && now == want {
			return c + ext, true
		}
	}
	return "", false
}

// markdown returns the new target of a Markdown link to a note (isNote) or
// to any other file, relative to the linking note, or false to keep it.
func (r relinker) markdown(raw, oldID, newID string, isNote bool) (string, bool) {
	if strings.Contains(raw, "://") || !isNote && (strings.ContainsAny(raw, `:"`) || strings.HasPrefix(raw, "/")) {
		return "", false // URLs, "mailto:", titles and absolute paths
	}
	t := raw
	if u, err := url.PathUnescape(t); err == nil {
		t = u
	}
	was := path.Clean(path.Join(path.Dir(oldID), t))
	if isNote {
		if _, ok := r.before.byID[was]; !ok {
			return "", false
		}
	} else if strings.HasSuffix(was, mdExt) || was == ".." || strings.HasPrefix(was, "../") {
		return "", false
	}
	want := movePath(r.moves, was)
	if path.Clean(path.Join(path.Dir(newID), t)) == want {
		return "", false
	}
	rel := relPath(path.Dir(newID), want)
	if raw != t {
		parts := strings.Split(rel, "/")
		for i, s := range parts {
			parts[i] = url.PathEscape(s)
		}
		rel = strings.Join(parts, "/")
	}
	return rel, true
}

// relPath is the slash path to target from folder dir, both from the root.
func relPath(dir, target string) string {
	if dir == "." {
		return target
	}
	d, t := strings.Split(dir, "/"), strings.Split(target, "/")
	i := 0
	for i < len(d) && i < len(t)-1 && d[i] == t[i] {
		i++
	}
	return strings.Repeat("../", len(d)-i) + strings.Join(t[i:], "/")
}
