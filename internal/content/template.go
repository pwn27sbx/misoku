package content

import (
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/core"
)

// templateDateLayout is how {{fecha}} and {{date}} write the day.
const templateDateLayout = "2006-01-02"

// InsertTemplate puts a template (the "/plantilla" slash item) into the
// note at the cursor. {{fecha}} and {{date}} become today, as YYYY-MM-DD,
// and {{titulo}} and {{title}} the note's title. The template's front
// matter joins the note's: keys the note lacks are added with the
// template's lines, keys it has keep the note's value, and a note without
// a block gets one, so there is never a second block. The rest of the
// template lands at the cursor, or right below the note's front matter
// when the cursor is inside it.
//
// It is always one change of its own, even inside an INSERT session, so a
// single u takes all of it back.
func (m *Model) InsertTemplate(template, title string, today time.Time) {
	text := strings.NewReplacer(
		"{{fecha}}", today.Format(templateDateLayout),
		"{{date}}", today.Format(templateDateLayout),
		"{{titulo}}", title,
		"{{title}}", title,
	).Replace(template)
	tfm := core.ParseFrontMatter(text)

	m.record()
	ta := &m.textarea
	if end := core.ParseFrontMatter(ta.Value()).EndLine; end >= 0 {
		if row, _ := ta.RowCol(); row <= end {
			ta.SetRowCol(end+1, 0)
		}
	}
	ta.InsertString(tfm.Body)
	if added := missingFields(tfm, text, core.ParseFrontMatter(ta.Value())); len(added) > 0 {
		row, col := ta.RowCol()
		updated, shift := addFrontMatterLines(ta.Value(), added)
		ta.SetValue(updated)
		ta.SetRowCol(row+shift, col)
	}
	ta.EnsureVisible()
	m.record()
}

// missingFields returns the lines of the template's fields (from its
// expanded text) whose key the note's front matter does not have yet, in
// the template's order; a key repeated in the template is taken once.
func missingFields(tfm core.FrontMatter, text string, note core.FrontMatter) []string {
	lines := strings.Split(text, "\n")
	seen := map[string]bool{}
	var out []string
	for _, f := range tfm.Fields {
		key := strings.ToLower(f.Key)
		if _, has := note.Get(f.Key); has || seen[key] {
			continue
		}
		seen[key] = true
		for _, l := range lines[f.Line : f.EndLine+1] {
			out = append(out, strings.TrimRight(l, "\r"))
		}
	}
	return out
}

// addFrontMatterLines puts field lines into text's front matter, before its
// closer, or in a new block on top when it has none, with the note's line
// endings. It returns the text and how many lines were added above the
// body.
func addFrontMatterLines(text string, fields []string) (string, int) {
	fm := core.ParseFrontMatter(text)
	lines := strings.Split(text, "\n")
	cr := ""
	if fm.EndLine >= 0 && strings.HasSuffix(lines[0], "\r") || fm.EndLine < 0 && strings.Contains(text, "\r\n") {
		cr = "\r"
	}
	add := make([]string, len(fields))
	for i, f := range fields {
		add[i] = f + cr
	}
	if fm.EndLine < 0 {
		block := append(append([]string{"---" + cr}, add...), "---"+cr)
		return strings.Join(append(block, lines...), "\n"), len(block)
	}
	out := append(append([]string{}, lines[:fm.EndLine]...), add...)
	return strings.Join(append(out, lines[fm.EndLine:]...), "\n"), len(add)
}
