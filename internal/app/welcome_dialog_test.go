package app_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// The banner's two rows, as the welcome dialog draws them.
const (
	bannerTop    = "█▀▄▀█ █ █▀ █▀█ █▄▀ █ █"
	bannerBottom = "█ ▀ █ █ ▄█ █▄█ █ █ █▄█"
)

// welcomeHarness opens Misoku at w x ht with the first-run dialog pending.
func welcomeHarness(t *testing.T, w, ht int) (*harness, string) {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	h := newHarnessSize(t, w, ht, newTestStore(t), app.WithLearnOffer(true), app.WithConfigPath(cfgPath),
		fakeTutorial(new(*exec.Cmd), nil, nil))
	return h, cfgPath
}

// selectedOption is the title of the option the ▸ marks, "" when none is.
func selectedOption(t *testing.T, v string) string {
	t.Helper()
	for _, l := range strings.Split(v, "\n") {
		if _, after, ok := strings.Cut(l, "▸ "); ok {
			for _, title := range []string{"Soy nuevo", "Ya tengo experiencia", "Ahora no"} {
				if strings.Contains(after, title) {
					return title
				}
			}
		}
	}
	return ""
}

// S1: on a roomy terminal the dialog shows the banner, the cat, the tagline,
// the question and the three options with their descriptions, the level
// count taken from the tutorial itself.
func TestWelcomeDialogShowsTheFullDesign(t *testing.T) {
	h, _ := welcomeHarness(t, 100, 40)
	v := h.screen()
	_, levels := tutor.Scan(t.TempDir()).Count()
	for _, want := range []string{
		bannerTop, bannerBottom, "/\\_/\\", "( ^.^ )", "> ♥ <", "───────",
		"Tus notas en Markdown, en la terminal,", "al estilo de Vim.",
		"¿Es tu primera vez aquí?",
		"▸ n  Soy nuevo", "Tutorial interactivo · " + strconv.Itoa(levels) + " niveles · ~10 min",
		"e  Ya tengo experiencia", "Vas directo a tus notas",
		"a  Ahora no", "Te espera en «Learn» (tecla l) cuando quieras",
		"j/k mover · enter elegir",
	} {
		if !strings.Contains(v, want) {
			t.Errorf("the welcome dialog should show %q:\n%s", want, v)
		}
	}
	// The banner's rows line up, column for column.
	lines := strings.Split(v, "\n")
	found := false
	for i, l := range lines[:len(lines)-1] {
		c := strings.Index(l, bannerTop)
		if c < 0 {
			continue
		}
		found = true
		c2 := strings.Index(lines[i+1], bannerBottom)
		if c2 < 0 || ansi.StringWidth(l[:c]) != ansi.StringWidth(lines[i+1][:c2]) {
			t.Errorf("the banner's rows do not align:\n%s\n%s", l, lines[i+1])
		}
	}
	if !found {
		t.Fatalf("no banner:\n%s", v)
	}
	// Every glyph of the banner, the cat and the marker is one cell wide.
	for _, s := range []string{bannerTop, bannerBottom, "( ^.^ )", "> ♥ <", "▸", "─", "·", "«»"} {
		if ansi.StringWidth(s) != len([]rune(s)) {
			t.Errorf("%q is %d cells for %d runes", s, ansi.StringWidth(s), len([]rune(s)))
		}
	}
}

// S1/S4: the full design needs 19 rows; one row short, the compact variant
// shows instead: no banner or tagline, the small cat by the question, and
// the same menu.
func TestWelcomeDialogFallsBackToTheCompactVariant(t *testing.T) {
	full, _ := welcomeHarness(t, 80, 19)
	if v := full.screen(); !strings.Contains(v, bannerTop) {
		t.Fatalf("19 rows fit the full design:\n%s", v)
	}
	h, _ := welcomeHarness(t, 80, 18)
	v := h.screen()
	for _, gone := range []string{bannerTop, "Tus notas en Markdown", "^.^"} {
		if strings.Contains(v, gone) {
			t.Errorf("the compact dialog should not show %q:\n%s", gone, v)
		}
	}
	for _, want := range []string{
		"/\\_/\\", "( o.o )", "> ^ <", "¿Es tu primera vez en Misoku?",
		"▸ n  Soy nuevo", "Tutorial interactivo", "e  Ya tengo experiencia", "a  Ahora no",
		"j/k mover · enter elegir",
	} {
		if !strings.Contains(v, want) {
			t.Errorf("the compact dialog should show %q:\n%s", want, v)
		}
	}
	// The question sits on the cat's face row.
	if !strings.Contains(v, "( o.o )   ¿Es tu primera vez en Misoku?") {
		t.Errorf("the question should sit by the cat:\n%s", v)
	}
	small, _ := welcomeHarness(t, 80, 14)
	if v := small.screen(); !strings.Contains(v, "Soy nuevo") {
		t.Fatalf("14 rows fit the compact dialog:\n%s", v)
	}
}

// S2: j/k and the arrows move the ▸, stopping at both ends; Soy nuevo is
// highlighted first.
func TestWelcomeDialogMovesTheHighlight(t *testing.T) {
	h, _ := welcomeHarness(t, 100, 40)
	steps := []struct {
		move func()
		want string
	}{
		{func() {}, "Soy nuevo"},
		{func() { h.keys("k") }, "Soy nuevo"},
		{func() { h.key(tea.KeyUp) }, "Soy nuevo"},
		{func() { h.keys("j") }, "Ya tengo experiencia"},
		{func() { h.key(tea.KeyDown) }, "Ahora no"},
		{func() { h.keys("j") }, "Ahora no"},
		{func() { h.key(tea.KeyDown) }, "Ahora no"},
		{func() { h.keys("k") }, "Ya tengo experiencia"},
		{func() { h.key(tea.KeyUp) }, "Soy nuevo"},
	}
	for i, s := range steps {
		s.move()
		if got := selectedOption(t, h.screen()); got != s.want {
			t.Fatalf("step %d: ▸ on %q, want %q:\n%s", i, got, s.want, h.screen())
		}
	}
}

// S2: no other key chooses anything or moves the highlight, so a stray key
// never answers for the user.
func TestWelcomeDialogIgnoresOtherKeys(t *testing.T) {
	h, cfgPath := welcomeHarness(t, 100, 40)
	h.keys("syqlhNEA 1")
	for _, k := range []tea.KeyType{tea.KeyTab, tea.KeyShiftTab, tea.KeyLeft, tea.KeyRight, tea.KeySpace} {
		h.key(k)
	}
	v := h.screen()
	if got := selectedOption(t, v); got != "Soy nuevo" {
		t.Fatalf("a stray key moved the highlight to %q:\n%s", got, v)
	}
	if h.quit {
		t.Fatal("a stray key quit Misoku")
	}
	if _, err := os.Stat(cfgPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a stray key recorded an answer: %v", err)
	}
}

// S4: a resize to a size only the compact dialog fits keeps it open (and
// its highlight); one that fits neither closes it and keeps it pending,
// so it comes back when the terminal grows.
func TestWelcomeDialogFollowsResizes(t *testing.T) {
	h, cfgPath := welcomeHarness(t, 100, 40)
	h.keys("j")
	h.send(tea.WindowSizeMsg{Width: 80, Height: 16})
	v := h.screen()
	if !strings.Contains(v, "o.o") || strings.Contains(v, bannerTop) {
		t.Fatalf("the dialog should turn compact:\n%s", v)
	}
	if got := selectedOption(t, v); got != "Ya tengo experiencia" {
		t.Fatalf("the resize lost the highlight: %q", got)
	}
	h.send(tea.WindowSizeMsg{Width: 80, Height: 13})
	if v := h.screen(); strings.Contains(v, "Soy nuevo") {
		t.Fatalf("the dialog stayed open on a terminal it does not fit:\n%s", v)
	}
	if _, err := os.Stat(cfgPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("closing on a resize recorded an answer: %v", err)
	}
	h.send(tea.WindowSizeMsg{Width: 100, Height: 40})
	if v := h.screen(); !strings.Contains(v, bannerTop) {
		t.Fatalf("the dialog should come back, still unanswered:\n%s", v)
	}
}

// S4: a terminal that fits neither variant never shows the dialog; it waits.
func TestWelcomeDialogWaitsWhenNothingFits(t *testing.T) {
	h, _ := welcomeHarness(t, 80, 13)
	if v := h.screen(); strings.Contains(v, "Soy nuevo") {
		t.Fatalf("the dialog opened on a terminal it does not fit:\n%s", v)
	}
	h.send(tea.WindowSizeMsg{Width: 80, Height: 14})
	if v := h.screen(); !strings.Contains(v, "Soy nuevo") {
		t.Fatalf("the dialog should open once it fits:\n%s", v)
	}
}

// S2/S3: a click on an option's title or description chooses it, as the
// old buttons took clicks; a click elsewhere in the box chooses nothing.
func TestWelcomeDialogTakesAClickOnAnOption(t *testing.T) {
	for _, c := range []struct {
		text     string
		tutorial bool
	}{
		{"Tutorial interactivo", true},
		{"Soy nuevo", true},
		{"Ahora no", false},
		{"Te espera en", false},
	} {
		t.Run(c.text, func(t *testing.T) {
			cfgPath := filepath.Join(t.TempDir(), "config.json")
			var got *exec.Cmd
			h := newHarnessSize(t, 100, 40, newTestStore(t), app.WithLearnOffer(true), app.WithConfigPath(cfgPath),
				fakeTutorial(&got, nil, nil))
			clickOn(h, "¿Es tu primera vez aquí?")
			if v := h.screen(); !strings.Contains(v, "Soy nuevo") {
				t.Fatalf("a click off the options closed the dialog:\n%s", v)
			}
			clickOn(h, c.text)
			if v := h.screen(); strings.Contains(v, "Vas directo a tus notas") {
				t.Fatalf("the click should close the dialog:\n%s", v)
			}
			if !learnOffered(t, cfgPath) {
				t.Fatal("the click was not remembered")
			}
			if started := got != nil; started != c.tutorial {
				t.Errorf("started the tutorial: %v, want %v", started, c.tutorial)
			}
		})
	}
}

// clickOn clicks the first occurrence of text on the screen.
func clickOn(h *harness, text string) {
	h.t.Helper()
	for y, l := range strings.Split(h.screen(), "\n") {
		if i := strings.Index(l, text); i >= 0 {
			h.click(ansi.StringWidth(l[:i]), y)
			return
		}
	}
	h.t.Fatalf("%q not on screen:\n%s", text, h.screen())
}
