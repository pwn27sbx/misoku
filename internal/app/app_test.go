package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const (
	termW = 140
	termH = 40
)

// realHome is the HOME the tests started with, before any t.Setenv.
var realHome, _ = os.UserHomeDir()

func newTestStore(t *testing.T) *core.Store {
	t.Helper()
	// Deleting moves notes to the Trash: keep the tests away from the real
	// one, and from the one TestMain shares. A test that already set its own
	// HOME keeps it.
	if h, _ := os.UserHomeDir(); h == realHome || h == testHome {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	}
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error creating store: %v", err)
	}
	return store
}

// harness drives the model like the Bubble Tea runtime: every returned
// command is executed and its message fed back in. Timer-based commands
// (cursor blinks, toasts, the workspace poll) are dropped; see runCmd.
type harness struct {
	t     *testing.T
	m     tea.Model
	quit  bool
	store *core.Store
}

func newHarness(t *testing.T, store *core.Store, opts ...app.Option) *harness {
	h := &harness{t: t, m: app.New(store, opts...), store: store}
	h.send(tea.WindowSizeMsg{Width: termW, Height: termH})
	return h
}

// runCmd runs a command and returns its message. Commands that only wait
// for a clock (cursor blinks, toasts, the workspace poll) are not run: a
// test never waits for them. Every other command is waited for, however
// slow the machine is; a fixed short wait used to drop a click's message on
// a loaded CI runner and leave the previous note open.
func runCmd(c tea.Cmd) tea.Msg {
	if isTimer(c) {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- c() }()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(cmdLimit):
		return nil
	}
}

// cmdLimit bounds a command that never returns, so a stuck test still ends.
const cmdLimit = 10 * time.Second

// timerFuncs are the function names of the commands that wait for a clock.
var timerFuncs = []string{
	"github.com/charmbracelet/bubbletea.Tick.",
	"github.com/charmbracelet/bubbletea.Every.",
	"github.com/charmbracelet/bubbles/cursor.(*Model).BlinkCmd.",
}

// isTimer reports whether c is one of the clock commands above.
func isTimer(c tea.Cmd) bool {
	fn := runtime.FuncForPC(reflect.ValueOf(c).Pointer())
	if fn == nil {
		return false
	}
	for _, prefix := range timerFuncs {
		if strings.HasPrefix(fn.Name(), prefix) {
			return true
		}
	}
	return false
}

func (h *harness) send(msgs ...tea.Msg) {
	h.t.Helper()
	for _, msg := range msgs {
		var cmd tea.Cmd
		h.m, cmd = h.m.Update(msg)
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			out := runCmd(c)
			switch out := out.(type) {
			case nil:
			case tea.BatchMsg:
				queue = append(queue, out...)
			case tea.QuitMsg:
				h.quit = true
			default:
				if name := fmt.Sprintf("%T", out); strings.Contains(name, "cursor.") || strings.Contains(name, "Blink") {
					continue
				}
				var next tea.Cmd
				h.m, next = h.m.Update(out)
				queue = append(queue, next)
			}
		}
	}
}

func (h *harness) keys(s string) {
	h.t.Helper()
	for _, r := range s {
		h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func (h *harness) key(t tea.KeyType) { h.t.Helper(); h.send(tea.KeyMsg{Type: t}) }

func (h *harness) click(x, y int) {
	h.t.Helper()
	h.send(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
}

func (h *harness) view() string {
	h.t.Helper()
	v := h.m.View()
	assertFrame(h.t, v)
	return ansi.Strip(v)
}

// assertFrame checks the frame is exactly termW × termH cells, which is what
// keeps pane dividers straight.
func assertFrame(t *testing.T, v string) {
	t.Helper()
	lines := strings.Split(v, "\n")
	if len(lines) != termH {
		t.Fatalf("frame has %d rows, want %d", len(lines), termH)
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w != termW {
			t.Fatalf("row %d is %d cells wide, want %d: %q", i, w, termW, ansi.Strip(l))
		}
	}
}

// find returns the screen position of the first occurrence of text.
func stripped(s string) string { return ansi.Strip(s) }

func find(t *testing.T, view, text string) (int, int) {
	t.Helper()
	for y, l := range strings.Split(view, "\n") {
		if i := strings.Index(l, text); i >= 0 {
			return ansi.StringWidth(l[:i]), y
		}
	}
	t.Fatalf("%q not on screen:\n%s", text, view)
	return 0, 0
}

func TestLeaderKeysMoveFocusPanesAndTogglePreview(t *testing.T) {
	store := newTestStore(t)
	h := newHarness(t, store)
	h.keys(" ") // leader pending
	h.keys("e") // focus sidebar
	h.keys(" ")
	h.keys("p") // toggle preview
	h.keys(" ")
	h.keys("d")  // go home
	h.keys("\t") // move pane
	h.keys("\t")
	h.keys("\x1b") // esc back to editor if possible
	h.view()
}

func TestDashboardQuickActions(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Create("hello", ""); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.view()
	// Dashboard listens for digits / enter on "New Note" etc.; just make sure
	// browsing doesn't panic.
	h.key(tea.KeyDown)
	h.key(tea.KeyEnter)
	h.view()
}

func TestSaveAsDialogTabAndWindowStart(t *testing.T) {
	store := newTestStore(t)
	h := newHarness(t, store)
	h.keys("n")
	h.key(tea.KeyEsc)
	h.keys(":w")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Save As") {
		t.Fatalf("expected Save As dialog:\n%s", v)
	}
	h.key(tea.KeyTab)      // move to the folder field / focusField
	h.key(tea.KeyShiftTab) // move back
	h.key(tea.KeyCtrlN)    // exercise dialog update paths
	h.key(tea.KeyEsc)
	h.view()
}

func TestVimQuitAndSaveMessages(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Create("body", ""); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.send(core.PageSelectedMsg{ID: "body.md"})
	h.view()
	h.send(core.VimQuitMsg{Force: false})
	if h.quit {
		t.Log("quitted immediately")
	}
	h.send(core.VimQuitMsg{Force: true})
	if !h.quit {
		t.Error("expected :q! to quit")
	}
}

func TestDashboardAndNewNoteSaveAs(t *testing.T) {
	store := newTestStore(t)
	if err := os.MkdirAll(filepath.Join(store.Root(), "journal"), 0755); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)

	v := h.view()
	for _, want := range []string{"New Note", "Find Note", "HOME"} {
		if !strings.Contains(v, want) {
			t.Fatalf("dashboard missing %q:\n%s", want, v)
		}
	}

	// n opens an in-memory Untitled-1 tab, straight into INSERT mode.
	h.keys("n")
	v = h.view()
	if !strings.Contains(v, "Untitled-1") || !strings.Contains(v, "INSERT") {
		t.Fatalf("expected Untitled-1 tab in INSERT mode:\n%s", v)
	}
	if pages := store.List(); len(pages) != 0 { // empty journal folder is hidden
		t.Fatalf("nothing may be written before saving, got %v", pages)
	}

	h.keys("hello")
	h.key(tea.KeyEsc)
	h.keys(":w")
	h.key(tea.KeyEnter)
	v = h.view()
	if !strings.Contains(v, "Save As") || !strings.Contains(v, "workspace root") {
		t.Fatalf("expected the Save As dialog:\n%s", v)
	}

	// Pick the journal folder in the dialog with the mouse, rename, save.
	_, top := find(t, v, "Folder")
	x, y := find(t, strings.Join(strings.Split(v, "\n")[top:], "\n"), "journal")
	h.click(x, y+top)
	for range "Untitled-1" {
		h.key(tea.KeyBackspace)
	}
	h.keys("day one.txt")
	if v = h.view(); !strings.Contains(v, "journal/day one.txt.md") {
		t.Fatalf("expected forced .md target path in dialog:\n%s", v)
	}
	h.key(tea.KeyEnter)

	got, err := store.Get("journal/day one.txt.md")
	if err != nil || got.Content != "hello" {
		t.Fatalf("expected saved note, got %+v, %v", got, err)
	}
	v = h.view()
	if strings.Contains(v, "Save As") || !strings.Contains(v, "day one.txt.md") {
		t.Fatalf("dialog should close and the tab show the file name:\n%s", v)
	}
}

func TestUnsavedEditsSurviveTabSwitchAndQuitAsks(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"alpha", "beta"} {
		if _, err := store.SaveAs("", n, n+" body"); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e") // explorer
	h.key(tea.KeyEnter)
	h.keys("A!") // append in alpha
	h.key(tea.KeyEsc)
	h.keys("[") // back to the explorer? no: [ cycles tabs; only one tab yet

	// Open beta from the explorer via Ctrl+P search.
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("beta")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "beta.md") || !strings.Contains(v, "alpha.md") {
		t.Fatalf("expected two tabs:\n%s", v)
	}

	h.keys("[")
	v := h.view()
	if !strings.Contains(v, "alpha body!") {
		t.Fatalf("unsaved edit lost after switching tabs:\n%s", v)
	}
	if got, _ := store.Get("alpha.md"); got.Content != "alpha body" {
		t.Fatalf("unsaved edit must not touch disk, got %q", got.Content)
	}

	h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	if h.quit {
		t.Fatal("quit with unsaved changes must ask first")
	}
	if v = h.view(); !strings.Contains(v, "Save All") {
		t.Fatalf("expected quit confirmation:\n%s", v)
	}
	h.keys("s") // Save All
	if !h.quit {
		t.Fatal("expected quit after Save All")
	}
	if got, _ := store.Get("alpha.md"); got.Content != "alpha body!" {
		t.Fatalf("Save All did not write alpha, got %q", got.Content)
	}
}

func TestCloseTabAndCtrlS(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "x"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1") // open the recent note
	h.keys("A2")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got, _ := store.Get("doc.md"); got.Content != "x2" {
		t.Fatalf("Ctrl+S in INSERT mode should save, got %q", got.Content)
	}
	h.key(tea.KeyEsc)
	h.keys("A3")
	h.key(tea.KeyEsc)
	h.keys(" x") // leader x closes the tab
	if v := h.view(); !strings.Contains(v, "Don't Save") {
		t.Fatalf("closing a dirty tab must ask:\n%s", v)
	}
	h.keys("d")
	v := h.view()
	if strings.Contains(v, "doc.md  ") && strings.Contains(v, "●") {
		t.Fatalf("tab should be closed:\n%s", v)
	}
	if got, _ := store.Get("doc.md"); got.Content != "x2" {
		t.Fatalf("Don't Save must keep disk content, got %q", got.Content)
	}
}

func TestMouseTabsAndPanes(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"one", "two"} {
		if _, err := store.SaveAs("", n, n); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("e")
	v := h.view()
	x, y := find(t, v, "one.md")
	h.click(x, y)
	v = h.view()
	x, y = find(t, v, "two.md")
	h.click(x, y)
	v = h.view()

	// Clicking a tab switches to it.
	x, _ = find(t, strings.Split(v, "\n")[0], "one.md")
	h.click(x, 0)
	if v = h.view(); !strings.Contains(strings.Split(v, "\n")[1], "1 one") {
		t.Fatalf("expected one.md active after clicking its tab:\n%s", v)
	}

	// The "+" button opens a new draft.
	x, _ = find(t, strings.Split(v, "\n")[0], "\uf067")
	h.click(x+1, 0)
	if v = h.view(); !strings.Contains(v, "Untitled-1") {
		t.Fatalf("expected new draft from + button:\n%s", v)
	}

	// Wheel over the explorer must not steal focus or crash.
	h.send(tea.MouseMsg{X: 3, Y: 5, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	h.view()
}

func TestRenameKeepsContentAndDeleteAsks(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "Only Page", "precious"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("e")
	h.keys("r")
	for range "Only Page" {
		h.key(tea.KeyBackspace)
	}
	h.keys("Renamed")
	h.key(tea.KeyEnter)
	got, err := store.Get("Renamed.md")
	if err != nil || got.Content != "precious" {
		t.Fatalf("rename must keep content, got %+v %v", got, err)
	}

	h.keys("d")
	if len(store.List()) != 1 {
		t.Fatal("delete must wait for confirmation")
	}
	if v := h.view(); !strings.Contains(v, "Delete Renamed.md?") {
		t.Fatalf("expected delete confirmation:\n%s", v)
	}
	h.key(tea.KeyEnter)
	if len(store.List()) != 0 {
		t.Fatalf("expected file deleted, got %v", store.List())
	}
	if v := h.view(); !strings.Contains(v, "New Note") {
		t.Fatalf("empty workspace should return home:\n%s", v)
	}
}

func TestLongFileLoadsCompletely(t *testing.T) {
	store := newTestStore(t)
	var b strings.Builder
	for i := 1; i <= 300; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	if _, err := store.SaveAs("", "long", b.String()); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.keys("G")
	v := h.view()
	if !strings.Contains(v, "line 300") {
		t.Fatalf("expected the end of a 300-line file to be reachable:\n%s", v)
	}
	// Mouse wheel scrolls the editor back up.
	for i := 0; i < 120; i++ {
		h.send(tea.MouseMsg{X: 60, Y: 10, Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	}
	if v = h.view(); !strings.Contains(v, "line 1\n") && !strings.Contains(v, "line 1 ") {
		t.Fatalf("expected wheel to scroll to the top:\n%s", v)
	}
}

func TestKeymapModalAndSmallScreens(t *testing.T) {
	h := newHarness(t, newTestStore(t))
	h.keys("?")
	if v := h.view(); !strings.Contains(v, "MISOKU KEYMAPS") {
		t.Fatalf("expected keymap modal:\n%s", v)
	}
	h.key(tea.KeyEsc)
	if v := h.view(); strings.Contains(v, "MISOKU KEYMAPS") {
		t.Fatal("modal should close on Esc")
	}

	for _, size := range [][2]int{{60, 12}, {80, 24}, {200, 60}} {
		m, _ := app.New(newTestStore(t)).Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		lines := strings.Split(m.View(), "\n")
		if len(lines) != size[1] {
			t.Errorf("%v: %d rows", size, len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w != size[0] {
				t.Errorf("%v: row %d width %d", size, i, w)
				break
			}
		}
	}
}

func TestFinderArrowKeysAndFilter(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"alpha", "beta", "gamma"} {
		if _, err := store.SaveAs("notes", n, n+" text"); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("f")
	h.key(tea.KeyDown) // move selection (covers finder.move)
	h.key(tea.KeyUp)
	h.key(tea.KeyCtrlN) // vim-style down in the finder
	h.keys("gam")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(strings.Split(v, "\n")[0], "gamma.md") {
		t.Fatalf("expected gamma.md open:\n%s", v)
	}
}

func TestFinderFromHomeOpensInCurrentBuffer(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"alpha", "beta", "gamma"} {
		if _, err := store.SaveAs("notes", n, n+" text"); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)

	// Find from the home screen is a modal over the home screen.
	h.keys("f")
	v := h.view()
	if !strings.Contains(v, "Find Note") || !strings.Contains(v, "HOME") {
		t.Fatalf("expected finder over the home screen:\n%s", v)
	}
	h.keys("bta") // fuzzy: b-e-t-a
	h.key(tea.KeyEnter)
	v = h.view()
	if !strings.Contains(strings.Split(v, "\n")[0], "beta.md") || strings.Contains(v, "Find Note") {
		t.Fatalf("expected beta.md open and the finder closed:\n%s", v)
	}

	// Ctrl+P from the editor replaces the (clean) current tab.
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("gamma")
	h.key(tea.KeyEnter)
	tabs := strings.Split(h.view(), "\n")[0]
	if !strings.Contains(tabs, "gamma.md") || strings.Contains(tabs, "beta.md") {
		t.Fatalf("expected gamma.md to replace beta.md in the same tab: %q", tabs)
	}

	// A dirty tab is never replaced.
	h.keys("A!")
	h.key(tea.KeyEsc)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("alpha")
	h.key(tea.KeyEnter)
	tabs = strings.Split(h.view(), "\n")[0]
	if !strings.Contains(tabs, "gamma.md") || !strings.Contains(tabs, "alpha.md") {
		t.Fatalf("expected dirty gamma.md kept next to alpha.md: %q", tabs)
	}

	// Esc closes the finder without changing anything.
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.key(tea.KeyEsc)
	if strings.Contains(h.view(), "Find Note") {
		t.Fatal("Esc should close the finder")
	}
}

func TestCtrlBFocusesExplorerFromInsertMode(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"a", "b"} {
		if _, err := store.SaveAs("", n, n); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.keys("n")                            // draft in INSERT mode
	h.send(tea.KeyMsg{Type: tea.KeyCtrlB}) // explorer visible: hides it
	h.send(tea.KeyMsg{Type: tea.KeyCtrlB}) // hidden: shows it with the focus
	if v := h.view(); !strings.Contains(v, "EXPLORER") || strings.Contains(v, "INSERT") {
		t.Fatalf("expected explorer focus after Ctrl+B:\n%s", v)
	}
	h.keys("j")
	h.key(tea.KeyEnter)
	if tabs := strings.Split(h.view(), "\n")[0]; !strings.Contains(tabs, "b.md") {
		t.Fatalf("expected tree keys to work after Ctrl+B: %q", tabs)
	}
}

func TestThemePickerPreviewRevertAndSave(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	h := newHarness(t, newTestStore(t), app.WithConfigPath(cfgPath))

	h.keys("t")
	if v := h.view(); !strings.Contains(v, "Themes") || !strings.Contains(v, "onedark") {
		t.Fatalf("expected theme picker:\n%s", v)
	}
	h.keys("gruvbox_light")
	h.key(tea.KeyEsc) // revert
	if _, err := os.Stat(cfgPath); err == nil {
		t.Fatal("Esc must not save a theme")
	}

	h.keys("t")
	h.keys("gruvbox_light")
	h.key(tea.KeyEnter)
	data, err := os.ReadFile(cfgPath)
	if err != nil || !strings.Contains(string(data), "gruvbox_light") {
		t.Fatalf("expected theme saved, got %q %v", data, err)
	}

	// :colorscheme from the editor.
	h.keys("n")
	h.key(tea.KeyEsc)
	h.keys(":colorscheme nord")
	h.key(tea.KeyEnter)
	if data, _ := os.ReadFile(cfgPath); !strings.Contains(string(data), "nord") {
		t.Fatalf("expected :colorscheme to save, got %q", data)
	}
	h.keys(":colo nosuch")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Cannot find color scheme") {
		t.Fatalf("expected error for unknown theme:\n%s", v)
	}
}

func TestEveryThemeRendersFullFrames(t *testing.T) {
	for _, name := range theme.Names() {
		h := newHarness(t, newTestStore(t), app.WithTheme(name))
		h.view()
		h.keys("n")
		h.view()
	}
}

func TestFinderSearchesTextAcrossAllFolders(t *testing.T) {
	store := newTestStore(t)
	body := "# Trip\n\nline two\npack the passport\n"
	if _, err := store.SaveAs("travel/2026/japan", "itinerary", body); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "inbox", "nothing here"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("f")
	h.keys("passport")
	v := h.view()
	if !strings.Contains(v, "travel/2026/japan/itinerary.md:4") {
		t.Fatalf("expected a text hit with path and line from a nested folder:\n%s", v)
	}
	h.key(tea.KeyEnter)
	v = h.view()
	if !strings.Contains(strings.Split(v, "\n")[0], "itinerary.md") || !strings.Contains(v, "Ln 4, Col 1") {
		t.Fatalf("expected itinerary.md open with the cursor on line 4:\n%s", v)
	}
}

func TestSlashLinkToPage(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("projects", "Roadmap Q4", "plan"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("journal", "today", ""); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\\'}}) // backslash = find
	h.keys("today")
	h.key(tea.KeyEnter)
	h.keys("i/link")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Link to Note") {
		t.Fatalf("expected link picker:\n%s", v)
	}
	h.keys("roadmap")
	h.key(tea.KeyEnter)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	got, _ := store.Get("journal/today.md")
	if got.Content != "[Roadmap Q4](<../projects/Roadmap Q4.md>)" {
		t.Fatalf("unexpected link: %q", got.Content)
	}
}

func TestTabIndentsAndLeaderTabSwitchesTabs(t *testing.T) {
	store := newTestStore(t)
	for _, n := range []string{"one", "two"} {
		if _, err := store.SaveAs("", n, n); err != nil {
			t.Fatal(err)
		}
	}
	h := newHarness(t, store)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("one")
	h.key(tea.KeyEnter)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlP})
	h.keys("two")
	h.send(tea.KeyMsg{Type: tea.KeyCtrlT})
	activeTab := func() string {
		row := strings.Split(h.view(), "\n")[0]
		return row[strings.Index(row, "▎"):][:24]
	}
	if !strings.Contains(activeTab(), "two") {
		t.Fatalf("expected two.md active, got %q", activeTab())
	}
	h.keys(" ")
	h.key(tea.KeyTab)
	if !strings.Contains(activeTab(), "one") {
		t.Fatalf("Space Tab should switch to the next tab, got %q", activeTab())
	}

	h.keys("I")
	h.key(tea.KeyTab)
	if v := h.view(); !strings.Contains(v, "INSERT") {
		t.Fatalf("Tab in INSERT mode must not leave the editor:\n%s", v)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got, _ := store.Get("one.md"); got.Content != "  one" {
		t.Fatalf("Tab in INSERT mode should indent, got %q", got.Content)
	}
}

func TestSlashImageUsesFileBrowserAndCopiesIntoAssets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
	t.Setenv("MISOKU_NATIVE_PICKER", "0")
	dl := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(filepath.Join(dl, "trips"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{"photo.png": "png", "notes.txt": "txt", "trips/beach.jpg": "jpg"} {
		if err := os.WriteFile(filepath.Join(dl, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store := newTestStore(t)
	if _, err := store.SaveAs("journal", "day", ""); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.keys("i/image")
	h.key(tea.KeyEnter)
	v := h.view()
	if !strings.Contains(v, "Choose an image") || !strings.Contains(v, "photo.png") || strings.Contains(v, "notes.txt") {
		t.Fatalf("expected image browser in ~/Downloads showing only images and folders:\n%s", v)
	}

	// Into a sub-folder and pick the image there.
	h.keys("trips")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEnter)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	got, _ := store.Get("journal/day.md")
	if got.Content != "![beach](assets/beach.jpg)" {
		t.Fatalf("unexpected note content %q", got.Content)
	}
	if data, err := os.ReadFile(filepath.Join(store.Root(), "journal/assets/beach.jpg")); err != nil || string(data) != "jpg" {
		t.Fatalf("expected image copied into journal/assets, err %v", err)
	}

	// Any file via /file, chosen by typing a path.
	h.keys(" /file")
	h.key(tea.KeyEnter)
	h.keys("~/Downloads/notes.txt")
	h.key(tea.KeyEnter)
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got, _ := store.Get("journal/day.md"); !strings.Contains(got.Content, "(assets/notes.txt)") {
		t.Fatalf("expected file link, got %q", got.Content)
	}
}

func TestCopyByDragYankAndVisualMode(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "hello world\nsecond line\nthird"); err != nil {
		t.Fatal(err)
	}
	var copied []string
	h := newHarness(t, store, app.WithClipboard(func(s string) error { copied = append(copied, s); return nil }))
	h.keys("1")

	// yy copies the line and shows a toast.
	h.keys("yy")
	if len(copied) != 1 || copied[0] != "hello world\n" {
		t.Fatalf("yy copied %q", copied)
	}
	if v := h.view(); !strings.Contains(v, "Copied 12 characters") {
		t.Fatalf("expected a copy toast:\n%s", v)
	}

	// V j y copies two whole lines.
	h.keys("Vjy")
	if got := copied[len(copied)-1]; got != "hello world\nsecond line" {
		t.Fatalf("V-line copy got %q", got)
	}

	// Mouse drag across "world" auto-copies on release.
	v := h.view()
	x, y := find(t, v, "world")
	h.send(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	h.send(tea.MouseMsg{X: x + 4, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion})
	h.send(tea.MouseMsg{X: x + 4, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	if got := copied[len(copied)-1]; got != "world" {
		t.Fatalf("drag copy got %q", got)
	}

	// v + d cuts (and copies) the selection.
	h.keys("0vlld")
	if got, _ := store.Get("doc.md"); got.Content != "hello world\nsecond line\nthird" {
		t.Fatalf("cut must not save by itself, disk: %q", got.Content)
	}
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	if got, _ := store.Get("doc.md"); !strings.HasPrefix(got.Content, "lo world") {
		t.Fatalf("expected 'hel' cut, got %q", got.Content)
	}
}

func TestSlashCoverSetsFrontMatter(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
	t.Setenv("MISOKU_NATIVE_PICKER", "0")
	_ = os.MkdirAll(filepath.Join(home, "Downloads"), 0o755)
	if err := os.WriteFile(filepath.Join(home, "Downloads", "gcp.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := newTestStore(t)
	if _, err := store.SaveAs("", "ace", "# GCP- ACE"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.keys("o/cover")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEnter) // pick gcp.png
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	got, _ := store.Get("ace.md")
	if !strings.HasPrefix(got.Content, "---\ncover: assets/gcp.png\n---\n# GCP- ACE") {
		t.Fatalf("expected cover front matter, got %q", got.Content)
	}
}

// openWith opens a single note with the given text, preview visible.
func openWith(t *testing.T, text string) *harness {
	t.Helper()
	store := newTestStore(t)
	if _, err := store.SaveAs("", "views", text); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store)
	h.keys("1")
	h.key(tea.KeyEsc)
	return h
}

// clickText clicks the first occurrence of text in the right half (preview).
func (h *harness) clickText(text string) {
	h.t.Helper()
	for y, l := range strings.Split(h.view(), "\n") {
		half := []rune(l)
		if len(half) < termW/2 {
			continue
		}
		right := string(half[termW/2:])
		if i := strings.Index(right, text); i >= 0 {
			h.click(termW/2+ansi.StringWidth(right[:i]), y)
			return
		}
	}
	h.t.Fatalf("%q not in preview:\n%s", text, h.view())
}

// clickLeftmost clicks the occurrence of text furthest left in the preview.
func (h *harness) clickLeftmost(text string) {
	h.t.Helper()
	bx, by := -1, -1
	for y, l := range strings.Split(h.view(), "\n") {
		r := []rune(l)
		if len(r) < termW/2 {
			continue
		}
		right := string(r[termW/2:])
		if i := strings.Index(right, text); i >= 0 {
			if x := termW/2 + ansi.StringWidth(right[:i]); bx < 0 || x < bx {
				bx, by = x, y
			}
		}
	}
	if bx < 0 {
		h.t.Fatalf("%q not in preview", text)
	}
	h.click(bx, by)
}

func (h *harness) text() string {
	h.t.Helper()
	h.send(tea.KeyMsg{Type: tea.KeyCtrlS})
	p, err := h.store.Get("views.md")
	if err != nil {
		h.t.Fatal(err)
	}
	return p.Content
}

func TestInteractiveBoard(t *testing.T) {
	h := openWith(t, "```board\n## Todo\n- Write\n  details\n## Done\n- Ship\n```")
	h.clickLeftmost("+ New page")
	h.keys("Review")
	h.key(tea.KeyEnter)
	if got := h.text(); !strings.Contains(got, "- Write\n  details\n- Review\n## Done") {
		t.Fatalf("new card should go at the end of Todo:\n%s", got)
	}
	h.clickText("Write")
	if v := h.view(); !strings.Contains(v, "Move to → Done") {
		t.Fatalf("expected card menu:\n%s", v)
	}
	h.keys("2") // Move to → Done
	if got := h.text(); !strings.Contains(got, "## Done\n- Ship\n- Write\n  details") {
		t.Fatalf("card (with description) should move to Done:\n%s", got)
	}
}

func TestInteractiveFormCalendarTimelineChart(t *testing.T) {
	h := openWith(t, "```form\ntitle: Survey\n? Name\n? Pick (choice): A | B\n? Rate (rating)\n```")
	h.clickText("Respondent's answer")
	h.keys("Jai")
	h.key(tea.KeyEnter)
	h.clickText("○ B")
	if got := h.text(); !strings.Contains(got, "? Name\n  = Jai\n? Pick (choice): A | B\n  = B") {
		t.Fatalf("form answers not stored:\n%s", got)
	}
	h.clickText("● B") // clicking again clears it
	if got := h.text(); strings.Contains(got, "= B") {
		t.Fatalf("second click should clear the choice:\n%s", got)
	}

	h = openWith(t, "```calendar\nmonth: 2026-10\n2026-10-15: Launch\n```")
	h.clickText(" 20 ")
	h.keys("Party")
	h.key(tea.KeyEnter)
	if got := h.text(); !strings.Contains(got, "2026-10-20: Party") {
		t.Fatalf("clicking a day should add an event:\n%s", got)
	}

	h = openWith(t, "```timeline\nAlpha: 2026-10-01 -> 2026-10-04\n```")
	h.clickText("Alpha")
	h.keys("5") // Extend by 1 day
	if got := h.text(); !strings.Contains(got, "Alpha: 2026-10-01 -> 2026-10-05") {
		t.Fatalf("extend should move the end date:\n%s", got)
	}

	h = openWith(t, "```chart\ntype: hbar\nGo: 5\nRust: 2\n```")
	h.clickText("Rust")
	h.keys("1")
	for range "2" {
		h.key(tea.KeyBackspace)
	}
	h.keys("9")
	h.key(tea.KeyEnter)
	if got := h.text(); !strings.Contains(got, "Rust: 9") {
		t.Fatalf("editing a chart value should rewrite its line:\n%s", got)
	}
}

func TestEmojiVariationSelectorsNeverReachTheScreen(t *testing.T) {
	h := openWith(t, "icon: ☁️ and 👨‍👩‍👧 family\n\n☁️ cloud")
	if v := h.m.View(); strings.ContainsAny(v, "️‍") {
		t.Fatal("variation selectors / joiners must be stripped so rows keep their width")
	}
}

func TestAddChoiceQuestionAndChangeType(t *testing.T) {
	h := openWith(t, "```form\ntitle: Survey\n? Name\n```")
	h.clickText("add question")
	h.keys("Favourite colour")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Single choice") || !strings.Contains(v, "Rating") {
		t.Fatalf("expected the question type menu:\n%s", v)
	}
	h.keys("3") // Single choice
	for range "Option 1 | Option 2 | Option 3" {
		h.key(tea.KeyBackspace)
	}
	h.keys("Red, Green | Blue")
	h.key(tea.KeyEnter)
	if got := h.text(); !strings.Contains(got, "? Favourite colour (choice): Red | Green | Blue") {
		t.Fatalf("choice question not added:\n%s", got)
	}

	// Change "Name" into a rating question.
	h.clickText("Name")
	h.keys("2") // Change type
	h.keys("5") // Rating
	if got := h.text(); !strings.Contains(got, "? Name (rating)") {
		t.Fatalf("type change not applied:\n%s", got)
	}
}

func TestTimelineNewItemPicksStartAndEndDates(t *testing.T) {
	h := openWith(t, "```timeline\nAlpha: 2026-10-01 -> 2026-10-04\n```")
	h.clickText("+ New")
	h.keys("Launch")
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Start date · Launch") {
		t.Fatalf("expected the start date picker:\n%s", v)
	}
	today := time.Now()
	h.keys("ll") // start = today + 2
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "End date · Launch") {
		t.Fatalf("expected the end date picker:\n%s", v)
	}
	h.keys("hh") // cannot go before the start
	h.keys("j")  // + 1 week
	h.key(tea.KeyEnter)
	start := today.AddDate(0, 0, 2).Format("2006-01-02")
	end := today.AddDate(0, 0, 9).Format("2006-01-02")
	if got := h.text(); !strings.Contains(got, "Launch: "+start+" -> "+end) {
		t.Fatalf("expected Launch %s -> %s:\n%s", start, end, got)
	}
}

func TestTableEditingFromPreview(t *testing.T) {
	h := openWith(t, "| Name | Role |\n| --- | --- |\n| Ada | Eng |")
	h.clickText("+ Add row")
	if got := h.text(); !strings.HasSuffix(got, "| Ada  | Eng  |\n|      |      |") {
		t.Fatalf("+ Add row should append an empty row:\n%s", got)
	}
	h.clickText("+ Add column")
	if got := h.text(); !strings.Contains(got, "| Name | Role | Column |") {
		t.Fatalf("+ Add column should add a column:\n%s", got)
	}

	h.clickText("Ada")
	if v := h.view(); !strings.Contains(v, "Delete row") {
		t.Fatalf("expected the cell menu:\n%s", v)
	}
	h.keys("1") // Edit cell
	for range "Ada" {
		h.key(tea.KeyBackspace)
	}
	h.keys("Grace")
	h.key(tea.KeyEnter)
	if got := h.text(); !strings.Contains(got, "| Grace | Eng  |") {
		t.Fatalf("cell edit not applied:\n%s", got)
	}

	h.clickText("Column")
	if v := h.view(); strings.Contains(v, "Delete row") || !strings.Contains(v, "Rename column") {
		t.Fatalf("header cells offer column actions only:\n%s", v)
	}
	h.keys("4") // Delete column
	if got := h.text(); strings.Contains(got, "Column") {
		t.Fatalf("column should be deleted:\n%s", got)
	}
}
