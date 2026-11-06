package content_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// strikeNote has its cursor on line 0, so the tasks below it are drawn
// concealed.
const strikeNote = "cursor\n" +
	"- [x] done **bold** [[Note]]\n" +
	"- [X] also done\n" +
	"- [ ] open task\n" +
	"- [x] raw on cursor"

// fgAt is the 24-bit foreground in force at the first rune of text in row,
// as "r;g;b", or "" when there is none.
func fgAt(row, text string) string {
	plain := ansi.Strip(row)
	at := strings.Index(plain, text)
	if at < 0 {
		return ""
	}
	want := len([]rune(plain[:at]))
	sgr, n := "", 0
	for len(row) > 0 {
		if loc := sgrRe.FindStringSubmatchIndex(row); loc != nil && loc[0] == 0 {
			sgr, row = row[loc[0]:loc[1]], row[loc[1]:]
			continue
		}
		if n == want {
			fg, _ := colours(sgr)
			for k := range fg {
				return k
			}
			return ""
		}
		_, size := firstRune(row)
		row = row[size:]
		n++
	}
	return ""
}

func firstRune(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return 0, 0
}

func TestLiveStrikesCheckedTasks(t *testing.T) {
	th := theme.DefaultTheme()
	c := liveEditor(t, 60, strikeNote)
	styled, plain := rows(c)
	if plain[1] != "󰄲 done bold Note" || plain[2] != "󰄲 also done" || plain[3] != "󰄱 open task" {
		t.Fatalf("task rows changed text or width: %q", plain[1:4])
	}
	muted := rgb(th.GreyFg2)
	for _, tc := range []struct {
		row  int
		text string
	}{
		{1, "done"}, {1, "bold"}, {1, "Note"}, {2, "also"},
	} {
		if !styledWith(styled[tc.row], tc.text, "9") {
			t.Errorf("checked %q on row %d lacks strikethrough (SGR 9): %q", tc.text, tc.row, styled[tc.row])
		}
		if got := fgAt(styled[tc.row], tc.text); got != muted {
			t.Errorf("checked %q on row %d drawn in %q, want muted %q", tc.text, tc.row, got, muted)
		}
	}
	if !styledWith(styled[1], "bold", "1") {
		t.Errorf("bold inside a checked task lost its bold: %q", styled[1])
	}
	if styledWith(styled[1], "󰄲", "9") {
		t.Errorf("the checked box itself is struck: %q", styled[1])
	}
	if styledWith(styled[3], "open", "9") || fgAt(styled[3], "open") == muted {
		t.Errorf("unchecked task is struck or muted: %q", styled[3])
	}
}

func TestLiveCheckedTaskOnCursorLineStaysRaw(t *testing.T) {
	c := liveEditor(t, 60, strikeNote)
	c.GotoLine(5)
	styled, plain := rows(c)
	if !strings.HasPrefix(plain[4], "- [x] raw on cursor") {
		t.Fatalf("cursor line not raw: %q", plain[4])
	}
	if styledWith(styled[4], "raw", "9") {
		t.Errorf("checked task on the cursor line is struck: %q", styled[4])
	}
}

// TestLiveUncheckedTaskRendersAsBefore pins the styled row of an unchecked
// task to what the live view drew before checked tasks were struck.
func TestLiveUncheckedTaskRendersAsBefore(t *testing.T) {
	c := liveEditor(t, 60, strikeNote)
	styled, _ := rows(c)
	want := "\x1b[37m\x1b[37m\x1b[0m\x1b[0m\x1b[38;2;111;115;123m\U000f0131 \x1b[0m" +
		"\x1b[38;2;171;178;191mopen task\x1b[0m\x1b[37m" + strings.Repeat(" ", 48) + "\x1b[0m"
	if got := strings.TrimRight(styled[3], " "); got != want {
		t.Errorf("unchecked task row = %q, want %q", got, want)
	}
}
