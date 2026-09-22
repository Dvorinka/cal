//go:build windows

package main

import (
	"context"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// CREATE_NO_WINDOW: cal.exe is GUI-subsystem, so console-subsystem children
// (initdb, pg_ctl, postgres) would each get a fresh console window without it.
const createNoWindow = 0x08000000

type jobHandle = windows.Handle

func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}

// killWithParent assigns postgres to a Job Object with KILL_ON_JOB_CLOSE so a
// crashed cal.exe can never orphan a postmaster.
func killWithParent(cmd *exec.Cmd) (jobHandle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	var info windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	proc, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	defer windows.CloseHandle(proc)
	if err := windows.AssignProcessToJobObject(job, proc); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

func closeJob(h jobHandle) {
	if h != 0 {
		_ = windows.CloseHandle(h)
	}
}

// gracefulStop uses pg_ctl: Windows postgres has no POSIX signals — pg_ctl
// delivers the shutdown through the postmaster's own signal queue.
func gracefulStop(ctx context.Context, p *embeddedPG) error {
	cmd := exec.CommandContext(ctx, pgBin(p.binDir, "pg_ctl"),
		"stop", "-D", p.dataDir, "-m", "fast", "-w")
	cmd.Stdout = p.logFile
	cmd.Stderr = p.logFile
	hideConsole(cmd)
	return cmd.Run()
}

// pidIsPostmaster just checks liveness: pgkill's signal delivery is
// postgres-specific, so a foreign PID in postmaster.pid fails harmlessly.
func pidIsPostmaster(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = windows.CloseHandle(h)
	return true
}
