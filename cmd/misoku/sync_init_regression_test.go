package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A common git setting rewrites the URL the user types (for example
// https://github.com/ to git@github.com:). Running init again with the
// same URL must still say the folder is already set up, not refuse it as
// another origin; and no credential is ever printed on any path.
func TestSyncInitAlreadySetUpThroughInsteadOf(t *testing.T) {
	needGit(t)
	remote, dir := newRemote(t), t.TempDir()
	writeFile(t, dir, "a.md", "a\n")
	typed := "https://me:s3cret-tok@example.com/notes.git"
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "url."+filepath.ToSlash(remote)+".insteadOf")
	t.Setenv("GIT_CONFIG_VALUE_0", typed)
	cfg := filepath.Join(t.TempDir(), "config.json")

	code, out, errOut := initVault(t, dir, typed, cfg)
	if code != 0 {
		t.Fatalf("first init: exit %d\n%s\n%s", code, out, errOut)
	}
	if strings.Contains(out+errOut, "s3cret") {
		t.Errorf("credentials printed:\n%s%s", out, errOut)
	}
	if data, _ := os.ReadFile(cfg); strings.Contains(string(data), "s3cret") {
		t.Errorf("credentials written to the config:\n%s", data)
	}
	before := treeHash(t, dir)
	code, out, errOut = initVault(t, dir, typed, cfg)
	if code != 0 || !strings.Contains(out, "Already set up") {
		t.Errorf("second init with the same URL: exit %d, want 0 and \"Already set up\"\nstdout:\n%s\nstderr:\n%s", code, out, errOut)
	}
	if strings.Contains(out+errOut, "s3cret") {
		t.Errorf("credentials printed:\n%s%s", out, errOut)
	}
	if treeHash(t, dir) != before {
		t.Error("the vault changed")
	}
}

// Dotfiles managers (GNU Stow, chezmoi symlink mode) keep the config as a
// symlink into the dotfiles repository: listing the vault must update
// that file, not replace the link with a copy.
func TestSyncInitKeepsASymlinkedConfig(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote, dir := newRemote(t), t.TempDir()
	writeFile(t, dir, "a.md", "a\n")
	dotfiles := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(dotfiles, []byte(`{"theme":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.Symlink(dotfiles, cfg); err != nil {
		t.Skip("no symlinks here:", err)
	}
	if code, out, errOut := initVault(t, dir, remote, cfg); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errOut)
	}
	if info, err := os.Lstat(cfg); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the config symlink was replaced by a regular file (%v)", err)
	}
	if data, _ := os.ReadFile(dotfiles); !strings.Contains(string(data), "vaults") || !strings.Contains(string(data), `"theme"`) {
		t.Errorf("the dotfiles copy did not get the vault:\n%s", data)
	}
}

// A config the user made private stays private.
func TestSyncInitKeepsTheConfigMode(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote, dir := newRemote(t), t.TempDir()
	writeFile(t, dir, "a.md", "a\n")
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfg, []byte(`{"theme":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(cfg, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, errOut := initVault(t, dir, remote, cfg); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errOut)
	}
	info, err := os.Stat(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if want := os.FileMode(0o600); info.Mode().Perm() != want && os.PathSeparator == '/' {
		t.Errorf("config mode %v, want %v", info.Mode().Perm(), want)
	}
}

// A GitHub-made remote (README, LICENSE, .gitignore) joined with a vault
// that has its own README and .gitignore: every byte of both sides is
// still somewhere in the vault and on origin, and the rest of the config
// survives.
func TestSyncInitGitHubRemoteLosesNothing(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote, dir := newRemote(t), t.TempDir()
	seedRemote(t, remote, map[string]string{"README.md": "# notas\n", "LICENSE": "MIT\n", ".gitignore": "*.log\n"})
	writeFile(t, dir, "README.md", "mi readme\r\n")
	writeFile(t, dir, ".gitignore", "local\n")
	writeFile(t, dir, "keep.log", "ignored by origin's rules\n")
	writeFile(t, dir, "sub/ñandú.md", "nota\n")
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfg, []byte(`{"keymaps":{"<C-s>":"save"},"x":"日本"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, out, errOut := initVault(t, dir, remote, cfg); code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, out, errOut)
	}
	wantSynced(t, dir, remote)
	for name, files := range map[string]map[string]string{"vault": vaultFiles(t, dir), "origin": remoteFiles(t, remote)} {
		var all []string
		for _, v := range files {
			all = append(all, v)
		}
		joined := strings.Join(all, "\x00")
		for _, want := range []string{"# notas\n", "MIT\n", "*.log\n", "mi readme\r\n", "local\n", "ignored by origin's rules\n", "nota\n"} {
			if !strings.Contains(joined, want) {
				t.Errorf("%s lost %q; files: %v", name, want, sortedKeys(files))
			}
		}
	}
	// Read as JSON: on Windows the path's backslashes are escaped in the
	// file, so the raw text never holds the path as written.
	data, _ := os.ReadFile(cfg)
	var got struct {
		Keymaps map[string]string `json:"keymaps"`
		X       string            `json:"x"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("config is not valid JSON: %v\n%s", err, data)
	}
	if got.Keymaps["<C-s>"] != "save" || got.X != "日本" {
		t.Errorf("other settings changed: %+v\n%s", got, data)
	}
	listed := false
	for key := range listedVaults(t, cfg) {
		a, errA := os.Stat(key)
		b, errB := os.Stat(dir)
		if filepath.Clean(key) == filepath.Clean(dir) || (errA == nil && errB == nil && os.SameFile(a, b)) {
			listed = true
		}
	}
	if !listed {
		t.Errorf("sync.vaults lacks %s:\n%s", dir, data)
	}
}

// Every refusal leaves the folder (its .git included) and the config
// byte for byte.
func TestSyncInitRefusalsChangeNothing(t *testing.T) {
	t.Parallel()
	needGit(t)
	for name, setup := range map[string]func(t *testing.T, dir string){
		"detached HEAD": func(t *testing.T, dir string) {
			git(t, dir, "init", "-q", "-b", "main")
			writeFile(t, dir, "a.md", "a\n")
			git(t, dir, "add", "-A")
			git(t, dir, "commit", "-qm", "a")
			writeFile(t, dir, "a.md", "b\n")
			git(t, dir, "commit", "-qam", "b")
			git(t, dir, "checkout", "-q", "HEAD~1")
			writeFile(t, dir, "new.md", "untracked\n")
		},
		"index.lock": func(t *testing.T, dir string) {
			git(t, dir, "init", "-q", "-b", "main")
			writeFile(t, dir, "a.md", "a\n")
			writeFile(t, dir, ".git/index.lock", "")
		},
		"inside another repository": func(t *testing.T, dir string) {
			git(t, filepath.Dir(dir), "init", "-q", "-b", "main")
			writeFile(t, dir, "a.md", "a\n")
		},
	} {
		t.Run(name, func(t *testing.T) {
			remote := newRemote(t)
			dir := filepath.Join(t.TempDir(), "vault")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			setup(t, dir)
			cfg := filepath.Join(t.TempDir(), "config.json")
			cfgData := []byte("{\n  \"theme\": \"x\", \"keymaps\": {\"<C-s>\": \"save\"}\n}\n")
			if err := os.WriteFile(cfg, cfgData, 0o644); err != nil {
				t.Fatal(err)
			}
			before, remoteBefore := treeHash(t, filepath.Dir(dir)), treeHash(t, remote)
			code, out, errOut := initVault(t, dir, remote, cfg)
			if code != 1 {
				t.Fatalf("exit %d, want 1\n%s\n%s", code, out, errOut)
			}
			if treeHash(t, filepath.Dir(dir)) != before {
				t.Error("the folder changed")
			}
			if treeHash(t, remote) != remoteBefore {
				t.Error("origin changed")
			}
			if got, _ := os.ReadFile(cfg); string(got) != string(cfgData) {
				t.Errorf("config changed:\n%s", got)
			}
		})
	}
}

// With core.autocrlf=true (Git for Windows' default) the first local
// commit would turn CRLF into LF: init writes .gitattributes before it,
// taking origin's own when origin has one, so no note changes and no
// conflict copy of .gitattributes appears.
func TestSyncInitKeepsLineEndingsWithAutocrlf(t *testing.T) {
	needGit(t)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.autocrlf")
	t.Setenv("GIT_CONFIG_VALUE_0", "true")
	for name, c := range map[string]struct {
		origin    map[string]string
		wantAttrs string
	}{
		"origin without .gitattributes": {map[string]string{"README.md": "# notes\n"}, wantAttributes},
		"origin with its own":           {map[string]string{"README.md": "# notes\n", ".gitattributes": "* -text\n"}, "* -text\n"},
	} {
		t.Run(name, func(t *testing.T) {
			remote := newRemote(t)
			seedRemote(t, remote, c.origin)
			dir := t.TempDir()
			writeFile(t, dir, "mine.md", "crlf\r\nline\r\n")
			if code, out, errOut := initVault(t, dir, remote, filepath.Join(t.TempDir(), "c.json")); code != 0 {
				t.Fatalf("exit %d\n%s\n%s", code, out, errOut)
			}
			want := map[string]string{"README.md": "# notes\n", "mine.md": "crlf\r\nline\r\n", ".gitattributes": c.wantAttrs}
			wantFiles(t, "origin", remoteFiles(t, remote), want)
			wantFiles(t, "vault", vaultFiles(t, dir), want)
			wantSynced(t, dir, remote)
		})
	}
}

// --dir naming a folder that does not exist yet, in a folder that does:
// it is the "another device" case, so init creates it. A missing parent
// is refused, creating nothing.
func TestSyncInitCreatesAMissingVaultFolder(t *testing.T) {
	t.Parallel()
	needGit(t)
	remote, first := newRemote(t), t.TempDir()
	writeFile(t, first, "a.md", "a\n")
	if code, out, errOut := initVault(t, first, remote, filepath.Join(t.TempDir(), "c.json")); code != 0 {
		t.Fatalf("first device: exit %d\n%s\n%s", code, out, errOut)
	}

	parent := t.TempDir()
	dir := filepath.Join(parent, "notes")
	code, stdout, stderr := initVault(t, dir, remote, filepath.Join(t.TempDir(), "c.json"))
	if code != 0 {
		t.Fatalf("exit %d\nstdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	wantFiles(t, "vault", vaultFiles(t, dir), map[string]string{"a.md": "a\n", ".gitattributes": wantAttributes})
	wantSynced(t, dir, remote)
	wantOutput(t, stdout, "Created the folder "+dir)

	missing := filepath.Join(parent, "no", "such")
	code, _, stderr = initVault(t, missing, remote, filepath.Join(t.TempDir(), "c.json"))
	if code != 1 {
		t.Fatalf("missing parent: exit %d, want 1", code)
	}
	wantOutput(t, stderr, "does not exist")
	if _, err := os.Stat(filepath.Join(parent, "no")); !os.IsNotExist(err) {
		t.Errorf("nothing may be created when the parent is missing: %v", err)
	}
}

func TestSyncInitHelpExitsZero(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"sync", "--help"}, {"sync", "-h"}, {"sync", "init", "-h"}, {"sync", "init", "--help"}} {
		code, out := runMain(t, args...)
		if code != 0 {
			t.Errorf("misoku %s: exit %d, want 0:\n%s", strings.Join(args, " "), code, out)
		}
		wantOutput(t, out, "Usage: misoku sync init <url>")
	}
}
