//go:build !windows

// Bring my notes inside the welcome screen (issue #1): the setup
// assistant asks in the app's own dialogs, over the screen, against the
// fake gh of sync_setup_unix_test.go.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// expect fails unless the screen shows every one of want.
func (s *screenRun) expect(t *testing.T, want ...string) {
	t.Helper()
	view := s.m.View()
	for _, w := range want {
		if !strings.Contains(view, w) {
			t.Fatalf("the screen does not show %q:\n%s", w, view)
		}
	}
}

// bringScreen opens Bring my notes on a 120×40 screen and takes the
// folder it proposes; the assistant then asks whether to start. gh's
// sign-in runs at once (runNow).
func bringScreen(t *testing.T, g *greeting) *screenRun {
	t.Helper()
	s := g.newScreen(t)
	s.m.exec = runNow
	s.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	s.press("2")
	s.expect(t, "Folder for your notes", "~/Notes")
	s.press("enter")
	return s
}

// runNow runs gh's sign-in at once, in place of tea.ExecProcess.
func runNow(c *exec.Cmd, done tea.ExecCallback) tea.Cmd {
	return func() tea.Msg { return done(c.Run()) }
}

func TestWelcomeScreenBringsTheNotesInsideTheScreen(t *testing.T) {
	f := newFakeGH(t, true)
	g := newGreeting(t)
	seedRemote(t, f.remote(t, "notes"), map[string]string{"from-laptop.md": "written on the laptop\n"})
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`)
	s := bringScreen(t, g)
	s.press("enter") // Start
	s.expect(t, "ana/notes", "I don't have notes on another computer")
	if strings.Contains(s.m.View(), "Create a new private repository") {
		t.Fatalf("Bring my notes must not offer to create a repository:\n%s", s.m.View())
	}
	s.press("1")
	s.expect(t, "Sync is set up", "Pushed to origin")
	s.press("enter") // Close
	dir, code, out := g.finish(t, s)
	want := filepath.Join(g.home, "Notes")
	if dir != want || code != 0 {
		t.Fatalf("got %q exit %d, want %q:\n%s", dir, code, want, out)
	}
	if strings.Contains(out, "Folder for your notes [") || strings.Contains(out, "Start? [") {
		t.Errorf("the terminal asked instead of the screen:\n%s", out)
	}
	if got := vaultFiles(t, want)["from-laptop.md"]; got != "written on the laptop\n" {
		t.Errorf("from-laptop.md = %q, want the repository's note", got)
	}
	if got := g.notesDir(t); got != "~/Notes" {
		t.Errorf("notes_dir = %q, want ~/Notes", got)
	}
}

// Not signed in and with no git identity: the sign-in runs outside the
// screen, and the name comes from the GitHub account, already typed.
func TestWelcomeScreenBringSignsInAndTakesTheIdentityFromGitHub(t *testing.T) {
	f := newFakeGH(t, false)
	f.writeGlobal(t, false)
	writeFile(t, f.dir, "login", "ok")
	writeFile(t, f.dir, "user.json", anaUser)
	g := newGreeting(t)
	seedRemote(t, f.remote(t, "notes"), map[string]string{"from-laptop.md": "laptop\n"})
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`)
	s := bringScreen(t, g)
	s.press("enter") // Start
	s.expect(t, "Sign in")
	s.press("enter")
	s.expect(t, "Your name, for git", "Ana Ruiz", "From your GitHub account")
	s.press("enter", "enter", "enter") // the name, the email, Save
	s.press("1", "enter")              // ana/notes, Close
	if dir, code, out := g.finish(t, s); dir != filepath.Join(g.home, "Notes") || code != 0 {
		t.Fatalf("got %q exit %d:\n%s", dir, code, out)
	}
	if !f.called(t, "auth login") {
		t.Error("gh's sign-in did not run")
	}
}

// Leaving the assistant at any of its questions goes back to the
// choices, saying so; nothing is saved, and git's global config is as
// it was.
func TestWelcomeScreenBringCancelledAtEachStepSavesNothing(t *testing.T) {
	repos := `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`
	for name, tc := range map[string]struct {
		signedIn bool
		keys     []string
	}{
		"Esc at Start":             {true, []string{"esc"}},
		"Esc at the sign-in":       {false, []string{"enter", "esc"}},
		"Esc at the list":          {true, []string{"enter", "esc"}},
		"Ctrl+C at Start":          {true, []string{"ctrl+c"}},
		"Esc at the identity name": {false, []string{"enter", "enter", "esc"}},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFakeGH(t, tc.signedIn)
			if !tc.signedIn {
				f.writeGlobal(t, false)
				writeFile(t, f.dir, "login", "ok")
				writeFile(t, f.dir, "user.json", anaUser)
			}
			f.setRepos(t, repos)
			before := fileHash(t, f.global)
			g := newGreeting(t)
			s := bringScreen(t, g)
			s.press(tc.keys...)
			if s.quit {
				t.Fatal("the screen closed")
			}
			s.expect(t, "cancelled", "Start my notes")
			g.nothingSaved(t)
			if fileHash(t, f.global) != before {
				t.Error("git's global config changed")
			}
			s.press("esc")
			if dir, code, out := g.finish(t, s); dir != "" || code != 0 {
				t.Errorf("got %q exit %d, want nothing opened:\n%s", dir, code, out)
			}
		})
	}
}

// In a home folder that is a git repository, ~/Notes gets a repository of
// its own, asked once in one clear question; leaving the list afterwards
// removes the folder and the repository the run created.
func TestWelcomeScreenBringOwnRepoAsksOnceAndCancelRemovesIt(t *testing.T) {
	f := newFakeGH(t, true)
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`)
	g := newGreeting(t)
	git(t, g.home, "init", "-q")
	s := bringScreen(t, g)
	s.expect(t, "A repository for your notes")
	view := s.m.View()
	if n := strings.Count(view, "inside"); n != 1 || strings.Contains(view, "Misoku can run") {
		t.Errorf("the question must say it once (\"inside\" %d times):\n%s", n, view)
	}
	s.press("enter") // Create
	notes := filepath.Join(g.home, "Notes")
	if _, err := os.Stat(filepath.Join(notes, ".git")); err != nil {
		t.Fatalf("no repository of its own (%v):\n%s", err, s.m.View())
	}
	s.press("enter", "esc") // Start, leave the list
	s.expect(t, "cancelled")
	if _, err := os.Lstat(notes); !os.IsNotExist(err) {
		t.Errorf("~/Notes was left behind (%v)", err)
	}
	g.nothingSaved(t)
}

// A pick the guard refuses is said in the screen, over the list, which
// stays open; nothing is saved.
func TestWelcomeScreenBringShowsTheGuardRefusal(t *testing.T) {
	f := newFakeGH(t, true)
	seedRemote(t, f.remote(t, "app"), map[string]string{"go.mod": "module app\n", "main.go": "package main\n"})
	f.setRepos(t, `[{"nameWithOwner":"ana/app","visibility":"PRIVATE","url":"https://github.com/ana/app"}]`)
	g := newGreeting(t)
	s := bringScreen(t, g)
	s.press("enter", "1") // Start, the code project
	if s.quit {
		t.Fatal("the screen closed")
	}
	s.expect(t, "Which repository has your notes?")
	if text := strings.Join(strings.Fields(s.m.View()), " "); !strings.Contains(text, "looks like a code project") {
		t.Errorf("no refusal shown:\n%s", s.m.View())
	}
	g.nothingSaved(t)
}

// In a list of ten or more, digits number an item and only Enter picks
// it: "1" then "2" is item 12, not item 1.
func TestWelcomeScreenBringLongListPicksWithEnter(t *testing.T) {
	f := newFakeGH(t, true)
	var repos []string
	for i := 1; i <= 12; i++ {
		repos = append(repos, fmt.Sprintf(`{"nameWithOwner":"ana/r%02d","visibility":"PRIVATE","url":"https://github.com/ana/r%02d"}`, i, i))
	}
	f.setRepos(t, "["+strings.Join(repos, ",")+"]")
	seedRemote(t, f.remote(t, "r12"), map[string]string{"twelve.md": "twelve\n"})
	g := newGreeting(t)
	s := bringScreen(t, g)
	s.press("enter", "1", "2")
	s.expect(t, "Number: 12")
	g.nothingSaved(t)
	s.press("enter", "enter") // pick 12, Close
	dir, _, out := g.finish(t, s)
	if got := vaultFiles(t, dir)["twelve.md"]; got != "twelve\n" {
		t.Fatalf("twelve.md = %q, want ana/r12's note:\n%s", got, out)
	}
}

// "I don't have notes on another computer" goes on with Start my notes,
// in the screen, without touching GitHub.
func TestWelcomeScreenBringWithoutNotesElsewhereStartsThem(t *testing.T) {
	f := newFakeGH(t, true)
	f.setRepos(t, `[{"nameWithOwner":"ana/site","visibility":"PRIVATE","url":"https://github.com/ana/site"}]`)
	g := newGreeting(t)
	s := bringScreen(t, g)
	s.press("enter", "2") // Start, I don't have notes
	s.expect(t, "Folder for your notes", "~/Notes")
	s.press("enter")
	if dir, code, out := g.finish(t, s); dir != filepath.Join(g.home, "Notes") || code != 0 {
		t.Fatalf("got %q exit %d:\n%s", dir, code, out)
	}
	if f.called(t, "repo create") || f.called(t, "auth setup-git") {
		t.Errorf("starting new notes must not touch GitHub: %v", f.calls(t))
	}
}

// hangFetch puts a git on PATH that runs the real one, except a fetch into
// the folder dir: that one never answers until it is killed, as on a dead
// network while the notes are brought. The file it returns appears once
// that fetch hangs.
func hangFetch(t *testing.T, dir string) string {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	bin := t.TempDir()
	hung := filepath.Join(bin, "hung")
	script := fmt.Sprintf(`#!/bin/sh
case " $* " in
*" -C %s "*" fetch "*) touch %q; exec sleep 30 ;;
esac
exec %q "$@"
`, dir, hung, real)
	writeFile(t, bin, "git", script)
	if err := os.Chmod(filepath.Join(bin, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return hung
}

// Esc (or Ctrl+C) while the notes are being brought stops git at once and
// goes back to the choices, saying the setup was cancelled: the folder the
// run created is removed, and nothing is saved.
func TestWelcomeScreenBringEscStopsARunningStep(t *testing.T) {
	for _, key := range []string{"esc", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			f := newFakeGH(t, true)
			seedRemote(t, f.remote(t, "notes"), map[string]string{"from-laptop.md": "laptop\n"})
			f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`)
			g := newGreeting(t)
			hung := hangFetch(t, filepath.Join(g.home, "Notes"))
			s := bringScreen(t, g)
			s.press("enter") // Start
			s.expect(t, "ana/notes")
			next, cmd := s.m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
			s.m = next.(welcomeScreen)
			done := make(chan tea.Msg, 1)
			go func() { done <- cmd() }()
			waitFor(t, hung)
			s.press(key)
			if s.quit {
				t.Fatal("the screen closed")
			}
			s.expect(t, "Cancelling")
			select {
			case msg := <-done:
				s.send(msg)
			case <-time.After(10 * time.Second):
				t.Fatal(key + " did not stop git")
			}
			s.expect(t, "cancelled", "Start my notes")
			if strings.Contains(s.m.View(), "Sync setup stopped") {
				t.Fatalf("a cancel must not show as a failure:\n%s", s.m.View())
			}
			g.nothingSaved(t)
		})
	}
}

// waitFor waits until the file path exists: the step reached its hang.
func waitFor(t *testing.T, path string) {
	t.Helper()
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(path); err == nil {
			return
		}
	}
	t.Fatal("the step never reached the hanging git")
}
