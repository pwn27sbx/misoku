package textarea_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/pwn27sbx/misoku/internal/textarea"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// selectionGolden holds the sha256 of View for each selection shape. The
// hashes were recorded from the painter before it stopped copying the model
// per rune, so any change to the painted bytes fails here.
var selectionGolden = map[string]string{
	"normal":   "2b09b03bf85b7c2d95f097e8569eeed7ecca4e7f27b9c3c5f77db4d25b9755c6",
	"linewise": "03f8d3f3fa43b56cd91a3e4d0e5c0862d9b137bc210bd69a9b8472df6dd06807",
	"oneline":  "ea6c289a39958603e1171aaaa02ab36190f692e0b88099ba4b7b7c7099f41277",
	"inverted": "929b4f9dc1fe6304f0f15933be900f4bb912c7877d44fb69d50ff979c9fcdb8b",
}

// selectionEditor returns a small colored editor whose text has a soft-wrapped
// line and wide CJK runes, with a visible selection style and per-rune colors
// on the first two lines.
func selectionEditor() textarea.Model {
	m := textarea.New()
	m.SetWidth(24)
	m.SetHeight(8)
	m.SetValue("alpha beta gamma\n漢字 wide runes here\nthis line is long enough that it soft-wraps twice over\nlast line")
	m.SelectionStyle = lipgloss.NewStyle().Background(lipgloss.Color("4")).Foreground(lipgloss.Color("15"))
	red, green := lipgloss.Color("1"), lipgloss.Color("2")
	m.LineColors = [][]lipgloss.Color{
		{red, red, red, red, red, "", green, green, green, green},
		{"", green, green},
	}
	m.Focus()
	m.GotoTop()
	return m
}

func TestViewSelectionGolden(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	cases := map[string]func(m *textarea.Model){
		"normal": func(m *textarea.Model) {
			m.SetAnchor(0, 3)
			m.SetRowCol(2, 30)
		},
		"linewise": func(m *textarea.Model) {
			m.SelectLinewise = true
			m.SetAnchor(1, 2)
			m.SetRowCol(2, 4)
		},
		"oneline": func(m *textarea.Model) {
			m.SetAnchor(0, 2)
			m.SetRowCol(0, 8)
		},
		"inverted": func(m *textarea.Model) {
			m.SetAnchor(2, 40)
			m.SetRowCol(0, 7)
		},
	}
	plain := selectionEditor().View()
	for name, set := range cases {
		t.Run(name, func(t *testing.T) {
			m := selectionEditor()
			set(&m)
			v := m.View()
			if v == plain {
				t.Fatal("selection is not painted: view equals the unselected view")
			}
			sum := sha256.Sum256([]byte(v))
			if got := hex.EncodeToString(sum[:]); got != selectionGolden[name] {
				t.Errorf("view hash = %s, want %s\nview:\n%q", got, selectionGolden[name], v)
			}
		})
	}
}
