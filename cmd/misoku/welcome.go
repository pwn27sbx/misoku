package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// The welcome greets someone who runs misoku with no folder and no notes
// yet: it asks where their notes are to live, remembers that folder
// (notes_dir) and opens it, before the app starts. It is a screen of its
// own (welcome_screen.go) when the terminal can draw one, else it asks in
// the terminal, line by line like `misoku sync setup`.

// welcomeCheck is everything that decides whether misoku greets with the
// welcome instead of opening Choice.Dir.
type welcomeCheck struct {
	Choice     workspaceChoice
	ConfigPath string
	// ConfigErr is the error reading the config, if any.
	ConfigErr error
	Config    config.Config
	// Terminal: someone is there to answer (stdin and stdout are a
	// terminal).
	Terminal bool
	Home     string
}

// welcomeDue reports whether misoku shows the welcome: only with someone
// to answer it, a config it can read and write the choice to, and no
// folder given (--dir, a folder argument or $MISOKU_WORKSPACE) or
// remembered. Then a remembered folder that is gone, or a terminal folder
// that is not a vault of notes or is no place for them (home, root, a
// system folder), is greeted.
func welcomeDue(c welcomeCheck) bool {
	if !c.Terminal || c.ConfigErr != nil || c.ConfigPath == "" || c.Choice.Warning != "" {
		return false
	}
	switch c.Choice.Source {
	case notesDirMissing:
		return true
	case fromCwd:
		return gitsync.UnsafeFolder(c.Choice.Dir, c.Home) || !holdsNotes(c.Config, c.Choice.Dir)
	}
	return false
}

// canUseFolder reports whether the welcome offers "Use this folder" for
// dir: it holds notes and is a folder of its own, so the home folder
// (~/.ssh included) is never the one synced.
func canUseFolder(cfg config.Config, dir, home string) bool {
	return !gitsync.UnsafeFolder(dir, home) && holdsNotes(cfg, dir)
}

// holdsNotes reports whether dir is a vault: listed in sync.vaults, or
// with a Markdown note at its first level. A folder that cannot be read
// counts as one, so misoku opens it as before and says what is wrong.
func holdsNotes(cfg config.Config, dir string) bool {
	vaults, _ := cfg.Sync.Parse() // a vault that cannot be used is not one
	if _, ok := gitsync.Listed(vaults, dir); ok {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".md") {
			return true
		}
	}
	return false
}

// defaultNotes is the folder the welcome proposes, under the home folder.
const defaultNotes = "Notes"

// freeFolder is the folder Bring my notes proposes, as written in the
// config: ~/name, or ~/name-2, ~/name-3… when it already has something
// in it, so the notes brought never mix with others.
func freeFolder(home, name string) string {
	for n := 1; n < 100; n++ {
		candidate := name
		if n > 1 {
			candidate = name + "-" + strconv.Itoa(n)
		}
		entries, err := os.ReadDir(filepath.Join(home, candidate))
		if errors.Is(err, fs.ErrNotExist) || (err == nil && len(entries) == 0) {
			return "~/" + candidate
		}
	}
	return "~/" + name + "-new"
}

// welcomeOptions are the settings of runWelcome.
type welcomeOptions struct {
	Stdin    io.Reader
	Out, Err io.Writer
	// ConfigPath is where the chosen folder is remembered.
	ConfigPath string
	Home       string
	// Missing is the remembered notes folder that is gone, if any.
	Missing string
	// Cwd is the terminal's folder when "Use this folder" is offered
	// (see canUseFolder), else "".
	Cwd string
	// Interrupt brings Ctrl+C: a question waiting is then cancelled.
	Interrupt <-chan os.Signal
	// ClassicKeys: switch to the classic keyboard mode while asking
	// (classicKeysOn).
	ClassicKeys bool
}

// welcomeFor is what the welcome says for choice: the remembered folder
// that is gone, and the terminal's folder when it can be used as it is
// (canUseFolder). The terminal's folder is choice.Dir, or, when that is
// the missing folder, getwd's.
func welcomeFor(choice workspaceChoice, cfg config.Config, home string, getwd func() (string, error)) welcomeOptions {
	o := welcomeOptions{Home: home}
	cwd := choice.Dir
	if choice.Source == notesDirMissing {
		o.Missing = choice.Dir
		var err error
		if cwd, err = getwd(); err != nil {
			return o // no folder to offer
		}
	}
	if !canUseFolder(cfg, cwd, home) {
		return o
	}
	if abs, err := filepath.Abs(cwd); err == nil {
		o.Cwd = abs
	}
	return o
}

// Choices of the welcome, by number.
const (
	choiceStart = iota + 1
	choiceBring
	choiceUseThis
)

// runWelcome asks where the notes live and returns the folder to open,
// once it exists and is remembered. "" means open nothing and exit with
// the code: 0 when the user stopped, 1 when something failed.
func runWelcome(ctx context.Context, o welcomeOptions) (string, int) {
	t := &setupTerminal{in: bufio.NewReader(o.Stdin), stdin: o.Stdin, out: o.Out, err: o.Err, interrupt: o.Interrupt}
	defer classicKeyboard(o.Out, o.ClassicKeys)()
	fmt.Fprintln(o.Out, "Welcome to Misoku.")
	if o.Missing != "" {
		fmt.Fprintf(o.Out, "I can't find your notes at %s.\n", o.Missing)
	}
	fmt.Fprintln(o.Out, "Where do your notes live?")
	fmt.Fprintf(o.Out, "  %d) Start my notes\n", choiceStart)
	fmt.Fprintf(o.Out, "  %d) Bring my notes from another computer\n", choiceBring)
	last := choiceBring
	if o.Cwd != "" {
		last = choiceUseThis
		fmt.Fprintf(o.Out, "  %d) Use this folder: %s\n", choiceUseThis, o.Cwd)
	}
	for {
		text, ok := t.prompt("Number (Enter cancels): ")
		if !ok || text == "" {
			return stopped(o.Out)
		}
		switch n, _ := strconv.Atoi(text); {
		case n == choiceStart:
			return startNotes(t, o)
		case n == choiceBring:
			return bringNotes(ctx, t, o)
		case n == choiceUseThis && last == choiceUseThis:
			return remember(o, o.Cwd, o.Cwd), 0
		}
		fmt.Fprintf(o.Out, "! Type %s.\n", choicesText(last))
	}
}

// choicesText is "1 or 2" or "1, 2 or 3".
func choicesText(last int) string {
	if last == choiceBring {
		return "1 or 2"
	}
	return "1, 2 or 3"
}

// stopped ends the welcome when the user stops it.
func stopped(out io.Writer) (string, int) {
	fmt.Fprintln(out, "Nothing was changed. Run misoku again when you are ready.")
	return "", 0
}

// prompt reads one answer; ok is false at the end of the input or on
// Ctrl+C.
func (t *setupTerminal) prompt(question string) (string, bool) {
	type answer struct {
		text string
		ok   bool
	}
	a := interruptible(t, func() answer {
		text, ok := t.line(question)
		return answer{text, ok}
	}, answer{})
	return a.text, a.ok
}

// askFolder asks for the notes folder, offering def, until it gets one
// that can hold notes (checkFolder). written is the folder as typed
// ("~/Notes" stays so), abs the folder itself; ok is false when the user
// stopped.
func askFolder(t *setupTerminal, home, def string) (written, abs string, ok bool) {
	for {
		text, answered := t.prompt("Folder for your notes [" + def + "]: ")
		if !answered {
			return "", "", false
		}
		written, abs, problem := checkFolder(text, def, home)
		if problem == "" {
			return written, abs, true
		}
		fmt.Fprintln(t.out, "! "+problem)
	}
}

// checkFolder reads text, the folder typed for the notes (def when
// empty), as the welcome takes it: written is the folder to remember
// ("~/Notes" stays so, any other is made absolute and clean), abs the
// folder itself. problem, when not "", says why it cannot hold notes: it
// is not a full path, or it is the home folder, the root of a disk or a
// system folder.
func checkFolder(text, def, home string) (written, abs, problem string) {
	if text == "" {
		text = def
	}
	full, err := config.ExpandHome(text)
	switch {
	case err != nil:
		return "", "", "Type a full path, such as ~/Notes."
	case gitsync.UnsafeFolder(full, home):
		return "", "", full + " is your home folder, the root of a disk or a system folder: choose a folder of its own for your notes, such as ~/Notes."
	}
	if !strings.HasPrefix(text, "~") {
		text = full
	}
	return text, full, ""
}

// startNotes is "Start my notes": it asks for the folder, creates it (not
// its parent: a typo must not make folders elsewhere), remembers it and
// opens it.
func startNotes(t *setupTerminal, o welcomeOptions) (string, int) {
	for {
		written, abs, ok := askFolder(t, o.Home, "~/"+defaultNotes)
		if !ok {
			return stopped(o.Out)
		}
		if err := makeFolder(abs); err != nil {
			fmt.Fprintf(o.Out, "! %v.\n", err)
			continue
		}
		return remember(o, written, abs), 0
	}
}

// makeFolder creates dir, or takes it as it is when it is a folder
// already. Its parent must exist.
func makeFolder(dir string) error {
	err := os.Mkdir(dir, 0o755)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, fs.ErrExist):
		if info, statErr := os.Stat(dir); statErr == nil && info.IsDir() {
			return nil
		}
		return fmt.Errorf("%s is a file, not a folder: choose another name", dir)
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("the folder %s does not exist: create it first, or choose another folder", filepath.Dir(dir))
	}
	return fmt.Errorf("could not create %s (%w)", dir, err)
}

// bringNotes is "Bring my notes from another computer": the setup
// assistant, in its Bring mode, brings the notes of a repository into a
// free folder; then that folder is remembered and opened. Choosing "I
// don't have notes on another computer" goes on with Start my notes.
func bringNotes(ctx context.Context, t *setupTerminal, o welcomeOptions) (string, int) {
	written, abs, ok := askFolder(t, o.Home, freeFolder(o.Home, defaultNotes))
	if !ok {
		return stopped(o.Out)
	}
	t.s = gitsync.NewSetup(gitsync.SetupOptions{
		Dir: abs, ConfigPath: o.ConfigPath, Timeout: setupTimeout, InitTimeout: initTimeout,
		Interactive: true, Bring: true,
	})
	switch t.run(ctx).Outcome {
	case gitsync.SetupDone, gitsync.SetupAlready:
		return remember(o, written, abs), 0
	case gitsync.SetupStartNotes:
		return startNotes(t, o)
	case gitsync.SetupCancelled:
		return "", 0
	}
	return "", 1
}

// remember saves written as notes_dir and returns abs, the folder to
// open. When it cannot be saved, the folder still opens, and the welcome
// says it will ask again next time.
func remember(o welcomeOptions, written, abs string) string {
	if err := config.SetNotesDir(o.ConfigPath, written); err != nil {
		fmt.Fprintf(o.Err, "Could not remember %s as your notes folder (%v); misoku will ask again next time.\n", abs, err)
	}
	return abs
}
