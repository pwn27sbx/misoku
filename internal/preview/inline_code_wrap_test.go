package preview_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

const dockerSpan = "docker run -d -p 8080:80 nginx"

func plainLines(out string) []string {
	return strings.Split(ansi.Strip(out), "\n")
}

// rowWith returns the index of the first line containing s, or -1.
func rowWith(lines []string, s string) int {
	for i, l := range lines {
		if strings.Contains(l, s) {
			return i
		}
	}
	return -1
}

func TestInlineCodeThatFitsIsNeverSplit(t *testing.T) {
	cases := []struct {
		name, md string
		width    int
	}{
		{"paragraph", "Start a web server on port 8080 with `" + dockerSpan + "`.", 50},
		{"paragraph at the observed width", "Start a web server on port 8080 with `" + dockerSpan + "`.", 58},
		{"heading", "## Start a web server with `" + dockerSpan + "` now", 50},
		{"bullet", "- Start a web server on port 8080 with `" + dockerSpan + "`.", 50},
		{"numbered", "1. Start a web server on port 8080 with `" + dockerSpan + "`.", 50},
		{"checkbox", "- [ ] Start a web server on port 8080 with `" + dockerSpan + "`.", 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := plainLines(preview.Compile(tc.md, theme.DefaultTheme(), tc.width))
			row := rowWith(lines, dockerSpan)
			if row < 0 {
				t.Fatalf("inline code split across lines:\n%s", strings.Join(lines, "\n"))
			}
			if row == rowWith(lines, "Start") {
				t.Errorf("expected the span to move to the next line:\n%s", strings.Join(lines, "\n"))
			}
			for _, l := range lines {
				if w := ansi.StringWidth(l); w > tc.width {
					t.Errorf("line wider than %d (%d): %q", tc.width, w, l)
				}
			}
		})
	}
}

func TestInlineCodeLongerThanTheLineStillWraps(t *testing.T) {
	words := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf"}
	md := "Run `" + strings.Join(words, " ") + "` here."
	const width = 30
	lines := plainLines(preview.Compile(md, theme.DefaultTheme(), width))
	if len(lines) < 2 {
		t.Fatalf("expected the long span to wrap:\n%s", strings.Join(lines, "\n"))
	}
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > width {
			t.Errorf("line wider than %d (%d): %q", width, w, l)
		}
	}
	// It still breaks between words, as before.
	for _, w := range words {
		if rowWith(lines, w) < 0 {
			t.Errorf("word %q was cut in half:\n%s", w, strings.Join(lines, "\n"))
		}
	}
}

func TestInlineCodeKeepsItsSpacesAndHyphens(t *testing.T) {
	out := preview.Compile("Use `"+dockerSpan+"` and a - b.", theme.DefaultTheme(), 80)
	if strings.ContainsAny(out, " ‑") {
		t.Errorf("protected characters leaked into the output: %q", out)
	}
	if !strings.Contains(ansi.Strip(out), dockerSpan) {
		t.Errorf("span text changed: %q", ansi.Strip(out))
	}
}

func TestInlineCodeInATableLeavesNoMarker(t *testing.T) {
	md := "| cmd | note |\n|---|---|\n| `" + dockerSpan + "` | ok |"
	out := preview.Compile(md, theme.DefaultTheme(), 80)
	if strings.Contains(out, "\x1b_") {
		t.Errorf("inline code marker leaked into the output: %q", out)
	}
	if !strings.Contains(ansi.Strip(out), dockerSpan) {
		t.Errorf("missing span in table: %q", ansi.Strip(out))
	}
}

func TestLinkAfterMovedInlineCodeStillMaps(t *testing.T) {
	md := "Start a web server on port 8080 with `" + dockerSpan + "` then [site](https://x.dev) end"
	out, hits := preview.CompileHits(md, theme.DefaultTheme(), 50, "", preview.CalendarView{})
	lines := plainLines(out)
	if rowWith(lines, dockerSpan) < 0 {
		t.Fatalf("inline code split across lines:\n%s", strings.Join(lines, "\n"))
	}
	got := linkHits(hits)
	if len(got) != 1 {
		t.Fatalf("got %d link hits, want 1: %+v", len(got), got)
	}
	h := got[0]
	if label := string([]rune(lines[h.Row])[h.X0:h.X1]); h.Arg != "https://x.dev" || label != "site" {
		t.Errorf("hit %q over %q; want https://x.dev over \"site\"", h.Arg, label)
	}
}

func TestInlineCodeKeepsANonBreakingSpaceItAlreadyHas(t *testing.T) {
	span := "a b-c"
	out := ansi.Strip(preview.Compile("Use `"+span+"` here.", theme.DefaultTheme(), 80))
	if !strings.Contains(out, span) {
		t.Errorf("span text changed: %q", out)
	}
}
