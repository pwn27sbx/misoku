package globalgraph

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// TestFadeBlendsTowardBaseOnATransparentTheme: a transparent theme clears
// Bg, so the map's fades must blend toward the opaque Base instead of
// falling back to the unfaded colour.
func TestFadeBlendsTowardBaseOnATransparentTheme(t *testing.T) {
	th := theme.DefaultTheme()
	th.Bg, th.DarkerBg, th.StatusBg = "#101112", "#0a0b0c", "#131415"
	th.Base, th.PopupBg = th.Bg, th.DarkerBg
	th = th.Transparent()
	d := &drawing{th: th}
	const fallback = lipgloss.Color("#ffffff")
	got := d.fade("#e00010", 0.5, fallback)
	if want := fade("#e00010", "#101112", 0.5, fallback); got != want || got == fallback {
		t.Errorf("fade on a transparent theme = %q, want %q (toward Base)", got, want)
	}
}
