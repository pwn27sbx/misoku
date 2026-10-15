package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/gitsync"

	tea "github.com/charmbracelet/bubbletea"
)

// exportDoc exports the vault's doc.md to doc.pdf with space P, as the user
// would.
func exportDoc(t *testing.T, h *harness, vault string) {
	t.Helper()
	h.keys("1")
	h.key(tea.KeyEsc)
	h.keys(" P")
	if _, err := os.Stat(filepath.Join(vault, "doc.pdf")); err != nil {
		t.Fatalf("space P must export doc.pdf: %v", err)
	}
}

// nestedGitRepo makes a repository with one commit at name in vault.
func nestedGitRepo(t *testing.T, vault, name string) {
	t.Helper()
	writeOutside(t, vault, name+"/inner.md", "inner")
	dir := filepath.Join(vault, filepath.FromSlash(name))
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "inner")
}

// S15: the PDF of :export is part of the notes and syncs like any file.
func TestSyncPushesAnExportedPDF(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	h := window(t, vault, syncConfig(t, vault))
	exportDoc(t, h, vault)
	h.finish(h.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, "doc.pdf") {
		t.Fatal("the exported PDF must be pushed")
	}
}

// S15: unless the user ignores it.
func TestSyncLeavesOutAPDFTheUserIgnores(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	writeOutside(t, vault, ".gitignore", "*.pdf\n")
	h := window(t, vault, syncConfig(t, vault))
	exportDoc(t, h, vault)
	h.finish(h.update(app.SyncNowMsg{}))
	if !onRemote(t, remote, ".gitignore") {
		t.Fatal("the sync must push the .gitignore")
	}
	if onRemote(t, remote, "doc.pdf") {
		t.Fatal("an ignored PDF must stay out of sync")
	}
	if _, err := os.Stat(filepath.Join(vault, "doc.pdf")); err != nil {
		t.Fatalf("the ignored PDF must stay on disk: %v", err)
	}
}

// S15: a repository inside the vault is named in the sync result, in the
// indicator and in :sync status, and none of it is pushed.
func TestSyncWarnsAboutANestedRepository(t *testing.T) {
	t.Parallel()
	remote, vault := syncedVault(t)
	nestedGitRepo(t, vault, "code")
	writeOutside(t, vault, "new.md", "fresh")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantBar(t, h, "code is a separate git repository")
	if !onRemote(t, remote, "new.md") {
		t.Fatal("the rest of the vault must still sync")
	}
	if onRemote(t, remote, "code") {
		t.Fatal("the nested repository must stay out of the commit")
	}
	wantBar(t, h, "1 file to check")

	text := noSpace(dialogText(openSyncStatus(h)))
	for _, want := range []string{"Files that do not sync as they are (1):", "code is a separate git repository"} {
		if !strings.Contains(text, noSpace(want)) {
			t.Errorf(":sync status must show %q:\n%s", want, text)
		}
	}
}

// S15: several such files are counted in the status bar and listed in
// :sync status.
func TestSyncCountsSeveralFilesThatDoNotSync(t *testing.T) {
	t.Parallel()
	_, vault := syncedVault(t)
	nestedGitRepo(t, vault, "one")
	nestedGitRepo(t, vault, "two")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantBar(t, h, "2 files do not sync as they are")

	text := noSpace(dialogText(openSyncStatus(h)))
	for _, want := range []string{"one is a separate git repository", "two is a separate git repository"} {
		if !strings.Contains(text, noSpace(want)) {
			t.Errorf(":sync status must show %q:\n%s", want, text)
		}
	}
}

// The status bar and the indicator count the same files: big ones (D7)
// and those of S15 together.
func TestSyncBarAndIndicatorCountTheSameFiles(t *testing.T) {
	_, vault := syncedVault(t)
	old := gitsync.MaxFileSize
	gitsync.MaxFileSize = 10
	t.Cleanup(func() { gitsync.MaxFileSize = old })
	writeOutside(t, vault, "big.png", "eleven byte")
	nestedGitRepo(t, vault, "code")
	h := window(t, vault, syncConfig(t, vault))
	h.finish(h.update(app.SyncNowMsg{}))
	wantBar(t, h, "2 files do not sync as they are")
	wantBar(t, h, "2 files to check")
}
