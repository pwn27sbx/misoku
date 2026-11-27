// Package rich renders a note as a page of pixels, sent to the terminal as
// Kitty tiles behind Unicode placeholders. It is the reading view that beats
// a character renderer: real fonts and heading sizes, drawn maths and
// diagrams, images in the page, and the note still one key away from editing.
//
// The pipeline has one seam per stage: Doc (doc.go) is the parsed note,
// Page (layout.go) is the display list in cell units, RenderTile
// (raster.go) draws pixels for a slice of it, and View (view.go) shows the
// tiles as placeholder cells and owns the terminal side.
package rich

import (
	"image/color"
	"strings"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// Kind is what a block draws as.
type Kind int

const (
	KindParagraph Kind = iota
	KindHeading
	KindListItem
	KindQuote
	KindCode
	KindRule
	KindTable
	KindImage
	KindProperties // the note's front matter, as a table of its top-level keys
	KindMath       // a $$…$$ or ```math block: a sized box, drawn by the math engine
	KindDiagram    // a ```mermaid or ```flow block: a sized box, drawn by the diagram engine
)

// Block is one top-level part of a note, with the source lines it came from
// (0-based, End exclusive) so the view can keep the reader's place and the
// editable reading view can be opened at the same line.
type Block struct {
	Kind   Kind
	Level  int      // heading level, 1–6
	Lines  []string // logical text lines, already flattened from inline Markdown
	Code   []string // code, math or diagram source lines
	Lang   string   // fence language
	Rows   [][]string
	Src    string // image source
	Line   int
	End    int
	Blocks []Block // list items and quote contents
}

// Document is a parsed note.
type Document struct {
	Blocks []Block
}

// Palette is the page's colours, resolved from the app theme once per
// compile. A zero colour means "draw nothing", which keeps ui.transparent
// working: the page's background is simply not painted.
type Palette struct {
	Bg      color.RGBA
	Fg      color.RGBA
	Muted   color.RGBA
	Link    color.RGBA
	CodeBg  color.RGBA
	Stripe  color.RGBA // every other table row
	Rule    color.RGBA
	Heading [6]color.RGBA
	Alert   [5]color.RGBA // NOTE, TIP, IMPORTANT, WARNING, CAUTION
}

// PaletteFromTheme maps the app's base46 theme to page colours. The accents
// cycle through the heading levels so a page reads like the text compiler's
// themed headings.
func PaletteFromTheme(th theme.Theme) Palette {
	p := Palette{
		Bg:     hexColor(string(th.Bg)),
		Fg:     hexColor(string(th.Fg)),
		Muted:  hexColor(string(th.GreyFg)),
		Link:   hexColor(string(th.Blue)),
		CodeBg: hexColor(string(th.Bg2)),
		Stripe: hexColor(string(th.OneBg)),
		Rule:   hexColor(string(th.Line)),
	}
	heads := []string{string(th.Blue), string(th.Purple), string(th.Green), string(th.Yellow), string(th.Orange), string(th.Red)}
	for i, c := range heads {
		p.Heading[i] = hexColor(c)
	}
	alerts := []string{string(th.Blue), string(th.Green), string(th.Purple), string(th.Yellow), string(th.Red)}
	for i, c := range alerts {
		p.Alert[i] = hexColor(c)
	}
	return p
}

// hexColor turns one of the theme's lipgloss colours ("#rrggbb", or "" for a
// transparent surface) into RGBA. Unknown strings keep the zero colour, which
// draws nothing.
func hexColor(c string) color.RGBA {
	s := strings.TrimSpace(c)
	if len(s) != 7 || s[0] != '#' {
		return color.RGBA{}
	}
	var v uint32
	for i := 1; i < 7; i++ {
		d := s[i]
		switch {
		case d >= '0' && d <= '9':
			v = v<<4 | uint32(d-'0')
		case d >= 'a' && d <= 'f':
			v = v<<4 | uint32(d-'a'+10)
		case d >= 'A' && d <= 'F':
			v = v<<4 | uint32(d-'A'+10)
		default:
			return color.RGBA{}
		}
	}
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}
