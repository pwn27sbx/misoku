package core

import "testing"

// RED tests for Should #3 & #4: short titles (<3 runes) must not match as
// unlinked mentions, 4-backtick fences must not close early on 3 backticks, and
// 4-space indented code blocks must be skipped for both mentions and tags.
func TestMentionsSkipsShortTitles(t *testing.T) {
	for _, title := range []string{"A", "Go", "To", "hi"} {
		if got := Mentions("Go to A and say hi", title); len(got) != 0 {
			t.Fatalf("Mentions for short title %q = %v, want empty", title, got)
		}
	}
	if got := Mentions("See Misoku here", "Misoku"); len(got) != 1 {
		t.Fatalf("Mentions for 6-rune title = %v, want 1", got)
	}
}

func TestMentionsAndTagsRespectFenceLengthAndIndentedCode(t *testing.T) {
	text := "" +
		"````md\n" +
		"```\n" +
		"Misoku #inside-four-fence\n" +
		"````\n\n" +
		"    Misoku #inside-indented\n\n" +
		"Real Misoku #real-tag\n"

	ms := Mentions(text, "Misoku")
	if len(ms) != 1 {
		t.Fatalf("Mentions = %d matches (%v), want 1 (outside 4-backtick fence and indented code)", len(ms), ms)
	}
	tags := Tags(text)
	if len(tags) != 1 || tags[0] != "real-tag" {
		t.Fatalf("Tags = %v, want [real-tag]", tags)
	}
}
