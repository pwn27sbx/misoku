package preview_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/charmbracelet/x/ansi"
)

// longProse is one paragraph long enough to wrap at every pane width.
var longProse = strings.Repeat("lorem ipsum dolor sit amet ", 40)

// wideCell is a cell wider than any prose page.
var wideCell = strings.Repeat("w", 130)

// wideTable is a three-column table whose natural width (about 150 columns)
// is above the prose width and below the wide width of a 250-column pane.
var wideTable = "| a | middle | z |\n|---|---|---|\n| 1 | " + wideCell + " | 9 |"

// codeLine is a code line of 130 columns.
var codeLine = strings.Repeat("c", 125) + "_END_"

// shownRows renders note in a preview pane w columns wide and returns its
// plain rows.
func shownRows(t *testing.T, w int, note string, notes preview.NoteLookup) []string {
	t.Helper()
	p := preview.New(theme.DefaultTheme())
	p.SetSize(w, 80)
	if notes != nil {
		p.SetNotes(notes)
	}
	p.SetPage(core.Page{ID: "p1", Content: note})
	return strings.Split(ansi.Strip(p.View()), "\n")
}

// shownRow returns the first row containing text, or fails.
func shownRow(t *testing.T, rows []string, text string) string {
	t.Helper()
	for _, r := range rows {
		if strings.Contains(r, text) {
			return r
		}
	}
	t.Fatalf("%q not rendered:\n%s", text, strings.Join(rows, "\n"))
	return ""
}

// leftMargin is the count of blank columns before the row's text.
func leftMargin(row string) int { return len(row) - len(strings.TrimLeft(row, " ")) }

// textWidth is the width of a row from the page margin to its last
// non-blank column.
func textWidth(row string, margin int) int {
	return ansi.StringWidth(strings.TrimRight(row, " ")) - margin
}

// S1: the prose page is 60 % of the pane, never less than 88 columns nor
// more than 120, and stays centred.
func TestProsePageGrowsWithThePane(t *testing.T) {
	cases := []struct{ w, margin, prose int }{
		{60, 2, 56},
		{88, 4, 80},
		{100, 6, 88},
		{148, 30, 88},
		{149, 30, 89},
		{200, 40, 120},
		{250, 65, 120},
		{400, 140, 120}, // the cap
	}
	for _, tc := range cases {
		if got := preview.PageMargin(tc.w); got != tc.margin {
			t.Errorf("PageMargin(%d) = %d, want %d", tc.w, got, tc.margin)
		}
		rows := shownRows(t, tc.w, longProse, nil)
		widest := 0
		for _, r := range rows {
			if strings.TrimSpace(r) == "" {
				continue
			}
			if m := leftMargin(r); m != tc.margin {
				t.Errorf("w=%d: row starts at column %d, want %d: %q", tc.w, m, tc.margin, r)
				break
			}
			widest = max(widest, textWidth(r, tc.margin))
		}
		// Words of 6 columns at most: the wrap fills the page to within 6.
		if widest > tc.prose || widest < tc.prose-6 {
			t.Errorf("w=%d: prose fills %d columns, want about %d", tc.w, widest, tc.prose)
		}
	}
}

// S2, S7: on a wide pane a table wider than the prose is drawn whole, from
// the page's left edge, and still truncated as before when it does not fit.
func TestWideTableUsesTheWideBudget(t *testing.T) {
	rows := shownRows(t, 250, wideTable, nil)
	row := shownRow(t, rows, wideCell)
	if strings.Contains(row, "…") {
		t.Fatalf("the table fits 250 columns and should not be cut: %q", row)
	}
	if m := leftMargin(row); m != 65 {
		t.Fatalf("the table should start at the page margin 65, got %d", m)
	}

	// At 140 columns the wide budget is 140 - 26 - 4 = 110: the table is
	// cut, but uses the right margin.
	rows = shownRows(t, 140, wideTable, nil)
	top := shownRow(t, rows, "┌")
	if got := textWidth(top, 26); got != 110 {
		t.Fatalf("at 140 the table should be 110 wide, got %d: %q", got, top)
	}
	if !strings.Contains(shownRow(t, rows, "www"), "…") {
		t.Fatalf("at 140 the wide cell should still be cut")
	}

	// At 80 columns wide and prose widths are both 72: cut as today.
	rows = shownRows(t, 80, wideTable, nil)
	if top := shownRow(t, rows, "┌"); textWidth(top, 4) != 72 {
		t.Fatalf("at 80 the table should be 72 wide: %q", top)
	}
	if !strings.Contains(shownRow(t, rows, "www"), "…") {
		t.Fatalf("at 80 the wide cell should be cut")
	}
}

// S2: a long code line widens its box up to the wide budget; a short code
// block keeps the prose-wide box.
func TestCodeBlockBoxWidth(t *testing.T) {
	long := "```\n" + codeLine + "\n```"
	rows := shownRows(t, 250, long, nil)
	if row := shownRow(t, rows, "_END_"); strings.Contains(row, "…") {
		t.Fatalf("a 130-column code line fits 250 columns: %q", row)
	}
	if top := shownRow(t, rows, "╭"); textWidth(top, 65) != 130+4 {
		t.Fatalf("the box should hug the 130-column line: %q", top)
	}

	rows = shownRows(t, 140, long, nil)
	if top := shownRow(t, rows, "╭"); textWidth(top, 26) != 110 {
		t.Fatalf("at 140 the box should take the wide width 110: %q", top)
	}
	if strings.Contains(strings.Join(rows, "\n"), "_END_") {
		t.Fatalf("at 140 the line should be cut")
	}

	short := "```\nfmt.Println(1)\n```"
	for _, w := range []int{80, 140, 250} {
		margin := preview.PageMargin(w)
		rows = shownRows(t, w, short, nil)
		if top := shownRow(t, rows, "╭"); textWidth(top, margin) != w-2*margin {
			t.Fatalf("w=%d: a short code box should be the prose width %d: %q", w, w-2*margin, top)
		}
	}
}

// S3: on a wide pane everything but tables and code stays at the prose
// width: paragraphs, headings, columns (even holding a wide table) and
// embeds (even of a wide table).
func TestProseBlocksStayAtProseWidth(t *testing.T) {
	other := "Inside the embed " + longProse + "\n\n" + wideTable
	notes := func(target string) (preview.Note, bool) {
		if target == "other" {
			return preview.Note{ID: "other", Title: "other", Content: other}, true
		}
		return preview.Note{}, false
	}
	note := "# " + longProse + "\n\n" + longProse +
		"\n\n~~~columns\n" + longProse + "\n+++\n" + wideTable + "\n~~~\n\n![[other]]\n\nend"
	rows := shownRows(t, 250, note, notes)
	shownRow(t, rows, "Inside the embed")
	for _, r := range rows {
		if strings.TrimSpace(r) == "" {
			continue
		}
		if got := textWidth(r, 65); got > 120 {
			t.Errorf("row is %d wide, past the prose width 120: %q", got, strings.TrimSpace(r))
		}
	}
}

// S5: up to 148 columns the page renders as before: the margin is
// (w - 88) / 2 or the minimum, and prose and a table that fits the page are
// exactly what a compile at the page width draws.
func TestNarrowAndMediumPagesUnchanged(t *testing.T) {
	note := "# Title\n\n" + longProse + "\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n- item " + longProse
	for _, w := range []int{50, 60, 100, 120, 140, 141, 148} {
		margin := preview.PageMargin(w)
		minimum := 2
		if w >= 70 {
			minimum = 4
		}
		if want := max((w-88)/2, minimum); margin != want {
			t.Fatalf("PageMargin(%d) = %d, want %d", w, margin, want)
		}
		want := strings.Split(ansi.Strip(preview.Compile(note, theme.DefaultTheme(), w-2*margin)), "\n")
		rows := shownRows(t, w, note, nil)
		for i, l := range want {
			got := strings.TrimRight(rows[i], " ")
			if l = strings.TrimRight(l, " "); l != "" {
				l = strings.Repeat(" ", margin) + l
			}
			if got != l {
				t.Fatalf("w=%d row %d:\n got %q\nwant %q", w, i, got, l)
			}
		}
	}
}
