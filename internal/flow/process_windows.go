//go:build windows

package flow

import (
	"syscall"
)

const stillActive = 259

// IsProcessAlive checks whether the process with given PID is currently active on Windows
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	const processQueryLimitedInformation = 0x1000
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)

	var exitCode uint32
	err = syscall.GetExitCodeProcess(h, &exitCode)
	if err != nil {
		return false
	}

	return exitCode == stillActive
}
