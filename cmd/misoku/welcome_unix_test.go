//go:build !windows

// Bring my notes runs the setup assistant against the fake gh, a shell
// script (see sync_setup_unix_test.go).

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBringMyNotesBringsRemembersAndOpensThem(t *testing.T) {
	f := newFakeGH(t, true)
	g := newGreeting(t)
	seedRemote(t, f.remote(t, "notes"), map[string]string{"from-laptop.md": "written on the laptop\n"})
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`)
	dir, code, out := g.run(t, "2\n\n\n1\n")
	want := filepath.Join(g.home, "Notes")
	if dir != want || code != 0 {
		t.Fatalf("got %q exit %d, want %q:\n%s", dir, code, want, out)
	}
	if got := vaultFiles(t, want)["from-laptop.md"]; got != "written on the laptop\n" {
		t.Errorf("from-laptop.md = %q, want the repository's note", got)
	}
	if got := g.notesDir(t); got != "~/Notes" {
		t.Errorf("notes_dir = %q, want ~/Notes", got)
	}
	wantOutput(t, out, "1) ana/notes", "2) I don't have notes on another computer → Start my notes", "Added "+want+" to sync.vaults")
	if strings.Contains(out, "Create a new private repository") {
		t.Errorf("Bring my notes must not offer to create a repository:\n%s", out)
	}
}

func TestBringMyNotesWithoutNotesElsewhereStartsThem(t *testing.T) {
	f := newFakeGH(t, true)
	g := newGreeting(t)
	f.setRepos(t, `[{"nameWithOwner":"ana/site","visibility":"PRIVATE","url":"https://github.com/ana/site"}]`)
	dir, code, out := g.run(t, "2\n\n\n2\n\n")
	want := filepath.Join(g.home, "Notes")
	if dir != want || code != 0 {
		t.Fatalf("got %q exit %d, want %q:\n%s", dir, code, want, out)
	}
	if got := g.notesDir(t); got != "~/Notes" {
		t.Errorf("notes_dir = %q, want ~/Notes", got)
	}
	if f.called(t, "repo create") || f.called(t, "auth setup-git") {
		t.Errorf("starting new notes must not touch GitHub: %v", f.calls(t))
	}
}

func TestBringMyNotesCancelledSavesNothing(t *testing.T) {
	f := newFakeGH(t, true)
	g := newGreeting(t)
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`)
	before := fileHash(t, f.global)
	dir, code, out := g.run(t, "2\n\n\n\n")
	if dir != "" || code != 0 {
		t.Errorf("got %q exit %d, want nothing opened:\n%s", dir, code, out)
	}
	wantOutput(t, out, "Setup cancelled")
	g.nothingSaved(t)
	if fileHash(t, f.global) != before {
		t.Error("git's global config changed")
	}
}

func TestBringMyNotesThatFailsSavesNothingAndExitsOne(t *testing.T) {
	f := newFakeGH(t, true)
	g := newGreeting(t)
	writeFile(t, f.dir, "listfail", "")
	dir, code, out := g.run(t, "2\n\n\n")
	if dir != "" || code != 1 {
		t.Errorf("got %q exit %d, want nothing opened, exit 1:\n%s", dir, code, out)
	}
	wantOutput(t, out, "gh could not list your repositories")
	g.nothingSaved(t)
}

// Bring my notes that fails after creating ~/Notes and its repository
// removes both, says so, and the next Bring proposes ~/Notes again.
func TestBringMyNotesThatFailsRemovesWhatItCreated(t *testing.T) {
	f := newFakeGH(t, true)
	g := newGreeting(t)
	cutHistory(t, f.remote(t, "gone")) // setup reads its tip; bringing it fails
	f.setRepos(t, `[{"nameWithOwner":"ana/gone","visibility":"PRIVATE","url":"https://github.com/ana/gone"}]`)
	_, code, out := g.run(t, "2\n\n\n1\n")
	if code != 1 {
		t.Fatalf("exit %d, want 1:\n%s", code, out)
	}
	notes := filepath.Join(g.home, "Notes")
	wantOutput(t, out, "Created the folder "+notes, "The folder "+notes+" it created was removed", "The git repository it created was removed")
	if _, err := os.Lstat(notes); !os.IsNotExist(err) {
		t.Errorf("~/Notes was left behind (%v)", err)
	}
	if got := freeFolder(g.home, defaultNotes); got != "~/Notes" {
		t.Errorf("next proposal %q, want ~/Notes", got)
	}
}

// Bring my notes on a computer with no git identity and no GitHub
// sign-in: Enter at every question, the repository's number aside,
// brings the notes.
func TestBringMyNotesWithEnterAtEveryQuestionBringsThem(t *testing.T) {
	f := newFakeGH(t, false)
	f.writeGlobal(t, false)
	writeFile(t, f.dir, "login", "ok")
	writeFile(t, f.dir, "user.json", anaUser)
	g := newGreeting(t)
	seedRemote(t, f.remote(t, "notes"), map[string]string{"from-laptop.md": "written on the laptop\n"})
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`)
	// 2) Bring; the folder, Start, sign in, name, email, save them:
	// Enter; 1) ana/notes.
	dir, code, out := g.run(t, "2\n\n\n\n\n\n\n1\n")
	want := filepath.Join(g.home, "Notes")
	if dir != want || code != 0 {
		t.Fatalf("got %q exit %d, want %q:\n%s", dir, code, want, out)
	}
	if got := vaultFiles(t, want)["from-laptop.md"]; got != "written on the laptop\n" {
		t.Errorf("from-laptop.md = %q, want the repository's note", got)
	}
}
