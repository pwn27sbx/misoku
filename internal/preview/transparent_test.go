package preview

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// transparentTheme is a user palette whose bg, darker_bg and status_bg no
// other shade shares, made transparent, rendered in 24-bit colour.
func transparentTheme(t *testing.T) theme.Theme {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
	dir := t.TempDir()
	theme.SetUserThemesPath(dir)
	t.Cleanup(func() { theme.SetUserThemesPath("") })
	body := `{
		"fg": "#c0c1c2", "grey": "#808182", "grey_fg": "#909192", "grey_fg2": "#a0a1a2",
		"light_grey": "#b0b1b2", "yellow": "#e0d000", "red": "#e00010", "green": "#10e020",
		"blue": "#1050e0", "nord_blue": "#2060d0", "purple": "#a020e0", "dark_purple": "#9030c0",
		"teal": "#10c0c0", "orange": "#e08010", "cyan": "#20d0f0", "pink": "#f020a0",
		"folder": "#60a0f0", "line": "#404142",
		"darker_bg": "#0a0b0c", "bg": "#101112", "status_bg": "#131415",
		"bg2": "#1b1c1d", "one_bg": "#232425", "one_bg2": "#2b2c2d",
		"one_bg3": "#333435", "light_bg": "#3b3c3d"
	}`
	if err := os.WriteFile(filepath.Join(dir, "distinct.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	th, ok := theme.Get("distinct")
	if !ok {
		t.Fatal("distinct theme not loaded")
	}
	return th.Transparent()
}

// sgrOf is the 24-bit colour c as an SGR parameter list (38 foreground,
// 48 background).
func sgrOf(kind int, c lipgloss.Color) string {
	p := parseHex(string(c))
	return fmt.Sprintf("%d;2;%d;%d;%d", kind, p.R, p.G, p.B)
}

// TestAccentTextKeepsDarkFgOnATransparentTheme: board pills and the today
// marks of calendars and timelines draw DarkFg on their accent.
func TestAccentTextKeepsDarkFgOnATransparentTheme(t *testing.T) {
	th := transparentTheme(t)
	now := time.Now()
	day := func(d int) string { return now.AddDate(0, 0, d).Format("2006-01-02") }
	for _, c := range []struct{ name, md string }{
		{"board pill", "```board\n## Todo\n- A\n```\n"},
		{"calendar today", "```calendar\nmonth: " + now.Format("2006-01") + "\n```\n"},
		{"timeline today", "```timeline\nAlpha: " + day(-5) + " -> " + day(5) + "\n```\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := CompileWith(c.md, th, 80, "", CalendarView{})
			if !strings.Contains(out, sgrOf(38, th.DarkFg)+";48;2;") {
				t.Errorf("no DarkFg text on an accent in:\n%q", out)
			}
		})
	}
}

// TestColourMathsUseBaseOnATransparentTheme: heading bands and pictures
// blend toward the opaque Base, not the cleared Bg.
func TestColourMathsUseBaseOnATransparentTheme(t *testing.T) {
	th := transparentTheme(t)

	look := DefaultLook()
	look.HeadingBackground = true
	out, _, _ := CompileBlocks("## Sub\n", th, look, 40, "", CalendarView{})
	if want := sgrOf(48, band(th.Purple, th.Base)); !strings.Contains(out, want) {
		t.Errorf("the heading band should blend its colour toward Base:\n%q", out)
	}

	// A fully transparent picture composites on the background colour.
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "clear.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	// Covers posterize their colours, which flattens the distinct palette's
	// near-black Base to black: a bright Base keeps the two apart.
	th.Base = "#c08040"
	cover := fmt.Sprintf("38;2;%d;%d;%d", posterize(0xc0), posterize(0x80), posterize(0x40))
	for name, c := range map[string]struct{ md, want string }{
		"image": {"![clear](clear.png)\n", sgrOf(38, th.Base)},
		"cover": {"---\ncover: clear.png\n---\n# T\n", cover},
	} {
		if out := CompileIn(c.md, th, 20, dir); !strings.Contains(out, c.want) {
			t.Errorf("%s: a clear picture should show Base %s:\n%q", name, th.Base, out)
		}
	}
}
