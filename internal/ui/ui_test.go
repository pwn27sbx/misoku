package ui_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/pwn27sbx/misoku/internal/ui"
)

func TestPaintAppliesDefaultColours(t *testing.T) {
	r := lipgloss.NewRenderer(nil)
	r.SetColorProfile(termenv.TrueColor)
	lipgloss.SetDefaultRenderer(r)
	paint := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Background(lipgloss.Color("#000000"))
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render("red")
	out := ui.Paint("plain "+red+" after\nline2", paint)
	for _, l := range strings.Split(out, "\n") {
		if !strings.HasPrefix(l, "\x1b[") || !strings.Contains(l, "48;2;0;0;0") {
			t.Fatalf("line not painted: %q", l)
		}
	}
	// After the red span resets, the background must come back.
	if !strings.Contains(out, "\x1b[0m\x1b[") {
		t.Fatalf("expected default colours restored after a reset: %q", out)
	}
}

func TestColumnRendersDivider(t *testing.T) {
	out := ui.Column("│", 3, lipgloss.NewStyle())
	if got := strings.Split(out, "\n"); len(got) != 3 {
		t.Fatalf("Column gave %d rows", len(got))
	}
	if ui.Column("│", 0, lipgloss.NewStyle()) != "" {
		t.Error("expected empty for h=0")
	}
}

func TestCenter(t *testing.T) {
	x, y := ui.Center(100, 50, 20, 10)
	if x != 40 || y != 20 {
		t.Errorf("Center = %d,%d", x, y)
	}
	if x, y := ui.Center(4, 4, 20, 20); x != 0 || y != 0 {
		t.Errorf("Center overflow = %d,%d", x, y)
	}
}

func TestTruncate(t *testing.T) {
	if got := ui.Truncate("hello world", 5); got != "hell…" {
		t.Errorf("Truncate = %q", got)
	}
	if got := ui.Truncate("hi", 10); got != "hi" {
		t.Errorf("Truncate short = %q", got)
	}
	if got := ui.Truncate("hi", 1); got != "…" {
		t.Errorf("Truncate w=1 = %q", got)
	}
	if got := ui.Truncate("hi", 0); got != "" {
		t.Errorf("Truncate w=0 = %q", got)
	}
}

func TestFitEdges(t *testing.T) {
	if ui.Fit("x", 0, 5, lipgloss.NewStyle()) != "" {
		t.Error("expected empty for w=0")
	}
	out := ui.Fit("a\nb\nc\nd", 3, 2, lipgloss.NewStyle())
	if got := len(strings.Split(out, "\n")); got != 2 {
		t.Errorf("expected 2 rows, got %d", got)
	}
}

func TestFitAndOverlayKeepWidth(t *testing.T) {
	plain := lipgloss.NewStyle()
	box := ui.Fit("abc\ndefghijkl", 5, 3, plain)
	lines := strings.Split(box, "\n")
	if len(lines) != 3 || lipgloss.Width(lines[1]) != 5 {
		t.Fatalf("Fit gave %q", box)
	}
	out := ui.Overlay("..........\n..........", "XX", 4, 1)
	if strings.Split(out, "\n")[1] != "....\x1b[0mXX\x1b[0m...." {
		t.Fatalf("Overlay gave %q", out)
	}
}

func TestFitLineEllipsis(t *testing.T) {
	plain := lipgloss.NewStyle()
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"hello world", 5, "hell…"},
		{"hi", 5, "hi   "},
		{"hello", 5, "hello"},
		{"日本語", 4, "日… "}, // the wide 本 does not fit beside …: padded
		{"", 0, ""},
	}
	for _, c := range cases {
		got := ui.FitLineEllipsis(c.in, c.w, plain)
		if got != c.want {
			t.Errorf("FitLineEllipsis(%q, %d) = %q, want %q", c.in, c.w, got, c.want)
		}
	}
	styled := lipgloss.NewStyle().Bold(true).Render("hello world")
	if got := ui.FitLineEllipsis(styled, 6, plain); ansi.StringWidth(got) != 6 || !strings.HasSuffix(ansi.Strip(got), "…") {
		t.Errorf("styled cut = %q", got)
	}
	// FitLine itself still cuts without an ellipsis.
	if got := ui.FitLine("hello world", 5, plain); got != "hello" {
		t.Errorf("FitLine = %q", got)
	}
}
