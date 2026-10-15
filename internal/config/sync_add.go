package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// AddSyncVault lists vault in sync.vaults of the config file at path, with
// Misoku's defaults ({}), for `misoku sync init`. It reports false, and
// leaves the file byte for byte, when vault is already a key there.
//
// Every other setting stays, including ones this version of Misoku does
// not know (unlike Save, which writes only the fields of Config): the file
// is read as raw JSON and only sync.vaults gains a key. Like any save, the
// file is written back with two-space indentation and its keys sorted.
// A config that is a symlink (dotfiles managers such as GNU Stow) has its
// target written, so the link stays; the file keeps its permissions.
// A file that is not a JSON object, or whose "sync" or "sync.vaults" is
// not an object, is never rewritten: the error says why. The new file
// replaces the old one in one rename, so a failed write leaves the old one.
func AddSyncVault(path, vault string) (bool, error) {
	if path == "" {
		return false, errors.New("there is no config folder on this system")
	}
	path, err := followLinks(path)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	top, err := object(data, path)
	if err != nil {
		return false, err
	}
	sync, err := object(top["sync"], `"sync" in `+path)
	if err != nil {
		return false, err
	}
	vaults, err := object(sync["vaults"], `"sync.vaults" in `+path)
	if err != nil {
		return false, err
	}
	if _, ok := vaults[vault]; ok {
		return false, nil
	}
	vaults[vault] = json.RawMessage("{}")
	if sync["vaults"], err = marshal(vaults); err != nil {
		return false, err
	}
	if top["sync"], err = marshal(sync); err != nil {
		return false, err
	}
	out, err := marshal(top)
	if err != nil {
		return false, err
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, out, "", "  "); err != nil {
		return false, err
	}
	indented.WriteByte('\n')
	return true, replaceFile(path, indented.Bytes())
}

// object reads raw (absent or null: empty) as a JSON object whose values
// stay as written.
func object(raw []byte, what string) (map[string]json.RawMessage, error) {
	m := map[string]json.RawMessage{}
	if t := bytes.TrimSpace(raw); len(t) == 0 || string(t) == "null" {
		return m, nil
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("%s is not a JSON object, so it was left as it is: %w", what, err)
	}
	return m, nil
}

// marshal writes v as compact JSON without escaping <, > and &, so keys
// such as "<C-s>" stay readable.
func marshal(v any) (json.RawMessage, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// followLinks returns the file path names after every symlink, even one
// whose target does not exist yet. More than 40 links in a row is a loop.
func followLinks(path string) (string, error) {
	for range 40 {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return path, nil // a missing file is created where path says
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		path = target
	}
	return "", fmt.Errorf("%s: too many symlinks", path)
}

// replaceFile writes data to a temporary file next to path and renames it
// over path, creating the folder first. The file keeps its permissions
// (0644 when new).
func replaceFile(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp, mode)
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		_ = os.Remove(tmp) // best effort: the old config is untouched
	}
	return err
}
