package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// realHome is the HOME the tests started with, before any t.Setenv.
var realHome, _ = os.UserHomeDir()

func newTestStore(t *testing.T) *core.Store {
	t.Helper()
	// Deleting moves notes to the Trash: keep the tests away from the real
	// one. A test that already set its own HOME keeps it.
	if h, _ := os.UserHomeDir(); h == realHome {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	}
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error creating store: %v", err)
	}
	return store
}

func TestStoreCreate(t *testing.T) {
	tests := []struct {
		name          string
		inputTitle    string
		expectedTitle string
	}{
		{name: "Create with empty title generates default title", inputTitle: "", expectedTitle: "Untitled"},
		{name: "Create with custom title", inputTitle: "Meeting Notes", expectedTitle: "Meeting Notes"},
		{name: "Create with special characters", inputTitle: "🔑 Credentials & Keys", expectedTitle: "🔑 Credentials & Keys"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			p, err := store.Create(tt.inputTitle, "")
			if err != nil {
				t.Fatalf("unexpected error creating page: %v", err)
			}
			if p.Title != tt.expectedTitle {
				t.Errorf("expected title %q, got %q", tt.expectedTitle, p.Title)
			}
			if p.ID == "" {
				t.Error("expected non-empty page ID")
			}
			if _, err := os.Stat(filepath.Join(store.Root(), filepath.FromSlash(p.ID))); err != nil {
				t.Errorf("expected a real file on disk at %q: %v", p.ID, err)
			}
		})
	}
}

func TestStoreCreateDuplicateTitlesAreDisambiguated(t *testing.T) {
	store := newTestStore(t)

	first, err := store.Create("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := store.Create("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.Title != "Untitled" {
		t.Errorf("expected first page title 'Untitled', got %q", first.Title)
	}
	if second.Title != "Untitled 2" {
		t.Errorf("expected second page title 'Untitled 2', got %q", second.Title)
	}
}

func TestStoreCreateChild(t *testing.T) {
	store := newTestStore(t)

	parent, err := store.Create("Parent 1", "")
	if err != nil {
		t.Fatalf("unexpected error creating parent page: %v", err)
	}

	child, err := store.Create("Nested Tasks", parent.ID)
	if err != nil {
		t.Fatalf("unexpected error creating child page: %v", err)
	}
	if child.ParentID != parent.ID {
		t.Errorf("expected parent ID %q, got %q", parent.ID, child.ParentID)
	}
	if child.Title != "Nested Tasks" {
		t.Errorf("expected title 'Nested Tasks', got %q", child.Title)
	}

	// The sidecar folder should exist alongside the parent file.
	sidecar := filepath.Join(store.Root(), "Parent 1")
	if info, err := os.Stat(sidecar); err != nil || !info.IsDir() {
		t.Errorf("expected sidecar folder %q to exist", sidecar)
	}

	_, err = store.Create("Child", "non-existent.md")
	if err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound for non-existent parent, got %v", err)
	}
}

func TestRenameFolderViaUpdate(t *testing.T) {
	store := newTestStore(t)
	if err := os.MkdirAll(filepath.Join(store.Root(), "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update(core.Page{ID: "assets", Title: "images", IsFolder: true})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != "images" || !updated.IsFolder {
		t.Errorf("got %+v", updated)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "images")); err != nil {
		t.Error("folder not renamed on disk")
	}
	// Same-name rename is a no-op, empty title keeps current name.
	again, err := store.Update(core.Page{ID: "images", Title: "", IsFolder: true})
	if err != nil || again.Title != "images" {
		t.Errorf("got %+v %v", again, err)
	}
}

func TestRenameFolderConflict(t *testing.T) {
	store := newTestStore(t)
	os.MkdirAll(filepath.Join(store.Root(), "a"), 0755)
	os.MkdirAll(filepath.Join(store.Root(), "b"), 0755)
	if _, err := store.Update(core.Page{ID: "a", Title: "b", IsFolder: true}); err == nil {
		t.Error("expected conflict error")
	}
}

func TestChildDir(t *testing.T) {
	store := newTestStore(t)
	if got := store.ChildDir(""); got != "" {
		t.Errorf("ChildDir(\"\") = %q", got)
	}
	os.MkdirAll(filepath.Join(store.Root(), "notes"), 0755)
	if got := store.ChildDir("notes"); got != "notes" {
		t.Errorf("ChildDir(notes) = %q", got)
	}
}

func TestStorePlainFoldersAreNavigable(t *testing.T) {
	store := newTestStore(t)

	if err := os.MkdirAll(filepath.Join(store.Root(), "assets"), 0755); err != nil {
		t.Fatalf("failed to seed plain folder: %v", err)
	}
	page, err := store.Create("notes", "assets")
	if err != nil {
		t.Fatalf("unexpected error creating page inside plain folder: %v", err)
	}
	if page.ParentID != "assets" {
		t.Errorf("expected parent ID 'assets', got %q", page.ParentID)
	}

	pages := store.List()
	var folder core.Page
	found := false
	for _, p := range pages {
		if p.ID == "assets" {
			folder = p
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'assets' folder in List(), got %v", pages)
	}
	if !folder.IsFolder {
		t.Errorf("expected 'assets' to be marked IsFolder, got %v", folder)
	}
}

func TestStoreUpdateRenamesFileAndSidecarFolder(t *testing.T) {
	store := newTestStore(t)

	parent, err := store.Create("Parent 1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := store.Create("child 1", parent.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parent.Title = "Renamed Parent"
	parent.Content = "# Renamed Parent"
	updated, err := store.Update(parent)
	if err != nil {
		t.Fatalf("unexpected error updating parent: %v", err)
	}
	if updated.ID != "Renamed Parent.md" {
		t.Errorf("expected renamed ID 'Renamed Parent.md', got %q", updated.ID)
	}

	// The sidecar folder (and the child inside it) must have moved too.
	movedChild := filepath.Join(store.Root(), "Renamed Parent", "child 1.md")
	if _, err := os.Stat(movedChild); err != nil {
		t.Errorf("expected child page to move with renamed sidecar folder: %v", err)
	}

	got, err := store.Get(updated.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching renamed page: %v", err)
	}
	if got.Content != "# Renamed Parent" {
		t.Errorf("expected content to persist through rename, got %q", got.Content)
	}

	// Sanity: the child's own ID now reflects the new parent path.
	pages := store.List()
	foundChild := false
	for _, p := range pages {
		if p.Title == "child 1" {
			foundChild = true
			if p.ParentID != updated.ID {
				t.Errorf("expected child ParentID %q, got %q", updated.ID, p.ParentID)
			}
		}
	}
	if !foundChild {
		t.Errorf("expected to find child 1 after rename, got %v", pages)
	}
}

func TestStoreDeleteCascadesToSidecarFolder(t *testing.T) {
	store := newTestStore(t)

	parent, err := store.Create("Parent 1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := store.Create("child 1", parent.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := store.Delete(parent.ID); err != nil {
		t.Fatalf("unexpected error deleting parent: %v", err)
	}

	if pages := store.List(); len(pages) != 0 {
		t.Errorf("expected deleting a parent to cascade to its children, got %v", pages)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), "Parent 1")); !os.IsNotExist(err) {
		t.Errorf("expected sidecar folder to be removed, stat err = %v", err)
	}
}

func TestStoreEdgeCases(t *testing.T) {
	store := newTestStore(t)

	if _, err := store.Get("non-existent.md"); err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound, got %v", err)
	}
	if _, err := store.Update(core.Page{ID: "invalid-id.md"}); err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound on update, got %v", err)
	}
	if err := store.Delete("invalid-id.md"); err != core.ErrPageNotFound {
		t.Errorf("expected ErrPageNotFound on delete, got %v", err)
	}
}

func TestStoreListReflectsRealDirectoryOrdering(t *testing.T) {
	store := newTestStore(t)

	if _, err := store.Create("Zebra", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := store.Create("Apple", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := store.SaveAs("Middle Folder", "inside", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pages := store.List()
	if len(pages) != 4 {
		t.Fatalf("expected 3 entries, got %d: %v", len(pages), pages)
	}
	// Folders sort before files; both groups are alphabetical.
	if pages[0].Title != "Middle Folder" || !pages[0].IsFolder {
		t.Errorf("expected folder first, got %v", pages[0])
	}
	if pages[2].Title != "Apple" || pages[3].Title != "Zebra" {
		t.Errorf("expected files alphabetically after folders, got %v, %v", pages[2], pages[3])
	}
}

func TestNewStoreCreatesMissingWorkspaceDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist-yet")
	store, err := core.NewStore(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info, err := os.Stat(store.Root()); err != nil || !info.IsDir() {
		t.Errorf("expected workspace directory to be created, got err=%v", err)
	}
}

func TestNormalizeNotePath(t *testing.T) {
	cases := []struct {
		dir, name, want string
		wantErr         bool
	}{
		{"", "ideas", "ideas.md", false},
		{"notes/work", "plan.md", "notes/work/plan.md", false},
		{"/notes/", "Plan.MD", "notes/Plan.md", false},
		{"", "v1.2 notes", "v1.2 notes.md", false},
		{"", "draft.txt", "draft.txt.md", false},
		{"", "  ", "", true},
		{"", ".md", "", true},
		{"", "a/b", "", true},
		{"..", "x", "", true},
		{".git", "x", "", true},
	}
	for _, c := range cases {
		got, err := core.NormalizeNotePath(c.dir, c.name)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("NormalizeNotePath(%q, %q) = %q, %v; want %q, err=%v", c.dir, c.name, got, err, c.want, c.wantErr)
		}
	}
}

func TestSaveAsCreatesFoldersAndRefusesOverwrite(t *testing.T) {
	s := newTestStore(t)
	p, err := s.SaveAs("new/folder", "note", "# hi")
	if err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	if p.ID != "new/folder/note.md" || p.Title != "note" {
		t.Fatalf("unexpected page %+v", p)
	}
	got, err := s.Get(p.ID)
	if err != nil || got.Content != "# hi" {
		t.Fatalf("Get after SaveAs = %+v, %v", got, err)
	}
	if _, err := s.SaveAs("new/folder", "note.md", "x"); err == nil {
		t.Fatal("expected overwrite to be refused")
	}
	dirs := s.Dirs()
	want := []string{"", "new", "new/folder"}
	if strings.Join(dirs, ",") != strings.Join(want, ",") {
		t.Fatalf("Dirs = %v, want %v", dirs, want)
	}
}

func TestRenameKeepsContent(t *testing.T) {
	s := newTestStore(t)
	p, err := s.SaveAs("", "old", "keep me")
	if err != nil {
		t.Fatal(err)
	}
	renamed, err := s.Rename(p.ID, "new.md")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	got, err := s.Get(renamed.ID)
	if err != nil || got.Content != "keep me" || renamed.ID != "new.md" {
		t.Fatalf("after rename got %+v (id %q), err %v", got, renamed.ID, err)
	}
}

func TestListHidesFoldersWithoutMarkdown(t *testing.T) {
	s := newTestStore(t)
	root := s.Root()
	for _, d := range []string{"bin", "cmd/misoku", "docs/guides/deep", "empty"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "cmd/misoku/main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveAs("docs/guides/deep", "howto", "x"); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, p := range s.List() {
		ids = append(ids, p.ID)
	}
	got := strings.Join(ids, ",")
	if got != "docs,docs/guides,docs/guides/deep,docs/guides/deep/howto.md" {
		t.Fatalf("expected only folders leading to notes, got %s", got)
	}
}

func TestAttachFileCopiesOutsideFilesAndLinksInsideOnes(t *testing.T) {
	s := newTestStore(t)
	outside := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(outside, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	link, err := s.AttachFile("notes", outside)
	if err != nil || link != "assets/photo.png" {
		t.Fatalf("first attach = %q, %v", link, err)
	}
	if data, _ := os.ReadFile(filepath.Join(s.Root(), "notes/assets/photo.png")); string(data) != "png" {
		t.Fatal("expected the file copied into notes/assets")
	}
	link, _ = s.AttachFile("notes", outside)
	if link != "assets/photo-2.png" {
		t.Fatalf("expected a fresh name for a second copy, got %q", link)
	}

	inside := filepath.Join(s.Root(), "media", "clip.mp4")
	_ = os.MkdirAll(filepath.Dir(inside), 0o755)
	_ = os.WriteFile(inside, []byte("mp4"), 0o644)
	link, err = s.AttachFile("notes", inside)
	if err != nil || link != "../media/clip.mp4" {
		t.Fatalf("workspace file should be linked in place, got %q, %v", link, err)
	}
}

func TestWindowsFriendlyNamesAndLineEndings(t *testing.T) {
	for _, bad := range []string{"a:b", "what?", "CON", "nul.md", "x|y"} {
		if _, err := core.NormalizeNotePath("", bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
	s := newTestStore(t)
	p, err := s.Create("Q: what?", "")
	if err != nil || strings.ContainsAny(p.ID, ":?") {
		t.Fatalf("Create should sanitise the title, got %q %v", p.ID, err)
	}

	abs := filepath.Join(s.Root(), "win.md")
	if err := os.WriteFile(abs, []byte("one\r\ntwo\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get("win.md")
	if got.Content != "one\ntwo\n" {
		t.Fatalf("CRLF should be normalised for editing, got %q", got.Content)
	}
	got.Content = "one\ntwo\nthree\n"
	if _, err := s.Update(got); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(abs); string(data) != "one\r\ntwo\r\nthree\r\n" {
		t.Fatalf("CRLF file should stay CRLF, got %q", data)
	}
}
