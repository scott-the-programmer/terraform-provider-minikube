//go:build !windows

package worker

import (
	"os/exec"
	"syscall"
)

// childProcessAttr opts worker children into their own process group so the
// parent can terminate driver-spawned descendants together with the worker.
func childProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup force-kills the worker and everything it spawned.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
