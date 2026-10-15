//go:build !windows

package gitsync_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// A file git cannot replace (on Windows, one open in another app; here, a
// folder without write permission) stops the fast-forward half-way. The
// sync says which file and tries again next time; once the file is free
// the next sync finishes and nothing is lost.
func TestSyncRetriesWhenAFileCannotBeReplaced(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root writes to read-only folders")
	}
	remote, laptop, desk := pair(t, map[string]string{"busy/x.md": "one\n", "b.md": "one\n"})
	write(t, laptop, "busy/x.md", "two\n")
	write(t, laptop, "b.md", "two\n")
	wantOutcome(t, syncVault(t, laptop, "laptop", false), gitsync.Synced)

	busy := filepath.Join(desk, "busy")
	if err := os.Chmod(busy, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(busy, 0o755) })
	res := syncVault(t, desk, "desk", false)
	wantOutcome(t, res, gitsync.SyncRetry)
	if !strings.Contains(res.Err.Error(), "busy/x.md") {
		t.Fatalf("the error must name the file, got %v", res.Err)
	}

	if err := os.Chmod(busy, 0o755); err != nil {
		t.Fatal(err)
	}
	wantOutcome(t, syncVault(t, desk, "desk", false), gitsync.Synced)
	wantFile(t, desk, "busy/x.md", "two\n")
	wantFile(t, desk, "b.md", "two\n")
	wantInSync(t, remote, desk, "main")
}
