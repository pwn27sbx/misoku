package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/pwn27sbx/misoku/internal/config"
)

// workspaceSource says where the folder misoku opens came from.
type workspaceSource int

const (
	fromFlag        workspaceSource = iota + 1 // --dir
	fromArg                                    // misoku <folder>
	fromEnv                                    // $MISOKU_WORKSPACE
	fromNotesDir                               // notes_dir in the config
	fromCwd                                    // the folder misoku runs in
	notesDirMissing                            // notes_dir is set, but its folder is gone
)

// workspaceInput is everything that decides which folder misoku opens.
type workspaceInput struct {
	Flag   string // --dir
	Arg    string // the folder argument
	Env    string // $MISOKU_WORKSPACE
	Config config.Config
	Getwd  func() (string, error)
}

// workspaceChoice is the folder to open and why. With notesDirMissing,
// Dir is the remembered folder that no longer exists: it must not be
// opened (opening creates it), but explained.
type workspaceChoice struct {
	Dir    string
	Source workspaceSource
	// Warning, when set, is a notes_dir that could not be used and was
	// skipped.
	Warning string
}

// pickWorkspace chooses the folder misoku opens: --dir or the folder
// argument (the caller refuses both at once), then $MISOKU_WORKSPACE, then
// notes_dir, then the current folder ("." if that cannot be read). --dir,
// the argument and the variable are used as written, as before notes_dir
// existed.
func pickWorkspace(in workspaceInput) workspaceChoice {
	switch {
	case in.Flag != "":
		return workspaceChoice{Dir: in.Flag, Source: fromFlag}
	case in.Arg != "":
		return workspaceChoice{Dir: in.Arg, Source: fromArg}
	case in.Env != "":
		return workspaceChoice{Dir: in.Env, Source: fromEnv}
	}
	var warning string
	notes, err := in.Config.NotesFolder()
	switch {
	case err != nil:
		warning = fmt.Sprintf("Ignoring %v", err)
	case notes != "":
		// Only a folder that is surely gone counts as missing; any other
		// trouble is reported when the folder is opened.
		if _, err := os.Stat(notes); errors.Is(err, fs.ErrNotExist) {
			return workspaceChoice{Dir: notes, Source: notesDirMissing}
		}
		return workspaceChoice{Dir: notes, Source: fromNotesDir}
	}
	cwd, err := in.Getwd()
	if err != nil {
		cwd = "."
	}
	return workspaceChoice{Dir: cwd, Source: fromCwd, Warning: warning}
}

// errUsage means the command line is not one misoku understands; the
// caller prints the usage.
var errUsage = errors.New("usage")

// parseOpenArgs reads `misoku [folder] [flags]` from args, with the flags
// allowed before and after the folder. More than one folder, or a folder
// together with --dir, is errUsage.
func parseOpenArgs(fs *flag.FlagSet, args []string) (*options, string, error) {
	opts := defineFlags(fs)
	folders, err := positionals(fs, args)
	if err != nil {
		return nil, "", err
	}
	switch {
	case len(folders) > 1, len(folders) == 1 && opts.dir != "":
		return nil, "", errUsage
	case len(folders) == 1:
		return opts, folders[0], nil
	}
	return opts, "", nil
}

// missingNotesMessage explains a remembered notes folder that is gone.
func missingNotesMessage(dir string) string {
	return fmt.Sprintf("Your notes folder %s is missing; open another with misoku <folder>", dir)
}
