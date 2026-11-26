//go:build windows

package worker

import (
	"os/exec"
	"syscall"
)

// Windows cleanup kills only the worker; drivers must reap their descendants.
func childProcessAttr() *syscall.SysProcAttr {
	return nil
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
