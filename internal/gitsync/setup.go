package gitsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
)

// The setup assistant (`misoku sync setup` and `:sync setup`) connects a
// vault to a private GitHub repository for someone who does not know git.
// It signs in, creates or lists repositories through gh, GitHub's own
// command-line tool, and then runs the same InitVault as
// `misoku sync init`. gh keeps the access to GitHub in the system's
// keychain: Misoku never sees, stores, prints or passes a token.
//
// Setup is one engine with two front ends: the command line asks in the
// terminal, the app in dialogs. Each call does its work (git, gh, the
// init) and returns the next SetupStep, which says what to ask; the front
// end asks it and passes the answer to Next. A call may run external
// programs, so the app makes it from a tea.Cmd, never from Update.

// SetupAsk is what the assistant needs from the user next.
type SetupAsk int

const (
	// SetupFinished: nothing more to ask; Outcome says how it ended and
	// Lines what happened.
	SetupFinished SetupAsk = iota
	// AskStart: Lines explain what will happen; start, or cancel.
	AskStart
	// AskIdentity: git has no name or email; it comes after the sign-in.
	// Name and Email are the values git has, with a missing one filled in
	// from the GitHub account when gh can tell (FromGitHub), else empty;
	// the answer brings both, and OK only once the user confirmed Confirm,
	// the question about saving them in git's global settings.
	AskIdentity
	// AskSignIn: gh is not signed in to GitHub. Lines explain it (and,
	// on a computer with no browser to open, where to sign in instead); on
	// yes the front end runs SignInCommand in the foreground of the
	// terminal, then calls Next with OK.
	AskSignIn
	// AskRepo: pick one of Repos (Choice), name one (Text), or create a
	// new private repository (Choice == len(Repos)).
	AskRepo
	// AskRepoName: the name of the new private repository (Text), Default
	// when empty.
	AskRepoName
	// AskOwnRepo: the folder is inside another git repository that does
	// not track anything in it. Lines explain it and Confirm asks; on yes
	// the assistant runs `git init` in the folder and goes on.
	AskOwnRepo
	// AskUnrelated: the picked repository (Repo) holds notes that do not
	// share the folder's history. Lines list its top level and Confirm
	// asks whether to mix the two; only OK uses it, so a front end's
	// default answer is no.
	AskUnrelated
)

// SetupOutcome is how the assistant ended.
type SetupOutcome int

const (
	// SetupAsking: it has not ended; the step asks something.
	SetupAsking SetupOutcome = iota
	// SetupDone: the vault is set up and listed in sync.vaults.
	SetupDone
	// SetupAlready: the vault was set up before; nothing changed.
	SetupAlready
	// SetupCancelled: the user stopped it; nothing was changed.
	SetupCancelled
	// SetupFailed: something must be fixed first; Lines say what.
	SetupFailed
	// SetupStartNotes: with SetupOptions.Bring, the user has no notes on
	// another computer; the front end starts new notes instead. Lines say
	// what the run changed before (its identity, say), if anything.
	SetupStartNotes
)

// SetupStep is what the assistant says and asks next.
type SetupStep struct {
	Ask     SetupAsk
	Outcome SetupOutcome
	// Title names the step, for a dialog.
	Title string
	// Lines is what to tell the user (on SetupFinished, how it ended).
	Lines []string
	// Notes are the checks passed since the previous step ("✓ git 2.47"),
	// to show as progress.
	Notes []string
	// Problem says why the previous answer was not taken; the same
	// question is asked again.
	Problem string
	// Name and Email: AskIdentity's current values.
	Name, Email string
	// FromGitHub: AskIdentity filled in a missing Name or Email from the
	// signed-in GitHub account; the user confirms or edits them.
	FromGitHub bool
	// Confirm: the yes-or-no question of AskIdentity (before saving) and
	// of AskOwnRepo (before `git init`).
	Confirm string
	// Repos: AskRepo's choices, private ones first.
	Repos []Repo
	// NewChoice: the label of AskRepo's extra choice (Choice ==
	// len(Repos)): a new private repository, or, with SetupOptions.Bring,
	// starting new notes.
	NewChoice string
	// NewFirst: list NewChoice before the repositories, where a long list
	// cannot hide it (a new private repository); otherwise after them.
	NewFirst bool
	// Default: AskRepoName's suggested name.
	Default string
	// Repo: the repository the vault syncs with, once known.
	Repo string
}

// SetupAnswer is the user's answer to a SetupStep.
type SetupAnswer struct {
	// OK is false when the user cancelled.
	OK bool
	// Name and Email answer AskIdentity.
	Name, Email string
	// Choice answers AskRepo: an index in Repos, or len(Repos) to create
	// a new repository. Ignored when Text is set.
	Choice int
	// Text answers AskRepoName (the name) and AskRepo (a repository
	// typed by name, such as "notes" or "ana/notes").
	Text string
}

// Repo is one of the user's GitHub repositories.
type Repo struct {
	// Name is "owner/name".
	Name string `json:"nameWithOwner"`
	// Visibility is PRIVATE, PUBLIC or INTERNAL.
	Visibility string `json:"visibility"`
	// URL is the repository's page, such as https://github.com/ana/notes.
	URL string `json:"url"`
}

// Private reports whether only the owner (and who they invite) can read
// the repository. INTERNAL repositories are readable by a whole
// organization, so they are not.
func (r Repo) Private() bool { return strings.EqualFold(r.Visibility, "private") }

// cloneURL is the HTTPS address git syncs with, stored as origin. Any
// user name or password in the listed URL is dropped: the access is gh's,
// and it never goes into .git/config.
func (r Repo) cloneURL() string {
	u := strings.TrimSpace(r.URL)
	if parsed, err := url.Parse(u); err == nil && parsed.Host != "" {
		parsed.User, parsed.RawQuery, parsed.Fragment = nil, "", ""
		u = parsed.String()
	}
	u = strings.TrimSuffix(u, "/")
	if !strings.HasSuffix(u, ".git") {
		u += ".git"
	}
	return u
}

// SetupOptions are the settings of NewSetup.
type SetupOptions struct {
	// Dir is the vault's folder, absolute.
	Dir string
	// ConfigPath is Misoku's config file, where the vault is listed.
	ConfigPath string
	// GH is the gh program; empty means "gh" on PATH.
	GH string
	// Timeout bounds each gh and git command that runs in the background
	// (not the sign-in, which the user drives).
	Timeout time.Duration
	// InitTimeout bounds each git command of the init.
	InitTimeout time.Duration
	// Interactive lets the init's network commands ask the user (the
	// command line, in the foreground). Inside the app they never ask: gh
	// already gave git the access (gh auth setup-git).
	Interactive bool
	// OS picks the install hint for gh; empty means runtime.GOOS.
	OS string
	// Now gives the local time used in conflict copy names; nil means
	// time.Now.
	Now func() time.Time
	// Bring is the welcome's "Bring my notes from another computer": the
	// notes come from a repository the user already has, so AskRepo offers
	// no new repository; its last choice ends the run with
	// SetupStartNotes instead.
	Bring bool
}

// Setup is one run of the assistant. Its calls are made one at a time.
type Setup struct {
	opts   SetupOptions
	at     SetupAsk
	vault  config.Vault
	listed bool
	repos  []Repo
	// changed lists what this run changed so far, for the message when it
	// stops before the end: it never says nothing changed when something
	// did.
	changed []string
	// madeRepo: this run created the folder's repository (AskOwnRepo);
	// madeAt is its line in changed. A cancel removes it while it has no
	// commits.
	madeRepo bool
	madeAt   int
	// newDir: the folder does not exist yet and is inside another
	// repository; AskOwnRepo's yes creates it before `git init`.
	newDir bool
	// madeDir: this run created the folder; dirAt is its line in changed.
	// A cancel or a failure removes it once it is empty again.
	madeDir bool
	dirAt   int
	// waitedGit: git did not run while Apple's installer for it did, so
	// Begin could not look at the folder; the next git check does.
	waitedGit bool
	// gitReady: git uses gh's access already (setupGit ran).
	gitReady bool
	// pending: the repository AskUnrelated asks about.
	pending Repo
}

// appleGitProblem is AskStart's Problem while Apple's installer for git
// runs.
const appleGitProblem = "Apple is installing git; press Enter when it finishes."

// NewSetup returns the assistant for opts.Dir. Nothing runs until Begin.
func NewSetup(opts SetupOptions) *Setup {
	if opts.GH == "" {
		opts.GH = "gh"
	}
	if opts.OS == "" {
		opts.OS = runtime.GOOS
	}
	if opts.Timeout <= 0 {
		opts.Timeout = time.Minute
	}
	if opts.InitTimeout <= 0 {
		opts.InitTimeout = opts.Timeout
	}
	return &Setup{opts: opts}
}

// SameFolder reports whether a and b are the same folder, as the operating
// system sees it (see Listed): symlinks, letter case and macOS's /var ->
// /private/var do not make two folders of one.
func SameFolder(a, b string) bool { return sameDir(a, b) }

// githubHost is the only host the assistant signs in to.
const githubHost = "github.com"

// SignInCommand is `gh auth login` in the browser, for the front end to
// run in the foreground of the user's terminal: gh shows a one-time code
// and opens the browser, and keeps the access itself. It has no time
// limit: the user drives it, and Ctrl+C stops it.
//
// gh asks nothing of its own: the flags answer its questions about the
// host and the protocol, and the environment answers "Authenticate Git
// with your GitHub credentials?" (see withGHHelper).
//
// Its stdout is the terminal (os.Stdout); a caller that captures the
// output sets its own.
func (s *Setup) SignInCommand() *exec.Cmd {
	cmd := exec.Command(s.opts.GH, "auth", "login", "--web", "--git-protocol", "https", "--hostname", githubHost)
	cmd.Env = withGHHelper(os.Environ())
	// gh writes to the terminal itself. Left unset, tea.ExecProcess would
	// give it the program output, which is not an *os.File when frames are
	// wrapped for synchronized output, and gh would get a pipe, not a TTY.
	cmd.Stdout = os.Stdout
	return cmd
}

// ghHelper is gh's git credential helper, as `gh auth setup-git` writes it.
const ghHelper = "!gh auth git-credential"

// withGHHelper adds to env, for git alone and for this one run, gh's
// credential helper for github.com. Seeing it, `gh auth login` takes git
// as set up and does not ask whether to set it up: setup does that itself
// later, with `gh auth setup-git`, once the user picked a repository. git
// reads the GIT_CONFIG_COUNT pairs after its config files; settings the
// user's environment already carries this way are kept.
func withGHHelper(env []string) []string {
	n := 0
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "GIT_CONFIG_COUNT="); ok {
			if c, err := strconv.Atoi(v); err == nil && c > 0 {
				n = c
			}
		}
	}
	setEnv(&env, "GIT_CONFIG_COUNT", strconv.Itoa(n+1))
	setEnv(&env, fmt.Sprintf("GIT_CONFIG_KEY_%d", n), "credential.https://"+githubHost+".helper")
	setEnv(&env, fmt.Sprintf("GIT_CONFIG_VALUE_%d", n), ghHelper)
	return env
}

// Begin reads the config and the folder, without changing anything. A
// vault already set up, or a folder setup must not touch (one inside
// another repository; or, unless listed, the home folder, the root of a
// disk or a system folder), ends here; otherwise it explains what will
// happen (AskStart).
func (s *Setup) Begin(ctx context.Context) SetupStep {
	cfg, err := config.Load(s.opts.ConfigPath)
	if err != nil {
		return failed(fmt.Sprintf("The config file %s could not be read (%v); fix it, then run setup again.", s.opts.ConfigPath, err))
	}
	vaults, _ := cfg.Sync.Parse()
	s.vault, s.listed = Listed(vaults, s.opts.Dir)
	if !s.listed {
		if vs, _ := (config.Sync{Vaults: map[string]config.SyncVault{s.opts.Dir: {}}}).Parse(); len(vs) == 1 {
			s.vault = vs[0]
		}
	}
	// A vault already listed is left as it was: v0.7.0 let a home folder
	// be set up, and its sync goes on. Only a new one is refused.
	if !s.listed {
		if err := RefuseSyncFolder(s.opts.Dir, ""); err != nil {
			return failed(err.Error())
		}
	}
	if done, step := s.inspect(ctx); done {
		return step
	}
	return s.startStep()
}

// startStep explains what will happen (AskStart).
func (s *Setup) startStep() SetupStep {
	s.at = AskStart
	if s.opts.Bring {
		return SetupStep{Ask: AskStart, Title: "Bring your notes", Lines: []string{
			"Your notes come from the private GitHub repository your other computer syncs with, and stay in sync from then on.",
			ghAccessLine,
		}}
	}
	return SetupStep{Ask: AskStart, Title: "Sync this vault", Lines: []string{
		"Your notes will be kept in a private GitHub repository and reach your other computers on their own.",
		ghAccessLine,
		"In the repository your notes are not encrypted, so it stays private.",
	}}
}

// ghAccessLine says who keeps the access to GitHub.
const ghAccessLine = "You need a GitHub account. Misoku signs in with gh, GitHub's command-line tool, which keeps the access: Misoku never sees or stores it."

// Labels of AskRepo's last choice (SetupStep.NewChoice).
const (
	createChoice = "Create a new private repository"
	freshChoice  = "I don't have notes on another computer → Start my notes"
)

// inspect looks at the folder: when it already has an origin, or is one
// setup must not touch, the run ends with done and the step saying why.
// A folder inside another repository that does not track it is offered
// a repository of its own (AskOwnRepo), also with done.
// When git does not run, it says nothing: the git check after AskStart
// tells the user how to fix it.
func (s *Setup) inspect(ctx context.Context) (bool, SetupStep) {
	dir := s.opts.Dir
	info, err := os.Stat(dir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// A new computer's vault: InitVault creates it, if its parent is
		// there.
		if p, err := os.Stat(filepath.Dir(dir)); err != nil || !p.IsDir() {
			return true, failed(fmt.Sprintf("%s does not exist, nor does its folder %s; create it first, or choose another folder.", dir, filepath.Dir(dir)))
		}
		return s.inspectNew(ctx)
	case err != nil:
		return true, failed(err.Error())
	case !info.IsDir():
		return true, failed(dir + " is not a folder.")
	}
	r := New(dir, s.opts.Timeout)
	if _, ok, err := r.Version(ctx); err != nil || !ok {
		return false, SetupStep{}
	}
	isRepo, origin, err := folderState(ctx, r)
	var nested *nestedError
	switch {
	case errors.As(err, &nested) && !nested.tracks():
		s.at = AskOwnRepo
		return true, ownRepoStep(nested)
	case err != nil:
		return true, failed(refusal(err))
	}
	if !isRepo || origin == "" {
		return false, SetupStep{}
	}
	shown := RedactURL(origin)
	if s.listed {
		lines := []string{fmt.Sprintf("This vault is already set up: it syncs with %s; nothing changed.", shown)}
		if st := Detect(ctx, r); st.Status != Ready {
			lines = append(lines, "Sync is paused, though: "+st.Message+".")
		}
		lines = append(lines, "`:sync status` in Misoku shows how it is going.")
		return true, SetupStep{Outcome: SetupAlready, Title: "Sync setup", Lines: lines, Repo: shown}
	}
	return true, SetupStep{Outcome: SetupAlready, Title: "Sync setup", Repo: shown, Lines: []string{
		fmt.Sprintf("This folder is already set up with the git repository %s, though Misoku does not sync it yet; nothing changed.", shown),
		fmt.Sprintf("To sync it with that repository, run `misoku sync init %s`.", shown),
	}}
}

// ownRepoStep offers the folder inside nested's repository a repository
// of its own (AskOwnRepo): what is wrong, said once, then the question.
func ownRepoStep(nested *nestedError) SetupStep {
	return SetupStep{Ask: AskOwnRepo, Title: "A repository for your notes", Confirm: ownRepoQuestion, Lines: []string{
		fmt.Sprintf("This folder is inside the git repository at %s, which does not track it: syncing it as it is would commit that whole repository. A separate repository leaves that one as it is.", nested.top),
	}}
}

// inspectNew looks at a folder that does not exist yet as inspect looks
// at an existing one: inside another repository it is offered a
// repository of its own (AskOwnRepo, which creates the folder on yes), or
// refused when that repository tracks files there. git can only be asked
// about a folder that exists, so it is made empty for that moment and
// removed again before anything else happens.
func (s *Setup) inspectNew(ctx context.Context) (bool, SetupStep) {
	dir := s.opts.Dir
	if _, ok, err := New(filepath.Dir(dir), s.opts.Timeout).Version(ctx); err != nil || !ok {
		return false, SetupStep{} // the git check after AskStart says how to fix git
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		return true, failed(fmt.Sprintf("Setup could not look at %s (%v). Setup changed nothing.", dir, err))
	}
	_, _, err := folderState(ctx, New(dir, s.opts.Timeout))
	if rerr := os.Remove(dir); rerr != nil {
		return true, failed(fmt.Sprintf("Setup made the empty folder %s to look at it and could not remove it again (%v); delete it, then run setup again.", dir, rerr))
	}
	var nested *nestedError
	switch {
	case errors.As(err, &nested) && !nested.tracks():
		s.at, s.newDir = AskOwnRepo, true
		return true, ownRepoStep(nested)
	case errors.As(err, &nested):
		return true, failed(refusal(err))
	}
	// Not inside another repository (or git could not tell): InitVault
	// creates the folder and checks it again.
	return false, SetupStep{}
}

// ownRepoQuestion is AskOwnRepo's question.
const ownRepoQuestion = "Create a separate repository for your notes here?"

// refusal is folderState's refusal in setup's words.
func refusal(err error) string {
	var nested *nestedError
	if errors.As(err, &nested) {
		return nested.message("setup") + "."
	}
	return redactText(err.Error()) + ". Setup changed nothing."
}

// Next takes the answer to the step the previous call returned and goes
// on to the next one. A cancel removes the repository this run created,
// if nothing was committed in it yet.
func (s *Setup) Next(ctx context.Context, ans SetupAnswer) SetupStep {
	if !ans.OK {
		s.at = SetupFinished
		s.undo(ctx)
		return SetupStep{Outcome: SetupCancelled, Title: "Sync setup", Lines: []string{s.StopText("Setup cancelled")}}
	}
	switch s.at {
	case AskOwnRepo:
		return s.ownRepo(ctx)
	case AskStart:
		return s.checkGit(ctx)
	case AskIdentity:
		return s.saveIdentity(ctx, ans)
	case AskSignIn:
		return s.signedIn(ctx)
	case AskRepo:
		return s.pick(ctx, ans)
	case AskRepoName:
		return s.create(ctx, ans)
	case AskUnrelated:
		return s.use(ctx, s.pending)
	}
	return s.fail(ctx, "Setup has already ended; run it again.")
}

// MadeRepo reports whether this run created the folder's git repository,
// or the folder, and still has it: a cancel then needs Next, which may
// remove it, rather than StopText alone.
func (s *Setup) MadeRepo() bool { return s.madeRepo || s.madeDir }

// ownRepo runs `git init` in the folder (AskOwnRepo's yes) and goes on
// to AskStart. The folder is looked at again first: the outer repository
// may have started tracking it, or the user may have made a repository
// there, while the question waited.
func (s *Setup) ownRepo(ctx context.Context) SetupStep {
	if s.newDir {
		s.newDir = false
		made, err := makeFolder(s.opts.Dir)
		if err != nil {
			return s.fail(ctx, err.Error()+".")
		}
		s.noteFolder(made)
	}
	r := New(s.opts.Dir, s.opts.Timeout)
	_, _, err := folderState(ctx, r)
	var nested *nestedError
	switch {
	case errors.As(err, &nested) && !nested.tracks():
		if _, err := r.Run(ctx, "init", "--quiet", "--initial-branch=main"); err != nil {
			return s.fail(ctx, fmt.Sprintf("git could not create a repository in this folder (%v).", redact(err)))
		}
		s.madeRepo, s.madeAt = true, len(s.changed)
		s.changed = append(s.changed, "A git repository of its own was created in this folder")
		step := s.startStep()
		step.Notes = []string{"✓ created a git repository of its own for your notes"}
		return step
	case err != nil:
		return s.fail(ctx, refusal(err))
	}
	// A repository of the folder's own already: as Begin would see it.
	if done, step := s.inspect(ctx); done {
		s.at = SetupFinished
		return step
	}
	return s.startStep()
}

// noteFolder records that this run created the folder, when made.
func (s *Setup) noteFolder(made bool) {
	if made && !s.madeDir {
		s.madeDir, s.dirAt = true, len(s.changed)
		s.changed = append(s.changed, "The folder "+s.opts.Dir+" was created")
	}
}

// noteInit records the folder and the repository InitVault created, for
// a run that stops after it.
func (s *Setup) noteInit(res InitResult) {
	s.noteFolder(res.CreatedFolder)
	if res.CreatedRepo && !s.madeRepo {
		s.madeRepo, s.madeAt = true, len(s.changed)
		s.changed = append(s.changed, "A git repository was created in "+s.opts.Dir)
	}
}

// undo removes what this run created and nothing uses yet: its
// repository while it has no commits, then its folder once that is empty
// again. What stays is said in changed. It runs even when ctx was
// cancelled (the step stopped by Ctrl+C or Esc): its git keeps only its
// own time limit.
func (s *Setup) undo(ctx context.Context) {
	ctx = context.WithoutCancel(ctx)
	s.undoRepo(ctx)
	if !s.madeDir {
		return
	}
	s.madeDir = false
	if err := os.Remove(s.opts.Dir); err != nil {
		if entries, rerr := os.ReadDir(s.opts.Dir); rerr == nil && len(entries) > 0 {
			s.changed[s.dirAt] += "; it was left in place, as it is not empty"
			return
		}
		s.changed[s.dirAt] += fmt.Sprintf("; it could not be removed again (%v)", err)
		return
	}
	s.changed[s.dirAt] = "The folder " + s.opts.Dir + " it created was removed"
}

// undoRepo removes the repository this run created, when it still has
// no commits: nothing in it is the user's yet. One with commits is left
// in place, and the stop message says so.
func (s *Setup) undoRepo(ctx context.Context) {
	if !s.madeRepo {
		return
	}
	s.madeRepo = false
	head, err := resolve(ctx, New(s.opts.Dir, s.opts.Timeout), "HEAD")
	gitDir := filepath.Join(s.opts.Dir, ".git")
	switch {
	case err != nil:
		s.changed[s.madeAt] += fmt.Sprintf("; it was left in place, as git could not tell whether it has commits (%v)", redact(err))
	case head != "":
		s.changed[s.madeAt] += "; it already has commits, so it was left in place"
	default:
		// Only the folder git init made: a .git that is a file or a link
		// now is not ours to delete.
		if info, err := os.Lstat(gitDir); err != nil || !info.IsDir() {
			s.changed[s.madeAt] += "; it was left in place, as " + gitDir + " is no longer the folder it created"
			return
		}
		if err := os.RemoveAll(gitDir); err != nil {
			s.changed[s.madeAt] += fmt.Sprintf("; it could not be removed again (%v): delete %s by hand", err, gitDir)
			return
		}
		s.changed[s.madeAt] = "The git repository it created was removed"
	}
}

// gitRunner runs git in the vault, or in its parent folder while the
// vault does not exist yet.
func (s *Setup) gitRunner() *Runner {
	dir := s.opts.Dir
	if _, err := os.Stat(dir); err != nil {
		dir = filepath.Dir(dir)
	}
	return New(dir, s.opts.Timeout)
}

// checkGit checks git's version and that gh is installed (step 2), then
// goes on to the sign-in. gh comes before the identity, so a missing gh
// never leaves an identity saved for nothing.
func (s *Setup) checkGit(ctx context.Context) SetupStep {
	r := s.gitRunner()
	version, ok, err := r.Version(ctx)
	if err != nil && s.opts.OS == "darwin" && appleInstallingGit(ctx) {
		// git comes with Apple's Command Line Tools, which the installer
		// started: Start (Enter) checks again once they are in.
		s.waitedGit = true
		step := s.startStep()
		step.Problem = appleGitProblem
		return step
	}
	if err != nil {
		return s.fail(ctx, fmt.Sprintf("git did not run (%v). Install git %s or newer, then run setup again.", err, MinVersion))
	}
	if !ok {
		return s.fail(ctx, fmt.Sprintf("git %s is too old: sync needs git %s or newer. Update git, then run setup again.", version, MinVersion))
	}
	notes := []string{"✓ git " + version}
	if s.waitedGit {
		// Begin could not look at the folder without git: it does now.
		s.waitedGit = false
		if done, step := s.inspect(ctx); done {
			step.Notes = append(notes, step.Notes...)
			return step
		}
	}
	path, err := lookGH(s.opts.GH)
	if err != nil {
		return s.fail(ctx,
			"gh, GitHub's command-line tool, is not installed: setup uses it to sign in to GitHub.",
			GHInstallHint(s.opts.OS),
		)
	}
	s.opts.GH = path
	if out, err := s.gh(ctx, "--version"); err == nil {
		if m := ghVersionRe.FindStringSubmatch(string(out)); m != nil {
			notes = append(notes, "✓ gh "+m[1])
		}
	}
	step := s.checkSignIn(ctx)
	step.Notes = append(notes, step.Notes...)
	return step
}

// identityStep asks for git's name and email; fromGitHub says a missing
// one was filled in from the GitHub account.
func identityStep(name, email, problem string, notes []string, fromGitHub bool) SetupStep {
	lines := []string{"git does not know your name and email yet: it writes them in every change it records (your notes' history)."}
	if fromGitHub {
		lines = append(lines, "Misoku filled them in from your GitHub account (the email is GitHub's private address for you); change them if you like.")
	}
	return SetupStep{Ask: AskIdentity, Title: "Your name for git", Notes: notes, Name: name, Email: email, Problem: problem,
		FromGitHub: fromGitHub, Lines: lines,
		Confirm: "Save them in git's global settings (git config --global), for every repository on this computer?"}
}

// askIdentity checks git's identity once signed in (step 4): when git
// lacks the name or the email, it asks for them, filling in what is
// missing from the GitHub account; otherwise it lists the repositories.
func (s *Setup) askIdentity(ctx context.Context) SetupStep {
	name, email, err := identity(ctx, s.gitRunner())
	if err != nil {
		return s.fail(ctx, redactText(err.Error()))
	}
	if name != "" && email != "" {
		return s.listRepos(ctx)
	}
	s.at = AskIdentity
	ghName, ghEmail, ok := s.githubIdentity(ctx)
	if !ok {
		return identityStep(name, email, "", nil, false)
	}
	if name == "" {
		name = ghName
	}
	if email == "" {
		email = ghEmail
	}
	return identityStep(name, email, "", nil, true)
}

// githubIdentity suggests a git name and email from the signed-in GitHub
// account (`gh api user`): the profile's name, or the login when it has
// none, and GitHub's private address <id>+<login>@users.noreply.github.com,
// which keeps the user's own email out of the history. ok is false when gh
// cannot tell; that is not an error: the user then types them, as before.
func (s *Setup) githubIdentity(ctx context.Context) (name, email string, ok bool) {
	out, err := s.gh(ctx, "api", "user")
	if err != nil {
		return "", "", false
	}
	var user struct {
		Login string `json:"login"`
		ID    int64  `json:"id"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(out, &user); err != nil || user.Login == "" || user.ID <= 0 {
		return "", "", false
	}
	name = strings.TrimSpace(user.Name)
	if name == "" {
		name = user.Login
	}
	return name, fmt.Sprintf("%d+%s@users.noreply.github.com", user.ID, user.Login), true
}

// identity returns git's user.name and user.email, "" for one not set.
func identity(ctx context.Context, r *Runner) (name, email string, err error) {
	get := func(key string) (string, error) {
		out, err := r.Run(ctx, "config", "--get", key)
		if err != nil && exitCode(err) != 1 { // 1: not set
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	}
	if name, err = get("user.name"); err != nil {
		return "", "", err
	}
	email, err = get("user.email")
	return name, email, err
}

// saveIdentity sets the confirmed name and email globally.
func (s *Setup) saveIdentity(ctx context.Context, ans SetupAnswer) SetupStep {
	name, email := strings.TrimSpace(ans.Name), strings.TrimSpace(ans.Email)
	again := func(problem string) SetupStep { return identityStep(name, email, problem, nil, false) }
	switch {
	case name == "" || strings.ContainsAny(name, "\n<>"):
		return again("Type your name (any name: it is only shown in the history).")
	case !validEmail(email):
		return again("Type an email such as you@example.com.")
	}
	r := s.gitRunner()
	if _, err := r.Run(ctx, "config", "--global", "user.name", name); err != nil {
		return s.fail(ctx, fmt.Sprintf("git could not save user.name (%v).", redact(err)))
	}
	if _, err := r.Run(ctx, "config", "--global", "user.email", email); err != nil {
		s.changed = append(s.changed, "Your git name was saved")
		return s.fail(ctx, fmt.Sprintf("git could not save user.email (%v).", redact(err)))
	}
	s.changed = append(s.changed, "Your git name and email were saved")
	step := s.listRepos(ctx)
	step.Notes = append([]string{"✓ git knows you: " + name + " <" + email + ">"}, step.Notes...)
	return step
}

// validEmail reports whether email looks like one: something@something,
// with no spaces.
func validEmail(email string) bool {
	at := strings.Index(email, "@")
	return at > 0 && at < len(email)-1 && !strings.ContainsAny(email, " \t\n<>")
}

// checkSignIn checks gh is signed in to GitHub (step 3).
func (s *Setup) checkSignIn(ctx context.Context) SetupStep {
	if s.signedInNow(ctx) {
		return s.afterSignIn(ctx)
	}
	s.at = AskSignIn
	lines := []string{
		"Sign in to GitHub: gh shows a one-time code and opens your browser.",
		"Sign in there and enter the code. gh keeps the access in your system's keychain; Misoku never sees it.",
	}
	if browserMissing(s.opts.OS) {
		lines = append(lines, "This computer has no browser gh can open: when gh shows the address https://github.com/login/device, open it on any device (your phone works), sign in and enter the code.")
	}
	return SetupStep{Ask: AskSignIn, Title: "Sign in to GitHub", Lines: lines}
}

// afterSignIn goes on once gh is signed in: the identity, then the
// repositories.
func (s *Setup) afterSignIn(ctx context.Context) SetupStep {
	step := s.askIdentity(ctx)
	step.Notes = append([]string{"✓ signed in to GitHub"}, step.Notes...)
	return step
}

// browserMissing reports whether gh finds no browser to open on the system
// goos names, and only prints the sign-in address: Linux without a desktop
// (no opener on PATH), or WSL without wslview, whose xdg-open has no
// browser behind it. gh uses GH_BROWSER or BROWSER first, when set.
func browserMissing(goos string) bool {
	if goos != "linux" || os.Getenv("GH_BROWSER") != "" || os.Getenv("BROWSER") != "" {
		return false
	}
	openers := []string{"xdg-open", "x-www-browser", "www-browser", "wslview"}
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		openers = []string{"wslview"}
	}
	for _, o := range openers {
		if _, err := exec.LookPath(o); err == nil {
			return false
		}
	}
	return true
}

var ghVersionRe = regexp.MustCompile(`gh version (\S+)`)

// GHInstallHint says how to install gh on the system goos names.
func GHInstallHint(goos string) string {
	switch goos {
	case "darwin":
		return "Install it with `brew install gh` (see https://cli.github.com), then run setup again."
	case "windows":
		return "Install it with `winget install --id GitHub.cli` (see https://cli.github.com), then run setup again."
	}
	return "Install it with your package manager, for example `sudo apt install gh` (see https://cli.github.com), then run setup again."
}

// signedInNow reports whether gh is signed in to GitHub. Its output
// names the token's place (and, in some versions, a masked token): it is
// read for the exit code only and never shown.
func (s *Setup) signedInNow(ctx context.Context) bool {
	_, err := s.gh(ctx, "auth", "status", "--hostname", githubHost)
	return err == nil
}

// signedIn goes on after the front end ran SignInCommand.
func (s *Setup) signedIn(ctx context.Context) SetupStep {
	if !s.signedInNow(ctx) {
		return s.fail(ctx, "You are not signed in to GitHub (the sign-in did not finish). Run setup again to try once more.")
	}
	return s.afterSignIn(ctx)
}

// listRepos lists the user's repositories (step 5).
func (s *Setup) listRepos(ctx context.Context) SetupStep {
	out, err := s.gh(ctx, "repo", "list", "--json", "nameWithOwner,visibility,url", "--limit", "1000")
	if err != nil {
		return s.fail(ctx, fmt.Sprintf("gh could not list your repositories (%v).", err))
	}
	var repos []Repo
	if err := json.Unmarshal(out, &repos); err != nil {
		return s.fail(ctx, fmt.Sprintf("gh listed your repositories in a way Misoku does not understand (%v).", err))
	}
	// Private first: they are the ones that can be picked.
	sort.SliceStable(repos, func(i, j int) bool { return repos[i].Private() && !repos[j].Private() })
	s.repos = repos
	return s.askRepo("")
}

// askRepo asks which repository to sync with (AskRepo).
func (s *Setup) askRepo(problem string) SetupStep {
	s.at = AskRepo
	if s.opts.Bring {
		return SetupStep{Ask: AskRepo, Title: "Which repository has your notes?", Repos: s.repos, NewChoice: freshChoice, Problem: problem, Lines: []string{
			"Pick the repository your other computer syncs with.",
		}}
	}
	return SetupStep{Ask: AskRepo, Title: "Which repository keeps this vault?", Repos: s.repos, NewChoice: createChoice, NewFirst: true, Problem: problem, Lines: []string{
		"Pick the repository your other computer already syncs with, or create a new private one.",
	}}
}

// pick takes the chosen repository: a private one is connected, a public
// one refused (notes are stored unencrypted), "new" asks for a name, or,
// with Bring, ends the run so the front end starts new notes.
func (s *Setup) pick(ctx context.Context, ans SetupAnswer) SetupStep {
	choice := ans.Choice
	if text := strings.TrimSpace(ans.Text); text != "" {
		if choice = FindRepo(s.repos, text); choice < 0 {
			return s.askRepo(text + " is not one of your repositories.")
		}
	}
	switch {
	case choice == len(s.repos) && s.opts.Bring:
		s.at = SetupFinished
		s.undo(ctx)
		return SetupStep{Outcome: SetupStartNotes, Title: "Start my notes", Lines: []string{s.StopText("No notes to bring")}}
	case choice == len(s.repos):
		s.at = AskRepoName
		return SetupStep{Ask: AskRepoName, Title: "Name of the new private repository", Default: RepoName(s.opts.Dir),
			Lines: []string{"Misoku creates it private on GitHub and pushes your notes to it."}}
	case choice < 0 || choice > len(s.repos):
		return s.askRepo("Choose one of the listed repositories.")
	}
	repo := s.repos[choice]
	if !repo.Private() {
		return s.askRepo(fmt.Sprintf("%s is public: anyone could read your notes, which are stored unencrypted. Choose a private repository or create a new one (or make it private on GitHub first).", repo.Name))
	}
	// The guard first: a repository it refuses leaves everything as it
	// was, git's access included.
	if step, ok := s.guard(ctx, repo); !ok {
		return step
	}
	return s.use(ctx, repo)
}

// use gives git gh's access and connects the vault with repo, a
// repository the user already has.
func (s *Setup) use(ctx context.Context, repo Repo) SetupStep {
	if err := s.setupGit(ctx); err != nil {
		return s.fail(ctx, err.Error()+".")
	}
	return s.connect(ctx, repo, false)
}

// FindRepo returns the index in repos of the repository text names, as
// "owner/name" or "name" in any letter case, or -1 when none has it.
func FindRepo(repos []Repo, text string) int {
	text = strings.TrimSpace(text)
	for i, r := range repos {
		_, short, _ := strings.Cut(r.Name, "/")
		if strings.EqualFold(r.Name, text) || strings.EqualFold(short, text) {
			return i
		}
	}
	return -1
}

// repoNameRe is what GitHub accepts as a repository name.
var repoNameRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// RepoName suggests a repository name for the vault in dir: its folder's
// name, with what GitHub does not accept turned into "-".
func RepoName(dir string) string {
	name := strings.Map(func(r rune) rune {
		if r < 0x80 && repoNameRe.MatchString(string(r)) {
			return r
		}
		return '-'
	}, filepath.Base(dir))
	name = strings.Trim(name, "-.")
	if name == "" {
		return "notes"
	}
	return name
}

// create creates the new private repository (step 5) and connects it.
func (s *Setup) create(ctx context.Context, ans SetupAnswer) SetupStep {
	name := strings.TrimSpace(ans.Text)
	if name == "" {
		name = RepoName(s.opts.Dir)
	}
	again := func(problem string) SetupStep {
		return SetupStep{Ask: AskRepoName, Title: "Name of the new private repository", Default: name, Problem: problem,
			Lines: []string{"Misoku creates it private on GitHub and pushes your notes to it."}}
	}
	if !repoNameRe.MatchString(name) || name == "." || name == ".." {
		return again("Use only letters, digits, '-', '_' and '.' (GitHub's rule).")
	}
	for _, r := range s.repos {
		if _, short, _ := strings.Cut(r.Name, "/"); strings.EqualFold(short, name) {
			return again(fmt.Sprintf("You already have a repository named %s: pick it from the list (run setup again), or choose another name.", r.Name))
		}
	}
	// Before creating anything: a repository made for a vault that then
	// cannot sync with it would be left half done.
	if err := s.setupGit(ctx); err != nil {
		return s.fail(ctx, err.Error()+".")
	}
	out, err := s.gh(ctx, "repo", "create", name, "--private")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") {
			return again(fmt.Sprintf("A repository named %s already exists in your GitHub account; choose another name.", name))
		}
		return s.fail(ctx, fmt.Sprintf("GitHub did not create the repository (%v).", err))
	}
	page := ""
	for _, f := range strings.Fields(string(out)) {
		if strings.HasPrefix(f, "https://") {
			page = f
		}
	}
	u := strings.TrimPrefix(page, "https://")
	host, path, _ := strings.Cut(u, "/")
	if host == "" || strings.Count(strings.Trim(path, "/"), "/") != 1 {
		s.changed = append(s.changed, "the private repository "+name+" was created on GitHub")
		return s.fail(ctx, fmt.Sprintf("GitHub created the repository %s, but gh did not say where it is. Run setup again and pick it from the list.", name))
	}
	s.changed = append(s.changed, "the private repository "+strings.TrimSuffix(strings.Trim(path, "/"), ".git")+" was created on GitHub")
	return s.connect(ctx, Repo{Name: strings.TrimSuffix(strings.Trim(path, "/"), ".git"), Visibility: "PRIVATE", URL: page}, true)
}

// setupGit makes git use gh's access for GitHub (`gh auth setup-git`), so
// every sync, in the background and without a terminal, signs in without
// asking. It changes only git's credential helper for github.com, once a
// run.
func (s *Setup) setupGit(ctx context.Context) error {
	if s.gitReady {
		return nil
	}
	if _, err := s.gh(ctx, "auth", "setup-git", "--hostname", githubHost); err != nil {
		return fmt.Errorf("gh could not give git the access to GitHub (%v)", err)
	}
	s.gitReady = true
	s.changed = append(s.changed, "GitHub access for git was set up with gh")
	return nil
}

// connect runs the init with repo and lists the vault (steps 6 and 7);
// git already uses gh's access (setupGit).
func (s *Setup) connect(ctx context.Context, repo Repo, created bool) SetupStep {
	s.at = SetupFinished
	url := repo.cloneURL()
	shown := RedactURL(url)
	var lines []string
	if created {
		lines = append(lines, "✓ Created the private repository "+repo.Name+".")
	}
	again := fmt.Sprintf("The repository %s is on GitHub: fix that, then run setup again and pick it.", repo.Name)
	res, err := InitVault(ctx, s.opts.Dir, InitOptions{URL: url, Device: s.vault.Device, Timeout: s.opts.InitTimeout, Now: s.opts.Now, NoPrompt: !s.opts.Interactive})
	if err != nil {
		s.noteInit(res)
		return s.fail(ctx, append(lines, err.Error()+".", again)...)
	}
	if res.CreatedFolder {
		lines = append(lines, "Created the folder "+s.opts.Dir+".")
	}
	lines = append(lines, res.Report(shown)...)
	if res.Sync.Outcome != Synced {
		s.noteInit(res)
		return s.fail(ctx, append(lines, errText(res.Sync.Err)+"; nothing was lost.", again)...)
	}
	interval := s.vault.Interval
	if !s.listed {
		if _, err := config.AddSyncVault(s.opts.ConfigPath, s.opts.Dir); err != nil {
			key, _ := json.Marshal(s.opts.Dir)
			return s.fail(ctx, append(lines, fmt.Sprintf("The vault is set up, but it could not be listed in the config (%v); add it to sync.vaults by hand: {\"sync\": {\"vaults\": {%s: {}}}}", err, key))...)
		}
		lines = append(lines, fmt.Sprintf("Added %s to sync.vaults in %s.", s.opts.Dir, s.opts.ConfigPath))
	}
	when := "while it is open"
	if interval > 0 {
		text := strings.TrimSuffix(interval.String(), "0s")
		if strings.HasSuffix(text, "h0m") {
			text = strings.TrimSuffix(text, "0m")
		}
		when = "every " + text + " while it is open"
	}
	lines = append(lines,
		fmt.Sprintf("Done: Misoku syncs this vault with %s %s, when it opens it and when you quit.", repo.Name, when),
		fmt.Sprintf("On your other computer: install Misoku, run misoku and choose \"Bring my notes from another computer\", then pick %s.", repo.Name),
	)
	return SetupStep{Outcome: SetupDone, Title: "Sync is set up", Lines: lines, Repo: repo.Name}
}

func errText(err error) string {
	if err == nil {
		return "the sync did not finish"
	}
	return err.Error()
}

// fail ends the run with lines saying what went wrong, and a last line
// saying what the run changed before it stopped. What it created and is
// still unused (see undo) is removed first.
func (s *Setup) fail(ctx context.Context, lines ...string) SetupStep {
	s.at = SetupFinished
	s.undo(ctx)
	last := "Nothing was changed."
	if len(s.changed) > 0 {
		last = "Setup stopped. " + s.changes()
	}
	return failed(append(lines, last)...)
}

// StopText is what to say when the run stops early: what (such as "Setup
// cancelled"), then "; nothing was changed." or, when the run already
// changed something (a confirmed identity, git's access through gh), what
// it changed and "Nothing else changed.".
func (s *Setup) StopText(what string) string {
	if len(s.changed) == 0 {
		return what + "; nothing was changed."
	}
	return what + ". " + s.changes()
}

// changes says what the run changed, and that nothing else did.
func (s *Setup) changes() string {
	return strings.Join(s.changed, "; ") + ". Nothing else changed."
}

// failed ends the run with lines saying what went wrong.
func failed(lines ...string) SetupStep {
	return SetupStep{Outcome: SetupFailed, Title: "Sync setup stopped", Lines: lines}
}

// tokenRe matches GitHub tokens (classic, OAuth, app and fine-grained).
var tokenRe = regexp.MustCompile(`\b(gh[pousr]_[A-Za-z0-9]{8,}|github_pat_[A-Za-z0-9_]{8,})`)

// hideSecrets removes credentials from text gh printed: URLs lose their
// user and password (RedactURL), tokens are masked.
func hideSecrets(text string) string {
	return tokenRe.ReplaceAllString(redactText(text), "[token hidden]")
}

// gh runs gh with args in the background: no terminal, empty stdin, no
// prompts, and a time limit. A failure carries gh's own message, without
// credentials.
func (s *Setup) gh(parent context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, s.opts.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.opts.GH, args...)
	cmd.Dir = s.gitRunner().dir
	env := os.Environ()
	for k, v := range map[string]string{
		"GH_PROMPT_DISABLED":    "1", // never ask: there is no one to answer
		"GH_NO_UPDATE_NOTIFIER": "1",
		"GH_SPINNER_DISABLED":   "1",
		"NO_COLOR":              "1",
		"GIT_TERMINAL_PROMPT":   "0",
		"LC_ALL":                "C",
	} {
		setEnv(&env, k, v)
	}
	cmd.Env = env
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.WaitDelay = 2 * time.Second
	err := run(cmd, nil)
	if err == nil {
		return []byte(stdout.String()), nil
	}
	msg := strings.TrimSpace(stderr.String())
	switch {
	case ctx.Err() != nil:
		msg = fmt.Sprintf("gh %s: no answer after %s", args[0], s.opts.Timeout)
	case msg == "":
		msg = err.Error()
	}
	return nil, errors.New(hideSecrets(oneLineText(msg)))
}

// oneLineText joins a multi-line message into one line.
func oneLineText(s string) string { return strings.Join(strings.Fields(s), " ") }
