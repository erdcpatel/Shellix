package flow

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"shellix/internal/core"
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

func TestParseDuplicateStepNames(t *testing.T) {
	yamlData := []byte(`
name: duplicate-steps
steps:
  - name: compile
    run: go build ./...
  - name: compile
    run: go test ./...
`)

	_, err := Parse(yamlData)
	if err == nil {
		t.Fatalf("expected error on duplicate step names, but got nil")
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

func TestStepOutputChaining(t *testing.T) {
	yamlData := []byte(`
name: output-chaining
steps:
  - name: generate
    run: echo "shellix-token-42"
  - name: consume
    run: 'echo received: {{steps.generate.output}}'
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

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Output != "shellix-token-42" {
		t.Errorf("expected step 0 output 'shellix-token-42', got '%s'", results[0].Output)
	}
	if results[1].Output != "received: shellix-token-42" {
		t.Errorf("expected step 1 output 'received: shellix-token-42', got '%s'", results[1].Output)
	}
}

func TestStepMetadataChaining(t *testing.T) {
	yamlData := []byte(`
name: metadata-chaining
steps:
  - name: first
    run: echo "hello-metadata"
  - name: second
    run: 'echo code: {{steps.first.exit_code}}'
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

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[1].Output != "code: 0" {
		t.Errorf("expected exit code 'code: 0', got '%s'", results[1].Output)
	}
}

func TestSplitCommandTokens(t *testing.T) {
	cmd := `mvn clean compile -DskipTests "quoted arg with spaces" 'single quoted'`
	tokens := splitCommandTokens(cmd)
	expected := []string{"mvn", "clean", "compile", "-DskipTests", "quoted arg with spaces", "single quoted"}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %+v", len(expected), len(tokens), tokens)
	}
	for i, exp := range expected {
		if tokens[i] != exp {
			t.Errorf("token %d expected '%s', got '%s'", i, exp, tokens[i])
		}
	}
}

func TestDAGDependencyExecution(t *testing.T) {
	yamlData := []byte(`
name: dag-flow
steps:
  - name: step-a
    run: echo "step-a-ok"
  - name: step-b
    depends_on: [step-a]
    run: echo "step-b-ok"
  - name: step-c
    depends_on: [nonexistent]
    run: echo "should not parse"
`)

	_, err := Parse(yamlData)
	if err == nil {
		t.Fatalf("expected error for unknown dependency 'nonexistent', got nil")
	}

	validYaml := []byte(`
name: valid-dag
steps:
  - name: compile
    run: echo "compiled"
  - name: test
    depends_on: [compile]
    run: echo "tested"
`)

	f, err := Parse(validYaml)
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
	if len(results) != 2 || !results[0].Success || !results[1].Success {
		t.Errorf("DAG execution failed: %+v", results)
	}
}

func TestConditionalStepExecution(t *testing.T) {
	yamlData := []byte(`
name: conditional-flow
steps:
  - name: always-run
    run: echo "running"
  - name: skip-me
    if: "'{{git.branch}}' == 'release'"
    run: echo "skipped"
  - name: run-me
    if: "'{{git.branch}}' == 'main'"
    run: echo "executed"
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

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !results[0].Success || results[0].Skipped {
		t.Errorf("step 0 should have run")
	}
	if !results[1].Skipped {
		t.Errorf("step 1 should have been skipped")
	}
	if !results[2].Success || results[2].Skipped {
		t.Errorf("step 2 should have run")
	}
}

func TestDAGCycleDetection(t *testing.T) {
	// 2-node cycle: A -> B -> A
	twoNodeCycle := []byte(`
name: two-node-cycle
steps:
  - name: step-a
    depends_on: [step-b]
    run: echo "a"
  - name: step-b
    depends_on: [step-a]
    run: echo "b"
`)
	_, err := Parse(twoNodeCycle)
	if err == nil {
		t.Fatalf("expected error for 2-node cycle, got nil")
	}
	if !strings.Contains(err.Error(), "dependency cycle detected") {
		t.Errorf("expected 'dependency cycle detected' in error, got: %v", err)
	}

	// 3-node cycle: A -> B -> C -> A
	threeNodeCycle := []byte(`
name: three-node-cycle
steps:
  - name: step-a
    depends_on: [step-c]
    run: echo "a"
  - name: step-b
    depends_on: [step-a]
    run: echo "b"
  - name: step-c
    depends_on: [step-b]
    run: echo "c"
`)
	_, err = Parse(threeNodeCycle)
	if err == nil {
		t.Fatalf("expected error for 3-node cycle, got nil")
	}
	if !strings.Contains(err.Error(), "dependency cycle detected") {
		t.Errorf("expected 'dependency cycle detected' in error, got: %v", err)
	}

	// Mixed valid steps and disconnected cycle: valid-1 -> valid-2, and cycle-x <-> cycle-y
	mixedCycle := []byte(`
name: mixed-cycle
steps:
  - name: valid-1
    run: echo "v1"
  - name: valid-2
    depends_on: [valid-1]
    run: echo "v2"
  - name: cycle-x
    depends_on: [cycle-y]
    run: echo "cx"
  - name: cycle-y
    depends_on: [cycle-x]
    run: echo "cy"
`)
	_, err = Parse(mixedCycle)
	if err == nil {
		t.Fatalf("expected error for mixed cycle, got nil")
	}
	if !strings.Contains(err.Error(), "dependency cycle detected") {
		t.Errorf("expected 'dependency cycle detected' in error, got: %v", err)
	}
}

func TestDuplicateDependenciesTopologicalSort(t *testing.T) {
	yamlData := []byte(`
name: dup-deps
steps:
  - name: step-1
    run: echo "s1"
  - name: step-2
    depends_on: [step-1, step-1]
    run: echo "s2"
`)
	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if len(f.Steps) != 2 || f.Steps[0].Name != "step-1" || f.Steps[1].Name != "step-2" {
		t.Errorf("expected [step-1, step-2], got %+v", f.Steps)
	}
	if len(f.Steps[1].DependsOn) != 1 {
		t.Errorf("expected deduplicated DependsOn length 1, got %d", len(f.Steps[1].DependsOn))
	}
}


func TestTopologicalSorting(t *testing.T) {
	// Step C depends on B, Step B depends on A. Defined in reverse order: C, B, A.
	reverseOrder := []byte(`
name: reverse-order
steps:
  - name: step-c
    depends_on: [step-b]
    run: echo "c"
  - name: step-b
    depends_on: [step-a]
    run: echo "b"
  - name: step-a
    run: echo "a"
`)
	f, err := Parse(reverseOrder)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(f.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(f.Steps))
	}
	if f.Steps[0].Name != "step-a" || f.Steps[1].Name != "step-b" || f.Steps[2].Name != "step-c" {
		t.Errorf("expected steps sorted topologically [step-a, step-b, step-c], got [%s, %s, %s]",
			f.Steps[0].Name, f.Steps[1].Name, f.Steps[2].Name)
	}

	ctx := NewContext(".", "main")
	exec := NewExecutor(f, ctx)
	exec.Silent = true
	results, err := exec.Execute()
	if err != nil {
		t.Fatalf("flow execution failed: %v", err)
	}
	for _, r := range results {
		if !r.Success {
			t.Errorf("step %s should have succeeded: %v", r.StepName, r.Error)
		}
	}
}

func TestBuildSanitizedEnvExcludesSecrets(t *testing.T) {
	_ = os.Setenv("MY_APP_SECRET_TOKEN", "super-secret-token-val")
	_ = os.Setenv("REGULAR_UNSAFE_VAR", "visible-val")
	defer func() {
		_ = os.Unsetenv("MY_APP_SECRET_TOKEN")
		_ = os.Unsetenv("REGULAR_UNSAFE_VAR")
	}()

	ctx := NewContext(".", "main")
	safeEnv := buildSanitizedEnv(nil, nil, ctx)

	for _, e := range safeEnv {
		if strings.HasPrefix(e, "MY_APP_SECRET_TOKEN=") {
			t.Errorf("sensitive env var leaked into child environment: %s", e)
		}
	}
}

func TestTerminalSecretRedaction(t *testing.T) {
	secretVal := "super-secret-token-xyz-1234"
	core.GlobalSanitizer.RegisterSecret(secretVal)

	yamlData := []byte(fmt.Sprintf(`
name: redact-test
steps:
  - name: fail-step
    run: echo "%s" && exit 1
`, secretVal))

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	ctx := NewContext(".", "main")
	exec := NewExecutor(f, ctx)
	exec.Silent = false
	_, _ = exec.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	outputStr := buf.String()

	if strings.Contains(outputStr, secretVal) {
		t.Errorf("expected secret '%s' to be redacted from terminal output, got: %s", secretVal, outputStr)
	}
	if !strings.Contains(outputStr, "[REDACTED_SECRET]") {
		t.Errorf("expected '[REDACTED_SECRET]' in terminal output, got: %s", outputStr)
	}
}

func TestParallelTerminalSecretRedaction(t *testing.T) {
	secretVal := "parallel-secret-token-9876"
	core.GlobalSanitizer.RegisterSecret(secretVal)

	yamlData := []byte(fmt.Sprintf(`
name: parallel-redact-test
steps:
  - name: parallel-fail-step
    parallel: true
    run: echo "%s" && exit 1
`, secretVal))

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	ctx := NewContext(".", "main")
	exec := NewExecutor(f, ctx)
	exec.Silent = false
	_, _ = exec.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	outputStr := buf.String()

	if strings.Contains(outputStr, secretVal) {
		t.Errorf("expected secret '%s' to be redacted from parallel terminal output, got: %s", secretVal, outputStr)
	}
	if !strings.Contains(outputStr, "[REDACTED_SECRET]") {
		t.Errorf("expected '[REDACTED_SECRET]' in parallel terminal output, got: %s", outputStr)
	}
}

func TestSafeParameterInterpolationCommandInjection(t *testing.T) {
	// A flow where step.Run is a single command template with a parameter
	yamlData := []byte(`
name: safe-param-flow
steps:
  - name: print-val
    run: echo "{{params.user_input}}"
`)

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	ctx := NewContext(".", "main")
	// Malicious payload attempting command injection via semicolon
	ctx.SetParam("user_input", "hello; echo INJECTED_EXECUTION")

	exec := NewExecutor(f, ctx)
	exec.Silent = true
	results, err := exec.Execute()
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	out := results[0].Output
	// The output must contain the literal semicolon argument, and not have been executed as a separate command
	if !strings.Contains(out, "hello; echo INJECTED_EXECUTION") && !strings.Contains(out, "hello; echo") {
		t.Errorf("expected literal string argument to be echoed safely, got: %q", out)
	}
}

func TestMultilineScriptCRLFExecution(t *testing.T) {
	// Multiline script with CRLF line endings
	crlfScript := "echo line1\r\necho line2\r\n"
	yamlData := []byte(fmt.Sprintf("name: crlf-flow\nsteps:\n  - name: multi\n    run: |\n      %s",
		strings.ReplaceAll(crlfScript, "\r\n", "\r\n      ")))

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

	if len(results) != 1 || !results[0].Success {
		t.Fatalf("multiline step failed: %+v", results)
	}

	if !strings.Contains(results[0].Output, "line1") || !strings.Contains(results[0].Output, "line2") {
		t.Errorf("unexpected output from multiline CRLF script: %q", results[0].Output)
	}
}

func TestParameterValidation(t *testing.T) {
	if err := ValidateParamKey("valid_key-1.name"); err != nil {
		t.Errorf("expected valid key, got: %v", err)
	}
	if err := ValidateParamKey("invalid key with spaces"); err == nil {
		t.Errorf("expected error for key with spaces")
	}
	if err := ValidateParamKey("key;inject"); err == nil {
		t.Errorf("expected error for key with semicolon")
	}
	if err := ValidateParamValue("safe-value"); err != nil {
		t.Errorf("expected valid value, got: %v", err)
	}
	if err := ValidateParamValue("danger\x00value"); err == nil {
		t.Errorf("expected error for value with null byte")
	}
	// Value exceeding 64KB should be rejected
	largeVal := strings.Repeat("A", 70000)
	if err := ValidateParamValue(largeVal); err == nil {
		t.Errorf("expected error for value exceeding 64KB")
	}
}

func TestSafeSubshellParameterInterpolation(t *testing.T) {
	// A flow step that explicitly uses subshell operators (| and &&)
	yamlData := []byte(`
name: safe-subshell-flow
steps:
  - name: piped-injection-attempt
    run: echo "{{params.injected_quote}}" | cat
  - name: unquoted-injection-attempt
    run: echo {{params.injected_semicolon}} && true
`)

	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	ctx := NewContext(".", "main")
	// Payload 1: Attempt to break out of double quotes inside a subshell pipeline
	ctx.SetParam("injected_quote", "legit\"; echo INJECTED_PIPELINE_PWN; echo \"")
	// Payload 2: Semicolon injection in unquoted subshell parameter
	ctx.SetParam("injected_semicolon", "legit_arg; echo INJECTED_CHAINED_PWN")

	exec := NewExecutor(f, ctx)
	exec.Silent = true
	results, err := exec.Execute()
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify step 1 output: must NOT contain execution of injected command
	out1 := results[0].Output
	if strings.Contains(out1, "INJECTED_PIPELINE_PWN") {
		// If it's executed, INJECTED_PIPELINE_PWN would appear on its own line
		lines := strings.Split(out1, "\n")
		for _, l := range lines {
			if strings.TrimSpace(l) == "INJECTED_PIPELINE_PWN" {
				t.Fatalf("CRITICAL SECURITY BUG: command injection succeeded in subshell pipeline! Output: %q", out1)
			}
		}
	}

	// Verify step 2 output: must NOT execute the chained command
	out2 := results[1].Output
	lines2 := strings.Split(out2, "\n")
	for _, l := range lines2 {
		if strings.TrimSpace(l) == "INJECTED_CHAINED_PWN" {
			t.Fatalf("CRITICAL SECURITY BUG: command injection succeeded in chained subshell! Output: %q", out2)
		}
	}
}



