package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pwn27sbx/misoku/internal/tutor"
)

// tutorWorkspace handles the arguments of "misoku tutor [--route R] [--reset]":
// it installs the bundled lessons in tutor.Dir (the missing ones only, so the
// player's progress stays; --reset moves the folder to the Trash first),
// prepares the tutorial's git repository for :history when git is available,
// and returns that folder, the route it asked for and the migration notice.
func tutorWorkspace(args []string) (dir, route, notice string, err error) {
	fs := flag.NewFlagSet("tutor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	reset := fs.Bool("reset", false, "start the tutorial over from the bundled lessons")
	routeFlag := fs.String("route", "", "route to open (B, I, A or E)")
	if err := fs.Parse(args); err != nil {
		return "", "", "", err
	}
	if fs.NArg() > 0 {
		return "", "", "", fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if *routeFlag != "" {
		route = strings.ToUpper(*routeFlag)
		if r, ok := tutor.Default().RouteByID(route); !ok || len(r.Lessons) == 0 {
			return "", "", "", fmt.Errorf("unknown route %q (want B, I, A or E)", *routeFlag)
		}
	}
	dir = tutor.Dir()
	notice, err = tutor.InstallWithNotice(tutor.Bundled(), dir, *reset)
	if err != nil {
		return "", "", "", err
	}
	if gitAvailable() {
		if err := initTutorialGit(dir); err != nil {
			// The lessons work without :history; say why it is missing.
			fmt.Fprintf(os.Stderr, "misoku tutor: could not prepare the tutorial's git history: %v\n", err)
		}
	}
	return dir, route, notice, nil
}

// tutorUsage is the help of "misoku tutor".
const tutorUsage = "Usage: misoku tutor [--route B|I|A|E] [--reset]"
