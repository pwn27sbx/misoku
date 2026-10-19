//go:build !windows

// Checks from the independent verification of P2 (the welcome).

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A home folder that is itself a git repository (dotfiles kept in ~) is
// the case the plan orders P4 before P2 for: Bring my notes into the
// proposed ~/Notes, which does not exist yet, must bring the notes in a
// repository of their own, as it does when ~/Notes already exists.
func TestBringMyNotesIntoANewFolderOfAHomeThatIsARepository(t *testing.T) {
	f := newFakeGH(t, true)
	g := newGreeting(t)
	git(t, g.home, "init", "-q")
	seedRemote(t, f.remote(t, "notes"), map[string]string{"from-laptop.md": "written on the laptop\n"})
	f.setRepos(t, `[{"nameWithOwner":"ana/notes","visibility":"PRIVATE","url":"https://github.com/ana/notes"}]`)
	// welcome 2, folder Enter, own repository Enter (asked as for an existing
	// folder inside another repository), Start Enter, repo 1. Today no
	// own-repository question comes, so the third Enter cancels at the list;
	// answering "1" there instead fails with "inside the git repository".
	dir, code, out := g.run(t, "2\n\n\n\n1\n")
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
}

// Bring my notes that fails after creating the folder says what it left:
// it must not claim "Nothing else changed" while ~/Notes holds a new .git.
func TestBringMyNotesThatFailsAfterCreatingTheFolderSaysSo(t *testing.T) {
	f := newFakeGH(t, true)
	g := newGreeting(t)
	cutHistory(t, f.remote(t, "gone")) // setup reads its tip; bringing it fails
	f.setRepos(t, `[{"nameWithOwner":"ana/gone","visibility":"PRIVATE","url":"https://github.com/ana/gone"}]`)
	dir, code, out := g.run(t, "2\n\n\n1\n")
	if dir != "" || code != 1 {
		t.Fatalf("got %q exit %d, want nothing opened, exit 1:\n%s", dir, code, out)
	}
	if c, err := os.Stat(g.cfg); err == nil && c.Size() > 0 && g.notesDir(t) != "" {
		t.Errorf("notes_dir was saved after a failure")
	}
	if _, err := os.Stat(filepath.Join(g.home, "Notes", ".git")); err == nil {
		wantOutput(t, out, "Created the folder")
		if strings.Contains(out, "Nothing else changed") {
			t.Errorf("the run left %s/Notes/.git but says nothing else changed:\n%s", g.home, out)
		}
	}
}
