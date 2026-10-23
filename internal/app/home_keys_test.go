package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// finderOpen reports whether the note finder is on screen.
func finderOpen(v string) bool { return strings.Contains(v, "Enter open here") }

// themePickerOpen reports whether the theme picker is on screen.
func themePickerOpen(v string) bool { return strings.Contains(v, "onedark") }

func TestHomeLeaderKeysFollowTheKeymap(t *testing.T) {
	// "o" means nothing on the home screen: only the leader binding can open
	// the finder.
	h := newHarness(t, newTestStore(t), app.WithKeys(map[string][]string{"find_note": {"space o"}}))
	h.keys(" ")
	if v := h.view(); !strings.Contains(v, "<Space>") {
		t.Fatalf("space on the home screen should wait for the next key:\n%s", v)
	}
	h.keys("o")
	if v := h.view(); !finderOpen(v) || !strings.Contains(v, "HOME") {
		t.Fatalf("space o is find_note in the config and should open the finder over home:\n%s", v)
	}
}

func TestHomeDefaultLeaderKeysWork(t *testing.T) {
	// "h" means nothing on the home screen; space h is the default help key.
	h := newHarness(t, newTestStore(t))
	h.keys(" h")
	if v := h.view(); !strings.Contains(v, "MISOKU KEYMAPS") {
		t.Fatalf("space h (a default help key) should open the keymaps from home:\n%s", v)
	}
}

func TestHomeLeaderIsResolvedBeforeTheMenuMoves(t *testing.T) {
	// Tab moves the home selection, but after space it is the second key of
	// the leader sequence.
	h := newHarness(t, newTestStore(t), app.WithKeys(map[string][]string{"find_note": {"space tab"}}))
	h.keys(" ")
	h.key(tea.KeyTab)
	if v := h.view(); !finderOpen(v) {
		t.Fatalf("space tab is find_note in the config and should open the finder:\n%s", v)
	}
}

func TestHomeUnknownLeaderKeyDoesNothing(t *testing.T) {
	h := newHarness(t, newTestStore(t))
	h.keys(" z")
	v := h.view()
	if !strings.Contains(v, "HOME") || strings.Contains(v, "<Space>") || finderOpen(v) || themePickerOpen(v) {
		t.Fatalf("space z is not bound: home stays and the leader is cancelled:\n%s", v)
	}
	// The cancelled leader does not swallow the next key.
	h.keys("t")
	if v := h.view(); !themePickerOpen(v) {
		t.Fatalf("t after a cancelled leader should still open the themes:\n%s", v)
	}
}

func TestHomeCtrlShortcutsFollowTheKeymap(t *testing.T) {
	h := newHarness(t, newTestStore(t), app.WithKeys(map[string][]string{"themes": {"ctrl+t"}}))
	h.key(tea.KeyCtrlT)
	if v := h.view(); !themePickerOpen(v) {
		t.Fatalf("ctrl+t is themes in the config and should open the picker from home:\n%s", v)
	}
}

func TestHomeGraphEntryOpensTheGraph(t *testing.T) {
	h := newHarness(t, newTestStore(t))
	if v := h.view(); !strings.Contains(v, "Graph") {
		t.Fatalf("the home screen should list Graph:\n%s", v)
	}
	// Down from New Note to Graph: Find, Explorer, Themes, Keymaps, Graph.
	for i := 0; i < 5; i++ {
		h.key(tea.KeyDown)
	}
	h.key(tea.KeyEnter)
	v := h.view()
	if strings.Contains(v, "HOME") || !strings.Contains(strings.Split(v, "\n")[0], "Graph") {
		t.Fatalf("Enter on Graph should open the graph tab:\n%s", v)
	}
}

func TestHomeLetterKeysStillWork(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "hello"); err != nil {
		t.Fatal(err)
	}

	h := newHarness(t, store)
	h.keys("t")
	if v := h.view(); !themePickerOpen(v) {
		t.Fatalf("t should open the theme picker:\n%s", v)
	}

	h = newHarness(t, store)
	h.keys("f")
	if v := h.view(); !finderOpen(v) {
		t.Fatalf("f should open the finder:\n%s", v)
	}

	h = newHarness(t, store)
	h.keys("g")
	if v := h.view(); !strings.Contains(strings.Split(v, "\n")[0], "Graph") {
		t.Fatalf("g should open the graph:\n%s", v)
	}

	h = newHarness(t, store)
	h.keys("1")
	if v := h.view(); !strings.Contains(strings.Split(v, "\n")[0], "doc.md") {
		t.Fatalf("1 should open the most recent note:\n%s", v)
	}

	h = newHarness(t, store)
	h.keys("n")
	if v := h.view(); !strings.Contains(strings.Split(v, "\n")[0], "Untitled-1") {
		t.Fatalf("n should open a new note:\n%s", v)
	}

	h = newHarness(t, store)
	h.keys("q")
	if !h.quit {
		t.Fatal("q should quit from the home screen")
	}
}

func TestHomeCtrlNewNoteStartsAtTheWorkspaceRoot(t *testing.T) {
	// The explorer's selection (the work folder) does not decide where a
	// note made on the home screen goes: like n, it starts at the root.
	store := newTestStore(t)
	if _, err := store.SaveAs("work", "a", "x"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.key(tea.KeyCtrlN)
	h.key(tea.KeyEsc)
	h.keys(":w")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Save As") || !strings.Contains(v, "Untitled-1.md") || strings.Contains(v, "work/Untitled-1.md") {
		t.Fatalf("ctrl+n on home should save to the workspace root by default:\n%s", v)
	}
}
