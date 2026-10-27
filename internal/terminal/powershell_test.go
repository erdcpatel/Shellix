package terminal

import (
	"strings"
	"testing"
)

func TestIsPowerShell(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"powershell", true},
		{"powershell.exe", true},
		{"pwsh", true},
		{"pwsh.exe", true},
		{`C:\Program Files\PowerShell\7\pwsh.exe`, true},
		{`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, true},
		{"/usr/local/bin/pwsh", true},
		{"cmd.exe", false},
		{"cmd", false},
		{"/bin/bash", false},
		{"/bin/zsh", false},
		{"fish", false},
	}

	for _, tc := range testCases {
		got := isPowerShell(tc.input)
		if got != tc.expected {
			t.Errorf("isPowerShell(%q) = %v; expected %v", tc.input, got, tc.expected)
		}
	}
}

func TestBuildPowerShellProfileScript(t *testing.T) {
	exePath := `C:\Shellix Tools\shellix.exe`
	script := BuildPowerShellProfileScript(exePath)

	requiredSnippets := []string{
		"[Console]::OutputEncoding = [System.Text.Encoding]::UTF8",
		"$env:SHELLIX_BIN = 'C:\\Shellix Tools\\shellix.exe'",
		"Remove-Item Alias:\\g",
		"Remove-Item Alias:\\st",
		"Remove-Item Alias:\\sx",
		"Remove-Item Alias:\\slx",
		"function global:g { & $env:SHELLIX_BIN g @args }",
		"function global:st { & $env:SHELLIX_BIN status @args }",
		"function global:sx { & $env:SHELLIX_BIN @args }",
		"function global:slx { & $env:SHELLIX_BIN @args }",
		"function global:prompt",
		"$origExit = $LASTEXITCODE",
		"$global:LASTEXITCODE = $origExit",
		".Trim()",
		"🐚 Shellix",
		"git rev-parse --abbrev-ref HEAD",
		"PS ' + $($executionContext.SessionState.Path.CurrentLocation) + '> '",
	}

	for _, snippet := range requiredSnippets {
		if !strings.Contains(script, snippet) {
			t.Errorf("expected script to contain %q, but got:\n%s", snippet, script)
		}
	}

	// Verify absence of double-quotes to protect Windows CLI argument parsing
	if strings.Contains(script, `"`) {
		t.Errorf("expected script to avoid double-quotes for bulletproof Windows CLI parsing, but found double-quotes:\n%s", script)
	}

	t.Run("quotes escaping", func(t *testing.T) {
		pathWithQuotes := `C:\User's Folder\shellix.exe`
		escapedScript := BuildPowerShellProfileScript(pathWithQuotes)
		expectedEscaped := `C:\User''s Folder\shellix.exe`
		if !strings.Contains(escapedScript, expectedEscaped) {
			t.Errorf("expected single quote to be escaped as '', got:\n%s", escapedScript)
		}
	})
}

func TestBuildShellArgs(t *testing.T) {
	exePath := `C:\bin\shellix.exe`

	t.Run("powershell args", func(t *testing.T) {
		args := buildShellArgs("powershell.exe", exePath)
		if len(args) < 6 {
			t.Fatalf("expected at least 6 arguments, got: %v", args)
		}

		joined := strings.Join(args, " ")
		if !strings.Contains(joined, "-NoExit") {
			t.Errorf("expected -NoExit in args: %v", args)
		}
		if !strings.Contains(joined, "-Command") {
			t.Errorf("expected -Command in args: %v", args)
		}
		if !strings.Contains(joined, "-ExecutionPolicy Bypass") {
			t.Errorf("expected -ExecutionPolicy Bypass in args: %v", args)
		}
		if !strings.Contains(joined, "SHELLIX_BIN") {
			t.Errorf("expected SHELLIX_BIN in args: %v", args)
		}
	})

	t.Run("pwsh args", func(t *testing.T) {
		args := buildShellArgs("pwsh.exe", exePath)
		if len(args) == 0 {
			t.Errorf("expected non-empty args for pwsh.exe")
		}
	})

	t.Run("cmd.exe args are nil", func(t *testing.T) {
		args := buildShellArgs("cmd.exe", exePath)
		if args != nil {
			t.Errorf("expected nil args for cmd.exe, got: %v", args)
		}
	})
}
