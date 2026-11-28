package preview_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

// Setting the page, how its links resolve and its backlinks inside one
// Batch compiles the note once the batch ends, with every one of them in
// the drawn page.
func TestBatchCompilesOnceAfterEverySetter(t *testing.T) {
	m := preview.New(theme.DefaultTheme())
	m.SetSize(80, 30)
	m.Batch(func() {
		m.SetPage(core.Page{ID: "Docker.md", Title: "Docker", Content: "to [[AWS Notes]]"})
		m.SetNotes(testNotes(map[string]string{"AWS Notes": ""}))
		m.SetBacklinks([]core.Backlink{{ID: "Study.md", Title: "Study", Line: 1, Text: "see Docker"}})
	})
	v := ansi.Strip(m.View())
	for _, want := range []string{"AWS Notes", "Linked from 1 note", "Study"} {
		if !strings.Contains(v, want) {
			t.Fatalf("%q missing from the page after Batch:\n%s", want, v)
		}
	}
}
