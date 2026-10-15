//go:build windows

package gitsync

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// run runs cmd without a console window of its own, inside a job object.
// Prompts are kept away by the environment (GCM_INTERACTIVE=never,
// GIT_TERMINAL_PROMPT=0). Windows has no process groups to kill, so the
// job is what reaches git's children (ssh, a credential helper): when
// cmd's context ends the whole job is terminated, and closing the job
// (KILL_ON_JOB_CLOSE) kills whatever is left, also if Misoku dies first.
//
// git starts suspended and only resumes once it is in the job: a child
// started before the assignment would escape it. started, when not nil,
// gets git's pid before git runs.
func run(cmd *exec.Cmd, started func(pid int)) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("create job object: %w", err)
	}
	defer windows.CloseHandle(job)
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		return fmt.Errorf("set up job object: %w", err)
	}

	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNoWindow | syscall.CREATE_NEW_PROCESS_GROUP | windows.CREATE_SUSPENDED,
	}
	cmd.Cancel = func() error {
		windows.TerminateJobObject(job, 1)
		return cmd.Process.Kill() // in case git never made it into the job
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := uint32(cmd.Process.Pid)
	if started != nil {
		started(cmd.Process.Pid)
	}
	// If the job cannot be joined (an outer job that forbids nesting), git
	// still runs and is still killed on timeout; only its children could
	// outlive it, as they would without a job. Syncing beats refusing to.
	_ = assign(job, pid)
	if err := resume(pid); err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		return err
	}
	return cmd.Wait()
}

// assign puts the process pid into job.
func assign(job windows.Handle, pid uint32) error {
	proc, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(proc)
	return windows.AssignProcessToJobObject(job, proc)
}

// resume resumes the threads of a process started with CREATE_SUSPENDED
// (just its main thread). The standard library closes the thread handle
// CreateProcess returns, so the thread is found by listing them.
func resume(pid uint32) error {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return fmt.Errorf("list threads: %w", err)
	}
	defer windows.CloseHandle(snap)
	resumed := false
	var te windows.ThreadEntry32
	te.Size = uint32(unsafe.Sizeof(te))
	for err = windows.Thread32First(snap, &te); err == nil; err = windows.Thread32Next(snap, &te) {
		if te.OwnerProcessID != pid {
			continue
		}
		th, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, te.ThreadID)
		if err != nil {
			return fmt.Errorf("open git's thread: %w", err)
		}
		_, err = windows.ResumeThread(th)
		windows.CloseHandle(th)
		if err != nil {
			return fmt.Errorf("resume git: %w", err)
		}
		resumed = true
	}
	if !resumed {
		return fmt.Errorf("resume git: no thread found for process %d", pid)
	}
	return nil
}

// processAlive reports whether a process with id pid runs. When it cannot
// tell (no right to open it) it says yes: the safe side for recovery.
func processAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return !errors.Is(err, windows.ERROR_INVALID_PARAMETER) // no such process
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return true
	}
	const stillActive = 259
	return code == stillActive
}

// gitDiesWithMisoku says whether git dies when Misoku does: yes, closing the job object kills it.
const gitDiesWithMisoku = true
