//go:build !windows && !darwin

package flow

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallDaemonService provides user service file generation on Linux
func InstallDaemonService(binaryPath string) error {
	bin := binaryPath
	if bin == "" {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		bin = exe
	}

	home, _ := os.UserHomeDir()
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	_ = os.MkdirAll(unitDir, 0755)

	unitPath := filepath.Join(unitDir, "shellix-scheduler.service")
	content := fmt.Sprintf(`[Unit]
Description=Shellix Flow Cron Scheduler
After=network.target

[Service]
Type=simple
ExecStart=%s flow schedule start
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
`, bin)

	return os.WriteFile(unitPath, []byte(content), 0644)
}

// UninstallDaemonService removes the systemd user service
func UninstallDaemonService() error {
	home, _ := os.UserHomeDir()
	unitPath := filepath.Join(home, ".config", "systemd", "user", "shellix-scheduler.service")
	return os.Remove(unitPath)
}

// DaemonServiceStatus returns status on non-Windows/darwin platforms
func DaemonServiceStatus() (string, error) {
	home, _ := os.UserHomeDir()
	unitPath := filepath.Join(home, ".config", "systemd", "user", "shellix-scheduler.service")
	if _, err := os.Stat(unitPath); err == nil {
		return "CONFIGURED (~/.config/systemd/user/shellix-scheduler.service)", nil
	}
	return "NOT_CONFIGURED", nil
}
