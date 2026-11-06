package content_test

import "testing"

// S0: a note whose front matter closes with "..." gets the field inside that
// block, not a second "---" block prepended above it.
func TestSetFrontMatterUsesDotsCloser(t *testing.T) {
	c := newNote("---\ntitle: T\n...\nbody")
	c.SetFrontMatter("cover", "c.png")
	if got, want := c.Value(), "---\ntitle: T\ncover: c.png\n...\nbody"; got != want {
		t.Fatalf("Value = %q, want %q", got, want)
	}
}
