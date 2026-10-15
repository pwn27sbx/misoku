package gitsync_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

const testSecret = "s3cretTOKEN"

// httpOrigin is an in-process HTTP server answering every request with
// status, standing in for a git host that refuses or fails.
func httpOrigin(t *testing.T, status int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status == http.StatusUnauthorized {
			w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://")
}

func noSecret(t *testing.T, where, text string) {
	t.Helper()
	for _, bad := range []string{testSecret, "me:", "x-access-token:"} {
		if strings.Contains(text, bad) {
			t.Errorf("%s shows a credential (%q): %s", where, bad, text)
		}
	}
}

// whatever origin answers, over every URL shape that
// carries a credential, neither the sync's error nor `:sync status` (Inspect)
// shows it. Fetch side (origin's URL) and push side (pushurl) both.
func TestSyncNeverShowsACredentialWhateverOriginAnswers(t *testing.T) {
	t.Parallel()
	type tc struct {
		name, url string
		push      bool // the credential is in pushurl; fetch goes to the real remote
	}
	var cases []tc
	for _, status := range []int{401, 403, 404, 500} {
		host := httpOrigin(t, status)
		for i, shape := range []string{
			"http://me:" + testSecret + "@" + host + "/notes.git",
			"http://" + testSecret + "@" + host + "/notes.git",
			"http://x-access-token:" + testSecret + "@" + host + "/notes.git",
			"http://" + host + "/notes.git?token=" + testSecret,
		} {
			cases = append(cases, tc{fmt.Sprintf("fetch-%d-%d", status, i), shape, false})
			cases = append(cases, tc{fmt.Sprintf("push-%d-%d", status, i), shape, true})
		}
	}
	// Nobody listening: offline.
	cases = append(cases, tc{"refused", "http://me:" + testSecret + "@127.0.0.1:1/notes.git", false},
		tc{"refused-push", "http://me:" + testSecret + "@127.0.0.1:1/notes.git", true})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			remote, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
			_ = remote
			write(t, laptop, "a.md", "a\n")
			if c.push {
				git(t, laptop, "config", "remote.origin.pushurl", c.url)
			} else {
				git(t, laptop, "remote", "set-url", "origin", c.url)
			}
			start := time.Now()
			res := syncVault(t, laptop, "laptop", false)
			if took := time.Since(start); took > 15*time.Second {
				t.Errorf("took %s", took)
			}
			if res.Outcome == gitsync.Synced {
				t.Fatalf("cannot be synced: %+v", res)
			}
			if res.Err != nil {
				noSecret(t, "SyncResult.Err", res.Err.Error())
			}
			t.Logf("%v: %v", res.Outcome, res.Err)
			info := gitsync.Inspect(context.Background(), gitsync.New(laptop, 10*time.Second))
			noSecret(t, "Inspect.Remote", info.Remote)
			if info.Err != nil {
				noSecret(t, "Inspect.Err", info.Err.Error())
			}
			if !c.push && res.Pending < 1 {
				t.Errorf("the commit must stay pending: %+v", res)
			}
		})
	}
}

// shapes of URLs a helper or a host might print in an error.
func TestSyncRedactsEveryPrintedURLShape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake ssh is a shell script")
	}
	isolate(t)
	dir := clone(t, newRemote(t))
	msg := strings.Join([]string{
		"fatal: unable to access 'https://x-access-token:T1" + testSecret + "@github.com/o/r.git/': denied",
		"remote: see https://me:T2" + testSecret + "@example.com/help.",
		"(https://u:T3" + testSecret + "@h.example/x)",
		"`https://u:T4" + testSecret + "@h.example/x`",
		"https://h.example/x?access_token=T5" + testSecret,
		"ssh://git:T6" + testSecret + "@h.example/x",
		"Authentication failed for 'HTTPS://U:T7" + testSecret + "@H.EXAMPLE/x'",
		"[https://u:T8" + testSecret + "@h.example/x]",
		"https://u:T9" + testSecret + "@h.example:badport/x",
	}, "\n")
	fakeSSH(t, dir, msg)
	res := syncVault(t, dir, "laptop", false)
	if res.Err == nil {
		t.Fatal("want an error")
	}
	noSecret(t, "Err", res.Err.Error())
	t.Logf("%v", res.Err)
}

func TestRedactURLHandlesOddShapes(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"https://me:" + testSecret + "@h/x":         "https://h/x",
		"https://" + testSecret + "@h/x":            "https://h/x",
		"https://h/x?token=" + testSecret:           "https://h/x",
		"ssh://git:" + testSecret + "@h/x":          "ssh://git@h/x",
		"ssh://git@h/x":                             "ssh://git@h/x",
		"git@github.com:o/r.git":                    "git@github.com:o/r.git",
		"https://me:p%zz" + testSecret + "@h/x":     "https://h/x",
		"https://me:" + testSecret + "@h:badport/x": "https://h:badport/x",
	} {
		if got := gitsync.RedactURL(in); strings.Contains(got, testSecret) || got != want {
			t.Errorf("RedactURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// vaultHash hashes every file of the vault, git's folder included, except
// what a refused sync may change by design (Misoku's own marker and lock).
func vaultHash(t *testing.T, dir string) string {
	t.Helper()
	h := sha256.New()
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if n := d.Name(); n == "misoku-sync.running" || n == "misoku-sync.lock" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		fmt.Fprintf(h, "%s\x00%x\x00", rel, sha256.Sum256(b))
		return nil
	})
	return fmt.Sprintf("%x", h.Sum(nil))
}

// a recovery that refuses (a lock older than the
// command; a live git; a marker without pid) changes nothing in the
// worktree, the index, or the refs.
func TestSyncRefusedRecoveryChangesNothing(t *testing.T) {
	t.Parallel()
	for name, setup := range map[string]func(t *testing.T, dir string){
		"older-lock": func(t *testing.T, dir string) {
			start := time.Now()
			lock := interrupted(t, dir, deadPid(t), start)
			old := start.Add(-time.Hour)
			os.Chtimes(lock, old, old)
		},
		"live-git": func(t *testing.T, dir string) {
			interrupted(t, dir, os.Getpid(), time.Now().Add(-time.Second))
		},
		"no-pid": func(t *testing.T, dir string) {
			interrupted(t, dir, 0, time.Now().Add(-time.Second))
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, laptop, _ := pair(t, map[string]string{"seed.md": "s\n"})
			write(t, laptop, "a.md", "unsaved work\n")
			setup(t, laptop)
			before := vaultHash(t, laptop)
			for range 2 {
				res := syncVault(t, laptop, "laptop", false)
				if res.Outcome == gitsync.Synced {
					t.Fatalf("must not sync: %+v", res)
				}
				t.Logf("%v: %v", res.Outcome, res.Err)
			}
			if after := vaultHash(t, laptop); after != before {
				t.Fatal("a refused recovery changed the vault")
			}
			wantLock(t, filepath.Join(laptop, ".git", "index.lock"), true)
		})
	}
}
