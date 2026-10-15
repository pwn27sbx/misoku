//go:build !windows

package gitsync

import (
	"errors"
	"os/exec"
	"syscall"
)

// run runs cmd in a new session: it has no controlling terminal, so
// nothing it runs (ssh, gpg, a credential helper) can open /dev/tty. When
// cmd's context ends, the whole session's process group is killed,
// children included. started, when not nil, gets git's pid as soon as it
// runs.
func run(cmd *exec.Cmd, started func(pid int)) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if started != nil {
		started(cmd.Process.Pid)
	}
	return cmd.Wait()
}

// processAlive reports whether a process with id pid runs. When it cannot
// tell (another user's process) it says yes: the safe side for recovery.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || !errors.Is(err, syscall.ESRCH)
}

// gitDiesWithMisoku says whether git dies when Misoku does: no, it runs in a session of its own and finishes its work.
const gitDiesWithMisoku = false
