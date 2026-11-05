package preview_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// numbersLook is the Numbers preset with outline numbering on or off.
func numbersLook(on bool) preview.Look {
	look := preview.DefaultLook()
	look.HeadingIcons = [6]string{"① ", "② ", "③ ", "④ ", "⑤ ", "⑥ "}
	look.HeadingNumbers = on
	return look
}

// numbered renders content in a preview with look and returns the plain
// text and the model, for tests that fold.
func numbered(t *testing.T, look preview.Look, content string, notes preview.NoteLookup) (string, *preview.Model) {
	t.Helper()
	m := preview.New(theme.DefaultTheme())
	m.SetLook(look)
	if notes != nil {
		m.SetNotes(notes)
	}
	m.SetSize(60, 40)
	m.SetPage(core.Page{ID: "n.md", Title: "n", Content: content})
	return ansi.Strip(m.View()), &m
}

func wantHeadings(t *testing.T, plain string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(plain, "▾ "+w) && !strings.Contains(plain, "▸ "+w) {
			t.Errorf("preview should show heading %q:\n%s", w, plain)
		}
	}
}

func TestHeadingNumbersReplaceTheLevelIcon(t *testing.T) {
	md := "## Pendiente\n\n### Plan de implementación\n\n## Notas\n"
	plain, _ := numbered(t, numbersLook(true), md, nil)
	wantHeadings(t, plain, "1 Pendiente", "1.1 Plan de implementación", "2 Notas")
	if strings.ContainsAny(plain, "②③") {
		t.Errorf("the number replaces the level icon:\n%s", plain)
	}
}

func TestHeadingNumbersSkipLevelsWithoutZeros(t *testing.T) {
	plain, _ := numbered(t, numbersLook(true), "## A\n\n#### B\n\n### C\n", nil)
	wantHeadings(t, plain, "1 A", "1.1 B", "1.2 C")
}

func TestHeadingNumbersShallowerHeadingLaterIsTopLevel(t *testing.T) {
	plain, _ := numbered(t, numbersLook(true), "### A\n\n## B\n", nil)
	wantHeadings(t, plain, "1 A", "2 B")
}

func TestHeadingNumbersIgnoreCodeMathAndTables(t *testing.T) {
	md := "```\n## Fenced\n```\n\n$$\n## Math\n$$\n\n## a | b\n|---|---|\n| 1 | 2 |\n\n## ^anchor\n\n## B\n"
	plain, _ := numbered(t, numbersLook(true), md, nil)
	wantHeadings(t, plain, "1 B")
	for _, bad := range []string{"1 Fenced", "1 Math", "1 a", "2 B"} {
		if strings.Contains(plain, bad) {
			t.Errorf("only real headings take a number, found %q:\n%s", bad, plain)
		}
	}
}

func TestHeadingNumbersStayWhenASectionFolds(t *testing.T) {
	plain, m := numbered(t, numbersLook(true), "## A\n\n### A1\n\n## B\n\n### B1\n", nil)
	wantHeadings(t, plain, "1 A", "1.1 A1", "2 B", "2.1 B1")
	folded := false
	for y, l := range strings.Split(plain, "\n") {
		if !strings.Contains(l, "1 A") || strings.Contains(l, "A1") {
			continue
		}
		for x := 0; x < 10 && !folded; x++ {
			if h, ok := m.HitAt(x, y); ok && h.Kind == "fold" {
				m.ToggleFold(h.Arg)
				folded = true
			}
		}
		break
	}
	if !folded {
		t.Fatalf("no fold hit on the A heading:\n%s", plain)
	}
	plain = ansi.Strip(m.View())
	wantHeadings(t, plain, "1 A", "2 B", "2.1 B1")
	if strings.Contains(plain, "A1") {
		t.Errorf("A1 should be folded away:\n%s", plain)
	}
}

func TestHeadingNumbersOffKeepsTheIcons(t *testing.T) {
	md := "# Title\n\n## Sub\n\n### Deep\n"
	off, _ := numbered(t, numbersLook(false), md, nil)
	wantHeadings(t, off, "① Title", "② Sub", "③ Deep")
	if strings.Contains(off, "1 Sub") || strings.Contains(off, "1.1 Deep") {
		t.Errorf("numbering off renders the icons as before:\n%s", off)
	}
}

func TestHeadingNumbersLeaveEmbedsWithTheirIcons(t *testing.T) {
	notes := func(target string) (preview.Note, bool) {
		return preview.Note{ID: target + ".md", Title: target, Content: "## Inside\n\nText.\n"}, target == "Other"
	}
	plain, _ := numbered(t, numbersLook(true), "## Own\n\n![[Other]]\n", notes)
	wantHeadings(t, plain, "1 Own")
	if !strings.Contains(plain, "② Inside") || strings.Contains(plain, "2 Inside") || strings.Contains(plain, "1.1 Inside") {
		t.Errorf("an embedded heading keeps its level icon:\n%s", plain)
	}
}

func TestHeadingNumbersLeaveALoneTitleUnnumbered(t *testing.T) {
	plain, _ := numbered(t, numbersLook(true), "# Title\n\n## A\n\n## B\n", nil)
	wantHeadings(t, plain, "① Title", "1 A", "2 B")
	plain, _ = numbered(t, numbersLook(true), "# One\n\n## A\n\n# Two\n", nil)
	wantHeadings(t, plain, "1 One", "1.1 A", "2 Two")
}
