package flow

import (
	"container/heap"
	"fmt"
	"os"
	"strings"
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

	// Validate DAG dependency graph references
	for _, step := range f.Steps {
		for _, dep := range step.DependsOn {
			if dep == step.Name {
				return nil, fmt.Errorf("step '%s' cannot depend on itself in flow '%s'", step.Name, f.Name)
			}
			if !seenNames[dep] {
				return nil, fmt.Errorf("step '%s' depends on unknown step '%s' in flow '%s'", step.Name, dep, f.Name)
			}
		}
	}

	// Build dependency graph and compute in-degrees for Kahn's algorithm
	stepMap := make(map[string]Step, len(f.Steps))
	inDegree := make(map[string]int, len(f.Steps))
	dependents := make(map[string][]string, len(f.Steps))

	for i := range f.Steps {
		step := &f.Steps[i]
		// Deduplicate dependencies
		uniqueDeps := make([]string, 0, len(step.DependsOn))
		depSeen := make(map[string]bool)
		for _, dep := range step.DependsOn {
			if !depSeen[dep] {
				depSeen[dep] = true
				uniqueDeps = append(uniqueDeps, dep)
			}
		}
		step.DependsOn = uniqueDeps

		stepMap[step.Name] = *step
		inDegree[step.Name] = len(step.DependsOn)
		for _, dep := range step.DependsOn {
			dependents[dep] = append(dependents[dep], step.Name)
		}
	}

	stepIndex := make(map[string]int, len(f.Steps))
	for i, step := range f.Steps {
		stepIndex[step.Name] = i
	}

	// Ready nodes prioritized by their original definition index to preserve relative ordering
	h := &stepIndexHeap{}
	heap.Init(h)
	for _, step := range f.Steps {
		if inDegree[step.Name] == 0 {
			heap.Push(h, stepIndex[step.Name])
		}
	}

	sortedSteps := make([]Step, 0, len(f.Steps))
	for h.Len() > 0 {
		idx := heap.Pop(h).(int)
		curr := f.Steps[idx].Name

		sortedSteps = append(sortedSteps, stepMap[curr])

		for _, depName := range dependents[curr] {
			inDegree[depName]--
			if inDegree[depName] == 0 {
				heap.Push(h, stepIndex[depName])
			}
		}
	}

	if len(sortedSteps) < len(f.Steps) {
		var cyclicSteps []string
		for _, step := range f.Steps {
			if inDegree[step.Name] > 0 {
				cyclicSteps = append(cyclicSteps, step.Name)
			}
		}
		return nil, fmt.Errorf("dependency cycle detected in flow '%s': cycle involves steps [%s]", f.Name, strings.Join(cyclicSteps, ", "))
	}

	f.Steps = sortedSteps
	return &f, nil
}

type stepIndexHeap []int

func (h stepIndexHeap) Len() int           { return len(h) }
func (h stepIndexHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h stepIndexHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *stepIndexHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *stepIndexHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
