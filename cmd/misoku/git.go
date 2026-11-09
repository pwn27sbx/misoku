package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// gitInitTimeout bounds the three git commands that prepare the tutorial's
// history: a slow or broken git never holds the tutorial.
const gitInitTimeout = 5 * time.Second

// gitAvailable reports whether :history can work in the tutorial.
func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// initTutorialGit makes the tutorial folder a git repository with one commit
// of the bundled lessons, so :history has something to show. The progress
// file is ignored. An existing repository is left alone, and every command
// has a time limit.
func initTutorialGit(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return nil
	}
	ignore := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(ignore); os.IsNotExist(err) {
		if err := os.WriteFile(ignore, []byte("/.misoku-tutor.json\n*.tmp\n"), 0o644); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitInitTimeout)
	defer cancel()
	steps := [][]string{
		{"init", "-q"},
		{"add", "-A"},
		{"-c", "user.name=Misoku", "-c", "user.email=tutorial@local", "commit", "-q", "-m", "Tutorial"},
	}
	for _, args := range steps {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return nil
}
