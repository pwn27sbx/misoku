// Package config persists user preferences (currently the colour theme) in
// $XDG_CONFIG_HOME/misoku/config.json (or the OS equivalent).
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
func Save(path string, c Config) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
