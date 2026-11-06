package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pwn27sbx/misoku/internal/config"
)

// S7: the first-run dialog is offered only for a folder misoku found on its
// own, with a terminal and a config it can write the answer to.
func TestLearnDue(t *testing.T) {
	base := learnCheck{
		Choice:     workspaceChoice{Dir: "/notes", Source: fromNotesDir},
		ConfigPath: "/config/config.json",
		Terminal:   true,
	}
	for _, source := range []workspaceSource{fromNotesDir, fromCwd, fromEnv} {
		c := base
		c.Choice.Source = source
		if !learnDue(c) {
			t.Errorf("source %v: the dialog should be offered", source)
		}
	}

	cases := []struct {
		name  string
		check func(*learnCheck)
	}{
		{"--dir", func(c *learnCheck) { c.Choice.Source = fromFlag }},
		{"a folder argument", func(c *learnCheck) { c.Choice.Source = fromArg }},
		{"a missing notes folder", func(c *learnCheck) { c.Choice.Source = notesDirMissing }},
		{"no terminal", func(c *learnCheck) { c.Terminal = false }},
		{"no config file", func(c *learnCheck) { c.ConfigPath = "" }},
		{"an unreadable config", func(c *learnCheck) { c.ConfigErr = errors.New("boom") }},
		{"already answered", func(c *learnCheck) { c.Config.LearnOffered = true }},
		{"a tutorial session", func(c *learnCheck) { c.TutorDir = "/config/tutor" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			tc.check(&c)
			if learnDue(c) {
				t.Errorf("%s: the dialog should not be offered", tc.name)
			}
		})
	}
}

// S7: --dir shows nothing and leaves the flag alone.
func TestLearnGateLeavesTheFlagForAGivenFolder(t *testing.T) {
	fakeConfigHome(t)
	cfgPath := config.DefaultPath()
	check := learnCheck{
		Choice:     workspaceChoice{Dir: t.TempDir(), Source: fromFlag},
		ConfigPath: cfgPath,
		Terminal:   true,
	}
	if learnDue(check) {
		t.Fatal("--dir must not offer the dialog")
	}
	if _, err := os.Stat(cfgPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("-dir wrote the config: %v", err)
	}
	// The tutorial never offers it either, and a config that already holds
	// the answer stops it for good.
	check.Choice.Source = fromNotesDir
	check.TutorDir = filepath.Join(filepath.Dir(cfgPath), "tutor")
	if learnDue(check) {
		t.Fatal("misoku tutor must not offer the dialog")
	}
	check.TutorDir = ""
	check.Config.LearnOffered = true
	if learnDue(check) {
		t.Fatal("an answered dialog must not be offered again")
	}
}
