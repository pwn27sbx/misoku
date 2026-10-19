//go:build !windows

// The guard before an existing repository is used (issue #7), against a
// fake gh (a shell script: macOS and Linux) and local bare repositories
// that git's global config passes off as https://github.com/ana/<name>.
// They change PATH and git's global config, so none runs in parallel.

package gitsync_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// guardFakeGH is signed in and lists $FAKE_GH_DIR/repos.json.
const guardFakeGH = `#!/bin/sh
d="$FAKE_GH_DIR"
printf '%s\n' "$*" >> "$d/calls"
case "$1 $2" in
"--version ") echo "gh version 2.97.0" ;;
"auth status") exit 0 ;;
"auth setup-git") git config --global "credential.https://github.com.helper" "!gh auth git-credential" ;;
"repo list") IFS= read -r line < "$d/repos.json"; printf '%s\n' "$line" ;;
*) echo "fake gh: unknown command $*" >&2; exit 2 ;;
esac
`

// guardGitHub is the fake GitHub: its gh, and the folder of ana's
// repositories.
type guardGitHub struct{ dir, remotes string }

// newGuardGitHub puts the fake gh and git on PATH, with an identity, and
// lists the private repositories names.
func newGuardGitHub(t *testing.T, names ...string) *guardGitHub {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	g := &guardGitHub{dir: t.TempDir()}
	g.remotes = filepath.Join(g.dir, "remotes")
	bin := filepath.Join(g.dir, "bin")
	for _, d := range []string{bin, filepath.Join(g.remotes, "ana")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, filepath.Join(bin, "gh"), guardFakeGH, 0o755)
	if err := os.Symlink(gitPath, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	var list []string
	for _, n := range names {
		list = append(list, `{"nameWithOwner":"ana/`+n+`","visibility":"PRIVATE","url":"https://github.com/ana/`+n+`"}`)
	}
	writeTestFile(t, filepath.Join(g.dir, "repos.json"), "["+strings.Join(list, ",")+"]", 0o644)
	global := filepath.Join(g.dir, "gitconfig")
	writeTestFile(t, global, "[user]\n\tname = Test\n\temail = test@example.com\n[init]\n\tdefaultBranch = main\n"+
		"[url \"file://"+filepath.ToSlash(g.remotes)+"/\"]\n\tinsteadOf = https://github.com/\n", 0o644)
	t.Setenv("FAKE_GH_DIR", g.dir)
	t.Setenv("PATH", bin)
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	return g
}

func writeTestFile(t *testing.T, p, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

// runGit runs git in dir and returns its output, trimmed.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// repo makes ana/name, bare, with files on main (none: empty), and
// returns its folder.
func (g *guardGitHub) repo(t *testing.T, name string, files ...string) string {
	t.Helper()
	bare := filepath.Join(g.remotes, "ana", name+".git")
	runGit(t, g.dir, "init", "-q", "--bare", "-b", "main", bare)
	if len(files) == 0 {
		return bare
	}
	work := t.TempDir()
	runGit(t, work, "init", "-q", "-b", "main")
	for _, f := range files {
		writeTestFile(t, filepath.Join(work, f), "content of "+f+"\n", 0o644)
	}
	runGit(t, work, "add", "-A")
	runGit(t, work, "commit", "-q", "-m", "seed")
	runGit(t, work, "push", "-q", bare, "main")
	return bare
}

// toRepoList starts the assistant on dir and answers up to the list.
func toRepoList(t *testing.T, dir string) *gitsync.Setup {
	t.Helper()
	s := gitsync.NewSetup(gitsync.SetupOptions{
		Dir: dir, ConfigPath: filepath.Join(t.TempDir(), "config.json"), Timeout: 30 * time.Second,
	})
	ctx := context.Background()
	if step := s.Begin(ctx); step.Ask != gitsync.AskStart {
		t.Fatalf("begin: %+v", step)
	}
	if step := s.Next(ctx, gitsync.SetupAnswer{OK: true}); step.Ask != gitsync.AskRepo {
		t.Fatalf("want the list, got %+v", step)
	}
	return s
}

// notesFolder is a folder holding one note.
func notesFolder(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "mine.md"), "my note\n", 0o644)
	return dir
}

// untouched checks the folder got no repository and bare kept its main.
func untouched(t *testing.T, dir, bare, mainBefore string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Errorf("the folder must get no repository: %v", err)
	}
	if got := runGit(t, bare, "rev-parse", "main"); got != mainBefore {
		t.Errorf("the repository's main moved: %s, was %s", got, mainBefore)
	}
}

// The incident: a website's repository, picked by mistake, is refused,
// with what gave it away, and nothing is fetched or pushed.
func TestSetupRefusesARepositoryHoldingACodeProject(t *testing.T) {
	g := newGuardGitHub(t, "TRJS")
	bare := g.repo(t, "TRJS", "package.json", ".github/workflows/deploy.yml", "README.md", "content/post.md")
	before := runGit(t, bare, "rev-parse", "main")
	dir := notesFolder(t)
	s := toRepoList(t, dir)
	step := s.Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	want := "ana/TRJS looks like a code project (package.json, .github/workflows): Misoku won't mix your notes into it. Pick another repository or create a new private one."
	if step.Ask != gitsync.AskRepo || step.Problem != want {
		t.Fatalf("want the list again with\n%q\ngot %+v", want, step)
	}
	untouched(t, dir, bare, before)
	// The refusal changed nothing, not even git's access.
	if got := s.StopText("Setup cancelled"); got != "Setup cancelled; nothing was changed." {
		t.Errorf("after a refusal: %q", got)
	}
}

// Every marker of a code project, and a branch mostly of other files, is
// refused on its own.
func TestSetupRefusesEachSignOfACodeProject(t *testing.T) {
	for _, c := range []struct{ file, says string }{
		{"go.mod", "(go.mod)"},
		{"Cargo.toml", "(Cargo.toml)"},
		{"pyproject.toml", "(pyproject.toml)"},
		{"vercel.json", "(vercel.json)"},
		{"Dockerfile", "(Dockerfile)"},
		{"src/main.md", "(src)"},
		{"app/page.md", "(app)"},
		{"lib/node_modules/x.md", "(node_modules)"},
		{"web/package.json", "(package.json)"},
		{"tools/x/go.mod", "(go.mod)"},
		{"crates/a/Cargo.toml", "(Cargo.toml)"},
		{"_config.yml", "(_config.yml)"},
		{"config.toml", "(config.toml)"},
		{"hugo.toml", "(hugo.toml)"},
		{"hugo.yaml", "(hugo.yaml)"},
		{"hugo.json", "(hugo.json)"},
		{"mkdocs.yml", "(mkdocs.yml)"},
		{"netlify.toml", "(netlify.toml)"},
		{"CNAME", "(CNAME)"},
		{"Gemfile", "(Gemfile)"},
		{".gitmodules", "(.gitmodules)"},
	} {
		t.Run(c.file, func(t *testing.T) {
			g := newGuardGitHub(t, "repo")
			files := []string{c.file, "a.md", "b.md", "c.md", "d.md"}
			bare := g.repo(t, "repo", files...)
			before := runGit(t, bare, "rev-parse", "main")
			dir := notesFolder(t)
			step := toRepoList(t, dir).Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
			if step.Ask != gitsync.AskRepo || !strings.Contains(step.Problem, "looks like a code project "+c.says) {
				t.Fatalf("want a refusal naming %s, got %+v", c.says, step)
			}
			untouched(t, dir, bare, before)
		})
	}
}

// A Jekyll site is refused with its config and its folders.
func TestSetupRefusesAJekyllSite(t *testing.T) {
	g := newGuardGitHub(t, "blog")
	bare := g.repo(t, "blog", "_config.yml", "_posts/2026-01-01-hello.md", "_layouts/default.html", "index.md", "about.md")
	before := runGit(t, bare, "rev-parse", "main")
	dir := notesFolder(t)
	step := toRepoList(t, dir).Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	if step.Ask != gitsync.AskRepo || !strings.Contains(step.Problem, "ana/blog looks like a code project (_config.yml, _layouts, _posts)") {
		t.Fatalf("want the Jekyll site refused, got %+v", step)
	}
	untouched(t, dir, bare, before)
}

// A Hugo site whose theme is a submodule is refused, theme included.
func TestSetupRefusesAHugoSiteWithAThemeSubmodule(t *testing.T) {
	g := newGuardGitHub(t, "site")
	bare := g.repo(t, "site")
	work := t.TempDir()
	runGit(t, work, "init", "-q", "-b", "main")
	for _, f := range []string{"hugo.toml", "content/post.md", ".gitmodules"} {
		writeTestFile(t, filepath.Join(work, f), "x\n", 0o644)
	}
	runGit(t, work, "add", "-A")
	// The theme as git records a submodule: a commit link, no files.
	runGit(t, work, "update-index", "--add", "--cacheinfo", "160000,1111111111111111111111111111111111111111,themes/ananke")
	runGit(t, work, "commit", "-q", "-m", "site")
	runGit(t, work, "push", "-q", bare, "main")
	before := runGit(t, bare, "rev-parse", "main")
	dir := notesFolder(t)
	step := toRepoList(t, dir).Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	if step.Ask != gitsync.AskRepo || !strings.Contains(step.Problem, "ana/site looks like a code project (hugo.toml, .gitmodules, themes)") {
		t.Fatalf("want the Hugo site refused, got %+v", step)
	}
	untouched(t, dir, bare, before)
}

// A vault of notes and the attachments Misoku's "/" menu adds (PDFs,
// videos, audio) is notes: brought to a new computer without a question.
func TestSetupBringsAVaultWithAttachments(t *testing.T) {
	g := newGuardGitHub(t, "r10")
	g.repo(t, "r10", "1.md", "2.md", "3.md", "4.md", "5.md", "6.md",
		"attachments/a.pdf", "attachments/b.pdf", "attachments/c.pdf", "attachments/d.mp4", "attachments/e.m4a")
	dir := filepath.Join(t.TempDir(), "Notes")
	step := toRepoList(t, dir).Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	if step.Outcome != gitsync.SetupDone {
		t.Fatalf("want it brought, got %+v", step)
	}
}

// Folders named like a site's, without a site's config, are notes.
func TestSetupBringsNotesInFoldersNamedLikeASites(t *testing.T) {
	g := newGuardGitHub(t, "notes")
	g.repo(t, "notes", "themes/dark.md", "layouts/grid.md", "_posts/old.md", "index.md")
	step := toRepoList(t, filepath.Join(t.TempDir(), "Notes")).Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	if step.Outcome != gitsync.SetupDone {
		t.Fatalf("want it brought, got %+v", step)
	}
}

// Mostly other files, but no sign of a code project: asked, never refused,
// even for a new computer's empty folder; the question says why.
func TestSetupAsksAboutARepositoryOfMostlyOtherFiles(t *testing.T) {
	g := newGuardGitHub(t, "misc")
	g.repo(t, "misc", "a.md", "list.txt", "data.csv")
	dir := filepath.Join(t.TempDir(), "Notes")
	step := toRepoList(t, dir).Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	if step.Ask != gitsync.AskUnrelated || !strings.Contains(strings.Join(step.Lines, "\n"), "2 of 3 files are not notes") {
		t.Fatalf("want the question, saying how many files are not notes, got %+v", step)
	}
}

// A repository the guard cannot read stops the run: nothing is fetched,
// pushed or set up, and the message says why, on one line.
func TestSetupStopsWhenItCannotReadTheRepository(t *testing.T) {
	g := newGuardGitHub(t, "gone") // listed, but not there
	dir := notesFolder(t)
	s := toRepoList(t, dir)
	step := s.Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	text := strings.Join(step.Lines, "\n")
	if step.Outcome != gitsync.SetupFailed || !strings.Contains(text, "Misoku could not read ana/gone to check it before using it (") || !strings.Contains(text, "Nothing was changed.") {
		t.Fatalf("want the run stopped, got %+v", step)
	}
	if strings.Contains(step.Lines[0], "\n") {
		t.Errorf("the reason must be git's first line only: %q", step.Lines[0])
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Errorf("the folder must get no repository: %v", err)
	}
	calls, _ := os.ReadFile(filepath.Join(g.dir, "calls"))
	if strings.Contains(string(calls), "auth setup-git") {
		t.Error("git's access must not be set up for a repository that was not used")
	}
}

// An empty repository is used as before: the notes are pushed to it.
func TestSetupUsesAnEmptyRepository(t *testing.T) {
	g := newGuardGitHub(t, "notes")
	bare := g.repo(t, "notes")
	step := toRepoList(t, notesFolder(t)).Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	if step.Outcome != gitsync.SetupDone {
		t.Fatalf("want it set up, got %+v", step)
	}
	if got := runGit(t, bare, "ls-tree", "--name-only", "main"); !strings.Contains(got, "mine.md") {
		t.Errorf("the note must be pushed: %q", got)
	}
}

// A notes repository brought to a new computer's empty folder needs no
// question; one fifth of other files (git's own aside) still counts as notes.
func TestSetupBringsANotesRepositoryWithoutAsking(t *testing.T) {
	g := newGuardGitHub(t, "notes")
	g.repo(t, "notes", "a.md", "b.md", "img/c.png", "d.md", "e.txt", ".gitattributes")
	dir := filepath.Join(t.TempDir(), "Notes")
	step := toRepoList(t, dir).Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	if step.Outcome != gitsync.SetupDone {
		t.Fatalf("want it set up, got %+v", step)
	}
	if _, err := os.Stat(filepath.Join(dir, "img", "c.png")); err != nil {
		t.Errorf("the repository's files must be brought: %v", err)
	}
}

// Notes with another history are mixed with the folder's only after a yes:
// the question lists the repository's top level, and a no changes nothing.
func TestSetupAsksBeforeMixingNotesWithAnotherHistory(t *testing.T) {
	for _, yes := range []bool{false, true} {
		t.Run(map[bool]string{false: "no", true: "yes"}[yes], func(t *testing.T) {
			g := newGuardGitHub(t, "journal")
			bare := g.repo(t, "journal", "2026/oct.md", "ideas.md")
			before := runGit(t, bare, "rev-parse", "main")
			dir := notesFolder(t)
			s := toRepoList(t, dir)
			ctx := context.Background()
			step := s.Next(ctx, gitsync.SetupAnswer{OK: true, Choice: 0})
			text := strings.Join(step.Lines, "\n")
			if step.Ask != gitsync.AskUnrelated || step.Repo != "ana/journal" || !strings.Contains(text, "It holds: 2026/, ideas.md.") || step.Confirm == "" {
				t.Fatalf("want the question with the top level, got %+v", step)
			}
			if !yes {
				if step = s.Next(ctx, gitsync.SetupAnswer{}); step.Outcome != gitsync.SetupCancelled {
					t.Fatalf("a no cancels: %+v", step)
				}
				untouched(t, dir, bare, before)
				return
			}
			if step = s.Next(ctx, gitsync.SetupAnswer{OK: true}); step.Outcome != gitsync.SetupDone {
				t.Fatalf("a yes sets it up: %+v", step)
			}
			got := runGit(t, bare, "ls-tree", "-r", "--name-only", "main")
			if !strings.Contains(got, "mine.md") || !strings.Contains(got, "ideas.md") {
				t.Errorf("both sets of notes must be on main: %q", got)
			}
		})
	}
}

// A repository whose tip the folder's own history already holds is this
// vault's: no question.
func TestSetupUsesTheFoldersOwnHistoryWithoutAsking(t *testing.T) {
	g := newGuardGitHub(t, "notes")
	bare := g.repo(t, "notes")
	dir := notesFolder(t)
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "mine")
	runGit(t, dir, "push", "-q", bare, "main")
	step := toRepoList(t, dir).Next(context.Background(), gitsync.SetupAnswer{OK: true, Choice: 0})
	if step.Outcome != gitsync.SetupDone {
		t.Fatalf("want it set up, got %+v", step)
	}
}

// After a refusal, git's access is not set up a second time.
func TestSetupGivesGitTheAccessOnce(t *testing.T) {
	g := newGuardGitHub(t, "site", "notes")
	g.repo(t, "site", "package.json")
	g.repo(t, "notes")
	s := toRepoList(t, notesFolder(t))
	ctx := context.Background()
	if step := s.Next(ctx, gitsync.SetupAnswer{OK: true, Choice: 0}); step.Ask != gitsync.AskRepo {
		t.Fatalf("want a refusal: %+v", step)
	}
	if step := s.Next(ctx, gitsync.SetupAnswer{OK: true, Choice: 1}); step.Outcome != gitsync.SetupDone {
		t.Fatalf("want it set up: %+v", step)
	}
	calls, _ := os.ReadFile(filepath.Join(g.dir, "calls"))
	if n := strings.Count(string(calls), "auth setup-git"); n != 1 {
		t.Errorf("gh auth setup-git ran %d times, want once", n)
	}
	if text := s.StopText("x"); strings.Count(text, "GitHub access for git") != 1 {
		t.Errorf("the change is said once: %q", text)
	}
}
