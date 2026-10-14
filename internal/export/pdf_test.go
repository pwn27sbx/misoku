package export

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

const everything = "---\nicon: x\n---\n# Title\n\nText with **bold**, *italic*, `code`, ~~gone~~ and [a link](https://example.com).\n\n" +
	"- one\n  - nested\n1. first\n2. second\n- [ ] todo\n- [x] done\n\n> [!WARNING]\n> Careful\n\n> quote\n\n" +
	"```go\nfunc main() {\n\tprintln(\"hi\")\n}\n```\n\n| a | b |\n| --- | --- |\n| 1 | 2 |\n\n" +
	"```math\nE = mc^2\n```\n\n$$\\alpha^2$$\n\n---\n\n![missing](nope.png)\n\n<details>\n<summary>More</summary>\n\nHidden\n\n</details>\n\n" +
	"~~~columns\nLeft\n+++\nRight\n~~~\n\n" +
	"```board\n## Todo\n- Card\n```\n\n```chart\nA: 1\nB: 2\n```\n\n```flow\na[Start] --> b{Ok?}\n```\n\n" +
	"```calendar\nmonth: 2026-10\n2026-10-08: Ship\n```\n\n```timeline\nT: 2026-10-01 -> 2026-10-05\n```\n\n```form\ntitle: F\n? Q\n```\n\n" +
	"Icons \U000f0219 and emoji 🎉 and Ω ∑ √ ✓ ╭─╮\n"

func TestPDFRendersEveryBlock(t *testing.T) {
	var buf bytes.Buffer
	if err := PDF(everything, Options{Title: "T"}, &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) || buf.Len() < 1000 {
		t.Fatalf("not a PDF (%d bytes)", buf.Len())
	}
}

func TestPDFHandlesOddInput(t *testing.T) {
	for _, md := range []string{"", "```", "$$", "# ", "|a|\n|-|", "- ", "> ", strings.Repeat("x", 5000), strings.Repeat("- a\n", 400)} {
		var buf bytes.Buffer
		if err := PDF(md, Options{}, &buf); err != nil {
			t.Errorf("%q: %v", md[:min(len(md), 20)], err)
		}
	}
}

func TestPDFEmbedsImages(t *testing.T) {
	dir := t.TempDir()
	// 1×1 PNG.
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\xcf\xc0\xf0\x1f\x00\x05\x00\x01\xff\x89\x99=\x1d\x00\x00\x00\x00IEND\xaeB`\x82")
	if err := os.WriteFile(filepath.Join(dir, "dot.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	var with, without bytes.Buffer
	if err := PDF("![dot](dot.png)", Options{BaseDir: dir}, &with); err != nil {
		t.Fatal(err)
	}
	if err := PDF("![dot](dot.png)", Options{}, &without); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(with.Bytes(), []byte("/Subtype /Image")) {
		t.Error("image not embedded")
	}
	if bytes.Contains(without.Bytes(), []byte("/Subtype /Image")) {
		t.Error("unresolvable image should print as text")
	}
}

func TestTarget(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	cases := []struct{ arg, want string }{
		{"", filepath.Join(dir, "Note.pdf")},
		{"report", filepath.Join(dir, "report.pdf")},
		{"report.PDF", filepath.Join(dir, "report.PDF")},
		{"sub", filepath.Join(sub, "Note.pdf")},
		{"new/", filepath.Join(dir, "new", "Note.pdf")},
		{filepath.Join(dir, "abs", "x.pdf"), filepath.Join(dir, "abs", "x.pdf")},
		{"~/x", filepath.Join(home, "x.pdf")},
	}
	for _, c := range cases {
		if got := Target(c.arg, dir, "Note"); got != c.want {
			t.Errorf("Target(%q) = %q, want %q", c.arg, got, c.want)
		}
	}
}

func TestParseInline(t *testing.T) {
	runs := parseInline("a **b _c_** `d` [e](https://x.y) ~~f~~ \\*g\\* <b>h</b>")
	var got []string
	for _, r := range runs {
		flags := ""
		if r.bold {
			flags += "B"
		}
		if r.italic {
			flags += "I"
		}
		if r.code {
			flags += "C"
		}
		if r.strike {
			flags += "S"
		}
		if r.link != "" {
			flags += "L"
		}
		got = append(got, r.text+"/"+flags)
	}
	want := "a /|b /B|c/BI| /|d/C| /|e/L| /|f/S| *g* h/"
	if strings.Join(got, "|") != want {
		t.Errorf("runs = %s\nwant   %s", strings.Join(got, "|"), want)
	}
}

func TestSGRSegments(t *testing.T) {
	segs := sgrSegments("a\x1b[1;38;2;10;20;30;48;5;196mb\x1b[0mc")
	if len(segs) != 3 || segs[1].text != "b" || !segs[1].bold || *segs[1].fg != (rgb{10, 20, 30}) || *segs[1].bg != (rgb{255, 0, 0}) || segs[2].fg != nil {
		t.Errorf("unexpected segments: %+v", segs)
	}
}

func TestPrintedViewsHaveNoControls(t *testing.T) {
	th, _ := theme.Get(printTheme)
	blocks := map[string][]string{
		"form":     {"title: F", "? Q1", "? Q2 (choice): A | B", "? Q3 (multi): C | D"},
		"chart":    {"type: hbar", "Jan: 12", "Feb: 20"},
		"board":    {"## Todo", "- Card", "## Done", "- Other"},
		"calendar": {"month: 2026-10", "2026-10-08: Ship"},
		"timeline": {"T: 2026-10-01 -> 2026-10-05"},
	}
	for lang, body := range blocks {
		lines, ok := preview.PrintBlockLines(lang, body, th, blockCols)
		if !ok {
			t.Fatalf("%s: not a view", lang)
		}
		out := ansi.Strip(strings.Join(lines, "\n"))
		for _, c := range []string{"Add option", "add question", "Add value", "New page", "+ New", "Today"} {
			if strings.Contains(out, c) {
				t.Errorf("%s: printed %q:\n%s", lang, c, out)
			}
		}
	}
	// Empty views print nothing rather than editing hints.
	for _, lang := range []string{"chart", "timeline", "form"} {
		lines, _ := preview.PrintBlockLines(lang, nil, th, blockCols)
		if out := ansi.Strip(strings.Join(lines, "\n")); strings.Contains(out, "(empty") || strings.Contains(out, "(add questions") {
			t.Errorf("%s: printed an editing hint:\n%s", lang, out)
		}
	}
}

func TestSplitBlocks(t *testing.T) {
	got := splitBlocks([]segment{{text: "a██▊"}, {text: "███"}})
	var texts []string
	for _, s := range got {
		texts = append(texts, s.text)
	}
	if strings.Join(texts, "|") != "a|██|▊|███" {
		t.Errorf("splitBlocks = %q", texts)
	}
}
