// Package highlight colours Markdown and fenced code (via chroma) with the
// active theme. Results are per-rune colours so both the editor and the
// preview can paint them.
package highlight

import (
	"regexp"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/theme"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
)

// Colors holds one colour per rune for every line ("" = default colour).
type Colors [][]lipgloss.Color

// lexerFor finds a chroma lexer by fence name, falling back to content
// analysis and then plain text.
func lexerFor(lang, code string) chroma.Lexer {
	var l chroma.Lexer
	switch strings.ToLower(lang) {
	case "board", "kanban":
		lang = "markdown"
	case "calendar", "timeline", "gantt", "chart", "form":
		lang = "yaml"
	case "math", "katex":
		lang = "latex"
	}
	if lang != "" {
		l = lexers.Get(lang)
	}
	if l == nil && strings.TrimSpace(code) != "" {
		l = lexers.Analyse(code)
	}
	if l == nil {
		l = lexers.Fallback
	}
	return chroma.Coalesce(l)
}

func tokenColor(tt chroma.TokenType, th theme.Theme) lipgloss.Color {
	switch {
	case tt == chroma.CommentPreproc || tt == chroma.CommentPreprocFile:
		return th.Purple
	case tt.InCategory(chroma.Comment):
		return th.GreyFg2
	case tt == chroma.KeywordType:
		return th.Yellow
	case tt == chroma.KeywordConstant:
		return th.Orange
	case tt.InCategory(chroma.Keyword):
		return th.Purple
	case tt == chroma.NameFunction || tt == chroma.NameFunctionMagic:
		return th.Blue
	case tt == chroma.NameBuiltin || tt == chroma.NameBuiltinPseudo:
		return th.Cyan
	case tt == chroma.NameClass || tt == chroma.NameNamespace || tt == chroma.NameDecorator ||
		tt == chroma.NameAttribute || tt == chroma.NameException:
		return th.Yellow
	case tt == chroma.NameTag:
		return th.Red
	case tt == chroma.NameConstant || tt == chroma.NameEntity:
		return th.Orange
	case tt == chroma.NameVariable || tt == chroma.NameVariableInstance || tt == chroma.NameProperty:
		return th.Red
	case tt == chroma.LiteralStringEscape || tt == chroma.LiteralStringRegex:
		return th.Cyan
	case tt.InSubCategory(chroma.LiteralString):
		return th.Green
	case tt.InSubCategory(chroma.LiteralNumber) || tt == chroma.Literal:
		return th.Orange
	case tt.InCategory(chroma.Operator):
		return th.Cyan
	case tt == chroma.GenericHeading || tt == chroma.GenericSubheading:
		return th.Blue
	case tt == chroma.GenericDeleted:
		return th.Red
	case tt == chroma.GenericInserted:
		return th.Green
	}
	return ""
}

// Code returns per-rune colours for a block of source code.
func Code(lang, code string, th theme.Theme) Colors {
	lines := strings.Split(code, "\n")
	out := make(Colors, len(lines))
	for i, l := range lines {
		out[i] = make([]lipgloss.Color, len([]rune(l)))
	}
	it, err := lexerFor(lang, code).Tokenise(nil, code)
	if err != nil {
		return out
	}
	row, col := 0, 0
	for tok := it(); tok != chroma.EOF; tok = it() {
		c := tokenColor(tok.Type, th)
		for _, r := range tok.Value {
			if r == '\n' {
				row++
				col = 0
				continue
			}
			if row < len(out) && col < len(out[row]) {
				out[row][col] = c
			}
			col++
		}
	}
	return out
}

var (
	headingRe = regexp.MustCompile(`^(#{1,6})\s`)
	listRe    = regexp.MustCompile(`^(\s*)([-*+]|\d+[.)])(\s+)(\[[ xX]\]\s)?`)
	quoteRe   = regexp.MustCompile(`^\s*>`)
	ruleRe    = regexp.MustCompile(`^\s*(-{3,}|\*{3,}|_{3,})\s*$`)
	codeSpan  = regexp.MustCompile("`[^`]+`")
	linkRe    = regexp.MustCompile(`!?\[[^\]]*\]\([^)]*\)`)
	wikiRe    = regexp.MustCompile(`!?\[\[[^\[\]\n]+\]\]`)
	blockIDRe = regexp.MustCompile(`\s\^[A-Za-z0-9-]+\s*$`)
	tagRe     = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	emphRe    = regexp.MustCompile(`\*\*|__|~~`)
)

// headingColor matches the preview's heading colours.
func headingColor(level int, th theme.Theme) lipgloss.Color {
	return []lipgloss.Color{th.Blue, th.Purple, th.Green, th.Yellow, th.Cyan, th.GreyFg2}[level-1]
}

// Markdown returns per-rune colours for a whole Markdown document, with
// fenced code blocks highlighted for their language.
func Markdown(text string, th theme.Theme) Colors {
	lines := strings.Split(text, "\n")
	out := make(Colors, len(lines))
	inComment := false
	for i := colorFrontMatter(lines, core.FrontMatterEnd(lines), out, th); i < len(lines); {
		i, inComment = segment(lines, i, inComment, out, th)
	}
	return out
}

// colorFrontMatter colours lines 0..end as front matter and returns the
// first line after it (0 when end is -1).
func colorFrontMatter(lines []string, end int, out Colors, th theme.Theme) int {
	for j := 0; j <= end; j++ {
		out[j] = frontMatterColors(lines[j], j == 0 || j == end, th)
	}
	return end + 1
}

// segment colours the segment that starts on line i, outside front matter
// and code, and returns the line after it and whether an HTML comment is
// open there. A segment is one Markdown line, or a whole fenced block up to
// its closing fence (or the end of the text): chroma tokenises the block as
// a unit, so its lines only change together. inComment tells whether line i
// starts inside an HTML comment, whose text holds no tags; a fence inside
// one still opens a code block, as it did before tags were coloured, and
// leaves the comment open.
func segment(lines []string, i int, inComment bool, out Colors, th theme.Theme) (int, bool) {
	line := lines[i]
	trimmed := strings.TrimSpace(line)

	if isFence(trimmed) {
		cols := make([]lipgloss.Color, len([]rune(line)))
		out[i] = cols
		fence := trimmed[:3]
		lang := strings.TrimSpace(trimmed[3:])
		fill(cols, 0, len(cols), th.GreyFg2)
		start := i + 1
		end := start
		for end < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[end]), fence) {
			end++
		}
		code := Code(lang, strings.Join(lines[start:end], "\n"), th)
		for j := start; j < end; j++ {
			out[j] = code[j-start]
		}
		if end < len(lines) {
			closing := make([]lipgloss.Color, len([]rune(lines[end])))
			fill(closing, 0, len(closing), th.GreyFg2)
			out[end] = closing
		}
		return end + 1, inComment
	}

	out[i] = markdownLine(line, th)
	tags, open := commentTags(line, inComment)
	colorTags(line, tags, out[i], th)
	return i + 1, open
}

// isFence reports whether a trimmed line opens or closes fenced code.
func isFence(trimmed string) bool {
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

// markdownLine colours one Markdown line outside code, tags aside.
func markdownLine(line string, th theme.Theme) []lipgloss.Color {
	cols := make([]lipgloss.Color, len([]rune(line)))
	switch {
	case headingRe.MatchString(line):
		level := len(headingRe.FindStringSubmatch(line)[1])
		fill(cols, 0, len(cols), headingColor(level, th))
		return cols
	case ruleRe.MatchString(line):
		fill(cols, 0, len(cols), th.GreyFg)
		return cols
	case quoteRe.MatchString(line):
		idx := len([]rune(line[:strings.Index(line, ">")]))
		fill(cols, idx, idx+1, th.Purple)
	}
	if m := listRe.FindStringSubmatchIndex(line); m != nil {
		fill(cols, runeIdx(line, m[4]), runeIdx(line, m[5]), th.Red)
		if m[8] >= 0 {
			box := th.GreyFg2
			if strings.ContainsAny(line[m[8]:m[9]], "xX") {
				box = th.Green
			}
			fill(cols, runeIdx(line, m[8]), runeIdx(line, m[9]), box)
		}
	}
	inlineSyntax(line, cols, th)
	return cols
}

// commentTags returns the tag ranges of line (by core.TagSpans, which knows
// the comments that open on the line) and whether an HTML comment is still
// open at its end. inComment tells whether the line starts inside one: its
// text up to "-->" holds no tags.
func commentTags(line string, inComment bool) ([][2]int, bool) {
	off := 0
	if inComment {
		end := strings.Index(line, "-->")
		if end < 0 {
			return nil, true
		}
		off = end + len("-->")
	}
	rest := line[off:]
	tags := core.TagSpans(rest)
	for k := range tags {
		tags[k][0] += off
		tags[k][1] += off
	}
	return tags, opensComment(rest)
}

// opensComment reports whether s, outside code spans, opens an HTML comment
// it doesn't close.
func opensComment(s string) bool {
	s = codeSpan.ReplaceAllStringFunc(s, func(m string) string { return strings.Repeat(" ", len(m)) })
	for {
		open := strings.Index(s, "<!--")
		if open < 0 {
			return false
		}
		end := strings.Index(s[open+4:], "-->")
		if end < 0 {
			return true
		}
		s = s[open+4+end+3:]
	}
}

// colorTags paints the tag byte ranges of line with the tag accent. A tag
// only takes runes no other rule coloured (an HTML tag or emphasis marker
// keeps its colour), or a heading's text, so "# Title #go" shows the tag.
// Colours only: the number of cells never changes.
func colorTags(line string, tags [][2]int, cols []lipgloss.Color, th theme.Theme) {
	under := lipgloss.Color("")
	if len(cols) > 0 && headingRe.MatchString(line) {
		under = cols[0]
	}
	for _, t := range tags {
		for j := runeIdx(line, t[0]); j < runeIdx(line, t[1]) && j < len(cols); j++ {
			if cols[j] == under {
				cols[j] = th.Cyan
			}
		}
	}
}

func frontMatterColors(line string, delimiter bool, th theme.Theme) []lipgloss.Color {
	cols := make([]lipgloss.Color, len([]rune(line)))
	if delimiter {
		fill(cols, 0, len(cols), th.GreyFg)
		return cols
	}
	if i := strings.Index(line, ":"); i > 0 {
		k := runeIdx(line, i)
		fill(cols, 0, k, th.Red)
		fill(cols, k, k+1, th.GreyFg)
		fill(cols, k+1, len(cols), th.Green)
	}
	return cols
}

// inline colours the inline syntax of one Markdown line, its tags (by
// core.TagSpans) included.
func inline(line string, cols []lipgloss.Color, th theme.Theme) {
	inlineSyntax(line, cols, th)
	colorTags(line, core.TagSpans(line), cols, th)
}

// inlineSyntax colours the inline syntax of one Markdown line, tags aside.
func inlineSyntax(line string, cols []lipgloss.Color, th theme.Theme) {
	for _, m := range tagRe.FindAllStringIndex(line, -1) {
		fill(cols, runeIdx(line, m[0]), runeIdx(line, m[1]), th.Red)
	}
	for _, m := range emphRe.FindAllStringIndex(line, -1) {
		fill(cols, runeIdx(line, m[0]), runeIdx(line, m[1]), th.GreyFg)
	}
	for _, m := range linkRe.FindAllStringIndex(line, -1) {
		s := line[m[0]:m[1]]
		mid := strings.Index(s, "](")
		fill(cols, runeIdx(line, m[0]), runeIdx(line, m[0]+mid+1), th.Blue)
		fill(cols, runeIdx(line, m[0]+mid+1), runeIdx(line, m[1]), th.GreyFg2)
	}
	for _, m := range wikiRe.FindAllStringIndex(line, -1) {
		open := strings.Index(line[m[0]:m[1]], "[[") + 2
		fill(cols, runeIdx(line, m[0]), runeIdx(line, m[0]+open), th.GreyFg)
		fill(cols, runeIdx(line, m[0]+open), runeIdx(line, m[1]-2), th.Blue)
		fill(cols, runeIdx(line, m[1]-2), runeIdx(line, m[1]), th.GreyFg)
	}
	for _, m := range blockIDRe.FindAllStringIndex(line, -1) {
		fill(cols, runeIdx(line, m[0]), runeIdx(line, m[1]), th.GreyFg)
	}
	for _, m := range codeSpan.FindAllStringIndex(line, -1) {
		fill(cols, runeIdx(line, m[0]), runeIdx(line, m[1]), th.Orange)
	}
}

func runeIdx(s string, byteIdx int) int { return len([]rune(s[:byteIdx])) }

func fill(cols []lipgloss.Color, from, to int, c lipgloss.Color) {
	for i := max(from, 0); i < min(to, len(cols)); i++ {
		cols[i] = c
	}
}

// Render paints runes using per-rune colours on top of base.
func Render(runes []rune, colors []lipgloss.Color, base lipgloss.Style) string {
	var b strings.Builder
	start := 0
	color := func(i int) lipgloss.Color {
		if i < len(colors) {
			return colors[i]
		}
		return ""
	}
	for i := 1; i <= len(runes); i++ {
		if i < len(runes) && color(i) == color(start) {
			continue
		}
		st := base
		if c := color(start); c != "" {
			st = st.Foreground(c)
		}
		b.WriteString(st.Render(string(runes[start:i])))
		start = i
	}
	return b.String()
}
