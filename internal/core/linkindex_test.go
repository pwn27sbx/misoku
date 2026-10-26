package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// stamp gives each fixture note a distinct, non-zero modification time.
func stamp(n int) time.Time { return time.Unix(1_700_000_000+int64(n), 0) }

// memNotes turns ID → content into pages in the given order, plus a reader.
func memNotes(ids []string, content map[string]string) ([]Page, func(string) (string, error)) {
	var notes []Page
	for i, id := range ids {
		notes = append(notes, Page{ID: id, Title: strings.TrimSuffix(filepath.Base(id), mdExt), UpdatedAt: stamp(i)})
	}
	read := func(id string) (string, error) {
		c, ok := content[id]
		if !ok {
			return "", errors.New("missing")
		}
		return c, nil
	}
	return notes, read
}

// equivalenceText links to every case the resolver distinguishes: relative,
// root and suffix paths, ties, case, ".md", backslashes, empty targets,
// sections, embeds, Markdown links and code.
const equivalenceText = `[[aws]] [[AWS]] [[work/AWS.md]] [[docker]] [[Docker]]
[[b/deep note]] [[nope]] [[folder]] [[c/note]] [[note]] [[note2]] [[Note2.md]]
[[case/a]] [[../AWS]] [[plan]] [[sub/plan]] [[Projects/Plan]] [[work\Docker]]
[[  AWS  ]] [[/AWS/]] [[#Heading]] [[#^blk]] [[AWS#Section]] [[AWS#^blk]] [[AWS|alias]]
[[With Space]] [[AWS.MD]] [[aws.md]] ![[docker]] ![[pic.png]] [[Note]]
[m](AWS.md) [m](work/Docker.md#frag) [m](../AWS.md) [m](With%20Space.md)
[m](<With Space.md>) [m](aws.md) [m](https://x.com/a.md) [m](../../outside.md)
[m](Plan.md) [m](sub/Plan.md) ` + "`[[AWS]]` `[m](AWS.md)`" + `
` + "```" + `
[[docker]] [m](AWS.md)
` + "```" + `
last [[docker]]`

var equivalenceIDs = []string{
	"AWS.md", "work/AWS.md", "work/Docker.md", "a/b/Deep Note.md",
	"a/c/Note.md", "b/c/Note.md", "x/y/Note2.md", "z/Note2.md",
	"Case/a.md", "Case/A.md", "Projects/Plan.md", "Projects/sub/Plan.md",
	"With Space.md", "q.md", "b/c/k.md",
}

func TestLinkIndexResolvesLikeNoteLinks(t *testing.T) {
	content := map[string]string{}
	for _, id := range equivalenceIDs {
		content[id] = equivalenceText
	}
	notes, read := memNotes(equivalenceIDs, content)
	x := NewLinkIndex(nil)
	// Folders are not notes: the index ignores them, as LinkGraph does.
	x.Update(append([]Page{{ID: "folder", IsFolder: true}}, notes...), read)

	for _, p := range notes {
		p.Content = content[p.ID]
		want := noteLinks(notes, p)
		if got := x.entries[p.ID].links; !reflect.DeepEqual(got, want) {
			t.Errorf("links from %s:\n got  %v\n want %v", p.ID, got, want)
		}
	}
}

func TestLinkIndexTieBreaks(t *testing.T) {
	ids := []string{"a/c/Note.md", "b/c/Note.md", "x/y/Note2.md", "z/Note2.md", "Case/a.md", "Case/A.md", "b/c/k.md", "q.md"}
	content := map[string]string{"b/c/k.md": "[[c/note]]", "q.md": "[[note2]] [[case/a]]"}
	notes, read := memNotes(ids, content)
	x := NewLinkIndex(nil)
	x.Update(notes, read)
	want := []Edge{
		{"b/c/k.md", "b/c/Note.md"}, // equal score and length: the linking note's folder wins
		{"q.md", "z/Note2.md"},      // equal score: the shorter path wins
		{"q.md", "Case/a.md"},       // names differing only by case: the first listed wins
	}
	if got := x.Edges(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Edges = %v, want %v", got, want)
	}
}

func TestLinkIndexMatchesStore(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"A.md":        "to [[B]] and [[B#Top]]\nagain [[B]] and [[A]] and [[sub/C]]\n[c](sub/C.md)",
		"B.md":        "back to [[A]]\n![[A]]",
		"sub/C.md":    "[[A]] [[missing]]\n[up](../B.md)",
		"sub/D.md":    "nothing here",
		"Other/C.md":  "[[C]]",
		"Other/E.md":  "```\n[[A]]\n```\n`[[B]]`",
		"zeta.md":     "[[a]]",
		"Upper/Z.md":  "[[zeta]] [[Zeta.md]]",
		"Upper/ZZ.md": "[[../A]]",
	}
	for id, c := range files {
		abs := filepath.Join(s.Root(), filepath.FromSlash(id))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	x := NewLinkIndex(nil)
	x.Update(s.List(), storeReader(s))

	notes, edges := s.LinkGraph()
	if got := x.Notes(); !reflect.DeepEqual(got, notes) {
		t.Errorf("Notes = %v, want %v", got, notes)
	}
	if got := x.Edges(); !reflect.DeepEqual(got, edges) {
		t.Errorf("Edges = %v, want %v", got, edges)
	}
	for _, p := range notes {
		if got, want := x.Backlinks(p.ID), s.Backlinks(p.ID); !reflect.DeepEqual(got, want) {
			t.Errorf("Backlinks(%s) = %v, want %v", p.ID, got, want)
		}
	}
}

func storeReader(s *Store) func(string) (string, error) {
	return func(id string) (string, error) {
		p, err := s.Get(id)
		return p.Content, err
	}
}

func TestLinkIndexUnresolved(t *testing.T) {
	ids := []string{"A.md", "dir/B.md"}
	content := map[string]string{
		"A.md":     "[[New idea]] [[new IDEA]] [[#Heading]] ![[pic.png]]\n[[dir/B]] [[Later|alias]] `[[Code]]`\n[x](gone.md)",
		"dir/B.md": "see [[New idea#Part]]\n[[A]]",
	}
	notes, read := memNotes(ids, content)
	x := NewLinkIndex(func(target string) bool { return strings.HasSuffix(target, ".png") })
	x.Update(notes, read)
	want := []UnresolvedLink{
		{From: "A.md", Target: "New idea", Key: "new idea", Line: 1, Text: "[[New idea]] [[new IDEA]] [[#Heading]] ![[pic.png]]"},
		{From: "A.md", Target: "Later", Key: "later", Line: 2, Text: "[[dir/B]] [[Later|alias]] `[[Code]]`"},
		{From: "dir/B.md", Target: "New idea", Key: "new idea", Line: 1, Text: "see [[New idea#Part]]"},
	}
	if got := x.Unresolved(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Unresolved =\n %v\nwant\n %v", got, want)
	}

	// Without an attachment test, an embedded file that is not a note is
	// as unresolved as any other target.
	x = NewLinkIndex(nil)
	x.Update(notes, read)
	var targets []string
	for _, u := range x.Unresolved() {
		targets = append(targets, u.Target)
	}
	if want := []string{"New idea", "pic.png", "Later", "New idea"}; !reflect.DeepEqual(targets, want) {
		t.Fatalf("Unresolved targets = %v, want %v", targets, want)
	}
}

func TestLinkIndexEdgesDedupAndSkipSelf(t *testing.T) {
	ids := []string{"A.md", "B.md"}
	content := map[string]string{
		"A.md": "[[B]] [[B#x]] ![[B]]\n[b](B.md) [[A]] [[#^self]] [a](A.md)",
		"B.md": "[[A]] [[A]]\n[[B]]",
	}
	notes, read := memNotes(ids, content)
	x := NewLinkIndex(nil)
	x.Update(notes, read)
	want := []Edge{{"A.md", "B.md"}, {"B.md", "A.md"}}
	if got := x.Edges(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Edges = %v, want %v", got, want)
	}
	if got := x.Backlinks("A.md"); len(got) != 1 || got[0].ID != "B.md" || got[0].Line != 1 {
		t.Fatalf("Backlinks(A) = %v, want one from B on line 1", got)
	}
}

// indexSnapshot is everything a LinkIndex answers, for comparing two of them.
type indexSnapshot struct {
	Notes      []Page
	Edges      []Edge
	Unresolved []UnresolvedLink
	Backlinks  map[string][]Backlink
}

func snapshot(x *LinkIndex) indexSnapshot {
	s := indexSnapshot{Notes: x.Notes(), Edges: x.Edges(), Unresolved: x.Unresolved(), Backlinks: map[string][]Backlink{}}
	for _, p := range s.Notes {
		s.Backlinks[p.ID] = x.Backlinks(p.ID)
	}
	return s
}

func TestLinkIndexIncrementalUpdateEqualsRebuild(t *testing.T) {
	content := map[string]string{
		"AWS.md":           "[[Docker]] [[B]]",
		"work/x.md":        "[[AWS]] [[Docker]] [[B]]",
		"work/Old.md":      "[[x]]",
		"Docker.md":        "[[work/x]]",
		"deep/k/n.md":      "[[note]]",
		"far/away/note.md": "",
	}
	ids := []string{"AWS.md", "Docker.md", "deep/k/n.md", "far/away/note.md", "work/Old.md", "work/x.md"}
	steps := []struct {
		name string
		ids  []string
		edit map[string]string // new content; bumps the note's stamp
	}{
		{name: "start", ids: ids},
		{
			// work/AWS.md wins work/x.md's [[AWS]] (same folder) and B.md
			// resolves the dangling [[B]] of untouched notes.
			name: "add notes",
			ids:  []string{"AWS.md", "B.md", "Docker.md", "deep/k/n.md", "far/away/note.md", "work/AWS.md", "work/Old.md", "work/x.md"},
			edit: map[string]string{"B.md": "[[AWS]]", "work/AWS.md": ""},
		},
		{
			// Renaming Docker.md moves AWS.md's [[Docker]] to the new name;
			// deep/note.md becomes the shorter match for deep/k/n.md's [[note]].
			name: "rename",
			ids:  []string{"AWS.md", "B.md", "deep/k/n.md", "deep/note.md", "far/away/note.md", "work/AWS.md", "work/Docker.md", "work/Old.md", "work/x.md"},
			edit: map[string]string{"work/Docker.md": "[[work/x]]", "deep/note.md": ""},
		},
		{
			name: "delete and edit",
			ids:  []string{"AWS.md", "deep/k/n.md", "deep/note.md", "far/away/note.md", "work/AWS.md", "work/Docker.md", "work/Old.md", "work/x.md"},
			edit: map[string]string{"work/Old.md": "[[B]] [[AWS]]"},
		},
		{name: "unchanged", ids: []string{"AWS.md", "deep/k/n.md", "deep/note.md", "far/away/note.md", "work/AWS.md", "work/Docker.md", "work/Old.md", "work/x.md"}},
	}

	stamps := map[string]time.Time{}
	tick := 0
	inc := NewLinkIndex(nil)
	for _, step := range steps {
		for id, c := range step.edit {
			content[id] = c
			delete(stamps, id)
		}
		var notes []Page
		for _, id := range step.ids {
			if _, ok := stamps[id]; !ok {
				tick++
				stamps[id] = stamp(tick)
			}
			notes = append(notes, Page{ID: id, Title: strings.TrimSuffix(filepath.Base(id), mdExt), UpdatedAt: stamps[id]})
		}
		var reads []string
		read := func(id string) (string, error) {
			reads = append(reads, id)
			return content[id], nil
		}
		inc.Update(notes, read)

		wantReads := len(step.edit)
		if step.name == "start" {
			wantReads = len(step.ids)
		}
		if len(reads) != wantReads {
			t.Errorf("%s: read %v, want only the %d new or changed notes", step.name, reads, wantReads)
		}

		full := NewLinkIndex(nil)
		full.Update(notes, func(id string) (string, error) { return content[id], nil })
		if got, want := snapshot(inc), snapshot(full); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: incremental index differs from a rebuild:\n got  %+v\n want %+v", step.name, got, want)
		}
	}
	// The renamed and added notes must really have changed resolutions,
	// or the test above proves nothing about re-resolving untouched notes.
	if got := inc.Edges(); len(got) < 2 || !reflect.DeepEqual(got[:2], []Edge{{"AWS.md", "work/Docker.md"}, {"deep/k/n.md", "deep/note.md"}}) {
		t.Fatalf("final Edges = %v", got)
	}
}

func TestLinkIndexRereadsNotesWithoutStamp(t *testing.T) {
	notes := []Page{{ID: "A.md"}, {ID: "B.md"}}
	text := "[[B]]"
	read := func(string) (string, error) { return text, nil }
	x := NewLinkIndex(nil)
	x.Update(notes, read)
	text = ""
	x.Update(notes, read)
	if got := x.Edges(); len(got) != 0 {
		t.Fatalf("a note without a modification time must be re-read; Edges = %v", got)
	}
}

func TestLinkIndexReadErrorKeepsNote(t *testing.T) {
	notes, _ := memNotes([]string{"A.md", "B.md"}, nil)
	fail := true
	read := func(id string) (string, error) {
		if id == "A.md" && fail {
			return "", errors.New("boom")
		}
		if id == "A.md" {
			return "[[B]]", nil
		}
		return "[[A]]", nil
	}
	x := NewLinkIndex(nil)
	x.Update(notes, read)
	if got, want := x.Edges(), []Edge{{"B.md", "A.md"}}; len(x.Notes()) != 2 || !reflect.DeepEqual(got, want) {
		t.Fatalf("Notes = %v, Edges = %v; want both notes and %v", x.Notes(), got, want)
	}
	if got := x.Unread(); got != 1 {
		t.Fatalf("Unread = %d, want 1", got)
	}
	fail = false
	x.Update(notes, read)
	if got := x.Edges(); len(got) != 2 {
		t.Fatalf("a note that failed to read must be retried; Edges = %v", got)
	}
	if got := x.Unread(); got != 0 {
		t.Fatalf("Unread after the retry = %d, want 0", got)
	}
}

func TestLinkIndexSetContent(t *testing.T) {
	notes, read := memNotes([]string{"A.md", "B.md", "C.md"}, map[string]string{"A.md": "[[B]]", "B.md": "", "C.md": ""})
	x := NewLinkIndex(nil)
	x.Update(notes, read)
	x.SetContent("A.md", "[[C]] [[Nope]]")
	x.SetContent("missing.md", "[[A]]") // not a note: ignored
	if got, want := x.Edges(), []Edge{{"A.md", "C.md"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Edges = %v, want %v", got, want)
	}
	if got := x.Unresolved(); len(got) != 1 || got[0].Target != "Nope" {
		t.Fatalf("Unresolved = %v", got)
	}
	// The same stamp keeps the unsaved text instead of re-reading the file.
	x.Update(notes, func(string) (string, error) { t.Fatal("unchanged note re-read"); return "", nil })
	if got := x.Edges(); len(got) != 1 || got[0].To != "C.md" {
		t.Fatalf("Edges after Update = %v", got)
	}
}

// A clone is updated on its own: neither Update nor SetContent on one
// changes what the other answers.
func TestLinkIndexClone(t *testing.T) {
	// C links to a note only the clone gets: adding it resolves C's
	// link again in the clone, though C did not change.
	notes, read := memNotes([]string{"A.md", "B.md", "C.md"}, map[string]string{"A.md": "[[B]]", "B.md": "", "C.md": "[[D]]"})
	x := NewLinkIndex(nil)
	x.Update(notes, read)
	want := frozen(x)

	c := x.Clone()
	// A note added re-resolves every note; a changed one is re-read.
	more, readMore := memNotes([]string{"A.md", "B.md", "C.md", "D.md"}, map[string]string{"A.md": "[[C]]", "B.md": "[[D]]", "C.md": "[[D]]", "D.md": ""})
	more[0].UpdatedAt, more[1].UpdatedAt = stamp(100), stamp(101)
	c.Update(more, readMore)
	c.SetContent("C.md", "[[A]]")
	if got := frozen(x); !reflect.DeepEqual(got, want) {
		t.Fatalf("updating the clone changed the original:\n got %+v\nwant %+v", got, want)
	}
	if got, want := c.Edges(), []Edge{{"A.md", "C.md"}, {"B.md", "D.md"}, {"C.md", "A.md"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("clone Edges = %v, want %v", got, want)
	}

	cloned := frozen(c)
	x.SetContent("B.md", "[[C]]")
	if got := frozen(c); !reflect.DeepEqual(got, cloned) {
		t.Fatalf("updating the original changed the clone:\n got %+v\nwant %+v", got, cloned)
	}
	if got, want := x.Edges(), []Edge{{"A.md", "B.md"}, {"B.md", "C.md"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("original Edges = %v, want %v", got, want)
	}
}

// frozen is snapshot with its own copies of the index's slices, so a later
// change to the index can't change it too.
func frozen(x *LinkIndex) indexSnapshot {
	s := snapshot(x)
	s.Notes = append([]Page(nil), s.Notes...)
	s.Edges = append([]Edge(nil), s.Edges...)
	s.Unresolved = append([]UnresolvedLink(nil), s.Unresolved...)
	return s
}

func TestLinkIndexEmpty(t *testing.T) {
	x := NewLinkIndex(nil)
	x.Update(nil, nil)
	if len(x.Notes()) != 0 || len(x.Edges()) != 0 || len(x.Unresolved()) != 0 || x.Backlinks("A.md") != nil {
		t.Fatal("an empty workspace must have no notes, edges or links")
	}
}

// genVault builds n in-memory notes in n/100 folders, each with about ten
// wiki links (some dangling, some relative) and one Markdown link.
func genVault(n int) ([]Page, map[string]string) {
	notes := make([]Page, 0, n)
	content := make(map[string]string, n)
	folders := n/100 + 1
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("f%03d/note %d.md", i%folders, i)
		notes = append(notes, Page{ID: id, Title: fmt.Sprintf("note %d", i), UpdatedAt: stamp(i)})
		var b strings.Builder
		fmt.Fprintf(&b, "# Note %d\n\nSome text about things.\n", i)
		for k := 1; k <= 10; k++ {
			j := (i*7 + k*131) % (n + n/10) // about one in eleven is dangling
			fmt.Fprintf(&b, "- see [[note %d]] and more text\n", j)
		}
		fmt.Fprintf(&b, "[rel](note %d.md) [[f%03d/note %d#Top]]\n", i+folders, (i+1)%folders, i+1)
		content[id] = b.String()
	}
	return notes, content
}

func benchmarkLinkIndex(b *testing.B, n int) {
	notes, content := genVault(n)
	read := func(id string) (string, error) { return content[id], nil }
	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			NewLinkIndex(nil).Update(notes, read)
		}
	})
	b.Run("one-changed", func(b *testing.B) {
		x := NewLinkIndex(nil)
		x.Update(notes, read)
		changed := append([]Page(nil), notes...)
		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			changed[n/2].UpdatedAt = stamp(n + i)
			x.Update(changed, read)
		}
	})
	b.Run("set-content", func(b *testing.B) {
		x := NewLinkIndex(nil)
		x.Update(notes, read)
		b.ReportAllocs()
		for b.Loop() {
			x.SetContent(notes[n/2].ID, content[notes[n/3].ID])
		}
	})
}

func BenchmarkLinkIndex1k(b *testing.B) { benchmarkLinkIndex(b, 1000) }
func BenchmarkLinkIndex5k(b *testing.B) { benchmarkLinkIndex(b, 5000) }
