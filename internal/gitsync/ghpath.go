package gitsync

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// executable is Misoku's own executable (os.Executable); tests replace it.
var executable = os.Executable

// runtimeGOOS is the system the git lookup behaves as; tests replace it.
var runtimeGOOS = runtime.GOOS

// lookGH finds the gh program name: on PATH first, then, for a bare name,
// next to Misoku's own executable. The installer puts gh there, in a
// folder a new Terminal may not have on its PATH yet.
func lookGH(name string) (string, error) {
	return lookFirst(name, func() []string {
		exe, err := executable()
		if err != nil {
			return nil
		}
		files := []string{filepath.Join(filepath.Dir(exe), name)}
		if real, rerr := filepath.EvalSymlinks(exe); rerr == nil && filepath.Dir(real) != filepath.Dir(exe) {
			files = append(files, filepath.Join(filepath.Dir(real), name))
		}
		return files
	})
}

// gitProgram is the git that Runner starts: the one on PATH, or on Windows
// Git for Windows' own copy in Program Files. The installer adds git with
// winget, whose new PATH does not reach a window that is already open.
// When there is none it is plain "git", so the error stays exec's
// familiar "executable file not found".
func gitProgram() string {
	if runtimeGOOS != "windows" {
		// exec resolves the bare name on PATH, as it always has.
		return "git"
	}
	path, err := lookFirst("git", func() []string {
		if os.Getenv("ProgramFiles") == "" {
			return nil
		}
		return []string{filepath.Join(os.Getenv("ProgramFiles"), "Git", "cmd", "git.exe")}
	})
	if err != nil {
		return "git"
	}
	return path
}

// lookFirst finds the program name on PATH and, when it is a bare name
// that is not there, takes the first of the files listed by fallbacks
// that is a program. The error is the one from PATH.
func lookFirst(name string, fallbacks func() []string) (string, error) {
	path, err := exec.LookPath(name)
	if err == nil || filepath.Base(name) != name {
		return path, err
	}
	for _, f := range fallbacks() {
		if p, lerr := exec.LookPath(f); lerr == nil {
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
