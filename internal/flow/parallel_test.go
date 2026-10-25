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

func TestParallelStepSkipDoesNotAbort(t *testing.T) {
	yamlData := []byte(`
name: parallel-skip-test
steps:
  - name: failing-prep
    run: exit 1
    on_failure: continue
  - name: parallel-skipped
    depends_on: [failing-prep]
    parallel: true
    run: echo "should be skipped"
  - name: parallel-independent
    parallel: true
    run: echo "worker-independent-done"
  - name: final-step
    run: echo "final-ok"
`)

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	ctx := NewContext(".", "main")
	exec := NewExecutor(f, ctx)
	exec.Silent = true

	results, err := exec.Execute()
	if err != nil {
		t.Fatalf("execution should NOT fail when parallel step is skipped, got: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	if results[0].Success {
		t.Errorf("step 0 expected failure")
	}
	if !results[1].Skipped {
		t.Errorf("step 1 should be skipped due to unsatisfied dependency")
	}
	if !results[2].Success || results[2].Skipped {
		t.Errorf("step 2 should have succeeded")
	}
	if !results[3].Success {
		t.Errorf("step 3 should have succeeded")
	}
}

func TestParallelStepConditionSkip(t *testing.T) {
	yamlData := []byte(`
name: parallel-condition-skip
steps:
  - name: parallel-cond-skipped
    if: "'{{git.branch}}' == 'nonexistent-branch'"
    parallel: true
    run: echo "skip me"
  - name: parallel-cond-run
    parallel: true
    run: echo "run me"
`)

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	ctx := NewContext(".", "main")
	exec := NewExecutor(f, ctx)
	exec.Silent = true

	results, err := exec.Execute()
	if err != nil {
		t.Fatalf("execution should succeed, got: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Skipped {
		t.Errorf("expected step 0 to be skipped")
	}
	if !results[1].Success || results[1].Skipped {
		t.Errorf("expected step 1 to succeed")
	}
}

func TestParallelStepsWithInterdependencies(t *testing.T) {
	yamlData := []byte(`
name: parallel-interdep-test
steps:
  - name: step-1
    parallel: true
    run: echo "step1"
  - name: step-2
    parallel: true
    depends_on: [step-1]
    run: echo "step2"
  - name: step-3
    parallel: true
    depends_on: [step-1]
    run: echo "step3"
`)

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	ctx := NewContext(".", "main")
	exec := NewExecutor(f, ctx)
	exec.Silent = true

	results, err := exec.Execute()
	if err != nil {
		t.Fatalf("execution should succeed, got: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if !r.Success || r.Skipped {
			t.Errorf("expected step %s to succeed, got success=%v, skipped=%v, output=%s",
				r.StepName, r.Success, r.Skipped, r.Output)
		}
	}
}

func TestParallelDiamondDAG(t *testing.T) {
	// Reverse definition order diamond: D (depends on B, C), C (depends on A), B (depends on A), A
	yamlData := []byte(`
name: diamond-dag-test
steps:
  - name: step-d
    parallel: true
    depends_on: [step-b, step-c]
    run: echo "d-done"
  - name: step-c
    parallel: true
    depends_on: [step-a]
    run: echo "c-done"
  - name: step-b
    parallel: true
    depends_on: [step-a]
    run: echo "b-done"
  - name: step-a
    parallel: true
    run: echo "a-done"
`)

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if f.Steps[0].Name != "step-a" {
		t.Errorf("expected step-a first, got %s", f.Steps[0].Name)
	}
	if f.Steps[3].Name != "step-d" {
		t.Errorf("expected step-d last, got %s", f.Steps[3].Name)
	}

	ctx := NewContext(".", "main")
	exec := NewExecutor(f, ctx)
	exec.Silent = true

	results, err := exec.Execute()
	if err != nil {
		t.Fatalf("execution should succeed, got: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	for _, r := range results {
		if !r.Success || r.Skipped {
			t.Errorf("step %s should have succeeded, got success=%v, skipped=%v, output=%s",
				r.StepName, r.Success, r.Skipped, r.Output)
		}
	}
}


