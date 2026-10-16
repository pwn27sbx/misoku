//go:build !windows

// The setup assistant against a fake GitHub served over HTTP with basic
// auth, where git gets in only through gh's credential helper (from the
// independent verification of F2.14). The fake gh is a shell script, so
// these run on macOS and Linux.

package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// probeToken is the access the fake gh keeps: Misoku must never show,
// store or pass it.
const probeToken = "gho_PROBEtoken1234567890abcdefXYZ"

// probeGHScript answers like gh. Unlike the writer's fake, `auth
// setup-git` installs a real credential helper (this same script) for
// $PROBE_SERVER, and `auth git-credential get` answers with the token, so
// git signs in to an HTTP server only through gh. Every failure message
// and `auth status` carries the token, to catch any leak.
const probeGHScript = `#!/bin/sh
d="$PROBE_GH_DIR"
printf '%s\n' "$*" >> "$d/calls"
case "$1 $2" in
"--version ") echo "gh version 2.62.0 (2024-11-14)" ;;
"auth status")
	echo "  Logged in to github.com account ana (keyring)"
	echo "  - Token: ` + probeToken + `"
	[ -f "$d/loggedin" ] && exit 0
	echo "not logged in (` + probeToken + `)" >&2; exit 1 ;;
"auth setup-git")
	[ -f "$d/setupgitfail" ] && { echo "setup-git failed ` + probeToken + `" >&2; exit 1; }
	git config --global "credential.$PROBE_SERVER.helper" "!$d/bin/gh auth git-credential" ;;
"auth git-credential")
	cat >/dev/null
	[ "$3" = get ] && printf 'username=x-access-token\npassword=%s\n' "` + probeToken + `"
	exit 0 ;;
"repo list")
	[ -f "$d/listfail" ] && { echo "HTTP 502 https://ana:` + probeToken + `@api.github.com/graphql" >&2; exit 1; }
	cat "$d/repos.json" ;;
"repo create")
	[ -f "$d/createfail" ] && { echo "HTTP 403 (token ` + probeToken + `)" >&2; exit 1; }
	case "$*" in *--private*) ;; *) echo "repo create without --private" >&2; exit 3 ;; esac
	git init -q --bare "$d/remotes/ana/$3.git" && git -C "$d/remotes/ana/$3.git" config http.receivepack true || exit 1
	echo "https://github.com/ana/$3" ;;
*) echo "probe gh: unknown $*" >&2; exit 2 ;;
esac
`

// probeEnv is a fake GitHub served over HTTP with basic auth: git reaches
// it as https://github.com/ (url.insteadOf) and gets in only with the
// token gh's credential helper gives.
type probeEnv struct {
	dir, remotes, global, cfg, vault string
	srv                              *httptest.Server
	denied                           int
}

func newProbeEnv(t *testing.T) *probeEnv {
	t.Helper()
	needGit(t)
	gitPath, _ := exec.LookPath("git")
	e := &probeEnv{dir: t.TempDir()}
	e.remotes = filepath.Join(e.dir, "remotes")
	bin := filepath.Join(e.dir, "bin")
	for _, d := range []string{bin, filepath.Join(e.remotes, "ana")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(probeGHScript), 0o755); err != nil {
		t.Fatal(err)
	}
	backend := &cgi.Handler{Path: gitPath, Args: []string{"http-backend"},
		Env: []string{"GIT_PROJECT_ROOT=" + e.remotes, "GIT_HTTP_EXPORT_ALL=1"}}
	e.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u != "x-access-token" || p != probeToken {
			e.denied++
			w.Header().Set("WWW-Authenticate", `Basic realm="probe"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		backend.ServeHTTP(w, r)
	}))
	t.Cleanup(e.srv.Close)
	e.global = filepath.Join(e.dir, "gitconfig")
	cfg := "[user]\n\tname = Test\n\temail = test@example.com\n[init]\n\tdefaultBranch = main\n" +
		"[url \"" + e.srv.URL + "/\"]\n\tinsteadOf = https://github.com/\n"
	if err := os.WriteFile(e.global, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	writeFile(t, e.dir, "repos.json", "[]")
	writeFile(t, e.dir, "loggedin", "")
	t.Setenv("PROBE_GH_DIR", e.dir)
	t.Setenv("PROBE_SERVER", e.srv.URL)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_CONFIG_GLOBAL", e.global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// An askpass that records being asked: nothing may ever ask.
	askpass := filepath.Join(bin, "askpass")
	if err := os.WriteFile(askpass, []byte("#!/bin/sh\necho asked >> \""+e.dir+"/asked\"\necho wrong\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_ASKPASS", askpass)
	t.Setenv("SSH_ASKPASS", askpass)
	e.cfg = filepath.Join(t.TempDir(), "config.json")
	e.vault = t.TempDir()
	writeFile(t, e.vault, "hello.md", "# hello\n")
	return e
}

// remote makes ana/name on the fake GitHub, with files on main.
func (e *probeEnv) remote(t *testing.T, name string, files map[string]string) {
	t.Helper()
	bare := filepath.Join(e.remotes, "ana", name+".git")
	git(t, filepath.Dir(bare), "init", "-q", "--bare", name+".git")
	git(t, bare, "config", "http.receivepack", "true")
	if len(files) > 0 {
		seedRemote(t, bare, files)
	}
}

func (e *probeEnv) asked() bool {
	_, err := os.Stat(filepath.Join(e.dir, "asked"))
	return err == nil
}

// noToken fails when the token shows up in any file under dir.
func noToken(t *testing.T, label, dir string) {
	t.Helper()
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Base(p) == "gh" {
			return nil
		}
		if data, err := os.ReadFile(p); err == nil && bytes.Contains(data, []byte(probeToken)) {
			t.Errorf("%s: the token is stored in %s", label, p)
		}
		return nil
	})
}

func backgroundSync(t *testing.T, dir string) gitsync.SyncResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	r := gitsync.New(dir, 20*time.Second)
	if st := gitsync.Detect(ctx, r); st.Status != gitsync.Ready {
		t.Fatalf("after setup the vault must be ready: %s", st.Message)
	}
	lock, ok, err := gitsync.LockVault(ctx, r)
	if err != nil || !ok {
		t.Fatalf("lock: %v %v", ok, err)
	}
	defer lock.Unlock()
	return gitsync.SyncVault(ctx, r, lock, gitsync.SyncOptions{Device: "probe"})
}

// The command line creates a private repository over HTTPS, signs in only
// through gh's credential helper, stores no token anywhere, and the
// background sync afterwards pushes without ever asking.
func TestSyncSetupOverHTTPSSignsInOnlyThroughGHAndBackgroundNeverAsks(t *testing.T) {
	e := newProbeEnv(t)
	code, out := setup(t, e.vault, e.cfg, "\n1\nnotes\n")
	if code != 0 {
		t.Fatalf("setup: exit %d:\n%s", code, out)
	}
	if strings.Contains(out, probeToken) {
		t.Errorf("setup printed the token:\n%s", out)
	}
	if got := git(t, e.vault, "config", "--get", "remote.origin.url"); got != "https://github.com/ana/notes.git" {
		t.Errorf("origin = %q, want the plain HTTPS address", got)
	}
	writeFile(t, e.vault, "later.md", "after setup\n")
	if res := backgroundSync(t, e.vault); res.Outcome != gitsync.Synced {
		t.Fatalf("background sync over HTTPS: %v", res.Err)
	}
	if remoteFiles(t, filepath.Join(e.remotes, "ana", "notes.git"))["later.md"] == "" {
		t.Error("the background sync did not push the new note")
	}
	if e.asked() {
		t.Error("git asked for a password (askpass ran)")
	}
	for _, d := range []string{e.vault, filepath.Dir(e.cfg)} {
		noToken(t, "after setup", d)
	}
	if data, _ := os.ReadFile(e.global); bytes.Contains(data, []byte(probeToken)) {
		t.Error("the token is in git's global config")
	}
}

// Without gh's access, the background sync fails at once: it never asks
// and never hangs.
func TestBackgroundSyncWithoutGHAccessFailsAtOnceWithoutAsking(t *testing.T) {
	e := newProbeEnv(t)
	if code, out := setup(t, e.vault, e.cfg, "\n1\nnotes\n"); code != 0 {
		t.Fatalf("setup: exit %d:\n%s", code, out)
	}
	git(t, e.vault, "config", "--global", "--unset-all", "credential."+e.srv.URL+".helper")
	// GIT_ASKPASS and SSH_ASKPASS stay set (newProbeEnv): background git
	// must not run them.
	writeFile(t, e.vault, "later.md", "after setup\n")
	start := time.Now()
	res := backgroundSync(t, e.vault)
	if res.Outcome == gitsync.Synced {
		t.Fatal("synced without access")
	}
	if time.Since(start) > 15*time.Second {
		t.Errorf("the failing background sync took %s", time.Since(start))
	}
	if e.asked() {
		t.Error("the background sync asked for a password")
	}
}

// Inside the app the engine runs with Interactive false: the init itself
// never asks either, and reaches GitHub through gh's helper.
func TestSyncSetupEngineInTheAppConnectsWithoutAsking(t *testing.T) {
	e := newProbeEnv(t)
	e.remote(t, "notas", map[string]string{"remote.md": "from the other computer\n"})
	writeFile(t, e.dir, "repos.json", `[{"nameWithOwner":"ana/notas","visibility":"PRIVATE","url":"https://github.com/ana/notas"}]`)
	s := gitsync.NewSetup(gitsync.SetupOptions{Dir: e.vault, ConfigPath: e.cfg, Timeout: 30 * time.Second})
	ctx := context.Background()
	if st := s.Begin(ctx); st.Ask != gitsync.AskStart {
		t.Fatalf("begin: %+v", st)
	}
	if st := s.Next(ctx, gitsync.SetupAnswer{OK: true}); st.Ask != gitsync.AskRepo {
		t.Fatalf("after start: %+v", st)
	}
	st := s.Next(ctx, gitsync.SetupAnswer{OK: true, Choice: 0})
	if st.Outcome != gitsync.SetupDone {
		t.Fatalf("pick: %+v", st)
	}
	if e.asked() {
		t.Error("the in-app init asked for a password")
	}
	files := vaultFiles(t, e.vault)
	if files["remote.md"] == "" || files["hello.md"] == "" {
		t.Errorf("both sides must be kept, got %v", files)
	}
	noToken(t, "in-app setup", e.vault)
}

// Public and organization-internal repositories are refused whether
// picked by number or typed by name, and nothing is changed.
func TestSyncSetupRefusesPublicAndInternalByNumberAndName(t *testing.T) {
	for _, input := range []string{"\n1\n", "\n2\n", "\nana/pub\n", "\nint\n", "\nORG/INT\n"} {
		e := newProbeEnv(t)
		writeFile(t, e.dir, "repos.json", `[{"nameWithOwner":"ana/pub","visibility":"PUBLIC","url":"https://github.com/ana/pub"},{"nameWithOwner":"org/int","visibility":"INTERNAL","url":"https://github.com/org/int"}]`)
		before := treeHash(t, e.vault) + fileHash(t, e.cfg) + fileHash(t, e.global)
		code, out := setup(t, e.vault, e.cfg, input)
		if code == 0 || !strings.Contains(out, "not allowed") {
			t.Errorf("input %q: exit %d, want a refusal:\n%s", input, code, out)
		}
		if after := treeHash(t, e.vault) + fileHash(t, e.cfg) + fileHash(t, e.global); after != before {
			t.Errorf("input %q: a refused repository changed something", input)
		}
	}
}

// Every gh failure is reported without the token, and changes nothing in
// the vault or Misoku's config.
func TestSyncSetupGHFailuresNeverShowTheTokenOrChangeTheVault(t *testing.T) {
	for _, tc := range []struct{ name, flag, input string }{
		{"signed out, sign-in declined", "loggedout", "\nn\n"},
		{"list fails", "listfail", "\n"},
		{"setup-git fails on pick", "setupgitfail", "\n1\n"},
		{"setup-git fails on create", "setupgitfail", "\n2\nnew\n"},
		{"create fails", "createfail", "\n2\nnew\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newProbeEnv(t)
			e.remote(t, "notas", nil)
			writeFile(t, e.dir, "repos.json", `[{"nameWithOwner":"ana/notas","visibility":"PRIVATE","url":"https://github.com/ana/notas"}]`)
			if tc.flag == "loggedout" {
				if err := os.Remove(filepath.Join(e.dir, "loggedin")); err != nil {
					t.Fatal(err)
				}
			} else {
				writeFile(t, e.dir, tc.flag, "")
			}
			before := treeHash(t, e.vault) + fileHash(t, e.cfg)
			code, out := setup(t, e.vault, e.cfg, tc.input)
			if code == 0 {
				t.Errorf("exit 0, want a failure:\n%s", out)
			}
			if strings.Contains(out, probeToken) || strings.Contains(out, "gho_") {
				t.Errorf("the token was printed:\n%s", out)
			}
			if after := treeHash(t, e.vault) + fileHash(t, e.cfg); after != before {
				t.Error("a failed setup changed the vault or Misoku's config")
			}
		})
	}
}
