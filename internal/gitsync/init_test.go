package gitsync_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// Only the network commands of `misoku sync init` may ask; everything
// else, and every command of the background runner, runs with every prompt
// off.
func TestInteractiveRunnerAsksOnlyOnTheNetwork(t *testing.T) {
	t.Parallel()
	isolate(t)
	repo := newRepo(t)
	off := []string{"GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never", "SSH_ASKPASS_REQUIRE=never"}
	has := func(env []string, kv string) bool {
		for _, e := range env {
			if e == kv {
				return true
			}
		}
		return false
	}
	background, interactive := gitsync.New(repo, time.Second), gitsync.NewInteractive(repo, time.Second)
	for _, cmd := range []string{"ls-remote", "fetch", "push", "commit", "status", "merge", "add"} {
		env := background.EnvFor(t.Context(), cmd)
		for _, kv := range off {
			if !has(env, kv) {
				t.Errorf("background %s: %s missing", cmd, kv)
			}
		}
		network := cmd == "ls-remote" || cmd == "fetch" || cmd == "push"
		env = interactive.EnvFor(t.Context(), cmd)
		for _, kv := range off {
			k, v, _ := strings.Cut(kv, "=")
			if network && os.Getenv(k) == v {
				continue // the user's own setting, left alone
			}
			if has(env, kv) == network {
				if network {
					t.Errorf("sync init %s must let git ask, found %s", cmd, kv)
				} else {
					t.Errorf("sync init %s must not ask, %s missing", cmd, kv)
				}
			}
		}
	}
}
