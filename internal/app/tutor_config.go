package app

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// readConfigValues returns the values of the given dotted keys in the config
// file, as strings, for the tutorial's config checks. A missing file or key
// leaves the key out.
func readConfigValues(path string, keys []string) map[string]string {
	vals := map[string]string{}
	if path == "" || len(keys) == 0 {
		return vals
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return vals
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return vals
	}
	for _, key := range keys {
		if v, ok := lookupConfig(root, key); ok {
			vals[key] = v
		}
	}
	return vals
}

// readConfigValue is readConfigValues for one key.
func readConfigValue(path, key string) (string, bool) {
	v, ok := readConfigValues(path, []string{key})[key]
	return v, ok
}

// lookupConfig walks a parsed config by its dotted key.
func lookupConfig(root any, key string) (string, bool) {
	cur := root
	for _, part := range strings.Split(key, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		cur, ok = m[part]
		if !ok {
			return "", false
		}
	}
	switch v := cur.(type) {
	case string:
		return v, true
	case bool:
		if v {
			return "true", true
		}
		return "false", true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case nil:
		return "", true
	}
	data, err := json.Marshal(cur)
	if err != nil {
		return "", false
	}
	return string(data), true
}
