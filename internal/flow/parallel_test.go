package flow

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParallelExecution(t *testing.T) {
	yamlData := []byte(`
name: parallel-test
steps:
  - name: prep-step
    run: echo "prep done"
  - name: parallel-task-a
    run: echo "worker-a-done"
    parallel: true
  - name: parallel-task-b
    run: echo "worker-b-done"
    parallel: true
  - name: post-step
    run: echo "all-finished"
`)

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if !f.Steps[1].Parallel || !f.Steps[2].Parallel {
		t.Errorf("expected steps 1 and 2 to be marked parallel")
	}

	ctx := NewContext(".", "main")
	exec := NewExecutor(f, ctx)
	exec.Silent = true

	start := time.Now()
	results, err := exec.Execute()
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	for _, r := range results {
		if !r.Success {
			t.Errorf("step '%s' failed: %v", r.StepName, r.Error)
		}
	}

	if exec.Logger == nil || exec.Logger.LogPath == "" {
		t.Errorf("expected log file to be written")
	} else {
		// Verify log file content exists
		data, err := os.ReadFile(exec.Logger.LogPath)
		if err != nil {
			t.Errorf("failed to read generated log file: %v", err)
		}
		if !strings.Contains(string(data), "parallel-task-a") {
			t.Errorf("expected log to contain parallel-task-a")
		}
	}

	t.Logf("Parallel flow completed in %v", duration)
}
