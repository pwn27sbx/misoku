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
	"time"
)

// MinVersion is the oldest git Misoku syncs with: merge-tree --write-tree,
// which computes a merge without touching the working tree, needs 2.38.
const MinVersion = "2.38"

// Change is one entry of `git status`: the two-letter XY code and the path,
// exactly as stored on disk (no quoting or escaping).
type Change struct{ Code, Path string }

// Runner runs git inside one repository.
type Runner struct {
	dir     string
	timeout time.Duration
}

// New returns a Runner for the repository at dir. Each command gets at most
// timeout to finish.
func New(dir string, timeout time.Duration) *Runner { return &Runner{dir: dir, timeout: timeout} }

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
// A failure carries git's own error text.
func (r *Runner) Run(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
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
	cmd.Env = r.env(ctx)
	cmd.Stdin = nil // reads from the null device
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	detach(cmd)
	// After a kill, give pipes held by orphaned children this long to close
	// before Wait gives up on them.
	cmd.WaitDelay = 2 * time.Second

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return stdout.Bytes(), fmt.Errorf("git %s: no answer after %s", firstWord(args), r.timeout)
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return stdout.Bytes(), errors.New(msg)
	}
	return stdout.Bytes(), nil
}

// env is the environment git runs with: the user's, with every prompt off.
func (r *Runner) env(ctx context.Context) []string {
	env := os.Environ()
	set := func(k, v string) {
		for i, kv := range env {
			if strings.HasPrefix(kv, k+"=") {
				env[i] = k + "=" + v
				return
			}
		}
		env = append(env, k+"="+v)
	}
	set("GIT_TERMINAL_PROMPT", "0")     // no username/password questions
	set("GCM_INTERACTIVE", "never")     // Git Credential Manager: no windows
	set("SSH_ASKPASS_REQUIRE", "never") // ssh: never pop up an askpass helper
	set("LC_ALL", "C")                  // stable messages to parse
	set("LANGUAGE", "C")
	set("GIT_SSH_COMMAND", r.sshCommand(ctx)+" -o BatchMode=yes")
	return env
}

// sshCommand is the ssh the user already configured (GIT_SSH_COMMAND, then
// core.sshCommand), so BatchMode is added to it instead of replacing it.
func (r *Runner) sshCommand(ctx context.Context) string {
	if c := strings.TrimSpace(os.Getenv("GIT_SSH_COMMAND")); c != "" {
		return c
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", r.dir, "config", "--get", "core.sshCommand")
	cmd.Stdin = nil
	detach(cmd)
	if out, err := cmd.Output(); err == nil {
		if c := strings.TrimSpace(string(out)); c != "" {
			return c
		}
	}
	return "ssh"
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
	out, err := r.Run(ctx, "status", "--porcelain=v1", "-z", "--untracked-files=all")
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
		changes = append(changes, Change{Code: code, Path: string(f[3:])})
		if code[0] == 'R' || code[0] == 'C' {
			i++ // a rename or copy is followed by its original path
		}
	}
	return changes
}

func firstWord(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" || args[i] == "-C" {
			i++
			continue
		}
		return args[i]
	}
	return ""
}
