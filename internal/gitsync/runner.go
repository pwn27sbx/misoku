// Package gitsync runs the system git for Misoku's note sync.
//
// Every command runs in the background and must never stop to ask the user
// anything: Misoku owns the terminal, so a password prompt, an SSH host-key
// question, a GPG pinentry or a hook writing to the screen would corrupt it
// or hang forever. Runner therefore starts git without a controlling
// terminal and with empty stdin, turns every known prompt off, skips hooks
// and commit signing, and kills git (and its children, such as ssh) when a
// command runs past its time limit.
package gitsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MinVersion is the oldest git Misoku syncs with: merge-tree --write-tree,
// which computes a merge without touching the working tree, needs 2.38.
const MinVersion = "2.38"

// Change is one entry of `git status`: the two-letter XY code and the path,
// exactly as stored on disk (no quoting or escaping). From is the original
// path of a rename or copy (code R or C), empty otherwise.
type Change struct{ Code, Path, From string }

// Runner runs git inside one repository.
type Runner struct {
	dir     string
	timeout time.Duration

	// sshMu guards ssh, the GIT_SSH_COMMAND every command runs with
	// (sshSet false: leave the user's environment alone). Reading the
	// user's ssh setting costs a git process, so it is done once
	// (sshDone); a read cut short by the caller is done again next time.
	sshMu   sync.Mutex
	sshDone bool
	ssh     string
	sshSet  bool

	// marker, when set, is the file SyncVault keeps while each of its git
	// commands that take the index lock runs (takesIndexLock, recover.go),
	// so a lock a dead git left behind can be told apart from the user's.
	// Commands that never take it (ls-remote, fetch, push) leave no marker:
	// a lock seen after them can only be someone else's.
	marker string

	// interactive lets the network commands (askingCommands) ask the user,
	// for `misoku sync init` in the foreground (NewInteractive).
	interactive bool
}

// New returns a Runner for the repository at dir. Each command gets at most
// timeout to finish.
func New(dir string, timeout time.Duration) *Runner { return &Runner{dir: dir, timeout: timeout} }

// NewInteractive returns a Runner for `misoku sync init`, which runs in
// the foreground of the user's terminal (D8): its network commands
// (ls-remote, fetch and push) run with the user's own environment, stdin
// and terminal, so git, ssh or a credential helper may ask for a password
// or to accept a new SSH host key, and Ctrl+C reaches them. Every other
// command runs exactly as with New, prompts off.
func NewInteractive(dir string, timeout time.Duration) *Runner {
	return &Runner{dir: dir, timeout: timeout, interactive: true}
}

// askingCommands are the git commands that talk to the remote: the only
// ones an interactive Runner lets ask. None of them takes the index lock,
// so the marker recovery relies on (recover.go) never covers them.
var askingCommands = map[string]bool{"ls-remote": true, "fetch": true, "push": true}

// asks reports whether git subcommand cmd may ask the user.
func (r *Runner) asks(cmd string) bool { return r.interactive && askingCommands[cmd] }

var versionRe = regexp.MustCompile(`git version (\d+)\.(\d+)`)

// ParseVersion reads the output of `git version` and returns "major.minor"
// and whether it is at least MinVersion.
func ParseVersion(out string) (string, bool) {
	m := versionRe.FindStringSubmatch(out)
	if m == nil {
		return "", false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return m[1] + "." + m[2], major > 2 || (major == 2 && minor >= 38)
}

// Version returns the installed git's "major.minor" and whether Misoku can
// sync with it.
func (r *Runner) Version(ctx context.Context) (string, bool, error) {
	out, err := r.Run(ctx, "version")
	if err != nil {
		return "", false, err
	}
	v, ok := ParseVersion(string(out))
	return v, ok, nil
}

// Run runs git with args in the repository and returns its standard output.
// A failure carries git's own error text. When the command is stopped
// (its time limit, or the caller's context), the error matches the
// context's error with errors.Is and still carries what git printed.
func (r *Runner) Run(parent context.Context, args ...string) ([]byte, error) {
	return r.runWith(parent, nil, nil, args...)
}

// runWith is Run with extra environment variables (such as GIT_INDEX_FILE)
// and stdin fed from a buffer instead of the null device.
func (r *Runner) runWith(parent context.Context, extraEnv []string, stdin []byte, args ...string) ([]byte, error) {
	// Resolved before the time limit starts, so it never eats into it.
	asks := r.asks(firstWord(args))
	env := append(r.envFor(parent, firstWord(args)), extraEnv...)
	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()

	full := append([]string{
		"-C", r.dir,
		// No hooks: a hook could prompt, print to the terminal or take long.
		"-c", "core.hooksPath=" + os.DevNull,
		// No signing: gpg would open a pinentry.
		"-c", "commit.gpgsign=false",
		"-c", "tag.gpgsign=false",
	}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Env = env
	cmd.Stdin = nil // reads from the null device
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	// After a kill, give pipes held by orphaned children this long to close
	// before Wait gives up on them.
	cmd.WaitDelay = 2 * time.Second

	var started func(pid int)
	marked := r.marker != "" && takesIndexLock(firstWord(args))
	if marked {
		start := time.Now()
		if err := writeMarker(r.marker, 0, start, r.timeout); err != nil {
			return nil, fmt.Errorf("could not note the running git: %w", err)
		}
		started = func(pid int) {
			// Best effort: without the pid the marker still says the lock
			// would be Misoku's, and recovery stays on the safe side (it
			// never removes a lock whose git it cannot check).
			_ = writeMarker(r.marker, pid, start, r.timeout)
		}
	}
	var err error
	if asks {
		err = runInTerminal(cmd, started)
	} else {
		err = run(cmd, started)
	}
	if marked {
		// git is gone now. Killed at its time limit, it may have left
		// index.lock behind: that lock is removed here. Otherwise git
		// removed its own locks on the way out, and any lock there now is
		// someone else's.
		var ferr error
		if ctx.Err() != nil {
			ferr = recoverIndexLock(r.marker, true)
		} else {
			ferr = removeMarker(r.marker)
		}
		if err == nil && ferr != nil {
			return stdout.Bytes(), ferr
		}
	}
	if err == nil {
		return stdout.Bytes(), nil
	}
	msg := strings.TrimSpace(stderr.String())
	if ctxErr := ctx.Err(); ctxErr != nil {
		why := fmt.Sprintf("no answer after %s", r.timeout)
		if perr := parent.Err(); errors.Is(perr, context.Canceled) {
			why = "cancelled"
		} else if perr != nil {
			why = "the caller's time limit ran out"
		}
		why = "git " + firstWord(args) + ": " + why
		if msg != "" {
			why += ": " + msg
		}
		return stdout.Bytes(), &stoppedError{msg: why, cause: ctxErr}
	}
	if msg == "" {
		msg = err.Error()
	}
	return stdout.Bytes(), &gitError{msg: msg, cause: err}
}

// gitError is a git command that ran and failed. Its text is git's own
// error output; Unwrap gives the *exec.ExitError, so callers can tell exit
// codes apart (exitCode).
type gitError struct {
	msg   string
	cause error
}

func (e *gitError) Error() string { return e.msg }
func (e *gitError) Unwrap() error { return e.cause }

// exitCode returns the exit code of a git command that ran and failed, or
// -1 when err is not such a failure (git did not start, or was stopped).
func exitCode(err error) int {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

// stoppedError is a command killed because its context ended. Its text
// says why in words and keeps git's stderr; Unwrap gives the context error.
type stoppedError struct {
	msg   string
	cause error
}

func (e *stoppedError) Error() string { return e.msg }
func (e *stoppedError) Unwrap() error { return e.cause }

// runInTerminal runs cmd as a plain child of Misoku, in the user's
// terminal and with the user's stdin, so whatever git starts can ask the
// user (NewInteractive). Ctrl+C reaches git and ssh as it reaches Misoku.
func runInTerminal(cmd *exec.Cmd, started func(pid int)) error {
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if started != nil {
		started(cmd.Process.Pid)
	}
	return cmd.Wait()
}

// envFor is the environment git subcommand cmd runs with: when it may ask
// (asks), the user's own with only the language fixed; otherwise env's.
func (r *Runner) envFor(ctx context.Context, cmd string) []string {
	if r.asks(cmd) {
		env := os.Environ()
		setEnv(&env, "LC_ALL", "C") // stable messages to parse
		setEnv(&env, "LANGUAGE", "C")
		return env
	}
	return r.env(ctx)
}

// setEnv sets k to v in env, replacing a value already there.
func setEnv(env *[]string, k, v string) {
	for i, kv := range *env {
		if strings.HasPrefix(kv, k+"=") {
			(*env)[i] = k + "=" + v
			return
		}
	}
	*env = append(*env, k+"="+v)
}

// env is the environment git runs with: the user's, with every prompt off.
// ctx bounds reading the user's ssh setting.
func (r *Runner) env(ctx context.Context) []string {
	env := os.Environ()
	set := func(k, v string) { setEnv(&env, k, v) }
	set("GIT_TERMINAL_PROMPT", "0")     // no username/password questions
	set("GCM_INTERACTIVE", "never")     // Git Credential Manager: no windows
	set("SSH_ASKPASS_REQUIRE", "never") // ssh: never pop up an askpass helper
	set("LC_ALL", "C")                  // stable messages to parse
	set("LANGUAGE", "C")
	r.sshMu.Lock()
	if !r.sshDone {
		r.ssh, r.sshSet = r.resolveSSH(ctx)
		// Stopped by the caller, the read may have missed the setting:
		// the command fails anyway, and the next one reads it again.
		r.sshDone = ctx.Err() == nil
	}
	ssh, sshSet := r.ssh, r.sshSet
	r.sshMu.Unlock()
	if sshSet {
		set("GIT_SSH_COMMAND", ssh)
	}
	return env
}

// resolveSSH returns the ssh command the user already configured with its
// no-prompt flag added, and false when it cannot be added safely.
//
// It follows git's own order (GIT_SSH_COMMAND, core.sshCommand, GIT_SSH,
// plain ssh). GIT_SSH holds a program path, not a command line, and git
// ignores it once GIT_SSH_COMMAND is set, so it is quoted into one. The
// flag depends on the client: OpenSSH takes -o BatchMode=yes (first, as
// OpenSSH keeps the first value it sees), plink takes -batch, and git
// already passes -batch to TortoisePlink. A client of unknown kind gets
// nothing: an option it does not understand would break every sync, and
// git runs without a terminal and with SSH_ASKPASS_REQUIRE=never anyway.
func (r *Runner) resolveSSH(ctx context.Context) (string, bool) {
	cfg := r.sshConfig(ctx)
	command, isCmdline := strings.TrimSpace(os.Getenv("GIT_SSH_COMMAND")), true
	if command == "" {
		command = cfg["core.sshcommand"]
	}
	if command == "" {
		if p := os.Getenv("GIT_SSH"); p != "" {
			command, isCmdline = p, false
		}
	}
	if command == "" {
		command = "ssh"
	}
	program, rest := command, ""
	if isCmdline {
		program, rest = splitProgram(command)
	} else {
		command = shellQuote(command)
	}

	variant := strings.ToLower(os.Getenv("GIT_SSH_VARIANT"))
	if variant == "" {
		variant = strings.ToLower(cfg["ssh.variant"])
	}
	if variant == "putty" { // git's other name for the plink variant
		variant = "plink"
	}
	if variant == "" || variant == "auto" {
		base := strings.Trim(program, `"'`)
		if i := strings.LastIndexAny(base, `/\`); i >= 0 {
			base = base[i+1:]
		}
		variant = strings.TrimSuffix(strings.ToLower(base), ".exe")
	}
	switch variant {
	case "ssh":
		if isCmdline {
			return program + " -o BatchMode=yes" + rest, true
		}
		return command + " -o BatchMode=yes", true
	case "plink":
		if isCmdline {
			return program + " -batch" + rest, true
		}
		return command + " -batch", true
	case "tortoiseplink":
		return command, true
	}
	return "", false
}

// sshConfig reads core.sshCommand and ssh.variant (keys come back
// lowercased). A failure reads as "not configured": the lookup has its own
// short limit, within ctx, so it never blocks the command it is resolved
// for nor outlives its caller (Ctrl+C on quit stops it at once).
func (r *Runner) sshConfig(parent context.Context) map[string]string {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", r.dir, "config", "-z", "--get-regexp", `^(core\.sshcommand|ssh\.variant)$`)
	cmd.Stdin = nil
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.WaitDelay = 2 * time.Second
	cfg := map[string]string{}
	if run(cmd, nil) != nil {
		return cfg
	}
	for _, entry := range bytes.Split(out.Bytes(), []byte{0}) {
		if k, v, ok := strings.Cut(string(entry), "\n"); ok {
			cfg[k] = strings.TrimSpace(v) // the last value wins, as in git
		}
	}
	return cfg
}

// splitProgram splits a shell command line into its program (kept exactly
// as written, quotes included) and the rest, which starts with a space or
// is empty.
func splitProgram(command string) (string, string) {
	if q := command[0]; q == '"' || q == '\'' {
		if end := strings.IndexByte(command[1:], q); end >= 0 {
			return command[:end+2], command[end+2:]
		}
	}
	if i := strings.IndexAny(command, " \t"); i >= 0 {
		return command[:i], command[i:]
	}
	return command, ""
}

// shellQuote quotes s for the POSIX shell git runs GIT_SSH_COMMAND with.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Commit records the staged changes with message, without hooks or
// signing.
func (r *Runner) Commit(ctx context.Context, message string) ([]byte, error) {
	return r.Run(ctx, "commit", "--no-verify", "--quiet", "-m", message)
}

// Status lists the changed and untracked paths. It reads git's
// NUL-separated output, so names with accents, spaces or quotes come back
// exactly as they are on disk.
func (r *Runner) Status(ctx context.Context) ([]Change, error) {
	// --no-optional-locks: status never takes the index lock (see
	// indexLockCommands).
	out, err := r.Run(ctx, "--no-optional-locks", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	return parseStatusZ(out), nil
}

func parseStatusZ(out []byte) []Change {
	var changes []Change
	fields := bytes.Split(out, []byte{0})
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if len(f) < 4 {
			continue
		}
		code := string(f[:2])
		c := Change{Code: code, Path: string(f[3:])}
		if (code[0] == 'R' || code[0] == 'C') && i+1 < len(fields) {
			i++ // a rename or copy is followed by its original path
			c.From = string(fields[i])
		}
		changes = append(changes, c)
	}
	return changes
}

func firstWord(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" || args[i] == "-C" {
			i++
			continue
		}
		if args[i] == "--no-optional-locks" {
			continue
		}
		return args[i]
	}
	return ""
}
