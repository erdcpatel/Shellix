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
	StepName string
	Duration time.Duration
	Success  bool
	Error    error
	Output   string
}
