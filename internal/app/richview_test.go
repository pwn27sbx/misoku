package app

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/rich"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

type richSender struct{ sent []string }

func (r *richSender) Send(seqs ...string) error { r.sent = append(r.sent, seqs...); return nil }

// testRichFonts loads the Go fonts so the rich view never depends on what the
// machine has installed.
func testRichFonts(t *testing.T) *rich.Fonts {
	t.Helper()
	f, err := rich.NewFonts(goregular.TTF, gomono.TTF, rich.FontConfig{CellW: 12, CellH: 24, BodyPx: 16})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// withImages makes the preview report that the terminal can show pictures, and
// puts blocks mode back after the test.
func withImages(t *testing.T) {
	t.Helper()
	preview.ConfigureImages("kitty", true)
	t.Cleanup(func() { preview.ConfigureImages("blocks", false) })
}

// richModel is an app with a note open in the workspace, ready for Space V.
func richModel(t *testing.T, sender rich.Sender) *Model {
	t.Helper()
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var body strings.Builder
	for i := 0; i < 60; i++ {
		body.WriteString("A paragraph long enough to fill a row of the rich page.\n\n")
	}
	if _, err := store.SaveAs("", "Long", body.String()); err != nil {
		t.Fatal(err)
	}
	m := New(store, WithImageSender(sender))
	m.richFonts = testRichFonts(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm := next.(Model)
	mm.openInCurrentBuffer("Long.md")
	next, _ = mm.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm = next.(Model)
	return &mm
}

func pressRich(m *Model, k tea.KeyMsg) {
	next, _ := m.Update(k)
	*m = next.(Model)
}

func pressRunes(m *Model, s string) {
	pressRich(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
}

// openRich sends Space, V the way the runtime does.
func openRich(m *Model) {
	pressRich(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	pressRunes(m, "V")
}

func TestRichViewFallsBackWithoutImages(t *testing.T) {
	preview.ConfigureImages("blocks", false)
	m := richModel(t, &richSender{})
	openRich(m)
	if m.noteView == viewRich {
		t.Fatal("the rich view opened although the terminal cannot draw images")
	}
	if !strings.Contains(m.status, "Rich view unavailable") {
		t.Fatalf("status = %q, want the rich view's reason", m.status)
	}
}

func TestRichViewOpensAndDrawsPlaceholders(t *testing.T) {
	withImages(t)
	sender := &richSender{}
	m := richModel(t, sender)
	openRich(m)
	if m.noteView != viewRich {
		t.Fatalf("noteView = %v, want the rich view (status %q)", m.noteView, m.status)
	}
	if m.rich == nil || !m.rich.Ready() {
		t.Fatal("the rich view has no page after opening")
	}
	if len(sender.sent) == 0 {
		t.Fatal("opening the rich view sent no tile")
	}
	if view := m.View(); !strings.Contains(view, "\U0010EEEE") {
		t.Fatal("the composed screen has no placeholder cells")
	}
}

func TestRichViewScrollsAndReturnsToReading(t *testing.T) {
	withImages(t)
	m := richModel(t, &richSender{})
	openRich(m)
	if m.noteView != viewRich {
		t.Fatalf("noteView = %v, want the rich view", m.noteView)
	}
	top := m.rich.ScrollTop()
	pressRunes(m, "j")
	if m.rich.ScrollTop() <= top {
		t.Fatalf("j did not scroll: %d -> %d", top, m.rich.ScrollTop())
	}
	pressRich(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.noteView != viewReading {
		t.Fatalf("Esc left noteView = %v, want the editable reading view", m.noteView)
	}
}

func TestRichHintsFollowLinks(t *testing.T) {
	withImages(t)
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "From", "Go to [[To|the other note]] now.\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "To", "target\n"); err != nil {
		t.Fatal(err)
	}
	m := New(store, WithImageSender(&richSender{}))
	m.richFonts = testRichFonts(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm := next.(Model)
	mm.openInCurrentBuffer("From.md")
	next, _ = mm.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm = next.(Model)
	openRich(&mm)
	if mm.noteView != viewRich {
		t.Fatalf("noteView = %v", mm.noteView)
	}
	pressRunes(&mm, "f")
	if !mm.rich.HintsOn() {
		t.Fatal("f did not show the link hints")
	}
	pressRunes(&mm, "a")
	if mm.active != "To.md" {
		t.Fatalf("following the hint opened %q, want To.md", mm.active)
	}
}
