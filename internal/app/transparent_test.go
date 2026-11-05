package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// distinctTheme installs a user palette whose bg, darker_bg and status_bg
// hexes no other shade shares, so a background SGR found in a frame can
// only come from the field the assertion names.
func distinctTheme(t *testing.T) theme.Theme {
	t.Helper()
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
	return th
}

// paint is the SGR that opens a run of text drawn in style st.
func paint(st lipgloss.Style) string { return strings.SplitN(st.Render("x"), "x", 2)[0] }

// popup is the floating boxes' fill.
func popup(th theme.Theme) lipgloss.Style { return lipgloss.NewStyle().Background(th.PopupBg) }

// onAccent is text drawn in DarkFg on accent, bold, as badges and selections are.
func onAccent(th theme.Theme, accent lipgloss.Color) string {
	return paint(lipgloss.NewStyle().Background(accent).Foreground(th.DarkFg).Bold(true))
}

// transparentModel is a w × h model on the distinct theme made transparent,
// with two notes in its store.
func transparentModel(t *testing.T, w, h int) *Model {
	t.Helper()
	return distinctModel(t, w, h, true)
}

// distinctModel is a w × h model on the distinct theme, made transparent
// when clear is set, with two notes in its store.
func distinctModel(t *testing.T, w, h int, clear bool) *Model {
	t.Helper()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	th := distinctTheme(t)
	if clear {
		th = th.Transparent()
	}
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"doc", "other"} {
		if _, err := store.SaveAs("", n, n+" body"); err != nil {
			t.Fatal(err)
		}
	}
	tm, _ := New(store).Update(tea.WindowSizeMsg{Width: w, Height: h})
	m := tm.(Model)
	m.applyTheme(th, false)
	return &m
}

func blankScreen(w, h int) string {
	return strings.TrimSuffix(strings.Repeat(strings.Repeat(" ", w)+"\n", h), "\n")
}

// The distinct theme's fills as background SGRs: darker_bg (PopupBg), bg
// and status_bg.
const (
	popupSGR  = "48;2;10;11;12"
	bgSGR     = "48;2;16;17;18"
	statusSGR = "48;2;19;20;21"
)

// TestFloatingBoxesFollowTheTransparency: like a Neovim theme with
// transparent floats, every box drawn over the panes paints no background
// when the theme is transparent and is set apart only by its border and
// title colours; on the opaque theme it keeps its PopupBg fill. Text drawn
// on an accent keeps DarkFg in both.
func TestFloatingBoxesFollowTheTransparency(t *testing.T) {
	for _, c := range []struct {
		name string
		// box is set when render returns a whole box: its rows must be
		// equally wide, so the overlay replaces every cell under it.
		box    bool
		render func(m *Model) string
		// want lists what the box must contain, each built from m.theme.
		want func(th theme.Theme) []string
	}{
		{"confirm dialog", true, func(m *Model) string {
			d := &confirmDialog{title: "Delete note?", message: []string{"It goes to the Trash."}, buttons: []string{"Delete", "Cancel"}}
			box, _, _ := d.view(m.theme, 80, 24)
			return box
		}, func(th theme.Theme) []string {
			return []string{
				paint(popup(th).Foreground(th.Blue)) + "╭",                     // panel border
				paint(popup(th)) + strings.Repeat(" ", 10),                     // panel padding of an empty row
				paint(popup(th).Foreground(th.Fg).Bold(true)) + "Delete note?", // title row
				onAccent(th, th.Blue) + "  Delete  ",                           // selected button
			}
		}},
		{"save-as dialog", true, func(m *Model) string {
			m.openSaveAs(&buffer{}, nil)
			box, _, _ := m.saveAs.view(m.theme, 80, 24)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + "Save As"}
		}},
		{"menu", true, func(m *Model) string {
			m.menuBox = newMenu("Pick one", []string{"first", "second"}, nil)
			box, _, _ := m.menuBox.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + " Pick one"}
		}},
		{"palette", true, func(m *Model) string {
			m.openPalette()
			box, _, _ := m.palette.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + "MISOKU KEYMAPS & COMMANDS"}
		}},
		{"finder", true, func(m *Model) string {
			m.openFinder()
			box, _, _ := m.finder.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + "Find Note"}
		}},
		{"theme picker", true, func(m *Model) string {
			m.openThemePicker()
			box, _, _ := m.themes.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{
				paint(popup(th).Foreground(th.Fg).Bold(true)) + "Themes",    // title
				popup(th).Render("  ") + paint(popup(th).Foreground(th.Fg)), // an unselected row's marker, then its name
			}
		}},
		{"finder result rows", false, func(m *Model) string {
			m.openFinder()
			return m.finder.resultRow(m.theme, 1, 40) + "|" + m.finder.resultRow(m.theme, 99, 40)
		}, func(th theme.Theme) []string {
			return []string{
				popup(th).Render("  "),                    // an unselected row
				popup(th).Render(strings.Repeat(" ", 40)), // an empty row
			}
		}},
		{"finder preview", false, func(m *Model) string {
			return previewLine(m.theme, "plain text", 30)
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.GreyFg2)) + "plain text"}
		}},
		{"date picker", true, func(m *Model) string {
			m.pickDate("When", time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC), time.Time{}, nil)
			box, _, _ := m.datePick.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.GreyFg2)) + " Wed, 18 Nov 2026", onAccent(th, th.Blue) + " 18 "}
		}},
		{"media browser", true, func(m *Model) string {
			m.openFileBrowser("image")
			box, _, _ := m.browser.view(m)
			return box
		}, func(th theme.Theme) []string { return []string{paint(popup(th).Foreground(th.Fg).Bold(true))} }},
		{"prompt", true, func(m *Model) string {
			m.prompt("Rename", "value", "a hint", nil)
			box, _, _ := m.promptBox.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + "Rename"}
		}},
		{"leader guide", true, func(m *Model) string {
			return m.renderLeaderGuide()
		}, func(th theme.Theme) []string { return []string{paint(popup(th).Foreground(th.Fg))} }},
		{"toast", false, func(m *Model) string {
			m.showToast("Copied")
			return m.renderToast(blankScreen(m.width, m.height))
		}, func(th theme.Theme) []string {
			return []string{
				paint(popup(th).Foreground(th.Green)) + "╭",              // border
				paint(popup(th).Foreground(th.Green)) + " \U000f018f ",   // icon
				paint(popup(th).Foreground(th.Fg)) + "Copied ",           // text
				paint(popup(th)) + paint(popup(th).Foreground(th.Green)), // the box fill around its content
			}
		}},
	} {
		t.Run(c.name+"/transparent", func(t *testing.T) {
			m := transparentModel(t, 120, 40)
			box := c.render(m)
			for _, w := range c.want(m.theme) {
				if !strings.Contains(box, w) {
					t.Errorf("missing %q in:\n%q", w, box)
				}
			}
			forbid := []string{popupSGR, bgSGR, statusSGR}
			if c.name == "theme picker" {
				// Its swatch chips show each theme's own colours, the
				// distinct theme's bg among them.
				forbid = []string{popupSGR, statusSGR}
			}
			for _, sgr := range forbid {
				if strings.Contains(box, sgr) {
					t.Errorf("a transparent float painted the fill %s:\n%q", sgr, box)
				}
			}
			if c.box {
				assertRectangle(t, box)
			}
		})
		t.Run(c.name+"/opaque", func(t *testing.T) {
			m := distinctModel(t, 120, 40, false)
			box := c.render(m)
			for _, w := range append(c.want(m.theme), popupSGR) {
				if !strings.Contains(box, w) {
					t.Errorf("missing %q in:\n%q", w, box)
				}
			}
		})
	}
}

// assertRectangle fails unless every row of box is equally wide and the
// box has a border, so ui.Overlay covers each cell under it.
func assertRectangle(t *testing.T, box string) {
	t.Helper()
	rows := strings.Split(box, "\n")
	w := ansi.StringWidth(rows[0])
	for i, r := range rows {
		if got := ansi.StringWidth(r); got != w {
			t.Errorf("row %d is %d wide, want %d: %q", i, got, w, ansi.Strip(r))
		}
	}
	if plain := ansi.Strip(box); !strings.ContainsAny(plain, "╭┌") {
		t.Errorf("the box has no border to set it apart:\n%s", plain)
	}
}

// TestNarrowExplorerOverlayFollowsTheTransparency: on a narrow window the
// explorer floats over the editor. Transparent, it paints no fill but still
// replaces the editor text under it, and keeps its divider; opaque, it
// fills with PopupBg.
func TestNarrowExplorerOverlayFollowsTheTransparency(t *testing.T) {
	for _, clear := range []bool{true, false} {
		m := distinctModel(t, 60, 20, clear)
		m.sidebarOpen = true
		m.updateLayout()
		if !m.explorerOverlay() {
			t.Fatal("setup: the explorer should float at this width")
		}
		under := strings.TrimSuffix(strings.Repeat(strings.Repeat("X", m.width)+"\n", m.height), "\n")
		out := m.drawExplorerOverlay(under)
		for _, sgr := range []string{popupSGR, bgSGR, statusSGR} {
			if clear && strings.Contains(out, sgr) {
				t.Errorf("transparent: the explorer painted the fill %s:\n%q", sgr, out)
			}
		}
		if !clear && !strings.Contains(out, popupSGR) {
			t.Errorf("opaque: the explorer lost its PopupBg fill:\n%q", out)
		}
		l := m.layout()
		w := m.explorerW(l)
		rows := strings.Split(ansi.Strip(out), "\n")
		for y := l.BodyY; y < l.BodyY+l.BodyH; y++ {
			cells := []rune(rows[y])
			if len(cells) != m.width {
				t.Fatalf("transparent=%v: row %d is %d cells, want %d: %q", clear, y, len(cells), m.width, rows[y])
			}
			if strings.ContainsRune(string(cells[:w]), 'X') {
				t.Errorf("transparent=%v: editor text shows through the explorer on row %d: %q", clear, y, rows[y])
			}
			if cells[w] != '│' {
				t.Errorf("transparent=%v: row %d lost the explorer divider: %q", clear, y, rows[y])
			}
		}
	}
}

// TestStatuslineAccentTextKeepsDarkFg: the mode badge and the folder and
// position chips draw DarkFg on their accent.
func TestStatuslineAccentTextKeepsDarkFg(t *testing.T) {
	m := transparentModel(t, 160, 40)
	th := m.theme
	label, color := m.modeBlock()
	bar := m.statusline()
	plain := lipgloss.NewStyle().Foreground(th.DarkFg)
	for _, w := range []string{
		onAccent(th, color) + label,
		paint(plain.Background(th.Red)) + "󰉋 ",
		paint(plain.Background(th.Green)) + "\ue612 ",
	} {
		if !strings.Contains(bar, w) {
			t.Errorf("missing %q in:\n%q", w, bar)
		}
	}
}

// transparentOn is a model on the distinct theme with ui.transparent set.
func transparentOn(t *testing.T) *Model {
	t.Helper()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	distinctTheme(t)
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New(store, WithTheme("distinct"), WithUI(config.UI{Transparent: true}))
	return &m
}

// The theme picker previews, keeps and restores themes transparent, and
// turning the option off afterwards paints the theme's own background:
// the picker must not leave the transparent copy as the theme to go back to.
func TestThemePickerKeepsTransparency(t *testing.T) {
	m := transparentOn(t)
	opaque, _ := theme.Get("distinct")

	m.openThemePicker()
	m.themes.move(m, 1)
	if m.theme.Bg != "" || m.theme.Name == "distinct" {
		t.Fatalf("the previewed theme %q should be transparent, Bg = %q", m.theme.Name, m.theme.Bg)
	}
	m.themes.close(m, false)
	if m.theme.Name != "distinct" || m.theme.Bg != "" {
		t.Fatalf("Esc should restore the transparent distinct theme, got %q with Bg %q", m.theme.Name, m.theme.Bg)
	}

	m.chooseSetting(settingTransparent)
	if m.theme.Bg != opaque.Bg || m.theme.DarkerBg != opaque.DarkerBg || m.theme.StatusBg != opaque.StatusBg {
		t.Fatalf("off after the picker should paint the theme's own fills, got %q %q %q",
			m.theme.Bg, m.theme.DarkerBg, m.theme.StatusBg)
	}

	m.chooseSetting(settingTransparent)
	m.openThemePicker()
	m.themes.move(m, 1)
	m.themes.close(m, true)
	if m.theme.Bg != "" {
		t.Fatalf("Enter should keep the picked theme transparent, Bg = %q", m.theme.Bg)
	}
}

// TestHeadingBandsFollowTheTransparency: with heading_background on, a
// transparent theme drops the band, like a Neovim theme with transparent
// floats, and keeps the heading's colour, bold and icon; the opaque theme
// keeps the band. Toggling the option in Settings applies at once.
func TestHeadingBandsFollowTheTransparency(t *testing.T) {
	// band(purple, bg) of the distinct theme: #a020e0 blended 18% into #101112.
	const band = "48;2;42;20;55"
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	distinctTheme(t)
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := New(store, WithTheme("distinct"), WithUI(config.UI{Transparent: true, HeadingBackground: true}))
	heading := func() string {
		m.preview.SetSize(40, 6)
		m.preview.SetPage(core.Page{ID: "a.md", Content: "## Sub\n"})
		return m.preview.View()
	}
	title := paint(lipgloss.NewStyle().Foreground(m.theme.Purple).Bold(true)) + "Sub"

	out := heading()
	if strings.Contains(out, band) {
		t.Errorf("transparent: the heading band is painted:\n%q", out)
	}
	if !strings.Contains(out, title) || !strings.Contains(out, "󰉬 ") {
		t.Errorf("transparent: the heading lost its colour, bold or icon:\n%q", out)
	}

	m.chooseSetting(settingTransparent) // off
	if out := heading(); !strings.Contains(out, band) {
		t.Errorf("opaque: the heading band is missing:\n%q", out)
	}
	m.chooseSetting(settingTransparent) // on again, live
	if out := heading(); strings.Contains(out, band) {
		t.Errorf("toggled on: the heading band is still painted:\n%q", out)
	}
}
