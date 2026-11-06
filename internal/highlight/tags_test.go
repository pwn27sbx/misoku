package highlight

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// tagged returns, for one highlighted line, the runes painted with the tag
// colour as a string, so a test can compare them at a glance.
func tagged(line string, cols Colors, i int, th theme.Theme) string {
	var b strings.Builder
	runes := []rune(line)
	for j, c := range cols[i] {
		if c == th.Cyan {
			b.WriteRune(runes[j])
		} else if j > 0 && cols[i][j-1] == th.Cyan {
			b.WriteRune(' ')
		}
	}
	return strings.TrimSpace(b.String())
}

func TestInlineTagsUseTheTagColour(t *testing.T) {
	th := theme.DefaultTheme()
	cases := []struct{ line, want string }{
		{"see #go and #diseño here", "#go #diseño"},
		{"- [ ] task #todo/next", "#todo/next"},
		{"> quoted #idea", "#idea"},
		{"# Title #go", "#go"},
		{"## Sub", ""},
		{"# Heading", ""},
		{"#Heading is a tag", "#Heading"},
		{"https://x.dev/#frag", ""},
		{"a#b and ##x and &#123;", ""},
		{"`#code` but #real", "#real"},
		{"[#link](u) [[Note#Heading]] ![#i](p)", ""},
		{"<!-- #hidden --> #shown", "#shown"},
		{"**#bold** text", "#bold"},
	}
	for _, c := range cases {
		cols := Markdown(c.line, th)
		if got := tagged(c.line, cols, 0, th); got != c.want {
			t.Errorf("%q: tag colour on %q, want %q", c.line, got, c.want)
		}
	}
}

func TestTagsKeepOtherColoursAndWidths(t *testing.T) {
	th := theme.DefaultTheme()
	line := "**#go** `x` <b>#b</b>"
	cols := Markdown(line, th)[0]
	if len(cols) != len([]rune(line)) {
		t.Fatalf("%d colours for %d runes", len(cols), len([]rune(line)))
	}
	// The emphasis markers and the code span keep their colours.
	if cols[0] != th.GreyFg || cols[8] != th.Orange {
		t.Errorf("tag colouring repainted other syntax: %v", cols)
	}
	// "#b" between HTML tags: "<b>" stays red.
	if cols[12] != th.Red {
		t.Errorf("HTML tag repainted: %v", cols)
	}
	// Heading text around a tag keeps the heading colour.
	h := Markdown("# A #go", th)[0]
	if h[0] != headingColor(1, th) || h[2] != headingColor(1, th) || h[4] != th.Cyan {
		t.Errorf("heading with a tag: %v", h)
	}
}

func TestNoTagsInFencesFrontMatterOrComments(t *testing.T) {
	th := theme.DefaultTheme()
	text := strings.Join([]string{
		"---",           // 0
		"title: #fm",    // 1
		"tags: [#a]",    // 2
		"---",           // 3
		"```",           // 4
		"#fenced",       // 5
		"```",           // 6
		"<!-- open",     // 7
		"#inside",       // 8
		"still #inside", // 9
		"--> #after",    // 10
		"~~~",           // 11
		"#tilde",        // 12
		"~~~",           // 13
		"#last",         // 14
	}, "\n")
	lines := strings.Split(text, "\n")
	cols := Markdown(text, th)
	want := map[int]string{10: "#after", 14: "#last"}
	for i, l := range lines {
		if got := tagged(l, cols, i, th); got != want[i] {
			t.Errorf("line %d %q: tag colour on %q, want %q", i, l, got, want[i])
		}
	}
}

// A fence inside an unclosed HTML comment still opens a fence, as it did
// before tags, and the comment stays open after it: only tag colours depend
// on the comment.
func TestFenceInsideCommentStillAFence(t *testing.T) {
	th := theme.DefaultTheme()
	text := "<!-- open\n```go\nfunc f() {}\n```\n#inside\n--> #after"
	got := Markdown(text, th)
	if want := legacyMarkdown(text, th); !reflect.DeepEqual(got[:4], want[:4]) {
		t.Fatalf("fence colours changed:\n got %v\nwant %v", got, want)
	}
	if tg := tagged("#inside", got, 4, th); tg != "" {
		t.Errorf("tag inside the comment coloured: %q", tg)
	}
	if tg := tagged("--> #after", got, 5, th); tg != "#after" {
		t.Errorf("tag after the comment = %q", tg)
	}
}

// A note without tags is coloured exactly as before tags existed.
func TestNoteWithoutTagsUnchanged(t *testing.T) {
	text := strings.Join([]string{
		"---", "title: Hi", "icon: x", "---",
		"# Heading", "## Sub **b** and `c`", "##### five",
		"Text with a#b, ##x, &#123; and https://x.dev/#frag.",
		"[link](#frag) [[Note#Heading|alias]] ![img](a.png) ^block-1",
		"<!-- a comment", "spanning lines", "-->", "<b>html</b> ~~strike~~",
		"- [x] done", "- [ ] open", "1. step", "> quote", "***",
		"```go", "// #include\nfunc main() {}", "```", "~~~python", "# comment", "~~~",
		"日本語 **太字** ünïcödé", "line\r", "",
	}, "\n")
	for _, th := range []theme.Theme{theme.DefaultTheme(), otherTheme(t)} {
		if got, want := Markdown(text, th), legacyMarkdown(text, th); !reflect.DeepEqual(got, want) {
			t.Fatalf("tagless note changed colour:\n got %v\nwant %v", got, want)
		}
	}
}

var tagPool = []string{
	"", "#go", "see #diseño", "# Title #go", "<!-- open", "#inside", "-->", "--> #after",
	"<!-- one --> #x <!-- two", "```", "```go", "~~~", "---", "title: #fm", "`#c` #d",
	"a#b ##x", "https://x/#frag", "[[N#H]]", "- [ ] #todo", "> #q", "--> #c <!-- d -->",
}

// The incremental highlighter agrees with a full highlight while tags and
// comments spanning lines are typed, pasted and deleted.
func TestIncrementalTagsMatchFullHighlight(t *testing.T) {
	th := theme.DefaultTheme()
	for s := 0; s < 30; s++ {
		rng := rand.New(rand.NewSource(int64(s)))
		var lines []string
		for range 8 {
			lines = append(lines, tagPool[rng.Intn(len(tagPool))])
		}
		h := &Highlighter{}
		for step := 0; step < 40; step++ {
			at := rng.Intn(len(lines) + 1)
			switch rng.Intn(3) {
			case 0:
				lines = insertAt(lines, at, tagPool[rng.Intn(len(tagPool))])
			case 1:
				if at < len(lines) {
					lines[at] += " #t"
				}
			default:
				if at < len(lines) {
					lines = append(append([]string{}, lines[:at]...), lines[at+1:]...)
				}
			}
			text := strings.Join(lines, "\n")
			if got, want := h.Markdown(text, th), Markdown(text, th); !reflect.DeepEqual(got, want) {
				t.Fatalf("%s: incremental differs\ntext %q\n got %v\nwant %v", fmt.Sprint(s, "/", step), text, got, want)
			}
		}
	}
}

func TestEditingOneLineColoursItsNewTag(t *testing.T) {
	th := theme.DefaultTheme()
	h := &Highlighter{}
	h.Markdown("a\nplain\nb", th)
	cols := h.Markdown("a\nplain #go\nb", th)
	if got := tagged("plain #go", cols, 1, th); got != "#go" {
		t.Fatalf("edited line tags = %q", got)
	}
	if h.Recolored() != 1 {
		t.Errorf("recoloured %d lines, want 1", h.Recolored())
	}
}
