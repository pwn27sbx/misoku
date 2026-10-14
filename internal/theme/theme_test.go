package theme_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/theme"
)

// validPalette is a complete custom palette used across theme tests.
var validPalette = map[string]any{
	"light": false, "fg": "#ffffff", "darker_bg": "#050505", "bg": "#111111",
	"bg2": "#1a1a1a", "one_bg": "#222222", "one_bg2": "#2a2a2a", "one_bg3": "#333333",
	"grey": "#444444", "grey_fg": "#555555", "grey_fg2": "#666666", "light_grey": "#777777",
	"red": "#ff0000", "pink": "#ff00ff", "line": "#222222", "green": "#00ff00",
	"nord_blue": "#0000ff", "blue": "#0088ff", "yellow": "#ffff00", "purple": "#8800ff",
	"dark_purple": "#440088", "teal": "#00ffff", "orange": "#ff8800", "cyan": "#00aaff",
	"status_bg": "#1a1a1a", "light_bg": "#222222", "folder": "#0088ff",
}

func writePalette(t *testing.T, dir, name string, pal map[string]any) string {
	t.Helper()
	data, err := json.Marshal(pal)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBundledThemes(t *testing.T) {
	def := theme.DefaultTheme()
	if def.Name != theme.DefaultName {
		t.Errorf("expected default theme %q, got %q", theme.DefaultName, def.Name)
	}
	for _, name := range []string{"onedark", "darkknight"} {
		th, ok := theme.Get(name)
		if !ok {
			t.Fatalf("expected bundled theme %q to exist", name)
		}
		if th.Light {
			t.Errorf("expected %q to be dark theme", name)
		}
	}
	if _, ok := theme.Get("nonexistent-theme-xyz"); ok {
		t.Error("expected nonexistent theme to return false")
	}
}

func TestUserCustomThemeLoading(t *testing.T) {
	tmpDir := t.TempDir()
	restore := theme.SetUserThemesDirForTest(func() string {
		return tmpDir
	})
	defer restore()

	customTheme := map[string]any{
		"light":       false,
		"fg":          "#ffffff",
		"darker_bg":   "#050505",
		"bg":          "#111111",
		"bg2":         "#1a1a1a",
		"one_bg":      "#222222",
		"one_bg2":     "#2a2a2a",
		"one_bg3":     "#333333",
		"grey":        "#444444",
		"grey_fg":     "#555555",
		"grey_fg2":    "#666666",
		"light_grey":  "#777777",
		"red":         "#ff0000",
		"pink":        "#ff00ff",
		"line":        "#222222",
		"green":       "#00ff00",
		"nord_blue":   "#0000ff",
		"blue":        "#0088ff",
		"yellow":      "#ffff00",
		"purple":      "#8800ff",
		"dark_purple": "#440088",
		"teal":        "#00ffff",
		"orange":      "#ff8800",
		"cyan":        "#00aaff",
		"status_bg":   "#1a1a1a",
		"light_bg":    "#222222",
		"folder":      "#0088ff",
	}

	data, err := json.Marshal(customTheme)
	if err != nil {
		t.Fatalf("failed to marshal custom theme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "custom-test.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write custom theme: %v", err)
	}

	for _, name := range []string{"custom-test", "Custom-Test", "CUSTOM-TEST"} {
		th, ok := theme.Get(name)
		if !ok {
			t.Fatalf("expected custom theme %q to be loaded from disk", name)
		}
		if th.Light {
			t.Errorf("expected %q to be dark theme", name)
		}
		if string(th.Bg) != "#111111" {
			t.Errorf("expected Bg #111111, got %s", th.Bg)
		}
		if string(th.DarkerBg) != "#050505" {
			t.Errorf("expected DarkerBg #050505, got %s", th.DarkerBg)
		}
		if string(th.Blue) != "#0088ff" {
			t.Errorf("expected Blue #0088ff, got %s", th.Blue)
		}
	}

	names := theme.Names()
	found := false
	for _, n := range names {
		if n == "custom-test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'custom-test' in theme.Names()")
	}
}

func TestSampleThemeInTestdataLoads(t *testing.T) {
	restore := theme.SetUserThemesDirForTest(func() string { return "testdata" })
	defer restore()

	th, ok := theme.Get("tokyo-night")
	if !ok {
		t.Fatal("expected testdata/tokyo-night.json to load as a user theme")
	}
	if th.Light {
		t.Error("tokyo-night should be a dark theme")
	}
	if string(th.Bg) != "#1a1b26" || string(th.Blue) != "#7aa2f7" {
		t.Errorf("got Bg=%s Blue=%s", th.Bg, th.Blue)
	}
	found := false
	for _, n := range theme.Names() {
		if n == "tokyo-night" {
			found = true
		}
	}
	if !found {
		t.Error("tokyo-night missing from theme.Names()")
	}
}

func TestUserThemeFromSingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := writePalette(t, tmpDir, "single.json", validPalette)

	restore := theme.SetUserThemesDirForTest(func() string { return file })
	defer restore()

	th, ok := theme.Get("single")
	if !ok {
		t.Fatal("expected theme loaded from a single .json file path")
	}
	if string(th.Bg) != "#111111" {
		t.Errorf("Bg = %s, want #111111", th.Bg)
	}
}

func TestUserThemeDirectoryOverride(t *testing.T) {
	tmpDir := t.TempDir()
	writePalette(t, tmpDir, "dirone.json", validPalette)
	writePalette(t, tmpDir, "dirtwo.json", validPalette)
	// Non-json files and directories must be ignored.
	os.WriteFile(filepath.Join(tmpDir, "notes.txt"), []byte("{}"), 0o644)

	restore := theme.SetUserThemesDirForTest(func() string { return tmpDir })
	defer restore()

	for _, n := range []string{"dirone", "dirtwo"} {
		if _, ok := theme.Get(n); !ok {
			t.Errorf("expected %q to load from overridden directory", n)
		}
	}
	if _, ok := theme.Get("notes"); ok {
		t.Error("non-.json file must not be treated as a theme")
	}
}

func TestInvalidAndIncompletePalettesSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "broken.json"), []byte("{not json"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, "incomplete.json"), []byte(`{"red":"#fff"}`), 0o644)
	writePalette(t, tmpDir, "ok.json", validPalette)

	restore := theme.SetUserThemesDirForTest(func() string { return tmpDir })
	defer restore()

	if _, ok := theme.Get("broken"); ok {
		t.Error("malformed JSON must be skipped")
	}
	if _, ok := theme.Get("incomplete"); ok {
		t.Error("palette missing fg/bg must be skipped")
	}
	if _, ok := theme.Get("ok"); !ok {
		t.Error("valid palette must load alongside invalid ones")
	}
}

func TestBundledWinsNameCollision(t *testing.T) {
	tmpDir := t.TempDir()
	custom := dict(validPalette)
	custom["bg"] = "#abcdef"
	writePalette(t, tmpDir, "onedark.json", custom)

	restore := theme.SetUserThemesDirForTest(func() string { return tmpDir })
	defer restore()

	th, ok := theme.Get("onedark")
	if !ok {
		t.Fatal("onedark should still resolve")
	}
	if string(th.Bg) == "#abcdef" {
		t.Error("bundled theme must win over user theme with same name")
	}
}

func TestLightUserTheme(t *testing.T) {
	tmpDir := t.TempDir()
	pal := dict(validPalette)
	pal["light"] = true
	writePalette(t, tmpDir, "bright.json", pal)

	restore := theme.SetUserThemesDirForTest(func() string { return tmpDir })
	defer restore()

	th, ok := theme.Get("bright")
	if !ok || !th.Light {
		t.Errorf("expected light theme, got ok=%v light=%v", ok, th.Light)
	}
}

func TestMissingUserThemesPath(t *testing.T) {
	restore := theme.SetUserThemesDirForTest(func() string {
		return filepath.Join(t.TempDir(), "does-not-exist")
	})
	defer restore()

	if _, ok := theme.Get("anything-custom"); ok {
		t.Error("no custom themes should load from a missing directory")
	}
	if def := theme.DefaultTheme(); def.Name != theme.DefaultName {
		t.Error("bundled default theme must still work")
	}
}

func dict(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func TestUserThemesPathExpandsTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	themes := filepath.Join(home, "my-themes")
	if err := os.Mkdir(themes, 0o755); err != nil {
		t.Fatal(err)
	}
	writePalette(t, themes, "tilde.json", validPalette)

	restore := theme.SetUserThemesDirForTest(func() string { return "" })
	defer restore()
	theme.SetUserThemesPath("~/my-themes")

	if got := theme.UserThemesDir(); got != themes {
		t.Errorf("UserThemesDir() = %q, want %q", got, themes)
	}
	if _, ok := theme.Get("tilde"); !ok {
		t.Error("expected theme to load from a ~-prefixed path")
	}
}
