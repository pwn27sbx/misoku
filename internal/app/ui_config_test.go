package app_test

import (
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModeIconCanBeSet(t *testing.T) {
	h := openNote(t, app.WithUI(config.UI{ModeIcon: "󱗞"}))
	if v := h.view(); !strings.Contains(v, "󱗞 NORMAL") {
		t.Fatalf("statusline should show the configured mode icon:\n%s", v)
	}
	h.keys("i")
	if v := h.view(); !strings.Contains(v, "󱗞 INSERT") {
		t.Fatalf("every mode uses the same icon:\n%s", v)
	}
	if v := openNote(t).view(); strings.Contains(v, "󱗞") {
		t.Fatalf("without mode_icon each mode keeps its own icon:\n%s", v)
	}
}

func TestModifiedMarkBeforeTheTabName(t *testing.T) {
	h := openNote(t, app.WithUI(config.UI{ModifiedMark: "[+]"}))
	if tabs := strings.Split(h.view(), "\n")[0]; strings.Contains(tabs, "[+]") {
		t.Fatalf("a clean note has no mark: %q", tabs)
	}
	h.keys("A!")
	h.key(tea.KeyEsc)
	if tabs := strings.Split(h.view(), "\n")[0]; !strings.Contains(tabs, "[+] doc.md") {
		t.Fatalf("unsaved changes should show [+] before the name: %q", tabs)
	}
}

func TestBorderStyleCanBeSingle(t *testing.T) {
	t.Cleanup(func() { ui.SetBorder("rounded") })
	dialog := func(opts ...app.Option) string {
		h := openNote(t, opts...)
		h.keys("A!")
		h.key(tea.KeyEsc)
		h.send(tea.KeyMsg{Type: tea.KeyCtrlC}) // "Quit Misoku?" dialog
		return h.view()
	}
	if v := dialog(); !strings.Contains(v, "╭") {
		t.Fatalf("default dialogs are rounded:\n%s", v)
	}
	if v := dialog(app.WithUI(config.UI{Border: "single"})); !strings.Contains(v, "┌") || strings.Contains(v, "╭") {
		t.Fatalf("border single should draw square corners:\n%s", v)
	}
}

func TestUnknownBorderIsReported(t *testing.T) {
	t.Cleanup(func() { ui.SetBorder("rounded") })
	h := newHarness(t, newTestStore(t), app.WithUI(config.UI{Border: "squiggly"}))
	if v := h.view(); !strings.Contains(v, `unknown border "squiggly"`) {
		t.Fatalf("a wrong border name should be reported:\n%s", v)
	}
}
