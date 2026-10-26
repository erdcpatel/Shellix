//go:build windows

package terminal

import (
	"testing"
)

func TestWindowsShellResolution(t *testing.T) {
	shell := findBestWindowsShell("auto")
	if shell == "" {
		t.Errorf("expected non-empty windows shell")
	}
	custom := findBestWindowsShell("cmd.exe")
	if custom != "cmd.exe" {
		t.Errorf("expected cmd.exe, got: %s", custom)
	}
}
