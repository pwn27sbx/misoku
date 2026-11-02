package kitty

import "testing"

// The table is copied from kitty's rowcolumn-diacritics.txt; this pins its size
// and the entries the protocol's own example uses, so a bad edit is caught here
// instead of showing up as a shifted image.
func TestRowColumnTableMatchesKittysFile(t *testing.T) {
	t.Parallel()
	if len(rowColumn) != 297 {
		t.Fatalf("rowColumn has %d entries, want 297", len(rowColumn))
	}
	want := []rune{0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D}
	for i, r := range want {
		if rowColumn[i] != r {
			t.Errorf("rowColumn[%d] = %U, want %U", i, rowColumn[i], r)
		}
	}
	for i, r := range rowColumn {
		if r < 0x0300 {
			t.Errorf("rowColumn[%d] = %U is not a combining mark", i, r)
		}
	}
}
