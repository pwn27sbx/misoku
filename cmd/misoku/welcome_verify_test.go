// Checks from the independent verification of P2 (the welcome).

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Bring my notes proposes ~/Notes-2 when ~/Notes holds only a hidden
// file, so nothing is mixed with what is there.
func TestBringProposesAnotherFolderWhenNotesHoldsOnlyHiddenFiles(t *testing.T) {
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "Notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "Notes", ".DS_Store"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := freeFolder(home, defaultNotes); got != "~/Notes-2" {
		t.Errorf("freeFolder = %q, want ~/Notes-2", got)
	}
}

// Start my notes never creates a missing parent, and refuses the home
// folder, the root and system folders, asking again each time.
func TestStartMyNotesRefusesUnsafeAndParentlessFolders(t *testing.T) {
	g := newGreeting(t)
	parentless := filepath.Join(g.home, "no", "such")
	dir, code, out := g.run(t, "1\n"+parentless+"\n~\n/\n/etc\nNotes\n")
	if dir != "" || code != 0 {
		t.Fatalf("got %q exit %d, want the user to stop at the end:\n%s", dir, code, out)
	}
	wantOutput(t, out, "does not exist", "is your home folder", "Type a full path", "Nothing was changed.")
	if _, err := os.Stat(filepath.Join(g.home, "no")); err == nil {
		t.Error("the missing parent was created")
	}
	g.nothingSaved(t)
}
