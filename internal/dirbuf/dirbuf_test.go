package dirbuf

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func keys(t *testing.T, b *Buffer, seq ...string) Nav {
	t.Helper()
	var nav Nav
	for _, k := range seq {
		var msg tea.KeyMsg
		switch k {
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "ctrl+r":
			msg = tea.KeyMsg{Type: tea.KeyCtrlR}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		nav, _ = b.Update(msg)
	}
	return nav
}

func typed(s string) []string {
	out := make([]string, 0, len(s))
	for _, r := range s {
		out = append(out, string(r))
	}
	return out
}

func orig() []Row {
	return []Row{{OrigID: "a.md", Name: "a"}, {OrigID: "b.md", Name: "b"}, {OrigID: "c", Name: "c/"}}
}

func names(rows []Row) []string {
	out := []string{}
	for _, r := range rows {
		out = append(out, r.Name)
	}
	return out
}

func TestRenameKeepsOrigID(t *testing.T) {
	b := New("", orig())
	keys(t, b, "j", "A")
	keys(t, b, typed("x")...)
	keys(t, b, "esc")
	want := []Row{{OrigID: "a.md", Name: "a"}, {OrigID: "b.md", Name: "bx"}, {OrigID: "c", Name: "c/"}}
	if got := b.Rows(); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	if !b.Dirty() {
		t.Fatal("Dirty() = false after a rename")
	}
}

func TestChangeWordReplacesName(t *testing.T) {
	b := New("", orig())
	keys(t, b, "c", "w")
	keys(t, b, typed("z")...)
	keys(t, b, "enter")
	if got := b.Rows()[0]; got != (Row{OrigID: "a.md", Name: "z"}) {
		t.Fatalf("row = %v", got)
	}
}

func TestInsertAtStart(t *testing.T) {
	b := New("", orig())
	keys(t, b, "I")
	keys(t, b, typed("x")...)
	keys(t, b, "esc")
	if got := b.Rows()[0].Name; got != "xa" {
		t.Fatalf("name = %q", got)
	}
}

func TestCutPasteInOtherFolderIsMove(t *testing.T) {
	b := New("", orig())
	keys(t, b, "d", "d")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"b", "c/"}) {
		t.Fatalf("after dd: %v", got)
	}
	b.Open("c", []Row{{OrigID: "c/x.md", Name: "x"}})
	keys(t, b, "p")
	want := []Row{{OrigID: "c/x.md", Name: "x"}, {OrigID: "a.md", Name: "a"}}
	if got := b.Rows(); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	if got := b.Edited(""); len(got) != 2 {
		t.Fatalf("root rows = %v", got)
	}
}

func TestYankPasteIsCopy(t *testing.T) {
	b := New("", orig())
	keys(t, b, "y", "y", "G", "p")
	want := []Row{{OrigID: "a.md", Name: "a"}, {OrigID: "b.md", Name: "b"}, {OrigID: "c", Name: "c/"},
		{OrigID: "a.md", Name: "a", Yanked: true}}
	if got := b.Rows(); !reflect.DeepEqual(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	keys(t, b, "g", "g", "P")
	if got := b.Rows()[0]; got != (Row{OrigID: "a.md", Name: "a", Yanked: true}) {
		t.Fatalf("P row = %v", got)
	}
}

func TestNewLineBelowAndAbove(t *testing.T) {
	b := New("", orig())
	keys(t, b, "o")
	if !b.Editing() {
		t.Fatal("o did not start editing")
	}
	keys(t, b, typed("ideas/t")...)
	keys(t, b, "esc", "G", "O")
	keys(t, b, typed("n/")...)
	keys(t, b, "esc")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "ideas/t", "b", "n/", "c/"}) {
		t.Fatalf("names = %v", got)
	}
	if b.Rows()[1].OrigID != "" || b.Rows()[3].OrigID != "" {
		t.Fatalf("new rows carry an OrigID: %v", b.Rows())
	}
}

func TestEmptyNewLineIsDropped(t *testing.T) {
	b := New("", orig())
	keys(t, b, "o", "esc")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "b", "c/"}) {
		t.Fatalf("names = %v", got)
	}
	if b.Dirty() {
		t.Fatal("Dirty() after an empty o")
	}
}

func TestUndoRedoOneStepAtATime(t *testing.T) {
	b := New("", orig())
	keys(t, b, "d", "d")
	keys(t, b, "d", "d")
	keys(t, b, "u")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"b", "c/"}) {
		t.Fatalf("after one u: %v", got)
	}
	keys(t, b, "u")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "b", "c/"}) {
		t.Fatalf("after two u: %v", got)
	}
	keys(t, b, "u")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "b", "c/"}) {
		t.Fatalf("u past the start: %v", got)
	}
	keys(t, b, "ctrl+r")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"b", "c/"}) {
		t.Fatalf("after ctrl+r: %v", got)
	}
	keys(t, b, "ctrl+r", "ctrl+r")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"c/"}) {
		t.Fatalf("after ctrl+r past the end: %v", got)
	}
}

func TestUndoRename(t *testing.T) {
	b := New("", orig())
	keys(t, b, "c", "c")
	keys(t, b, typed("q")...)
	keys(t, b, "esc", "u")
	if !reflect.DeepEqual(b.Rows(), orig()) || b.Dirty() {
		t.Fatalf("rows = %v", b.Rows())
	}
}

func TestEditPersistsAcrossFolders(t *testing.T) {
	b := New("", orig())
	keys(t, b, "j", "d", "d")
	if nav := keys(t, b, "G", "l"); nav != NavChild {
		t.Fatalf("l nav = %v", nav)
	}
	if cur, ok := b.Current(); !ok || cur.OrigID != "c" {
		t.Fatalf("current = %v %v", cur, ok)
	}
	b.Open("c", []Row{{OrigID: "c/x.md", Name: "x"}})
	if nav := keys(t, b, "h"); nav != NavParent {
		t.Fatalf("h nav = %v", nav)
	}
	b.Open("", orig())
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "c/"}) {
		t.Fatalf("pending edits lost: %v", got)
	}
	if b.Cursor() != 1 {
		t.Fatalf("cursor = %d", b.Cursor())
	}
	if got := b.Folders(); !reflect.DeepEqual(got, []string{"", "c"}) {
		t.Fatalf("folders = %v", got)
	}
	if got := b.Original(""); !reflect.DeepEqual(got, orig()) {
		t.Fatalf("original = %v", got)
	}
}

func TestMotions(t *testing.T) {
	b := New("", orig())
	keys(t, b, "G")
	if b.Cursor() != 2 {
		t.Fatalf("G cursor = %d", b.Cursor())
	}
	keys(t, b, "j")
	if b.Cursor() != 2 {
		t.Fatalf("j past end = %d", b.Cursor())
	}
	keys(t, b, "k", "g", "g")
	if b.Cursor() != 0 {
		t.Fatalf("gg cursor = %d", b.Cursor())
	}
	keys(t, b, "d", "esc", "d")
	if len(b.Rows()) != 3 {
		t.Fatal("esc did not cancel the pending d")
	}
}

func TestRowsIsACopy(t *testing.T) {
	b := New("", orig())
	b.Rows()[0].Name = "mutated"
	if b.Rows()[0].Name != "a" {
		t.Fatal("Rows() exposes internal state")
	}
}
