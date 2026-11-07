package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/core"
)

// testModel returns a Model with its components ready and recovery off.
func testModel(t *testing.T, store *core.Store) Model {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	return New(store)
}

// recoveryFiles lists the snapshot files left in dir.
func recoveryFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), recoveryPrefix) && strings.HasSuffix(e.Name(), recoverySuffix) {
			out = append(out, e.Name())
		}
	}
	return out
}

// Only tabs that differ from disk (or hold draft text) are kept, in the
// private recovery folder, and the snapshot remembers its vault.
func TestSaveRecoveryKeepsOnlyDirtyTabs(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := testModel(t, store)
	m.recoveryDir = t.TempDir()
	m.buffers = []*buffer{
		{id: "a.md", title: "a", saved: "old", text: "new"},
		{id: "b.md", title: "b", saved: "same", text: "same"},
		{id: "draft:1", title: "Untitled-1", text: "draft"},
	}

	path, err := m.SaveRecovery()
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("no recovery file written")
	}
	if runtime.GOOS != "windows" { // Windows has no POSIX mode bits
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("recovery file mode = %o, want 600", perm)
		}
		dirInfo, err := os.Stat(m.recoveryDir)
		if err != nil {
			t.Fatal(err)
		}
		if perm := dirInfo.Mode().Perm(); perm != 0o700 {
			t.Errorf("recovery folder mode = %o, want 700", perm)
		}
	}

	files, err := LoadRecoveries(m.recoveryDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("loaded %d files, want 1", len(files))
	}
	snap := files[0].Snap
	if snap.Root != store.Root() {
		t.Errorf("snapshot root = %q, want %q", snap.Root, store.Root())
	}
	if len(snap.Tabs) != 2 {
		t.Fatalf("kept %d tabs, want 2: %+v", len(snap.Tabs), snap.Tabs)
	}
	if snap.Tabs[0].ID != "a.md" || snap.Tabs[0].Content != "new" {
		t.Errorf("first tab = %+v", snap.Tabs[0])
	}
	if snap.Tabs[1].ID != "" || snap.Tabs[1].Content != "draft" {
		t.Errorf("draft tab = %+v", snap.Tabs[1])
	}
}

// A crash between the temporary write and the rename leaves the previous
// snapshots whole and no half-written file behind.
func TestSaveRecoveryKeepsThePreviousFileWhenTheRenameFails(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	first := testModel(t, store)
	first.recoveryDir = dir
	first.buffers = []*buffer{{id: "a.md", title: "a", saved: "old", text: "first"}}
	if _, err := first.SaveRecovery(); err != nil {
		t.Fatal(err)
	}

	second := testModel(t, store)
	second.recoveryDir = dir
	second.buffers = []*buffer{{id: "a.md", title: "a", saved: "old", text: "second"}}
	old := renameRecovery
	renameRecovery = func(string, string) error { return errors.New("power lost") }
	t.Cleanup(func() { renameRecovery = old })
	if _, err := second.SaveRecovery(); err == nil {
		t.Fatal("a failed rename must be reported")
	}

	files, err := LoadRecoveries(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || len(files[0].Snap.Tabs) != 1 || files[0].Snap.Tabs[0].Content != "first" {
		t.Fatalf("the previous snapshot was lost: %+v", files)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".recovery-") {
			t.Errorf("a half-written temporary file was left: %s", e.Name())
		}
	}
}

// A later session never overwrites a snapshot the user chose to keep: both
// are loaded and restored, and both disappear only when the user acts.
func TestRecoveryKeepsSnapshotsFromDifferentSessions(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	first := testModel(t, store)
	first.recoveryDir = dir
	first.buffers = []*buffer{{id: "a.md", title: "a", saved: "", text: "first"}}
	if _, err := first.SaveRecovery(); err != nil {
		t.Fatal(err)
	}
	second := testModel(t, store)
	second.recoveryDir = dir
	second.buffers = []*buffer{{id: "b.md", title: "b", saved: "", text: "second"}}
	if _, err := second.SaveRecovery(); err != nil {
		t.Fatal(err)
	}
	if files := recoveryFiles(t, dir); len(files) != 2 {
		t.Fatalf("files = %v, want 2 snapshots", files)
	}

	m := testModel(t, store)
	m.recoveryDir = dir
	files, err := LoadRecoveries(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.applyRecovery(recoveryLoadedMsg{files: files})
	if m.confirm == nil {
		t.Fatal("no recovery prompt")
	}
	if got := len(m.confirm.message); got < 1 || !strings.Contains(strings.Join(m.confirm.message, " "), "2 tab(s)") {
		t.Fatalf("prompt does not offer both tabs: %v", m.confirm.message)
	}
	m.confirm.onChoose(&m, 0)

	if files := recoveryFiles(t, dir); len(files) != 0 {
		t.Errorf("files left after restore: %v", files)
	}
	for _, id := range []string{"a.md", "b.md"} {
		if i := m.bufferIndex(id); i < 0 {
			t.Errorf("%s was not restored", id)
		}
	}
}

// Temporary files a crash left mid-write are removed, and never mistaken for
// a snapshot.
func TestLoadRecoveryRemovesStaleTemps(t *testing.T) {
	dir := t.TempDir()
	snap := RecoverySnapshot{Version: recoveryVersion, Tabs: []Recovery{{ID: "a.md", Title: "a", Content: "x"}}}
	data, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, recoveryPrefix+"1"+recoverySuffix), data, 0o600); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(dir, ".recovery-123.tmp")
	if err := os.WriteFile(stale, []byte("half"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := LoadRecoveries(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || len(files[0].Snap.Tabs) != 1 || files[0].Snap.Tabs[0].Content != "x" {
		t.Fatalf("snapshots = %+v", files)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the stale temporary file was not removed: %v", err)
	}
}

// A truncated snapshot is reported and left where it is, never guessed at.
func TestLoadRecoveryRejectsATruncatedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, recoveryPrefix+"1"+recoverySuffix)
	if err := os.WriteFile(path, []byte(`{"version":1,"tabs":[`), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := LoadRecoveries(dir)
	if err == nil {
		t.Fatal("a truncated file must be an error")
	}
	if len(files) != 0 {
		t.Errorf("a truncated file was loaded: %+v", files)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the unreadable file was removed: %v", err)
	}
}

// Restoring brings the text back as a modified tab, never as a write to the
// note on disk, and removes the snapshot.
func TestRecoveryRestoreKeepsTheNoteOnDiskAndOpensAModifiedTab(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAs("", "a", "disk"); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()

	writer := testModel(t, store)
	writer.recoveryDir = dir
	writer.buffers = []*buffer{{id: "a.md", title: "a", saved: "disk", text: "recovered"}}
	if _, err := writer.SaveRecovery(); err != nil {
		t.Fatal(err)
	}

	m := testModel(t, store)
	m.recoveryDir = dir
	files, err := LoadRecoveries(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.applyRecovery(recoveryLoadedMsg{files: files})
	if m.confirm == nil {
		t.Fatal("no recovery prompt")
	}
	if len(m.confirm.buttons) != 3 || !m.confirm.deliberate || m.confirm.selected != 0 {
		t.Fatalf("prompt = %+v", m.confirm)
	}
	m.confirm.onChoose(&m, 0)

	if files := recoveryFiles(t, dir); len(files) != 0 {
		t.Errorf("files left after restore: %v", files)
	}
	p, err := store.Get("a.md")
	if err != nil || p.Content != "disk" {
		t.Fatalf("the note on disk changed: %+v, %v", p, err)
	}
	i := m.bufferIndex("a.md")
	if i < 0 {
		t.Fatal("the recovered note has no tab")
	}
	b := m.buffers[i]
	if got := m.liveText(b); got != "recovered" {
		t.Errorf("tab text = %q", got)
	}
	if !m.isDirty(b) {
		t.Error("the recovered tab must be modified, not silently saved")
	}
}

// Later keeps the snapshots for the next start and opens nothing.
func TestRecoveryLaterKeepsTheFiles(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writer := testModel(t, store)
	writer.recoveryDir = dir
	writer.buffers = []*buffer{{id: "a.md", title: "a", saved: "", text: "kept"}}
	if _, err := writer.SaveRecovery(); err != nil {
		t.Fatal(err)
	}
	m := testModel(t, store)
	m.recoveryDir = dir
	files, err := LoadRecoveries(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.applyRecovery(recoveryLoadedMsg{files: files})
	if m.confirm == nil {
		t.Fatal("no recovery prompt")
	}
	m.confirm.onChoose(&m, 2)
	if got := recoveryFiles(t, dir); len(got) != 1 {
		t.Errorf("Later removed the snapshot: %v", got)
	}
	if len(m.buffers) != 0 {
		t.Errorf("Later opened %d tabs", len(m.buffers))
	}
}

// Discard throws the snapshots away without opening anything.
func TestRecoveryDiscardRemovesTheFile(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writer := testModel(t, store)
	writer.recoveryDir = dir
	writer.buffers = []*buffer{{id: "a.md", title: "a", saved: "disk", text: "recovered"}}
	if _, err := writer.SaveRecovery(); err != nil {
		t.Fatal(err)
	}

	m := testModel(t, store)
	m.recoveryDir = dir
	files, err := LoadRecoveries(dir)
	if err != nil {
		t.Fatal(err)
	}
	m.applyRecovery(recoveryLoadedMsg{files: files})
	if m.confirm == nil {
		t.Fatal("no recovery prompt")
	}
	m.confirm.onChoose(&m, 1)
	if files := recoveryFiles(t, dir); len(files) != 0 {
		t.Errorf("files left after discard: %v", files)
	}
	if len(m.buffers) != 0 {
		t.Errorf("discard opened %d tabs", len(m.buffers))
	}
}

// A snapshot from another vault is named and kept, never offered here.
func TestRecoveryFromAnotherVaultIsNotOffered(t *testing.T) {
	store, err := core.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := testModel(t, store)
	m.recoveryDir = t.TempDir()
	m.applyRecovery(recoveryLoadedMsg{files: []RecoveryFile{{
		Path: filepath.Join(m.recoveryDir, recoveryPrefix+"1"+recoverySuffix),
		Snap: RecoverySnapshot{
			Version: recoveryVersion,
			Root:    "/somewhere/else",
			Tabs:    []Recovery{{ID: "a.md", Title: "a", Content: "x"}},
		},
	}}})
	if m.confirm != nil {
		t.Error("another vault's snapshot must not be offered")
	}
	if !strings.Contains(m.status, "Unsaved notes from") {
		t.Errorf("status = %q", m.status)
	}
}
