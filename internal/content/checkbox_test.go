package content_test

import (
	"testing"

	"github.com/pwn27sbx/misoku/internal/content"
)

// S2: the toggle is the LazyVim companion's toggle_line, character for
// character.
func TestToggleLine(t *testing.T) {
	cases := map[string]string{
		"- [ ] task":   "- [x] task",
		"- [x] task":   "- [ ] task",
		"- [X] task":   "- [ ] task",
		"- [-] task":   "- [ ] task",
		"  - [ ] deep": "  - [x] deep",
		"- [ ]":        "- [x]",
		"* item":       "* [ ] item",
		"+ item":       "+ [ ] item",
		"- item":       "- [ ] item",
		"1. item":      "1. [ ] item",
		"12. [ ] item": "12. [x] item",
		"- [?] item":   "- [ ] [?] item",
		"foo":          "- [ ] foo",
		"  foo":        "- [ ]   foo",
		"":             "- [ ] ",
		"-foo":         "- [ ] -foo",
		"1) item":      "- [ ] 1) item",
	}
	for in, want := range cases {
		if got := content.ToggleLine(in); got != want {
			t.Errorf("ToggleLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// S5: the lines Enter toggles.
func TestIsBoxLine(t *testing.T) {
	for line, want := range map[string]bool{
		"- [ ] a": true, "* [x] a": true, "+ [X] a": true, "1. [-] a": true, "  - [ ]": true,
		"- [?] a": false, "- a": false, "[ ] a": false, "plain": false, "": false, "-[ ] a": false,
	} {
		if got := content.IsBoxLine(line); got != want {
			t.Errorf("IsBoxLine(%q) = %v, want %v", line, got, want)
		}
	}
}

// S4: removing a box keeps the list item and drops one following space.
func TestRemoveBox(t *testing.T) {
	cases := map[string]string{
		"- [x] task":   "- task",
		"- [ ] task":   "- task",
		"  * [X] task": "  * task",
		"1. [-] task":  "1. task",
		"- [x]task":    "- task",
		"- [x]  two":   "-  two",
		"- [ ]":        "- ",
		"- [?] task":   "- [?] task",
		"- task":       "- task",
		"plain":        "plain",
		"":             "",
	}
	for in, want := range cases {
		if got := content.RemoveBox(in); got != want {
			t.Errorf("RemoveBox(%q) = %q, want %q", in, got, want)
		}
	}
}
