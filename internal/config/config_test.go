package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/config"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	c, err := config.Load(path)
	if err != nil || c.Theme != "" {
		t.Fatalf("missing file should load empty config, got %+v %v", c, err)
	}
	if err := config.Save(path, config.Config{Theme: "gruvbox"}); err != nil {
		t.Fatal(err)
	}
	c, err = config.Load(path)
	if err != nil || c.Theme != "gruvbox" {
		t.Fatalf("round trip got %+v %v", c, err)
	}
}

func TestThemesDirRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{Theme: "onedark", ThemesDir: `C:\themes\mine`}); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.ThemesDir != `C:\themes\mine` {
		t.Errorf("ThemesDir = %q", c.ThemesDir)
	}
}

func TestLoadCorruptConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte("{oops"), 0o644)
	if _, err := config.Load(path); err == nil {
		t.Error("expected error for corrupt JSON")
	}
}

func TestLoadEmptyPath(t *testing.T) {
	c, err := config.Load("")
	if err != nil || c.Theme != "" || c.ThemesDir != "" {
		t.Errorf("empty path should give empty config, got %+v %v", c, err)
	}
}

func TestSaveEmptyPathNoop(t *testing.T) {
	if err := config.Save("", config.Config{Theme: "x"}); err != nil {
		t.Errorf("Save with empty path should be a no-op, got %v", err)
	}
}

func TestDefaultPath(t *testing.T) {
	if p := config.DefaultPath(); p == "" {
		t.Skip("no user config dir on this platform")
	} else if !filepath.IsAbs(p) {
		t.Errorf("DefaultPath = %q, want absolute path", p)
	}
}

func TestSaveToBadPath(t *testing.T) {
	if err := config.Save(filepath.Join(t.TempDir(), string(filepath.Separator)+"\x00bad"), config.Config{}); err == nil {
		t.Skip("OS accepted this path")
	}
}

func TestKeysAreReadAndSurviveASave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{"theme": "onedark", "keys": {"toggle_explorer": ["space e"], "close_tab": []}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Keys["toggle_explorer"]; len(got) != 1 || got[0] != "space e" {
		t.Fatalf("keys not read: %v", cfg.Keys)
	}
	if got, ok := cfg.Keys["close_tab"]; !ok || len(got) != 0 {
		t.Fatalf("an empty list must be kept (it unbinds the action): %v", cfg.Keys)
	}

	// The theme picker loads, changes the theme and saves: keys must survive.
	cfg.Theme = "nord"
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	again, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.Theme != "nord" || len(again.Keys["toggle_explorer"]) != 1 {
		t.Fatalf("saving the theme lost the keys: %+v", again)
	}
	if _, ok := again.Keys["close_tab"]; !ok {
		t.Fatalf("saving dropped the unbound action: %+v", again.Keys)
	}
}

func TestUISectionIsRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{"ui": {"heading_icons": ["① ", "② "], "heading_background": true,
	  "bullets": ["●", "○"], "checkbox": {"unchecked": "󰄱", "checked": "󰱒", "todo": "󰥔"}}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	ui := cfg.UI
	if len(ui.HeadingIcons) != 2 || ui.HeadingIcons[1] != "② " || !ui.HeadingBackground ||
		len(ui.Bullets) != 2 || ui.Checkbox.Checked != "󰱒" || ui.Checkbox.Todo != "󰥔" {
		t.Fatalf("ui not read: %+v", ui)
	}
}
