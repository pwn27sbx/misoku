package core_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// moveTestTree creates files (path → content, slash-separated, relative to the
// store root); a path ending in "/" is an empty folder.
func moveTestTree(t *testing.T, store *core.Store, files map[string]string) {
	t.Helper()
	for p, content := range files {
		abs := filepath.Join(store.Root(), filepath.FromSlash(p))
		if strings.HasSuffix(p, "/") {
			if err := os.MkdirAll(abs, 0755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// moveTestSnapshot lists every file and folder under the store root, hidden ones
// included, with each file's bytes: two equal snapshots mean the disk did not
// change.
func moveTestSnapshot(t *testing.T, store *core.Store) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(store.Root(), func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(store.Root(), abs)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			lines = append(lines, rel+"/")
			return nil
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return err
		}
		lines = append(lines, rel+"="+string(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func moveTestRead(t *testing.T, store *core.Store, id string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(store.Root(), filepath.FromSlash(id)))
	if err != nil {
		t.Fatalf("reading %s: %v", id, err)
	}
	return string(data)
}

func moveTestGone(t *testing.T, store *core.Store, id string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(store.Root(), filepath.FromSlash(id))); !os.IsNotExist(err) {
		t.Errorf("%s should be gone after the move (stat err %v)", id, err)
	}
}

func TestMoveNoteIntoFolder(t *testing.T) {
	store := newTestStore(t)
	moveTestTree(t, store, map[string]string{"a.md": "alpha", "dest/": ""})

	got, err := store.Move("a.md", "dest")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got.ID != "dest/a.md" || got.Title != "a" || got.ParentID != "dest" || got.IsFolder {
		t.Errorf("moved page = %+v, want ID dest/a.md, title a, parent dest", got)
	}
	if c := moveTestRead(t, store, "dest/a.md"); c != "alpha" {
		t.Errorf("content = %q, want alpha", c)
	}
	moveTestGone(t, store, "a.md")
}

func TestMoveNoteTakesItsSidecarFolder(t *testing.T) {
	store := newTestStore(t)
	moveTestTree(t, store, map[string]string{
		"nota.md":        "parent",
		"nota/child.md":  "child",
		"nota/deep/x.md": "deep",
		"dest/":          "",
	})

	got, err := store.Move("nota.md", "dest")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got.ID != "dest/nota.md" {
		t.Errorf("ID = %q, want dest/nota.md", got.ID)
	}
	if c := moveTestRead(t, store, "dest/nota/child.md"); c != "child" {
		t.Errorf("sub-note content = %q", c)
	}
	if c := moveTestRead(t, store, "dest/nota/deep/x.md"); c != "deep" {
		t.Errorf("nested sub-note content = %q", c)
	}
	moveTestGone(t, store, "nota.md")
	moveTestGone(t, store, "nota")
}

func TestMoveFolderWithEverythingInside(t *testing.T) {
	store := newTestStore(t)
	moveTestTree(t, store, map[string]string{"f/x.md": "x", "f/sub/y.md": "y", "dest/": ""})

	got, err := store.Move("f", "dest")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got.ID != "dest/f" || !got.IsFolder || got.ParentID != "dest" || got.Title != "f" {
		t.Errorf("moved folder = %+v, want ID dest/f folder under dest", got)
	}
	if c := moveTestRead(t, store, "dest/f/sub/y.md"); c != "y" {
		t.Errorf("content = %q", c)
	}
	moveTestGone(t, store, "f")
}

func TestMoveIntoWorkspaceRoot(t *testing.T) {
	store := newTestStore(t)
	moveTestTree(t, store, map[string]string{"dest/a.md": "alpha", "dest/a/b.md": "beta"})

	got, err := store.Move("dest/a.md", "")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got.ID != "a.md" || got.ParentID != "" {
		t.Errorf("moved page = %+v, want ID a.md at the root", got)
	}
	if c := moveTestRead(t, store, "a/b.md"); c != "beta" {
		t.Errorf("sidecar content = %q", c)
	}
	moveTestGone(t, store, "dest/a.md")
	moveTestGone(t, store, "dest/a")
}

func TestMoveUnderANoteCreatesItsSidecarFolder(t *testing.T) {
	store := newTestStore(t)
	moveTestTree(t, store, map[string]string{"a.md": "alpha", "p.md": "parent"})

	got, err := store.Move("a.md", "p.md")
	if err != nil {
		t.Fatalf("Move: %v", err)
	}
	if got.ID != "p/a.md" || got.ParentID != "p.md" {
		t.Errorf("moved page = %+v, want ID p/a.md under p.md", got)
	}
	if c := moveTestRead(t, store, "p/a.md"); c != "alpha" {
		t.Errorf("content = %q", c)
	}
}

func TestMoveRefusalsLeaveTheDiskUntouched(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		id      string
		dest    string
		wantErr string
	}{
		{"note name taken", map[string]string{"a.md": "1", "dest/a.md": "2"}, "a.md", "dest", "already exists"},
		{"note would adopt a folder of its name", map[string]string{"a.md": "1", "dest/a/x.md": "2"}, "a.md", "dest", "already exists"},
		{"sidecar name taken", map[string]string{"a.md": "1", "a/s.md": "s", "dest/a/x.md": "2"}, "a.md", "dest", "already exists"},
		{"folder name taken", map[string]string{"f/x.md": "1", "dest/f/y.md": "2"}, "f", "dest", "already exists"},
		{"folder would become a note's sidecar", map[string]string{"f/x.md": "1", "dest/f.md": "2"}, "f", "dest", "already exists"},
		{"folder into itself", map[string]string{"f/x.md": "1"}, "f", "f", "into itself"},
		{"folder below itself", map[string]string{"f/sub/x.md": "1"}, "f", "f/sub", "into itself"},
		{"note into its own sidecar", map[string]string{"a.md": "1", "a/b.md": "2"}, "a.md", "a", "into itself"},
		{"note under itself", map[string]string{"a.md": "1"}, "a.md", "a.md", "into itself"},
		{"note below its own sidecar", map[string]string{"a.md": "1", "a/b.md": "2"}, "a.md", "a/b.md", "into itself"},
		{"workspace root", map[string]string{"dest/x.md": "1"}, "", "dest", "workspace"},
		{"missing page", map[string]string{"dest/x.md": "1"}, "nope.md", "dest", "not found"},
		{"missing destination", map[string]string{"a.md": "1"}, "a.md", "nowhere", "not found"},
		{"hidden destination", map[string]string{"a.md": "1", ".hidden/x.md": "2"}, "a.md", ".hidden", "hidden"},
		{"hidden folder on the destination path", map[string]string{"a.md": "1", "f/.h/x.md": "2"}, "a.md", "f/.h", "hidden"},
		{"hidden source", map[string]string{".h/x.md": "1", "dest/": ""}, ".h/x.md", "dest", "hidden"},
		{"destination escapes the workspace", map[string]string{"a.md": "1"}, "a.md", "../out", "not allowed"},
		{"source escapes the workspace", map[string]string{"dest/": ""}, "../a.md", "dest", "not allowed"},
		{"absolute destination", map[string]string{"a.md": "1"}, "a.md", "/tmp", "relative"},
		{"already in that folder", map[string]string{"dest/a.md": "1"}, "dest/a.md", "dest", "already in"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestStore(t)
			moveTestTree(t, store, tc.files)
			before := moveTestSnapshot(t, store)

			_, err := store.Move(tc.id, tc.dest)
			if err == nil {
				t.Fatalf("Move(%q, %q) succeeded, want an error containing %q", tc.id, tc.dest, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tc.wantErr)
			}
			if after := moveTestSnapshot(t, store); after != before {
				t.Errorf("the disk changed on a refused move:\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}
