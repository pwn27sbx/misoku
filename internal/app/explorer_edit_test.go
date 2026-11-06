package app_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// editHarness is a vault with aaa and ref (linking to aaa), a tab open on
// aaa and the explorer focused on its first row.
func editHarness(t *testing.T) *harness {
	t.Helper()
	h := relinkNotes(t, "aaa", "aaa body", "ref", "see [[aaa]]")
	h.openRow(0)
	h.keys(" e")
	h.keys("gg")
	return h
}

// retype replaces the selected line's name with name (cc, type, Esc).
func (h *harness) retype(name string) {
	h.t.Helper()
	h.keys("cc" + name)
	h.key(tea.KeyEsc)
}

// S8: m opens the folder editor; a renamed line is applied only after =
// and Yes, the tab follows and the links are rewritten.
func TestEditFolderRenameAppliesAfterTheConfirm(t *testing.T) {
	h := editHarness(t)
	h.keys("m")
	if v := h.view(); !strings.Contains(v, "EDIT /") {
		t.Fatalf("m must open the folder editor:\n%s", v)
	}
	h.retype("bbb")
	h.keys("=")
	v := h.view()
	for _, want := range []string{"CONFIRM FILE SYSTEM ACTIONS", "rename aaa.md → bbb.md", "Updates 1 link in 1 note"} {
		if !strings.Contains(v, want) {
			t.Fatalf("the confirm must show %q:\n%s", want, v)
		}
	}
	if !h.exists("aaa.md") || h.exists("bbb.md") {
		t.Fatal("nothing is renamed before the answer")
	}
	h.keys("y")
	if !h.exists("bbb.md") || h.exists("aaa.md") {
		t.Fatal("Yes must rename aaa.md to bbb.md")
	}
	if got := h.disk("ref.md"); got != "see [[bbb]]" {
		t.Errorf("ref.md = %q", got)
	}
	if tabs := app.OpenTabs(h.m); !slices.Equal(tabs, []string{"bbb.md"}) {
		t.Errorf("tabs = %q, want the renamed bbb.md", tabs)
	}
	if v := h.view(); strings.Contains(v, "EDIT /") {
		t.Errorf("an applied batch closes the editor:\n%s", v)
	}
}

// S8: :w is the same as =.
func TestEditFolderColonWAsksToo(t *testing.T) {
	h := editHarness(t)
	h.keys("m")
	h.retype("bbb")
	h.keys(":w")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "rename aaa.md → bbb.md") {
		t.Fatalf(":w must show the confirm:\n%s", v)
	}
}

// Leaving with pending edits (Esc or q) shows the same confirm as =, as
// mini.files does: cancel keeps editing, No drops the edits and closes,
// Yes applies them and closes. Leaving without edits asks nothing.
func TestEditFolderLeavingWithEditsConfirmsTheActions(t *testing.T) {
	h := editHarness(t)
	h.keys("m")
	h.key(tea.KeyEsc)
	if v := h.view(); strings.Contains(v, "EDIT /") || strings.Contains(v, "CONFIRM") {
		t.Fatalf("no edits: Esc leaves at once:\n%s", v)
	}

	h.keys("m")
	h.retype("bbb")
	h.key(tea.KeyEsc)
	v := h.view()
	for _, want := range []string{"CONFIRM FILE SYSTEM ACTIONS", "rename aaa.md → bbb.md", "Cancel"} {
		if !strings.Contains(v, want) {
			t.Fatalf("Esc with edits must show %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "Discard") {
		t.Fatalf("the old discard question is gone:\n%s", v)
	}
	h.key(tea.KeyEsc) // cancel
	if v := h.view(); !strings.Contains(v, "EDIT /") || strings.Contains(v, "CONFIRM") {
		t.Fatalf("cancel returns to the editor:\n%s", v)
	}
	h.keys("q")
	h.keys("n")
	if v := h.view(); strings.Contains(v, "EDIT /") {
		t.Fatalf("No closes the editor:\n%s", v)
	}
	if !h.exists("aaa.md") || h.exists("bbb.md") {
		t.Fatal("No writes nothing")
	}

	h.keys("m")
	h.retype("bbb")
	h.keys("q")
	h.keys("y")
	if !h.exists("bbb.md") || h.exists("aaa.md") {
		t.Fatal("Yes must apply the rename")
	}
	if v := h.view(); strings.Contains(v, "EDIT /") {
		t.Fatalf("Yes closes the editor:\n%s", v)
	}
}

// Typing names one per line with o, as in mini.files: a name ending in
// / is a folder, a/b is the nested path, exactly as n reads them.
func TestEditFolderTypedLinesCreateFoldersAndPaths(t *testing.T) {
	h := editHarness(t)
	h.keys("m")
	h.keys("Go")
	h.keys("ideas/")
	h.key(tea.KeyEsc)
	h.keys("o")
	h.keys("diario/hoy")
	h.key(tea.KeyEsc)
	h.keys("q")
	v := h.view()
	if !strings.Contains(v, "CONFIRM FILE SYSTEM ACTIONS") {
		t.Fatalf("leaving must confirm the new lines:\n%s", v)
	}
	h.keys("y")
	if fi, err := os.Stat(filepath.Join(h.store.Root(), "ideas")); err != nil || !fi.IsDir() {
		t.Fatalf("ideas/ must be a folder: %v", err)
	}
	if !h.exists("diario/hoy.md") {
		t.Fatal("diario/hoy must create the note hoy.md inside diario")
	}
}

// S8: l and h walk folders keeping each one's edits, so dd in one folder
// and p in another is a move.
func TestEditFolderMoveAcrossFolders(t *testing.T) {
	store := newTestStore(t)
	for _, n := range [][3]string{{"", "aaa", "A"}, {"Docs", "x", "X"}} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e")
	h.keys("Gm") // on the root note aaa: the editor opens on the root
	h.keys("gg")
	if v := h.view(); !strings.Contains(v, "EDIT /") || !strings.Contains(v, "Docs/") {
		t.Fatalf("the root, with folders ending in /:\n%s", v)
	}
	h.keys("jdd") // cut aaa (Docs/ is first)
	h.keys("ggl") // into Docs
	if v := h.view(); !strings.Contains(v, "EDIT Docs/") {
		t.Fatalf("l opens the folder under the cursor:\n%s", v)
	}
	h.keys("p")
	h.keys("h")
	h.keys("=")
	if v := h.view(); !strings.Contains(v, "move aaa.md → Docs/aaa.md") {
		t.Fatalf("the cut and paste must plan a move:\n%s", v)
	}
	h.keys("y")
	if !h.exists("Docs/aaa.md") || h.exists("aaa.md") {
		t.Fatal("Yes must move aaa.md into Docs")
	}
}

// Existing explorer keys keep working when the editor is closed, and are
// text while it is open.
func TestExplorerKeysUnchangedOutsideTheEditor(t *testing.T) {
	t.Run("n", func(t *testing.T) {
		h := editHarness(t)
		h.typeNew("zzz")
		if v := h.view(); !strings.Contains(v, "CREATE zzz.md") {
			t.Fatalf("n must still ask to create:\n%s", v)
		}
	})
	t.Run("rename", func(t *testing.T) {
		h := editHarness(t)
		h.renameRow(0, "aaa", "bbb")
		h.keys("y")
		if !h.exists("bbb.md") {
			t.Fatal("r must still rename")
		}
	})
	t.Run("delete", func(t *testing.T) {
		h := editHarness(t)
		h.keys("d")
		if v := h.view(); !strings.Contains(v, "Delete aaa.md?") {
			t.Fatalf("d must still ask to delete:\n%s", v)
		}
	})
	t.Run("M", func(t *testing.T) {
		fakeNvimOnPath(t)
		h, ran := fileManagerHarness(t, newTestStore(t), func(*exec.Cmd) error { return nil })
		h.keys("m")
		h.keys("M")
		if len(*ran) != 0 {
			t.Fatal("M is a key of the editor while it is open")
		}
		h.key(tea.KeyEsc)
		h.keys("M")
		if len(*ran) != 1 {
			t.Fatalf("M must still open mini.files, ran %d", len(*ran))
		}
	})
}

// In the tree, o is m then o, as in mini.files: on a folder row it opens a
// line below it in the folder holding it, and leaving confirms the new line.
func TestTreeOOpensALineInTheEditor(t *testing.T) {
	h, store := newEntryHarness(t) // FLM/ selected
	h.keys("o")
	h.keys("idea/")
	h.key(tea.KeyEsc)
	h.keys("q")
	if v := h.view(); !strings.Contains(v, "CONFIRM FILE SYSTEM ACTIONS") {
		t.Fatalf("leaving must confirm the new line:\n%s", v)
	}
	h.keys("y")
	if fi, err := os.Stat(filepath.Join(store.Root(), "idea")); err != nil || !fi.IsDir() {
		t.Fatalf("idea/ must be a folder at the root: %v", err)
	}
}

// In the tree, i is m then i on the selected entry: typing renames it.
func TestTreeIRenamesTheSelectedNote(t *testing.T) {
	h := editHarness(t) // aaa selected
	h.keys("i")
	h.keys("x")
	h.key(tea.KeyEsc)
	h.keys("q")
	h.keys("y")
	if !h.exists("xaaa.md") || h.exists("aaa.md") {
		t.Fatal("i then x must rename aaa.md to xaaa.md")
	}
}
