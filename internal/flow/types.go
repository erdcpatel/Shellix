package flow

import (
	"time"
)

// FailurePolicy defines how to handle step failures
type FailurePolicy string

const (
	OnFailureAbort    FailurePolicy = "abort"
	OnFailureContinue FailurePolicy = "continue"
	OnFailureRetry    FailurePolicy = "retry"
)

// Step represents a single unit of execution within a Flow
type Step struct {
	Name        string            `yaml:"name"`
	Run         string            `yaml:"run"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	Timeout     string            `yaml:"timeout,omitempty"`
	TimeoutDur  time.Duration     `yaml:"-"`
	OnFailure   FailurePolicy     `yaml:"on_failure,omitempty"`
	Retries     int               `yaml:"retries,omitempty"`
	RetryDelay  string            `yaml:"retry_delay,omitempty"`
	RetryDur    time.Duration     `yaml:"-"`
	Parallel    bool              `yaml:"parallel,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	If          string            `yaml:"if,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
}

// Flow represents a declarative pipeline
type Flow struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Steps       []Step            `yaml:"steps"`
}

// ExecutionResult captures the output and status of a flow step
type ExecutionResult struct {
	StepName   string
	Duration   time.Duration
	Success    bool
	Skipped    bool
	Error      error
	Output     string
	IsParallel bool
}

// FlowEventType denotes state changes during workflow execution
type FlowEventType int

const (
	EventStepStart FlowEventType = iota
	EventStepDone
	EventFlowDone
)

// FlowEvent communicates real-time progress to TUI or external listeners
type FlowEvent struct {
	Type       FlowEventType
	StepIndex  int
	StepName   string
	Result     *ExecutionResult
	Duration   time.Duration
	Error      error
	IsParallel bool
}

// RunLogSummary holds high-level metadata for a persisted flow run
type RunLogSummary struct {
	RunID     string    `json:"run_id"`
	FlowName  string    `json:"flow_name"`
	StartTime time.Time `json:"start_time"`
	Duration  string    `json:"duration"`
	Success   bool      `json:"success"`
	LogPath   string    `json:"log_path"`
}
