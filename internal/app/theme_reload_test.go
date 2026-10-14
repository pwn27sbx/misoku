package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/theme"
)

func writePalette(t *testing.T, path, fg string, when time.Time) {
	t.Helper()
	// The accent colors follow fg, so headings change with it too.
	body := fmt.Sprintf(`{"fg": %q, "bg": "#101010", "blue": %q, "nord_blue": %q, "purple": %q}`, fg, fg, fg, fg)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

// sgr is how the terminal is told to paint text in hex color c.
func sgr(c string) string {
	var r, g, b int
	fmt.Sscanf(c, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

func TestUserThemeReloadsWhenItsFileChanges(t *testing.T) {
	dir := t.TempDir()
	theme.SetUserThemesPath(dir)
	t.Cleanup(func() { theme.SetUserThemesPath("") })
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	path := filepath.Join(dir, "mine.json")
	start := time.Now().Add(-time.Hour)
	writePalette(t, path, "#aa0000", start)

	store := newTestStore(t)
	// A heading, so the editor's syntax colors are part of the check.
	if _, err := store.SaveAs("", "doc", "# Title\n\nhello"); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, app.WithTheme("mine"))
	h.keys("1")
	if v := h.m.View(); !strings.Contains(v, sgr("#aa0000")) {
		t.Fatalf("setup: the first palette should be in use")
	}

	// What `tema` does on a theme change: rewrite the same file.
	writePalette(t, path, "#00bb00", start.Add(time.Minute))
	h.send(app.CheckWorkspaceMsg{})
	v := h.m.View()
	if !strings.Contains(v, sgr("#00bb00")) || strings.Contains(v, sgr("#aa0000")) {
		t.Fatalf("the edited palette should replace the old one without restarting")
	}
}

func TestBundledThemeIgnoresUserThemeFiles(t *testing.T) {
	dir := t.TempDir()
	theme.SetUserThemesPath(dir)
	t.Cleanup(func() { theme.SetUserThemesPath("") })
	h := newHarness(t, newTestStore(t), app.WithTheme("nord"))
	writePalette(t, filepath.Join(dir, "other.json"), "#00bb00", time.Now())
	before := h.m.View()
	h.send(app.CheckWorkspaceMsg{})
	if after := h.m.View(); after != before {
		t.Fatal("a bundled theme must not change when user palettes do")
	}
}
