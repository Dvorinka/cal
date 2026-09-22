//go:build linux

package main

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

type jobHandle = uintptr

// hideConsole is a no-op on Linux — no console windows exist to hide.
// Pdeathsig makes postgres exit (SIGTERM → smart shutdown, instant with no
// clients) if cal dies without calling stop().
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
}

func killWithParent(_ *exec.Cmd) (jobHandle, error) { return 0, nil }

func closeJob(_ jobHandle) {}

// gracefulStop sends SIGINT to the postmaster — Fast Shutdown.
func gracefulStop(_ context.Context, p *embeddedPG) error {
	return p.cmd.Process.Signal(os.Interrupt)
}

// pidIsPostmaster checks /proc so a recycled PID in postmaster.pid can't get
// an innocent process killed by pg_ctl's real POSIX signals.
func pidIsPostmaster(pid int) bool {
	if syscall.Kill(pid, 0) != nil {
		return false
	}
	comm, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/comm")
	return err == nil && strings.Contains(string(comm), "postgres")
}
