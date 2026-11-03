package textarea_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/textarea"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// trueColour renders in 24-bit colour for the rest of the test, so the cursor
// cell's reverse video is in the output.
func trueColour(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// hideCursorEditor is frameEditor with the terminal cursor in charge.
func hideCursorEditor(text string) textarea.Model {
	m := frameEditor(40, 6, true, text)
	m.HideCursor = true
	return m
}

// The widget's own cursor is a reverse-video cell; with HideCursor the cell
// keeps the character and its colours, so only the terminal draws a cursor.
func TestHideCursorKeepsTheCharacterAndDropsTheReverseCell(t *testing.T) {
	trueColour(t)
	shown := frameEditor(40, 6, true, "hello\nworld")
	if !reverseVideo(shown.View()) {
		t.Fatal("setup: the editor should draw its own cursor")
	}

	hidden := hideCursorEditor("hello\nworld")
	view := hidden.View()
	if reverseVideo(view) {
		t.Fatalf("the cursor cell is still drawn:\n%q", view)
	}
	plain := stripANSI(view)
	if !strings.Contains(plain, "hello") || !strings.Contains(plain, "world") {
		t.Fatalf("the text is gone:\n%q", view)
	}
	// Same size, so the cursor cell is still a cell.
	if a, b := shown.Height(), hidden.Height(); a != b {
		t.Fatalf("HideCursor changed the height: %d then %d", a, b)
	}
	if got, want := lineWidth(t, view, 0), lineWidth(t, shown.View(), 0); got != want {
		t.Fatalf("HideCursor changed the width: %d, want %d", got, want)
	}
}

// The placeholder draws a cursor on its first character too.
func TestHideCursorAppliesToThePlaceholder(t *testing.T) {
	trueColour(t)
	shown := frameEditor(40, 6, true, "")
	if !reverseVideo(shown.View()) {
		t.Fatal("setup: the placeholder should draw a cursor")
	}

	view := hideCursorEditor("").View()
	if reverseVideo(view) {
		t.Fatalf("the placeholder cursor is still drawn:\n%q", view)
	}
	if !strings.Contains(stripANSI(view), "Start writing") {
		t.Fatalf("the placeholder is gone:\n%q", view)
	}
}

// reverseVideo reports an SGR sequence that turns reverse video on: how the
// editor draws its own cursor.
func reverseVideo(s string) bool {
	for i := 0; i+3 < len(s); i++ {
		if s[i] == 0x1b && s[i+1] == '[' && s[i+2] == '7' && (s[i+3] == 'm' || s[i+3] == ';') {
			return true
		}
	}
	return false
}

// lineWidth is the printed width of row i, ANSI removed.
func lineWidth(t *testing.T, view string, i int) int {
	t.Helper()
	rows := strings.Split(view, "\n")
	if i >= len(rows) {
		t.Fatalf("row %d is outside the %d-row view", i, len(rows))
	}
	return len([]rune(stripANSI(rows[i])))
}

// stripANSI drops CSI sequences, which is all the editor writes.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
