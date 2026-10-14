package preview

import (
	"image/color"
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

// Look holds the configurable glyphs of the preview (the "ui" section of
// config.json). Empty fields fall back to DefaultLook.
type Look struct {
	// HeadingIcons go before headings of level 1 to 6.
	HeadingIcons [6]string
	// HeadingBackground paints each heading on a band of its own color that
	// runs the width of the pane (render-markdown's "full" style); the H1
	// underline is then left out.
	HeadingBackground bool
	// Bullets are used by depth, cycling.
	Bullets []string
	// Unchecked and Checked mark "- [ ]" and "- [x]".
	Unchecked, Checked string
	// Todo marks "- [-]" as in progress. Empty keeps "[-]" as plain text,
	// since Markdown has no such checkbox.
	Todo string
}

// DefaultLook is Misoku's own look.
func DefaultLook() Look {
	return Look{
		HeadingIcons: [6]string{"󰉫 ", "󰉬 ", "󰉭 ", "󰉮 ", "󰉯 ", "󰉰 "},
		Bullets:      []string{"●", "○", "◆", "◇"},
		Unchecked:    "󰄱",
		Checked:      "󰄲",
	}
}

// withDefaults fills empty fields from DefaultLook.
func (l Look) withDefaults() Look {
	d := DefaultLook()
	for i := range l.HeadingIcons {
		if l.HeadingIcons[i] == "" {
			l.HeadingIcons[i] = d.HeadingIcons[i]
		}
	}
	if len(l.Bullets) == 0 {
		l.Bullets = d.Bullets
	}
	if l.Unchecked == "" {
		l.Unchecked = d.Unchecked
	}
	if l.Checked == "" {
		l.Checked = d.Checked
	}
	return l
}

// SetLook changes the preview's glyphs.
func (m *Model) SetLook(l Look) {
	m.look = l.withDefaults()
	m.recompile()
}

// band returns the background of a heading band: the heading color blended
// into the page background, so it reads as a tint of that color.
func band(fg, bg lipgloss.Color) lipgloss.Color {
	if !isHex(string(fg)) || !isHex(string(bg)) {
		return bg
	}
	f, b := parseHex(string(fg)), parseHex(string(bg))
	const amount = 0.18
	mix := func(x, y uint8) uint8 { return uint8(float64(x)*(1-amount) + float64(y)*amount + 0.5) }
	return lipgloss.Color(hexColor(color.RGBA{R: mix(b.R, f.R), G: mix(b.G, f.G), B: mix(b.B, f.B), A: 255}))
}

func isHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(s[1:], 16, 32)
	return err == nil
}
