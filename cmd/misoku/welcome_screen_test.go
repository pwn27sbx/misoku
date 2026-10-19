package main

// The welcome screen (issue #1): the same choices as the text menu, shown
// full screen before the app. Its model is driven through Update, the
// way the app's tests drive internal/app. These tests change HOME, so
// they are not parallel.

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// screenRun drives a welcome screen: every message goes through Update,
// and the commands it returns are run until the screen quits.
type screenRun struct {
	m    welcomeScreen
	quit bool
}

// newScreen starts the screen of g's welcome in an 80×24 terminal.
func (g *greeting) newScreen(t *testing.T) *screenRun {
	t.Helper()
	s := &screenRun{m: newWelcomeScreen(g.options(), theme.DefaultTheme())}
	s.send(tea.WindowSizeMsg{Width: 80, Height: 24})
	return s
}

// options are the welcome options of g, writing to a discarded buffer.
func (g *greeting) options() welcomeOptions {
	return welcomeOptions{
		Stdin: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{},
		ConfigPath: g.cfg, Home: g.home, Missing: g.missing, Cwd: g.cwd,
	}
}

func (s *screenRun) send(msg tea.Msg) {
	next, cmd := s.m.Update(msg)
	s.m = next.(welcomeScreen)
	s.run(cmd)
}

func (s *screenRun) run(cmd tea.Cmd) {
	if cmd == nil || isTimer(cmd) {
		return
	}
	switch msg := cmd().(type) {
	case nil:
	case tea.QuitMsg:
		s.quit = true
	case tea.BatchMsg:
		for _, c := range msg {
			s.run(c)
		}
	default:
		s.send(msg)
	}
}

// isTimer reports whether cmd only waits for a clock (a cursor blink in
// the assistant's text fields): it is not run, so a test never waits.
func isTimer(cmd tea.Cmd) bool {
	fn := runtime.FuncForPC(reflect.ValueOf(cmd).Pointer())
	return fn != nil && (strings.HasPrefix(fn.Name(), "github.com/charmbracelet/bubbles/cursor.(*Model).BlinkCmd.") ||
		strings.HasPrefix(fn.Name(), "github.com/charmbracelet/bubbletea.Tick."))
}

// press sends each key: a name such as "enter", "esc", "down", "ctrl+u",
// or text typed as it is.
func (s *screenRun) press(keys ...string) {
	named := map[string]tea.KeyType{
		"enter": tea.KeyEnter, "esc": tea.KeyEsc, "up": tea.KeyUp, "down": tea.KeyDown,
		"ctrl+c": tea.KeyCtrlC, "ctrl+u": tea.KeyCtrlU,
	}
	for _, k := range keys {
		if kt, ok := named[k]; ok {
			s.send(tea.KeyMsg{Type: kt})
			continue
		}
		s.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
	}
}

// finish ends the welcome as main does once the screen closed, and
// returns the folder to open, the exit code and what it printed.
func (g *greeting) finish(t *testing.T, s *screenRun) (string, int, string) {
	t.Helper()
	if !s.quit {
		t.Fatalf("the screen did not close:\n%s", s.m.View())
	}
	var out bytes.Buffer
	o := g.options()
	o.Out, o.Err = &out, &out
	dir, code := finishWelcome(o, s.m.outcome)
	return dir, code, out.String()
}

func TestWelcomeScreenShowsItsChoices(t *testing.T) {
	g := newGreeting(t)
	view := g.newScreen(t).m.View()
	wantOutput(t, view, "Welcome to Misoku", "Start my notes", "Bring my notes from another computer")
	if strings.Contains(view, "Use this folder") || strings.Contains(view, "I can't find") {
		t.Errorf("no folder offered and none missing:\n%s", view)
	}
}

func TestWelcomeScreenNamesTheMissingFolder(t *testing.T) {
	g := newGreeting(t)
	g.missing = filepath.Join(t.TempDir(), "Disk", "Notes")
	s := g.newScreen(t)
	s.send(tea.WindowSizeMsg{Width: 300, Height: 24}) // the whole folder on one line
	wantOutput(t, s.m.View(), "I can't find your notes at "+g.missing)
}

func TestWelcomeScreenStartCreatesRemembersAndOpensTheDefaultFolder(t *testing.T) {
	g := newGreeting(t)
	s := g.newScreen(t)
	s.press("enter")
	wantOutput(t, s.m.View(), "Folder for your notes", "~/Notes")
	s.press("enter")
	dir, code, out := g.finish(t, s)
	want := filepath.Join(g.home, "Notes")
	if dir != want || code != 0 {
		t.Fatalf("got %q exit %d, want %q:\n%s", dir, code, want, out)
	}
	if info, err := os.Stat(want); err != nil || !info.IsDir() {
		t.Errorf("~/Notes was not created (%v)", err)
	}
	if got := g.notesDir(t); got != "~/Notes" {
		t.Errorf("notes_dir = %q, want ~/Notes", got)
	}
}

func TestWelcomeScreenFolderCanBeEdited(t *testing.T) {
	g := newGreeting(t)
	s := g.newScreen(t)
	s.press("1", "ctrl+u", "~/Mine", "enter")
	dir, code, out := g.finish(t, s)
	if want := filepath.Join(g.home, "Mine"); dir != want || code != 0 {
		t.Fatalf("got %q exit %d, want %q:\n%s", dir, code, want, out)
	}
	if got := g.notesDir(t); got != "~/Mine" {
		t.Errorf("notes_dir = %q, want ~/Mine", got)
	}
}

// A folder that cannot hold notes is refused with the text menu's words,
// the screen stays on the folder, and nothing is created.
func TestWelcomeScreenRefusesAFolderItCannotUse(t *testing.T) {
	g := newGreeting(t)
	missingParent := filepath.Join(g.home, "no", "such", "Notes")
	for typed, want := range map[string]string{
		"~":           "is your home folder, the root of a disk or a system folder: choose a folder of its own for your notes, such as ~/Notes.",
		"Notes":       "Type a full path, such as ~/Notes.",
		missingParent: "the folder " + filepath.Join(g.home, "no", "such") + " does not exist",
	} {
		s := g.newScreen(t)
		s.send(tea.WindowSizeMsg{Width: 300, Height: 30}) // the whole message on one line
		s.press("enter", "ctrl+u", typed, "enter")
		if s.quit {
			t.Fatalf("%s: the screen closed", typed)
		}
		wantOutput(t, s.m.View(), want)
	}
	if _, err := os.Stat(filepath.Join(g.home, "no")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the missing parent was created (%v)", err)
	}
	g.nothingSaved(t)
}

func TestWelcomeScreenCancelSavesNothing(t *testing.T) {
	for name, keys := range map[string][]string{
		"Esc at the choices":    {"esc"},
		"q at the choices":      {"q"},
		"Ctrl+C at the choices": {"ctrl+c"},
		"Esc at the folder":     {"enter", "esc"},
	} {
		t.Run(name, func(t *testing.T) {
			g := newGreeting(t)
			s := g.newScreen(t)
			s.press(keys...)
			dir, code, out := g.finish(t, s)
			if dir != "" || code != 0 {
				t.Errorf("got %q exit %d, want nothing opened, exit 0:\n%s", dir, code, out)
			}
			wantOutput(t, out, "Nothing was changed")
			g.nothingSaved(t)
		})
	}
}

// q typed in the folder field is part of the name, not a cancel.
func TestWelcomeScreenQInTheFolderIsText(t *testing.T) {
	g := newGreeting(t)
	s := g.newScreen(t)
	s.press("enter", "ctrl+u", "~/q", "enter")
	if dir, _, out := g.finish(t, s); dir != filepath.Join(g.home, "q") {
		t.Fatalf("got %q:\n%s", dir, out)
	}
}

func TestWelcomeScreenUseThisFolderOnlyWhenOffered(t *testing.T) {
	g := newGreeting(t)
	s := g.newScreen(t)
	s.press("3", "down", "down", "down")
	if s.quit || s.m.cursor != 1 {
		t.Fatalf("3 or a third row without the offer: quit %v, row %d", s.quit, s.m.cursor)
	}

	g.cwd = withNote(t)
	s = g.newScreen(t)
	s.send(tea.WindowSizeMsg{Width: 200, Height: 24}) // wide enough for the whole folder
	wantOutput(t, s.m.View(), "Use this folder: "+g.cwd)
	s.press("down", "down", "enter")
	dir, code, out := g.finish(t, s)
	if dir != g.cwd || code != 0 {
		t.Fatalf("got %q exit %d, want %q:\n%s", dir, code, g.cwd, out)
	}
	if got := g.notesDir(t); got != g.cwd {
		t.Errorf("notes_dir = %q, want %q", got, g.cwd)
	}
}

func TestWelcomeScreenNumbersChooseToo(t *testing.T) {
	g := newGreeting(t)
	g.cwd = withNote(t)
	s := g.newScreen(t)
	s.press("3")
	if dir, _, out := g.finish(t, s); dir != g.cwd {
		t.Fatalf("3: got %q:\n%s", dir, out)
	}
	s = g.newScreen(t)
	s.press("2")
	if s.quit || !s.m.asking || s.m.askFor != choiceBring {
		t.Fatalf("2: quit %v, asking %v for %d", s.quit, s.m.asking, s.m.askFor)
	}
}

// Bring my notes stays in the screen and first proposes a free folder;
// Esc there leaves without changing anything.
func TestWelcomeScreenBringProposesAFreeFolder(t *testing.T) {
	g := newGreeting(t)
	writeFile(t, g.home, "Notes/mine.md", "mine\n")
	s := g.newScreen(t)
	s.press("down", "up")
	if s.m.cursor != 0 {
		t.Fatalf("down then up: row %d, want the first", s.m.cursor)
	}
	s.press("down", "enter")
	if s.quit {
		t.Fatal("Bring my notes closed the screen")
	}
	wantOutput(t, s.m.View(), "Folder for your notes", "~/Notes-2")
	s.press("esc")
	if _, code, out := g.finish(t, s); code != 0 || !strings.Contains(out, "Nothing was changed") {
		t.Errorf("exit %d, want 0 and nothing changed:\n%s", code, out)
	}
}

// The screen needs a terminal of at least 60×12 that can draw it; else
// the text menu greets.
func TestWelcomeScreenFitsOnlyAFullTerminal(t *testing.T) {
	for _, tc := range []struct {
		term string
		w, h int
		want bool
	}{
		{"xterm-256color", 60, 12, true},
		{"", 120, 40, true},
		{"xterm-256color", 59, 40, false},
		{"xterm-256color", 120, 11, false},
		{"dumb", 120, 40, false},
	} {
		if got := screenFits(tc.term, tc.w, tc.h); got != tc.want {
			t.Errorf("screenFits(%q, %d, %d) = %v, want %v", tc.term, tc.w, tc.h, got, tc.want)
		}
	}
}
