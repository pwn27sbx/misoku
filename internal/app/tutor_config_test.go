package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfigValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{"ui": {"transparent": true, "link_suggestions": "finder", "width": 80}, "theme": "nord"}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"ui.transparent":      "true",
		"ui.link_suggestions": "finder",
		"ui.width":            "80",
		"theme":               "nord",
	}
	for key, want := range cases {
		if got, ok := readConfigValue(path, key); !ok || got != want {
			t.Errorf("readConfigValue(%q) = %q, %v, want %q", key, got, ok, want)
		}
	}
	if _, ok := readConfigValue(path, "ui.missing"); ok {
		t.Error("a missing key was found")
	}
	if vals := readConfigValues(path, []string{"theme", "ui.width"}); len(vals) != 2 {
		t.Errorf("readConfigValues = %v", vals)
	}
	if _, ok := readConfigValue(filepath.Join(dir, "nope.json"), "theme"); ok {
		t.Error("a missing file was read")
	}
}
