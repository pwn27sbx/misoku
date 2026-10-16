package config_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pwn27sbx/misoku/internal/config"
)

// readTop reads the config file at path as raw JSON keys.
func readTop(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatalf("%v:\n%s", err, data)
	}
	return top
}

// SetNotesDir remembers the notes folder and keeps every other setting,
// those this version does not know included, like AddSyncVault.
func TestSetNotesDirKeepsEveryOtherSetting(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	before := `{
  "theme": "nord",
  "keys": {"save": ["<C-s>"]},
  "not_known_yet": {"a": [1, "two", null]},
  "sync": {"vaults": {"~/old": {"device": "mac"}}}
}`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.SetNotesDir(path, "~/Notes"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	want := `{
  "keys": {
    "save": [
      "<C-s>"
    ]
  },
  "not_known_yet": {
    "a": [
      1,
      "two",
      null
    ]
  },
  "notes_dir": "~/Notes",
  "sync": {
    "vaults": {
      "~/old": {
        "device": "mac"
      }
    }
  },
  "theme": "nord"
}
`
	if string(data) != want {
		t.Errorf("config is now:\n%s\nwant:\n%s", data, want)
	}
	c, err := config.Load(path)
	if err != nil || c.NotesDir != "~/Notes" || c.Theme != "nord" {
		t.Errorf("Load after SetNotesDir = %+v, %v", c, err)
	}
}

// A new config file is created with only notes_dir, and nothing else is
// left in its folder (the temporary file is renamed over it).
func TestSetNotesDirCreatesTheFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "misoku", "config.json")
	dir := filepath.Join(t.TempDir(), "Notes")
	if err := config.SetNotesDir(path, dir); err != nil {
		t.Fatal(err)
	}
	var got string
	if err := json.Unmarshal(readTop(t, path)["notes_dir"], &got); err != nil || got != dir {
		t.Errorf("notes_dir = %q (%v), want %q", got, err, dir)
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Errorf("only config.json may be left, got %v", entries)
	}
}

// A relative folder would depend on where misoku is started: refused,
// and the file is left as it was.
func TestSetNotesDirRefusesARelativeFolder(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	before := `{"theme":"x"}`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"", "notes", "./notes"} {
		if err := config.SetNotesDir(path, dir); err == nil {
			t.Errorf("SetNotesDir(%q) = nil, want an error", dir)
		}
	}
	if data, _ := os.ReadFile(path); string(data) != before {
		t.Errorf("the file must stay as it was, got:\n%s", data)
	}
}

// A config that is not a JSON object is never rewritten by either writer.
func TestNotesDirWritersNeverRewriteAnUnreadableConfig(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "Notes")
	for name, before := range map[string]string{
		"not JSON":       `{"theme": "nord",`,
		"top not object": `[1, 2]`,
	} {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := config.SetNotesDir(path, dir); err == nil {
			t.Errorf("%s: SetNotesDir = nil, want an error", name)
		}
		if err := config.Save(path, config.Config{Theme: "nord"}); err == nil {
			t.Errorf("%s: Save = nil, want an error", name)
		}
		if data, _ := os.ReadFile(path); string(data) != before {
			t.Errorf("%s: the file must stay as it was, got:\n%s", name, data)
		}
	}
}

// Dotfiles managers (GNU Stow) keep the config as a symlink: the target
// is written, the link stays, and a private file stays private.
func TestSetNotesDirKeepsASymlinkedConfigAndItsMode(t *testing.T) {
	t.Parallel()
	dotfiles := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(dotfiles, []byte(`{"theme":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dotfiles, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "config.json")
	if err := os.Symlink(dotfiles, link); err != nil {
		t.Skip("no symlinks here:", err)
	}
	dir := filepath.Join(t.TempDir(), "Notes")
	if err := config.SetNotesDir(link, dir); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the config symlink was replaced by a regular file (%v)", err)
	}
	if _, ok := readTop(t, dotfiles)["notes_dir"]; !ok {
		t.Errorf("the dotfiles copy did not get notes_dir")
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(dotfiles); err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("mode = %v (%v), want 0600", info.Mode().Perm(), err)
		}
	}
}

// Changing the theme (Load, change, Save) keeps the notes folder.
func TestThemeChangeKeepsNotesDir(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	dir := filepath.Join(t.TempDir(), "Notes")
	if err := config.SetNotesDir(path, dir); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	c.Theme = "nord"
	if err := config.Save(path, c); err != nil {
		t.Fatal(err)
	}
	again, err := config.Load(path)
	if err != nil || again.NotesDir != dir || again.Theme != "nord" {
		t.Errorf("after a theme change: %+v, %v", again, err)
	}
}

// Save writes back the keys it does not know, so an older or other
// writer never drops notes_dir or a future setting; sync.vaults stays.
func TestSaveKeepsKeysItDoesNotKnow(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	before := `{"theme":"x","from_the_future":{"a":[1,"<b>"]},"sync":{"vaults":{"~/notes":{"device":"mac"}}}}`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "Notes")
	if err := config.SetNotesDir(path, dir); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	c.Theme = "nord"
	if err := config.Save(path, c); err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, readTop(t, path)["from_the_future"]); err != nil || compact.String() != `{"a":[1,"<b>"]}` {
		t.Errorf("from_the_future = %s (%v)", compact.String(), err)
	}
	again, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.Theme != "nord" || again.NotesDir != dir || again.Sync.Vaults["~/notes"].Device != "mac" {
		t.Errorf("after Save: %+v", again)
	}
}

// A setting Save knows and is told to clear goes away: keeping unknown
// keys must not bring back a known one from the old file.
func TestSaveClearsAKnownSetting(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	before := `{"theme":"x","themes_dir":"/t","other":1}`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(path, config.Config{Theme: "nord"}); err != nil {
		t.Fatal(err)
	}
	top := readTop(t, path)
	if _, ok := top["themes_dir"]; ok {
		t.Errorf("themes_dir came back: %v", top)
	}
	if string(top["theme"]) != `"nord"` || string(top["other"]) != "1" {
		t.Errorf("config is now %v", top)
	}
}

// Save, like before, writes through a symlinked config and keeps its mode.
func TestSaveKeepsASymlinkedConfigAndItsMode(t *testing.T) {
	t.Parallel()
	dotfiles := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(dotfiles, []byte(`{"theme":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dotfiles, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "config.json")
	if err := os.Symlink(dotfiles, link); err != nil {
		t.Skip("no symlinks here:", err)
	}
	if err := config.Save(link, config.Config{Theme: "nord"}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the config symlink was replaced by a regular file (%v)", err)
	}
	if string(readTop(t, dotfiles)["theme"]) != `"nord"` {
		t.Errorf("the dotfiles copy was not written")
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(dotfiles); err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("mode = %v (%v), want 0600", info.Mode().Perm(), err)
		}
	}
}

// notes_dir may start with "~", like sync.vaults; unset means "".
func TestNotesFolderExpandsHome(t *testing.T) {
	home := t.TempDir() // not parallel: HOME is set
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	got, err := config.Config{NotesDir: "~/Notes"}.NotesFolder()
	if want := filepath.Join(home, "Notes"); err != nil || got != want {
		t.Errorf("NotesFolder = %q, %v; want %q", got, err, want)
	}
	if got, err := (config.Config{}).NotesFolder(); err != nil || got != "" {
		t.Errorf("unset NotesFolder = %q, %v; want empty", got, err)
	}
	if _, err := (config.Config{NotesDir: "notes"}).NotesFolder(); err == nil {
		t.Errorf("a relative notes_dir must be an error")
	}
}
