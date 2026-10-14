//go:build !windows

package gitsync

import (
	"os/exec"
	"syscall"
)

// run runs cmd in a new session: it has no controlling terminal, so
// nothing it runs (ssh, gpg, a credential helper) can open /dev/tty. When
// cmd's context ends, the whole session's process group is killed,
// children included.
func run(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	return cmd.Run()
}
