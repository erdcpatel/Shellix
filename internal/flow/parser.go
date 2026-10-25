package flow

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ParseFile reads and validates a flow YAML file
func ParseFile(path string) (*Flow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read flow file %s: %w", path, err)
	}

	return Parse(data)
}

// Parse unmarshals and validates flow YAML content
func Parse(data []byte) (*Flow, error) {
	var f Flow
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("invalid flow YAML: %w", err)
	}

	if f.Name == "" {
		return nil, fmt.Errorf("flow must have a 'name' field")
	}

	if len(f.Steps) == 0 {
		return nil, fmt.Errorf("flow '%s' contains no steps", f.Name)
	}

	seenNames := make(map[string]bool)
	for i := range f.Steps {
		step := &f.Steps[i]
		if step.Name == "" {
			step.Name = fmt.Sprintf("step-%d", i+1)
		}
		if seenNames[step.Name] {
			return nil, fmt.Errorf("duplicate step name '%s' in flow '%s' (step names must be unique)", step.Name, f.Name)
		}
		seenNames[step.Name] = true

		if step.Run == "" {
			return nil, fmt.Errorf("step '%s' must define a 'run' command", step.Name)
		}

		if step.OnFailure == "" {
			step.OnFailure = OnFailureAbort
		}

		if step.Timeout != "" {
			d, err := time.ParseDuration(step.Timeout)
			if err != nil {
				return nil, fmt.Errorf("invalid timeout '%s' in step '%s': %w", step.Timeout, step.Name, err)
			}
			step.TimeoutDur = d
		}

		if step.RetryDelay != "" {
			d, err := time.ParseDuration(step.RetryDelay)
			if err != nil {
				return nil, fmt.Errorf("invalid retry_delay '%s' in step '%s': %w", step.RetryDelay, step.Name, err)
			}
			step.RetryDur = d
		} else if step.Retries > 0 {
			step.RetryDur = time.Second
		}
	}

	return &f, nil
}
