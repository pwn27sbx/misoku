package textarea_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/testnote"
	"github.com/pwn27sbx/misoku/internal/textarea"
)

// benchSizes are the note lengths every benchmark runs on.
var benchSizes = []struct {
	name  string
	lines int
}{
	{"500", 500},
	{"5k", 5000},
}

// benchEditor returns a focused 120x40 editor holding a generated note, with
// the cursor at the top.
func benchEditor(lines int) textarea.Model {
	m := textarea.New()
	m.SetWidth(120)
	m.SetHeight(40)
	m.SetValue(testnote.Note(lines))
	m.Focus()
	m.GotoTop()
	return m
}

// BenchmarkView measures one View of the editor.
func BenchmarkView(b *testing.B) {
	for _, s := range benchSizes {
		b.Run(s.name, func(b *testing.B) {
			m := benchEditor(s.lines)
			b.ReportAllocs()
			for b.Loop() {
				_ = m.View()
			}
		})
	}
}
