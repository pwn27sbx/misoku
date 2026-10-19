//go:build !windows

// The repository list of the setup assistant inside the app (issue #7),
// against the fake gh of sync_setup_unix_test.go.

package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

const listTitle = "Which repository keeps this vault?"

// longList lists n private repositories, ana/r01 to ana/rNN.
func longList(n int) string {
	var repos []string
	for i := 1; i <= n; i++ {
		repos = append(repos, fmt.Sprintf(`{"nameWithOwner":"ana/r%02d","visibility":"PRIVATE","url":"https://github.com/ana/r%02d"}`, i, i))
	}
	return "[" + strings.Join(repos, ",") + "]"
}

// openRepoList starts the assistant, signed in and with an identity, on a
// vault of five notes, and answers Start: the list is open. It returns the
// fake gh's folder and its remotes.
func openRepoList(t *testing.T, repos string) (h *harness, fake, remotes, vault string) {
	t.Helper()
	fake, remotes = fakeGitHub(t, repos, true)
	writeOutside(t, fake, "loggedin", "")
	vault = t.TempDir()
	for i := 1; i <= 5; i++ {
		writeOutside(t, vault, fmt.Sprintf("n%d.md", i), fmt.Sprintf("# Note %d\n\nBODY-OF-NOTE-%d\n", i, i))
	}
	h = window(t, vault, filepath.Join(t.TempDir(), "config.json"))
	h.send(core.SyncSetupMsg{})
	h.key(tea.KeyEnter) // Start
	if !strings.Contains(h.view(), listTitle) {
		t.Fatalf("the repositories must be listed:\n%s", h.view())
	}
	return h, fake, remotes, vault
}

// ghCalled reports whether the fake gh ran a command starting with prefix.
func ghCalled(t *testing.T, fake, prefix string) bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fake, "calls"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

// bareRemote makes ana/name, bare and empty.
func bareRemote(t *testing.T, remotes, name string) string {
	t.Helper()
	bare := filepath.Join(remotes, "ana", name+".git")
	gitIn(t, remotes, "init", "-q", "--bare", "-b", "main", bare)
	return bare
}

// seededRemote makes ana/name with files on main.
func seededRemote(t *testing.T, remotes, name string, files ...string) string {
	t.Helper()
	bare := bareRemote(t, remotes, name)
	work := t.TempDir()
	gitIn(t, work, "init", "-q", "-b", "main")
	for _, f := range files {
		writeOutside(t, work, f, "content of "+f+"\n")
	}
	gitIn(t, work, "add", "-A")
	gitIn(t, work, "-c", "user.name=Seed", "-c", "user.email=seed@example.com", "commit", "-q", "-m", "seed")
	gitIn(t, work, "push", "-q", bare, "main")
	return bare
}

// The incident: on a list of 55, "5" then "4" picks nothing and moves
// nothing behind the dialog; the number waits, shown, for Enter.
func TestRepoListWaitsForTheWholeNumber(t *testing.T) {
	h, fake, remotes, vault := openRepoList(t, longList(54))
	h.keys("5")
	view := h.view()
	if !strings.Contains(view, listTitle) || !strings.Contains(view, "Number: 5_") {
		t.Fatalf("after 5 the list must stay open, showing the number:\n%s", view)
	}
	h.keys("4")
	view = h.view()
	if !strings.Contains(view, listTitle) || !strings.Contains(view, "Number: 54_") || !strings.Contains(view, "54  ana/r53") {
		t.Fatalf("after 54 the list must stay open, showing 54 and its repository:\n%s", view)
	}
	if strings.Contains(view, "BODY-OF-NOTE") {
		t.Fatalf("a digit reached the screen behind the dialog:\n%s", view)
	}
	if ghCalled(t, fake, "auth setup-git") {
		t.Fatal("a repository was picked before Enter")
	}
	if _, err := os.Stat(filepath.Join(vault, ".git")); !os.IsNotExist(err) {
		t.Fatalf("the vault was touched before Enter: %v", err)
	}
	bareRemote(t, remotes, "r53")
	h.key(tea.KeyEnter)
	if got := gitIn(t, vault, "config", "--get", "remote.origin.url"); got != "https://github.com/ana/r53.git" {
		t.Errorf("Enter must pick entry 54, ana/r53; origin = %q", got)
	}
}

// A number the list does not have stays as typed, marked: Enter picks
// nothing (the same trap as "54": "56" must not quietly become 5), and
// Backspace or Esc correct it.
func TestRepoListNumberBeyondTheListPicksNothing(t *testing.T) {
	h, _, _, vault := openRepoList(t, longList(54))
	h.keys("56")
	view := h.view()
	if !strings.Contains(view, "No item 56") || strings.Contains(view, "Number: 5_") {
		t.Fatalf("56 is beyond the list of 55: it must show as typed, marked:\n%s", view)
	}
	h.key(tea.KeyEnter)
	view = h.view()
	if !strings.Contains(view, listTitle) || !strings.Contains(view, "No item 56") {
		t.Fatalf("Enter on a number the list does not have must pick nothing:\n%s", view)
	}
	if _, err := os.Stat(filepath.Join(vault, ".git")); !os.IsNotExist(err) {
		t.Fatalf("Enter on 56 picked a repository: %v", err)
	}
	h.key(tea.KeyBackspace)
	if !strings.Contains(h.view(), "Number: 5_") {
		t.Fatalf("Backspace must leave 5:\n%s", h.view())
	}
}

// Backspace and Esc take a typed number back; 0 is no item.
func TestRepoListNumberCanBeTakenBack(t *testing.T) {
	h, _, _, _ := openRepoList(t, longList(54))
	h.keys("6")
	h.key(tea.KeyBackspace)
	if strings.Contains(h.view(), "Number:") {
		t.Fatalf("Backspace must take the digit back:\n%s", h.view())
	}
	h.keys("0")
	if !strings.Contains(h.view(), "No item 0") {
		t.Fatalf("0 is no item:\n%s", h.view())
	}
	h.key(tea.KeyEsc)
	h.keys("12")
	h.key(tea.KeyEsc)
	view := h.view()
	if !strings.Contains(view, listTitle) || strings.Contains(view, "Number:") || strings.Contains(view, "No item") {
		t.Fatalf("Esc must take the number back and keep the list:\n%s", view)
	}
	h.key(tea.KeyEsc)
	if strings.Contains(h.view(), listTitle) {
		t.Fatalf("a second Esc closes the list:\n%s", h.view())
	}
}

// "/" starts the filter, so digits can be searched for too.
func TestRepoListSlashFiltersDigits(t *testing.T) {
	h, _, _, _ := openRepoList(t, longList(54))
	h.keys("/12")
	view := h.view()
	if !strings.Contains(view, "ana/r12") || strings.Contains(view, "ana/r11") || strings.Contains(view, "Number:") {
		t.Fatalf("/12 must filter by 12:\n%s", view)
	}
	h.key(tea.KeyEsc)
	h.keys("12")
	if !strings.Contains(h.view(), "Number: 12_") {
		t.Fatalf("after Esc, digits are numbers again:\n%s", h.view())
	}
}

// In a short list a digit still picks at once.
func TestShortRepoListPicksADigitAtOnce(t *testing.T) {
	h, _, _, _ := openRepoList(t, longList(3))
	h.keys("1")
	if !strings.Contains(h.view(), "Name of the new private repository") {
		t.Fatalf("1 must pick Create at once in a short list:\n%s", h.view())
	}
}

// Create a new private repository is first, where a long list cannot hide
// it, and Enter on the open list picks it.
func TestRepoListPinsCreateFirst(t *testing.T) {
	h, _, _, _ := openRepoList(t, longList(54))
	view := h.view()
	create, first := strings.Index(view, "1  Create a new private repository"), strings.Index(view, "2  ana/r01")
	if create < 0 || first < 0 || create > first {
		t.Fatalf("Create must be entry 1, before ana/r01:\n%s", view)
	}
	h.key(tea.KeyEnter)
	if !strings.Contains(h.view(), "Name of the new private repository") {
		t.Fatalf("Enter on the open list must pick Create:\n%s", h.view())
	}
}

// Typing filters the list, in any letter case, keeping Create; Backspace
// and Esc take the text back. Digits after a letter are part of the text.
func TestRepoListFiltersAsYouType(t *testing.T) {
	h, _, remotes, vault := openRepoList(t, longList(54))
	h.keys("R5")
	view := h.view()
	for _, want := range []string{"Create a new private repository", "ana/r50", "ana/r54", "R5▏"} {
		if !strings.Contains(view, want) {
			t.Errorf("filtering R5 must list %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "ana/r49") || strings.Contains(view, "ana/r05") {
		t.Fatalf("filtering R5 must hide what does not match:\n%s", view)
	}
	h.keys("3")
	if view = h.view(); !strings.Contains(view, "ana/r53") || strings.Contains(view, "ana/r54") || strings.Contains(view, "Number:") {
		t.Fatalf("a digit after a letter filters:\n%s", view)
	}
	h.key(tea.KeyBackspace)
	if !strings.Contains(h.view(), "ana/r54") {
		t.Fatalf("Backspace must widen the filter again:\n%s", h.view())
	}
	h.keys("xyz")
	if !strings.Contains(h.view(), "Create a new private repository") {
		t.Fatalf("Create stays listed when nothing matches:\n%s", h.view())
	}
	h.key(tea.KeyEsc)
	if view = h.view(); !strings.Contains(view, listTitle) || !strings.Contains(view, "ana/r01") {
		t.Fatalf("Esc must clear the filter and keep the list:\n%s", view)
	}
	h.keys("r07")
	h.key(tea.KeyDown) // past Create
	bareRemote(t, remotes, "r07")
	h.key(tea.KeyEnter)
	if got := gitIn(t, vault, "config", "--get", "remote.origin.url"); got != "https://github.com/ana/r07.git" {
		t.Errorf("the filtered pick must be ana/r07; origin = %q", got)
	}
}

// A code project's repository is refused in the app too: the bar says
// what gave it away and the list comes back.
func TestRepoListRefusesACodeProjectInTheApp(t *testing.T) {
	h, _, remotes, vault := openRepoList(t, longList(3))
	bare := seededRemote(t, remotes, "r02", "package.json", ".github/workflows/deploy.yml", "README.md")
	before := gitIn(t, bare, "rev-parse", "main")
	h.keys("3") // Create, r01, r02
	wantBar(t, h, "ana/r02 looks like a code project (package.json, .github/workflows)")
	if !strings.Contains(h.view(), listTitle) {
		t.Fatalf("the list must come back:\n%s", h.view())
	}
	// The whole refusal is readable in the dialog, not only the bar's start.
	whole := "ana/r02 looks like a code project (package.json, .github/workflows): Misoku won't mix your notes into it. Pick another repository or create a new private one."
	if !strings.Contains(noSpace(dialogText(h.view())), noSpace(whole)) {
		t.Fatalf("the list must show the whole refusal:\n%s", h.view())
	}
	if gitIn(t, bare, "rev-parse", "main") != before {
		t.Error("the code project's main moved")
	}
	if _, err := os.Stat(filepath.Join(vault, ".git")); !os.IsNotExist(err) {
		t.Errorf("the vault must get no repository: %v", err)
	}
}

// Notes with another history are mixed only on a deliberate choice: the
// dialog lists what the repository holds, Cancel is the default, and no
// letter picks.
func TestRepoListAsksBeforeMixingNotesInTheApp(t *testing.T) {
	for _, mix := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "mix"}[mix], func(t *testing.T) {
			h, _, remotes, vault := openRepoList(t, longList(3))
			bare := seededRemote(t, remotes, "r01", "journal/oct.md", "ideas.md")
			before := gitIn(t, bare, "rev-parse", "main")
			h.keys("2")
			text := noSpace(dialogText(h.view()))
			if !strings.Contains(text, noSpace("It holds: ideas.md, journal/.")) || !strings.Contains(text, noSpace("Mix this folder's notes with ana/r01?")) {
				t.Fatalf("the question must list the repository's top level:\n%s", h.view())
			}
			h.keys("m") // no letter picks
			if !strings.Contains(h.view(), "Mix this folder") {
				t.Fatalf("a letter must not answer:\n%s", h.view())
			}
			if !mix {
				h.key(tea.KeyEnter) // Cancel, the default
				wantBar(t, h, "Sync setup cancelled")
				if gitIn(t, bare, "rev-parse", "main") != before {
					t.Error("cancelling moved the repository's main")
				}
				if _, err := os.Stat(filepath.Join(vault, ".git")); !os.IsNotExist(err) {
					t.Errorf("cancelling must leave the vault alone: %v", err)
				}
				return
			}
			h.key(tea.KeyLeft)  // Mix them
			h.key(tea.KeyEnter) // the end
			if !onRemote(t, bare, "n1.md") || !onRemote(t, bare, "ideas.md") {
				t.Error("mixing must keep both sets of notes")
			}
		})
	}
}

// Leaving the list ends the assistant as its Cancel buttons do: the bar
// says so, rather than the list just closing with the run left open.
func TestSyncSetupListEscCancelsTheAssistant(t *testing.T) {
	h, _, _, vault := openRepoList(t, longList(2))
	h.key(tea.KeyEsc)
	if view := h.view(); strings.Contains(view, listTitle) || !strings.Contains(view, "Sync setup cancelled") {
		t.Fatalf("Esc must cancel the assistant:\n%s", view)
	}
	if _, err := os.Stat(filepath.Join(vault, ".git")); !os.IsNotExist(err) {
		t.Errorf("a repository was left in the vault (%v)", err)
	}
}

// Leaving the new repository's name field cancels the assistant too.
func TestSyncSetupNameEscCancelsTheAssistant(t *testing.T) {
	h, fake, _, _ := openRepoList(t, longList(2))
	h.key(tea.KeyEnter) // the new private repository, pinned first
	if !strings.Contains(h.view(), "Name of the new private repository") {
		t.Fatalf("the name must be asked:\n%s", h.view())
	}
	h.key(tea.KeyEsc)
	if !strings.Contains(h.view(), "Sync setup cancelled") {
		t.Fatalf("Esc must cancel the assistant:\n%s", h.view())
	}
	if ghCalled(t, fake, "repo create") {
		t.Error("a repository was created")
	}
}
