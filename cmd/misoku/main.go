// Command misoku is the entry point for the Misoku terminal notebook application.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/pwn27sbx/misoku/internal/app"
	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/core"
	"github.com/pwn27sbx/misoku/internal/export"
	"github.com/pwn27sbx/misoku/internal/preview"
	"github.com/pwn27sbx/misoku/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// version is stamped at build time via -ldflags "-X main.version=...".
// GoReleaser sets this automatically for tagged release builds.
var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "export" {
		os.Exit(runExport(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "sync" {
		os.Exit(runSync(os.Args[2:]))
	}

	flag.Usage = func() { usage(flag.CommandLine.Output()) }
	opts, folder, err := parseOpenArgs(flag.CommandLine, os.Args[1:])
	if err != nil { // a bad flag has already exited (flag.ExitOnError)
		usage(os.Stderr)
		os.Exit(2)
	}
	dir, themeName, themesDir, showVer, listThemes := opts.dir, opts.theme, opts.themesDir, opts.showVer, opts.listThemes

	cfgPath := config.DefaultPath()
	cfg, cfgErr := config.Load(cfgPath)
	if cfgErr != nil {
		fmt.Fprintf(os.Stderr, "Ignoring unreadable config %s: %v\n", cfgPath, cfgErr)
	}
	// Custom themes location must be registered before any theme lookup.
	if themesDir == "" {
		themesDir = cfg.ThemesDir
	}
	if themesDir != "" {
		theme.SetUserThemesPath(themesDir)
		if _, err := os.Stat(theme.UserThemesDir()); err != nil {
			fmt.Fprintf(os.Stderr, "Custom themes path %s not found; using bundled themes only\n", theme.UserThemesDir())
		}
	}

	if showVer {
		fmt.Printf("misoku %s\n", version)
		return
	}
	if listThemes {
		for _, n := range theme.Names() {
			fmt.Println(n)
		}
		return
	}

	if themeName == "" {
		themeName = cfg.Theme
	}
	if themeName != "" {
		if _, ok := theme.Get(themeName); !ok {
			fmt.Fprintf(os.Stderr, "Unknown theme %q (run misoku --list-themes)\n", themeName)
			os.Exit(2)
		}
	}

	choice := pickWorkspace(workspaceInput{Flag: dir, Arg: folder, Env: os.Getenv("MISOKU_WORKSPACE"), Config: cfg, Getwd: os.Getwd})
	if choice.Warning != "" {
		fmt.Fprintln(os.Stderr, choice.Warning)
	}
	home, _ := os.UserHomeDir() // without one, only the root and system folders are no place for notes
	if welcomeDue(welcomeCheck{Choice: choice, ConfigPath: cfgPath, ConfigErr: cfgErr, Config: cfg, Home: home,
		Terminal: isTerminal(os.Stdin) && isTerminal(os.Stdout)}) {
		dir, code := greet(cfgPath, cfg, home, choice, welcomeTheme(themeName))
		if dir == "" {
			os.Exit(code)
		}
		choice = workspaceChoice{Dir: dir, Source: fromNotesDir}
	}
	if choice.Source == notesDirMissing {
		// Opening would create it empty, beside the user's real notes.
		fmt.Fprintln(os.Stderr, missingNotesMessage(choice.Dir))
		os.Exit(1)
	}
	workspace := choice.Dir

	store, err := core.NewStore(workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open workspace %q: %v\n", workspace, err)
		os.Exit(1)
	}

	logPath := logFilePath()
	f, err := tea.LogToFile(logPath, "debug")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open log file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	log.Printf("Starting Misoku (workspace: %s)\n", store.Root())

	appModel := app.New(store, app.WithTheme(themeName), app.WithConfigPath(cfgPath), app.WithKeys(cfg.Keys), app.WithUI(cfg.UI))

	p := tea.NewProgram(appModel, tea.WithAltScreen(), tea.WithMouseAllMotion())
	final, err := p.Run()
	if err != nil {
		log.Printf("Fatal error running program: %v\n", err)
		fmt.Fprintf(os.Stderr, "Error running Misoku application: %v\n", err)
		os.Exit(1)
	}
	if m, ok := final.(app.Model); ok {
		pushOnQuit(m)
	}

	log.Println("Misoku exited cleanly.")
}

// greet shows the welcome, drawn in th, and returns the folder to open,
// or "" and the exit code: the welcome screen when the terminal can draw
// it (screenFits), else the text menu, where Ctrl+C cancels a question
// that waits.
func greet(cfgPath string, cfg config.Config, home string, choice workspaceChoice, th theme.Theme) (string, int) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	o := welcomeFor(choice, cfg, home, os.Getwd)
	o.Stdin, o.Out, o.Err, o.ConfigPath, o.Interrupt = os.Stdin, os.Stdout, os.Stderr, cfgPath, interrupt
	o.ClassicKeys = classicKeysOn(runtime.GOOS, onTerminal(os.Stdin, os.Stdout))
	if w, h, ok := terminalSize(os.Stdout); ok && screenFits(os.Getenv("TERM"), w, h) {
		return runWelcomeScreen(o, th)
	}
	return runWelcome(context.Background(), o)
}

// welcomeTheme is the theme named name (already checked), or the
// default one when none is named.
func welcomeTheme(name string) theme.Theme {
	if th, ok := theme.Get(name); ok {
		return th
	}
	return theme.DefaultTheme()
}

// options are the flags of `misoku` itself.
type options struct {
	dir, theme, themesDir string
	showVer, listThemes   bool
}

// defineFlags declares misoku's flags on fs.
func defineFlags(fs *flag.FlagSet) *options {
	o := &options{}
	fs.StringVar(&o.dir, "dir", "", "folder to open, like misoku <folder> (defaults to $MISOKU_WORKSPACE, then notes_dir in the config, then the current folder)")
	fs.BoolVar(&o.showVer, "version", false, "print the Misoku version and exit")
	fs.StringVar(&o.theme, "theme", "", "colour theme for this session (see --list-themes)")
	fs.StringVar(&o.themesDir, "themes-dir", "", "directory or .json file with custom themes (overrides config themes_dir)")
	fs.BoolVar(&o.listThemes, "list-themes", false, "list the available colour themes and exit")
	return o
}

// pushOnQuit pushes what was saved in a synced vault once the screen has
// closed (S14), for at most app.QuitSyncLimit. Ctrl+C stops it at once:
// git is killed and the commits wait for the next sync.
func pushOnQuit(m app.Model) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	m.SyncOnQuit(ctx, os.Stderr, app.QuitSyncLimit)
}

// resolveWorkspace picks the vault folder of `misoku sync init` and
// `misoku sync setup`: an explicit --dir flag wins, then
// $MISOKU_WORKSPACE, falling back to the current directory. Unlike
// misoku itself, they never use notes_dir: they set up the folder they
// are pointed at.
func resolveWorkspace(flagDir string) string {
	return pickWorkspace(workspaceInput{Flag: flagDir, Env: os.Getenv("MISOKU_WORKSPACE"), Getwd: os.Getwd}).Dir
}

// logFilePath keeps the debug log out of the user's notes workspace,
// writing it to the OS user cache directory instead.
func logFilePath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "misoku.log"
	}
	dir := filepath.Join(cacheDir, "misoku")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "misoku.log"
	}
	return filepath.Join(dir, "misoku.log")
}

// runExport handles "misoku export note.md [-o out.pdf]".
func runExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	out := fs.String("o", "", "output file or folder (defaults to the note's name with .pdf, next to it)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: misoku export <note.md> [-o out.pdf]")
		fs.PrintDefaults()
	}
	// Allow the flag after the note as well as before it.
	notes, err := positionals(fs, args)
	if err != nil {
		return 2
	}
	if len(notes) != 1 {
		fs.Usage()
		return 2
	}
	note := notes[0]
	data, err := os.ReadFile(note)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read %s: %v\n", note, err)
		return 1
	}
	dir := filepath.Dir(note)
	name := strings.TrimSuffix(filepath.Base(note), filepath.Ext(note))
	target := *out
	if target != "" && !filepath.IsAbs(target) {
		// -o is relative to where the command runs, not to the note.
		if cwd, err := os.Getwd(); err == nil {
			folder := strings.HasSuffix(target, "/") || strings.HasSuffix(target, string(filepath.Separator))
			target = filepath.Join(cwd, target)
			if folder {
				target += string(filepath.Separator)
			}
		}
	}
	dest := export.Target(target, dir, name)
	text := string(data)
	// ![[embeds]] resolve against the notes in the note's folder.
	if store, err := core.NewStore(dir); err == nil {
		notes := preview.StoreNotes(store, nil, filepath.Base(note), nil)
		text = export.Printable(text, func(t string) (preview.Note, bool) {
			if t == "" {
				return preview.Note{Title: name, Content: string(data), Dir: dir}, true
			}
			return notes(t)
		})
	}
	if err := export.WriteFile(text, export.Options{Title: name, BaseDir: dir}, dest); err != nil {
		fmt.Fprintf(os.Stderr, "Export failed: %v\n", err)
		return 1
	}
	fmt.Println(dest)
	return 0
}
