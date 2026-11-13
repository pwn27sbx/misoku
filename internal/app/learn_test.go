package app_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/i18n"
	"github.com/pwn27sbx/misoku/internal/tutor"

	tea "github.com/charmbracelet/bubbletea"
)

// tutorialVault installs the bundled lessons in the tutorial folder of a
// temporary HOME and returns that folder, so nothing here touches the real
// config, vault or binary. On Windows os.UserConfigDir, os.UserCacheDir and
// os.UserHomeDir read %AppData%, %LocalAppData% and %UserProfile% instead.
// The interface, and so the tutorial, is Spanish for the rest of the test.
func tutorialVault(t *testing.T) string {
	t.Helper()
	useLang(t, i18n.ES)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	t.Setenv("USERPROFILE", home)
	dir := tutor.Dir()
	if dir == "" {
		t.Fatal("tutor.Dir is empty")
	}
	if err := tutor.Install(tutor.Bundled(), dir, false); err != nil {
		t.Fatal(err)
	}
	return dir
}

// tutorialLearn reads the tutorial's progress from dir, as main does before
// opening it or the home screen.
func tutorialLearn(t *testing.T, dir string) (*tutor.Store, tutor.Catalog) {
	t.Helper()
	cat := tutor.Default()
	store, err := tutor.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	store.ScanDir(cat, dir, tutor.DirEnv(dir))
	return store, cat
}

// passLevel writes the fixture answer of a lesson, as a player would.
func passLevel(t *testing.T, dir, file string) {
	t.Helper()
	path := filepath.Join(dir, file)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	answer := map[string]string{
		"1 Básico/B13 Títulos.md": "\n# Viaje\n\n## Equipaje\n\n### Día 1\n\n#### Detalle\n",
	}[file]
	if answer == "" {
		t.Fatalf("no fixture answer for %s", file)
	}
	if err := os.WriteFile(path, append(data, answer...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fakeTutorial replaces how a program runs outside the screen: it records the
// command and returns a result that calls back with err, after running
// during (when set), as the tutorial leaves the disk behind it.
func fakeTutorial(got **exec.Cmd, err error, during func()) app.Option {
	return app.WithExec(func(cmd *exec.Cmd, cb tea.ExecCallback) tea.Cmd {
		*got = cmd
		if during != nil {
			during()
		}
		return func() tea.Msg { return cb(err) }
	})
}

// S3: the home screen shows the tutorial's progress, and the entry is gone
// without the tutorial (also inside a tutorial session).
func TestLearnEntryShowsTheTutorialProgress(t *testing.T) {
	dir := tutorialVault(t)
	passLevel(t, dir, "1 Básico/B13 Títulos.md")
	store := newTestStore(t)

	h := newHarness(t, store, app.WithLearn(tutorialLearn(t, dir)))
	if v := h.view(); !strings.Contains(v, "Learn · Básico 1/26") {
		t.Fatalf("the home screen should count the tutorial's progress:\n%s", v)
	}

	plain := newHarness(t, newTestStore(t))
	if v := plain.view(); strings.Contains(v, "Learn · Básico 0/7") {
		t.Fatalf("without the tutorial no Learn entry may show:\n%s", v)
	}
	hidden := newHarness(t, newTestStore(t), app.WithLearn(nil, tutor.Default()))
	if v := hidden.view(); strings.Contains(v, "Learn · Básico 0/7") {
		t.Fatalf("a nil progress hides the entry:\n%s", v)
	}
}

// S3: l starts the tutorial, running misoku itself with its tutor command.
func TestLearnKeyStartsTheTutorial(t *testing.T) {
	dir := tutorialVault(t)
	var got *exec.Cmd
	h := newHarness(t, newTestStore(t), app.WithLearn(tutorialLearn(t, dir)), fakeTutorial(&got, nil, nil))
	h.keys("l")
	if got == nil {
		t.Fatal("pressing l did not start the tutorial")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != exe {
		t.Errorf("the tutorial runs %q, want misoku itself (%q)", got.Path, exe)
	}
	if len(got.Args) != 4 || got.Args[1] != "tutor" || got.Args[2] != "--route" || got.Args[3] != "B" {
		t.Errorf("the tutorial runs %q, want misoku tutor --route B", got.Args)
	}
}

// S9: the tutorial runs on the terminal itself. os/exec would hand a program
// written to Misoku's wrapped output a pipe, and a piped terminal never says
// its size: the tutorial would stay on "loading…".
func TestTutorialRunsOnTheTerminalItself(t *testing.T) {
	dir := tutorialVault(t)
	var got *exec.Cmd
	h := newHarness(t, newTestStore(t), app.WithLearn(tutorialLearn(t, dir)), fakeTutorial(&got, nil, nil))
	h.keys("l")
	if got == nil {
		t.Fatal("pressing l did not start the tutorial")
	}
	if got.Stdout != os.Stdout {
		t.Errorf("the tutorial's stdout is %T, want the terminal (os.Stdout)", got.Stdout)
	}
	if got.Stdin != os.Stdin {
		t.Errorf("the tutorial's stdin is %T, want the terminal (os.Stdin)", got.Stdin)
	}
	if got.Stderr != os.Stderr {
		t.Errorf("the tutorial's stderr is %T, want the terminal (os.Stderr)", got.Stderr)
	}
}

// S9: coming back from the tutorial reads its progress again and turns mouse
// motion reports back on, which the terminal lost while the tutorial ran.
func TestTutorialReturnRescansAndEnablesTheMouse(t *testing.T) {
	dir := tutorialVault(t)
	var got *exec.Cmd
	h := newHarness(t, newTestStore(t), app.WithLearn(tutorialLearn(t, dir)),
		fakeTutorial(&got, nil, func() { passLevel(t, dir, "1 Básico/B13 Títulos.md") }))
	h.keys("l")
	if v := h.view(); !strings.Contains(v, "Learn · Básico 1/26") {
		t.Fatalf("the entry should count the level the tutorial passed:\n%s", v)
	}
	progress, cat := tutorialLearn(t, dir)
	cmd := h.update(app.TutorialDone(nil, progress, cat))
	if cmd == nil {
		t.Fatal("the return from the tutorial returned no command")
	}
	if msg := runCmd(cmd); msg != tea.EnableMouseAllMotion() {
		t.Fatalf("the return should enable mouse motion reports, got %T", msg)
	}
}

// S9: a tutorial that fails says so, without losing its progress.
func TestTutorialReturnReportsAFailedExit(t *testing.T) {
	dir := tutorialVault(t)
	var got *exec.Cmd
	h := newHarness(t, newTestStore(t), app.WithLearn(tutorialLearn(t, dir)),
		fakeTutorial(&got, errors.New("exit status 2"), func() { passLevel(t, dir, "1 Básico/B13 Títulos.md") }))
	h.keys("l")
	if s := app.StatusText(h.m); !strings.Contains(s, "exit status 2") {
		t.Fatalf("the failed tutorial should be reported, status %q", s)
	}
	if v := h.view(); !strings.Contains(v, "Learn · Básico 1/26") {
		t.Fatalf("a failed exit must not lose the progress:\n%s", v)
	}
}

// S9/A3/A5: after a language change the home screen's Learn entries are the
// other language's tutorial, read off the update loop from its own folder
// with its own progress; switching back finds the Spanish progress intact.
func TestLearnEntriesFollowTheLanguage(t *testing.T) {
	dir := tutorialVault(t)
	passLevel(t, dir, "1 Básico/B13 Títulos.md")
	h := newHarness(t, newTestStore(t), app.WithLearn(tutorialLearn(t, dir)))
	if v := h.view(); !strings.Contains(v, "Learn · Básico 1/26") {
		t.Fatalf("the Spanish entry should count its progress:\n%s", v)
	}

	i18n.Set(i18n.EN)
	if tutor.Dir() == dir {
		t.Fatalf("the English tutorial would read the Spanish folder %q", dir)
	}
	h.send(cmdMsg(app.ReloadLearn(h.m))...)
	v := h.view()
	if !strings.Contains(v, "Learn · Basic 0/26") || strings.Contains(v, "Básico") {
		t.Fatalf("in English the entry is the English tutorial, with its own progress:\n%s", v)
	}

	i18n.Set(i18n.ES)
	h.send(cmdMsg(app.ReloadLearn(h.m))...)
	if v := h.view(); !strings.Contains(v, "Learn · Básico 1/26") {
		t.Fatalf("back in Spanish the entry should find its progress:\n%s", v)
	}

	// Inside a tutorial session the home screen has no entries to reload.
	if cmd := app.ReloadLearn(newHarness(t, newTestStore(t)).m); cmd != nil {
		t.Fatal("without Learn entries there is nothing to reload")
	}
}

// A5: in English, Learn reads the English tutorial's folder and names its
// routes in English.
func TestLearnEntryInEnglish(t *testing.T) {
	tutorialVault(t) // the Spanish tutorial exists and must not count
	i18n.Set(i18n.EN)
	dir := tutor.Dir()
	if err := tutor.Install(tutor.Bundled(), dir, false); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, newTestStore(t), app.WithLearn(tutorialLearn(t, dir)))
	if v := h.view(); !strings.Contains(v, "Learn · Basic 0/26") || !strings.Contains(v, "Learn · Intermediate 0/30") {
		t.Fatalf("the English entries should show:\n%s", v)
	}
}
