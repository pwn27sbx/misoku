package preview_test

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"
)

const lookDoc = "# Title\n\n## Sub\n\n- item\n  - nested\n\n- [ ] open\n- [x] done\n- [-] doing\n"

func render(t *testing.T, look *preview.Look) (raw, plain string) {
	t.Helper()
	m := preview.New(theme.DefaultTheme())
	if look != nil {
		m.SetLook(*look)
	}
	m.SetSize(60, 30)
	m.SetPage(core.Page{ID: "look.md", Title: "look", Content: lookDoc})
	raw = m.View()
	return raw, ansi.Strip(raw)
}

func lineWith(s, sub string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, sub) {
			return l
		}
	}
	return ""
}

func TestDefaultLookIsUnchanged(t *testing.T) {
	_, v := render(t, nil)
	for _, want := range []string{"󰉫 Title", "󰉬 Sub", "● item", "○ nested", "󰄱 open", "󰄲 done"} {
		if !strings.Contains(v, want) {
			t.Errorf("default preview should show %q:\n%s", want, v)
		}
	}
	// "- [-]" is not a checkbox unless a todo glyph is configured.
	if !strings.Contains(v, "● [-] doing") {
		t.Errorf("without a todo glyph [-] stays a plain bullet:\n%s", v)
	}
}

func TestLookLikeLazyVim(t *testing.T) {
	look := preview.DefaultLook()
	look.HeadingIcons = [6]string{"① ", "② ", "③ ", "④ ", "⑤ ", "⑥ "}
	look.HeadingBackground = true
	look.Checked = "󰱒"
	look.Todo = "󰥔"
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)
	raw, v := render(t, &look)

	for _, want := range []string{"① Title", "② Sub", "󰄱 open", "󰱒 done", "󰥔 doing", "● item"} {
		if !strings.Contains(v, want) {
			t.Errorf("preview should show %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "󰉫") || strings.Contains(v, "󰄲") {
		t.Errorf("the configured glyphs replace the defaults:\n%s", v)
	}
	// Full-width band: the heading row is painted to the end of the pane.
	h1 := lineWith(raw, "Title")
	if !strings.Contains(h1, "48;2;") {
		t.Fatalf("heading should have a background color: %q", h1)
	}
	// After the title the band keeps going: the background is set again for
	// the padding that reaches the right edge.
	afterTitle := h1[strings.Index(h1, "Title")+len("Title"):]
	if !strings.Contains(afterTitle, "48;2;") {
		t.Fatalf("the band should run past the title to the right edge: %q", afterTitle)
	}
	if strings.Contains(v, "━━━") {
		t.Errorf("with the band, the H1 underline is not drawn:\n%s", v)
	}
}

func TestBulletsCanBeConfigured(t *testing.T) {
	look := preview.DefaultLook()
	look.Bullets = []string{"▸", "▹"}
	_, v := render(t, &look)
	if !strings.Contains(v, "▸ item") || !strings.Contains(v, "▹ nested") {
		t.Fatalf("custom bullets per depth:\n%s", v)
	}
}
