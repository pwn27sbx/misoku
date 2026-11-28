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

// The app lays the page out with the maths engine installed: a note with
// inline and display maths arrives as drawn boxes, never as its source.
func TestRichViewDrawsMath(t *testing.T) {
	withImages(t)
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := "Energy $E=mc^2$ is mass.\n\n$$\n\\frac{a}{b}\n$$\n"
	if _, err := store.SaveAs("", "Math", body); err != nil {
		t.Fatal(err)
	}
	m := New(store, WithImageSender(&richSender{}))
	m.richFonts = testRichFonts(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm := next.(Model)
	mm.openInCurrentBuffer("Math.md")
	next, _ = mm.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm = next.(Model)
	openRich(&mm)
	if mm.noteView != viewRich || mm.rich == nil || mm.rich.Page == nil {
		t.Fatalf("the rich view has no page (noteView %v, status %q)", mm.noteView, mm.status)
	}
	boxes := 0
	for _, it := range mm.rich.Page.Items {
		if it.Kind == rich.ItemBox {
			boxes++
		}
	}
	if boxes != 2 {
		t.Fatalf("maths boxes on the app's page = %d, want 2 (inline + display)", boxes)
	}
}

// Enter on a diagram block zooms it to the pane, and Esc goes back to the
// page where the reader was.
func TestRichViewZoomsADiagram(t *testing.T) {
	withImages(t)
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := "# Flow\n\nBefore.\n\n```mermaid\nflowchart LR\nA[Start] --> B[Done]\n```\n\nAfter.\n"
	if _, err := store.SaveAs("", "Flow", body); err != nil {
		t.Fatal(err)
	}
	m := New(store, WithImageSender(&richSender{}))
	m.richFonts = testRichFonts(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm := next.(Model)
	mm.openInCurrentBuffer("Flow.md")
	next, _ = mm.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm = next.(Model)
	openRich(&mm)
	if mm.noteView != viewRich || mm.rich.Page == nil || len(mm.rich.Page.Diagrams) != 1 {
		t.Fatalf("no diagram on the page (noteView %v, status %q)", mm.noteView, mm.status)
	}
	page := mm.rich.Page
	pressRunes(&mm, "j")
	top := mm.rich.ScrollTop()
	pressRich(&mm, tea.KeyMsg{Type: tea.KeyEnter})
	if mm.rich.Page == page {
		t.Fatalf("Enter did not zoom the diagram (status %q)", mm.status)
	}
	zoomed := mm.rich.Page
	if len(zoomed.Diagrams) != 0 || len(zoomed.Items) == 0 {
		t.Fatalf("zoom page is not a single diagram: %d items", len(zoomed.Items))
	}
	pressRich(&mm, tea.KeyMsg{Type: tea.KeyEsc})
	if mm.rich.Page != page {
		t.Fatal("Esc did not go back to the page")
	}
	if got := mm.rich.ScrollTop(); got != top {
		t.Fatalf("Esc left the page at row %d, want %d", got, top)
	}
}

// A rebuild while zoomed (a resize, a theme change) returns to the page: the
// zoom's rows belong to a picture that is about to be replaced.
func TestRichViewRebuildLeavesTheZoom(t *testing.T) {
	withImages(t)
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := "# Flow\n\n```mermaid\nflowchart LR\nA[Start] --> B[Done]\n```\n"
	if _, err := store.SaveAs("", "Flow", body); err != nil {
		t.Fatal(err)
	}
	m := New(store, WithImageSender(&richSender{}))
	m.richFonts = testRichFonts(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm := next.(Model)
	mm.openInCurrentBuffer("Flow.md")
	next, _ = mm.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	mm = next.(Model)
	openRich(&mm)
	page := mm.rich.Page
	pressRich(&mm, tea.KeyMsg{Type: tea.KeyEnter})
	if mm.rich.Page == page {
		t.Skip("no diagram to zoom in this pane")
	}
	next, _ = mm.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	mm = next.(Model)
	if mm.richZoomBack != nil {
		t.Fatal("the zoom survived a resize")
	}
	if mm.rich.Page == nil || len(mm.rich.Page.Diagrams) != 1 {
		t.Fatal("the page was not rebuilt after the resize")
	}
}

// A refusal is never silent: the bar names the missing piece.
func TestRichViewSaysWhyItCannotOpen(t *testing.T) {
	preview.ConfigureImages("blocks", false)
	t.Cleanup(func() { preview.ConfigureImages("blocks", false) })
	m := richModel(t, nil)
	openRich(m)
	if m.noteView == viewRich {
		t.Fatal("the rich view opened without images or an output")
	}
	if m.status == "" {
		t.Fatal("no reason in the bar")
	}

	// The caller's hint (tmux) wins over the generic message.
	withImages(t)
	hinted := richModel(t, &richSender{})
	hinted.richHint = "tmux passthrough is off"
	openRich(hinted)
	if hinted.status != "tmux passthrough is off" {
		t.Fatalf("status = %q, want the hint", hinted.status)
	}
}

// runRichCommand runs ":rich" through the editor command line and dispatches
// the resulting message to the model, matching what the user triggers.
func runRichCommand(t *testing.T, m *Model) {
	t.Helper()
	pressRunes(m, ":")
	for _, ch := range "rich" {
		pressRunes(m, string(ch))
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	*m = next.(Model)
	if cmd == nil {
		t.Fatal(":rich produced no command")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			next, _ = m.Update(c())
			*m = next.(Model)
		}
		return
	}
	next, _ = m.Update(msg)
	*m = next.(Model)
}

func TestRichCommandRefusalIsVisibleInBar(t *testing.T) {
	t.Run("blocks mode shows reason in a 90-column statusline", func(t *testing.T) {
		preview.ConfigureImages("blocks", false)
		t.Cleanup(func() { preview.ConfigureImages("blocks", false) })
		m := richModel(t, &richSender{})
		next, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 30})
		mm := next.(Model)
		runRichCommand(t, &mm)
		if mm.noteView == viewRich {
			t.Fatal(":rich opened in Blocks mode")
		}
		if !strings.Contains(mm.status, "Rich view unavailable") {
			t.Fatalf("status = %q, want Rich view unavailable", mm.status)
		}
		if bar := mm.statusline(); !strings.Contains(bar, "Rich view unavailable") {
			t.Fatalf("statusline did not show the refusal reason:\n%s", bar)
		}
	})

	t.Run("narrow pane reports required columns before actual columns", func(t *testing.T) {
		withImages(t)
		m := richModel(t, &richSender{})
		next, _ := m.Update(tea.WindowSizeMsg{Width: 35, Height: 30})
		mm := next.(Model)
		runRichCommand(t, &mm)
		if mm.noteView == viewRich {
			t.Fatal(":rich opened in a narrow pane")
		}
		if !strings.Contains(mm.status, "at least 40 columns") {
			t.Fatalf("status = %q, want 'at least 40 columns'", mm.status)
		}
	})

	t.Run("draft note refuses with a reason in the bar", func(t *testing.T) {
		withImages(t)
		m := richModel(t, &richSender{})
		m.newDraft("")
		pressRich(m, tea.KeyMsg{Type: tea.KeyEsc})
		runRichCommand(t, m)
		if m.noteView == viewRich {
			t.Fatal(":rich opened on an unsaved draft")
		}
		if !strings.Contains(m.status, "save this draft first") {
			t.Fatalf("status = %q, want draft refusal reason", m.status)
		}
		if bar := m.statusline(); !strings.Contains(bar, "save this draft first") {
			t.Fatalf("statusline did not show draft refusal:\n%s", bar)
		}
	})
}

// The + and - keys zoom the page: the faces and the cells they sit in grow
// together, so the page takes more cells and the text keeps its proportion.
func TestRichViewZoomsThePage(t *testing.T) {
	withImages(t)
	m := richModel(t, &richSender{})
	openRich(m)
	before := m.richFonts.CellW
	pressRunes(m, "+")
	if m.richFonts.CellW <= before {
		t.Fatalf("+ left the cells at %d", m.richFonts.CellW)
	}
	if m.rich == nil || m.rich.Page == nil {
		t.Fatal("the page was lost after zooming")
	}
	if !strings.Contains(m.status, "page zoom") {
		t.Fatalf("status = %q, want the zoom notice", m.status)
	}
	pressRunes(m, "-")
	pressRunes(m, "-")
	if m.richFonts.CellW >= before {
		t.Fatalf("- left the cells at %d, want fewer than %d", m.richFonts.CellW, before)
	}
}

// A resize keeps the note line the reader was on at the top of the page.
func TestRichViewResizeKeepsTheLine(t *testing.T) {
	withImages(t)
	m := richModel(t, &richSender{})
	openRich(m)
	for i := 0; i < 6; i++ {
		pressRunes(m, "j")
	}
	line := m.rich.SourceLine()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 38})
	mm := next.(Model)
	if got := mm.rich.SourceLine(); got < line-1 || got > line+1 {
		t.Fatalf("resize moved the top from line %d to %d", line, got)
	}
}

// The wheel scrolls and a click on a text row opens the text view there.
func TestRichViewMouseScrollsAndClicks(t *testing.T) {
	withImages(t)
	m := richModel(t, &richSender{})
	openRich(m)
	top := m.rich.ScrollTop()
	sendMsg(m, tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	if m.rich.ScrollTop() <= top {
		t.Fatal("the wheel did not scroll")
	}
	l := m.layout()
	row := l.NoteY() + 2
	sendMsg(m, tea.MouseMsg{X: 5, Y: row, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	sendMsg(m, tea.MouseMsg{X: 5, Y: row, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	if m.noteView != viewReading {
		t.Fatalf("a click left noteView = %v, want the text view", m.noteView)
	}
}

// sendMsg drives one arbitrary message through the model, the way the runtime
// does.
func sendMsg(m *Model, msg tea.Msg) {
	next, _ := m.Update(msg)
	*m = next.(Model)
}
