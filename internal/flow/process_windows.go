//go:build windows

package flow

import (
	"syscall"
)

const (
	stillActive = 259
	waitTimeout = 0x00000102
	waitObject0 = 0x00000000
)

var (
	kernel32Dll             = syscall.NewLazyDLL("kernel32.dll")
	procWaitForSingleObject = kernel32Dll.NewProc("WaitForSingleObject")
)

// IsProcessAlive checks whether the process with given PID is currently active on Windows
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	const processQueryLimitedInformation = 0x1000
	const synchronize = 0x00100000

	// Request SYNCHRONIZE along with query permissions to support WaitForSingleObject
	h, err := syscall.OpenProcess(processQueryLimitedInformation|synchronize, false, uint32(pid))
	if err != nil {
		// If access is denied, the process exists and is active (e.g. running under another user or elevated)
		if errno, ok := err.(syscall.Errno); ok && errno == syscall.ERROR_ACCESS_DENIED {
			return true
		}
		// Fallback retry with just limited info permission
		h, err = syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
		if err != nil {
			if errno, ok := err.(syscall.Errno); ok && errno == syscall.ERROR_ACCESS_DENIED {
				return true
			}
			return false
		}
	}
	defer syscall.CloseHandle(h)

	// WaitForSingleObject with 0ms timeout: returns WAIT_TIMEOUT if still running, WAIT_OBJECT_0 if terminated
	ret, _, _ := procWaitForSingleObject.Call(uintptr(h), 0)
	if ret == waitTimeout {
		return true
	}
	if ret == waitObject0 {
		return false
	}

	// Secondary fallback to GetExitCodeProcess if WaitForSingleObject was ambiguous
	var exitCode uint32
	if err := syscall.GetExitCodeProcess(h, &exitCode); err == nil {
		return exitCode == stillActive
	}

	return false
}
