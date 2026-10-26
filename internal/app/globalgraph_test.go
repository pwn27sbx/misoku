package app_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

// globalStore is a vault with a folder: Home links to work/Plan and to
// Ghost, a note that doesn't exist yet; work/Plan links back; Orphan links
// to nothing.
func globalStore(t *testing.T) *harness {
	t.Helper()
	store := newTestStore(t)
	for _, n := range [][3]string{
		{"", "Home", "go to [[work/Plan]] and [[Ghost]]"},
		{"work", "Plan", "back [[Home]]"},
		{"", "Orphan", "alone"},
	} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	return newHarness(t, store)
}

// openHome opens Home in a tab, with the focus on the editor.
func (h *harness) openHome() {
	h.t.Helper()
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("home")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
}

// globalWorkspace has Home open and the global graph shown, from ":graph all".
func globalWorkspace(t *testing.T) *harness {
	t.Helper()
	h := globalStore(t)
	h.openHome()
	h.ex("graph all")
	return h
}

// globalTitle is the global graph's title row, or "" when it isn't showing.
func globalTitle(h *harness) string {
	h.t.Helper()
	for _, l := range strings.Split(editorPane(h.view()), "\n") {
		if strings.Contains(l, " notes · ") || strings.Contains(l, " note · ") {
			return strings.TrimSpace(l)
		}
	}
	return ""
}

func TestGraphAllOpensTheGlobalGraph(t *testing.T) {
	h := globalWorkspace(t)
	title := globalTitle(h)
	for _, want := range []string{"Home", "3 notes", "1 orphan", "1 unresolved"} {
		if !strings.Contains(title, want) {
			t.Fatalf("title %q lacks %q:\n%s", title, want, h.view())
		}
	}
	h.keys("++") // close up: every name is drawn
	v := h.view()
	if tabs := strings.Split(v, "\n")[0]; !strings.Contains(tabs, "Global graph") {
		t.Fatalf("the global graph should open in its own tab:\n%s", v)
	}
	ed := editorPane(v)
	for _, name := range []string{"Orphan", "Ghost"} {
		if !strings.Contains(ed, name) {
			t.Fatalf("every note and unresolved link should be on the map, %s is not:\n%s", name, ed)
		}
	}
	if !strings.Contains(previewPane(v), "go to") {
		t.Fatalf("the preview should show the note the graph was opened from:\n%s", v)
	}
}

// ":graph" alone still opens the local graph; an unknown argument is an
// error and opens nothing.
func TestGraphCommandArguments(t *testing.T) {
	h := globalStore(t)
	h.openHome()
	h.ex("graph foo")
	v := h.view()
	if !strings.Contains(v, "E488: Trailing characters: foo (use :graph or :graph all)") {
		t.Fatalf(":graph foo should be an error:\n%s", v)
	}
	if strings.Contains(strings.Split(v, "\n")[0], "Graph") || strings.Contains(strings.Split(v, "\n")[0], "graph") {
		t.Fatalf(":graph foo should open no graph:\n%s", v)
	}
	h.ex("graph")
	if ed := editorPane(h.view()); !strings.Contains(ed, "Home · 1 link · 1 backlink") || globalTitle(h) != "" {
		t.Fatalf(":graph should still open the local graph:\n%s", h.view())
	}
}

func TestSpaceShiftGOpensTheGlobalGraph(t *testing.T) {
	h := globalStore(t)
	h.openHome()
	h.keys(" G")
	if !strings.Contains(globalTitle(h), "3 notes") {
		t.Fatalf("space G from a note should open the global graph:\n%s", h.view())
	}

	h = globalStore(t) // from the home screen
	h.keys(" G")
	if title := globalTitle(h); !strings.Contains(title, "3 notes") || !strings.Contains(title, "All notes") {
		t.Fatalf("space G from home should open the global graph, nothing selected:\n%s", h.view())
	}

	h = globalStore(t) // from the palette opened on the home screen
	h.keys("?")
	typeQuery(h, "graph of every")
	h.key(tea.KeyEnter)
	if !strings.Contains(globalTitle(h), "3 notes") {
		t.Fatalf("the palette on home should open the global graph:\n%s", h.view())
	}
}

func TestGlobalGraphKeyFollowsTheConfig(t *testing.T) {
	store := globalStore(t).store
	h := newHarness(t, store, app.WithKeys(map[string][]string{"global_graph": {"space y"}}))
	h.openHome()
	h.keys(" G")
	if globalTitle(h) != "" {
		t.Fatalf("space G was remapped away and should not open the global graph:\n%s", h.view())
	}
	h.keys(" y")
	if !strings.Contains(globalTitle(h), "3 notes") {
		t.Fatalf("space y is global_graph in the config:\n%s", h.view())
	}
}

// A user binding wins over the graph's own letter, as in the local graph.
func TestUserBindingStealsAGlobalGraphLetter(t *testing.T) {
	store := globalStore(t).store
	h := newHarness(t, store, app.WithKeys(map[string][]string{"themes": {"n"}}))
	h.openHome()
	h.ex("graph all")
	h.keys("n")
	if v := h.view(); !themePickerOpen(v) {
		t.Fatalf("n is themes in the config and should open the picker, not go round the links:\n%s", v)
	}
}

// The tab has no file behind it: it never looks unsaved, never saves,
// never asks before quitting.
func TestGlobalGraphTabIsNotAnUnsavedNote(t *testing.T) {
	h := globalWorkspace(t)
	if v := h.view(); strings.Contains(v, "Global graph new") || strings.Contains(v, "unsaved") || strings.Contains(v, "0 words") {
		t.Fatalf("the global graph tab should not look like an unsaved note:\n%s", v)
	}
	h.keys(" w") // save
	if v := h.view(); strings.Contains(v, "Save as") || !strings.Contains(globalTitle(h), "3 notes") {
		t.Fatalf("saving the global graph tab should do nothing:\n%s", v)
	}
	h.clickPreview("Plan") // the preview of Home links to work/Plan
	if v := h.view(); strings.Contains(v, "unsaved changes") || !strings.Contains(v, "1 back [[Home]]") {
		t.Fatalf("following a link from the global graph tab should just open the note:\n%s", v)
	}
	h.key(tea.KeyCtrlC)
	if !h.quit {
		t.Fatalf("quitting should not ask about the global graph tab:\n%s", h.view())
	}
}

func TestGlobalGraphSelectionPreviewsAndOpens(t *testing.T) {
	h := globalWorkspace(t)
	h.keys("n") // Home's first link
	if title := globalTitle(h); !strings.Contains(title, "Ghost") {
		t.Fatalf("n should select Home's first link, Ghost:\n%s", h.view())
	}
	h.keys("n")
	if !strings.Contains(globalTitle(h), "Plan") || !strings.Contains(previewPane(h.view()), "back") {
		t.Fatalf("selecting work/Plan should preview it:\n%s", h.view())
	}
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "1 back [[Home]]") || !strings.Contains(strings.Split(v, "\n")[0], "Global graph") {
		t.Fatalf("enter should open work/Plan next to the global graph tab:\n%s", v)
	}

	h = globalWorkspace(t)
	h.keys("++")
	h.clickGraph("Orphan")
	if !strings.Contains(previewPane(h.view()), "alone") {
		t.Fatalf("clicking Orphan should preview it:\n%s", h.view())
	}
	h.clickGraph("● Orphan") // its dot: the title row names it now too
	if v := h.view(); !strings.Contains(v, "1 alone") {
		t.Fatalf("a second click should open Orphan:\n%s", v)
	}
}

// Enter on a link target that has no note starts it as a draft, as
// following the link does: next to the note linking to it, nothing written.
func TestGlobalGraphEnterOnAnUnresolvedTargetStartsADraft(t *testing.T) {
	h := globalWorkspace(t)
	h.keys("n")
	h.key(tea.KeyEnter)
	v := h.view()
	if !strings.Contains(v, `New note "Ghost": save to create it`) || !strings.Contains(strings.Split(v, "\n")[0], "Ghost") {
		t.Fatalf("enter on Ghost should start it as a new note:\n%s", v)
	}
	if notes := h.store.List(); len(notes) != 4 { // 3 notes and the work folder
		t.Fatalf("nothing should be written before saving, the vault has %d entries", len(notes))
	}
}

func TestSwitchingTabsKeepsEachGraphsSelection(t *testing.T) {
	h := globalStore(t)
	h.openHome()
	h.ex("graph") // the local graph, centred on Home
	h.keys(" G")
	h.keys("++")
	h.clickGraph("Orphan")
	h.clickTab("Graph")
	if ed := editorPane(h.view()); !strings.Contains(ed, "Home · 1 link · 1 backlink") || !strings.Contains(previewPane(h.view()), "go to") {
		t.Fatalf("the local graph should keep Home selected and preview it:\n%s", h.view())
	}
	h.clickTab("Global graph")
	if !strings.Contains(globalTitle(h), "Orphan") || !strings.Contains(previewPane(h.view()), "alone") {
		t.Fatalf("the global graph should keep Orphan selected and preview it:\n%s", h.view())
	}
}

func TestClosingTheGlobalGraphTab(t *testing.T) {
	h := globalWorkspace(t)
	h.keys(" x")
	if v := h.view(); strings.Contains(strings.Split(v, "\n")[0], "Global graph") || globalTitle(h) != "" {
		t.Fatalf("space x should close the global graph tab:\n%s", v)
	}
	if v := h.view(); !strings.Contains(v, "go to [[work/Plan]]") {
		t.Fatalf("Home should show again:\n%s", v)
	}
}

// A link added by another program reaches the map on the next check.
func TestGlobalGraphFollowsLinksAddedOutside(t *testing.T) {
	h := globalWorkspace(t)
	editOutside(t, h.store.Root(), "Orphan.md", "now [[Home]]")
	h.send(app.CheckWorkspaceMsg{})
	if title := globalTitle(h); !strings.Contains(title, "0 orphans") {
		t.Fatalf("Orphan links to Home now:\n%s", h.view())
	}
}

// Links typed but not saved show in the map, and stay through a refresh of
// the index that another note's change starts.
func TestGlobalGraphShowsUnsavedLinks(t *testing.T) {
	h := globalStore(t)
	h.openHome()
	h.keys("ggi")
	h.paste("[[Orphan]] [[Later]] ") // typed, "[[" would open the note finder
	h.key(tea.KeyEsc)
	h.ex("graph all")
	if title := globalTitle(h); !strings.Contains(title, "0 orphans") || !strings.Contains(title, "2 unresolved") {
		t.Fatalf("the unsaved links of Home should show:\n%s", h.view())
	}
	editOutside(t, h.store.Root(), "work/Plan.md", "back [[Home]] again")
	h.send(app.CheckWorkspaceMsg{})
	if title := globalTitle(h); !strings.Contains(title, "0 orphans") || !strings.Contains(title, "2 unresolved") {
		t.Fatalf("the unsaved links should stay after the index refresh:\n%s", h.view())
	}
}

// heldMaps runs cmd and every command it leads to, like the runtime, but
// keeps back the global graph's laid-out maps instead of delivering them.
func (h *harness) heldMaps(cmd tea.Cmd) []tea.Msg { h.t.Helper(); return h.held(cmd, "globalMap") }

// held is heldMaps for the messages whose type name contains kind.
func (h *harness) held(cmd tea.Cmd, kind string) []tea.Msg {
	h.t.Helper()
	var held []tea.Msg
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch out := runCmd(c).(type) {
		case nil:
		case tea.BatchMsg:
			queue = append(queue, out...)
		default:
			if strings.Contains(fmt.Sprintf("%T", out), kind) {
				held = append(held, out)
				continue
			}
			queue = append(queue, h.update(out))
		}
	}
	return held
}

// A map laid out from an older index that lands after a newer one is
// dropped: the newer links stay.
func TestGlobalGraphDropsAMapOlderThanTheOneShown(t *testing.T) {
	h := globalWorkspace(t)
	editOutside(t, h.store.Root(), "Orphan.md", "now [[Home]]")
	older := h.heldMaps(h.update(app.CheckWorkspaceMsg{}))
	editOutside(t, h.store.Root(), "Orphan.md", "now [[Home]] and [[Elsewhere]]")
	newer := h.heldMaps(h.update(app.CheckWorkspaceMsg{}))
	if len(older) != 1 || len(newer) != 1 {
		t.Fatalf("each check should lay out one map, got %d and %d", len(older), len(newer))
	}
	h.send(newer[0])
	h.send(older[0])
	if title := globalTitle(h); !strings.Contains(title, "2 unresolved") {
		t.Fatalf("the older map should be dropped, keeping Elsewhere:\n%s", h.view())
	}
}

// Opened before the first link index lands, the map waits for it, then
// shows up with it.
func TestGlobalGraphWaitsForTheFirstIndex(t *testing.T) {
	store := globalStore(t).store
	h := &harness{t: t, m: app.New(store), store: store}
	builds := h.held(h.update(tea.WindowSizeMsg{Width: termW, Height: termH}), "linkIndexed")
	var cmd tea.Cmd
	for _, r := range " G" {
		cmd = h.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		builds = append(builds, h.held(cmd, "linkIndexed")...)
	}
	if len(builds) == 0 {
		t.Fatal("the first link index build should have started")
	}
	if title := globalTitle(h); title != "" {
		t.Fatalf("there is no map before the first index:\n%s", h.view())
	}
	h.send(builds[len(builds)-1])
	if title := globalTitle(h); !strings.Contains(title, "3 notes") {
		t.Fatalf("the map should show once the index lands:\n%s", h.view())
	}
}

// A missing note named without a folder is started next to the note that
// links to it, as following that link would.
func TestGlobalGraphStartsAMissingNoteNextToItsLink(t *testing.T) {
	h := globalStore(t)
	if _, err := h.store.SaveAs("work", "Ideas", "see [[Someday]]"); err != nil {
		t.Fatal(err)
	}
	h = newHarness(t, h.store)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}})
	h.keys("ideas")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.ex("graph all")
	h.keys("n") // its only link
	if !strings.Contains(globalTitle(h), "Someday") {
		t.Fatalf("n should select Someday:\n%s", h.view())
	}
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.ex("w")
	if v := h.view(); !strings.Contains(v, "work/Someday.md") {
		t.Fatalf("Someday should be offered in work, next to Ideas:\n%s", v)
	}
}

// Links typed in a note while the global graph's tab is open behind it
// reach the map, as they reach the graph.
func TestGlobalGraphFollowsLinksTypedInAnotherTab(t *testing.T) {
	h := globalWorkspace(t)
	h.clickTab("Home.md")
	h.keys("ggi")
	h.paste("[[Orphan]] ")
	h.key(tea.KeyEsc)
	h.clickTab("Global graph")
	if title := globalTitle(h); !strings.Contains(title, "0 orphans") {
		t.Fatalf("the link typed in Home should reach the map:\n%s", h.view())
	}
}

// While the filter prompt is open the app's own keys are text, as in the
// editor's INSERT mode: the leader, ? for the palette, [ and ] for tabs, \
// to find a note and tab for the next pane.
func TestGlobalGraphFilterPromptTakesAppKeysAsText(t *testing.T) {
	h := globalWorkspace(t)
	h.keys(`/ ?[]\G`)
	h.key(tea.KeyTab)
	h.key(tea.KeyBackspace) // reaches the prompt only if the focus stayed
	v := h.view()
	if strings.Contains(v, paletteTitle) || !strings.Contains(strings.Split(v, "\n")[0], "Global graph") {
		t.Fatalf("keys typed in the prompt should not reach the app:\n%s", v)
	}
	if ed := editorPane(v); !strings.Contains(ed, `/  ?[]\`) || strings.Contains(ed, `/  ?[]\G`) {
		t.Fatalf("the prompt should hold every key typed:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.keys("?")
	if v := h.view(); !strings.Contains(v, paletteTitle) {
		t.Fatalf("with the prompt closed, ? opens the palette again:\n%s", v)
	}
}

// / jumps to the note found and previews it; o and u hide orphans and
// missing notes, and the title says so.
func TestGlobalGraphFilterAndToggles(t *testing.T) {
	h := globalWorkspace(t)
	h.keys("/plan")
	h.key(tea.KeyEnter)
	if title := globalTitle(h); !strings.Contains(title, "Plan") || !strings.Contains(previewPane(h.view()), "back") {
		t.Fatalf("enter should select work/Plan and preview it:\n%s", h.view())
	}
	h = globalWorkspace(t)
	h.keys("ou++") // close up: without the toggles Orphan and Ghost show here
	ed := editorPane(h.view())
	if !strings.Contains(globalTitle(h), "1 orphan hidden") || strings.Contains(ed, "Orphan") || strings.Contains(ed, "Ghost") || !strings.Contains(ed, "Home") {
		t.Fatalf("o and u should hide Orphan and the missing Ghost:\n%s", h.view())
	}
}

// A user binding wins over the filter's and the toggles' letters, but not
// while the prompt is open: then the letter is text.
func TestUserBindingStealsAToggleLetter(t *testing.T) {
	store := globalStore(t).store
	h := newHarness(t, store, app.WithKeys(map[string][]string{"themes": {"o"}}))
	h.openHome()
	h.ex("graph all")
	h.keys("/o")
	if v := h.view(); themePickerOpen(v) {
		t.Fatalf("o typed in the prompt is text:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.keys("o")
	if v := h.view(); !themePickerOpen(v) || strings.Contains(v, "orphan hidden") {
		t.Fatalf("o is themes in the config and should open the picker, not hide orphans:\n%s", v)
	}
}
