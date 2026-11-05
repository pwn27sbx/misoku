package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// The base fills of the "clear" palette: no other shade shares these hexes,
// so finding one in a frame means that fill was painted.
const (
	clearBg       = "#101112"
	clearDarkerBg = "#0a0b0c"
	clearStatusBg = "#131415"
)

// clearPalette is a user palette whose bg, darker_bg and status_bg differ
// from every other shade.
func clearPalette(bg string) string {
	return fmt.Sprintf(`{
		"fg": "#c0c1c2", "grey": "#808182", "grey_fg": "#909192", "grey_fg2": "#a0a1a2",
		"light_grey": "#b0b1b2", "yellow": "#e0d000", "red": "#e00010", "green": "#10e020",
		"blue": "#1050e0", "nord_blue": "#2060d0", "purple": "#a020e0", "dark_purple": "#9030c0",
		"teal": "#10c0c0", "orange": "#e08010", "cyan": "#20d0f0", "pink": "#f020a0",
		"folder": "#60a0f0", "line": "#404142",
		"darker_bg": %q, "bg": %q, "status_bg": %q,
		"bg2": "#1b1c1d", "one_bg": "#232425", "one_bg2": "#2b2c2d",
		"one_bg3": "#333435", "light_bg": "#3b3c3d"
	}`, clearDarkerBg, bg, clearStatusBg)
}

// useClearTheme installs the "clear" user palette and paints in true colour
// for the rest of the test. It returns the palette file.
func useClearTheme(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	theme.SetUserThemesPath(dir)
	t.Cleanup(func() { theme.SetUserThemesPath("") })
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	path := filepath.Join(dir, "clear.json")
	if err := os.WriteFile(path, []byte(clearPalette(clearBg)), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// fill is how the terminal is told to paint the background in hex color c.
func fill(c string) string {
	var r, g, b int
	fmt.Sscanf(c, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
}

// baseFills are the fills the transparent option leaves to the terminal.
var baseFills = map[string]string{"bg": clearBg, "darker_bg": clearDarkerBg, "status_bg": clearStatusBg}

// assertClear fails when frame paints any of the base fills.
func assertClear(t *testing.T, name, frame string) {
	t.Helper()
	for field, hex := range baseFills {
		if strings.Contains(frame, fill(hex)) {
			t.Errorf("%s: %s (%s) is painted, but the background should be the terminal's", name, field, hex)
		}
	}
}

// assertOpaque fails when frame lacks any of the base fills.
func assertOpaque(t *testing.T, name, frame string) {
	t.Helper()
	for field, hex := range baseFills {
		if !strings.Contains(frame, fill(hex)) {
			t.Errorf("%s: %s (%s) is not painted", name, field, hex)
		}
	}
}

// homeAndEditor are the home screen (the first frame) and the editor with
// a note open, drawn by h.
func homeAndEditor(t *testing.T, h *harness) (string, string) {
	t.Helper()
	home := h.m.View()
	h.keys("1")
	h.key(tea.KeyEsc)
	return home, h.m.View()
}

// With ui.transparent the home screen and the editor leave bg, darker_bg
// and status_bg to the terminal from the first frame on, whichever order
// the options come in (main passes WithTheme before WithUI), and still
// paint the cursor line.
func TestTransparentOptionClearsTheBaseFills(t *testing.T) {
	for _, c := range []struct {
		name  string
		order func(theme, ui app.Option) []app.Option
	}{
		{"theme then ui", func(th, ui app.Option) []app.Option { return []app.Option{th, ui} }},
		{"ui then theme", func(th, ui app.Option) []app.Option { return []app.Option{ui, th} }},
	} {
		t.Run(c.name, func(t *testing.T) {
			useClearTheme(t)
			store := newTestStore(t)
			if _, err := store.SaveAs("notes", "doc", "# Title\n\nhello"); err != nil {
				t.Fatal(err)
			}
			opts := c.order(app.WithTheme("clear"), app.WithUI(config.UI{Transparent: true}))
			home, editor := homeAndEditor(t, newHarness(t, store, opts...))
			assertClear(t, "home", home)
			assertClear(t, "editor", editor)
			if !strings.Contains(editor, fill("#1b1c1d")) {
				t.Error("the cursor line (bg2) should stay painted")
			}
		})
	}
}

// Without the option the same frames paint every base fill, as before.
func TestWithoutTransparentOptionTheBaseIsPainted(t *testing.T) {
	useClearTheme(t)
	store := newTestStore(t)
	if _, err := store.SaveAs("notes", "doc", "# Title\n\nhello"); err != nil {
		t.Fatal(err)
	}
	home, editor := homeAndEditor(t, newHarness(t, store, app.WithTheme("clear"), app.WithUI(config.UI{})))
	if !strings.Contains(home, fill(clearBg)) {
		t.Error("home: bg is not painted")
	}
	assertOpaque(t, "editor", editor)
}

// The settings row turns the option on, off and on again, live; off paints
// the theme's own background again, and only ui.transparent is saved.
func TestSettingsTransparentTogglesLiveAndIsSaved(t *testing.T) {
	useClearTheme(t)
	path := tempConfig(t, userConfig)
	h := settingsHarness(t, path, app.WithTheme("clear"))
	var status string
	toggle := func() string {
		t.Helper()
		h.openSettings()
		h.keys("5")
		status = stripped(h.m.View())
		h.key(tea.KeyEsc)
		return h.m.View()
	}

	assertClear(t, "on", toggle())
	if !strings.Contains(status, "Transparent background: on") {
		t.Errorf("status should say the option is on:\n%s", status)
	}
	assertOnlyUIChanged(t, path, map[string]any{"transparent": true})

	assertOpaque(t, "off", toggle())
	assertOnlyUIChanged(t, path, map[string]any{})

	assertClear(t, "on again", toggle())
	assertOnlyUIChanged(t, path, map[string]any{"transparent": true})
}

// The settings menu shows the option's row with its state.
func TestSettingsMenuShowsTransparentBackground(t *testing.T) {
	h := settingsHarness(t, tempConfig(t, ""), app.WithUI(config.UI{Transparent: true}))
	h.openSettings()
	if v := h.view(); !strings.Contains(v, "Transparent background on") {
		t.Fatalf("settings menu lacks the transparent row:\n%s", v)
	}
}

// A user theme rewritten on disk (Omarchy's theme switch) is reloaded and
// stays transparent.
func TestUserThemeReloadKeepsTransparency(t *testing.T) {
	path := useClearTheme(t)
	start := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, start, start); err != nil {
		t.Fatal(err)
	}
	store := newTestStore(t)
	if _, err := store.SaveAs("notes", "doc", "hello"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithTheme("clear"), app.WithUI(config.UI{Transparent: true}))
	h.keys("1")

	const newBg = "#161718"
	if err := os.WriteFile(path, []byte(clearPalette(newBg)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, start.Add(time.Minute), start.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	h.send(app.CheckWorkspaceMsg{})
	v := h.m.View()
	assertClear(t, "after reload", v)
	if strings.Contains(v, fill(newBg)) {
		t.Error("the reloaded palette's bg is painted")
	}
}
