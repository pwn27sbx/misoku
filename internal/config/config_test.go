package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestSyncSectionIsReadAndSurvivesASave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := `{"theme": "onedark", "sync": {"vaults": {"~/notes": {"interval": "5m", "device": "mac"}}}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if v := cfg.Sync.Vaults["~/notes"]; v.Interval != "5m" || v.Device != "mac" {
		t.Fatalf("sync not read: %+v", cfg.Sync)
	}

	// The theme picker loads, changes the theme and saves: sync must survive
	// exactly as written (still "~/notes", still "5m").
	cfg.Theme = "nord"
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	again, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := again.Sync.Vaults["~/notes"]; !ok || v.Interval != "5m" || v.Device != "mac" {
		t.Fatalf("saving the theme lost the sync section: %+v", again.Sync)
	}
}

func TestNoSyncSectionIsNotWritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{Theme: "nord"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sync") {
		t.Fatalf("an unset sync section must not be written:\n%s", data)
	}
}

func TestSyncParseExpandsHomeAndUsesDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	abs := filepath.Join(t.TempDir(), "elsewhere")
	s := config.Sync{Vaults: map[string]config.SyncVault{
		"~/notes":                               {Interval: "90s", Device: "mac"},
		"~":                                     {},
		abs:                                     {Interval: "1h"},
		"~" + string(os.PathSeparator) + "deep": {},
	}}
	vaults, warnings := s.Parse()
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	got := map[string]config.Vault{}
	for _, v := range vaults {
		got[v.Path] = v
	}
	notes, ok := got[filepath.Join(home, "notes")]
	if !ok || notes.Interval != 90*time.Second || notes.Device != "mac" {
		t.Fatalf("~/notes not expanded or parsed: %+v", vaults)
	}
	if v, ok := got[home]; !ok || v.Interval != config.DefaultSyncInterval {
		t.Fatalf("~ alone must be the home folder with the default interval: %+v", vaults)
	}
	if v, ok := got[abs]; !ok || v.Interval != time.Hour {
		t.Fatalf("absolute path not kept: %+v", vaults)
	}
	if _, ok := got[filepath.Join(home, "deep")]; !ok {
		t.Fatalf("~ with the OS separator not expanded: %+v", vaults)
	}
	if v := got[home]; v.Device == "" || strings.ContainsAny(v.Device, `<>:"/\|?*`) {
		t.Fatalf("default device must be a file-name-safe host name, got %q", v.Device)
	}
	if host, err := os.Hostname(); err == nil && !strings.ContainsAny(host, `<>:"/\|?*`) {
		if want := strings.Trim(host, " ."); want != "" && got[home].Device != want {
			t.Fatalf("default device = %q, want the host name %q", got[home].Device, want)
		}
	}
}

func TestSyncParseWarnsInsteadOfFailing(t *testing.T) {
	good := filepath.Join(t.TempDir(), "good")
	s := config.Sync{Vaults: map[string]config.SyncVault{
		good:            {Interval: "soon"},
		"relative/path": {},
		"":              {},
	}}
	vaults, warnings := s.Parse()
	if len(vaults) != 1 || vaults[0].Path != good {
		t.Fatalf("only the absolute vault must stay: %+v", vaults)
	}
	if vaults[0].Interval != config.DefaultSyncInterval {
		t.Fatalf("a bad interval must fall back to the default, got %v", vaults[0].Interval)
	}
	if len(warnings) != 3 {
		t.Fatalf("want 3 warnings (bad interval, relative path, empty path), got %q", warnings)
	}
	for _, want := range []string{"soon", "relative/path"} {
		if !strings.Contains(strings.Join(warnings, "\n"), want) {
			t.Errorf("warnings must name %q: %q", want, warnings)
		}
	}

	for _, bad := range []string{"0s", "-5m"} {
		vs, ws := config.Sync{Vaults: map[string]config.SyncVault{good: {Interval: bad}}}.Parse()
		if len(ws) != 1 || vs[0].Interval != config.DefaultSyncInterval {
			t.Errorf("interval %q must warn and use the default: %+v %q", bad, vs, ws)
		}
	}
}

// One folder listed twice (written with ~ and as an absolute path, say)
// gives one vault with the same settings every time, and a warning, instead
// of whichever entry map order happened to put first.
func TestSyncParseKeepsOneEntryPerFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	abs := filepath.Join(home, "notes")
	s := config.Sync{Vaults: map[string]config.SyncVault{
		"~/notes":                      {Device: "tilde"},
		abs:                            {Device: "absolute"},
		abs + string(os.PathSeparator): {Device: "slash"},
	}}
	first := ""
	for i := 0; i < 20; i++ {
		vaults, warnings := s.Parse()
		if len(vaults) != 1 || vaults[0].Path != abs {
			t.Fatalf("want one vault at %s, got %+v", abs, vaults)
		}
		if first == "" {
			first = vaults[0].Device
		} else if vaults[0].Device != first {
			t.Fatalf("the settings changed between reads: %q, then %q", first, vaults[0].Device)
		}
		if len(warnings) != 2 {
			t.Fatalf("want a warning for each entry left out, got %q", warnings)
		}
	}
}

func TestSyncDeviceIsMadeFileNameSafe(t *testing.T) {
	dir := t.TempDir()
	vs, ws := config.Sync{Vaults: map[string]config.SyncVault{dir: {Device: ` my:mac/2? `}}}.Parse()
	if len(ws) != 0 || len(vs) != 1 {
		t.Fatalf("got %+v %q", vs, ws)
	}
	if vs[0].Device != "my-mac-2-" {
		t.Fatalf("device = %q, want %q", vs[0].Device, "my-mac-2-")
	}
}

func TestAddSyncVaultKeepsEveryOtherSetting(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	before := `{
  "theme": "nord",
  "keys": {"save": ["<C-s>"]},
  "not_known_yet": {"a": [1, "two", null], "b": true},
  "sync": {"other": 1, "vaults": {"~/old": {"device": "mac", "interval": "1m"}}}
}`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	added, err := config.AddSyncVault(path, "/notes/new")
	if err != nil || !added {
		t.Fatalf("AddSyncVault = %v, %v", added, err)
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
    ],
    "b": true
  },
  "sync": {
    "other": 1,
    "vaults": {
      "/notes/new": {},
      "~/old": {
        "device": "mac",
        "interval": "1m"
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
	if err != nil || c.Theme != "nord" || len(c.Sync.Vaults) != 2 || c.Sync.Vaults["~/old"].Device != "mac" {
		t.Errorf("Load after AddSyncVault = %+v, %v", c, err)
	}
}

func TestAddSyncVaultCreatesTheFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "misoku", "config.json")
	added, err := config.AddSyncVault(path, `C:\notes`)
	if err != nil || !added {
		t.Fatalf("AddSyncVault = %v, %v", added, err)
	}
	data, _ := os.ReadFile(path)
	if want := "{\n  \"sync\": {\n    \"vaults\": {\n      \"C:\\\\notes\": {}\n    }\n  }\n}\n"; string(data) != want {
		t.Errorf("config is:\n%s\nwant:\n%s", data, want)
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Errorf("only config.json may be left, got %v", entries)
	}
}

func TestAddSyncVaultLeavesAListedVaultAlone(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	before := `{"sync":{"vaults":{"/notes":{"device":"desk"}}}}`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	added, err := config.AddSyncVault(path, "/notes")
	if err != nil || added {
		t.Fatalf("AddSyncVault = %v, %v; want false, nil", added, err)
	}
	if data, _ := os.ReadFile(path); string(data) != before {
		t.Errorf("a listed vault must leave the file byte for byte, got:\n%s", data)
	}
}

func TestAddSyncVaultNeverRewritesAnUnreadableConfig(t *testing.T) {
	t.Parallel()
	for name, before := range map[string]string{
		"not JSON":        `{"theme": "nord",`,
		"sync not object": `{"sync": [1]}`,
		"vaults not obj":  `{"sync": {"vaults": "x"}}`,
		"top not object":  `[1, 2]`,
	} {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
			t.Fatal(err)
		}
		if added, err := config.AddSyncVault(path, "/notes"); err == nil || added {
			t.Errorf("%s: AddSyncVault = %v, %v; want an error", name, added, err)
		}
		if data, _ := os.ReadFile(path); string(data) != before {
			t.Errorf("%s: the file must stay as it was, got:\n%s", name, data)
		}
	}
}

// "off" turns timed sync off for a vault on purpose: no warning, and an
// interval of zero, which the app reads as "never on a timer".
func TestSyncParseIntervalOffTurnsTimedSyncOff(t *testing.T) {
	good := filepath.Join(t.TempDir(), "good")
	for _, off := range []string{"off", "OFF", " off "} {
		vs, ws := config.Sync{Vaults: map[string]config.SyncVault{good: {Interval: off}}}.Parse()
		if len(ws) != 0 || len(vs) != 1 || vs[0].Interval != 0 {
			t.Errorf("interval %q must turn timed sync off without a warning: %+v %q", off, vs, ws)
		}
	}
}

// An interval shorter than MinSyncInterval would fetch and push all the
// time: it warns, naming the minimum, and uses the default. The minimum
// itself is accepted.
func TestSyncParseTooShortIntervalWarnsAndUsesTheDefault(t *testing.T) {
	good := filepath.Join(t.TempDir(), "good")
	for _, short := range []string{"10s", "59s", "1ns"} {
		vs, ws := config.Sync{Vaults: map[string]config.SyncVault{good: {Interval: short}}}.Parse()
		if len(ws) != 1 || vs[0].Interval != config.DefaultSyncInterval {
			t.Errorf("interval %q must warn and use the default: %+v %q", short, vs, ws)
			continue
		}
		if !strings.Contains(ws[0], short) || !strings.Contains(ws[0], config.MinSyncInterval.String()) {
			t.Errorf("the warning must name %q and the minimum: %q", short, ws[0])
		}
	}
	vs, ws := config.Sync{Vaults: map[string]config.SyncVault{good: {Interval: config.MinSyncInterval.String()}}}.Parse()
	if len(ws) != 0 || vs[0].Interval != config.MinSyncInterval {
		t.Fatalf("the minimum itself must be accepted: %+v %q", vs, ws)
	}
}
