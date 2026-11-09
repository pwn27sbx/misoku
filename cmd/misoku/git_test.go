package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// E14: the tutorial folder becomes a git repository with one commit of the
// lessons, and the progress file stays out of it.
func TestInitTutorialGit(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "B01 Títulos.md"), []byte("# Nivel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".misoku-tutor.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // the second call must not add another commit
		if err := initTutorialGit(dir); err != nil {
			t.Fatal(err)
		}
	}
	out, err := exec.Command("git", "-C", dir, "log", "--oneline").Output()
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(strings.TrimSpace(string(out)), "\n") + 1; lines != 1 {
		t.Fatalf("the tutorial has %d commits, want 1:\n%s", lines, out)
	}
	files, err := exec.Command("git", "-C", dir, "-c", "core.quotePath=false", "ls-files").Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(files), "B01 Títulos.md") {
		t.Fatalf("the lesson is not committed:\n%s", files)
	}
	if strings.Contains(string(files), ".misoku-tutor.json") {
		t.Fatalf("the progress file was committed:\n%s", files)
	}
}
