package gitsync

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// accessProblems are what git, ssh or a credential helper print when
// origin was reached but would not let Misoku in (S8), and what the user
// can do about it. Sync never asks for a password or a host key: it runs
// without a terminal, so these fail at once instead of hanging.
//
// initHint, when set, replaces hint inside `misoku sync init`, where the
// user already runs init in a terminal.
var accessProblems = []struct {
	signs    []string // any of them, lowercased
	hint     string
	initHint string
}{
	{[]string{"remote host identification has changed", "has changed and you have requested strict checking"},
		"origin's SSH host key changed, which can mean an attack or a reinstalled server: check the key with ssh in a terminal before syncing again", ""},
	{[]string{"host key verification failed", "host key is known for", "no ed25519 host key is known", "no rsa host key is known", "no ecdsa host key is known"},
		"origin's SSH host key is not known yet: run `misoku sync init <url>` in a terminal once to accept it",
		"origin's SSH host key was not accepted: check that the address is right and accept the key when ssh asks, then run init again"},
	{[]string{"permission denied (publickey", "permission denied, please try again"},
		"origin refused your SSH key: add the key to your account (and to ssh-agent), or run `misoku sync init <url>` in a terminal",
		"origin refused your SSH key: add the key to your account (and to ssh-agent), then run init again"},
	{[]string{"could not read username", "could not read password", "authentication failed", "terminal prompts disabled",
		"invalid username or password", "http basic: access denied", "returned error: 401", "returned error: 403", "repository not found"},
		"could not sign in to origin: run `gh auth setup-git` (GitHub over HTTPS) or set up your git credentials, then `misoku sync init <url>` in a terminal if it still fails",
		"could not sign in to origin: check the URL, then run `gh auth setup-git` (GitHub over HTTPS) or set up your git credentials, then run init again"},
}

// remoteFailure is the outcome of a talk with origin that failed (what
// names the step): SyncDenied with the fix when origin would not let
// Misoku in (S8), SyncOffline when it could not be reached (S7). r says
// whose words the fix takes: init's for an interactive Runner.
func remoteFailure(r *Runner, what string, err error) (SyncOutcome, error) {
	text := strings.ToLower(err.Error())
	for _, p := range accessProblems {
		for _, sign := range p.signs {
			if strings.Contains(text, sign) {
				hint := p.hint
				if r.interactive && p.initHint != "" {
					hint = p.initHint
				}
				return SyncDenied, fmt.Errorf("%s (%s: %w)", hint, what, err)
			}
		}
	}
	return SyncOffline, fmt.Errorf("%s: %w", what, err)
}

// pushRefusal reads `git push --porcelain`. rejected is true when origin
// turned the push down because it has commits the vault lacks (another
// device pushed first: the next sync joins them). refused is the reason
// when origin turned it down for its own rules (a hook, a protected
// branch, a file too large), which no retry fixes.
func pushRefusal(porcelain []byte) (rejected bool, refused string) {
	for _, line := range strings.Split(string(porcelain), "\n") {
		if !strings.HasPrefix(line, "!\t") {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		summary := ""
		if len(parts) == 3 {
			summary = strings.TrimSpace(parts[2])
		}
		if strings.HasPrefix(summary, "[rejected]") {
			return true, ""
		}
		if summary == "" {
			summary = "no reason given"
		}
		return false, summary
	}
	return false, ""
}

// urlRe finds URLs in a message, for redactText.
var urlRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^\s'"<>]+`)

// RedactURL returns raw without what could be a secret: the user and
// password of a URL (a token is often the user part alone) and its query.
// An ssh URL keeps its user name, which is no secret (git@), but loses any
// password. scp-like addresses (git@host:path) and paths are returned as
// they are.
func RedactURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		if i := strings.Index(raw, "://"); i >= 0 {
			// Unparseable: drop everything up to an @ in the authority.
			rest := raw[i+3:]
			if at := strings.LastIndex(strings.SplitN(rest, "/", 2)[0], "@"); at >= 0 {
				return raw[:i+3] + rest[at+1:]
			}
		}
		return raw
	}
	if u.User != nil {
		if u.Scheme == "ssh" || u.Scheme == "git+ssh" {
			u.User = url.User(u.User.Username())
		} else {
			u.User = nil
		}
	}
	u.RawQuery, u.Fragment = "", ""
	return u.String()
}

// redactText runs RedactURL on every URL in s, so a message git or a
// helper printed never shows credentials.
func redactText(s string) string {
	return urlRe.ReplaceAllStringFunc(s, func(m string) string {
		// Keep punctuation that ends a sentence out of the URL.
		trail := ""
		for len(m) > 0 && strings.ContainsRune(".,;:)", rune(m[len(m)-1])) {
			trail = m[len(m)-1:] + trail
			m = m[:len(m)-1]
		}
		return RedactURL(m) + trail
	})
}

// redactedError is an error whose text went through redactText; Unwrap
// still reaches the original, so errors.Is keeps working.
type redactedError struct {
	msg   string
	cause error
}

func (e *redactedError) Error() string { return e.msg }
func (e *redactedError) Unwrap() error { return e.cause }

// redact returns err with any credentials in its text removed.
func redact(err error) error {
	if err == nil {
		return nil
	}
	msg := redactText(err.Error())
	if msg == err.Error() {
		return err
	}
	return &redactedError{msg: msg, cause: err}
}
