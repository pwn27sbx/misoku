package ui_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/pwn27sbx/misoku/internal/kitty"
	"github.com/pwn27sbx/misoku/internal/ui"
)

// S6: a picture behind a modal box is hidden, its cells blank and as wide
// as before; a dimmed pane keeps its pictures.
func TestBackdropHidesPictures(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
	row := "ab" + kitty.Placeholders(7, 3, 1)[0] + "cd"
	got := ui.DimBackdrop(row, "#808182")
	if strings.ContainsRune(got, 0x10EEEE) {
		t.Errorf("the picture survived the backdrop: %q", got)
	}
	if ansi.StringWidth(got) != ansi.StringWidth(row) {
		t.Errorf("the row changed width: %d, want %d", ansi.StringWidth(got), ansi.StringWidth(row))
	}
	if plain := ansi.Strip(got); plain != "ab   cd" {
		t.Errorf("got %q, want the picture's 3 cells blank", plain)
	}
	if ui.Dim(row, "#808182") != row {
		t.Error("a dimmed pane should keep its picture")
	}
}

// A backdrop keeps no background, basic or true colour, and no reverse video;
// a dimmed pane keeps its backgrounds.
func TestDimBackdropDropsBackgrounds(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
	row := "\x1b[41ma\x1b[0m \x1b[48;2;1;2;3mb\x1b[0m \x1b[7;1mc\x1b[0m \x1b[101md\x1b[0m"
	got := ui.DimBackdrop(row, "#808182")
	if strings.Contains(got, "48;2;1;2;3") {
		t.Errorf("the backdrop kept a true-colour background: %q", got)
	}
	for _, seq := range strings.Split(got, "\x1b[")[1:] {
		for _, p := range strings.Split(strings.SplitN(seq, "m", 2)[0], ";") {
			if p == "41" || p == "101" || p == "7" {
				t.Errorf("the backdrop kept SGR %s: %q", p, got)
			}
		}
	}
	if !strings.Contains(got, "\x1b[1m") {
		t.Errorf("the backdrop dropped bold: %q", got)
	}
	if plain := ansi.Strip(got); plain != "a b c d" {
		t.Errorf("the text changed: %q", plain)
	}
	if pane := ui.Dim(row, "#808182"); !strings.Contains(pane, "41") || !strings.Contains(pane, "48;2;1;2;3") {
		t.Errorf("a dimmed pane lost its backgrounds: %q", pane)
	}
}
