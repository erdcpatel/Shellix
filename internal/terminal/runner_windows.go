//go:build windows

package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

const (
	enableVirtualTerminalProcessing = 0x0004
	enableProcessedOutput           = 0x0001
	enableWrapAtEOLOutput           = 0x0002
	enableVirtualTerminalInput      = 0x0200
)

// enableVTProcessing enables ANSI and 24-bit color sequence processing in Windows Console Output
func enableVTProcessing(h syscall.Handle) func() {
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return func() {}
	}
	newMode := mode | enableVirtualTerminalProcessing | enableProcessedOutput | enableWrapAtEOLOutput
	procSetConsoleMode.Call(uintptr(h), uintptr(newMode))
	return func() {
		procSetConsoleMode.Call(uintptr(h), uintptr(mode))
	}
}

// enableVTInput enables Virtual Terminal Input on Windows Console Stdin (arrow keys, mouse, xterm keys)
func enableVTInput(h syscall.Handle) func() {
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return func() {}
	}
	newMode := mode | enableVirtualTerminalInput
	procSetConsoleMode.Call(uintptr(h), uintptr(newMode))
	return func() {
		procSetConsoleMode.Call(uintptr(h), uintptr(mode))
	}
}

// findBestWindowsShell discovers the highest-fidelity shell available
func findBestWindowsShell(customShell string) string {
	if customShell != "" && customShell != "auto" {
		return customShell
	}
	// 1. Prefer modern PowerShell 7+ (pwsh.exe)
	if p, err := exec.LookPath("pwsh.exe"); err == nil {
		return p
	}
	// 2. Windows PowerShell 5.1
	if p, err := exec.LookPath("powershell.exe"); err == nil {
		return p
	}
	// 3. Fallback to cmd.exe
	if p, err := exec.LookPath("cmd.exe"); err == nil {
		return p
	}
	return "powershell.exe"
}

// RunShell launches an interactive shell on Windows with Virtual Terminal Processing enabled
func RunShell(customShell string) error {
	shell := findBestWindowsShell(customShell)

	// Enable Virtual Terminal Processing on stdout and stderr
	restoreOut := enableVTProcessing(syscall.Handle(os.Stdout.Fd()))
	defer restoreOut()
	restoreErr := enableVTProcessing(syscall.Handle(os.Stderr.Fd()))
	defer restoreErr()
	restoreIn := enableVTInput(syscall.Handle(os.Stdin.Fd()))
	defer restoreIn()

	cmd := exec.Command(shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Subshell exited cleanly with non-zero exit code (e.g. exit 1); reap gracefully
			return nil
		}
		return fmt.Errorf("shell exited with error: %w", err)
	}
	return nil
}
