package flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// SendDesktopNotification triggers native desktop notifications on Windows and macOS
func SendDesktopNotification(title, message string, isSuccess bool) error {
	switch runtime.GOOS {
	case "windows":
		// Windows Notification via PowerShell System.Windows.Forms BalloonTip
		escapedTitle := strings.ReplaceAll(title, `"`, "`\"")
		escapedMsg := strings.ReplaceAll(message, `"`, "`\"")
		tipIcon := "[System.Windows.Forms.ToolTipIcon]::Info"
		if !isSuccess {
			tipIcon = "[System.Windows.Forms.ToolTipIcon]::Error"
		}

		psScript := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$balloon = New-Object System.Windows.Forms.NotifyIcon
$balloon.Icon = [System.Drawing.SystemIcons]::Information
$balloon.Visible = $true
$balloon.ShowBalloonTip(4000, "%s", "%s", %s)
Start-Sleep -Milliseconds 200
$balloon.Dispose()
`, escapedTitle, escapedMsg, tipIcon)

		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
		return cmd.Start() // Non-blocking

	case "darwin":
		// macOS Notification via osascript
		escapedTitle := strings.ReplaceAll(title, `"`, `\"`)
		escapedMsg := strings.ReplaceAll(message, `"`, `\"`)
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, escapedMsg, escapedTitle)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Start() // Non-blocking

	default:
		// Linux: notify-send if installed
		if path, err := exec.LookPath("notify-send"); err == nil {
			cmd := exec.Command(path, title, message)
			return cmd.Start()
		}
		return nil
	}
}

// SendWebhookNotification delivers a JSON payload to a configured webhook endpoint
func SendWebhookNotification(webhookURL, flowName string, success bool, duration time.Duration, errMsg string) error {
	if webhookURL == "" {
		return nil
	}

	payload := map[string]interface{}{
		"flow":      flowName,
		"success":   success,
		"duration":  duration.String(),
		"error":     errMsg,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook responded with HTTP %d", resp.StatusCode)
	}

	return nil
}
