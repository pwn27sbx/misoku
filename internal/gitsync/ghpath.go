package gitsync

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// executable is Misoku's own executable (os.Executable); tests replace it.
var executable = os.Executable

// lookGH finds the gh program name: on PATH first, then, for a bare name,
// next to Misoku's own executable. The installer puts gh there, in a
// folder a new Terminal may not have on its PATH yet.
func lookGH(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err == nil || filepath.Base(name) != name {
		return path, err
	}
	exe, xerr := executable()
	if xerr != nil {
		return "", err
	}
	dirs := []string{filepath.Dir(exe)}
	if real, rerr := filepath.EvalSymlinks(exe); rerr == nil && filepath.Dir(real) != dirs[0] {
		dirs = append(dirs, filepath.Dir(real))
	}
	for _, d := range dirs {
		if p, lerr := exec.LookPath(filepath.Join(d, name)); lerr == nil {
			return p, nil
		}
	}
	return "", err
}

// appleInstaller is the process of Apple's installer for the Command Line
// Tools, which bring git to a Mac.
const appleInstaller = "Install Command Line Developer Tools.app"

// appleInstallingGit reports whether Apple's installer for git (the
// Command Line Tools) is running; tests replace it.
var appleInstallingGit = func(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "pgrep", "-f", appleInstaller).Run() == nil
}
