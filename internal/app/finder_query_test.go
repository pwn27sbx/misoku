package app

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"

	"github.com/charmbracelet/bubbles/textinput"
)

// S2: the finder's query splits into qualifiers and the words left to match.
func TestParseFinderQuery(t *testing.T) {
	for _, c := range []struct {
		in   string
		want finderQuery
	}{
		{"docker", finderQuery{text: "docker"}},
		{"  two  words ", finderQuery{text: "two  words"}},
		{`"quoted words"`, finderQuery{text: `"quoted words"`}},
		{"status:active tag:go folder:Zentro docker", finderQuery{qualified: true,
			statuses: []string{"active"}, tags: []string{"go"}, folders: [][]string{{"Zentro"}}, text: "docker"}},
		{"docker status:on-hold more", finderQuery{qualified: true, statuses: []string{"on-hold"}, text: "docker more"}},
		{"#go/web notes", finderQuery{qualified: true, tags: []string{"go/web"}, text: "notes"}},
		{"TAG:Go tag:#Web/", finderQuery{qualified: true, tags: []string{"go", "web"}}},
		{`tag:"My Tag" x`, finderQuery{qualified: true, tags: []string{"my tag"}, text: "x"}},
		{`folder:"a b" x`, finderQuery{qualified: true, folders: [][]string{{"a b"}}, text: "x"}},
		{`folder:"/Zentro/sub/"`, finderQuery{qualified: true, folders: [][]string{{"Zentro", "sub"}}}},
		{"status:", finderQuery{qualified: true, statuses: []string{""}}},
		{"tag: folder:", finderQuery{qualified: true}},
		{"foo:bar baz", finderQuery{text: "foo:bar baz"}},
		{"# heading", finderQuery{text: "# heading"}},
		{"a#b ##x", finderQuery{text: "a#b ##x"}},
		{"", finderQuery{}},
	} {
		if got := parseFinderQuery(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseFinderQuery(%q) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

// testFinder is a finder over notes (ID → text), the first one the newest.
func testFinder(notes [][2]string) *finder {
	f := &finder{input: textinput.New(), lines: map[string][]string{}}
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for i, n := range notes {
		title := strings.TrimSuffix(n[0][strings.LastIndex(n[0], "/")+1:], ".md")
		f.notes = append(f.notes, core.Page{ID: n[0], Title: title, UpdatedAt: at.Add(-time.Duration(i) * time.Hour)})
		f.lines[n[0]] = strings.Split(n[1], "\n")
	}
	return f
}

// search types q into f and lists the notes found, once each, in order.
func (f *finder) search(q string) []string {
	f.input.SetValue(q)
	f.refilter()
	var ids []string
	seen := map[string]bool{}
	for _, mt := range f.matches {
		if !seen[mt.page.ID] {
			seen[mt.page.ID] = true
			ids = append(ids, mt.page.ID)
		}
	}
	return ids
}

// qualifiedNotes each fail one part of "status:active tag:go folder:Zentro
// docker", but Zentro/alpha.md.
var qualifiedNotes = [][2]string{
	{"Zentro/alpha.md", "---\nstatus: Active\n---\nRun docker with #go"},
	{"Zentro/bravo.md", "---\nstatus: onhold\n---\nRun docker with #go"},
	{"Zentro/charlie.md", "---\nstatus: active\ntags: [python]\n---\nRun docker"},
	{"ZentroOld/delta.md", "---\nstatus: active\n---\nRun docker with #go"},
	{"Zentro/echo.md", "---\nstatus: active\n---\nNothing here #go"},
	{"Other/docker.md", "---\nstatus: active\n---\ndocker #go"},
}

// S2: the qualifiers AND together, and the words left match the name and
// the text of the notes that pass them only.
func TestFinderQualifiersNarrowNameAndTextHits(t *testing.T) {
	f := testFinder(qualifiedNotes)
	if got := f.search("docker"); len(got) != 5 {
		t.Fatalf("without qualifiers docker is in five notes, got %v", got)
	}
	if got := f.search("status:active tag:go folder:Zentro docker"); !reflect.DeepEqual(got, []string{"Zentro/alpha.md"}) {
		t.Fatalf("want only Zentro/alpha.md, got %v", got)
	}
	for _, mt := range f.matches {
		if mt.line == 0 {
			t.Errorf("alpha.md's name has no docker in it, yet it is a name hit: %+v", mt)
		}
	}
}

// S2: folder: matches whole path segments, in any letter case, and quotes
// hold a name with spaces.
func TestFinderFolderQualifierMatchesSegments(t *testing.T) {
	f := testFinder([][2]string{
		{"Zentro/a.md", "a"},
		{"Zentro/sub/b.md", "b"},
		{"ZentroOld/c.md", "c"},
		{"My Notes/d.md", "d"},
		{"zentro.md", "e"},
	})
	for _, c := range []struct {
		q    string
		want []string
	}{
		{"folder:Zentro", []string{"Zentro/a.md", "Zentro/sub/b.md"}},
		{"folder:zENTRO", []string{"Zentro/a.md", "Zentro/sub/b.md"}},
		{"folder:sub", []string{"Zentro/sub/b.md"}},
		{"folder:Zentro/sub", []string{"Zentro/sub/b.md"}},
		{`folder:"my notes"`, []string{"My Notes/d.md"}},
		{"folder:Zen", nil},
	} {
		if got := f.search(c.q); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %v, want %v", c.q, got, c.want)
		}
	}
}

// S2/S4: tag:t and #t match t and its nested tags, from the text or the
// front matter, in any letter case; a tag that only starts the same doesn't.
func TestFinderTagQualifierMatchesNestedTags(t *testing.T) {
	f := testFinder([][2]string{
		{"a.md", "about #go/web"},
		{"b.md", "---\ntags: [Go, web]\n---\nb"},
		{"c.md", "about #gopher"},
		{"d.md", "`#go` in code only"},
	})
	for _, q := range []string{"tag:go", "#go", "tag:GO", "tag:#go"} {
		if got := f.search(q); !reflect.DeepEqual(got, []string{"a.md", "b.md"}) {
			t.Errorf("%s = %v, want a.md and b.md", q, got)
		}
	}
	if got := f.search("tag:go/web"); !reflect.DeepEqual(got, []string{"a.md"}) {
		t.Errorf("tag:go/web = %v, want a.md", got)
	}
}

// S2: only qualifiers list every note that passes them, newest first, and
// no text hits.
func TestFinderOnlyQualifiersListNewestFirst(t *testing.T) {
	f := testFinder(qualifiedNotes)
	// The store lists notes by path; their age is another order.
	age := map[string]int{"Zentro/echo.md": 0, "Zentro/charlie.md": 1, "ZentroOld/delta.md": 2, "Zentro/alpha.md": 3, "Zentro/bravo.md": 4, "Other/docker.md": 5}
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for i := range f.notes {
		f.notes[i].UpdatedAt = at.Add(-time.Duration(age[f.notes[i].ID]) * time.Hour)
	}
	sort.SliceStable(f.notes, func(i, j int) bool { return f.notes[i].ID < f.notes[j].ID })
	got := f.search("status:active")
	want := []string{"Zentro/echo.md", "Zentro/charlie.md", "ZentroOld/delta.md", "Zentro/alpha.md", "Other/docker.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("status:active = %v, want %v", got, want)
	}
	for _, mt := range f.matches {
		if mt.line != 0 {
			t.Fatalf("only qualifiers give no text hits: %+v", mt)
		}
	}
	if len(f.matches) != len(want) {
		t.Fatalf("one row per note, got %d", len(f.matches))
	}
}

// S1/S2: status: reads the front matter as :status does; an unknown value
// finds nothing, and status: alone finds everything while it asks for one.
func TestFinderStatusQualifier(t *testing.T) {
	f := testFinder([][2]string{
		{"a.md", "---\nstatus: on hold\n---\na"},
		{"b.md", "---\nstatus: completed\n---\nb"},
		{"c.md", "no status"},
	})
	for _, c := range []struct {
		q    string
		want []string
		hint bool
	}{
		{"status:onhold", []string{"a.md"}, false},
		{"status:On-Hold", []string{"a.md"}, false},
		{"status:completed", []string{"b.md"}, false},
		{"status:dropped", nil, false},
		{"status:nosuch", nil, true},
		{"status:", []string{"a.md", "b.md", "c.md"}, true},
		{"onhold", nil, false},
	} {
		if got := f.search(c.q); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %v, want %v", c.q, got, c.want)
		}
		if got := f.statusHint(); got != c.hint {
			t.Errorf("%s: status hint %v, want %v", c.q, got, c.hint)
		}
	}
}

// S2: an unknown qualifier is plain text.
func TestFinderUnknownQualifierIsText(t *testing.T) {
	f := testFinder([][2]string{{"a.md", "see foo:bar here"}, {"b.md", "nothing"}})
	if got := f.search("foo:bar"); !reflect.DeepEqual(got, []string{"a.md"}) {
		t.Fatalf("foo:bar = %v, want a.md by its text", got)
	}
}

// legacyRefilter is the finder's search before qualifiers, kept to pin that
// a query without them finds exactly what it found before.
func legacyRefilter(f *finder) []finderMatch {
	q := strings.TrimSpace(f.input.Value())
	var matches []finderMatch
	for _, p := range f.notes {
		if score, pos, ok := fuzzyScore(strings.ReplaceAll(q, " ", ""), p.ID); ok {
			matches = append(matches, finderMatch{page: p, score: score, pos: pos})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].page.UpdatedAt.After(matches[j].page.UpdatedAt)
	})
	if needle := strings.ToLower(q); len([]rune(needle)) >= 2 {
		byPath := append([]core.Page(nil), f.notes...)
		sort.Slice(byPath, func(i, j int) bool { return byPath[i].ID < byPath[j].ID })
		hits := 0
	scan:
		for _, p := range byPath {
			for n, l := range f.lines[p.ID] {
				lower := strings.ToLower(l)
				idx := strings.Index(lower, needle)
				if idx < 0 || len(lower) != len(l) {
					continue
				}
				trimmed := strings.TrimLeft(l, " ")
				shift := len(l) - len(trimmed)
				var pos []int
				for k := idx; k < idx+len(needle); k++ {
					pos = append(pos, k-shift)
				}
				matches = append(matches, finderMatch{page: p, pos: pos, line: n + 1, text: trimmed})
				if hits++; hits >= maxTextHits {
					break scan
				}
			}
		}
	}
	return matches
}

// S2: a query without qualifiers finds exactly what it found before.
func TestFinderPlainQueryUnchanged(t *testing.T) {
	f := testFinder(qualifiedNotes)
	for _, q := range []string{"", "d", "docker", "run docker", " Zentro ", "#", "# heading", "foo:bar", "a#b", `"run"`, "go"} {
		f.input.SetValue(q)
		want := legacyRefilter(f)
		f.refilter()
		if len(f.matches) != len(want) || (len(want) > 0 && !reflect.DeepEqual(f.matches, want)) {
			t.Errorf("%q: got %+v, want %+v", q, f.matches, want)
		}
	}
}

// S4: ":tags" before the index is built says so instead of an empty list.
func TestTagsBeforeTheIndexSaysIndexing(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New(store)
	m.openTagPicker()
	if m.status != "Indexing notes…" || m.menuBox != nil {
		t.Fatalf("status %q, menu %v: want Indexing notes… and no list", m.status, m.menuBox)
	}
}
