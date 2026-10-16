package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pwn27sbx/misoku/internal/gitsync"
)

const syncSetupUsage = "Usage: misoku sync setup [--dir <folder>]"

// setupTimeout bounds each gh command and local git check of the setup
// assistant; the sign-in has none (the user drives it).
const setupTimeout = time.Minute

// syncSetup is the setup assistant in the terminal (`misoku sync setup`):
// it asks on stdout and reads the answers from stdin, one line each, and
// runs gh's sign-in in the foreground. The end of the input cancels, and
// so does Ctrl+C while a question waits.
func syncSetup(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, cfgPath string) int {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	return runSetup(ctx, args, stdin, stdout, stderr, cfgPath, interrupt)
}

// runSetup is syncSetup with the channel that brings Ctrl+C.
func runSetup(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, cfgPath string, interrupt <-chan os.Signal) int {
	fs := flag.NewFlagSet("sync setup", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dirFlag := fs.String("dir", "", "vault folder (defaults to $MISOKU_WORKSPACE, then the current directory)")
	fs.Usage = func() {
		fmt.Fprintln(stderr, syncSetupUsage)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return 2
	}
	dir, err := filepath.Abs(resolveWorkspace(*dirFlag))
	if err != nil {
		fmt.Fprintf(stderr, "misoku sync setup: %v\n", err)
		return 1
	}
	s := gitsync.NewSetup(gitsync.SetupOptions{
		Dir: dir, ConfigPath: cfgPath, Timeout: setupTimeout, InitTimeout: initTimeout,
		Interactive: true,
	})
	t := &setupTerminal{in: bufio.NewReader(stdin), stdin: stdin, out: stdout, err: stderr, s: s, interrupt: interrupt}
	fmt.Fprintf(stdout, "Setting up sync of %s with GitHub.\n", dir)
	if step := t.run(ctx); step.Outcome == gitsync.SetupDone || step.Outcome == gitsync.SetupAlready {
		return 0
	}
	return 1
}

// setupTerminal asks the assistant's questions in the terminal.
type setupTerminal struct {
	in       *bufio.Reader
	stdin    io.Reader
	out, err io.Writer
	s        *gitsync.Setup
	// interrupt brings Ctrl+C: while a question waits it cancels, so the
	// engine can undo what this run did (a repository it created). A
	// Ctrl+C during a step stops that step's git and gh (work) and, if a
	// question still follows, cancels it too.
	interrupt <-chan os.Signal
}

// run drives the assistant from Begin to its end, asking in the
// terminal, and returns its last step.
func (t *setupTerminal) run(ctx context.Context) gitsync.SetupStep {
	step := t.work(ctx, t.s.Begin)
	for {
		t.show(step)
		if step.Ask == gitsync.SetupFinished {
			return step
		}
		ans := t.answer(step)
		step = t.work(ctx, func(ctx context.Context) gitsync.SetupStep { return t.s.Next(ctx, ans) })
	}
}

// work runs one call of the assistant. Ctrl+C during it stops the git and
// gh it runs, so it ends cleanly; while a question waits, Ctrl+C cancels
// (answer).
func (t *setupTerminal) work(ctx context.Context, call func(context.Context) gitsync.SetupStep) gitsync.SetupStep {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	return call(ctx)
}

// show prints the step: the checks passed, why the last answer was not
// taken, and what it says.
func (t *setupTerminal) show(step gitsync.SetupStep) {
	for _, n := range step.Notes {
		fmt.Fprintln(t.out, n)
	}
	if step.Problem != "" {
		fmt.Fprintln(t.out, "! "+step.Problem)
	}
	w := t.out
	if step.Outcome == gitsync.SetupFailed {
		w = t.err
	}
	for _, l := range step.Lines {
		fmt.Fprintln(w, l)
	}
}

// line reads one answer, trimmed; ok is false at the end of the input.
func (t *setupTerminal) line(prompt string) (string, bool) {
	fmt.Fprint(t.out, prompt)
	text, err := t.in.ReadString('\n')
	if err != nil && text == "" {
		fmt.Fprintln(t.out)
		return "", false
	}
	return strings.TrimSpace(text), true
}

// yes asks a yes/no question; an empty answer is def.
func (t *setupTerminal) yes(question string, def bool) bool {
	hint := " [y/N] "
	if def {
		hint = " [Y/n] "
	}
	text, ok := t.line(question + hint)
	if !ok {
		return false
	}
	if text == "" {
		return def
	}
	switch strings.ToLower(text) {
	case "y", "yes":
		return true
	}
	return false
}

// answer asks what step needs, as ask does; Ctrl+C meanwhile is a
// cancel. The question's read is then left waiting: the assistant ends
// right after, and nothing reads its answer.
func (t *setupTerminal) answer(step gitsync.SetupStep) gitsync.SetupAnswer {
	return interruptible(t, func() gitsync.SetupAnswer { return t.ask(step) }, gitsync.SetupAnswer{})
}

// interruptible runs question, which reads from t, and returns its
// answer, or cancel when Ctrl+C comes first (t.interrupt). The read is
// then left waiting: the caller stops asking, and nothing reads its
// answer.
func interruptible[T any](t *setupTerminal, question func() T, cancel T) T {
	got := make(chan T, 1)
	go func() { got <- question() }()
	select {
	case ans := <-got:
		return ans
	case <-t.interrupt:
		fmt.Fprintln(t.out)
		return cancel
	}
}

// ask asks what step needs and returns the answer.
func (t *setupTerminal) ask(step gitsync.SetupStep) gitsync.SetupAnswer {
	cancel := gitsync.SetupAnswer{}
	switch step.Ask {
	case gitsync.AskOwnRepo:
		return gitsync.SetupAnswer{OK: t.yes(step.Confirm, true)}
	case gitsync.AskStart:
		return gitsync.SetupAnswer{OK: t.yes("Start?", true)}
	case gitsync.AskIdentity:
		name, ok := t.withDefault("Your name", step.Name)
		if !ok {
			return cancel
		}
		email, ok := t.withDefault("Your email", step.Email)
		if !ok || !t.yes(step.Confirm, false) {
			return cancel
		}
		return gitsync.SetupAnswer{OK: true, Name: name, Email: email}
	case gitsync.AskSignIn:
		if !t.yes("Sign in to GitHub now?", true) {
			return cancel
		}
		t.signIn()
		return gitsync.SetupAnswer{OK: true}
	case gitsync.AskRepo:
		for i, r := range step.Repos {
			fmt.Fprintf(t.out, "  %d) %s  %s\n", i+1, r.Name, visibility(r))
		}
		fmt.Fprintf(t.out, "  %d) %s\n", len(step.Repos)+1, step.NewChoice)
		text, ok := t.line("Number or name (Enter cancels): ")
		if !ok || text == "" {
			return cancel
		}
		// A number is a list number only when the list has it: a
		// repository named "2024" in a shorter list is read as a name.
		if n, err := strconv.Atoi(text); err == nil && n >= 1 && n <= len(step.Repos)+1 {
			return gitsync.SetupAnswer{OK: true, Choice: n - 1}
		}
		return gitsync.SetupAnswer{OK: true, Text: text}
	case gitsync.AskRepoName:
		name, ok := t.withDefault("Repository name", step.Default)
		if !ok {
			return cancel
		}
		return gitsync.SetupAnswer{OK: true, Text: name}
	}
	return cancel
}

// withDefault asks for a value, offering def; ok is false at the end of
// the input.
func (t *setupTerminal) withDefault(label, def string) (string, bool) {
	prompt := label + ": "
	if def != "" {
		prompt = label + " [" + def + "]: "
	}
	text, ok := t.line(prompt)
	if text == "" {
		text = def
	}
	return text, ok
}

// signIn runs gh's sign-in in the foreground: it prints the one-time code
// and opens the browser in the user's own terminal. Its outcome is read
// afterwards from gh itself (Next), so its exit status is not needed.
func (t *setupTerminal) signIn() {
	cmd := t.s.SignInCommand()
	cmd.Stdout, cmd.Stderr = t.out, t.err
	// A real terminal is handed over as it is; a reader that is not one
	// (a test) is not shared with gh, which gets no input.
	if f, ok := t.stdin.(*os.File); ok {
		cmd.Stdin = f
	}
	_ = cmd.Run() // gh says how it went; Next checks whether it signed in
}

// visibility labels a repository in the list.
func visibility(r gitsync.Repo) string {
	if r.Private() {
		return "private"
	}
	return strings.ToLower(r.Visibility) + ": not allowed, notes are stored unencrypted"
}
