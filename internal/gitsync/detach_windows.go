//go:build windows

package gitsync

import (
	"os/exec"
	"syscall"
)

// detach starts git without a console window of its own. Prompts are kept
// away by the environment (GCM_INTERACTIVE=never, GIT_TERMINAL_PROMPT=0);
// on timeout the process is killed and WaitDelay stops Wait from hanging on
// pipes held by surviving children.
func detach(cmd *exec.Cmd) {
	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP}
}
