//go:build !windows && !linux

package main

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type jobHandle = uintptr

// hideConsole is a no-op off Windows; no Pdeathsig equivalent exists on
// darwin/*BSD — the stale-postmaster recovery in postgres.go covers orphans.
func hideConsole(_ *exec.Cmd) {}

func killWithParent(_ *exec.Cmd) (jobHandle, error) { return 0, nil }

func closeJob(_ jobHandle) {}

// gracefulStop sends SIGINT to the postmaster — Fast Shutdown.
func gracefulStop(_ context.Context, p *embeddedPG) error {
	return p.cmd.Process.Signal(os.Interrupt)
}

func pidIsPostmaster(pid int) bool {
	if err := exec.Command("kill", "-0", strconv.Itoa(pid)).Run(); err != nil {
		return false
	}
	out, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	return err == nil && strings.Contains(string(out), "postgres")
}
