package content_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
	"github.com/pwn27sbx/misoku/internal/core"
)

// S3: :e name opens or starts a note; bare :e, :enew and :new keep opening
// an empty tab.
func TestEditCommandWithAName(t *testing.T) {
	cases := map[string]any{
		"e Existing":     core.VimEditMsg{Target: "Existing"},
		"edit Existing":  core.VimEditMsg{Target: "Existing"},
		"e Projects/AWS": core.VimEditMsg{Target: "Projects/AWS"},
		"e New one":      core.VimEditMsg{Target: "New one"},
		"e":              core.VimNewBufferMsg{},
		"e   ":           core.VimNewBufferMsg{},
		"enew":           core.VimNewBufferMsg{},
		"new":            core.VimNewBufferMsg{},
	}
	for line, want := range cases {
		if got := content.RunCommand(line, ""); got != want {
			t.Errorf(":%s = %#v, want %#v", line, got, want)
		}
	}
}
