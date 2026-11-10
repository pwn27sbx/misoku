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
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "bs":
			msg = tea.KeyMsg{Type: tea.KeyBackspace}
		case "del":
			msg = tea.KeyMsg{Type: tea.KeyDelete}
		case "ctrl+a":
			msg = tea.KeyMsg{Type: tea.KeyCtrlA}
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

func TestInsertKeysOnAFolderWithLinesEditTheCurrentName(t *testing.T) {
	cases := []struct{ key, want string }{
		{"i", "xa"},
		{"I", "xa"},
		{"a", "ax"},
		{"A", "ax"},
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			b := New("", orig())
			keys(t, b, c.key)
			keys(t, b, typed("x")...)
			keys(t, b, "esc")
			want := []string{c.want, "b", "c/"}
			if got := names(b.Rows()); !reflect.DeepEqual(got, want) {
				t.Fatalf("names = %v, want %v", got, want)
			}
			if got := b.Rows()[0].OrigID; got != "a.md" {
				t.Fatalf("OrigID = %q, the rename must keep it", got)
			}
		})
	}
}

func TestChangeOnAnEmptyFolderDoesNotStartTyping(t *testing.T) {
	for _, seq := range [][]string{{"c", "w"}, {"c", "c"}} {
		t.Run(seq[0]+seq[1], func(t *testing.T) {
			b := New("", nil)
			keys(t, b, seq...)
			if b.Editing() {
				t.Fatal("change started typing with no line to change")
			}
			if got := b.Rows(); len(got) != 0 {
				t.Fatalf("rows = %v, want none", got)
			}
		})
	}
}

func TestInsertOnAnEmptyFolderThenEscDropsTheNewLine(t *testing.T) {
	b := New("", nil)
	keys(t, b, "i")
	if !b.Editing() {
		t.Fatal("i did not start typing on an empty folder")
	}
	if got := len(b.Rows()); got != 1 {
		t.Fatalf("rows while typing = %d, want the new line", got)
	}
	keys(t, b, "esc")
	if b.Editing() {
		t.Fatal("still typing after Esc")
	}
	if got := b.Rows(); len(got) != 0 {
		t.Fatalf("rows = %v, want none", got)
	}
	if b.Dirty() {
		t.Fatal("Dirty() after an empty i")
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

// S1: Enter while typing a new name keeps it as a line and opens an empty
// line below, still typing, as Oil does: it asks the host for nothing. The
// whole insert, every line it typed, is one undo step.
func TestEnterWhileTypingANewNameOpensALine(t *testing.T) {
	b := New("", orig())
	keys(t, b, "o")
	keys(t, b, typed("x/")...)
	if nav := keys(t, b, "enter"); nav != NavNone {
		t.Fatalf("Enter while typing = %v, want NavNone", nav)
	}
	if !b.Editing() {
		t.Fatal("Enter must go on typing on the new line")
	}
	keys(t, b, typed("y")...)
	keys(t, b, "esc")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "x/", "y", "b", "c/"}) {
		t.Fatalf("names = %v", got)
	}
	keys(t, b, "u")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "b", "c/"}) {
		t.Fatalf("one u undoes the whole insert, names = %v", got)
	}
}

// Enter while typing an existing line keeps its name and opens an empty
// line below, still typing, as Vim does.
func TestEnterWhileTypingAnExistingLineOpensOneBelow(t *testing.T) {
	b := New("", orig())
	keys(t, b, "i")
	keys(t, b, typed("z")...)
	if nav := keys(t, b, "enter"); nav != NavNone {
		t.Fatalf("Enter = %v, want NavNone", nav)
	}
	if !b.Editing() {
		t.Fatal("Enter must go on typing on the new line")
	}
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"za", "", "b", "c/"}) {
		t.Fatalf("names = %v", got)
	}
}

// Enter on an empty new line keeps typing on it: no blank lines pile up,
// and Esc then drops it.
func TestEnterOnAnEmptyNewLineKeepsTyping(t *testing.T) {
	b := New("", orig())
	if nav := keys(t, b, "o", "enter", "enter"); nav != NavNone {
		t.Fatalf("Enter = %v, want NavNone", nav)
	}
	if !b.Editing() {
		t.Fatal("Enter on an empty line goes on typing")
	}
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "", "b", "c/"}) {
		t.Fatalf("names = %v", got)
	}
	keys(t, b, "esc")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "b", "c/"}) {
		t.Fatalf("Esc drops the empty line, names = %v", got)
	}
}

// Enter in normal mode asks the host to open the line under the cursor:
// enter a folder, open a note.
func TestEnterInNormalAsksToOpen(t *testing.T) {
	b := New("", orig())
	if nav := keys(t, b, "enter"); nav != NavOpen {
		t.Fatalf("Enter in normal = %v, want NavOpen", nav)
	}
	if b.Editing() {
		t.Fatal("Enter in normal must not start typing")
	}
}

func TestSelectMovesTheCursorToAnEntry(t *testing.T) {
	b := New("", []Row{{OrigID: "a.md", Name: "a"}, {OrigID: "b.md", Name: "b"}})
	if !b.Select("b.md") || b.Cursor() != 1 {
		t.Fatalf("Select(b.md): cursor = %d, want 1", b.Cursor())
	}
	if b.Select("") || b.Select("zz.md") || b.Cursor() != 1 {
		t.Fatalf("an unknown entry keeps the cursor, got %d", b.Cursor())
	}
}

func TestArrowsMoveInNormalMode(t *testing.T) {
	b := New("", orig())
	keys(t, b, "down", "down")
	if b.Cursor() != 2 {
		t.Fatalf("cursor = %d after two downs, want 2", b.Cursor())
	}
	keys(t, b, "up")
	if b.Cursor() != 1 {
		t.Fatalf("cursor = %d after up, want 1", b.Cursor())
	}
}

func TestArrowsInInsertKeepTheNameAndMoveTheLine(t *testing.T) {
	b := New("", orig())
	keys(t, b, "A")
	keys(t, b, typed("x")...)
	keys(t, b, "down")
	if !b.Editing() || b.Cursor() != 1 {
		t.Fatalf("editing=%v cursor=%d, want insert on line 1", b.Editing(), b.Cursor())
	}
	if got := b.Rows()[0].Name; got != "ax" {
		t.Fatalf("line 0 = %q, want the typed ax kept", got)
	}
	if b.InputView() == "" || b.InputCursorX() != 1 {
		t.Fatalf("cursor x = %d, want the end of b", b.InputCursorX())
	}
	keys(t, b, typed("y")...)
	keys(t, b, "up")
	keys(t, b, typed("z")...)
	keys(t, b, "esc")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"axz", "by", "c/"}) {
		t.Fatalf("names = %v", got)
	}
	keys(t, b, "u")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "b", "c/"}) {
		t.Fatalf("one u must undo the whole insert, names = %v", got)
	}
}

func TestArrowsInInsertStayAtTheEdges(t *testing.T) {
	b := New("", orig())
	keys(t, b, "A", "up")
	if !b.Editing() || b.Cursor() != 0 {
		t.Fatalf("up on the first line: editing=%v cursor=%d", b.Editing(), b.Cursor())
	}
	keys(t, b, "esc", "G", "A", "down")
	if !b.Editing() || b.Cursor() != 2 {
		t.Fatalf("down on the last line: editing=%v cursor=%d", b.Editing(), b.Cursor())
	}
}

func TestArrowsInInsertDropAnEmptyNewLine(t *testing.T) {
	b := New("", orig())
	keys(t, b, "o", "down")
	if !b.Editing() || b.Cursor() != 1 || b.Rows()[1].Name != "b" {
		t.Fatalf("down from an empty new line: cursor=%d names=%v", b.Cursor(), names(b.Rows()))
	}
	keys(t, b, "esc", "O", "up")
	if !b.Editing() || b.Cursor() != 0 || len(b.Rows()) != 3 {
		t.Fatalf("up from an empty new line: cursor=%d names=%v", b.Cursor(), names(b.Rows()))
	}
}

// S8: Backspace at the start of a line joins it into the line above, as
// Vim does: an emptied line disappears (a planned delete), typing goes on
// at the end of the line above, and it is all one undo step.
func TestBackspaceAtStartJoinsIntoTheLineAbove(t *testing.T) {
	b := New("", orig())
	keys(t, b, "j", "a", "bs", "bs")
	if !b.Editing() || b.Cursor() != 0 {
		t.Fatalf("typing goes on on the line above: editing %v, cursor %d", b.Editing(), b.Cursor())
	}
	keys(t, b, typed("x")...)
	keys(t, b, "esc")
	if got := b.Rows(); !reflect.DeepEqual(got, []Row{{OrigID: "a.md", Name: "ax"}, {OrigID: "c", Name: "c/"}}) {
		t.Fatalf("rows = %v", got)
	}
	keys(t, b, "u")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a", "b", "c/"}) {
		t.Fatalf("one u undoes the join, names = %v", got)
	}
}

// S8: Backspace at the start of a line with text left joins that text at
// the end of the line above, with the cursor at the join.
func TestBackspaceJoinsTheRestOfTheLine(t *testing.T) {
	b := New("", orig())
	keys(t, b, "j", "i", "bs")
	keys(t, b, typed("-")...)
	keys(t, b, "esc")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a-b", "c/"}) {
		t.Fatalf("names = %v", got)
	}
	keys(t, b, "gg", "i", "bs")
	if got := names(b.Edited("")); !reflect.DeepEqual(got, []string{"a-b", "c/"}) {
		t.Fatalf("Backspace on the first line does nothing, names = %v", got)
	}
}

// S8: Delete at the end of a line joins the line below into it; typing
// goes on at the join.
func TestDeleteAtEndJoinsTheLineBelow(t *testing.T) {
	b := New("", orig())
	keys(t, b, "a", "del")
	keys(t, b, typed("!")...)
	keys(t, b, "esc")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"a!b", "c/"}) {
		t.Fatalf("names = %v", got)
	}
}

// Edited shows the name being typed as if Esc were pressed now, so the
// host can plan while typing (Ctrl+S); an empty new line is left out.
func TestEditedIncludesTheNameBeingTyped(t *testing.T) {
	b := New("", orig())
	keys(t, b, "a")
	keys(t, b, typed("z")...)
	if got := names(b.Edited("")); !reflect.DeepEqual(got, []string{"az", "b", "c/"}) {
		t.Fatalf("Edited = %v", got)
	}
	keys(t, b, "enter")
	if got := names(b.Edited("")); !reflect.DeepEqual(got, []string{"az", "b", "c/"}) {
		t.Fatalf("Edited leaves the empty new line out, got %v", got)
	}
	if got := len(b.Rows()); got != 4 {
		t.Fatalf("Rows still shows the line being typed: %d rows", got)
	}
}

// Reload restarts on rows read from disk with the cursor at the given row
// and every folder's edits forgotten; typing goes on when it was.
func TestReloadKeepsTyping(t *testing.T) {
	b := New("", orig())
	keys(t, b, "a")
	keys(t, b, typed("z")...)
	b.Reload([]Row{{OrigID: "az.md", Name: "az"}, {OrigID: "b.md", Name: "b"}}, 1)
	if !b.Editing() || b.Cursor() != 1 || b.Dirty() {
		t.Fatalf("editing %v, cursor %d, dirty %v", b.Editing(), b.Cursor(), b.Dirty())
	}
	keys(t, b, typed("!")...)
	keys(t, b, "esc")
	if got := names(b.Rows()); !reflect.DeepEqual(got, []string{"az", "b!"}) {
		t.Fatalf("names = %v", got)
	}
}
