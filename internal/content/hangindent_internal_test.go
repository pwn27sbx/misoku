package content

import "testing"

// S1: the hanging indent is the rune count of the item's marker, leading
// indentation, quote marks and task box included, a tab counting as one.
func TestHangIndentMarkers(t *testing.T) {
	cases := []struct {
		line string
		want int
	}{
		{"- item", 2},
		{"* item", 2},
		{"+ item", 2},
		{"1. item", 3},
		{"12) item", 4},
		{"- [ ] task", 6},
		{"- [x] task", 6},
		{"- [X] task", 6},
		{"- [-] task", 6},
		{"1. [ ] task", 7},
		{"  - nested", 4},
		{"    - [ ] deeper", 10},
		{"\t- tab", 3},
		{"> quote", 2},
		{">quote", 1},
		{"> > twice", 4},
		{">> twice", 3},
		{"> - [ ] task in a quote", 8},
		{"> 1. ordered in a quote", 5},
		{"plain text", 0},
		{"    indented code", 0},
		{"-dash", 0},
		{"---", 0},
		{"1.5 million", 0},
		{"*emphasis* here", 0},
		{"# heading", 0},
		{"", 0},
	}
	for _, c := range cases {
		if got := hangIndent([]rune(c.line)); got != c.want {
			t.Errorf("hangIndent(%q) = %d, want %d", c.line, got, c.want)
		}
	}
}
