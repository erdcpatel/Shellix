package flow

import (
	"os"
	"testing"
	"time"
)

func TestParseValidFlow(t *testing.T) {
	yamlData := []byte(`
name: test-flow
description: A sample verification flow
env:
  TEST_VAR: hello
steps:
  - name: step-one
    run: echo "hello world"
    timeout: 10s
  - name: step-two
    run: echo "{{env.TEST_VAR}}"
    on_failure: continue
`)

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if f.Name != "test-flow" {
		t.Errorf("expected name 'test-flow', got '%s'", f.Name)
	}
	if len(f.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(f.Steps))
	}
	if f.Steps[0].TimeoutDur != 10*time.Second {
		t.Errorf("expected 10s duration, got %v", f.Steps[0].TimeoutDur)
	}
	if f.Steps[1].OnFailure != OnFailureContinue {
		t.Errorf("expected OnFailureContinue, got %v", f.Steps[1].OnFailure)
	}
}

func TestInterpolation(t *testing.T) {
	_ = os.Setenv("SHELLIX_TEST_KEY", "awesome_value")
	defer func() { _ = os.Unsetenv("SHELLIX_TEST_KEY") }()

	ctx := NewContext("/tmp/my-project", "feature/auth")
	ctx.CustomVars["app_port"] = "8080"

	input := "deploy --dir {{project_root}} --branch {{git.branch}} --env {{env.SHELLIX_TEST_KEY}} --port {{app_port}}"
	expected := "deploy --dir /tmp/my-project --branch feature/auth --env awesome_value --port 8080"

	out := ctx.Interpolate(input)
	if out != expected {
		t.Errorf("expected '%s', got '%s'", expected, out)
	}
}

func TestExecuteFlowSuccess(t *testing.T) {
	yamlData := []byte(`
name: simple-exec
steps:
  - name: echo-test
    run: echo "shellix rocks"
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
		t.Fatalf("execution failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Success {
		t.Errorf("step expected success, got error: %v", results[0].Error)
	}
	if results[0].Output != "shellix rocks" {
		t.Errorf("expected 'shellix rocks', got '%s'", results[0].Output)
	}
}

func TestExecuteFlowContinueOnFailure(t *testing.T) {
	yamlData := []byte(`
name: failure-continue
steps:
  - name: fail-step
    run: exit 1
    on_failure: continue
  - name: pass-step
    run: echo "survived"
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
		t.Fatalf("flow should have continued, but got err: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Success {
		t.Errorf("step 1 should have failed")
	}
	if !results[1].Success {
		t.Errorf("step 2 should have succeeded")
	}
}
