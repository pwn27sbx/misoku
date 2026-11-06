package main

import "github.com/pwn27sbx/misoku/internal/config"

// learnCheck is everything that decides whether misoku offers the first-run
// dialog instead of going straight to the home screen.
type learnCheck struct {
	Choice workspaceChoice
	// TutorDir is the tutorial vault, when misoku tutor is played: a
	// tutorial session never offers the dialog, so sessions cannot nest.
	TutorDir string
	// ConfigPath is where the answer is remembered ("" = nowhere).
	ConfigPath string
	// ConfigErr is the error reading the config, if any.
	ConfigErr error
	Config    config.Config
	// Terminal: someone is there to answer (stdin and stdout are a
	// terminal), as the welcome needs.
	Terminal bool
}

// learnDue reports whether misoku offers the first-run dialog: only with
// someone to answer it, a config it can write the answer to, and a folder
// misoku found on its own (notes_dir, $MISOKU_WORKSPACE or the terminal's).
// A folder the user gave (--dir, misoku <folder>) never sees it, and its
// flag stays untouched.
func learnDue(c learnCheck) bool {
	if !c.Terminal || c.TutorDir != "" || c.ConfigErr != nil || c.ConfigPath == "" || c.Config.LearnOffered {
		return false
	}
	switch c.Choice.Source {
	case fromNotesDir, fromCwd, fromEnv:
		return true
	}
	return false
}
