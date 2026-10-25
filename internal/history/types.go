package history

import (
	"time"
)

// HistoryEntry represents a single executed command in Shellix history
type HistoryEntry struct {
	ID        string        `json:"id"`
	Command   string        `json:"command"`
	Directory string        `json:"directory"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration"`
	ExitCode  int           `json:"exit_code"`
	Count     int           `json:"count"` // Frequency count for deduplicated view
}

// CommandFreq represents an aggregated command frequency statistic
type CommandFreq struct {
	Command   string        `json:"command"`
	Count     int           `json:"count"`
	Successes int           `json:"successes"`
	Failures  int           `json:"failures"`
	LastUsed  time.Time     `json:"last_used"`
	AvgTime   time.Duration `json:"avg_time"`
}

// HistoryStats summarizes historical execution metrics
type HistoryStats struct {
	TotalCommands  int           `json:"total_commands"`
	UniqueCommands int           `json:"unique_commands"`
	SuccessCount   int           `json:"success_count"`
	FailureCount   int           `json:"failure_count"`
	SuccessRate    float64       `json:"success_rate"`
	AvgDuration    time.Duration `json:"avg_duration"`
	TopCommands    []CommandFreq `json:"top_commands"`
}
