package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pwn27sbx/misoku/internal/config"
	"github.com/pwn27sbx/misoku/internal/gitsync"
)

// initTimeout bounds each git command of `misoku sync init`: long enough
// to type a password or check an SSH host key. Ctrl+C stops it at once.
const initTimeout = 10 * time.Minute

const syncInitUsage = "Usage: misoku sync init <url> [--dir <folder>]"

// runSync handles "misoku sync init <url>".
func runSync(args []string) int {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "-help" || args[0] == "--help") {
		fmt.Fprintln(os.Stdout, syncInitUsage)
		return 0
	}
	if len(args) == 0 || args[0] != "init" {
		fmt.Fprintln(os.Stderr, syncInitUsage)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return syncInit(ctx, args[1:], os.Stdout, os.Stderr, config.DefaultPath())
}

// syncInit sets up the vault to sync with a git remote (S10) and lists it
// in sync.vaults of the config at cfgPath, keeping every other setting.
// It runs in the foreground: git may ask for a password or to accept a
// new SSH host key. The URL is never printed with its credentials.
func syncInit(ctx context.Context, args []string, stdout, stderr io.Writer, cfgPath string) int {
	fs := flag.NewFlagSet("sync init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dirFlag := fs.String("dir", "", "vault folder (defaults to $MISOKU_WORKSPACE, then the current directory)")
	fs.Usage = func() {
		fmt.Fprintln(stderr, syncInitUsage)
		fs.PrintDefaults()
	}
	// Allow the flag after the URL as well as before it.
	var urls []string
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return 0 // asked for: the usage is already printed
			}
			return 2
		}
		if fs.NArg() == 0 {
			break
		}
		urls = append(urls, fs.Arg(0))
		args = fs.Args()[1:]
	}
	if len(urls) != 1 || strings.TrimSpace(urls[0]) == "" {
		fs.Usage()
		return 2
	}
	url := urls[0]
	fail := func(format string, a ...any) int {
		fmt.Fprintf(stderr, "misoku sync init: "+format+"\n", a...)
		return 1
	}
	dir, err := filepath.Abs(resolveWorkspace(*dirFlag))
	if err != nil {
		return fail("%v", err)
	}
	// Read first: a config that cannot be read could not list the vault
	// either, so nothing is set up until it is fixed.
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fail("the config file %s could not be read (%v); fix it, then run init again", cfgPath, err)
	}
	vaults, _ := cfg.Sync.Parse()
	vault, listed := gitsync.Listed(vaults, dir)
	if !listed {
		// The device a vault listed with no settings gets.
		if vs, _ := (config.Sync{Vaults: map[string]config.SyncVault{dir: {}}}).Parse(); len(vs) == 1 {
			vault = vs[0]
		}
	}

	shown := gitsync.RedactURL(url)
	fmt.Fprintf(stdout, "Setting up sync of %s with %s…\n", dir, shown)
	fmt.Fprintln(stdout, "Your notes are stored unencrypted in that repository: keep it private.")
	res, err := gitsync.InitVault(ctx, dir, gitsync.InitOptions{URL: url, Device: vault.Device, Timeout: initTimeout})
	if err != nil {
		return fail("%v", err)
	}
	if res.CreatedFolder {
		fmt.Fprintf(stdout, "Created the folder %s.\n", dir)
	}
	report(stdout, res, shown)
	if res.Sync.Outcome != gitsync.Synced {
		return fail("%v\nNothing was lost; fix that, then run `misoku sync init <url>` again.", res.Sync.Err)
	}
	if !listed {
		if _, err := config.AddSyncVault(cfgPath, dir); err != nil {
			key, _ := json.Marshal(dir)
			return fail("the vault is set up, but it could not be listed in the config (%v); add it to sync.vaults by hand:\n  {\"sync\": {\"vaults\": {%s: {}}}}", err, key)
		}
		fmt.Fprintf(stdout, "Added %s to sync.vaults in %s.\n", dir, cfgPath)
	}
	if res.AlreadySetUp && listed {
		fmt.Fprintf(stdout, "Already set up: this folder syncs with %s (branch %s); nothing changed.\n", shown, res.Sync.Branch)
		return 0
	}
	fmt.Fprintln(stdout, "Done: Misoku syncs this folder when it opens it, and with :sync.")
	return 0
}

// report says what InitVault did, one line per step. url is already
// without credentials.
func report(w io.Writer, res gitsync.InitResult, url string) {
	if res.CreatedRepo {
		fmt.Fprintf(w, "Created a git repository (branch %s).\n", res.Branch)
	}
	if res.AddedOrigin {
		fmt.Fprintf(w, "Added origin %s.\n", url)
	}
	if res.WroteAttributes {
		fmt.Fprintln(w, "Wrote .gitattributes (* -text): files stay byte for byte, with no line-ending conversion.")
	}
	if res.KeptAttributes {
		fmt.Fprintln(w, "Kept your .gitattributes as it is; add the line `* -text` to it so line endings never change between systems.")
	}
	for _, big := range res.Sync.Commit.Skipped {
		fmt.Fprintf(w, "Left out %s (%d MB): files that big do not sync.\n", big.Path, big.Size>>20)
	}
	// The files that do not sync as plain files (S15), in the words
	// `:sync status` uses.
	for _, n := range res.Sync.Commit.Notices {
		fmt.Fprintln(w, n.String())
	}
	if res.Files > 0 {
		noun := "files"
		if res.Files == 1 {
			noun = "file"
		}
		fmt.Fprintf(w, "Committed %d %s.\n", res.Files, noun)
	}
	if res.Sync.Updated {
		fmt.Fprintln(w, "Brought origin's notes.")
	}
	if n := len(res.Sync.Conflicts); n > 0 {
		fmt.Fprintf(w, "Kept both versions of %d note(s) changed on both sides; the other version is in:\n", n)
		for _, c := range res.Sync.Conflicts {
			fmt.Fprintf(w, "  %s\n", c)
		}
		fmt.Fprintln(w, "Resolve each by deleting the copy (or copying what you want into the note).")
	}
	if res.Sync.Pushed {
		fmt.Fprintf(w, "Pushed to origin (branch %s).\n", res.Sync.Branch)
	}
}

// usage is `misoku --help`: the commands, then the flags.
func usage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  misoku [flags]                           open the notes in a folder
  misoku export <note.md> [-o out.pdf]     export a note to PDF
  misoku sync init <url> [--dir <folder>]  set up git sync of a folder (run once per device)

Flags:
`)
	fs := flag.NewFlagSet("misoku", flag.ContinueOnError)
	defineFlags(fs)
	fs.SetOutput(w)
	fs.PrintDefaults()
}
