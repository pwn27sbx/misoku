package core

import (
	"path"
	"strings"
	"time"
)

// LinkIndex holds the links between every note of a workspace, so the note
// graph and backlinks don't re-read and re-resolve the whole workspace on
// each query. It gives the same answers as LinkGraph and Backlinks, plus
// the links that name no note yet.
//
// Each note's links are parsed once and kept until its modification time
// changes; resolving them goes through maps instead of scanning every note.
// A LinkIndex is not safe for concurrent use.
type LinkIndex struct {
	isAttachment func(target string) bool

	notes    []Page
	byID     map[string]int
	byKey    map[string][]int // noteKey(ID) → positions in notes, in order
	bySuffix map[string][]int // each part of noteKey(ID) after a "/" → positions
	entries  map[string]*indexEntry

	edges      []Edge
	backlinks  map[string][]Backlink
	unresolved []UnresolvedLink
}

// indexEntry is what the index keeps of one note.
type indexEntry struct {
	stamp      time.Time // modification time the links were parsed at
	refs       []linkRef
	links      []noteLink // refs that resolve, in order
	unresolved []UnresolvedLink
	stale      bool // refs changed since they were last resolved
}

// UnresolvedLink is a wiki link to a note that doesn't exist yet.
type UnresolvedLink struct {
	From   string // ID of the linking note
	Target string // the target as written, such as "Projects/New idea"
	Key    string // Target normalised, the same for every spelling of one note
	Line   int    // 1-based line of the first such link in From
	Text   string // that line, trimmed
}

// NewLinkIndex returns an empty index. isAttachment, when not nil, tells
// which wiki-link targets name files that are not notes ("pic.png"); those
// are left out of Unresolved.
func NewLinkIndex(isAttachment func(target string) bool) *LinkIndex {
	return &LinkIndex{isAttachment: isAttachment, entries: map[string]*indexEntry{}}
}

// Update brings the index up to date with notes, as listed by Store.List
// (folders are skipped). read returns a note's text; it is called only for
// notes that are new or whose UpdatedAt changed, and always for notes with
// no UpdatedAt. A note that fails to read keeps its node, has no links, and
// is read again on the next Update.
//
// When the set of notes changes, every note's links are resolved again,
// because a new or renamed note can capture a link in a note that did not
// change.
func (x *LinkIndex) Update(notes []Page, read func(id string) (string, error)) {
	var list []Page
	for _, p := range notes {
		if !p.IsFolder {
			list = append(list, p)
		}
	}
	setChanged := !sameNoteIDs(x.notes, list)
	x.notes = list
	if setChanged {
		x.buildLookup()
	}

	entries := make(map[string]*indexEntry, len(list))
	for _, p := range list {
		e := x.entries[p.ID]
		if e == nil || p.UpdatedAt.IsZero() || !e.stamp.Equal(p.UpdatedAt) {
			content, err := read(p.ID)
			if err != nil {
				continue
			}
			e = newIndexEntry(p.ID, content, p.UpdatedAt)
		}
		entries[p.ID] = e
	}
	x.entries = entries
	for id, e := range entries {
		if setChanged || e.stale {
			x.resolve(id, e)
		}
	}
	x.rebuild()
}

// SetContent replaces the text of note id, such as the unsaved text of an
// open note, and resolves only that note's links. The text stands until a
// later Update sees a new modification time for the note. Unknown IDs are
// ignored.
func (x *LinkIndex) SetContent(id, content string) {
	i, ok := x.byID[id]
	if !ok {
		return
	}
	e := newIndexEntry(id, content, x.notes[i].UpdatedAt)
	x.entries[id] = e
	x.resolve(id, e)
	x.rebuild()
}

// Clone returns a copy of the index that can be updated without changing
// x, and the other way round, so one goroutine can update the copy while
// another still reads x. Neither may be used by two goroutines at once.
//
// The copy shares what Update and SetContent only ever replace, never
// change in place (the note list, lookups, edges, parsed links); each
// note's entry is copied, since resolving it again rewrites it.
func (x *LinkIndex) Clone() *LinkIndex {
	c := *x
	c.entries = make(map[string]*indexEntry, len(x.entries))
	for id, e := range x.entries {
		copied := *e
		c.entries[id] = &copied
	}
	return &c
}

// Notes lists the indexed notes in the order Update received them. The
// slice must not be modified.
func (x *LinkIndex) Notes() []Page { return x.notes }

// Edges lists the links between notes like LinkGraph: each pair once per
// direction, without self-links. The slice must not be modified.
func (x *LinkIndex) Edges() []Edge { return x.edges }

// Backlinks lists the notes linking to id like Store.Backlinks, sorted by
// title.
func (x *LinkIndex) Backlinks(id string) []Backlink {
	in := x.backlinks[id]
	if len(in) == 0 {
		return nil
	}
	out := append([]Backlink(nil), in...)
	sortBacklinks(out)
	return out
}

// Unresolved lists the wiki links to notes that don't exist, once per
// linking note and target, in note order. The slice must not be modified.
func (x *LinkIndex) Unresolved() []UnresolvedLink { return x.unresolved }

func newIndexEntry(id, content string, stamp time.Time) *indexEntry {
	refs := parseLinks(Page{ID: id, Content: content})
	for i := range refs {
		// Copies, so the cache doesn't keep every note's whole text alive.
		refs[i].target = strings.Clone(refs[i].target)
		refs[i].text = strings.Clone(refs[i].text)
	}
	return &indexEntry{stamp: stamp, refs: refs, stale: true}
}

func sameNoteIDs(a, b []Page) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			return false
		}
	}
	return true
}

// buildLookup indexes the notes by every key ResolveNote matches a target
// against: the whole path and each path suffix after a "/".
func (x *LinkIndex) buildLookup() {
	x.byID = make(map[string]int, len(x.notes))
	x.byKey = make(map[string][]int, len(x.notes))
	x.bySuffix = make(map[string][]int, 2*len(x.notes))
	for i, p := range x.notes {
		x.byID[p.ID] = i
		k := noteKey(p.ID)
		x.byKey[k] = append(x.byKey[k], i)
		for j := 0; j < len(k); j++ {
			if k[j] == '/' {
				x.bySuffix[k[j+1:]] = append(x.bySuffix[k[j+1:]], i)
			}
		}
	}
}

// resolve resolves e's links as seen from note id.
func (x *LinkIndex) resolve(id string, e *indexEntry) {
	e.links, e.unresolved, e.stale = nil, nil, false
	dangling := map[string]bool{}
	for _, r := range e.refs {
		if r.md {
			if _, ok := x.byID[r.target]; ok {
				e.links = append(e.links, noteLink{r.target, r.line, r.text})
			}
			continue
		}
		if to, ok := x.resolveWiki(r.target, id); ok {
			e.links = append(e.links, noteLink{to, r.line, r.text})
			continue
		}
		key := linkKey(r.target)
		if key == "" || dangling[key] || x.isAttachment != nil && x.isAttachment(r.target) {
			continue
		}
		dangling[key] = true
		e.unresolved = append(e.unresolved, UnresolvedLink{From: id, Target: r.target, Key: key, Line: r.line, Text: r.text})
	}
}

// resolveWiki gives the same answer as ResolveNote over the indexed notes:
// the best score wins (3 relative to fromID's folder, 2 from the root, 1 a
// path suffix), then the note closer to fromID, then the first listed.
func (x *LinkIndex) resolveWiki(target, fromID string) (string, bool) {
	t := linkKey(target)
	if t == "" {
		return "", false
	}
	fromDir := strings.ToLower(path.Dir(fromID))
	best, bestScore := -1, -1
	consider := func(candidates []int, score int) {
		// Candidates come in note order and closer is strict, so the first
		// listed of equally close notes stays, as in ResolveNote.
		for _, i := range candidates {
			if score > bestScore || score == bestScore && closer(x.notes[i].ID, x.notes[best].ID, fromDir) {
				best, bestScore = i, score
			}
		}
	}
	consider(x.byKey[path.Join(fromDir, t)], 3)
	consider(x.byKey[t], 2)
	consider(x.bySuffix[t], 1)
	if best < 0 {
		return "", false
	}
	return x.notes[best].ID, true
}

// rebuild derives the edges, backlinks and unresolved links from every
// note's resolved links, in note order.
func (x *LinkIndex) rebuild() {
	// Sized from the last build: one changed note rarely changes the totals.
	seen := make(map[Edge]bool, len(x.edges))
	x.edges = make([]Edge, 0, len(x.edges))
	x.backlinks = make(map[string][]Backlink, len(x.backlinks))
	x.unresolved = nil
	for _, p := range x.notes {
		e := x.entries[p.ID]
		if e == nil {
			continue
		}
		for _, l := range e.links {
			edge := Edge{From: p.ID, To: l.to}
			if l.to == p.ID || seen[edge] {
				continue
			}
			seen[edge] = true
			x.edges = append(x.edges, edge)
			// The first link to a note is the one its backlink shows.
			x.backlinks[l.to] = append(x.backlinks[l.to], Backlink{ID: p.ID, Title: p.Title, Line: l.line, Text: l.text})
		}
		x.unresolved = append(x.unresolved, e.unresolved...)
	}
}
