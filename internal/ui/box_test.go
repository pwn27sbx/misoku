package ui_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/ui"
)

// TitledBox frames rows in the current border, its title centred on the top
// edge between spaces, like Neovim's floating windows.
func TestTitledBoxCentresTheTitleOnTheTopBorder(t *testing.T) {
	box := ansi.Strip(ui.TitledBox([]string{"› hi"}, 20, "Cmdline", lipgloss.Color("4"), lipgloss.Color("")))
	rows := strings.Split(box, "\n")
	want := []string{
		"╭───── Cmdline ──────╮",
		"│› hi                │",
		"╰────────────────────╯",
	}
	if strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Fatalf("box:\n%s\nwant:\n%s", box, strings.Join(want, "\n"))
	}
}

// A title wider than the box is cut, so the box keeps its width.
func TestTitledBoxCutsALongTitle(t *testing.T) {
	box := ansi.Strip(ui.TitledBox([]string{""}, 6, "A very long title", lipgloss.Color("4"), lipgloss.Color("")))
	for i, r := range strings.Split(box, "\n") {
		if w := ansi.StringWidth(r); w != 8 {
			t.Fatalf("row %d is %d wide, want 8: %q", i, w, r)
		}
	}
}
