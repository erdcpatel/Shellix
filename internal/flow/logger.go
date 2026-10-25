package flow

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// FlowLogger records comprehensive, audit-ready logs for every flow run
type FlowLogger struct {
	FlowName  string
	RunID     string
	StartTime time.Time
	LogFile   *os.File
	LogPath   string
}

// GetLogDir returns ~/.shellix/logs/flows
func GetLogDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".shellix", "logs", "flows")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// NewFlowLogger initializes a new log file for the specified flow
func NewFlowLogger(flowName string) (*FlowLogger, error) {
	dir, err := GetLogDir()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	runID := now.Format("20060102-150405")
	cleanName := strings.ReplaceAll(flowName, " ", "_")
	fileName := fmt.Sprintf("%s-%s.log", cleanName, runID)
	logPath := filepath.Join(dir, fileName)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	hostname, _ := os.Hostname()

	fl := &FlowLogger{
		FlowName:  flowName,
		RunID:     runID,
		StartTime: now,
		LogFile:   f,
		LogPath:   logPath,
	}

	header := fmt.Sprintf(`================================================================================
SHELLIX FLOW EXECUTION AUDIT LOG
Flow Name   : %s
Run ID      : %s
Timestamp   : %s
Host        : %s (%s/%s)
================================================================================

`, flowName, runID, now.Format(time.RFC3339), hostname, runtime.GOOS, runtime.GOARCH)

	_, _ = f.WriteString(header)
	return fl, nil
}

// LogStep writes detailed step telemetry and captured output
func (fl *FlowLogger) LogStep(step *Step, res ExecutionResult) {
	if fl.LogFile == nil {
		return
	}

	statusStr := "SUCCESS"
	if !res.Success {
		statusStr = fmt.Sprintf("FAILED (%v)", res.Error)
	}

	parallelStr := "Sequential"
	if res.IsParallel {
		parallelStr = "Parallel (Concurrent Worker)"
	}

	entry := fmt.Sprintf(`--------------------------------------------------------------------------------
STEP: %s [%s]
Command    : %s
Execution  : %s
Duration   : %s
Status     : %s
--- Output ---
%s
--------------

`, step.Name, statusStr, step.Run, parallelStr, res.Duration, statusStr, res.Output)

	_, _ = fl.LogFile.WriteString(entry)
}

// Close finalizes the execution summary header and closes the file
func (fl *FlowLogger) Close(success bool, totalDuration time.Duration) string {
	if fl.LogFile == nil {
		return fl.LogPath
	}

	finalStatus := "COMPLETED SUCCESSFULLY"
	if !success {
		finalStatus = "ABORTED WITH ERROR"
	}

	footer := fmt.Sprintf(`================================================================================
FINAL OUTCOME: %s
Total Duration: %s
Log File: %s
================================================================================
`, finalStatus, totalDuration, fl.LogPath)

	_, _ = fl.LogFile.WriteString(footer)
	_ = fl.LogFile.Close()
	return fl.LogPath
}

// ListLogs returns all previous log files sorted newest first
func ListLogs() ([]string, error) {
	dir, err := GetLogDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var logs []string
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			logs = append(logs, filepath.Join(dir, e.Name()))
		}
	}
	return logs, nil
}
