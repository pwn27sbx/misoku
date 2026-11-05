package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// The welcome screen honours ui.transparent: its background is the
// terminal's (no fill of the palette's bg, which no other shade shares),
// and without the option it is painted as before.
func TestWelcomeScreenHonoursTransparent(t *testing.T) {
	dir := t.TempDir()
	theme.SetUserThemesPath(dir)
	t.Cleanup(func() { theme.SetUserThemesPath("") })
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	body := `{"fg": "#c0c1c2", "bg": "#101112", "darker_bg": "#0a0b0c", "status_bg": "#131415", "blue": "#1050e0"}`
	if err := os.WriteFile(filepath.Join(dir, "clear.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	const bgFill = "48;2;16;17;18"
	for _, c := range []struct {
		transparent bool
		painted     bool
	}{{true, false}, {false, true}} {
		o := welcomeOptions{Home: t.TempDir()}
		s := newWelcomeScreen(o, welcomeTheme("clear", c.transparent))
		next, _ := s.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		v := next.(welcomeScreen).View()
		if got := strings.Contains(v, bgFill); got != c.painted {
			t.Errorf("transparent=%v: bg painted = %v, want %v", c.transparent, got, c.painted)
		}
	}
}
