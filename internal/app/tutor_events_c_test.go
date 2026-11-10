package app_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// The events of routes Avanzado and Experto (F3c). Each test plays a lesson
// whose step waits for one event, does what the lesson asks, and checks the
// step passed through the progress store.

// eventLesson is a one-lesson Avanzado catalog whose only step waits for
// event with every value in values.
func eventLesson(event string, values ...string) tutor.Catalog {
	return eventLessonSteps([]tutor.Step{{
		Text:  "Hazlo",
		Check: tutor.Check{Kind: tutor.KindAction, Event: event, Value: tutor.Values(values)},
	}})
}

// eventLessonSteps is eventLesson with the given steps.
func eventLessonSteps(steps []tutor.Step) tutor.Catalog {
	return tutor.Catalog{Routes: []tutor.Route{{
		ID: "A", Name: "Avanzado", Index: "A00 Avanzado.md",
		Lessons: []tutor.Lesson{{
			ID: "A01", Title: "Evento", File: "A01 Evento.md", Steps: steps,
		}},
	}}}
}

// eventLessonFile is the lesson note of every event test.
const eventLessonFile = "A01 Evento.md"

// eventVault is a fresh test vault holding the lesson note and any other
// notes.
func eventVault(t *testing.T, text string, notes map[string]string) *core.Store {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", strings.TrimSuffix(eventLessonFile, ".md"), text); err != nil {
		t.Fatal(err)
	}
	for name, body := range notes {
		if _, err := store.SaveAs("", name, body); err != nil {
			t.Fatal(err)
		}
	}
	fresh, err := core.NewStore(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

// eventHarness opens the lesson note of cat in a vault of other notes and
// plays it with the tutorial on.
func eventHarness(t *testing.T, cat tutor.Catalog, text string, notes map[string]string, opts ...app.Option) (*harness, *tutor.Store, tutor.Lesson) {
	t.Helper()
	store := eventVault(t, text, notes)
	progress := &tutor.Store{}
	all := append([]app.Option{app.WithTutor(progress, cat, store.Root(), true)}, opts...)
	h := newHarness(t, store, all...)
	m, cmd := app.OpenInCurrentBuffer(h.m, eventLessonFile)
	h.m = m
	sendCmd(t, h, cmd)
	h.key(tea.KeyEsc) // an update makes the lesson the current one
	l, _ := cat.Lesson(eventLessonFile)
	return h, progress, l
}

// sendCmd feeds cmd's messages to the harness, flattening the batch the
// open helper returns (the harness' send would drop a BatchMsg).
func sendCmd(t *testing.T, h *harness, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	out := runCmd(cmd)
	if batch, ok := out.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			if msg := runCmd(c); msg != nil {
				h.send(msg)
			}
		}
		return
	}
	if out != nil {
		h.send(out)
	}
}

// passed fails the test unless step of the lesson passed.
func passed(t *testing.T, progress *tutor.Store, l tutor.Lesson, step int) {
	t.Helper()
	if !progress.Passed(l, step) {
		t.Fatalf("%s#%d did not pass", l.ID, step)
	}
}

// notPassed fails the test when step of the lesson passed.
func notPassed(t *testing.T, progress *tutor.Store, l tutor.Lesson, step int) {
	t.Helper()
	if progress.Passed(l, step) {
		t.Fatalf("%s#%d passed when it should not", l.ID, step)
	}
}

// ---------------------------------------------------------------------------
// Graph (A11, A12, A13, E18)

// A11.1 / A12.1: opening the local graph and the global one report which
// one opened.
func TestTutorGraphOpenEvents(t *testing.T) {
	cat := eventLessonSteps([]tutor.Step{
		{Text: "Abre el grafo local", Check: tutor.Check{Kind: tutor.KindAction, Event: "graph_open", Value: tutor.Values{"local"}}},
		{Text: "Abre el global", Check: tutor.Check{Kind: tutor.KindAction, Event: "graph_open", Value: tutor.Values{"global"}}},
	})
	h, progress, l := eventHarness(t, cat, "# A01\n\ncuerpo\n", nil)
	h.keys(" g")
	h.keys(" G")
	passed(t, progress, l, 1)
	passed(t, progress, l, 2)
}

// A11.2 / E18.1: the graph keys are credited once the map is there.
func TestTutorGraphMoveEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("graph_move", "hl"), "# A01\n\ncuerpo\n", nil)
	h.keys(" g")
	h.keys("h")
	passed(t, progress, l, 1)
}

// A11.3: a click on the local map credits the mouse gesture.
func TestTutorGraphMouseEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("graph_mouse"), "# A01\n\ncuerpo\n", nil)
	h.keys(" g")
	div := dividers(h.view(), bodyRow)
	h.click(div[0]+2, bodyRow)
	passed(t, progress, l, 1)
}

// E18.2: / opens the global map's filter.
func TestTutorGraphFilterEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("graph_filter"), "# A01\n\ncuerpo\n", nil)
	h.keys(" G")
	h.keys("/")
	passed(t, progress, l, 1)
}

// A12.4: a link written in another tab refreshes the open map.
func TestTutorGraphRefreshEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("graph_refresh"), "# A01\n\ncuerpo\n", nil)
	h.keys(" g")
	h.keys("L") // back to the lesson's tab; the graph stays open in its own
	h.keys("i")
	h.keys("[[Otra]]")
	h.key(tea.KeyEsc)
	h.ex("w")
	passed(t, progress, l, 1)
}

// ---------------------------------------------------------------------------
// History (A06)

// historyEventHarness plays a lesson inside a git repository with the lesson
// committed twice, so :history has versions.
func historyEventHarness(t *testing.T, cat tutor.Catalog) (*harness, *tutor.Store, tutor.Lesson) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	newTestStore(t) // keeps HOME and the Trash away from the real ones
	vault := t.TempDir()
	gitIn(t, vault, "init", "-q")
	writeOutside(t, vault, eventLessonFile, "# A01\n\nprimera\n")
	gitIn(t, vault, "add", "-A")
	gitIn(t, vault, "commit", "-q", "-m", "primera")
	writeOutside(t, vault, eventLessonFile, "# A01\n\nsegunda\n")
	gitIn(t, vault, "add", "-A")
	gitIn(t, vault, "commit", "-q", "-m", "segunda")
	store, err := core.NewStore(vault)
	if err != nil {
		t.Fatal(err)
	}
	progress := &tutor.Store{}
	h := newHarness(t, store, app.WithTutor(progress, cat, vault, true))
	m, cmd := app.OpenInCurrentBuffer(h.m, eventLessonFile)
	h.m = m
	sendCmd(t, h, cmd)
	h.key(tea.KeyEsc)
	l, _ := cat.Lesson(eventLessonFile)
	return h, progress, l
}

// A06.1: :history on a note with commits opens the list.
func TestTutorHistoryOpenEvent(t *testing.T) {
	h, progress, l := historyEventHarness(t, eventLesson("history_open"))
	h.ex("history")
	passed(t, progress, l, 1)
}

// A06.2: moving over the versions credits the preview.
func TestTutorHistoryPreviewEvent(t *testing.T) {
	h, progress, l := historyEventHarness(t, eventLesson("history_preview"))
	h.ex("history")
	h.keys("j")
	passed(t, progress, l, 1)
}

// A06.4: a note with no commits reports the error, not the list.
func TestTutorHistoryErrorEvent(t *testing.T) {
	cat := eventLessonSteps([]tutor.Step{
		{Text: "Prueba una nota sin commits", Check: tutor.Check{Kind: tutor.KindAction, Event: "history_error"}},
		{Text: "Abre una con commits", Check: tutor.Check{Kind: tutor.KindAction, Event: "history_open"}},
	})
	h, progress, l := historyEventHarness(t, cat)
	// A note never committed, opened while the lesson stays current.
	writeOutside(t, h.store.Root(), "nueva.md", "sin commits\n")
	h.send(app.CheckWorkspaceMsg{})
	h.keys("\\")
	h.keys("nueva")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEsc)
	h.ex("history")
	passed(t, progress, l, 1)
	notPassed(t, progress, l, 2)
}

// ---------------------------------------------------------------------------
// Backlinks and mentions (A07)

// A07.1: the backlinks list the preview draws under the current lesson.
func TestTutorBacklinksEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("backlinks"), "# A01\n\ncuerpo\n",
		map[string]string{"Otra": "ver [[A01 Evento]]"})
	if v := h.view(); !strings.Contains(v, "Linked from 1 note") {
		t.Fatalf("setup: the backlinks should show:\n%s", v)
	}
	passed(t, progress, l, 1)
}

// A07.2: a click on an unlinked mention's row is its own event.
func TestTutorMentionOpenEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("mention_open"), "# A01\n\ncuerpo\n",
		map[string]string{"Otra": "hablamos de A01 Evento en clase"})
	if v := h.view(); !strings.Contains(v, "Unlinked mentions") {
		t.Fatalf("setup: the mentions should show:\n%s", v)
	}
	h.clickPreview("󰈙 Otra")
	passed(t, progress, l, 1)
}

// A07.4: a click on a backlink's row is its own event.
func TestTutorBacklinkOpenEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("backlink_open"), "# A01\n\ncuerpo\n",
		map[string]string{"Otra": "ver [[A01 Evento]]"})
	h.clickPreview("󰈙 Otra")
	passed(t, progress, l, 1)
}

// ---------------------------------------------------------------------------
// Preview and links (A10, A18)

// A10.2: a link the system may run asks first; the dialog is the event.
func TestTutorLinkConfirmEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("link_confirm", "Cancel"),
		"# A01\n\nAbre [raro](file:///etc/hosts)\n", nil)
	h.clickPreview("raro")
	passed(t, progress, l, 1)
}

// A18.3: resting the mouse on a block puts its handles on screen.
func TestTutorBlockHandlesEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("block_handles", "visible"), "# A01\n\ncuerpo\n", nil)
	h.hoverPreview()
	passed(t, progress, l, 1)
}

// ---------------------------------------------------------------------------
// Status glyphs (A04, A05)

// A05.2: the four statuses draw a glyph in the explorer.
func TestTutorStatusGlyphEvent(t *testing.T) {
	cat := eventLesson("status_glyph", "active", "onhold", "completed", "dropped")
	text := "---\nstatus: active\n---\n\n# A01\n\ncuerpo\n"
	h, progress, l := eventHarness(t, cat, text, map[string]string{
		"B onhold":    "---\nstatus: onhold\n---\n\nb\n",
		"C completed": "---\nstatus: completed\n---\n\nc\n",
		"D dropped":   "---\nstatus: dropped\n---\n\nd\n",
	})
	// A save reads the vault's statuses again.
	h.keys("G")
	h.keys("o")
	h.keys("x")
	h.key(tea.KeyEsc)
	h.ex("w")
	passed(t, progress, l, 1)
}

// A05.3: a status the app does not know draws no glyph.
func TestTutorStatusGlyphNoneEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("status_glyph", "none"),
		"---\nstatus: cualquier cosa\n---\n\n# A01\n\ncuerpo\n", nil)
	h.keys("G")
	h.keys("o")
	h.keys("x")
	h.key(tea.KeyEsc)
	h.ex("w")
	passed(t, progress, l, 1)
}

// ---------------------------------------------------------------------------
// Settings, themes and the cursor (E04, E07, E08)

// E04.1: opening :settings reports the menu.
func TestTutorSettingsOpenEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("settings_open"), "# A01\n\ncuerpo\n", nil)
	h.ex("settings")
	passed(t, progress, l, 1)
}

// E04.1: a row changed and saved reports the save.
func TestTutorSettingsSavedEvent(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(cfg, config.Config{}); err != nil {
		t.Fatal(err)
	}
	h, progress, l := eventHarness(t, eventLesson("settings_saved"), "# A01\n\ncuerpo\n", nil, app.WithConfigPath(cfg))
	h.ex("settings")
	h.key(tea.KeyEnter) // the first row: accordion folders, applied and saved
	passed(t, progress, l, 1)
}

// E08.3: opening the picker shows the whole list.
func TestTutorThemeListEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("theme_list"), "# A01\n\ncuerpo\n", nil)
	h.keys(" t")
	passed(t, progress, l, 1)
}

// E08.1: Enter keeps the theme the picker previewed.
func TestTutorThemePickEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("theme_pick"), "# A01\n\ncuerpo\n", nil)
	h.keys(" t")
	h.key(tea.KeyEnter)
	passed(t, progress, l, 1)
}

// E08.2: Esc puts the theme back and reports it as a revert; no settings are
// written, so a cancelled pick never credits the save (E08.1).
func TestTutorThemeRevertEvent(t *testing.T) {
	cat := eventLessonSteps([]tutor.Step{
		{Text: "Sal con Esc", Check: tutor.Check{Kind: tutor.KindAction, Event: "theme_pick", Value: tutor.Values{"revert"}}},
		{Text: "Guarda", Check: tutor.Check{Kind: tutor.KindAction, Event: "settings_saved"}},
	})
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(cfg, config.Config{}); err != nil {
		t.Fatal(err)
	}
	h, progress, l := eventHarness(t, cat, "# A01\n\ncuerpo\n", nil, app.WithConfigPath(cfg))
	h.keys(" t")
	h.key(tea.KeyEsc)
	passed(t, progress, l, 1)
	notPassed(t, progress, l, 2)
}

// E07.2: the terminal cursor takes a bar in INSERT and a block in NORMAL.
func TestTutorCursorShapeEvent(t *testing.T) {
	_, cursorOpt := newCursorOut()
	h, progress, l := eventHarness(t, eventLesson("cursor_shape", "block", "bar"), "# A01\n\ncuerpo\n", nil, cursorOpt)
	h.keys("i")
	h.key(tea.KeyEsc)
	passed(t, progress, l, 1)
}

// ---------------------------------------------------------------------------
// Explorer, quit-time changes and vim (A19, A20, E19)

// A20.1: m opens the folder editor.
func TestTutorDirbufOpenEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("dirbuf_open"), "# A01\n\ncuerpo\n", nil)
	h.keys(" e") // focus the explorer
	h.keys("m")
	passed(t, progress, l, 1)
}

// A20.3: i, a and o open the folder editor on the selected note.
func TestTutorDirbufInsertKeysEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("dirbuf_open", "i", "a", "o"), "# A01\n\ncuerpo\n", nil)
	h.keys(" e")
	for _, k := range []string{"i", "a", "o"} {
		h.keys(k)
		h.key(tea.KeyEsc) // leaves the name being edited
		h.key(tea.KeyEsc) // closes the folder editor
	}
	passed(t, progress, l, 1)
}

// ---------------------------------------------------------------------------
// Keys and the leader (E01, E03)

// E01.1 / E01.2: the guide and its second page.
func TestTutorLeaderGuideEvent(t *testing.T) {
	cat := eventLessonSteps([]tutor.Step{
		{Text: "Espera la guía", Check: tutor.Check{Kind: tutor.KindAction, Event: "leader_guide"}},
		{Text: "Abre la página de misoku", Check: tutor.Check{Kind: tutor.KindAction, Event: "leader_guide", Value: tutor.Values{"subpage"}}},
	})
	h, progress, l := eventHarness(t, cat, "# A01\n\ncuerpo\n", nil)
	h.keys(" ")
	tickGuide(h)
	h.key(tea.KeyEsc)
	h.keys(" ")
	h.keys("o")
	tickGuide(h)
	passed(t, progress, l, 1)
	passed(t, progress, l, 2)
}

// E03.1: a key bound twice at startup warns, and the warning is credited.
func TestTutorKeyClashEvent(t *testing.T) {
	h, progress, l := eventHarness(t, eventLesson("key_clash"), "# A01\n\ncuerpo\n", nil,
		app.WithKeys(map[string][]string{"toggle_explorer": {"space e"}}))
	h.ex("messages")
	if v := h.view(); !strings.Contains(v, "bound to both") {
		t.Fatalf("the clash should be in :messages:\n%s", v)
	}
	h.key(tea.KeyEsc)
	passed(t, progress, l, 1)
}
