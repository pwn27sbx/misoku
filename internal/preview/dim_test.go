package preview_test

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// trueColour renders in 24-bit colour for the rest of the test.
func trueColour(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// styledCell is one terminal cell: its rune and colours ("#rrggbb", "" unset).
type styledCell struct {
	r      rune
	fg, bg string
}

// styledCells splits a rendered line into cells, following the SGR colour
// state the way a terminal does.
func styledCells(line string) []styledCell {
	var out []styledCell
	var fg, bg string
	for i := 0; i < len(line); {
		if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
			j := i + 2
			for j < len(line) && (line[j] < 0x40 || line[j] > 0x7e) {
				j++
			}
			if j < len(line) && line[j] == 'm' {
				p := strings.Split(line[i+2:j], ";")
				for k := 0; k < len(p); k++ {
					switch p[k] {
					case "", "0":
						fg, bg = "", ""
					case "39":
						fg = ""
					case "49":
						bg = ""
					case "38", "48":
						if k+4 < len(p) && p[k+1] == "2" {
							var c [3]int
							for n := range c {
								c[n], _ = strconv.Atoi(p[k+2+n])
							}
							h := fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])
							if p[k] == "38" {
								fg = h
							} else {
								bg = h
							}
							k += 4
						}
					}
				}
			}
			i = j + 1
			continue
		}
		r, n := utf8.DecodeRuneInString(line[i:])
		out = append(out, styledCell{r, fg, bg})
		i += n
	}
	return out
}

// blockGlyph reports the shapes that carry a block's colour: picture and
// chart half blocks, callout and quote bars, board dots.
func blockGlyph(r rune) bool { return r >= 0x2580 && r <= 0x25ff }

// dimmablePreview shows a note with text, a callout, a board, a bar chart
// and a picture.
func dimmablePreview(t *testing.T) preview.Model {
	t.Helper()
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 20, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.RGBA{uint8(200 - x*5), 40, uint8(y * 30), 255})
		}
	}
	f, err := os.Create(filepath.Join(dir, "pic.png"))
	if err != nil {
		t.Fatal(err)
	}
	_ = png.Encode(f, img)
	f.Close()

	md := "# Title\n\nplain **bold** text\n\n> [!WARNING]\n> careful\n\n" +
		"```board\n## Todo\n- A\n## Done\n- B\n```\n\n" +
		"```chart\ntype: bar\nJan: 3\nFeb: 6\n```\n\n![pic](pic.png)\n"
	p := preview.New(theme.DefaultTheme())
	p.SetSize(60, 70)
	p.SetBaseDir(dir)
	p.SetPage(core.Page{ID: "n.md", Title: "n", Content: md})
	return p
}

func TestPreviewSetDimmedGreysTextKeepsBlockColours(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	grey := styledCells(lipgloss.NewStyle().Foreground(th.Grey).Render("x"))[0].fg
	p := dimmablePreview(t)
	lit := p.View()
	p.SetDimmed(true)
	dim := p.View()
	if ansi.Strip(dim) != ansi.Strip(lit) {
		t.Fatalf("dimming changed the text:\n%s", ansi.Strip(dim))
	}
	litRows, dimRows := strings.Split(lit, "\n"), strings.Split(dim, "\n")
	var kept, greyed, litText int
	for y := range litRows {
		a, b := styledCells(litRows[y]), styledCells(dimRows[y])
		for x := range a {
			switch {
			case a[x].bg != b[x].bg:
				t.Fatalf("row %d col %d %q: background %s became %s", y, x, a[x].r, a[x].bg, b[x].bg)
			case blockGlyph(a[x].r) && a[x].fg != "":
				if b[x].fg != a[x].fg {
					t.Fatalf("row %d col %d %q: block colour %s became %s", y, x, a[x].r, a[x].fg, b[x].fg)
				}
				kept++
			case a[x].r != ' ':
				if a[x].fg != grey {
					litText++
				}
				if b[x].fg != grey {
					t.Fatalf("row %d col %d %q: text is %s, want grey %s", y, x, a[x].r, b[x].fg, grey)
				}
				greyed++
			}
		}
	}
	if kept < 100 || greyed < 20 || litText < 20 {
		t.Fatalf("too little to tell: %d block cells kept, %d text cells greyed, %d lit:\n%s", kept, greyed, litText, ansi.Strip(lit))
	}
}

// Undimmed, the preview renders exactly as it did before dimming existed.
func TestPreviewUndimmedRendersAsBefore(t *testing.T) {
	trueColour(t)
	p := dimmablePreview(t)
	before := p.View()
	p.SetDimmed(true)
	_ = p.View()
	p.SetDimmed(false)
	if p.View() != before {
		t.Error("SetDimmed(false) does not restore the lit rendering")
	}
}
