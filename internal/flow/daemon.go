package flow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DaemonInfo records runtime state of the active scheduler background runner
type DaemonInfo struct {
	PID           int       `json:"pid"`
	StartedAt     time.Time `json:"started_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	Hostname      string    `json:"hostname"`
	Version       string    `json:"version,omitempty"`
}

func getPIDFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".shellix", "scheduler.pid")
}

// GetSchedulerLogPath returns the canonical path for scheduler daemon logs
func GetSchedulerLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".shellix", "logs", "scheduler.log")
}

// AcquireDaemonLock checks for running instances, cleans stale locks, and registers PID
func AcquireDaemonLock() (*DaemonInfo, error) {
	pidPath := getPIDFilePath()
	dir := filepath.Dir(pidPath)
	_ = os.MkdirAll(dir, 0700)

	if data, err := os.ReadFile(pidPath); err == nil && len(data) > 0 {
		var existing DaemonInfo
		if jsonErr := json.Unmarshal(data, &existing); jsonErr == nil && existing.PID > 0 {
			if IsProcessAlive(existing.PID) {
				return nil, fmt.Errorf("scheduler daemon is already running (PID: %d, started at %s)",
					existing.PID, existing.StartedAt.Format("2006-01-02 15:04:05"))
			}
			// Process is dead, stale lock can be safely cleaned up
			_ = os.Remove(pidPath)
		}
	}

	hostname, _ := os.Hostname()
	info := DaemonInfo{
		PID:           os.Getpid(),
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Hostname:      hostname,
	}

	raw, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(pidPath, raw, 0600); err != nil {
		return nil, fmt.Errorf("failed to write scheduler PID file: %w", err)
	}

	return &info, nil
}

// ReleaseDaemonLock removes the PID file when scheduler shuts down
func ReleaseDaemonLock() {
	_ = os.Remove(getPIDFilePath())
}

// UpdateHeartbeat refreshes the last active timestamp on the PID file
func UpdateHeartbeat() {
	pidPath := getPIDFilePath()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}

	var info DaemonInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return
	}

	info.LastHeartbeat = time.Now()
	raw, err := json.MarshalIndent(info, "", "  ")
	if err == nil {
		_ = os.WriteFile(pidPath, raw, 0600)
	}
}

// GetDaemonStatus checks whether a background scheduler is currently alive
func GetDaemonStatus() (*DaemonInfo, bool, error) {
	pidPath := getPIDFilePath()
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		return nil, false, nil
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		return nil, false, err
	}

	var info DaemonInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, false, fmt.Errorf("invalid PID lock format: %w", err)
	}

	alive := IsProcessAlive(info.PID)
	return &info, alive, nil
}

// SchedulerLogger writes structured timestamps to ~/.shellix/logs/scheduler.log and stdout
type SchedulerLogger struct {
	file       *os.File
	alsoStdout bool
	mu         sync.Mutex
}

// NewSchedulerLogger opens the log file in ~/.shellix/logs/
func NewSchedulerLogger(alsoStdout bool) (*SchedulerLogger, error) {
	logPath := GetSchedulerLogPath()
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, fmt.Errorf("failed creating log directory: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed opening scheduler log file: %w", err)
	}

	return &SchedulerLogger{
		file:       f,
		alsoStdout: alsoStdout,
	}, nil
}

func (l *SchedulerLogger) log(level, tag, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("[%s] [%-7s] [%s] %s\n", timestamp, level, tag, msg)

	if l.file != nil {
		_, _ = l.file.WriteString(line)
	}

	if l.alsoStdout {
		fmt.Print(line)
	}
}

func (l *SchedulerLogger) Info(tag, format string, args ...interface{}) {
	l.log("INFO", tag, format, args...)
}

func (l *SchedulerLogger) Success(tag, format string, args ...interface{}) {
	l.log("SUCCESS", tag, format, args...)
}

func (l *SchedulerLogger) Warn(tag, format string, args ...interface{}) {
	l.log("WARN", tag, format, args...)
}

func (l *SchedulerLogger) Error(tag, format string, args ...interface{}) {
	l.log("ERROR", tag, format, args...)
}

func (l *SchedulerLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}
