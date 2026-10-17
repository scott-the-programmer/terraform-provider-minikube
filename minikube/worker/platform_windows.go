//go:build windows

package worker

import (
	"os/exec"
	"syscall"
)

// childProcessAttr returns no attributes on Windows; process-group kill is not
// available here, so descendant cleanup depends on the driver's own reaping.
func childProcessAttr() *syscall.SysProcAttr {
	return nil
}

// killProcessGroup falls back to terminating the worker process itself.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
