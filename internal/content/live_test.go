package content_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"
)

const liveNote = "cursor line **raw**\n" +
	"# Heading\n" +
	"some **bold** and *it* and `code`\n" +
	"see [text](http://x) and [[Note|alias]] and [[Other]]\n" +
	"- [ ] todo\n" +
	"- [x] done\n" +
	"- item\n" +
	"> quote\n" +
	"```\n" +
	"**inside** [[Note]]\n" +
	"```\n" +
	"| a | **b** |"

// liveEditor is a lit editor on text, w columns wide, with the live view on.
func liveEditor(t *testing.T, w int, text string) content.Model {
	t.Helper()
	trueColour(t)
	c := content.New(theme.DefaultTheme())
	c.SetSize(w, 20)
	c.SetPage(core.Page{ID: "n.md", Title: "n", Content: text})
	c.SetLive(true)
	return c
}

// rows is the View's screen rows: styled, and with escapes stripped and
// trailing blanks trimmed.
func rows(c content.Model) (styled, plain []string) {
	styled = strings.Split(c.ConcealedView(), "\n")
	for _, r := range styled {
		plain = append(plain, strings.TrimRight(ansi.Strip(r), " "))
	}
	return styled, plain
}

func TestLiveConcealsMarkersOffTheCursorLine(t *testing.T) {
	c := liveEditor(t, 60, liveNote)
	if !c.Live() {
		t.Fatal("Live() = false after SetLive(true)")
	}
	_, got := rows(c)
	want := []string{
		"cursor line **raw**",
		"Heading",
		"some bold and it and code",
		"see text and alias and Other",
		"󰄱 todo",
		"󰄲 done",
		"• item",
		"▎ quote",
		"```",
		"**inside** [[Note]]",
		"```",
		"| a | **b** |",
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("row %d = %q, want %q", i, got[i], w)
		}
	}
}

func TestLiveStylesWhatItHides(t *testing.T) {
	c := liveEditor(t, 60, liveNote)
	styled, _ := rows(c)
	for _, tc := range []struct {
		row  int
		text string
		attr string
		what string
	}{
		{1, "Heading", "1", "bold heading"},
		{2, "bold", "1", "bold text"},
		{2, "it", "3", "italic text"},
		{3, "text", "4", "underlined link"},
		{3, "alias", "4", "underlined wiki link"},
	} {
		if !styledWith(styled[tc.row], tc.text, tc.attr) {
			t.Errorf("%s: %q on row %d lacks SGR %s in %q", tc.what, tc.text, tc.row, tc.attr, styled[tc.row])
		}
	}
	if styledWith(styled[2], "some", "1") {
		t.Errorf("plain text came out bold: %q", styled[2])
	}
}

// styledWith reports whether the first rune of text in row is drawn under
// an SGR sequence carrying attribute attr (1 bold, 3 italic, 4 underline).
func styledWith(row, text, attr string) bool {
	var plain strings.Builder
	var params []string // the SGR in force for each rune of plain
	sgr := ""
	for len(row) > 0 {
		if loc := sgrRe.FindStringSubmatchIndex(row); loc != nil && loc[0] == 0 {
			sgr, row = row[loc[2]:loc[3]], row[loc[1]:]
			continue
		}
		r, n := utf8.DecodeRuneInString(row)
		plain.WriteRune(r)
		params = append(params, sgr)
		row = row[n:]
	}
	at := strings.Index(plain.String(), text)
	if at < 0 {
		return false
	}
	for _, p := range strings.Split(params[utf8.RuneCountInString(plain.String()[:at])], ";") {
		if p == attr {
			return true
		}
	}
	return false
}

func TestLiveCursorLineStaysRaw(t *testing.T) {
	c := liveEditor(t, 60, liveNote)
	for line, want := range map[int]string{
		2: "# Heading",
		3: "some **bold** and *it* and `code`",
		5: "- [ ] todo",
		8: "> quote",
	} {
		c.GotoLine(line)
		_, got := rows(c)
		if got[line-1] != want {
			t.Errorf("cursor on line %d shows %q, want the raw %q", line, got[line-1], want)
		}
	}
}

func TestLiveKeepsFrontMatterRaw(t *testing.T) {
	c := liveEditor(t, 60, "---\ntitle: **x**\n- yaml\n---\n# Body")
	c.GotoLine(5)
	_, got := rows(c)
	for i, w := range []string{"---", "title: **x**", "- yaml", "---", "# Body"} {
		if got[i] != w {
			t.Errorf("row %d = %q, want %q", i, got[i], w)
		}
	}
}

// A wrapped line keeps its style on every row, and keeps the rows its raw
// text wraps to.
func TestLiveWrappedLineKeepsStyle(t *testing.T) {
	c := liveEditor(t, 21, "top\n**alpha beta gamma delta**\nend")
	styled, got := rows(c)
	if got[1] != "alpha beta gamma" || got[2] != "delta" || got[3] != "end" {
		t.Fatalf("rows = %q, want the bold line on rows 1-2 and end on 3", got[:4])
	}
	if !styledWith(styled[2], "delta", "1") {
		t.Errorf("second row lost the bold: %q", styled[2])
	}

	// The space dropped at the break is plain: the bold starts right after.
	c = liveEditor(t, 21, "top\nplain words here **boldword**\nend")
	styled, got = rows(c)
	if got[1] != "plain words here" || got[2] != "boldword" {
		t.Fatalf("rows = %q, want the line broken before boldword", got[:3])
	}
	if !styledWith(styled[2], "boldword", "1") {
		t.Errorf("boldword after the break is not bold: %q", styled[2])
	}
}

func TestLiveHidesLineNumbers(t *testing.T) {
	c := liveEditor(t, 60, liveNote)
	if _, got := rows(c); !strings.HasPrefix(got[0], "cursor") {
		t.Errorf("row 0 = %q, want no line number", got[0])
	}
}

// With the switch off, the editor draws exactly what it drew before the
// live view existed: same rows, same numbers, same escapes.
func TestLiveOffRendersAsBefore(t *testing.T) {
	trueColour(t)
	before := content.New(theme.DefaultTheme())
	before.SetSize(60, 20)
	before.SetPage(core.Page{ID: "n.md", Title: "n", Content: liveNote})
	want := before.View()

	c := liveEditor(t, 60, liveNote)
	_ = c.View()
	c.SetLive(false)
	if c.Live() {
		t.Fatal("Live() = true after SetLive(false)")
	}
	if got := c.View(); got != want {
		t.Errorf("View after SetLive(false) differs from before:\n%q\nwant\n%q", got, want)
	}
}

// Dimmed, the live view stays off: the grey editor draws every line raw.
func TestLiveDimmedDrawsRaw(t *testing.T) {
	c := liveEditor(t, 60, liveNote)
	c.SetDimmed(true)
	if _, got := rows(c); got[1] != "# Heading" {
		t.Errorf("dimmed row 1 = %q, want the raw %q", got[1], "# Heading")
	}
}
