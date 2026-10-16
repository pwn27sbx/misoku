// Package config persists user preferences (currently the colour theme) in
// $XDG_CONFIG_HOME/misoku/config.json (or the OS equivalent).
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

// Config holds user preferences.
type Config struct {
	Theme string `json:"theme,omitempty"`
	// ThemesDir optionally points at a custom location for user themes:
	// either a directory containing *.json palettes or a single .json
	// palette file. Empty means the default OS config themes directory.
	ThemesDir string `json:"themes_dir,omitempty"`
	// Keys replaces the shortcuts of the listed actions, for example
	// {"toggle_explorer": ["space e"]}. An empty list unbinds the action.
	// See internal/keymap for the actions and their defaults.
	Keys map[string][]string `json:"keys,omitempty"`
	// UI changes how notes look in the preview. Anything left out keeps
	// Misoku's default.
	UI UI `json:"ui,omitzero"`
	// Sync lists the vaults Misoku keeps in sync with git. A folder that is
	// not listed never syncs.
	Sync Sync `json:"sync,omitzero"`
	// NotesDir is the folder `misoku` opens when no folder is given: an
	// absolute path or one starting with "~". See NotesFolder.
	NotesDir string `json:"notes_dir,omitempty"`
}

// Sync is the "sync" section of config.json:
//
//	{"sync": {"vaults": {"~/notes": {"interval": "5m", "device": "mac"}}}}
//
// It is kept as written (paths with "~", intervals as text) so that saving
// another setting writes it back unchanged; Parse turns it into Vaults.
type Sync struct {
	Vaults map[string]SyncVault `json:"vaults,omitempty"`
}

// SyncVault is one entry of sync.vaults, as written in config.json.
type SyncVault struct {
	// Interval between timed syncs, such as "5m" or "90s", at least
	// MinSyncInterval. Empty means DefaultSyncInterval; IntervalOff turns
	// timed sync off (the vault still syncs on opening, on request and on
	// quitting).
	Interval string `json:"interval,omitempty"`
	// Device names this computer in commits and conflict copies. Empty
	// means the host name.
	Device string `json:"device,omitempty"`
}

// DefaultSyncInterval is used when a vault sets no interval, or a bad one.
const DefaultSyncInterval = 5 * time.Minute

// MinSyncInterval is the shortest interval a vault may set.
const MinSyncInterval = time.Minute

// Vault is a sync.vaults entry ready to use: an absolute path ("~"
// expanded), an interval and a device name that is safe in a file name.
type Vault struct {
	Path string
	// Interval is the time between timed syncs, at least MinSyncInterval;
	// 0 means timed sync is off.
	Interval time.Duration
	Device   string
}

// Parse returns the usable vaults, sorted by path, and a warning for each
// value it could not use. A bad value never stops Misoku: a bad interval
// (unreadable, not positive or shorter than MinSyncInterval) falls back to
// DefaultSyncInterval, and a vault whose path cannot be
// resolved is left out (it does not sync, which is the safe side). A folder
// listed twice (as "~/notes" and as its absolute path, say) keeps the entry
// that sorts first, so the same settings win every time, and the others
// are warned about.
func (s Sync) Parse() ([]Vault, []string) {
	var vaults []Vault
	var warnings []string
	raws := make([]string, 0, len(s.Vaults))
	for raw := range s.Vaults {
		raws = append(raws, raw)
	}
	// Map order is random: sorted, the same entry wins every time.
	sort.Strings(raws)
	seen := map[string]string{} // path -> the entry that set it
	for _, raw := range raws {
		sv := s.Vaults[raw]
		path, err := expandHome(raw)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("sync: vault %q skipped: %v", raw, err))
			continue
		}
		if first, ok := seen[path]; ok {
			warnings = append(warnings, fmt.Sprintf("sync: vault %q skipped: it is the same folder as %q, whose settings are used", raw, first))
			continue
		}
		seen[path] = raw
		v := Vault{Path: path, Interval: DefaultSyncInterval, Device: safeDevice(sv.Device)}
		if sv.Interval != "" {
			d, warning := parseInterval(sv.Interval)
			v.Interval = d
			if warning != "" {
				warnings = append(warnings, fmt.Sprintf("sync: vault %q: interval %q %s; using %s", raw, sv.Interval, warning, DefaultSyncInterval))
			}
		}
		if v.Device == "" {
			host, _ := os.Hostname()
			v.Device = safeDevice(host)
		}
		if v.Device == "" {
			v.Device = "device"
		}
		vaults = append(vaults, v)
	}
	sort.Slice(vaults, func(i, j int) bool { return vaults[i].Path < vaults[j].Path })
	sort.Strings(warnings) // map order is random; keep messages stable
	return vaults, warnings
}

// IntervalOff is the interval that turns timed sync off for a vault.
const IntervalOff = "off"

// parseInterval reads a vault's interval: IntervalOff gives 0 (no timed
// sync); a value it cannot use gives DefaultSyncInterval and says why.
func parseInterval(text string) (time.Duration, string) {
	if strings.EqualFold(strings.TrimSpace(text), IntervalOff) {
		return 0, ""
	}
	d, err := time.ParseDuration(text)
	switch {
	case err != nil:
		return DefaultSyncInterval, `is not a duration like "5m" (or "off")`
	case d <= 0:
		return DefaultSyncInterval, "must be positive"
	case d < MinSyncInterval:
		// Fetching and pushing every few seconds would only load the
		// remote: saves are committed right away anyway.
		return DefaultSyncInterval, "is shorter than the minimum, " + MinSyncInterval.String()
	}
	return d, ""
}

// expandHome turns "~", "~/x" (and "~\x" on Windows) into a path under the
// user's home folder. Relative paths are rejected: they would depend on the
// folder Misoku happens to be started from.
func expandHome(p string) (string, error) {
	if p == "" {
		return "", errors.New("empty path")
	}
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~"+string(os.PathSeparator)) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(home, p[1:])
	}
	if !filepath.IsAbs(p) {
		return "", errors.New("the path must be absolute or start with ~")
	}
	return filepath.Clean(p), nil
}

// safeDevice makes name usable inside a file name on every OS: characters
// Windows forbids become "-", and leading or trailing spaces and dots
// (which Windows drops) are trimmed.
func safeDevice(name string) string {
	name = strings.Trim(name, " .")
	return strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '-'
		}
		return r
	}, name)
}

// UI is the "ui" section of config.json.
type UI struct {
	// HeadingIcons go before headings of level 1, 2, … (up to 6), e.g.
	// ["① ", "② "]. Missing levels keep their default icon.
	HeadingIcons []string `json:"heading_icons,omitempty"`
	// HeadingBackground paints headings on a full-width band of their color,
	// like render-markdown's "full" style in Neovim.
	HeadingBackground bool `json:"heading_background,omitempty"`
	// Bullets are used by list depth, cycling.
	Bullets  []string `json:"bullets,omitempty"`
	Checkbox struct {
		Unchecked string `json:"unchecked,omitempty"`
		Checked   string `json:"checked,omitempty"`
		// Todo marks "- [-]" as in progress; empty leaves "[-]" as text.
		Todo string `json:"todo,omitempty"`
	} `json:"checkbox,omitzero"`
	// ModeIcon replaces the per-mode icon of the statusline with one icon
	// for every mode, like lualine ("󱗞").
	ModeIcon string `json:"mode_icon,omitempty"`
	// Border is the frame of dialogs, menus and preview blocks: "rounded"
	// (default) or "single".
	Border string `json:"border,omitempty"`
	// ModifiedMark goes before the tab name of a note with unsaved changes,
	// like incline's "[+]".
	ModifiedMark string `json:"modified_mark,omitempty"`
}

// DefaultPath returns the config file location, or "" if the OS has no
// user config directory.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "misoku", "config.json")
}

// Load reads the config at path. A missing file is not an error.
func Load(path string) (Config, error) {
	var c Config
	if path == "" {
		return c, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(data, &c)
}

// Save writes the config to path, creating its directory.
//
// Settings of Config are written from c (an empty one is left out); every
// other key already in the file is kept as it was, so a version of Misoku
// that does not know a setting (notes_dir, say) never drops it. Like
// AddSyncVault, it writes through a symlinked config, keeps the file's
// permissions, replaces the file in one rename, and never rewrites a file
// that is not a JSON object.
func Save(path string, c Config) error {
	if path == "" {
		return nil
	}
	raw, err := marshal(c)
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	_, err = editFile(path, func(top map[string]json.RawMessage) (bool, error) {
		for _, key := range knownKeys() {
			delete(top, key)
		}
		maps.Copy(top, fields)
		return true, nil
	})
	return err
}

// knownKeys lists the JSON keys of Config's fields: the settings Save
// writes from a Config rather than keeping from the file.
func knownKeys() []string {
	t := reflect.TypeFor[Config]()
	keys := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ","); name != "" && name != "-" {
			keys = append(keys, name)
		}
	}
	return keys
}
