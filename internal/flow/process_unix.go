//go:build !windows

package flow

import (
	"os"
	"syscall"
)

// IsProcessAlive checks whether the process with given PID is currently active on Unix
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}
