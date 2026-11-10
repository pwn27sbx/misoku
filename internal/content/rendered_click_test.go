package content

import "testing"

// S5: whatever the rows and the column, the answer is a position inside the
// block's own lines, and a block with no matching word lands on its start.
func TestSourceAtStaysInsideTheBlock(t *testing.T) {
	lines := []string{"# Título 1", "", "naïve **café** text"}
	rows := []string{"", "▾ 󰉫 Título 1", "", "naïve café text  … 3 lines", "日本語 word"}
	for n := 1; n <= len(rows); n++ {
		for col := -3; col < 40; col++ {
			line, c := sourceAt(lines, rows[:n], col)
			if line < 0 || line >= len(lines) || c < 0 || c > len([]rune(lines[line])) {
				t.Fatalf("rows %d col %d: (%d, %d) is outside the block", n, col, line, c)
			}
		}
	}
	if line, c := sourceAt([]string{"![pic](a.png)"}, []string{"▀▀▀▀"}, 2); line != 0 || c != 0 {
		t.Fatalf("no word should land on the start, got (%d, %d)", line, c)
	}
	if line, c := sourceAt(lines, rows[:4], 7); line != 2 || c != 9 {
		t.Fatalf("the a of café should be (2, 9), got (%d, %d)", line, c)
	}
}
