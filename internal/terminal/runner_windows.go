//go:build windows

package terminal

import (
	"fmt"
	"os"
	"os/exec"
)

// RunShell launches an interactive shell on Windows
func RunShell(customShell string) error {
	shell := customShell
	if shell == "" || shell == "auto" {
		shell = "powershell.exe"
	}

	cmd := exec.Command(shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("shell exited with error: %w", err)
	}
	return nil
}
