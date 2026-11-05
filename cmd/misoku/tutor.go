package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/pwn27sbx/misoku/internal/tutor"
)

// tutorWorkspace handles the arguments of "misoku tutor [--reset]": it
// installs the bundled lessons in tutor.Dir (the missing ones only, so the
// player's progress stays; --reset moves the folder to the Trash first) and
// returns that folder for misoku to open.
func tutorWorkspace(args []string) (dir string, err error) {
	fs := flag.NewFlagSet("tutor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	reset := fs.Bool("reset", false, "start the tutorial over from the bundled lessons")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() > 0 {
		return "", fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	dir = tutor.Dir()
	if err := tutor.Install(tutor.Bundled(), dir, *reset); err != nil {
		return "", err
	}
	return dir, nil
}

// tutorUsage is the help of "misoku tutor".
const tutorUsage = "Usage: misoku tutor [--reset]"
