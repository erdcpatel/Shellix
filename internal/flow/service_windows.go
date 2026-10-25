//go:build windows

package flow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const TaskName = "ShellixFlowScheduler"

// InstallDaemonService registers Shellix Flow Scheduler in Windows Task Scheduler
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

	taskRun := fmt.Sprintf("\"%s\" flow schedule start", absBin)
	cmd := exec.Command("schtasks.exe", "/Create", "/SC", "ONLOGON", "/TN", TaskName, "/TR", taskRun, "/F")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed registering Windows scheduled task: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// UninstallDaemonService removes the Shellix task from Windows Task Scheduler
func UninstallDaemonService() error {
	cmd := exec.Command("schtasks.exe", "/Delete", "/TN", TaskName, "/F")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed deleting Windows scheduled task: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// DaemonServiceStatus queries the status of the Windows scheduled task
func DaemonServiceStatus() (string, error) {
	cmd := exec.Command("schtasks.exe", "/Query", "/TN", TaskName, "/FO", "LIST")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "NOT_INSTALLED", nil
	}
	return strings.TrimSpace(string(out)), nil
}
