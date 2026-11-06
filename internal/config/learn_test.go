package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pwn27sbx/misoku/internal/config"
)

// unmarshalTop decodes one top-level key of the config file at path into v.
func unmarshalTop(t *testing.T, path, key string, v any) error {
	t.Helper()
	raw, ok := readTop(t, path)[key]
	if !ok {
		t.Fatalf("%s is not in the config:\n%s", key, readTop(t, path))
	}
	return json.Unmarshal(raw, v)
}

// sameValue reports whether two raw JSON values hold the same setting,
// however each is spaced or escaped.
func sameValue(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatalf("%s: %v", b, err)
	}
	return reflect.DeepEqual(x, y)
}

// S2: answering the first-run dialog is remembered as a top-level
// learn_offered, and every other setting is kept as it was.
func TestSetLearnOfferedKeepsEveryOtherSetting(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	before := `{
  "theme": "nord",
  "keys": {"save": ["<C-s>"]},
  "not_known_yet": {"a": [1, "two", null]},
  "sync": {"vaults": {"~/old": {"device": "mac"}}}
}`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	want := readTop(t, path)
	if err := config.SetLearnOffered(path); err != nil {
		t.Fatal(err)
	}
	var got bool
	if err := unmarshalTop(t, path, "learn_offered", &got); err != nil || !got {
		t.Fatalf("learn_offered = %v (%v), want true", got, err)
	}
	// Every other key keeps its value (the file is rewritten with two-space
	// indentation, so the bytes differ; the settings must not).
	after := readTop(t, path)
	delete(after, "learn_offered")
	if len(after) != len(want) {
		t.Fatalf("the config gained or lost keys: %v, want %v", after, want)
	}
	for key, raw := range want {
		if !sameValue(t, raw, after[key]) {
			t.Errorf("%s changed from %s to %s", key, raw, after[key])
		}
	}
	// Load reads it back, for the gate that decides whether to ask.
	cfg, err := config.Load(path)
	if err != nil || !cfg.LearnOffered {
		t.Fatalf("Load gave LearnOffered %v (%v), want true", cfg.LearnOffered, err)
	}
}

// Answering twice writes nothing the second time: the file stays as it is.
func TestSetLearnOfferedTwiceLeavesTheFileAlone(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.SetLearnOffered(path); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := config.SetLearnOffered(path); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatalf("the second answer rewrote the config:\n%s\n%s", first, second)
	}
}

// A file that is not a JSON object is never rewritten, and says why.
func TestSetLearnOfferedLeavesABrokenConfig(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	broken := `[1, 2]`
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.SetLearnOffered(path); err == nil {
		t.Fatal("SetLearnOffered rewrote a config that is not an object")
	}
	if got, _ := os.ReadFile(path); string(got) != broken {
		t.Fatalf("the broken config was rewritten:\n%s", got)
	}
}

// Saving another setting never drops the answer: it is a Config field.
func TestSaveKeepsLearnOffered(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.SetLearnOffered(path); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Theme = "nord"
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	var got bool
	if err := unmarshalTop(t, path, "learn_offered", &got); err != nil || !got {
		t.Fatalf("learn_offered after a save = %v (%v), want true", got, err)
	}
}
