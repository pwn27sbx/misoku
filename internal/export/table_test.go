package export

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// layout builds markdown and returns the document with the drawing
// commands of each page, uncompressed. Font resources are left out: fpdf
// writes them in map order, so only the pages compare byte for byte.
func layout(t *testing.T, md string) (*doc, []string) {
	t.Helper()
	th, _ := theme.Get(printTheme)
	d := build(md, Options{Title: "T"}, th)
	d.pdf.SetCompression(false)
	var buf bytes.Buffer
	if err := d.pdf.Output(&buf); err != nil {
		t.Fatal(err)
	}
	// Page contents are the first streams fpdf writes, one per page.
	var pages []string
	rest := buf.String()
	for range d.pdf.PageCount() {
		_, after, ok := strings.Cut(rest, "stream\n")
		if !ok {
			t.Fatal("missing page stream")
		}
		page, tail, _ := strings.Cut(after, "endstream")
		pages, rest = append(pages, page), tail
	}
	return d, pages
}

// smallTables are tables of up to three columns, which must print exactly
// as they did before wide tables were handled.
const smallTables = "# Small\n\nIntro text.\n\n| Name | Value |\n| --- | --- |\n| alpha | 1 |\n| beta | a longer cell that wraps over more than one line because it keeps going and going and going on |\n\n" +
	"| Code | Date | Notes |\n| --- | --- | --- |\n| zen-723-kasalinda | 2026-08-26 09:35 | feature/warehouse-redesign and more words here |\n| x | y | supercalifragilisticexpialidociousandthensomemoreletters |\n\nAfter.\n"

func TestSmallTablesPrintAsBefore(t *testing.T) {
	_, pages := layout(t, smallTables)
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(pages, "\f"))))
	if got != smallTablesSum {
		t.Errorf("small tables changed: sha256 %s, want %s", got, smallTablesSum)
	}
}

// smallTablesSum is the sha256 of smallTables as printed before wide tables
// were handled (commit 277464b).
const smallTablesSum = "e6afc35ea061a778b57c745d39f2ceb4ea7f745607b77f485f5ccf46fdb0812a"

// table parses a Markdown table into rows the way blocks does.
func table(md string) [][]string {
	var rows [][]string
	for i, line := range strings.Split(strings.TrimSpace(md), "\n") {
		if i != 1 { // skip the separator
			rows = append(rows, splitRow(line))
		}
	}
	return rows
}

// words are the pieces a cell may never be broken inside: text between
// spaces, each ending after any "-" or "/".
func words(text string) []string {
	var out []string
	for _, f := range strings.Fields(text) {
		start := 0
		for i, r := range f {
			if r == '-' || r == '/' {
				out = append(out, f[start:i+1])
				start = i + 1
			}
		}
		if start < len(f) {
			out = append(out, f[start:])
		}
	}
	return out
}

// sixColumns has codes, dates and paths that today break letter by letter.
const sixColumns = `
| Worktree | Branch | Created | Last activity | Size | Notes |
| --- | --- | --- | --- | --- | --- |
| zen-283-pedido-de-shopify-permite-eliminar | zen-283-pedido-de-shopify-permite-eliminar-el-producto-principal-y | 2026-08-27 12:16 | 2026-10-07 08:57 | 1.1G | getCachedTopDepartments corta el top |
| warehouse-redesign | feature/warehouse-redesign | 2026-08-07 10:06 | 2026-08-24 17:15 | 1.1G | carpeta y rama con nombre distinto |
| fusionsubagentes | fusionsubagentes | 2026-08-07 11:06 | 2026-09-23 11:42 | 101M | Fusionsubagentes |
`

func TestWideTableWrapsBetweenWords(t *testing.T) {
	th, _ := theme.Get(printTheme)
	d := newDoc(Options{}, th)
	rows := table(sixColumns)
	lay := d.tableLayout(rows, 6)
	for ri, r := range rows {
		style := ""
		if ri == 0 {
			style = "B"
		}
		d.font("sans", style, lay.size)
		for c, cell := range r {
			room := lay.widths[c] - cellPad
			lines := d.wrapWords(cell, room)
			joined := "\n" + strings.Join(lines, "\n") + "\n"
			for _, w := range words(cell) {
				if d.pdf.GetStringWidth(w) > room {
					t.Errorf("column %d is %.1f mm, narrower than its word %q", c, room, w)
				}
				if !strings.Contains(strings.ReplaceAll(joined, " ", "\n"), w) {
					t.Errorf("row %d column %d: %q broken inside: %q", ri, c, w, lines)
				}
			}
			if got, want := strings.Join(lines, ""), strings.ReplaceAll(cell, " ", ""); strings.ReplaceAll(got, " ", "") != want {
				t.Errorf("row %d column %d: lines %q lose text of %q", ri, c, lines, cell)
			}
		}
	}
}

// longWords does not fit upright at the normal size, but does with smaller
// type (and would fit sideways at the normal size).
const longWords = `
| Responsabilidades | Departamentalizar | Internacionalizar | Desproporcionados | Inconstitucional | Electrocardiograma |
| --- | --- | --- | --- | --- | --- |
| cod-manager | 63 | dev | 12 | 30 | 15 |
`

func TestTableShrinksTypeBeforeTurningSideways(t *testing.T) {
	th, _ := theme.Get(printTheme)
	d := newDoc(Options{}, th)
	lay := d.tableLayout(table(longWords), 6)
	if lay.landscape || lay.size >= tableFont || lay.size < tableFontMin {
		t.Errorf("layout = %.1f pt landscape=%v, want smaller type on an upright page", lay.size, lay.landscape)
	}
}

// thirteenColumns is modelled on a real worktree report: too wide for an
// upright page even at the smallest type, and long enough for two pages.
func thirteenColumns() string {
	var b strings.Builder
	b.WriteString("| # | Creada | Worktree | Repo | Rama | Qué es | PR | Última actividad | Peso | Cambios sin commit | Sin subir | Abierto en herdr | Notas |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&b, "| %d | 2026-08-%02d 10:06 | `zen-%d-pedido-de-shopify` | cod-manager | `zen-%d-getcachedtopdepartments-corta-el-top-antes-de-agrupar` | ZEN-%d · Se borra getCachedTopDepartments, el ranking muerto que cortaba antes de agrupar | #%d OPEN | 2026-10-02 16:12 | 1.1G | 0 | 0 | — | carpeta y rama con nombre distinto |\n", i, i%28+1, 700+i, 700+i, 700+i, 1800+i)
	}
	return b.String()
}

// sideways reports whether page i (1-based) of d is sideways.
func sideways(d *doc, i int) bool {
	w, h, _ := d.pdf.PageSize(i)
	return w > h
}

func TestWideTableGetsItsOwnSidewaysPages(t *testing.T) {
	md := "# Report\n\nIntro text.\n\n" + thirteenColumns() + "\n## After\n\nClosing text.\n"
	d, pages := layout(t, md)
	n := len(pages)
	if n < 4 {
		t.Fatalf("got %d pages, want intro, two table pages and the closing page", n)
	}
	if sideways(d, 1) || sideways(d, n) {
		t.Errorf("first and last pages must stay upright")
	}
	for i := 2; i < n; i++ {
		if !sideways(d, i) {
			t.Errorf("table page %d is upright", i)
		}
		if got := strings.Count(pages[i-1], " re B"); got != 13 {
			t.Errorf("table page %d has %d header cells, want 13", i, got)
		}
		for _, f := range strings.Split(pages[i-1], " Tf")[:strings.Count(pages[i-1], " Tf")] {
			var size float64
			fields := strings.Fields(f)
			fmt.Sscan(fields[len(fields)-1], &size)
			if size < tableFontMin {
				t.Errorf("table page %d uses %.2f pt, below the %.1f pt minimum", i, size, tableFontMin)
			}
		}
	}
}

func TestSidewaysTableAtTheEndAddsNoBlankPage(t *testing.T) {
	d, pages := layout(t, "Intro.\n\n"+thirteenColumns())
	if n := len(pages); !sideways(d, n) {
		t.Errorf("last of %d pages is upright: a blank page was added after the table", n)
	}
}

func TestHeadingAboveSidewaysTableTurnsWithIt(t *testing.T) {
	md := "# Report\n\nIntro text.\n\n## Table\n\n" + thirteenColumns() + "\n## Second\n\n" + thirteenColumns()
	d, pages := layout(t, md)
	h2 := fmt.Sprintf("%.2f Tf", headingSizes[2])
	for i := 2; i <= len(pages); i++ {
		if !sideways(d, i) {
			t.Errorf("page %d of %d is upright: a heading was left alone before its table", i, len(pages))
		}
	}
	if strings.Contains(pages[0], h2) {
		t.Errorf("the heading stayed on the upright page, away from its table")
	}
}

// Text that breaks only at spaces wraps exactly as fpdf's SplitText wrapped
// every table before, down to a line exactly as wide as its room.
func TestWrapWordsMatchesSplitTextBetweenWords(t *testing.T) {
	th, _ := theme.Get(printTheme)
	d := newDoc(Options{}, th)
	d.font("sans", "", tableFont)
	text := "carpeta y rama con nombre distinto, nueve archivos sin commit"
	for _, room := range []float64{
		d.pdf.GetStringWidth(text),
		d.pdf.GetStringWidth("carpeta y rama"),
		d.pdf.GetStringWidth("carpeta y rama con nombre"),
		25, 40, 60,
	} {
		got, want := d.wrapWords(text, room), d.pdf.SplitText(text, room)
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("room %.2f mm: lines %q, want %q", room, got, want)
		}
	}
}
