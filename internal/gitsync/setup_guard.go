package gitsync

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pwn27sbx/misoku/internal/core"
)

// The guard before an existing repository is used (issue #7): a pick in a
// long list once connected a notes folder to a website's repository, and
// the first sync pushed the notes to the branch that deploys the site.
// Before git's access is set up and before InitVault fetches anything into
// the folder, setup looks at the tip of the repository's default branch in
// a throwaway repository of its own. A code project or a site is refused;
// notes that do not share the folder's history, and a repository mostly of
// other files, are used only once the user confirmed (AskUnrelated). A
// repository it cannot read stops the run.

// anyDepthFiles make a repository a code project wherever they are.
var anyDepthFiles = []string{"package.json", "go.mod", "Cargo.toml"}

// siteConfigs are the config files of a site generator (Jekyll, Hugo,
// MkDocs), at the top level.
var siteConfigs = []string{"_config.yml", "config.toml", "hugo.toml", "hugo.yaml", "hugo.json", "mkdocs.yml"}

// topFiles make a repository a code project or a site at the top level.
var topFiles = slices.Concat([]string{"pyproject.toml", "vercel.json", "Dockerfile"}, siteConfigs,
	[]string{"netlify.toml", "CNAME", "Gemfile", ".gitmodules"})

// codeDirs are top-level folders that make a repository a code project,
// shown as named here.
var codeDirs = []string{"src", "app", ".github/workflows"}

// siteDirs are a site's top-level folders; a notes vault may have folders
// of these names, so they count only next to a site's config file.
var siteDirs = []string{"themes", "layouts", "_layouts", "_posts"}

// noteExts are the files a vault is made of: notes, PDFs, and the images,
// videos and audio Misoku's "/" menu attaches.
var noteExts = func() map[string]bool {
	exts := map[string]bool{".md": true, ".pdf": true}
	for _, list := range [][]string{core.ImageExts, core.VideoExts, core.AudioExts} {
		for _, e := range list {
			exts[e] = true
		}
	}
	return exts
}()

// gitHousekeeping are files git itself reads; `misoku sync init` writes
// .gitattributes, so they never count against a notes repository.
var gitHousekeeping = map[string]bool{".gitattributes": true, ".gitignore": true}

// remoteLook is what the guard saw on a repository's default branch.
type remoteLook struct {
	// empty: the repository has no commits.
	empty bool
	// related: the folder's history already holds the branch's tip.
	related bool
	// code lists the signs of a code project or a site found ("package.json",
	// ".github/workflows"); empty for notes.
	code []string
	// other says how many files are neither notes nor attachments ("37 of
	// 40 files are not notes") when they are more than a fifth; "" else.
	other string
	// top lists the branch's top-level entries, folders ending in "/".
	top []string
}

// codeSigns lists the signs of a code project or a site among files
// (paths on a branch), in the words of the refusal: the files and folders
// above, node_modules anywhere.
func codeSigns(files []string) []string {
	has := map[string]bool{}
	for _, f := range files {
		has[f] = true
		parts := strings.Split(f, "/")
		if len(parts) > 1 {
			has[parts[0]+"/"] = true
		}
		if strings.HasPrefix(f, ".github/workflows/") {
			has[".github/workflows/"] = true
		}
		for _, p := range parts {
			if p == "node_modules" {
				has["node_modules/"] = true
			}
		}
		for _, name := range anyDepthFiles {
			if parts[len(parts)-1] == name {
				has["*"+name] = true
			}
		}
	}
	var signs []string
	for _, name := range anyDepthFiles {
		if has["*"+name] {
			signs = append(signs, name)
		}
	}
	site := false
	for _, name := range topFiles {
		if has[name] {
			signs = append(signs, name)
		}
	}
	for _, name := range siteConfigs {
		site = site || has[name]
	}
	dirs := slices.Concat(codeDirs, []string{"node_modules"})
	if site {
		dirs = slices.Concat(dirs, siteDirs)
	}
	for _, d := range dirs {
		if has[d+"/"] {
			signs = append(signs, d)
		}
	}
	return signs
}

// otherFiles says how many of files are neither notes nor attachments,
// when they are more than a fifth of them, or "" (git's own files aside).
func otherFiles(files []string) string {
	other, counted := 0, 0
	for _, f := range files {
		if gitHousekeeping[f] {
			continue
		}
		counted++
		if !noteExts[strings.ToLower(path.Ext(f))] {
			other++
		}
	}
	if counted == 0 || other*5 <= counted {
		return ""
	}
	return fmt.Sprintf("%d of %d files are not notes", other, counted)
}

// topLevel lists the first path element of files, once each, folders
// ending in "/", in order.
func topLevel(files []string) []string {
	seen := map[string]bool{}
	for _, f := range files {
		name, rest, dir := strings.Cut(f, "/")
		if dir && rest != "" {
			name += "/"
		}
		seen[name] = true
	}
	return sortedKeys(seen)
}

// lookAt fetches the tip of repo's default branch, alone, into a throwaway
// repository and reads its files; the folder, its repository and git's
// settings are not changed. git reaches GitHub through gh's helper, given
// to these commands alone (setupGit runs only once repo is taken).
// related compares that tip with the folder's own history.
func (s *Setup) lookAt(ctx context.Context, repo Repo) (remoteLook, error) {
	tmp, err := os.MkdirTemp("", "misoku-look-")
	if err != nil {
		return remoteLook{}, err
	}
	defer os.RemoveAll(tmp) // best effort: a throwaway copy
	r := New(tmp, s.opts.InitTimeout)
	url := repo.cloneURL()
	// The access: gh's helper alone, for these commands only. Unscoped,
	// so it also answers when the user's config rewrites github.com's
	// address (url.insteadOf): the empty value first drops every other
	// helper, which setup does not use.
	helper := []string{
		"-c", "credential.helper=",
		"-c", "credential.helper=!" + shellQuote(filepath.ToSlash(s.opts.GH)) + " auth git-credential",
	}
	branch, empty, err := defaultBranch(ctx, r, url, helper...)
	if err != nil {
		return remoteLook{}, err
	}
	if empty {
		return remoteLook{empty: true}, nil
	}
	if _, err := r.Run(ctx, "init", "--quiet", "--bare"); err != nil {
		return remoteLook{}, err
	}
	if _, err := r.Run(ctx, append(helper, "fetch", "--quiet", "--depth=1", "--no-tags", url, "refs/heads/"+branch)...); err != nil {
		return remoteLook{}, err
	}
	tip, err := resolve(ctx, r, "FETCH_HEAD")
	if err != nil {
		return remoteLook{}, err
	}
	tree, err := lsTree(ctx, r, "FETCH_HEAD")
	if err != nil {
		return remoteLook{}, err
	}
	files := sortedKeys(tree)
	return remoteLook{related: s.holds(ctx, tip), code: codeSigns(files), other: otherFiles(files), top: topLevel(files)}, nil
}

// holds reports whether the folder's own repository already has the
// commit tip: the repository is then this vault's history, not another.
func (s *Setup) holds(ctx context.Context, tip string) bool {
	if _, err := os.Stat(filepath.Join(s.opts.Dir, ".git")); err != nil {
		return false
	}
	_, err := New(s.opts.Dir, s.opts.Timeout).Run(ctx, "cat-file", "-e", tip+"^{commit}")
	return err == nil
}

// hasNotes reports whether the folder holds anything but git's own
// folder: notes a repository with another history would be mixed with.
func (s *Setup) hasNotes() bool {
	entries, err := os.ReadDir(s.opts.Dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			return true
		}
	}
	return false
}

// guard looks at repo before the vault uses it. ok is true when repo can be
// used right away; otherwise step is what comes next: the list again,
// saying why a code project is refused; the question before mixing
// (AskUnrelated); or the end of the run, when repo could not be read.
func (s *Setup) guard(ctx context.Context, repo Repo) (step SetupStep, ok bool) {
	look, err := s.lookAt(ctx, repo)
	switch {
	case err != nil:
		return s.fail(ctx,
			fmt.Sprintf("Misoku could not read %s to check it before using it (%s).", repo.Name, firstLine(redact(err).Error())),
			"Nothing was fetched into this folder or pushed. Check your connection and your access to it, then run setup again."), false
	case look.empty || look.related:
		return SetupStep{}, true
	case len(look.code) > 0:
		return s.askRepo(fmt.Sprintf("%s looks like a code project (%s): Misoku won't mix your notes into it. Pick another repository or create a new private one.", repo.Name, strings.Join(look.code, ", "))), false
	case look.other == "" && !s.hasNotes():
		// Notes for a new computer's empty folder.
		return SetupStep{}, true
	}
	s.at, s.pending = AskUnrelated, repo
	return unrelatedStep(repo, look), false
}

// firstLine is the first line of git's message.
func firstLine(msg string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(msg), "\n")
	return strings.TrimSpace(line)
}

// listedTop is how many top-level entries AskUnrelated lists by name.
const listedTop = 12

// unrelatedStep asks before using repo, whose files are not this folder's
// notes (AskUnrelated): it lists repo's top level, and says when most of
// its files are not notes.
func unrelatedStep(repo Repo, look remoteLook) SetupStep {
	list, more := look.top, ""
	if len(list) > listedTop {
		list, more = list[:listedTop], fmt.Sprintf(", and %d more", len(look.top)-listedTop)
	}
	lines := []string{fmt.Sprintf("%s does not share this folder's history. It holds: %s%s.", repo.Name, strings.Join(list, ", "), more)}
	if look.other != "" {
		lines = append(lines, fmt.Sprintf("It may not be a notes repository: %s.", look.other))
	}
	lines = append(lines,
		"Using it puts both together: its files come into this folder, and your notes are pushed to it.",
		"If it is not where your notes belong, cancel and create a new private repository.")
	return SetupStep{Ask: AskUnrelated, Title: "This repository already has other files", Repo: repo.Name,
		Confirm: fmt.Sprintf("Mix this folder's notes with %s?", repo.Name), Lines: lines}
}
