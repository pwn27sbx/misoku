package config

import (
	"encoding/json"
	"fmt"
)

// SetNotesDir remembers dir as the notes folder (notes_dir) in the config
// file at path, so `misoku` with no folder opens it. dir is written as
// given ("~/Notes" stays "~/Notes"); it must be absolute or start with
// "~". Every other setting stays, the ones this version does not know
// included; the file is written like AddSyncVault writes it (symlink
// followed, permissions kept, one atomic rename), and a file that is not
// a JSON object is never rewritten.
func SetNotesDir(path, dir string) error {
	if _, err := expandHome(dir); err != nil {
		return fmt.Errorf("notes folder %q: %w", dir, err)
	}
	_, err := editFile(path, func(top map[string]json.RawMessage) (bool, error) {
		value, err := marshal(dir)
		if err != nil {
			return false, err
		}
		if string(top["notes_dir"]) == string(value) {
			return false, nil // already remembered: the file stays byte for byte
		}
		top["notes_dir"] = value
		return true, nil
	})
	return err
}

// NotesFolder returns notes_dir as an absolute path ("~" expanded), or ""
// when it is not set. A relative notes_dir is an error: it would depend
// on the folder Misoku happens to be started from.
func (c Config) NotesFolder() (string, error) {
	if c.NotesDir == "" {
		return "", nil
	}
	dir, err := expandHome(c.NotesDir)
	if err != nil {
		return "", fmt.Errorf("notes_dir %q: %w", c.NotesDir, err)
	}
	return dir, nil
}
