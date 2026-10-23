package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/ui"
)

// userConfig is a config.json a user already has: settings the menu does
// not edit, and keys this version does not know, at the top level and
// inside "ui".
const userConfig = `{
  "theme": "onedark",
  "keys": {"toggle_explorer": ["space e"]},
  "notes_dir": "~/Notes",
  "future_top": {"x": [1, 2]},
  "ui": {"mode_icon": "󱗞", "future_ui": "keep me"}
}
`

// settingsHarness opens two notes in two folders, with the config at
// path, and leaves the editor in NORMAL mode.
func settingsHarness(t *testing.T, path string, opts ...app.Option) *harness {
	t.Helper()
	t.Cleanup(func() { ui.SetBorder("rounded") })
	store := newTestStore(t)
	// beta's note first: "1" on the home screen opens the latest, alpha/one.
	for _, n := range [][3]string{{"beta", "two", "two"}, {"alpha", "one", "# Head\n\n- item\n"}} {
		if _, err := store.SaveAs(n[0], n[1], n[2]); err != nil {
			t.Fatal(err)
		}
	}
	opts = append([]app.Option{app.WithConfigPath(path)}, opts...)
	h := newHarness(t, store, opts...)
	h.keys("1")
	h.key(tea.KeyEsc)
	return h
}

// explorer is the explorer column of the screen.
func (h *harness) explorer() string {
	h.t.Helper()
	// Each body row up to the explorer's right edge; the tab row and the
	// bars below are left blank, so the rows keep their screen y.
	lines := strings.Split(h.view(), "\n")
	var b strings.Builder
	for y, l := range lines {
		if y > 0 && y < len(lines)-2 {
			l, _, _ = strings.Cut(l, "│")
			b.WriteString(l)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// clickExplorer clicks text in the explorer column.
func (h *harness) clickExplorer(text string) {
	h.t.Helper()
	for y, l := range strings.Split(h.explorer(), "\n") {
		if i := strings.Index(l, text); i >= 0 {
			h.click(ansi.StringWidth(l[:i]), y)
			return
		}
	}
	h.t.Fatalf("%q not in the explorer:\n%s", text, h.explorer())
}

// openSettings types :settings in the editor.
func (h *harness) openSettings() {
	h.t.Helper()
	h.keys(":settings")
	h.key(tea.KeyEnter)
}

func tempConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if body != "" {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// configJSON reads the config file as generic JSON.
func configJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("%v:\n%s", err, data)
	}
	return v
}

// assertOnlyUIChanged checks that the file at path holds userConfig with
// only the given "ui" keys changed.
func assertOnlyUIChanged(t *testing.T, path string, ui map[string]any) {
	t.Helper()
	var want map[string]any
	if err := json.Unmarshal([]byte(userConfig), &want); err != nil {
		t.Fatal(err)
	}
	for k, v := range ui {
		want["ui"].(map[string]any)[k] = v
	}
	if got := configJSON(t, path); !reflect.DeepEqual(got, want) {
		t.Fatalf("config after the menu:\n got %v\nwant %v", got, want)
	}
}

func TestSettingsMenuListsTheFourOptions(t *testing.T) {
	h := settingsHarness(t, tempConfig(t, ""))
	h.openSettings()
	v := h.view()
	for _, want := range []string{"Settings", "Accordion folders", "on", "Explorer hide width", "120", "Icons & bullets", "Misoku", "Dialog border", "rounded"} {
		if !strings.Contains(v, want) {
			t.Fatalf("settings menu lacks %q:\n%s", want, v)
		}
	}
}

// Accordion off: opening a folder keeps its open sibling open, at once.
func TestSettingsAccordionAppliesLiveAndIsSaved(t *testing.T) {
	path := tempConfig(t, userConfig)
	h := settingsHarness(t, path)
	h.openSettings()
	h.keys("1")
	if v := h.view(); !strings.Contains(v, "Accordion folders") || !strings.Contains(v, "off") {
		t.Fatalf("the menu should stay open showing off:\n%s", v)
	}
	h.key(tea.KeyEsc)
	h.clickExplorer("beta") // alpha is open (its note is): beta opens next to it
	if v := h.explorer(); !strings.Contains(v, "one") || !strings.Contains(v, "two") {
		t.Fatalf("with accordion off both folders stay open:\n%s", v)
	}
	assertOnlyUIChanged(t, path, map[string]any{"accordion": false})
}

// The same click with accordion on (the default) closes alpha: the test
// above is about the menu, not the tree.
func TestAccordionOnClosesTheSibling(t *testing.T) {
	h := settingsHarness(t, tempConfig(t, ""))
	h.clickExplorer("beta")
	if v := h.explorer(); strings.Contains(v, "one") || !strings.Contains(v, "two") {
		t.Fatalf("with accordion on, opening beta closes alpha:\n%s", v)
	}
}

// accordion: false in the config is applied at start and shown in the menu.
func TestAccordionFromTheConfigIsApplied(t *testing.T) {
	off := false
	h := settingsHarness(t, tempConfig(t, ""), app.WithUI(config.UI{Accordion: &off, ExplorerHideWidth: 90}))
	h.clickExplorer("beta")
	if v := h.explorer(); !strings.Contains(v, "one") || !strings.Contains(v, "two") {
		t.Fatalf("accordion: false should keep alpha open:\n%s", v)
	}
	if w := h.m.(app.Model).ExplorerHideWidth(); w != 90 {
		t.Fatalf("explorer hide width = %d, want 90", w)
	}
}

// typeWidth replaces the prompt's value with s and presses Enter.
func (h *harness) typeWidth(s string) {
	h.t.Helper()
	h.key(tea.KeyCtrlU)
	h.keys(s)
	h.key(tea.KeyEnter)
}

func TestSettingsExplorerHideWidthAppliesLiveAndIsSaved(t *testing.T) {
	path := tempConfig(t, userConfig)
	h := settingsHarness(t, path)
	h.openSettings()
	h.keys("2")
	if v := h.view(); !strings.Contains(v, "Explorer hide width") {
		t.Fatalf("no width prompt:\n%s", v)
	}
	h.typeWidth("90")
	if w := h.m.(app.Model).ExplorerHideWidth(); w != 90 {
		t.Fatalf("explorer hide width = %d, want 90", w)
	}
	if v := h.view(); !strings.Contains(v, "90") || !strings.Contains(v, "Accordion folders") {
		t.Fatalf("back in the menu, showing 90:\n%s", v)
	}
	assertOnlyUIChanged(t, path, map[string]any{"explorer_hide_width": float64(90)})
}

func TestSettingsRejectsABadWidth(t *testing.T) {
	path := tempConfig(t, userConfig)
	h := settingsHarness(t, path)
	h.openSettings()
	h.keys("2")
	h.typeWidth("wide")
	if v := h.view(); !strings.Contains(v, "whole number") {
		t.Fatalf("a bad width should say why:\n%s", v)
	}
	if w := h.m.(app.Model).ExplorerHideWidth(); w != config.DefaultExplorerHideWidth {
		t.Fatalf("width changed to %d", w)
	}
	assertOnlyUIChanged(t, path, nil)
}

func TestSettingsIconsAndBulletsApplyLiveAndAreSaved(t *testing.T) {
	path := tempConfig(t, userConfig)
	h := settingsHarness(t, path)
	h.openSettings()
	h.keys("3")
	if v := h.view(); !strings.Contains(v, "Numbers") {
		t.Fatalf("no preset list:\n%s", v)
	}
	h.keys("2") // Numbers
	h.key(tea.KeyEsc)
	if p := h.preview(); !strings.Contains(p, "① Head") || !strings.Contains(p, "• item") {
		t.Fatalf("the preview should use the Numbers preset:\n%s", p)
	}
	assertOnlyUIChanged(t, path, map[string]any{
		"heading_icons": []any{"① ", "② ", "③ ", "④ ", "⑤ ", "⑥ "},
		"bullets":       []any{"•", "◦", "▪", "▫"},
	})
	// Back to Misoku's own look removes both keys.
	h.openSettings()
	h.keys("31")
	assertOnlyUIChanged(t, path, nil)
}

func TestSettingsBorderAppliesLiveAndIsSaved(t *testing.T) {
	path := tempConfig(t, userConfig)
	h := settingsHarness(t, path)
	h.openSettings()
	h.keys("4")
	h.keys("2") // single
	if v := h.view(); !strings.Contains(v, "┌") || strings.Contains(v, "╭") {
		t.Fatalf("the menu itself should now have square corners:\n%s", v)
	}
	assertOnlyUIChanged(t, path, map[string]any{"border": "single"})
}

// Choosing the value already in use writes nothing: no file is created,
// and an existing one is not rewritten.
func TestSettingsUnchangedValueWritesNothing(t *testing.T) {
	missing := tempConfig(t, "")
	h := settingsHarness(t, missing)
	h.openSettings()
	h.keys("41") // border: rounded, already in use
	h.keys("3")
	h.keys("1") // icons: Misoku, already in use
	h.keys("2")
	h.typeWidth("120")
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("an unchanged value created the config: %v", err)
	}

	path := tempConfig(t, `{"ui": {"border": "single"}}`)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	h = settingsHarness(t, path, app.WithUI(config.UI{Border: "single"}))
	h.openSettings()
	h.keys("42") // single, already in use
	if info, _ := os.Stat(path); !info.ModTime().Equal(old) {
		t.Fatalf("an unchanged value rewrote the config")
	}
}

// A config the menu cannot write: the change still applies, and the error
// is on screen.
func TestSettingsSaveErrorIsShown(t *testing.T) {
	h := settingsHarness(t, unreadableConfig(t))
	h.openSettings()
	h.keys("1")
	if v := h.view(); !strings.Contains(v, "Could not save settings") {
		t.Fatalf("no message after a failed save:\n%s", v)
	}
}

// Esc goes back without saving: a typed width that was not confirmed with
// Enter, and a sub-list closed without a pick, change nothing.
func TestSettingsEscSavesNothing(t *testing.T) {
	path := tempConfig(t, "")
	h := settingsHarness(t, path)
	h.openSettings()
	h.keys("2")
	h.key(tea.KeyCtrlU)
	h.keys("90")
	h.key(tea.KeyEsc)
	if v := h.view(); !strings.Contains(v, "Accordion folders") {
		t.Fatalf("Esc in the prompt should go back to the menu:\n%s", v)
	}
	h.keys("4")
	h.key(tea.KeyEsc) // border list, nothing picked
	h.key(tea.KeyEsc) // the menu
	if v := h.view(); strings.Contains(v, "Accordion folders") {
		t.Fatalf("Esc should close the menu:\n%s", v)
	}
	if w := h.m.(app.Model).ExplorerHideWidth(); w != config.DefaultExplorerHideWidth {
		t.Fatalf("width changed to %d", w)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Esc wrote the config: %v", err)
	}
}

// Two quick changes of one setting save in the background; when the older
// save runs last it must not undo the newer value.
func TestSettingsOlderSaveDoesNotUndoANewerOne(t *testing.T) {
	path := tempConfig(t, userConfig)
	h := settingsHarness(t, path)
	h.openSettings()
	one := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
	var off, on tea.Cmd
	h.m, off = h.m.Update(one) // accordion off
	h.m, on = h.m.Update(one)  // and on again
	runAll(t, on)
	runAll(t, off)
	assertOnlyUIChanged(t, path, map[string]any{"accordion": true})
}
