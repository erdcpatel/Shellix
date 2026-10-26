//go:build !windows

package terminal

import (
	"os"
	"testing"
)

func TestUnixShellResolution(t *testing.T) {
	oldShell := os.Getenv("SHELL")
	defer func() { _ = os.Setenv("SHELL", oldShell) }()

	// 1. Explicit custom shell
	if got := resolveUnixShell("/bin/zsh"); got != "/bin/zsh" {
		t.Errorf("expected /bin/zsh, got: %s", got)
	}

	// 2. SHELL env fallback
	_ = os.Setenv("SHELL", "/bin/sh")
	if got := resolveUnixShell("auto"); got != "/bin/sh" {
		t.Errorf("expected /bin/sh from SHELL env, got: %s", got)
	}

	// 3. Empty SHELL fallback to /bin/bash
	_ = os.Setenv("SHELL", "")
	if got := resolveUnixShell(""); got != "/bin/bash" {
		t.Errorf("expected /bin/bash fallback, got: %s", got)
	}
}
