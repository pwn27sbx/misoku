package app_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	historyFirst = "first draft words\nline two\nline three\nline four\n"
	historyToday = "today words\nline two\nline three\nline four\n"
)

// historyStore is a vault in the "notes" subfolder of a git repository:
// its note was committed as a.md, renamed to "b c.md" with git mv and
// committed, then edited and committed again.
func historyStore(t *testing.T) *core.Store {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	newTestStore(t) // keeps HOME and the Trash away from the real ones
	top := t.TempDir()
	vault := filepath.Join(top, "notes")
	if err := os.MkdirAll(vault, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, top, "init", "-q")
	if err := os.WriteFile(filepath.Join(vault, "a.md"), []byte(historyFirst), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, top, "add", "-A")
	gitIn(t, top, "commit", "-q", "-m", "first version")
	gitIn(t, top, "mv", "notes/a.md", "notes/b c.md")
	gitIn(t, top, "commit", "-q", "-m", "rename")
	if err := os.WriteFile(filepath.Join(vault, "b c.md"), []byte(historyToday), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, top, "commit", "-q", "-a", "-m", "add five")
	store, err := core.NewStore(vault)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

// openFirst opens the home screen's first recent note, in NORMAL mode.
func openFirst(h *harness) {
	h.t.Helper()
	h.keys("1")
	h.key(tea.KeyEsc)
}

// historyCommand types ":history" and Enter.
func historyCommand(h *harness) {
	h.t.Helper()
	h.keys(":history")
	h.key(tea.KeyEnter)
}

// S9: ":history" lists the note's commits across a rename, previews the
// selected version, and Enter puts it back in the buffer as one change
// that u undoes, writing nothing until the note is saved.
func TestHistoryRestoresTheVersionBeforeARename(t *testing.T) {
	store := historyStore(t)
	h := newHarness(t, store)
	openFirst(h)
	historyCommand(h)
	v := h.view()
	for _, want := range []string{"History of b c.md", "add five", "rename", "first version"} {
		if !strings.Contains(v, want) {
			t.Fatalf("%q missing from the history dialog:\n%s", want, v)
		}
	}
	if strings.Contains(v, "first draft words") {
		t.Fatalf("the newest version is selected, yet the oldest is shown:\n%s", v)
	}
	h.keys("j")
	h.key(tea.KeyDown)
	if v := h.view(); !strings.Contains(v, "first draft words") {
		t.Fatalf("the oldest version should be previewed:\n%s", v)
	}

	h.key(tea.KeyEnter)
	v = h.view()
	if strings.Contains(v, "History of") {
		t.Fatalf("Enter should close the dialog:\n%s", v)
	}
	if !strings.Contains(v, "Restored the version of") {
		t.Fatalf("no restore message:\n%s", v)
	}
	if strings.Contains(v, "today words") || !strings.Contains(v, "first draft words") {
		t.Fatalf("the buffer should hold the version before the rename:\n%s", v)
	}
	disk, err := os.ReadFile(filepath.Join(store.Root(), "b c.md"))
	if err != nil || string(disk) != historyToday {
		t.Fatalf("restoring must not write the note: disk %q, %v", disk, err)
	}

	h.keys("u")
	if v := h.view(); !strings.Contains(v, "today words") || strings.Contains(v, "first draft words") {
		t.Fatalf("u should bring today's text back:\n%s", v)
	}
	h.key(tea.KeyCtrlR)
	h.key(tea.KeyCtrlS)
	disk, err = os.ReadFile(filepath.Join(store.Root(), "b c.md"))
	if err != nil || string(disk) != historyFirst {
		t.Fatalf("saved after restore: %q, %v; want the version before the rename", disk, err)
	}
}

// S9: the dialog previews the version under the selection, and Esc
// closes it leaving the note as it was.
func TestHistoryPreviewsTheSelectedVersion(t *testing.T) {
	store := historyStore(t)
	h := newHarness(t, store)
	openFirst(h)
	historyCommand(h)
	if v := h.view(); strings.Contains(v, "first draft words") {
		t.Fatalf("the newest version should be previewed:\n%s", v)
	}
	h.keys("jj")
	if v := h.view(); !strings.Contains(v, "first draft words") {
		t.Fatalf("j should select and preview the oldest version:\n%s", v)
	}
	h.keys("kk")
	if v := h.view(); strings.Contains(v, "first draft words") {
		t.Fatalf("k should go back to the newest version:\n%s", v)
	}
	h.keys("jj")
	h.key(tea.KeyEsc)
	if v := h.view(); strings.Contains(v, "History of") || strings.Contains(v, "first draft words") {
		t.Fatalf("Esc should close the dialog and leave the note as it was:\n%s", v)
	}
}

// S9: what cannot have a history says why in the status bar.
func TestHistoryExplainsWhyThereIsNone(t *testing.T) {
	t.Run("not a repository", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not installed")
		}
		h := openWith(t, "# plain\n")
		historyCommand(h)
		if v := h.view(); !strings.Contains(v, "not a git repository") {
			t.Fatalf("want a not-a-repository message:\n%s", v)
		}
	})
	t.Run("untracked note", func(t *testing.T) {
		store := historyStore(t)
		if _, err := store.SaveAs("", "fresh", "never committed\n"); err != nil {
			t.Fatal(err)
		}
		h := newHarness(t, store)
		openFirst(h) // the newest note: fresh.md
		historyCommand(h)
		if v := h.view(); !strings.Contains(v, "fresh.md is not in git yet") {
			t.Fatalf("want an untracked message:\n%s", v)
		}
	})
	t.Run("draft", func(t *testing.T) {
		h := newHarness(t, historyStore(t))
		h.keys("n")
		h.key(tea.KeyEsc)
		historyCommand(h)
		if v := h.view(); !strings.Contains(v, "never saved") {
			t.Fatalf("want a draft message:\n%s", v)
		}
	})
	t.Run("no git", func(t *testing.T) {
		store := historyStore(t)
		h := newHarness(t, store)
		openFirst(h)
		t.Setenv("PATH", t.TempDir())
		t.Setenv("ProgramFiles", t.TempDir()) // Windows also looks there
		historyCommand(h)
		if v := h.view(); !strings.Contains(v, "git is not installed") {
			t.Fatalf("want a no-git message:\n%s", v)
		}
	})
}

// S9: a click selects a commit, a second click on it restores it, and a
// click outside the dialog closes it.
func TestHistoryMouseSelectsRestoresAndCloses(t *testing.T) {
	h := newHarness(t, historyStore(t))
	openFirst(h)
	historyCommand(h)
	x, y := find(t, h.view(), "first version")
	h.click(x, y)
	if v := h.view(); !strings.Contains(v, "History of") || !strings.Contains(v, "first draft words") {
		t.Fatalf("a click should select and preview that commit:\n%s", v)
	}
	h.click(x, y)
	if v := h.view(); strings.Contains(v, "History of") || !strings.Contains(v, "Restored the version of") {
		t.Fatalf("a click on the selected commit should restore it:\n%s", v)
	}
	historyCommand(h)
	h.click(0, 0)
	if v := h.view(); strings.Contains(v, "History of") {
		t.Fatalf("a click outside should close the dialog:\n%s", v)
	}
}
