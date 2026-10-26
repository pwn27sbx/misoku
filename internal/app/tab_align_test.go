package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
)

// cellColumn is the column of the first r in a stripped screen line, or -1.
func cellColumn(line string, r rune) int {
	for x, c := range []rune(line) {
		if c == r {
			return x
		}
	}
	return -1
}

// The active tab starts right above the explorer's divider: its marker sits
// in the divider's column, so the tab lines up with the line under it
// instead of one column to its right.
func TestActiveTabLinesUpWithTheExplorerDivider(t *testing.T) {
	h := openNote(t)
	rows := strings.Split(stripped(h.view()), "\n")
	divider := cellColumn(rows[1], '│')
	if divider < 0 {
		t.Fatalf("setup: no explorer divider on the first body row:\n%s", h.view())
	}
	if got := cellColumn(rows[0], '▎'); got != divider {
		t.Fatalf("active tab marker at column %d, divider at column %d:\n%s\n%s", got, divider, rows[0], rows[1])
	}
}

// With the explorer hidden there is no divider: the active tab still starts
// at the left edge of the window.
func TestActiveTabStartsAtTheLeftEdgeWithoutExplorer(t *testing.T) {
	h := openNote(t, app.WithKeys(map[string][]string{"toggle_explorer": {"space e"}}))
	h.keys(" e")
	if explorerVisible(h.view()) {
		t.Fatalf("setup: explorer should be hidden:\n%s", h.view())
	}
	rows := strings.Split(stripped(h.view()), "\n")
	if got := cellColumn(rows[0], '▎'); got != 0 {
		t.Fatalf("active tab marker at column %d, want 0:\n%s", got, rows[0])
	}
}
