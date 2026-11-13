package app_test

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/tutor"
)

// useLang switches the interface language for one test and puts English
// back after it. Tests that call it must not run in parallel.
func useLang(t *testing.T, l i18n.Lang) {
	t.Helper()
	i18n.Set(l)
	t.Cleanup(func() { i18n.Set(i18n.EN) })
}

// useSystemLanguage makes "Auto (system)" resolve to l for one test.
func useSystemLanguage(t *testing.T, l i18n.Lang) {
	t.Helper()
	t.Cleanup(app.WithSystemLanguage(l))
}

// S2: the Language row lists Auto (system), English and Español, and
// shows the saved choice.
func TestSettingsLanguageRowListsTheChoices(t *testing.T) {
	useLang(t, i18n.EN)
	h := settingsHarness(t, tempConfig(t, ""))
	h.openSettings()
	if v := h.view(); !strings.Contains(v, "Language") || !strings.Contains(v, "Auto (system)") {
		t.Fatalf("settings menu lacks the Language row:\n%s", v)
	}
	h.keys("8")
	v := h.view()
	for _, want := range []string{"Auto (system)", "English", "Español"} {
		if !strings.Contains(v, want) {
			t.Fatalf("the language list lacks %q:\n%s", want, v)
		}
	}

	h = settingsHarness(t, tempConfig(t, ""), app.WithUI(config.UI{Language: "es"}))
	h.openSettings()
	if v := h.view(); !strings.Contains(v, "Español") {
		t.Fatalf("the Language row should show the saved Español:\n%s", v)
	}
}

// S2, S9: a choice applies at once, redraws the menu in its language, and
// is saved at once; Auto removes the key and follows the system.
func TestSettingsLanguageAppliesLiveAndIsSaved(t *testing.T) {
	useLang(t, i18n.EN)
	useSystemLanguage(t, i18n.ES)
	path := tempConfig(t, userConfig)
	h := settingsHarness(t, path)
	h.openSettings()
	h.keys("83") // Language → Español
	if got := i18n.Current(); got != i18n.ES {
		t.Fatalf("after choosing Español the language is %q", got)
	}
	v := h.view()
	for _, want := range []string{"Ajustes", "Idioma", "Español", "Carpetas en acordeón"} {
		if !strings.Contains(v, want) {
			t.Fatalf("the menu should redraw in Spanish at once, lacks %q:\n%s", want, v)
		}
	}
	assertOnlyUIChanged(t, path, map[string]any{"language": "es"})

	h.keys("82") // English
	if got := i18n.Current(); got != i18n.EN {
		t.Fatalf("after choosing English the language is %q", got)
	}
	if v := h.view(); !strings.Contains(v, "Settings") || !strings.Contains(v, "Accordion folders") {
		t.Fatalf("the menu should redraw in English:\n%s", v)
	}
	assertOnlyUIChanged(t, path, map[string]any{"language": "en"})

	h.keys("81") // Auto (system): the system says Spanish here
	if got := i18n.Current(); got != i18n.ES {
		t.Fatalf("Auto should follow the system language (es), got %q", got)
	}
	if v := h.view(); !strings.Contains(v, "Automático (sistema)") {
		t.Fatalf("the Language row should show Auto in Spanish:\n%s", v)
	}
	assertOnlyUIChanged(t, path, nil)
}

// A language already chosen saves nothing.
func TestSettingsSameLanguageWritesNothing(t *testing.T) {
	useLang(t, i18n.ES)
	path := tempConfig(t, `{"ui": {"language": "es"}}`)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	h := settingsHarness(t, path, app.WithUI(config.UI{Language: "es"}))
	h.openSettings()
	h.keys("83") // Español, already in use
	if info, _ := os.Stat(path); !info.ModTime().Equal(old) {
		t.Fatalf("an unchanged language rewrote the config")
	}
}

// The label column is as wide as the longest label in the language shown:
// every row's value starts in the same column.
func TestSettingsLabelsAlignInEveryLanguage(t *testing.T) {
	for _, l := range i18n.Langs {
		t.Run(string(l), func(t *testing.T) {
			useLang(t, l)
			h := settingsHarness(t, tempConfig(t, ""))
			h.openSettings()
			v := h.view()
			values := map[i18n.Key]string{i18n.SettingHideWidth: "120", i18n.SettingBorder: "rounded"}
			col := -1
			for key, value := range values {
				label := i18n.T(key)
				var line string
				for _, l := range strings.Split(v, "\n") {
					if strings.Contains(l, label) {
						line = l
						break
					}
				}
				if line == "" {
					t.Fatalf("no row %q:\n%s", label, v)
				}
				at := strings.Index(line, label)
				rest := line[at+len(label):]
				i := strings.Index(rest, value)
				if i < 0 {
					t.Fatalf("row %q lacks %q: %q", label, value, line)
				}
				c := ansi.StringWidth(line[:at+len(label)+i])
				if col >= 0 && c != col {
					t.Fatalf("values start in different columns (%d and %d):\n%s", col, c, v)
				}
				col = c
			}
		})
	}
}

// Stable IDs: the table cell menu dispatches on the action, not on its
// shown words, so it works in Spanish.
func TestTableCellMenuWorksInSpanish(t *testing.T) {
	useLang(t, i18n.ES)
	h := openWith(t, "| Name | Role |\n| --- | --- |\n| Ada | Eng |\n| Bob | Ops |")
	h.clickText("Ada")
	if v := h.view(); !strings.Contains(v, "Eliminar fila") || !strings.Contains(v, "Editar celda") {
		t.Fatalf("expected the cell menu in Spanish:\n%s", v)
	}
	h.keys("5") // Eliminar fila
	if got := h.text(); strings.Contains(got, "Ada") || !strings.Contains(got, "Bob") {
		t.Fatalf("Eliminar fila should delete Ada's row:\n%s", got)
	}
	h.clickText("Bob")
	h.keys("1") // Editar celda
	for range "Bob" {
		h.key(tea.KeyBackspace)
	}
	h.keys("Eve")
	h.key(tea.KeyEnter)
	if got := h.text(); !strings.Contains(got, "| Eve") {
		t.Fatalf("Editar celda should edit the cell:\n%s", got)
	}
}

// Stable IDs: the preset in use is found by identity, not by its shown
// name, so choosing it again in Spanish saves nothing.
func TestSettingsSamePresetInSpanishWritesNothing(t *testing.T) {
	useLang(t, i18n.ES)
	body := `{"ui": {"heading_icons": ["① ", "② ", "③ ", "④ ", "⑤ ", "⑥ "], "bullets": ["•", "◦", "▪", "▫"]}}`
	path := tempConfig(t, body)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	ui := config.UI{HeadingIcons: []string{"① ", "② ", "③ ", "④ ", "⑤ ", "⑥ "}, Bullets: []string{"•", "◦", "▪", "▫"}}
	h := settingsHarness(t, path, app.WithUI(ui))
	h.openSettings()
	if v := h.view(); !strings.Contains(v, "Números") {
		t.Fatalf("the row should name the preset in Spanish:\n%s", v)
	}
	h.keys("32") // Números, already in use
	if info, _ := os.Stat(path); !info.ModTime().Equal(old) {
		t.Fatalf("an unchanged preset rewrote the config")
	}
	// Nothing changed, so nothing is applied or said.
	if got := app.Status(h.m.(app.Model)); strings.Contains(got, "Números") {
		t.Fatalf("choosing the preset in use reported a change: %q", got)
	}
}

// Stable IDs, S9: the "reading" line is cleared when the status arrives,
// even if the language changed in between.
func TestSyncStatusClearsItsReadingLineAcrossALanguageChange(t *testing.T) {
	useLang(t, i18n.ES)
	_, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	h.keys("1")
	h.key(tea.KeyEsc)
	cmd := h.update(core.SyncStatusMsg{}) // what `:sync status` asks for
	reading := app.Status(h.m.(app.Model))
	if reading == "" || reading == "Reading the sync status…" {
		t.Fatalf("the reading line should be in Spanish, got %q", reading)
	}
	i18n.Set(i18n.EN)
	h.finish(cmd)
	if got := app.Status(h.m.(app.Model)); got == reading {
		t.Fatalf("the reading line %q stayed after the status arrived", got)
	}
}

// Stable IDs: the tutorial's own failed save is told apart from the
// player's errors without reading its words, so it never passes a step
// that asks for an error, in any language.
func TestTutorSaveErrorIsNotAnErrorShownInSpanish(t *testing.T) {
	useLang(t, i18n.ES)
	h, p, l := eventFixture(t, "# Viaje\n", []tutor.Step{actionStep("error_shown")})
	h.send(app.TutorSaveFailed(errors.New("disco lleno")))
	if got := app.Status(h.m.(app.Model)); !strings.Contains(got, "disco lleno") || strings.Contains(got, "Could not") {
		t.Fatalf("the bar should show the failed save in Spanish, got %q", got)
	}
	h.key(tea.KeyEsc)
	assertNotPassed(t, p, l, 1)
}

// S9: the preview is drawn again in the language just chosen in
// :settings, without waiting for the note to change.
func TestSettingsLanguageRedrawsThePreview(t *testing.T) {
	useLang(t, i18n.EN)
	store := newTestStore(t)
	if _, err := store.SaveAs("", "doc", "```\nx := 1\n```\n"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithConfigPath(tempConfig(t, "")))
	h.keys("1")
	h.key(tea.KeyEsc)
	if v := h.view(); !strings.Contains(v, "code · 1 line") {
		t.Fatalf("the preview lacks the English code label:\n%s", v)
	}
	h.openSettings()
	h.keys("83") // Language → Español
	h.key(tea.KeyEsc)
	v := h.view()
	if !strings.Contains(v, "código · 1 línea") || strings.Contains(v, "code · 1 line") {
		t.Fatalf("the preview should redraw in Spanish at once:\n%s", v)
	}
}
