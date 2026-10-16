package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/app"
)

// unreadableConfig writes a config that is not a JSON object: Save
// refuses to rewrite it, so the theme cannot be saved.
func unreadableConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`["not", "an", "object"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// :colorscheme with a config it cannot save: the error stays on screen
// (the theme is gone after a restart) instead of "Theme: nord".
func TestUnsavedThemeIsReported(t *testing.T) {
	h := colorscheme(t, unreadableConfig(t), "nord")
	v := h.view()
	if !strings.Contains(v, "Could not save theme") {
		t.Errorf("no message in the app after a failed save:\n%s", v)
	}
	if strings.Contains(v, "Theme: nord") {
		t.Errorf("a failed save must not say the theme was set:\n%s", v)
	}
}

// The theme picker's Enter, same rule.
func TestUnsavedThemeFromThePickerIsReported(t *testing.T) {
	h := newHarness(t, newTestStore(t), app.WithConfigPath(unreadableConfig(t)))
	h.keys("n")
	h.key(tea.KeyEsc)
	h.keys(":colorscheme")
	h.key(tea.KeyEnter) // opens the picker
	h.key(tea.KeyDown)
	h.key(tea.KeyEnter) // keeps the highlighted theme
	v := h.view()
	if !strings.Contains(v, "Could not save theme") || strings.Contains(v, "Theme: ") {
		t.Errorf("after a failed save from the picker:\n%s", v)
	}
}

// A theme that is saved still says so.
func TestSavedThemeSaysSo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if v := colorscheme(t, path, "nord").view(); !strings.Contains(v, "Theme: nord") {
		t.Errorf("no status after a saved theme:\n%s", v)
	}
	h := newHarness(t, newTestStore(t), app.WithConfigPath(filepath.Join(t.TempDir(), "config.json")))
	h.keys("n")
	h.key(tea.KeyEsc)
	h.keys(":colorscheme")
	h.key(tea.KeyEnter)
	h.key(tea.KeyEnter)
	if v := h.view(); !strings.Contains(v, "Theme: ") {
		t.Errorf("no status after a theme saved from the picker:\n%s", v)
	}
}
