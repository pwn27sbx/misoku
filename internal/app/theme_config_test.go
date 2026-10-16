package app_test

// Changing the theme in the app (config.Save)
// keeps everything else in an existing config.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/pwn27sbx/misoku/internal/app"
)

// v070Config is a config a v0.7.0 user may have, plus keys a newer
// Misoku wrote (notes_dir) or that no version knows.
const v070Config = `{
  "theme": "onedark",
  "themes_dir": "~/themes",
  "keys": {"toggle_explorer": ["space e"], "save": ["<C-s>"]},
  "ui": {"heading_icons": ["# "], "bullets": ["-"], "checkbox": {"checked": "x"}, "border": "single"},
  "sync": {"vaults": {"~/notes": {"interval": "5m", "device": "mac"}}},
  "notes_dir": "~/Notes",
  "future_setting": {"a": [1, 2, 3]},
  "another": "kept"
}
`

func colorscheme(t *testing.T, cfgPath, name string) *harness {
	t.Helper()
	h := newHarness(t, newTestStore(t), app.WithConfigPath(cfgPath))
	h.keys("n")
	h.key(tea.KeyEsc)
	h.keys(":colorscheme " + name)
	h.key(tea.KeyEnter)
	return h
}

func topKeys(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("%v:\n%s", err, data)
	}
	return m
}

func TestThemeChangeKeepsEveryOtherSetting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(v070Config), 0o600); err != nil {
		t.Fatal(err)
	}
	colorscheme(t, path, "nord")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, want := topKeys(t, data), topKeys(t, []byte(v070Config))
	want["theme"] = "nord"
	if !reflect.DeepEqual(got, want) {
		t.Errorf("config after a theme change:\n%s\nwant every other key as before", data)
	}
	if bytes.Contains(data, []byte(`\u003c`)) {
		t.Errorf("<C-s> was escaped:\n%s", data)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
			t.Errorf("mode %v, want 0600", info.Mode().Perm())
		}
	}
}

// A Stow-managed config: a relative symlink into a dotfiles folder.
func TestThemeChangeKeepsAStowLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "dotfiles", "misoku", ".config", "misoku", "config.json")
	link := filepath.Join(root, "home", ".config", "misoku", "config.json")
	for _, d := range []string{filepath.Dir(target), filepath.Dir(link)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(target, []byte(v070Config), 0o600); err != nil {
		t.Fatal(err)
	}
	rel, _ := filepath.Rel(filepath.Dir(link), target)
	if err := os.Symlink(rel, link); err != nil {
		t.Fatal(err)
	}
	colorscheme(t, link, "nord")
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the link was replaced (%v)", err)
	}
	if dest, _ := os.Readlink(link); dest != rel {
		t.Errorf("link points to %q, want %q", dest, rel)
	}
	data, _ := os.ReadFile(target)
	if !strings.Contains(string(data), `"nord"`) || !strings.Contains(string(data), "future_setting") {
		t.Errorf("target not updated or lost keys:\n%s", data)
	}
	if info, _ := os.Stat(target); info.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, want 0600", info.Mode().Perm())
	}
	if entries, _ := os.ReadDir(filepath.Dir(link)); len(entries) != 1 {
		t.Errorf("left files beside the link: %v", entries)
	}
}

// A config that is not a JSON object is never rewritten (v0.7.0 replaced
// it with just the theme).
func TestThemeChangeNeverDestroysAnUnreadableConfig(t *testing.T) {
	for _, content := range []string{`{"theme": "onedark", oops}`, `["not", "an", "object"]`, `"text"`} {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		colorscheme(t, path, "nord")
		if data, _ := os.ReadFile(path); string(data) != content {
			t.Errorf("%s was rewritten as:\n%s", content, data)
		}
	}
}
