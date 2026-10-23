package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
)

// legacyConfig is a config.json written before the settings menu existed,
// with a key of the top level and one of "ui" that this version does not
// know (a later version's settings).
const legacyConfig = `{
  "theme": "nord",
  "keys": {"toggle_explorer": ["space e"]},
  "notes_dir": "~/Notes",
  "sync": {"vaults": {"~/Notes": {"interval": "10m"}}},
  "future_top": {"x": [1, 2]},
  "ui": {
    "mode_icon": "󱗞",
    "border": "single",
    "future_ui": "keep me"
  }
}
`

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// decode reads the file at path as generic JSON, to compare meaning
// rather than bytes (a write sorts keys and re-indents).
func decode(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("%v:\n%s", err, data)
	}
	return v
}

// The menu changes one key of "ui": every other key, top level and inside
// "ui", known or not, keeps its value; the file keeps its permissions.
func TestSetUIChangesOnlyTheGivenKey(t *testing.T) {
	path := writeConfig(t, legacyConfig)
	want := decode(t, path)
	changed, err := config.SetUI(path, map[string]any{config.UIAccordion: false})
	if err != nil || !changed {
		t.Fatalf("SetUI = %v, %v; want true, nil", changed, err)
	}
	got := decode(t, path)
	want["ui"].(map[string]any)["accordion"] = false
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("file after SetUI:\n got %v\nwant %v", got, want)
	}
	// Windows has no Unix permission bits to keep.
	if info, _ := os.Stat(path); runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

// A value equal to the one in the file is no change: the file stays byte
// for byte and is not even rewritten.
func TestSetUIWithoutChangeLeavesTheFileAlone(t *testing.T) {
	body := strings.ReplaceAll(legacyConfig, `"border": "single",`, `"border": "single", "bullets": [ "•", "◦" ],`)
	path := writeConfig(t, body)
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	changed, err := config.SetUI(path, map[string]any{config.UIBorder: "single", config.UIBullets: []string{"•", "◦"}})
	if err != nil || changed {
		t.Fatalf("SetUI = %v, %v; want false, nil", changed, err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != body {
		t.Fatalf("file changed:\n%s", data)
	}
	if info, _ := os.Stat(path); !info.ModTime().Equal(old) {
		t.Errorf("file was rewritten (mtime %v)", info.ModTime())
	}
}

// A nil value removes the key (back to Misoku's default); "ui" left empty
// is removed too.
func TestSetUINilRemovesTheKey(t *testing.T) {
	path := writeConfig(t, `{"theme": "nord", "ui": {"bullets": ["-"]}}`)
	if _, err := config.SetUI(path, map[string]any{config.UIBullets: nil, config.UIHeadingIcons: nil}); err != nil {
		t.Fatal(err)
	}
	got := decode(t, path)
	if _, ok := got["ui"]; ok || got["theme"] != "nord" {
		t.Fatalf("file after removing the only ui key: %v", got)
	}
}

// A new file is created, with only the key written.
func TestSetUICreatesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "misoku", "config.json")
	if _, err := config.SetUI(path, map[string]any{config.UIExplorerHideWidth: 90}); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil || cfg.UI.HideWidth() != 90 {
		t.Fatalf("Load = %+v, %v", cfg.UI, err)
	}
}

// A config whose "ui" is not an object, or that is not an object at all,
// is never rewritten: the error says why.
func TestSetUINeverRewritesAnUnreadableConfig(t *testing.T) {
	for _, body := range []string{`{"ui": ["x"]}`, `["not", "an", "object"]`} {
		path := writeConfig(t, body)
		if _, err := config.SetUI(path, map[string]any{config.UIBorder: "rounded"}); err == nil {
			t.Errorf("%s: no error", body)
		}
		if data, _ := os.ReadFile(path); string(data) != body {
			t.Errorf("%s: file changed to %s", body, data)
		}
	}
}

// Loading a config written before the menu existed reads every setting as
// before, and the menu's new settings take their defaults.
func TestLegacyConfigLoadsAsBefore(t *testing.T) {
	cfg, err := config.Load(writeConfig(t, legacyConfig))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "nord" || cfg.NotesDir != "~/Notes" || cfg.UI.ModeIcon != "󱗞" ||
		cfg.UI.Border != "single" || len(cfg.Keys["toggle_explorer"]) != 1 ||
		cfg.Sync.Vaults["~/Notes"].Interval != "10m" {
		t.Fatalf("legacy config read as %+v", cfg)
	}
	if !cfg.UI.AccordionOn() || cfg.UI.HideWidth() != config.DefaultExplorerHideWidth || cfg.UI.Accordion != nil {
		t.Fatalf("new settings should take their defaults: %+v", cfg.UI)
	}
}

// accordion: false survives a theme change (Load + Save), which writes
// "ui" from the Config struct.
func TestAccordionOffSurvivesASave(t *testing.T) {
	path := writeConfig(t, `{"ui": {"accordion": false, "explorer_hide_width": 80}}`)
	cfg, err := config.Load(path)
	if err != nil || cfg.UI.AccordionOn() || cfg.UI.HideWidth() != 80 {
		t.Fatalf("Load = %+v, %v", cfg.UI, err)
	}
	cfg.Theme = "nord"
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	again, _ := config.Load(path)
	if again.UI.AccordionOn() || again.UI.HideWidth() != 80 {
		t.Fatalf("after Save: %+v", again.UI)
	}
}
