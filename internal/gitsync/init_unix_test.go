//go:build !windows

package gitsync_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// envOfUploadPack runs `ls-remote` against a local remote through r, with
// an upload-pack that first writes the environment git gave it, and
// returns that environment.
func envOfUploadPack(t *testing.T, r *gitsync.Runner, remote string) string {
	t.Helper()
	dir := t.TempDir()
	envFile := filepath.Join(dir, "env")
	script := filepath.Join(dir, "upload-pack")
	body := "#!/bin/sh\nenv > '" + envFile + "'\nexec git-upload-pack \"$@\"\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(context.Background(), "ls-remote", "--upload-pack="+script, remote); err != nil {
		t.Fatal(err)
	}
	env, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	return string(env)
}

// The real git processes: `misoku sync init` lets git ask on the network
// step, and the background sync still never lets it.
func TestOnlyTheInteractiveRunnerLetsGitAsk(t *testing.T) {
	t.Parallel()
	isolate(t)
	remote := newRemote(t)
	repo := newRepo(t)

	env := envOfUploadPack(t, gitsync.New(repo, 10*time.Second), remote)
	for _, want := range []string{"GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never", "BatchMode=yes"} {
		if !strings.Contains(env, want) {
			t.Errorf("the background runner must keep %s; env was:\n%s", want, env)
		}
	}

	env = envOfUploadPack(t, gitsync.NewInteractive(repo, 10*time.Second), remote)
	for _, kv := range []string{"GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never", "BatchMode=yes", "SSH_ASKPASS_REQUIRE=never"} {
		k, _, _ := strings.Cut(kv, "=")
		if strings.Contains(os.Getenv(k), strings.TrimPrefix(kv, k+"=")) {
			continue // the user's own setting, left alone
		}
		if strings.Contains(env, kv) {
			t.Errorf("sync init must let git and ssh ask (%s found); env was:\n%s", kv, env)
		}
	}
}
