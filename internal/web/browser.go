package web

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser launches the OS default web browser to the specified URL
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows cmd start requires empty title quotes if URL has special chars
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		// Linux / BSD
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed opening browser: %w", err)
	}
	return nil
}
