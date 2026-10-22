package textarea_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/testnote"
	"github.com/pwn27sbx/misoku/internal/textarea"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// frameGolden holds the exact View output for a set of editor states. It was
// recorded while View still painted the whole note and cut the visible rows
// out with the viewport, so painting only the visible rows must reproduce it
// byte for byte. Regenerate with UPDATE_FRAME_GOLDEN=1 only for an intended
// change to what the editor draws.
var frameGolden = filepath.Join("testdata", "frames.golden")

// frameEditor returns an editor styled like Misoku's (colored cursor line,
// line numbers, end-of-buffer tildes) holding text, with the cursor at the top.
func frameEditor(width, height int, numbers bool, text string) textarea.Model {
	m := textarea.New()
	m.Prompt = ""
	m.ShowLineNumbers = numbers
	m.EndOfBufferCharacter = '~'
	m.CharLimit = 0
	m.MaxHeight = 0
	m.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0d0d0"))
	m.FocusedStyle.Base = lipgloss.NewStyle()
	m.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#303030")).Foreground(lipgloss.Color("#d0d0d0"))
	m.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
	m.FocusedStyle.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0d0d0"))
	m.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#606060"))
	m.FocusedStyle.CursorLineNumber = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0d0d0")).Bold(true)
	m.FocusedStyle.EndOfBuffer = lipgloss.NewStyle().Foreground(lipgloss.Color("#404040"))
	m.BlurredStyle = m.FocusedStyle
	m.BlurredStyle.CursorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#d0d0d0"))
	m.SelectionStyle = lipgloss.NewStyle().Background(lipgloss.Color("#3a3a6a")).Foreground(lipgloss.Color("#ffffff"))
	m.Placeholder = "Start writing… (i to insert)"
	m.SetWidth(width)
	m.SetHeight(height)
	m.SetValue(text)
	m.Focus()
	m.GotoTop()
	return m
}

// wideText is a note whose lines soft-wrap several times and mix wide CJK
// runes and emoji with plain words.
func wideText() string {
	extra := "漢字かな交じりの長い行は幅の広い文字で折り返される 漢字かな交じり 😀 emoji 😀 wide 😀 runes 😀 wrap here too"
	long := strings.Repeat("a long line of plain words that keeps going ", 6)
	return extra + "\n" + testnote.Note(60) + "\n" + long + "\n" + extra + "\nend"
}

type frameCase struct {
	name string
	make func() textarea.Model
}

func frameCases() []frameCase {
	note5k := testnote.Note(5000)
	note10k := testnote.Note(10050)
	note10000 := testnote.Note(10000)
	wide := wideText()
	var cases []frameCase
	for _, w := range []int{40, 80, 120} {
		cases = append(cases,
			frameCase{fmt.Sprintf("top-%d", w), func() textarea.Model {
				return frameEditor(w, 40, true, note5k)
			}},
			frameCase{fmt.Sprintf("middle-%d", w), func() textarea.Model {
				m := frameEditor(w, 40, true, note5k)
				m.SetRowCol(2500, 7)
				m.EnsureVisible()
				return m
			}},
			frameCase{fmt.Sprintf("bottom-%d", w), func() textarea.Model {
				m := frameEditor(w, 40, true, note5k)
				m.GotoBottom()
				m.EnsureVisible()
				return m
			}},
			frameCase{fmt.Sprintf("wheel-past-end-%d", w), func() textarea.Model {
				m := frameEditor(w, 40, true, note5k)
				m.GotoBottom()
				m.EnsureVisible()
				m.ScrollBy(500)
				return m
			}},
		)
	}
	cases = append(cases,
		frameCase{"middle-no-numbers-80", func() textarea.Model {
			m := frameEditor(80, 40, false, note5k)
			m.SetRowCol(2500, 7)
			m.EnsureVisible()
			return m
		}},
		frameCase{"bottom-no-numbers-80", func() textarea.Model {
			m := frameEditor(80, 40, false, note5k)
			m.GotoBottom()
			m.EnsureVisible()
			m.ScrollBy(30)
			return m
		}},
		frameCase{"bottom-10k-lines-80", func() textarea.Model {
			m := frameEditor(80, 40, true, note10k)
			m.GotoBottom()
			m.EnsureVisible()
			m.ScrollBy(20)
			return m
		}},
		frameCase{"bottom-10k-cursor-up-80", func() textarea.Model {
			m := frameEditor(80, 40, true, note10k)
			m.GotoBottom()
			m.EnsureVisible()
			m.SetRowCol(10040, 0)
			m.ScrollBy(25)
			return m
		}},
		frameCase{"10000-lines-cursor-above-80", func() textarea.Model {
			// Only the last, off-screen line has a five-digit number, and
			// it still widens the end-of-buffer padding.
			m := frameEditor(80, 40, true, note10000)
			m.GotoBottom()
			m.EnsureVisible()
			m.ScrollBy(39)
			m.SetRowCol(9990, 0)
			return m
		}},
		frameCase{"blurred-middle-80", func() textarea.Model {
			m := frameEditor(80, 40, true, note5k)
			m.SetRowCol(1200, 3)
			m.EnsureVisible()
			m.Blur()
			return m
		}},
		frameCase{"cursor-hidden-blink-80", func() textarea.Model {
			m := frameEditor(80, 40, true, note5k)
			m.SetRowCol(1200, 3)
			m.EnsureVisible()
			m.Cursor.Blink = true
			return m
		}},
		frameCase{"line-colors-80", func() textarea.Model {
			m := frameEditor(80, 40, true, note5k)
			m.SetRowCol(3000, 0)
			m.EnsureVisible()
			red, green := lipgloss.Color("1"), lipgloss.Color("2")
			m.LineColors = make([][]lipgloss.Color, 3010)
			for i := 2990; i < 3010; i++ {
				m.LineColors[i] = []lipgloss.Color{red, red, red, "", green, green}
			}
			return m
		}},
		frameCase{"selection-middle-80", func() textarea.Model {
			m := frameEditor(80, 40, true, note5k)
			m.SetAnchor(2495, 4)
			m.SetRowCol(2510, 9)
			m.EnsureVisible()
			return m
		}},
		frameCase{"placeholder-80", func() textarea.Model {
			return frameEditor(80, 12, true, "")
		}},
		frameCase{"placeholder-no-numbers-40", func() textarea.Model {
			return frameEditor(40, 12, false, "")
		}},
		frameCase{"deleted-under-offset-80", func() textarea.Model {
			// Most of the note is deleted while the view is scrolled far
			// down. (An offset past every row is only reachable from inside
			// the package; FuzzVisibleView covers it.)
			m := frameEditor(80, 40, true, note5k)
			m.GotoBottom()
			m.EnsureVisible()
			m.SelectLinewise = true
			m.SetAnchor(3, 0)
			m.DeleteSelection()
			return m
		}},
		frameCase{"prompt-func-80", func() textarea.Model {
			m := frameEditor(80, 20, true, note5k)
			m.SetPromptFunc(5, func(i int) string { return fmt.Sprintf("%d>", i%1000) })
			m.SetWidth(80)
			m.SetRowCol(700, 2)
			m.EnsureVisible()
			m.ScrollBy(3)
			return m
		}},
	)
	for _, w := range []int{40, 80} {
		cases = append(cases,
			frameCase{fmt.Sprintf("wide-cursor-wrapped-%d", w), func() textarea.Model {
				m := frameEditor(w, 12, true, wide)
				m.SetRowCol(0, 40) // deep in a line that wraps several rows
				m.EnsureVisible()
				return m
			}},
			frameCase{fmt.Sprintf("wide-offset-mid-line-%d", w), func() textarea.Model {
				// The first visible row is the middle of a wrapped line.
				m := frameEditor(w, 12, true, wide)
				m.SetRowCol(0, 0)
				m.ScrollBy(2)
				return m
			}},
			frameCase{fmt.Sprintf("wide-selection-wrapped-%d", w), func() textarea.Model {
				m := frameEditor(w, 12, true, wide)
				m.SetAnchor(0, 12)
				m.SetRowCol(2, 5)
				m.EnsureVisible()
				return m
			}},
			frameCase{fmt.Sprintf("wide-linewise-bottom-%d", w), func() textarea.Model {
				m := frameEditor(w, 12, false, wide)
				m.SelectLinewise = true
				n := m.LineCount()
				m.SetAnchor(n-4, 3)
				m.SetRowCol(n-2, 30)
				m.EnsureVisible()
				m.ScrollBy(4)
				return m
			}},
		)
	}
	return cases
}

// TestViewFrameGolden renders every frame case and compares the exact View
// string, and the scroll offset View leaves behind, with the recorded frames.
func TestViewFrameGolden(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	var b strings.Builder
	for _, c := range frameCases() {
		m := c.make()
		before := m.YOffset()
		v := m.View()
		fmt.Fprintf(&b, "=== %s yoff=%d->%d\n%s\n", c.name, before, m.YOffset(), v)
	}
	got := b.String()
	if os.Getenv("UPDATE_FRAME_GOLDEN") == "1" {
		if err := os.WriteFile(frameGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(frameGolden)
	if err != nil {
		t.Fatal(err)
	}
	if got == string(want) {
		return
	}
	gotFrames, wantFrames := strings.Split(got, "\n=== "), strings.Split(string(want), "\n=== ")
	for i := range min(len(gotFrames), len(wantFrames)) {
		if gotFrames[i] != wantFrames[i] {
			g, w := strings.Split(gotFrames[i], "\n"), strings.Split(wantFrames[i], "\n")
			for j := range min(len(g), len(w)) {
				if g[j] != w[j] {
					t.Fatalf("frame %q differs at line %d:\n got %q\nwant %q", strings.SplitN(wantFrames[i], "\n", 2)[0], j, g[j], w[j])
				}
			}
			t.Fatalf("frame %q differs in length: %d lines, want %d", strings.SplitN(wantFrames[i], "\n", 2)[0], len(g), len(w))
		}
	}
	t.Fatalf("frame count differs: %d, want %d", len(gotFrames), len(wantFrames))
}
