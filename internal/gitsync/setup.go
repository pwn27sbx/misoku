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
	// AskIdentity: git has no name or email. Name and Email are the
	// values git has (possibly empty); the answer brings both, and OK only
	// once the user confirmed Confirm, the question about saving them in
	// git's global settings.
	AskIdentity
	// AskSignIn: gh is not signed in to GitHub. Lines explain it; on yes
	// the front end runs SignInCommand in the foreground of the terminal,
	// then calls Next with OK.
	AskSignIn
	// AskRepo: pick one of Repos (Choice), name one (Text), or create a
	// new private repository (Choice == len(Repos)).
	AskRepo
	// AskRepoName: the name of the new private repository (Text), Default
	// when empty.
	AskRepoName
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
	// Confirm: AskIdentity's question before saving.
	Confirm string
	// Repos: AskRepo's choices, private ones first.
	Repos []Repo
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
}

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

// githubHost is the only host the assistant signs in to.
const githubHost = "github.com"

// SignInCommand is `gh auth login` in the browser, for the front end to
// run in the foreground of the user's terminal: gh shows a one-time code
// and opens the browser, and keeps the access itself. It has no time
// limit: the user drives it, and Ctrl+C stops it.
func (s *Setup) SignInCommand() *exec.Cmd {
	return exec.Command(s.opts.GH, "auth", "login", "--web", "--git-protocol", "https", "--hostname", githubHost)
}

// Begin reads the config and the folder, without changing anything. A
// vault already set up, or a folder setup must not touch, ends here;
// otherwise it explains what will happen (AskStart).
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
	if done, step := s.inspect(ctx); done {
		return step
	}
	s.at = AskStart
	return SetupStep{Ask: AskStart, Title: "Sync this vault", Lines: []string{
		"Your notes will be kept in a private GitHub repository and reach your other computers on their own.",
		"You need a GitHub account. Misoku signs in with gh, GitHub's command-line tool, which keeps the access: Misoku never sees or stores it.",
		"In the repository your notes are not encrypted, so it stays private.",
	}}
}

// inspect looks at the folder: when it already has an origin, or is one
// setup must not touch, the run ends with done and the step saying why.
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
		return false, SetupStep{}
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
	if err != nil {
		return true, failed(redactText(err.Error()) + ". Setup changed nothing.")
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

// Next takes the answer to the step the previous call returned and goes
// on to the next one.
func (s *Setup) Next(ctx context.Context, ans SetupAnswer) SetupStep {
	if !ans.OK {
		s.at = SetupFinished
		line := "Setup cancelled; nothing was changed."
		if len(s.changed) > 0 {
			line = "Setup cancelled. " + s.changes()
		}
		return SetupStep{Outcome: SetupCancelled, Title: "Sync setup", Lines: []string{line}}
	}
	switch s.at {
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
	}
	return s.fail("Setup has already ended; run it again.")
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

// checkGit checks git's version, that gh is installed, and git's
// identity (steps 2 and 3). gh comes before the identity, so a missing gh
// never leaves an identity saved for nothing.
func (s *Setup) checkGit(ctx context.Context) SetupStep {
	r := s.gitRunner()
	version, ok, err := r.Version(ctx)
	if err != nil {
		return s.fail(fmt.Sprintf("git did not run (%v). Install git %s or newer, then run setup again.", err, MinVersion))
	}
	if !ok {
		return s.fail(fmt.Sprintf("git %s is too old: sync needs git %s or newer. Update git, then run setup again.", version, MinVersion))
	}
	notes := []string{"✓ git " + version}
	path, err := exec.LookPath(s.opts.GH)
	if err != nil {
		return s.fail(
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
	name, email, err := identity(ctx, r)
	if err != nil {
		return s.fail(redactText(err.Error()))
	}
	if name == "" || email == "" {
		s.at = AskIdentity
		return identityStep(name, email, "", notes)
	}
	step := s.checkSignIn(ctx)
	step.Notes = append(notes, step.Notes...)
	return step
}

// identityStep asks for git's name and email.
func identityStep(name, email, problem string, notes []string) SetupStep {
	return SetupStep{Ask: AskIdentity, Title: "Your name for git", Notes: notes, Name: name, Email: email, Problem: problem,
		Lines:   []string{"git does not know your name and email yet: it writes them in every change it records (your notes' history)."},
		Confirm: "Save them in git's global settings (git config --global), for every repository on this computer?"}
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
	again := func(problem string) SetupStep { return identityStep(name, email, problem, nil) }
	switch {
	case name == "" || strings.ContainsAny(name, "\n<>"):
		return again("Type your name (any name: it is only shown in the history).")
	case !validEmail(email):
		return again("Type an email such as you@example.com.")
	}
	r := s.gitRunner()
	for _, kv := range [][2]string{{"user.name", name}, {"user.email", email}} {
		if _, err := r.Run(ctx, "config", "--global", kv[0], kv[1]); err != nil {
			return s.fail(fmt.Sprintf("git could not save %s (%v).", kv[0], redact(err)))
		}
		if kv[0] == "user.name" {
			s.changed = append(s.changed, "Your git name and email were saved")
		}
	}
	step := s.checkSignIn(ctx)
	step.Notes = append([]string{"✓ git knows you: " + name + " <" + email + ">"}, step.Notes...)
	return step
}

// validEmail reports whether email looks like one: something@something,
// with no spaces.
func validEmail(email string) bool {
	at := strings.Index(email, "@")
	return at > 0 && at < len(email)-1 && !strings.ContainsAny(email, " \t\n<>")
}

// checkSignIn checks gh is signed in to GitHub (step 4).
func (s *Setup) checkSignIn(ctx context.Context) SetupStep {
	if s.signedInNow(ctx) {
		step := s.listRepos(ctx)
		step.Notes = append([]string{"✓ signed in to GitHub"}, step.Notes...)
		return step
	}
	s.at = AskSignIn
	return SetupStep{Ask: AskSignIn, Title: "Sign in to GitHub", Lines: []string{
		"Sign in to GitHub: gh shows a one-time code and opens your browser.",
		"Sign in there and enter the code. gh keeps the access in your system's keychain; Misoku never sees it.",
	}}
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
		return s.fail("You are not signed in to GitHub (the sign-in did not finish). Run setup again to try once more.")
	}
	step := s.listRepos(ctx)
	step.Notes = append([]string{"✓ signed in to GitHub"}, step.Notes...)
	return step
}

// listRepos lists the user's repositories (step 5).
func (s *Setup) listRepos(ctx context.Context) SetupStep {
	out, err := s.gh(ctx, "repo", "list", "--json", "nameWithOwner,visibility,url", "--limit", "1000")
	if err != nil {
		return s.fail(fmt.Sprintf("gh could not list your repositories (%v).", err))
	}
	var repos []Repo
	if err := json.Unmarshal(out, &repos); err != nil {
		return s.fail(fmt.Sprintf("gh listed your repositories in a way Misoku does not understand (%v).", err))
	}
	// Private first: they are the ones that can be picked.
	sort.SliceStable(repos, func(i, j int) bool { return repos[i].Private() && !repos[j].Private() })
	s.repos = repos
	return s.askRepo("")
}

func (s *Setup) askRepo(problem string) SetupStep {
	s.at = AskRepo
	return SetupStep{Ask: AskRepo, Title: "Which repository keeps this vault?", Repos: s.repos, Problem: problem, Lines: []string{
		"Pick the repository your other computer already syncs with, or create a new private one.",
	}}
}

// pick takes the chosen repository: a private one is connected, a public
// one refused (notes are stored unencrypted), "new" asks for a name.
func (s *Setup) pick(ctx context.Context, ans SetupAnswer) SetupStep {
	choice := ans.Choice
	if text := strings.TrimSpace(ans.Text); text != "" {
		choice = -1
		for i, r := range s.repos {
			_, short, _ := strings.Cut(r.Name, "/")
			if strings.EqualFold(r.Name, text) || strings.EqualFold(short, text) {
				choice = i
				break
			}
		}
		if choice < 0 {
			return s.askRepo(text + " is not one of your repositories.")
		}
	}
	switch {
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
	return s.connect(ctx, repo, false)
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
		return s.fail(err.Error() + ".")
	}
	out, err := s.gh(ctx, "repo", "create", name, "--private")
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") {
			return again(fmt.Sprintf("A repository named %s already exists in your GitHub account; choose another name.", name))
		}
		return s.fail(fmt.Sprintf("GitHub did not create the repository (%v).", err))
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
		return s.fail(fmt.Sprintf("GitHub created the repository %s, but gh did not say where it is. Run setup again and pick it from the list.", name))
	}
	s.changed = append(s.changed, "the private repository "+strings.TrimSuffix(strings.Trim(path, "/"), ".git")+" was created on GitHub")
	return s.connect(ctx, Repo{Name: strings.TrimSuffix(strings.Trim(path, "/"), ".git"), Visibility: "PRIVATE", URL: page}, true)
}

// setupGit makes git use gh's access for GitHub (`gh auth setup-git`), so
// every sync, in the background and without a terminal, signs in without
// asking. It changes only git's credential helper for github.com.
func (s *Setup) setupGit(ctx context.Context) error {
	if _, err := s.gh(ctx, "auth", "setup-git", "--hostname", githubHost); err != nil {
		return fmt.Errorf("gh could not give git the access to GitHub (%v)", err)
	}
	s.changed = append(s.changed, "GitHub access for git was set up with gh")
	return nil
}

// connect runs the init with repo and lists the vault (steps 6 and 7).
func (s *Setup) connect(ctx context.Context, repo Repo, created bool) SetupStep {
	if !created {
		if err := s.setupGit(ctx); err != nil {
			return s.fail(err.Error() + ".")
		}
	}
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
		return s.fail(append(lines, err.Error()+".", again)...)
	}
	if res.CreatedFolder {
		lines = append(lines, "Created the folder "+s.opts.Dir+".")
	}
	lines = append(lines, res.Report(shown)...)
	if res.Sync.Outcome != Synced {
		return s.fail(append(lines, errText(res.Sync.Err)+"; nothing was lost.", again)...)
	}
	interval := s.vault.Interval
	if !s.listed {
		if _, err := config.AddSyncVault(s.opts.ConfigPath, s.opts.Dir); err != nil {
			key, _ := json.Marshal(s.opts.Dir)
			return s.fail(append(lines, fmt.Sprintf("The vault is set up, but it could not be listed in the config (%v); add it to sync.vaults by hand: {\"sync\": {\"vaults\": {%s: {}}}}", err, key))...)
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
		fmt.Sprintf("On your other computer: run `misoku sync setup` (or `:sync setup` in Misoku) and pick %s.", repo.Name),
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
// saying what the run changed before it stopped.
func (s *Setup) fail(lines ...string) SetupStep {
	s.at = SetupFinished
	last := "Nothing was changed."
	if len(s.changed) > 0 {
		last = "Setup stopped. " + s.changes()
	}
	return failed(append(lines, last)...)
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
