//go:build darwin

package flow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const LaunchAgentLabel = "com.shellix.flowscheduler"

func getPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", LaunchAgentLabel+".plist"), nil
}

// InstallDaemonService registers Shellix Flow Scheduler as a macOS LaunchAgent
func InstallDaemonService(binaryPath string) error {
	bin := binaryPath
	if bin == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("unable to determine shellix executable path: %w", err)
		}
		bin = exe
	}

	absBin, err := filepath.Abs(bin)
	if err != nil {
		absBin = bin
	}

	plistPath, err := getPlistPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(plistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>flow</string>
        <string>schedule</string>
        <string>start</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>`, LaunchAgentLabel, absBin)

	if err := os.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed writing plist: %w", err)
	}

	// Unload first if previously running
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	cmd := exec.Command("launchctl", "load", plistPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// UninstallDaemonService removes and unloads the macOS LaunchAgent
func UninstallDaemonService() error {
	plistPath, err := getPlistPath()
	if err != nil {
		return err
	}

	_ = exec.Command("launchctl", "unload", plistPath).Run()
	_ = os.Remove(plistPath)
	return nil
}

// DaemonServiceStatus checks if the LaunchAgent is registered
func DaemonServiceStatus() (string, error) {
	cmd := exec.Command("launchctl", "list", LaunchAgentLabel)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "NOT_RUNNING", nil
	}
	return "RUNNING: " + strings.TrimSpace(string(out)), nil
}
