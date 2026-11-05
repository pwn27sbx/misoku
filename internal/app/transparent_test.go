package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })
	th := distinctTheme(t).Transparent()
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

// TestFloatingBoxesStayOpaqueOnATransparentTheme: every box drawn over the
// panes keeps its PopupBg fill when the panes go clear, and text drawn on
// an accent keeps DarkFg instead of the terminal's default colour.
func TestFloatingBoxesStayOpaqueOnATransparentTheme(t *testing.T) {
	for _, c := range []struct {
		name   string
		render func(m *Model) string
		// want lists what the box must contain, each built from m.theme.
		want func(th theme.Theme) []string
	}{
		{"confirm dialog", func(m *Model) string {
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
		{"save-as dialog", func(m *Model) string {
			m.openSaveAs(&buffer{}, nil)
			box, _, _ := m.saveAs.view(m.theme, 80, 24)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + "Save As"}
		}},
		{"menu", func(m *Model) string {
			m.menuBox = newMenu("Pick one", []string{"first", "second"}, nil)
			box, _, _ := m.menuBox.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + " Pick one"}
		}},
		{"palette", func(m *Model) string {
			m.openPalette()
			box, _, _ := m.palette.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + "MISOKU KEYMAPS & COMMANDS"}
		}},
		{"finder", func(m *Model) string {
			m.openFinder()
			box, _, _ := m.finder.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + "Find Note"}
		}},
		{"theme picker", func(m *Model) string {
			m.openThemePicker()
			box, _, _ := m.themes.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{
				paint(popup(th).Foreground(th.Fg).Bold(true)) + "Themes",    // title
				popup(th).Render("  ") + paint(popup(th).Foreground(th.Fg)), // an unselected row's marker, then its name
			}
		}},
		{"finder result rows", func(m *Model) string {
			m.openFinder()
			return m.finder.resultRow(m.theme, 1, 40) + "|" + m.finder.resultRow(m.theme, 99, 40)
		}, func(th theme.Theme) []string {
			return []string{
				popup(th).Render("  "),                    // an unselected row
				popup(th).Render(strings.Repeat(" ", 40)), // an empty row
			}
		}},
		{"finder preview", func(m *Model) string {
			return previewLine(m.theme, "plain text", 30)
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.GreyFg2)) + "plain text"}
		}},
		{"date picker", func(m *Model) string {
			m.pickDate("When", time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC), time.Time{}, nil)
			box, _, _ := m.datePick.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{"48;2;10;11;12", paint(popup(th).Foreground(th.GreyFg2)) + " Wed, 18 Nov 2026", onAccent(th, th.Blue) + " 18 "}
		}},
		{"media browser", func(m *Model) string {
			m.openFileBrowser("image")
			box, _, _ := m.browser.view(m)
			return box
		}, func(th theme.Theme) []string { return []string{paint(popup(th).Foreground(th.Fg).Bold(true))} }},
		{"prompt", func(m *Model) string {
			m.prompt("Rename", "value", "a hint", nil)
			box, _, _ := m.promptBox.view(m)
			return box
		}, func(th theme.Theme) []string {
			return []string{paint(popup(th).Foreground(th.Fg).Bold(true)) + "Rename"}
		}},
		{"leader guide", func(m *Model) string {
			return m.renderLeaderGuide()
		}, func(th theme.Theme) []string { return []string{paint(popup(th).Foreground(th.Fg))} }},
		{"toast", func(m *Model) string {
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
		t.Run(c.name, func(t *testing.T) {
			m := transparentModel(t, 120, 40)
			box := c.render(m)
			for _, w := range c.want(m.theme) {
				if !strings.Contains(box, w) {
					t.Errorf("missing %q in:\n%q", w, box)
				}
			}
		})
	}
}

// TestNarrowExplorerOverlayStaysOpaque: on a narrow window the explorer
// floats over the editor, so it fills with PopupBg like the other boxes.
func TestNarrowExplorerOverlayStaysOpaque(t *testing.T) {
	m := transparentModel(t, 60, 20)
	m.sidebarOpen = true
	m.updateLayout()
	if !m.explorerOverlay() {
		t.Fatal("setup: the explorer should float at this width")
	}
	out := m.drawExplorerOverlay(blankScreen(m.width, m.height))
	if !strings.Contains(out, paint(popup(m.theme))) {
		t.Errorf("the floating explorer should fill with PopupBg:\n%q", out)
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
