package preview_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// tagCells returns the runes the preview paints with the tag colour, runs
// separated by a space.
func tagCells(out string, th theme.Theme) string {
	want := strings.ToLower(fmt.Sprint(th.Cyan))
	var b strings.Builder
	prev := false
	for _, line := range strings.Split(out, "\n") {
		for _, c := range styledCells(line) {
			on := strings.ToLower(c.fg) == want
			if on {
				b.WriteRune(c.r)
			} else if prev {
				b.WriteRune(' ')
			}
			prev = on
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func TestPreviewColoursInlineTags(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	cases := []struct{ md, tags, plain string }{
		{"see #go and #diseño here", "#go #diseño", "see #go and #diseño here"},
		{"**bold #b** and *it #i*", "#b #i", "bold #b and it #i"},
		{`escaped \#go stays plain`, "", "escaped #go stays plain"},
		{"`#code` and [#link](https://x.dev/#frag) and https://x.dev/#frag", "", ""},
		{"[[Note#Heading]] and a#b, ##x, &#123;", "", ""},
		{"<a href=\"#x\">#in</a> after", "", ""},
	}
	for _, c := range cases {
		out := preview.Compile(c.md, th, 80)
		if got := tagCells(out, th); got != c.tags {
			t.Errorf("%q: tag colour on %q, want %q", c.md, got, c.tags)
		}
		if c.plain != "" && strings.TrimSpace(ansi.Strip(out)) != c.plain {
			t.Errorf("%q: plain text %q, want %q", c.md, strings.TrimSpace(ansi.Strip(out)), c.plain)
		}
	}
}

func TestPreviewTagsKeepTextAndWidth(t *testing.T) {
	trueColour(t)
	th := theme.DefaultTheme()
	md := "A #go note with #diseño/ux tags, long enough to wrap across a narrow preview width"
	tagged := ansi.Strip(preview.Compile(md, th, 30))
	plain := ansi.Strip(preview.Compile(strings.NewReplacer("#go", "Xgo", "#diseño", "Xdiseño").Replace(md), th, 30))
	if strings.NewReplacer("#", "X").Replace(tagged) != plain {
		t.Fatalf("tags changed the layout:\n%s\n---\n%s", tagged, plain)
	}
}
