package dirplan_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/dirplan"
)

// vault is the list Store.List returns for:
//
//	a.md (with sub-notes a/sub.md), b.md, Docs/ (Docs/x.md, Docs/In/y.md)
var vault = []core.Page{
	{ID: "a.md", Title: "a"},
	{ID: "a/sub.md", Title: "sub", ParentID: "a.md"},
	{ID: "Docs", Title: "Docs", IsFolder: true},
	{ID: "Docs/In", Title: "In", ParentID: "Docs", IsFolder: true},
	{ID: "Docs/In/y.md", Title: "y", ParentID: "Docs/In"},
	{ID: "Docs/x.md", Title: "x", ParentID: "Docs"},
	{ID: "b.md", Title: "b"},
}

// root is the root folder as it stands, before any edit.
func root() []dirplan.Row {
	return []dirplan.Row{{OrigID: "a.md", Name: "a"}, {OrigID: "Docs", Name: "Docs/"}, {OrigID: "b.md", Name: "b"}}
}

func docs() []dirplan.Row {
	return []dirplan.Row{{OrigID: "Docs/In", Name: "In/"}, {OrigID: "Docs/x.md", Name: "x"}}
}

func show(p dirplan.Plan) string {
	var out []string
	for _, op := range p.Ops {
		s := op.Kind.String() + " " + op.From + ">" + op.To
		if op.Folder {
			s += "/"
		}
		out = append(out, s)
	}
	return strings.Join(out, "; ")
}

func with(rows []dirplan.Row, edit func([]dirplan.Row) []dirplan.Row) []dirplan.Row {
	return edit(rows)
}

func TestBuildPlans(t *testing.T) {
	cases := []struct {
		name    string
		folders []dirplan.Folder
		want    string
	}{
		{"nothing changed", []dirplan.Folder{{ID: "", Rows: root()}}, ""},
		{"new note", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "test"})}},
			"create >test.md"},
		{"new folder", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "test/"})}},
			"mkdir >test/"},
		{"new note in a new folder, same parser as n", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "ideas/test"})}},
			"mkdir >ideas/; create >ideas/test.md"},
		{"new note in an existing folder makes no folder", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "Docs/new"})}},
			"create >Docs/new.md"},
		{"invalid chars cleaned the way n cleans them", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: " what? "})}},
			"create >what-.md"},
		{"reserved name cleaned the way n cleans it", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "CON"})}},
			"create >CON-note.md"},
		{"new note under a note goes in its sub-note folder", []dirplan.Folder{{ID: "a.md", Rows: []dirplan.Row{{OrigID: "a/sub.md", Name: "sub"}, {Name: "two"}}}},
			"create >a/two.md"},
		{"rename a note with its sub-notes is one op", []dirplan.Folder{{ID: "", Rows: with(root(), func(r []dirplan.Row) []dirplan.Row { r[0].Name = "z"; return r })}},
			"rename a.md>z.md"},
		{"rename a folder", []dirplan.Folder{{ID: "", Rows: with(root(), func(r []dirplan.Row) []dirplan.Row { r[1].Name = "Papers/"; return r })}},
			"rename Docs>Papers/"},
		{"case-only rename is not a collision", []dirplan.Folder{{ID: "", Rows: with(root(), func(r []dirplan.Row) []dirplan.Row { r[2].Name = "B"; return r })}},
			"rename b.md>B.md"},
		{"deleted row is a delete", []dirplan.Folder{{ID: "", Rows: root()[:2]}},
			"delete b.md>"},
		{"cut in one folder and pasted in another is a move", []dirplan.Folder{
			{ID: "", Rows: root()[:2]},
			{ID: "Docs", Rows: append(docs(), dirplan.Row{OrigID: "b.md", Name: "b"})},
		}, "move b.md>Docs/b.md"},
		{"pasted while its folder is not open still moves", []dirplan.Folder{
			{ID: "Docs/In", Rows: []dirplan.Row{{OrigID: "Docs/In/y.md", Name: "y"}, {OrigID: "b.md", Name: "b"}}},
		}, "move b.md>Docs/In/b.md"},
		{"rename and move of the same row is one move", []dirplan.Folder{
			{ID: "", Rows: root()[:2]},
			{ID: "Docs", Rows: append(docs(), dirplan.Row{OrigID: "b.md", Name: "c"})},
		}, "move b.md>Docs/c.md"},
		{"a move inside a renamed folder follows it", []dirplan.Folder{
			{ID: "", Rows: with(root(), func(r []dirplan.Row) []dirplan.Row { r[1].Name = "Papers/"; return r })},
			{ID: "Docs", Rows: []dirplan.Row{{OrigID: "Docs/In", Name: "In/"}, {OrigID: "Docs/x.md", Name: "w"}}},
		}, "rename Docs>Papers/; rename Papers/x.md>Papers/w.md"},
		{"a delete inside a renamed folder follows it", []dirplan.Folder{
			{ID: "", Rows: with(root(), func(r []dirplan.Row) []dirplan.Row { r[1].Name = "Papers/"; return r })},
			{ID: "Docs", Rows: docs()[:1]},
		}, "rename Docs>Papers/; delete Papers/x.md>"},
		{"a deleted folder's contents get no delete of their own", []dirplan.Folder{
			{ID: "", Rows: []dirplan.Row{{OrigID: "a.md", Name: "a"}, {OrigID: "b.md", Name: "b"}}},
			{ID: "Docs", Rows: docs()[:1]},
		}, "delete Docs>/"},
		{"order: folders, moves, notes, deletes last", []dirplan.Folder{{ID: "", Rows: []dirplan.Row{
			{Name: "n1"}, {OrigID: "a.md", Name: "a2"}, {Name: "f/"}, {OrigID: "Docs", Name: "Docs/"},
		}}}, "mkdir >f/; rename a.md>a2.md; create >n1.md; delete b.md>"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := dirplan.Build(vault, c.folders)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := show(p); got != c.want {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

func TestBuildRefuses(t *testing.T) {
	cases := []struct {
		name    string
		folders []dirplan.Folder
		why     string
	}{
		{"two new lines with one name", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "n"}, dirplan.Row{Name: "N"})}}, "twice"},
		{"rename onto a sibling", []dirplan.Folder{{ID: "", Rows: with(root(), func(r []dirplan.Row) []dirplan.Row { r[0].Name = "b"; return r })}}, "already exists"},
		{"new note with an existing name", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "b"})}}, "already exists"},
		{"new folder over a note with sub-notes", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "a/"})}}, "already exists"},
		{"new note deeper with an existing name", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "Docs/x"})}}, "already exists"},
		{"pasted next to a namesake", []dirplan.Folder{
			{ID: "", Rows: root()[:2]},
			{ID: "Docs", Rows: append(docs(), dirplan.Row{OrigID: "b.md", Name: "x"})},
		}, "already exists"},
		{"cut row pasted twice", []dirplan.Folder{
			{ID: "", Rows: root()[:2]},
			{ID: "Docs", Rows: append(docs(), dirplan.Row{OrigID: "b.md", Name: "b"}, dirplan.Row{OrigID: "b.md", Name: "b2"})},
		}, "more than once"},
		{"folder into itself", []dirplan.Folder{
			{ID: "", Rows: []dirplan.Row{{OrigID: "a.md", Name: "a"}, {OrigID: "b.md", Name: "b"}}},
			{ID: "Docs/In", Rows: []dirplan.Row{{OrigID: "Docs/In/y.md", Name: "y"}, {OrigID: "Docs", Name: "Docs/"}}},
		}, "into itself"},
		{"note into its own sub-notes", []dirplan.Folder{
			{ID: "", Rows: root()[1:]},
			{ID: "a.md", Rows: []dirplan.Row{{OrigID: "a/sub.md", Name: "sub"}, {OrigID: "a.md", Name: "a"}}},
		}, "into itself"},
		{"parent path", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "../x"})}}, `".."`},
		{"hidden name", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: ".git/"})}}, "hidden"},
		{"absolute path", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{Name: "/etc/x"})}}, "relative"},
		{"empty rename", []dirplan.Folder{{ID: "", Rows: with(root(), func(r []dirplan.Row) []dirplan.Row { r[2].Name = " "; return r })}}, "empty"},
		{"rename into a path", []dirplan.Folder{{ID: "", Rows: with(root(), func(r []dirplan.Row) []dirplan.Row { r[2].Name = "Docs/b"; return r })}}, "not a path"},
		{"note renamed as a folder", []dirplan.Folder{{ID: "", Rows: with(root(), func(r []dirplan.Row) []dirplan.Row { r[2].Name = "b/"; return r })}}, "folder"},
		{"unknown entry", []dirplan.Folder{{ID: "", Rows: append(root(), dirplan.Row{OrigID: "gone.md", Name: "gone"})}}, "no longer exists"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := dirplan.Build(vault, c.folders)
			if err == nil || !strings.Contains(err.Error(), c.why) {
				t.Fatalf("want an error with %q, got %v (plan %q)", c.why, err, show(p))
			}
			if len(p.Ops) != 0 {
				t.Errorf("a refused plan has no ops, got %q", show(p))
			}
		})
	}
}
