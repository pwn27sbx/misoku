package content

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/clipimage"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/textarea"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// pasteTextMsg stands in for the textarea's paste message.
type pasteTextMsg struct{ text string }

// withPaste replaces both halves of the clipboard for one test: the text it
// would paste and the picture it would find.
func withPaste(t *testing.T, text string, img *core.ClipboardImageMsg, err error) {
	t.Helper()
	oldText, oldImage := pasteText, readClipboardImage
	pasteText = func() tea.Msg { return pasteTextMsg{text: text} }
	readClipboardImage = func(ctx context.Context) (core.ClipboardImageMsg, bool, error) {
		if img != nil {
			return *img, true, nil
		}
		return core.ClipboardImageMsg{}, false, err
	}
	t.Cleanup(func() { pasteText, readClipboardImage = oldText, oldImage })
}

// editor is a model showing text, focused, in insert mode at the end of it.
func editor(t *testing.T, text string) *Model {
	t.Helper()
	m := New(theme.DefaultTheme())
	m.SetSize(80, 20)
	m.SetPage(core.Page{ID: "notes/a.md", Title: "a", Content: text})
	m.Focus()
	m.EnterInsert()
	send(&m, tea.KeyMsg{Type: tea.KeyEnd})
	return &m
}

// send runs one message through the model, following the command it returns.
func send(m *Model, msg tea.Msg) {
	var cmd tea.Cmd
	*m, cmd = m.Update(msg)
	if cmd == nil {
		return
	}
	if next := cmd(); next != nil {
		*m, _ = m.Update(next)
	}
}

func ctrlV() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyCtrlV} }

// A clipboard holding text pastes exactly what it pasted before pictures
// existed: the same message the textarea has always produced.
func TestCtrlVPastesTextWhenThereIsNoPicture(t *testing.T) {
	withPaste(t, "hola", nil, nil)
	m := editor(t, "note\n")

	_, cmd := m.Update(ctrlV())
	if cmd == nil {
		t.Fatal("ctrl+v returned no command")
	}
	if _, ok := cmd().(pasteTextMsg); !ok {
		t.Fatalf("ctrl+v with text returned %T, want the textarea's paste message", cmd())
	}
}

// A picture becomes a message for the app to attach, and nothing is typed.
func TestCtrlVReportsAPictureInsteadOfTyping(t *testing.T) {
	img := core.ClipboardImageMsg{Path: "/tmp/pasted.png", MIME: "image/png", Ext: ".png", Bytes: 12}
	withPaste(t, "unused", &img, nil)
	m := editor(t, "note\n")
	before := m.Value()

	_, cmd := m.Update(ctrlV())
	got, ok := cmd().(core.ClipboardImageMsg)
	if !ok || got.Path != "/tmp/pasted.png" {
		t.Fatalf("expected the picture, got %#v", cmd())
	}
	if m.Value() != before {
		t.Fatalf("a picture must not be typed into the note: %q", m.Value())
	}
}

// A missing image tool must never take the text paste away: the text clipboard
// may work through a tool this package does not probe (xsel).
func TestAMissingImageToolStillPastesText(t *testing.T) {
	withPaste(t, "hola", nil, clipimage.ErrNoTool)
	m := editor(t, "note\n")

	_, cmd := m.Update(ctrlV())
	if _, ok := cmd().(pasteTextMsg); !ok {
		t.Fatalf("expected the text paste, got %#v", cmd())
	}
}

// When the text paste fails too, the status line says what to install.
func TestPastingWithNoToolAtAllSaysWhatToInstall(t *testing.T) {
	withPaste(t, "", nil, clipimage.ErrNoTool)
	pasteText = func() tea.Msg { return pasteErrForTest{} } // the text paste failed too
	m := editor(t, "note\n")

	_, cmd := m.Update(ctrlV())
	status, ok := cmd().(core.StatusMsg)
	if !ok || !status.Error {
		t.Fatalf("expected an error status, got %#v", cmd())
	}
	if !strings.Contains(status.Text, "Paste image needs") {
		t.Fatalf("status = %q", status.Text)
	}
}

// pasteErrForTest behaves like the textarea's failure message, which carries
// its error.
type pasteErrForTest struct{}

func (pasteErrForTest) Error() string { return "no clipboard" }

// Bracketed paste carries text: it is typed and the clipboard is not consulted.
func TestBracketedPasteWithTextNeverAsksTheClipboard(t *testing.T) {
	asked := false
	withPaste(t, "nope", nil, nil)
	old := readClipboardImage
	readClipboardImage = func(context.Context) (core.ClipboardImageMsg, bool, error) {
		asked = true
		return core.ClipboardImageMsg{}, false, nil
	}
	t.Cleanup(func() { readClipboardImage = old })

	m := editor(t, "")
	send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("pegado"), Paste: true})
	if asked {
		t.Fatal("a bracketed paste with text must not probe the clipboard")
	}
	if !strings.Contains(m.Value(), "pegado") {
		t.Fatalf("the pasted text was not inserted: %q", m.Value())
	}
}

// An empty bracketed paste is the shape a terminal would use for a clipboard
// with no text, so it asks the clipboard once.
func TestEmptyBracketedPasteAsksForAPicture(t *testing.T) {
	img := core.ClipboardImageMsg{Path: "/tmp/x.png", Ext: ".png"}
	withPaste(t, "", &img, nil)
	m := editor(t, "")

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Paste: true})
	if _, ok := cmd().(core.ClipboardImageMsg); !ok {
		t.Fatalf("expected the picture, got %#v", cmd())
	}
}

// PasteClipboard is the bound action: the same command ctrl+v runs.
func TestPasteClipboardIsTheBoundAction(t *testing.T) {
	img := core.ClipboardImageMsg{Path: "/tmp/y.png"}
	withPaste(t, "", &img, nil)
	m := editor(t, "")

	if _, ok := m.PasteClipboard()().(core.ClipboardImageMsg); !ok {
		t.Fatal("PasteClipboard did not read the clipboard")
	}
}

// The link of a picture sits on a line of its own, which is what makes the
// preview draw it instead of the placeholder icon.
func TestInsertBlockPutsALinkOnItsOwnLine(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		end  bool
		want []string
	}{
		{"an empty line", "", false, []string{"![](x.png)"}},
		{"after the text", "Hola", true, []string{"Hola", "![](x.png)"}},
		{"at the start of a line with text", "Hola", false, []string{"![](x.png)", "Hola"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := editor(t, tc.text)
			if !tc.end {
				send(m, tea.KeyMsg{Type: tea.KeyHome})
			}
			m.InsertBlock("![](x.png)")
			value := m.Value()
			lines := strings.Split(strings.TrimSuffix(value, "\n"), "\n")
			for _, line := range lines {
				if strings.Contains(line, "![](x.png)") && strings.TrimSpace(line) != "![](x.png)" {
					t.Fatalf("the link shares a line with text: %q", lines)
				}
			}
			link, textLine := -1, -1
			for i, line := range lines {
				if strings.TrimSpace(line) == "![](x.png)" {
					link = i
				} else if strings.TrimSpace(line) == strings.TrimSpace(tc.text) && tc.text != "" {
					textLine = i
				}
			}
			if link < 0 {
				t.Fatalf("no link line in %q", lines)
			}
			if textLine >= 0 && tc.end && textLine > link {
				t.Fatalf("the text should stay above the link: %q", lines)
			}
			if textLine >= 0 && !tc.end && textLine < link {
				t.Fatalf("the link should sit above the text: %q", lines)
			}
		})
	}
}

// Text after the cursor ends up below the link, never glued to it.
func TestInsertBlockPushesFollowingTextDown(t *testing.T) {
	m := editor(t, "Hola mundo")
	send(m, tea.KeyMsg{Type: tea.KeyHome})
	for range 5 {
		send(m, tea.KeyMsg{Type: tea.KeyRight}) // after "Hola "
	}
	m.InsertBlock("![](x.png)")

	lines := strings.Split(m.Value(), "\n")
	for _, line := range lines {
		if strings.Contains(line, "![](x.png)") && strings.TrimSpace(line) != "![](x.png)" {
			t.Fatalf("the link shares a line with text: %q", lines)
		}
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"Hola", "mundo", "![](x.png)"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("lost %q in %q", want, lines)
		}
	}
	// The link must sit between the two halves of the split line.
	link := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "![](x.png)" {
			link = i
		}
	}
	above, below := "", ""
	for i := link - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			above = lines[i]
			break
		}
	}
	for i := link + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			below = lines[i]
			break
		}
	}
	if !strings.Contains(above, "Hola") || !strings.Contains(below, "mundo") {
		t.Fatalf("the link is not between the split halves: %q", lines)
	}
}

// A pasted picture carries the whole command through the exported action, and
// the error path of the text paste is untouched.
func TestPasteTextErrorsStillReachTheStatusline(t *testing.T) {
	oldText := pasteText
	pasteText = func() tea.Msg { return textarea.Paste() } // the real one, no seam
	t.Cleanup(func() { pasteText = oldText })

	m := editor(t, "")
	withPaste(t, "", nil, errors.New("boom"))
	if _, cmd := m.Update(ctrlV()); cmd == nil {
		t.Fatal("no command")
	}
}
