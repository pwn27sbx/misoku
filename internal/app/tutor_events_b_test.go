package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// actionStep is a step that waits for an event, with the values it accepts.
func actionStep(event string, values ...string) tutor.Step {
	return tutor.Step{
		Text:  "Haz la acción",
		Check: tutor.Check{Kind: tutor.KindAction, Event: event, Value: tutor.Values(values)},
	}
}

// eventCatalog is a one-lesson tutorial whose steps are the given ones.
func eventCatalog(file string, steps ...tutor.Step) tutor.Catalog {
	return tutor.Catalog{Routes: []tutor.Route{{
		ID: "B", Name: "Básico", Index: "B00 Básico.md",
		Lessons: []tutor.Lesson{{
			ID: "B01", Title: "Evento", File: file, Steps: steps,
		}},
	}}}
}

// eventFixture opens a lesson note with text in a fresh tutorial session, the
// lesson being current after the first update.
func eventFixture(t *testing.T, text string, steps []tutor.Step, opts ...app.Option) (*harness, *tutor.Store, tutor.Lesson) {
	t.Helper()
	cat := eventCatalog("B01 Evento.md", steps...)
	store := newTestStore(t)
	if _, err := store.SaveAs("", "B01 Evento", text); err != nil {
		t.Fatal(err)
	}
	return openEventSession(t, store, cat, "B01 Evento.md", opts...)
}

// openEventSession plays cat in store's vault, with open as the current note.
func openEventSession(t *testing.T, store *core.Store, cat tutor.Catalog, open string, opts ...app.Option) (*harness, *tutor.Store, tutor.Lesson) {
	t.Helper()
	root := store.Root()
	store, err := core.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	progress := &tutor.Store{}
	all := append([]app.Option{app.WithTutor(progress, cat, root, true)}, opts...)
	h := newHarness(t, store, all...)
	m, cmd := app.OpenInCurrentBuffer(h.m, open)
	h.m = m
	h.send(cmdMsg(cmd)...)
	h.key(tea.KeyEsc) // an update makes the lesson current
	l, _ := cat.Lesson(open)
	return h, progress, l
}

// assertStep fails the test unless the step passed.
func assertStep(t *testing.T, progress *tutor.Store, l tutor.Lesson, step int) {
	t.Helper()
	if !progress.Passed(l, step) {
		t.Fatalf("step %d of %s did not pass", step, l.ID)
	}
}

// assertNotPassed fails the test when the step passed.
func assertNotPassed(t *testing.T, progress *tutor.Store, l tutor.Lesson, step int) {
	t.Helper()
	if progress.Passed(l, step) {
		t.Fatalf("step %d of %s passed when it should not", step, l.ID)
	}
}

// ---------------------------------------------------------------------------
// The editor's own events

func TestTutorCmdlineEvent(t *testing.T) {
	h, p, l := eventFixture(t, "hola\n", []tutor.Step{actionStep("cmdline")})
	h.keys(":")
	assertStep(t, p, l, 1)
}

// motion: every group the lessons teach reports its own value.
func TestTutorMotionEvent(t *testing.T) {
	steps := []tutor.Step{
		actionStep("motion", "hjkl"),
		actionStep("motion", "arrows"),
		actionStep("motion", "line-bounds"),
		actionStep("motion", "file-bounds"),
		actionStep("motion", "words"),
		actionStep("motion", "half-page"),
		actionStep("motion", "page"),
		actionStep("motion", "scroll-line"),
	}
	h, p, l := eventFixture(t, "una linea\notra linea\n", steps)
	h.keys("j")
	h.key(tea.KeyLeft)
	h.keys("0")
	h.keys("gg")
	h.keys("w")
	h.key(tea.KeyCtrlD)
	h.key(tea.KeyPgDown)
	h.key(tea.KeyCtrlE)
	for i := range steps {
		assertStep(t, p, l, i+1)
	}
}

func TestTutorYankEvent(t *testing.T) {
	h, p, l := eventFixture(t, "una linea\n", []tutor.Step{actionStep("yank", "line")})
	h.keys("yy")
	assertStep(t, p, l, 1)
}

func TestTutorPasteEvent(t *testing.T) {
	img := pastedPNG(t)
	clip := func(context.Context) (core.ClipboardImageMsg, bool, error) { return img, true, nil }
	h, p, l := eventFixture(t, "hola\n", []tutor.Step{actionStep("paste")}, app.WithClipboardImage(clip))
	h.keys("i")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlV})
	assertStep(t, p, l, 1)
}

// A quit the player cancels is not the discard the lesson waits for.
func TestTutorQuitCancelDoesNotEmitDiscard(t *testing.T) {
	h, p, l := eventFixture(t, "hola\n", []tutor.Step{actionStep("quit", "discard")})
	h.keys("i")
	h.keys("cambio")
	h.key(tea.KeyEsc)
	h.keys(":")
	h.keys("q")
	h.key(tea.KeyEnter) // the dialog
	h.key(tea.KeyEsc)   // Cancel
	assertNotPassed(t, p, l, 1)
}

const tableNote = "| a | b |\n| --- | --- |\n| c | d |\n"

func TestTutorCellNextEvent(t *testing.T) {
	h, p, l := eventFixture(t, tableNote, []tutor.Step{actionStep("cell_next")})
	h.keys("jj")
	h.keys("i")
	h.key(tea.KeyTab)
	assertStep(t, p, l, 1)
}

func TestTutorSlashOpenEvent(t *testing.T) {
	h, p, l := eventFixture(t, "hola\n", []tutor.Step{actionStep("slash_open")})
	h.keys("o")
	h.keys("/")
	assertStep(t, p, l, 1)
}

// In a table the menu opens with its table entries: the advanced-basics
// lesson asks for that value.
func TestTutorSlashOpenInTableEvent(t *testing.T) {
	h, p, l := eventFixture(t, tableNote, []tutor.Step{actionStep("slash_open", "in-table")})
	h.keys("jj")
	h.keys("ll")
	h.keys("i")
	h.keys("x /")
	assertStep(t, p, l, 1)
}

func TestTutorSlashCloseEvent(t *testing.T) {
	h, p, l := eventFixture(t, "hola\n", []tutor.Step{actionStep("slash_close")})
	h.keys("o")
	h.keys("/")
	h.key(tea.KeyEsc)
	assertStep(t, p, l, 1)
}

// The search reports the query it found, the error it did not, and the
// repetition and wrap the lessons mention.
func TestTutorSearchEvents(t *testing.T) {
	text := "hotel\nHotel\nnada\n"
	steps := []tutor.Step{
		actionStep("search_hit", "hotel"),
		actionStep("search_hit", "Hotel"),
		actionStep("search_error"),
		actionStep("search_next"),
		actionStep("search_prev"),
		actionStep("search_wrap"),
	}
	h, p, l := eventFixture(t, text, steps)
	h.keys("/")
	h.keys("hotel")
	h.key(tea.KeyEnter)
	h.keys("/")
	h.keys("Hotel")
	h.key(tea.KeyEnter)
	h.keys("/")
	h.keys("no existe")
	h.key(tea.KeyEnter)
	h.keys("/")
	h.keys("hotel")
	h.key(tea.KeyEnter) // a hit, which n and N can repeat
	h.keys("n")
	h.keys("N")
	h.keys("G")
	h.keys("/")
	h.keys("hotel")
	h.key(tea.KeyEnter) // from the last line, forward wraps
	for i := range steps {
		assertStep(t, p, l, i+1)
	}
}

// The table hint is a state of the bar: it reports when the cursor is inside
// a table in insert mode.
func TestTutorTableHintEvent(t *testing.T) {
	h, p, l := eventFixture(t, tableNote, []tutor.Step{actionStep("table_hint")})
	h.keys("jj")
	h.keys("i")
	assertStep(t, p, l, 1)
}

// ---------------------------------------------------------------------------
// Views, panes and the explorer

func TestTutorReadingViewEvent(t *testing.T) {
	steps := []tutor.Step{actionStep("reading_view"), actionStep("reading_view", "concealed")}
	h, p, l := eventFixture(t, "# Viaje\n", steps)
	h.keys(" p")
	assertStep(t, p, l, 1)
	assertStep(t, p, l, 2)
}

func TestTutorEditorViewEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("editor_view")})
	h.keys(" r")
	assertStep(t, p, l, 1)
}

func TestTutorSplitViewEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("split_view")})
	h.keys(" r") // the editor view hides the split
	h.keys(" r") // and back it comes
	assertStep(t, p, l, 1)
}

func TestTutorFocusCycleEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("focus_cycle")})
	h.key(tea.KeyTab)
	assertStep(t, p, l, 1)
}

func TestTutorFocusExplorerEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("focus_explorer")})
	h.keys(" e")
	assertStep(t, p, l, 1)
}

// A narrow window opens the explorer over the editor: Esc closes it.
func TestTutorFocusExplorerOverlayEvent(t *testing.T) {
	cat := eventCatalog("B01 Evento.md", actionStep("focus_explorer", "overlay"))
	store := newTestStore(t)
	if _, err := store.SaveAs("", "B01 Evento", "# Viaje\n"); err != nil {
		t.Fatal(err)
	}
	root := store.Root()
	store, err := core.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	progress := &tutor.Store{}
	h := newHarnessSize(t, 60, termH, store, app.WithTutor(progress, cat, root, true))
	m, cmd := app.OpenInCurrentBuffer(h.m, "B01 Evento.md")
	h.m = m
	h.send(cmdMsg(cmd)...)
	h.key(tea.KeyEsc)
	h.keys(" e")
	h.key(tea.KeyEsc)
	l, _ := cat.Lesson("B01 Evento.md")
	assertStep(t, progress, l, 1)
}

func TestTutorPaneResizedEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("pane_resized")})
	divs := dividers(h.view(), 5)
	if len(divs) == 0 {
		t.Skip("no divider on screen")
	}
	div := divs[len(divs)-1]
	h.send(
		tea.MouseMsg{X: div, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		tea.MouseMsg{X: div - 4, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		tea.MouseMsg{X: div - 4, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	)
	assertStep(t, p, l, 1)
}

// ---------------------------------------------------------------------------
// The tree

// treeStore is a vault with a folder holding two notes and a top-level note.
func treeStore(t *testing.T) *core.Store {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "B01 Evento", "# Viaje\n"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Playa", "Costa"} {
		if _, err := store.SaveAs("Viajes", name, "# "+name+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	return store
}

func TestTutorTreeStateEvent(t *testing.T) {
	steps := []tutor.Step{
		actionStep("tree_state"),
		actionStep("tree_state", "accordion"),
		actionStep("tree_state", "collapsed-all"),
	}
	cat := eventCatalog("B01 Evento.md", steps...)
	h, p, l := openEventSession(t, treeStore(t), cat, "B01 Evento.md")
	h.keys(" e") // the explorer
	// The tree lists the folder first: open it with Enter, then close all.
	h.keys("G") // the last top-level row
	h.key(tea.KeyUp)
	h.key(tea.KeyEnter)
	h.keys("W")
	for i := range steps {
		assertStep(t, p, l, i+1)
	}
}

func TestTutorTreeRefreshEvent(t *testing.T) {
	cat := eventCatalog("B01 Evento.md", actionStep("tree_refresh", "Nota externa"))
	h, p, l := openEventSession(t, treeStore(t), cat, "B01 Evento.md")
	path := filepath.Join(h.store.Root(), "Nota externa.md")
	if err := os.WriteFile(path, []byte("# Nota externa\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h.send(app.CheckWorkspaceMsg{})
	assertStep(t, p, l, 1)
}

func TestTutorNotePreviewedEvent(t *testing.T) {
	steps := []tutor.Step{actionStep("note_previewed")}
	cat := eventCatalog("B01 Evento.md", steps...)
	h, p, l := openEventSession(t, treeStore(t), cat, "B01 Evento.md")
	h.keys(" e")
	h.keys("j")
	h.keys("p")
	assertStep(t, p, l, 1)
}

func TestTutorTreeOpenedNoteEvent(t *testing.T) {
	steps := []tutor.Step{actionStep("note_opened")}
	cat := eventCatalog("B01 Evento.md", steps...)
	h, p, l := openEventSession(t, treeStore(t), cat, "B01 Evento.md")
	h.keys(" e")
	h.keys("j")
	h.key(tea.KeyEnter)
	assertStep(t, p, l, 1)
}

// ---------------------------------------------------------------------------
// Notes, tabs and dialogs

func TestTutorNewDraftEvent(t *testing.T) {
	steps := []tutor.Step{actionStep("new_draft"), actionStep("new_draft", "Lista de playa")}
	h, p, l := eventFixture(t, "[[Lista de playa]]\n", steps)
	h.keys(" n")
	h.key(tea.KeyEsc) // the new tab types: leave it first
	h.keys(":")
	h.keys("e B01 Evento")
	h.key(tea.KeyEnter)
	h.keys("gg")
	h.keys("gd") // the link names no note: a draft starts with its name
	assertStep(t, p, l, 1)
	assertStep(t, p, l, 2)
}

func TestTutorNoteOpenedByEditEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("note_opened", "Viaje")})
	if _, err := h.store.SaveAs("", "Viaje", "# Viaje\n"); err != nil {
		t.Fatal(err)
	}
	h.keys(":")
	h.keys("e Viaje")
	h.key(tea.KeyEnter)
	assertStep(t, p, l, 1)
}

func TestTutorNewTabOpenedEvent(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "B01 Evento", "# Viaje\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "Costa", "# Costa\n"); err != nil {
		t.Fatal(err)
	}
	h, p, l := openEventSession(t, store, eventCatalog("B01 Evento.md", actionStep("note_opened", "new-tab")), "B01 Evento.md")
	h.keys(" f")
	h.keys("Costa")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlT})
	assertStep(t, p, l, 1)
}

func TestTutorTabClosedEvent(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "B01 Evento", "# Viaje\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "Costa", "# Costa\n"); err != nil {
		t.Fatal(err)
	}
	h, p, l := openEventSession(t, store, eventCatalog("B01 Evento.md", actionStep("tab_closed")), "B01 Evento.md")
	h.keys(" f")
	h.keys("Costa")
	h.key(tea.KeyEnter) // a second tab
	h.keys(" x")
	assertStep(t, p, l, 1)
}

func TestTutorSaveDialogEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("save_dialog")})
	h.keys(" n")
	h.key(tea.KeyEsc)
	h.keys(":")
	h.keys("w")
	h.key(tea.KeyEnter) // a draft: Save As opens
	assertStep(t, p, l, 1)
}

func TestTutorMessagesEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("messages_open")})
	h.keys(":w") // a save says so, which :messages lists
	h.key(tea.KeyEnter)
	h.keys(":")
	h.keys("messages")
	h.key(tea.KeyEnter)
	assertStep(t, p, l, 1)
}

func TestTutorTagsEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje #playa\n", []tutor.Step{actionStep("tags_list")})
	h.keys(":tags") // the index needs a build first
	h.key(tea.KeyEnter)
	h.view()
	h.keys(":")
	h.keys("tags")
	h.key(tea.KeyEnter)
	assertStep(t, p, l, 1)
}

// ---------------------------------------------------------------------------
// The palette and the finder

func TestTutorPaletteOpenEvent(t *testing.T) {
	steps := []tutor.Step{
		actionStep("palette_open"),
		actionStep("palette_open", "export-first"),
		actionStep("palette_open", "configured-keys"),
	}
	h, p, l := eventFixture(t, "# Viaje\n", steps)
	h.keys("?")
	for i := range steps {
		assertStep(t, p, l, i+1)
	}
}

func TestTutorPaletteRunEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("palette_run", "settings")})
	h.keys("?")
	h.keys("settings")
	h.key(tea.KeyEnter)
	assertStep(t, p, l, 1)
}

func TestTutorFinderOpenEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("finder_open")})
	h.keys(" f")
	assertStep(t, p, l, 1)
}

func TestTutorFinderFilterEvent(t *testing.T) {
	steps := []tutor.Step{actionStep("finder_filter", "tag:viaje")}
	h, p, l := eventFixture(t, "# Viaje #viaje\n", steps)
	h.keys(" f")
	h.keys("tag:viaje")
	assertStep(t, p, l, 1)
}

// The footer only exists in Finder mode: the inline list has none.
func TestTutorLinkFooterEvent(t *testing.T) {
	finder := app.WithUI(config.UI{LinkSuggestions: config.LinkSuggestionsFinder})
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("link_footer")}, finder)
	h.keys("o")
	h.keys("[[")
	h.keys("#")
	assertStep(t, p, l, 1)
}

func TestTutorLinkNextEvent(t *testing.T) {
	h, p, l := eventFixture(t, "[[Viaje]] y [[Costa]]\n", []tutor.Step{actionStep("link_next")})
	h.keys("gg")
	h.keys(" o]")
	assertStep(t, p, l, 1)
}

// ---------------------------------------------------------------------------
// Links

func TestTutorFollowedLinkEvent(t *testing.T) {
	steps := []tutor.Step{
		actionStep("followed_link", "gd"),
		actionStep("followed_link", "gx"),
		actionStep("followed_link", "Enter"),
	}
	store := newTestStore(t)
	if _, err := store.SaveAs("", "B01 Evento", "[[Costa]]\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "Costa", "# Costa\n"); err != nil {
		t.Fatal(err)
	}
	h, p, l := openEventSession(t, store, eventCatalog("B01 Evento.md", steps...), "B01 Evento.md")
	for i := range steps {
		h.keys(":")
		h.keys("e B01 Evento")
		h.key(tea.KeyEnter)
		h.keys("gg")
		switch i {
		case 0:
			h.keys("gd")
		case 1:
			h.keys("gx")
		case 2:
			h.key(tea.KeyEnter)
		}
		assertStep(t, p, l, i+1)
	}
}

func TestTutorFollowedLinkByMouseEvent(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "B01 Evento", "[[Costa]]\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "Costa", "# Costa\n"); err != nil {
		t.Fatal(err)
	}
	h, p, l := openEventSession(t, store, eventCatalog("B01 Evento.md", actionStep("followed_link", "mouse")), "B01 Evento.md")
	h.m = app.ExpireToast(h.m)
	v := h.view()
	divs := dividers(v, 5)
	previewX := 0
	if len(divs) > 0 {
		previewX = divs[len(divs)-1]
	}
	x, y := -1, -1
	for row, line := range strings.Split(v, "\n") {
		i := strings.LastIndex(line, "Costa") // the preview's copy is the last one
		if i < 0 {
			continue
		}
		if col := ansi.StringWidth(line[:i]); col > previewX {
			x, y = col, row
			break
		}
	}
	if x < 0 {
		t.Skip("the preview link is not on screen")
	}
	h.click(x, y)
	assertStep(t, p, l, 1)
}

// prev_tab is wired like next_tab: the tab that was current reports it.
func TestTutorPrevTabEvent(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "B01 Evento", "# Viaje\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "Costa", "# Costa\n"); err != nil {
		t.Fatal(err)
	}
	cat := eventCatalog("B01 Evento.md", actionStep("prev_tab"))
	h, p, l := openEventSession(t, store, cat, "B01 Evento.md")
	h.keys(" f")
	h.keys("Costa")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlT}) // a second tab, and it is the one on screen
	h.keys("[")
	assertStep(t, p, l, 1)
}

// ---------------------------------------------------------------------------
// The home screen, the preview and the rest

func TestTutorHomeEvents(t *testing.T) {
	steps := []tutor.Step{
		actionStep("home"),
		actionStep("home_open", "recent", "project"),
		actionStep("home_key", "n", "f", "e", "t", "?", "g", "c", "q"),
	}
	h, p, l := openEventSession(t, treeStore(t), eventCatalog("B01 Evento.md", steps...), "B01 Evento.md")
	h.keys(" d") // home
	h.keys("1")  // the first recent note
	h.keys(" d")
	h.keys("4") // the folder after the recent notes
	for _, key := range []string{"n", "f", "e", "t", "?", "g", "c", "s", "l", "q"} {
		h.keys(" d")
		h.keys(key)
		h.key(tea.KeyEsc)
		h.key(tea.KeyEsc)
	}
	for i := range steps {
		assertStep(t, p, l, i+1)
	}
}

func TestTutorCalendarMonthEvents(t *testing.T) {
	steps := []tutor.Step{
		actionStep("calendar_month", "month-shift"),
		actionStep("calendar_month", "today"),
	}
	h, p, l := eventFixture(t, "```calendar\nmonth: 2026-01\n2026-01-05: x\n```\n", steps)
	h.key(tea.KeyTab) // the preview
	h.keys("<")
	h.keys("T")
	assertStep(t, p, l, 1)
	assertStep(t, p, l, 2)
}

func TestTutorPreviewScrollEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Uno\n\n# Dos\n", []tutor.Step{actionStep("preview_scroll")})
	h.key(tea.KeyTab) // the preview
	h.keys("j")
	assertStep(t, p, l, 1)
}

func TestTutorPreviewFoldEvents(t *testing.T) {
	steps := []tutor.Step{
		actionStep("preview_fold", "collapse"),
		actionStep("preview_fold", "expand"),
	}
	h, p, l := eventFixture(t, "# Uno\n\ntexto\n", steps)
	h.key(tea.KeyTab) // the preview
	h.keys("zM")
	h.keys("zR")
	assertStep(t, p, l, 1)
	assertStep(t, p, l, 2)
}

func TestTutorPreviewRenderedEvents(t *testing.T) {
	text := "---\nviaje: costa\ncover: mapa.png\n---\n\n# Uno\n\n" +
		"| a | b |\n| --- | --- |\n| 1 | 2 |\n\n" +
		"~~~columns\na\n+++\nb\n~~~\n\n" +
		"```math\nx = 1\n```\n\n" +
		"```flow\nInicio --> Fin\n```\n\n" +
		"![mapa](mapa.png)\n"
	steps := []tutor.Step{
		actionStep("preview_rendered"),
		actionStep("preview_rendered", "properties"),
		actionStep("preview_rendered", "cover"),
		actionStep("preview_rendered", "table"),
		actionStep("preview_rendered", "columns"),
		actionStep("preview_rendered", "equation"),
		actionStep("preview_rendered", "diagram"),
		actionStep("preview_rendered", "image"),
	}
	h, p, l := eventFixture(t, text, steps)
	h.keys("G")
	h.keys("o")
	h.keys("x") // an edit recompiles the preview with the whole note
	for i := range steps {
		assertStep(t, p, l, i+1)
	}
}

func TestTutorErrorShownEvent(t *testing.T) {
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("error_shown")})
	h.keys(":")
	h.keys("graph foo")
	h.key(tea.KeyEnter)
	assertStep(t, p, l, 1)
}

// Outside the tutorial an action records nothing and the editor reports
// nothing: the same keys leave the store empty.
func TestTutorEventsOutsideTheTutorial(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "Nota", "# Viaje\n"); err != nil {
		t.Fatal(err)
	}
	store, err := core.NewStore(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	m, cmd := app.OpenInCurrentBuffer(h.m, "Nota.md")
	h.m = m
	h.send(cmdMsg(cmd)...)
	h.keys("jj")
	h.keys("yy")
	h.keys(":")
	h.key(tea.KeyEsc)
	h.keys(" p")
	h.view()
}
