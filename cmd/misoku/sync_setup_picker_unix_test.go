//go:build !windows

// The repository list of `misoku sync setup` (issue #7), against the fake
// gh of sync_setup_unix_test.go.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// privateRepos lists ana's private repositories names, as gh prints them.
func privateRepos(names ...string) string {
	var list []string
	for _, n := range names {
		list = append(list, fmt.Sprintf(`{"nameWithOwner":"ana/%s","visibility":"PRIVATE","url":"https://github.com/ana/%s"}`, n, n))
	}
	return "[" + strings.Join(list, ",") + "]"
}

// Create a new private repository is 1), before the repositories.
func TestSyncSetupListsCreateFirst(t *testing.T) {
	f := newFakeGH(t, true)
	f.setRepos(t, privateRepos("notes", "site"))
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\n1\nfresh\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	create, first := strings.Index(out, "  1) Create a new private repository"), strings.Index(out, "  2) ana/notes")
	if create < 0 || first < 0 || create > first || !strings.Contains(out, "  3) ana/site") {
		t.Errorf("Create must be 1), then the repositories:\n%s", out)
	}
	if got := git(t, dir, "config", "--get", "remote.origin.url"); got != "https://github.com/ana/fresh.git" {
		t.Errorf("1 must create; origin = %q", got)
	}
}

// Part of a name lists the matching repositories with their numbers, and
// the question comes again; a whole number of two digits is read whole.
func TestSyncSetupSearchesTheList(t *testing.T) {
	f := newFakeGH(t, true)
	names := []string{"notes", "site", "notes-old"}
	for i := 1; i <= 9; i++ {
		names = append(names, fmt.Sprintf("r%d", i))
	}
	f.setRepos(t, privateRepos(names...))
	f.remote(t, "r8")
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	code, out := setup(t, dir, cfg, "\nNOTE\n12\n")
	if code != 0 {
		t.Fatalf("exit %d, want 0:\n%s", code, out)
	}
	_, found, _ := strings.Cut(out, `Repositories matching "NOTE":`)
	found, _, _ = strings.Cut(found, "Number")
	if !strings.Contains(found, "2) ana/notes") || !strings.Contains(found, "4) ana/notes-old") || strings.Contains(found, "site") {
		t.Errorf("the search must list the matches with their numbers, and only them:\n%s", out)
	}
	if got := git(t, dir, "config", "--get", "remote.origin.url"); got != "https://github.com/ana/r8.git" {
		t.Errorf("12 must pick entry 12, ana/r8; origin = %q", got)
	}
}

// A code project's repository is refused in the terminal too, and nothing
// changes in the vault, the config or the repository.
func TestSyncSetupRefusesACodeProject(t *testing.T) {
	f := newFakeGH(t, true)
	remote := f.remote(t, "TRJS")
	seedRemote(t, remote, map[string]string{"package.json": "{}\n", ".github/workflows/deploy.yml": "on: push\n", "README.md": "site\n"})
	before := git(t, remote, "rev-parse", "main")
	f.setRepos(t, privateRepos("TRJS"))
	dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
	vaultBefore := f.snapshot(t, dir, cfg) // git's global config included
	code, out := setup(t, dir, cfg, "\n2\n\n")
	if code != 1 {
		t.Errorf("exit %d, want 1:\n%s", code, out)
	}
	wantOutput(t, out, "! ana/TRJS looks like a code project (package.json, .github/workflows): Misoku won't mix your notes into it. Pick another repository or create a new private one.")
	if f.snapshot(t, dir, cfg) != vaultBefore {
		t.Error("the refusal changed the vault, the config or git's access")
	}
	wantOutput(t, out, "Setup cancelled; nothing was changed.")
	if git(t, remote, "rev-parse", "main") != before {
		t.Error("the code project's main moved")
	}
}

// Notes with another history: Enter (no) changes nothing, y mixes them.
func TestSyncSetupAsksBeforeMixingNotes(t *testing.T) {
	for _, c := range []struct {
		answer string
		code   int
	}{{"", 1}, {"y", 0}} {
		t.Run(fmt.Sprintf("answer %q", c.answer), func(t *testing.T) {
			f := newFakeGH(t, true)
			remote := f.remote(t, "journal")
			seedRemote(t, remote, map[string]string{"ideas.md": "idea\n", "2026/oct.md": "october\n"})
			before := git(t, remote, "rev-parse", "main")
			f.setRepos(t, privateRepos("journal"))
			dir, cfg := notesVault(t), filepath.Join(t.TempDir(), "config.json")
			vaultBefore := treeHash(t, dir) + fileHash(t, cfg)
			code, out := setup(t, dir, cfg, "\n2\n"+c.answer+"\n")
			if code != c.code {
				t.Fatalf("exit %d, want %d:\n%s", code, c.code, out)
			}
			wantOutput(t, out, "It holds: 2026/, ideas.md.", "Mix this folder's notes with ana/journal? [y/N]")
			if c.code == 1 {
				if treeHash(t, dir)+fileHash(t, cfg) != vaultBefore || git(t, remote, "rev-parse", "main") != before {
					t.Error("answering no changed something")
				}
				return
			}
			files := remoteFiles(t, remote)
			if files["hello.md"] == "" || files["ideas.md"] == "" {
				t.Errorf("both sets of notes must be on main: %v", files)
			}
		})
	}
}

// cutHistory gives the bare repository a note in two commits, then loses
// the first commit: its tip can still be read (setup's check fetches it
// alone), but bringing its whole history fails, before anything is
// committed in the vault.
func cutHistory(t *testing.T, bare string) {
	t.Helper()
	work := t.TempDir()
	git(t, work, "init", "-q", "-b", "main")
	for _, text := range []string{"first\n", "second\n"} {
		writeFile(t, work, "note.md", text)
		git(t, work, "add", "-A")
		git(t, work, "-c", "user.name=Seed", "-c", "user.email=seed@example.com", "commit", "-q", "-m", text)
	}
	git(t, work, "push", "-q", bare, "main")
	first := git(t, bare, "rev-parse", "main~1")
	if err := os.Remove(filepath.Join(bare, "objects", first[:2], first[2:])); err != nil {
		t.Fatalf("the first commit is not a loose object: %v", err)
	}
}
