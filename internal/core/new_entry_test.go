package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

func TestParseNewEntryPaths(t *testing.T) {
	store := newTestStore(t)
	if err := os.MkdirAll(filepath.Join(store.Root(), "FLM"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		parent, input string
		path          string   // shown in the confirmation
		folders       []string // folder IDs on the path, outermost first
		note          string   // note ID, "" for a folder
	}{
		{"", "test/", "test/", []string{"test"}, ""},
		{"", "test", "test.md", nil, "test.md"},
		{"", "test.md", "test.md", nil, "test.md"},
		{"", "ideas/test", "ideas/test.md", []string{"ideas"}, "ideas/test.md"},
		{"", "a/b/", "a/b/", []string{"a", "a/b"}, ""},
		{"", "a/b/c", "a/b/c.md", []string{"a", "a/b"}, "a/b/c.md"},
		{"FLM", "test/", "FLM/test/", []string{"FLM/test"}, ""},
		{"FLM", " what? ", "FLM/what-.md", nil, "FLM/what-.md"},
	}
	for _, c := range cases {
		e, err := store.ParseNewEntry(c.parent, c.input)
		if err != nil {
			t.Fatalf("%q in %q: %v", c.input, c.parent, err)
		}
		var folders []string
		for _, f := range e.Folders {
			folders = append(folders, f.ID)
		}
		if e.Path != c.path || e.Note.ID != c.note || strings.Join(folders, ",") != strings.Join(c.folders, ",") {
			t.Errorf("%q in %q: got path %q folders %v note %q, want %q %v %q",
				c.input, c.parent, e.Path, folders, e.Note.ID, c.path, c.folders, c.note)
		}
		if e.Exists {
			t.Errorf("%q: nothing exists yet", c.input)
		}
	}
	if entries, _ := os.ReadDir(store.Root()); len(entries) != 1 {
		t.Fatalf("parsing must not write anything, got %v", entries)
	}
}

func TestParseNewEntryRejectsUnsafePaths(t *testing.T) {
	store := newTestStore(t)
	// Each refusal says why, so the status bar tells the user what to fix.
	for input, why := range map[string]string{
		"..": `".."`, "../x": `".."`, "a/../../x": `".."`, "a/..": `".."`, "./x": `"."`, "a/./b": `"."`,
		"/etc/x": "relative", `\x`: "relative", "C:/x": "relative", "/": "relative",
		"a//b": "empty", "a//": "empty", ".hidden": "hidden", "a/.git/": "hidden",
	} {
		_, err := store.ParseNewEntry("", input)
		if err == nil || !strings.Contains(err.Error(), why) {
			t.Errorf("%q must be refused with %s, got %v", input, why, err)
		}
	}
}

func TestCreateEntryMakesFoldersAndNote(t *testing.T) {
	store := newTestStore(t)
	e, err := store.ParseNewEntry("", "ideas/test")
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.CreateEntry(e)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "ideas/test.md" || p.Title != "test" || p.ParentID != "ideas" {
		t.Fatalf("got %+v", p)
	}
	if got, err := store.Get("ideas/test.md"); err != nil || got.Content != "" {
		t.Fatalf("note not created: %+v %v", got, err)
	}

	e, _ = store.ParseNewEntry("", "a/b/")
	f, err := store.CreateEntry(e)
	if err != nil || f.ID != "a/b" || !f.IsFolder {
		t.Fatalf("folder: got %+v %v", f, err)
	}
	if info, err := os.Stat(filepath.Join(store.Root(), "a", "b")); err != nil || !info.IsDir() {
		t.Fatalf("folder a/b not on disk: %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Join(store.Root(), "a", "b")); len(entries) != 0 {
		t.Fatalf("a folder entry creates no note, got %v", entries)
	}
}

func TestCreateEntryNeverOverwrites(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.SaveAs("ideas", "test", "precious"); err != nil {
		t.Fatal(err)
	}
	e, err := store.ParseNewEntry("", "ideas/test")
	if err != nil {
		t.Fatal(err)
	}
	if !e.Exists {
		t.Fatal("ideas/test.md exists")
	}
	if _, err := store.CreateEntry(e); err == nil {
		t.Fatal("creating over an existing note must fail")
	}
	if got, _ := store.Get("ideas/test.md"); got.Content != "precious" {
		t.Fatalf("existing note changed: %q", got.Content)
	}
	if e, _ := store.ParseNewEntry("", "ideas/"); !e.Exists {
		t.Fatal("folder ideas exists")
	}
}

func TestSplitEntryPathIsTheParserOfN(t *testing.T) {
	cases := []struct {
		input  string
		segs   string
		folder bool
	}{
		{"test", "test", false},
		{"test.md", "test", false},
		{"test/", "test", true},
		{" ideas / what? ", "ideas,what-", false},
		{"a/b/", "a,b", true},
		{"CON", "CON-note", false},
	}
	for _, c := range cases {
		segs, folder, err := core.SplitEntryPath(c.input)
		if err != nil || strings.Join(segs, ",") != c.segs || folder != c.folder {
			t.Errorf("%q: got %v %v %v, want %q %v", c.input, segs, folder, err, c.segs, c.folder)
		}
	}
	if _, _, err := core.SplitEntryPath("a/../x"); err == nil {
		t.Error(`"a/../x" must be refused`)
	}
}
