package terminal

import (
	"fmt"
	"strings"
)

// isPowerShell checks if the given shell path corresponds to PowerShell (pwsh or powershell)
func isPowerShell(shell string) bool {
	clean := shell
	if idx := strings.LastIndexAny(clean, `/\`); idx != -1 {
		clean = clean[idx+1:]
	}
	clean = strings.ToLower(clean)
	clean = strings.TrimSuffix(clean, ".exe")
	return clean == "pwsh" || clean == "powershell"
}

// BuildPowerShellProfileScript builds the startup commands for PowerShell sessions,
// defining superpowers aliases/functions and a Git-aware prompt.
func BuildPowerShellProfileScript(exePath string) string {
	escapedExe := strings.ReplaceAll(exePath, "'", "''")
	return fmt.Sprintf(`[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; `+
		`$env:SHELLIX_BIN = '%s'; `+
		`if (Test-Path Alias:\g) { Remove-Item Alias:\g -Force -ErrorAction SilentlyContinue }; `+
		`if (Test-Path Alias:\st) { Remove-Item Alias:\st -Force -ErrorAction SilentlyContinue }; `+
		`if (Test-Path Alias:\sx) { Remove-Item Alias:\sx -Force -ErrorAction SilentlyContinue }; `+
		`if (Test-Path Alias:\slx) { Remove-Item Alias:\slx -Force -ErrorAction SilentlyContinue }; `+
		`function global:g { & $env:SHELLIX_BIN g @args }; `+
		`function global:st { & $env:SHELLIX_BIN status @args }; `+
		`function global:sx { & $env:SHELLIX_BIN @args }; `+
		`function global:slx { & $env:SHELLIX_BIN @args }; `+
		`function global:prompt { `+
		`$origExit = $LASTEXITCODE; `+
		`$b = ''; `+
		`if (Get-Command git -ErrorAction SilentlyContinue) { `+
		`$out = $(git rev-parse --abbrev-ref HEAD 2>$null); `+
		`if ($out) { `+
		`$out = $out.Trim(); `+
		`if ($out -eq 'HEAD') { `+
		`$short = $(git rev-parse --short HEAD 2>$null); `+
		`if ($short) { $out = $short.Trim() }; `+
		`}; `+
		`if ($out) { $b = ' [' + $out + ']' }; `+
		`}; `+
		`}; `+
		`$global:LASTEXITCODE = $origExit; `+
		`'🐚 Shellix' + $b + ' PS ' + $($executionContext.SessionState.Path.CurrentLocation) + '> '; `+
		`}`, escapedExe)
}

// buildShellArgs returns extra command-line flags when launching PowerShell
func buildShellArgs(shell, exePath string) []string {
	if isPowerShell(shell) {
		return []string{
			"-NoLogo",
			"-NoExit",
			"-ExecutionPolicy", "Bypass",
			"-Command", BuildPowerShellProfileScript(exePath),
		}
	}
	return nil
}
