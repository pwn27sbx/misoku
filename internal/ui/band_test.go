package ui_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/pwn27sbx/misoku/internal/kitty"
	"github.com/pwn27sbx/misoku/internal/ui"
)

const (
	bandColour = lipgloss.Color("#303030")
	bandSeq    = "48;2;48;48;48"
)

func trueColourProfile(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// bgCells returns, for each rune drawn in line, the background in effect
// ("" for the terminal's own). Picture diacritics count as their own runes.
func bgCells(line string) (runes []rune, bgs []string) {
	bg := ""
	for i := 0; i < len(line); {
		if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
			j := i + 2
			for j < len(line) && (line[j] < 0x40 || line[j] > 0x7e) {
				j++
			}
			if j < len(line) && line[j] == 'm' {
				p := strings.Split(line[i+2:j], ";")
				for k := 0; k < len(p); k++ {
					switch {
					case p[k] == "" || p[k] == "0" || p[k] == "49":
						bg = ""
					case p[k] == "48" && k+4 < len(p) && p[k+1] == "2":
						bg = strings.Join(p[k:k+5], ";")
						k += 4
					case p[k] == "38" && k+4 < len(p) && p[k+1] == "2":
						k += 4
					case p[k] == "38" && k+2 < len(p) && p[k+1] == "5":
						k += 2
					case p[k] == "7" || len(p[k]) == 2 && p[k][0] == '4' || len(p[k]) == 3 && p[k][:2] == "10":
						bg = p[k]
					}
				}
			}
			i = j + 1
			continue
		}
		r := []rune(line[i:])[0]
		runes = append(runes, r)
		bgs = append(bgs, bg)
		i += len(string(r))
	}
	return runes, bgs
}

// S15: every cell of a plain row takes the band, and the row is padded to
// the width with it.
func TestBandPaintsEveryCellAndPads(t *testing.T) {
	trueColourProfile(t)
	got := ui.Band("ab c", bandColour, 8)
	if w := ansi.StringWidth(got); w != 8 {
		t.Fatalf("width %d, want 8: %q", w, got)
	}
	runes, bgs := bgCells(got)
	for i, bg := range bgs {
		if bg != bandSeq {
			t.Fatalf("cell %d %q has background %q, want the band (%q)", i, runes[i], bg, got)
		}
	}
}

// S15: a reset inside the row does not end the band: it is put back.
func TestBandSurvivesAReset(t *testing.T) {
	trueColourProfile(t)
	row := "\x1b[1mab\x1b[0mcd\x1b[38;2;1;2;3mef\x1b[m gh"
	runes, bgs := bgCells(ui.Band(row, bandColour, 0))
	if string(runes) != "abcdef gh" {
		t.Fatalf("the text changed: %q", string(runes))
	}
	for i, bg := range bgs {
		if bg != bandSeq {
			t.Fatalf("cell %d %q lost the band after a reset: %q", i, runes[i], bg)
		}
	}
}

// S15: cells with a background of their own (code, a calendar's day) keep
// it; the band comes back once that background ends.
func TestBandSparesCellsWithTheirOwnBackground(t *testing.T) {
	trueColourProfile(t)
	row := "a\x1b[48;2;1;2;3mXY\x1b[49mb\x1b[44mZ\x1b[0mc\x1b[7mR\x1b[27md"
	runes, bgs := bgCells(ui.Band(row, bandColour, 0))
	want := map[rune]string{'a': bandSeq, 'X': "48;2;1;2;3", 'Y': "48;2;1;2;3", 'b': bandSeq, 'c': bandSeq, 'd': bandSeq}
	for i, r := range runes {
		if w, ok := want[r]; ok && bgs[i] != w {
			t.Fatalf("cell %q has background %q, want %q", r, bgs[i], w)
		}
		if (r == 'Z' || r == 'R') && bgs[i] == bandSeq {
			t.Fatalf("cell %q took the band over its own colour", r)
		}
	}
}

// S15: picture cells are left as the terminal draws them: no band behind the
// placeholders, which keep their diacritics and colour.
func TestBandNeverTouchesPictureCells(t *testing.T) {
	trueColourProfile(t)
	pic := kitty.Placeholders(7, 3, 1)[0]
	row := "ab" + pic + "cd"
	got := ui.Band(row, bandColour, 0)
	if ansi.Strip(got) != ansi.Strip(row) || strings.Count(got, "\x1b[38;5;7m") != 1 {
		t.Fatalf("the picture's cells changed: %q", got)
	}
	runes, bgs := bgCells(got)
	for i, r := range runes {
		switch {
		case r == 0x10EEEE && bgs[i] != "":
			t.Fatalf("a picture cell got the background %q", bgs[i])
		case (r == 'a' || r == 'd') && bgs[i] != bandSeq:
			t.Fatalf("text cell %q lost the band: %q", r, bgs[i])
		}
	}
}

// Without a colour profile there is nothing to paint: the row is returned
// as it was, so tests and plain terminals see no escape codes.
func TestBandWithoutColourIsANoOp(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
	if got := ui.Band("ab", bandColour, 5); got != "ab" {
		t.Fatalf("got %q, want the row untouched", got)
	}
}
