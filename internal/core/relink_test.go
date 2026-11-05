package core

import (
	"reflect"
	"testing"
)

// note is one fixture note: its ID and text, in list order.
type note struct{ id, text string }

// relinkRun plans the rewrite for notes after moves, applies it, checks
// that every link that resolved before still resolves to its note mapped
// through the moves, and returns the rewritten text of each changed note
// by its new ID.
func relinkRun(t *testing.T, notes []note, moves ...Move) map[string]string {
	t.Helper()
	var before []Page
	content := map[string]string{}
	for _, n := range notes {
		before = append(before, Page{ID: n.id})
		content[n.id] = n.text
	}
	got := map[string]string{}
	for _, r := range PlanRelinks(before, moves, content) {
		if r.Links == 0 || r.From == r.To {
			t.Errorf("%s: empty relink %+v", r.ID, r)
		}
		got[r.ID] = r.To
	}

	// Invariant: link i of every note names the same note it did before.
	var after []Page
	for _, p := range before {
		after = append(after, Page{ID: movePath(moves, p.ID)})
	}
	for _, n := range notes {
		newID := movePath(moves, n.id)
		text, ok := got[newID]
		if !ok {
			text = n.text
		}
		was := parseLinks(Page{ID: n.id, Content: n.text})
		now := parseLinks(Page{ID: newID, Content: text})
		if len(was) != len(now) {
			t.Fatalf("%s: %d links became %d", n.id, len(was), len(now))
		}
		for i := range was {
			from, ok := resolveRef(before, was[i], n.id)
			if !ok {
				continue
			}
			if to, _ := resolveRef(after, now[i], newID); to != movePath(moves, from) {
				t.Errorf("%s: link %d resolved to %q, now to %q", n.id, i, from, to)
			}
		}
	}
	return got
}

// resolveRef resolves one parsed link the way noteLinks does.
func resolveRef(notes []Page, r linkRef, fromID string) (string, bool) {
	if !r.md {
		p, ok := ResolveNote(notes, r.target, fromID)
		return p.ID, ok
	}
	for _, p := range notes {
		if p.ID == r.target {
			return p.ID, true
		}
	}
	return "", false
}

func TestRelinkRenameKeepsEveryLinkForm(t *testing.T) {
	got := relinkRun(t, []note{
		{"old.md", "x"},
		{"old x.md", "y"},
		{"a.md", "[[old]] [[old|Alias]] [[old#Head]] [[old#^blk]] ![[old]] [[ old.md | A ]]\n" +
			"[t](old.md) [t](old.md#head) [t](<old x.md>) [t](old%20x.md)"},
	}, Move{"old.md", "new.md"}, Move{"old x.md", "new x.md"})
	want := "[[new]] [[new|Alias]] [[new#Head]] [[new#^blk]] ![[new]] [[ new.md | A ]]\n" +
		"[t](new.md) [t](new.md#head) [t](<new x.md>) [t](new%20x.md)"
	if got["a.md"] != want {
		t.Errorf("got  %q\nwant %q", got["a.md"], want)
	}
}

func TestRelinkSkipsCode(t *testing.T) {
	text := "`[[old]]` `[t](old.md)`\n```\n[[old]] [t](old.md)\n```\n[[old]]"
	got := relinkRun(t, []note{{"old.md", ""}, {"a.md", text}}, Move{"old.md", "new.md"})
	if want := "`[[old]]` `[t](old.md)`\n```\n[[old]] [t](old.md)\n```\n[[new]]"; got["a.md"] != want {
		t.Errorf("got %q, want %q", got["a.md"], want)
	}
}

func TestRelinkPathQualifiedAndCrossFolder(t *testing.T) {
	got := relinkRun(t, []note{
		{"Projects/old.md", ""},
		{"Other/a.md", "[[Projects/old]] [t](../Projects/old.md)"},
	}, Move{"Projects/old.md", "Projects/new.md"})
	if want := "[[new]] [t](../Projects/new.md)"; got["Other/a.md"] != want {
		t.Errorf("got %q, want %q", got["Other/a.md"], want)
	}
}

func TestRelinkSidecarAndFolder(t *testing.T) {
	got := relinkRun(t, []note{
		{"p.md", ""},
		{"p/c.md", ""},
		{"A/n.md", ""},
		{"r.md", "[[p/c]] [x](p/c.md) [[A/n]] [y](A/n.md)"},
	}, Move{"p.md", "q.md"}, Move{"A", "B"})
	if want := "[[c]] [x](q/c.md) [[n]] [y](B/n.md)"; got["r.md"] != want {
		t.Errorf("got %q, want %q", got["r.md"], want)
	}
}

func TestRelinkNeverCapturesSameName(t *testing.T) {
	got := relinkRun(t, []note{
		{"old.md", ""},
		{"X/new.md", ""},
		{"l.md", "[[new]] [[old]]"},
	}, Move{"old.md", "new.md"})
	if want := "[[X/new]] [[new]]"; got["l.md"] != want {
		t.Errorf("got %q, want %q", got["l.md"], want)
	}
}

func TestRelinkAmbiguousNameGetsRootPath(t *testing.T) {
	got := relinkRun(t, []note{
		{"B/new.md", ""},
		{"A/old.md", ""},
		{"l.md", "[[old]]"},
	}, Move{"A/old.md", "A/new.md"})
	if want := "[[A/new]]"; got["l.md"] != want {
		t.Errorf("got %q, want %q", got["l.md"], want)
	}
}

func TestRelinkMovedNoteOwnLinks(t *testing.T) {
	got := relinkRun(t, []note{
		{"b.md", ""},
		{"X/b.md", ""},
		{"X/a.md", "[[b]] [t](b.md) ![](assets/pic.png) [f](file.pdf) [w](https://x.io/p.png) ![[b]]"},
	}, Move{"X/a.md", "Y/a.md"})
	want := "[[X/b]] [t](../X/b.md) ![](../X/assets/pic.png) [f](../X/file.pdf) [w](https://x.io/p.png) ![[X/b]]"
	if got["Y/a.md"] != want {
		t.Errorf("got  %q\nwant %q", got["Y/a.md"], want)
	}
}

func TestRelinkFilesMoveWithTheirFolder(t *testing.T) {
	got := relinkRun(t, []note{
		{"X/a.md", "![](assets/pic.png) [[b]]"},
		{"X/b.md", ""},
		{"r.md", "![](X/assets/pic.png)"},
	}, Move{"X", "Z"})
	if _, ok := got["Z/a.md"]; ok {
		t.Errorf("a moved with its files and links, got rewrite %q", got["Z/a.md"])
	}
	if want := "![](Z/assets/pic.png)"; got["r.md"] != want {
		t.Errorf("got %q, want %q", got["r.md"], want)
	}
}

func TestRelinkLeavesUnresolved(t *testing.T) {
	text := "[[nothing]] [t](missing.md) [[ghost]]"
	got := relinkRun(t, []note{{"old.md", ""}, {"a.md", text}}, Move{"old.md", "ghost.md"})
	if len(got) != 0 {
		t.Errorf("unexpected rewrites %v", got)
	}
}

func TestExpandMoves(t *testing.T) {
	before := []Page{
		{ID: "p.md"}, {ID: "p/c.md"}, {ID: "p/d/e.md"}, {ID: "pp.md"},
		{ID: "A", IsFolder: true}, {ID: "A/n.md"}, {ID: "Ab.md"},
	}
	got := ExpandMoves(before, []Move{{"p.md", "q.md"}, {"A", "B"}})
	want := []Move{{"p.md", "q.md"}, {"p/c.md", "q/c.md"}, {"p/d/e.md", "q/d/e.md"}, {"A/n.md", "B/n.md"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExpandMoves = %v, want %v", got, want)
	}
}
