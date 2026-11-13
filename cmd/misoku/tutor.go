package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pwn27sbx/misoku/internal/i18n"
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
	reset := fs.Bool("reset", false, i18n.T(i18n.CliTutorFlagReset))
	routeFlag := fs.String("route", "", i18n.T(i18n.CliTutorFlagRoute))
	if err := fs.Parse(args); err != nil {
		return "", "", "", err
	}
	if fs.NArg() > 0 {
		return "", "", "", errors.New(i18n.T(i18n.CliTutorUnexpected, fs.Arg(0)))
	}
	routeSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "route" {
			routeSet = true
		}
	})
	if routeSet && strings.TrimSpace(*routeFlag) == "" {
		return "", "", "", errors.New(i18n.T(i18n.CliTutorRouteNeeds))
	}
	if *routeFlag != "" {
		route = strings.ToUpper(*routeFlag)
		if r, ok := tutor.Default().RouteByID(route); !ok || len(r.Lessons) == 0 {
			return "", "", "", errors.New(i18n.T(i18n.CliTutorRouteBad, *routeFlag))
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
			fmt.Fprintln(os.Stderr, "misoku tutor: "+i18n.T(i18n.CliTutorGitFailed, err))
		}
	}
	return dir, route, notice, nil
}

// tutorialProgress reads the tutorial's progress. A malformed or newer file
// is reported and ignored: Load returns an empty store that is safe to use,
// so the tutorial still opens and only records new progress (this is the old
// empty-store recovery path).
func tutorialProgress(dir string) *tutor.Store {
	store, err := tutor.Load(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "misoku tutor: "+i18n.T(i18n.CliTutorBadProgress, err))
	}
	return store
}
