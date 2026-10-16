//go:build !windows

package gitsync_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// Runner runs the git it found in Program Files (simulated: only a Unix
// system can run the fake script).
func TestRunnerRunsProgramFilesGit(t *testing.T) {
	defer gitsync.SetGOOS("windows")()
	pf := t.TempDir()
	gitAt(t, filepath.Join(pf, "Git", "cmd"), "git.exe")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ProgramFiles", pf)

	version, ok, err := gitsync.New(t.TempDir(), 10*time.Second).Version(context.Background())
	if err != nil || !ok || version != "2.45" {
		t.Fatalf("Version = %q, %v, %v; want 2.45 from Program Files", version, ok, err)
	}
}
