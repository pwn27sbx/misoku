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

// Home has no note to centre a local graph on: its Graph entry opens the
// global graph, every note of the vault.
func TestHomeGraphEntryOpensTheGlobalGraph(t *testing.T) {
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
	if strings.Contains(v, "HOME") || !strings.Contains(strings.Split(v, "\n")[0], "Global graph") {
		t.Fatalf("Enter on Graph should open the global graph tab:\n%s", v)
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
	if v := h.view(); !strings.Contains(strings.Split(v, "\n")[0], "Global graph") || !strings.Contains(v, "1 note · ") {
		t.Fatalf("g should open the global graph:\n%s", v)
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

// Space on the home screen shows the same leader guide as in a note: after
// the delay it lists the keys, an unknown key closes it with a warning, and
// a leader key still runs while it is open.
func TestHomeSpaceOpensTheLeaderGuide(t *testing.T) {
	h := newHarness(t, newTestStore(t))
	m, cmd := h.m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	h.m = m
	if cmd == nil {
		t.Fatal("Space on the home screen started no guide timer")
	}
	tickGuide(h)
	if v := h.view(); !guideOpen(h) || !strings.Contains(v, "HOME") {
		t.Fatalf("the leader guide should show over the home screen:\n%s", v)
	}
	h.keys("z")
	v := h.view()
	if guideOpen(h) || !strings.Contains(v, "z: not a leader key") || !strings.Contains(v, "HOME") {
		t.Fatalf("an unknown key should close the guide, warn and stay home:\n%s", v)
	}
	h.keys(" ")
	tickGuide(h)
	h.keys("f")
	if v := h.view(); guideOpen(h) || !finderOpen(v) {
		t.Fatalf("space f should open the finder from the guide:\n%s", v)
	}
}

// The home screen lists Settings: its key c and a click on the entry open
// the settings menu.
func TestHomeSettingsEntryOpensTheSettingsMenu(t *testing.T) {
	settingsOpen := func(v string) bool { return strings.Contains(v, "Accordion folders") }
	h := newHarness(t, newTestStore(t))
	if v := h.view(); !strings.Contains(v, "Settings") || settingsOpen(v) {
		t.Fatalf("the home screen should list a Settings entry:\n%s", v)
	}
	h.keys("c")
	if v := h.view(); !settingsOpen(v) || !strings.Contains(v, "HOME") {
		t.Fatalf("c should open the settings menu over home:\n%s", v)
	}
	h = newHarness(t, newTestStore(t))
	x, y := find(t, h.view(), "Settings")
	h.click(x, y)
	if v := h.view(); !settingsOpen(v) {
		t.Fatalf("clicking Settings should open the settings menu:\n%s", v)
	}
}
