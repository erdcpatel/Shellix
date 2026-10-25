package flow

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Executor manages execution of a Flow
type Executor struct {
	Flow        *Flow
	Context     *Context
	Silent      bool
	Logger      *FlowLogger
	mu          sync.RWMutex
	stepResults map[string]ExecutionResult
}

// NewExecutor creates a runner for a given flow and context
func NewExecutor(f *Flow, ctx *Context) *Executor {
	logger, _ := NewFlowLogger(f.Name)
	return &Executor{
		Flow:        f,
		Context:     ctx,
		Logger:      logger,
		stepResults: make(map[string]ExecutionResult),
	}
}

func (e *Executor) recordStepResult(res ExecutionResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stepResults[res.StepName] = res
}

func (e *Executor) isStepSuccessful(name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res, ok := e.stepResults[name]
	return ok && res.Success && !res.Skipped
}

// Execute runs steps in the flow, handling parallel batches and sequential execution
func (e *Executor) Execute() ([]ExecutionResult, error) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	stepHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	parallelHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))
	errStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4500"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	if !e.Silent {
		fmt.Println(titleStyle.Render(fmt.Sprintf("🚀 Executing Flow: %s", e.Flow.Name)))
		if e.Flow.Description != "" {
			fmt.Printf("   %s\n", metaStyle.Render(e.Flow.Description))
		}
		fmt.Printf("   Total Steps: %d\n\n", len(e.Flow.Steps))
	}

	flowStart := time.Now()
	var allResults []ExecutionResult
	var flowErr error

	i := 0
	for i < len(e.Flow.Steps) {
		current := e.Flow.Steps[i]

		// 1. Check if this begins a batch of parallel steps
		if current.Parallel {
			batch := []Step{current}
			j := i + 1
			for j < len(e.Flow.Steps) && e.Flow.Steps[j].Parallel {
				batch = append(batch, e.Flow.Steps[j])
				j++
			}

			if !e.Silent {
				fmt.Printf("%s Running %d parallel build steps concurrently...\n",
					parallelHeaderStyle.Render("⚡ [PARALLEL BATCH]"), len(batch))
				for _, s := range batch {
					fmt.Printf("   • %s\n", stepHeaderStyle.Render(s.Name))
				}
				fmt.Println()
			}

			batchResults := e.executeParallelBatch(batch)
			allResults = append(allResults, batchResults...)

			// Process outcomes for the batch
			var batchErr error
			for idx, res := range batchResults {
				st := batch[idx]
				if e.Logger != nil {
					e.Logger.LogStep(&st, res)
				}

				if res.Success {
					if !e.Silent {
						fmt.Printf("   %s %s in %s\n",
							successStyle.Render("✔ [PARALLEL DONE]"), st.Name, metaStyle.Render(res.Duration.String()))
					}
				} else {
					if !e.Silent {
						fmt.Printf("   %s %s: %v (%s)\n",
							errStyle.Render("✖ [PARALLEL FAIL]"), st.Name, res.Error, metaStyle.Render(res.Duration.String()))
						if res.Output != "" {
							fmt.Printf("   Output:\n%s\n", indent(res.Output, "     "))
						}
					}
					if st.OnFailure != OnFailureContinue && batchErr == nil {
						batchErr = fmt.Errorf("parallel step '%s' failed: %w", st.Name, res.Error)
					}
				}
			}

			if batchErr != nil {
				flowErr = batchErr
				break
			}

			if !e.Silent {
				fmt.Println()
			}
			i = j
			continue
		}

		// 2. Sequential step execution
		stepNum := fmt.Sprintf("[%d/%d]", i+1, len(e.Flow.Steps))
		if !e.Silent {
			fmt.Printf("%s %s ...\n", metaStyle.Render(stepNum), stepHeaderStyle.Render(current.Name))
		}

		res := e.executeStep(&current, false)
		allResults = append(allResults, res)
		if e.Logger != nil {
			e.Logger.LogStep(&current, res)
		}

		if res.Skipped {
			if !e.Silent {
				fmt.Printf("   %s (%s)\n\n", warnStyle.Render("⏭ Skipped"), res.Output)
			}
			i++
			continue
		}

		if res.Success {
			if !e.Silent {
				fmt.Printf("   %s in %s\n\n", successStyle.Render("✔ Completed"), metaStyle.Render(res.Duration.String()))
			}
		} else {
			if !e.Silent {
				fmt.Printf("   %s: %v (%s)\n", errStyle.Render("✖ Failed"), res.Error, metaStyle.Render(res.Duration.String()))
				if res.Output != "" {
					fmt.Printf("   Output:\n%s\n", indent(res.Output, "     "))
				}
			}

			if current.OnFailure == OnFailureContinue {
				if !e.Silent {
					fmt.Println(warnStyle.Render("   ⚠ Step failed with 'continue' policy; proceeding..."))
					fmt.Println()
				}
				i++
				continue
			}

			flowErr = fmt.Errorf("flow aborted at step '%s': %w", current.Name, res.Error)
			break
		}
		i++
	}

	totalDuration := time.Since(flowStart)
	success := flowErr == nil

	var savedLogPath string
	if e.Logger != nil {
		savedLogPath = e.Logger.Close(success, totalDuration)
	}

	if !e.Silent {
		if success {
			fmt.Println(successStyle.Render(fmt.Sprintf("✨ Flow '%s' finished successfully in %s", e.Flow.Name, totalDuration)))
		} else {
			fmt.Println(errStyle.Render(fmt.Sprintf("🚨 Flow '%s' halted with error: %v", e.Flow.Name, flowErr)))
		}
		if savedLogPath != "" {
			fmt.Printf("📄 Detailed audit log: %s\n", metaStyle.Render(savedLogPath))
		}
	}

	return allResults, flowErr
}

func (e *Executor) executeParallelBatch(steps []Step) []ExecutionResult {
	results := make([]ExecutionResult, len(steps))
	var wg sync.WaitGroup

	for idx := range steps {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = e.executeStep(&steps[i], true)
		}(idx)
	}

	wg.Wait()
	return results
}

func (e *Executor) executeStep(step *Step, isParallel bool) ExecutionResult {
	// 1. DAG dependency verification
	for _, dep := range step.DependsOn {
		if !e.isStepSuccessful(dep) {
			res := ExecutionResult{
				StepName:   step.Name,
				Duration:   0,
				Success:    false,
				Skipped:    true,
				Error:      fmt.Errorf("prerequisite dependency '%s' was not satisfied", dep),
				Output:     fmt.Sprintf("Prerequisite '%s' failed or was not satisfied", dep),
				IsParallel: isParallel,
			}
			e.recordStepResult(res)
			return res
		}
	}

	// 2. Conditional execution evaluation
	if step.If != "" {
		cond := e.Context.Interpolate(step.If)
		if !evalCondition(cond) {
			res := ExecutionResult{
				StepName:   step.Name,
				Duration:   0,
				Success:    true,
				Skipped:    true,
				Output:     fmt.Sprintf("condition '%s' evaluated to false", cond),
				IsParallel: isParallel,
			}
			e.recordStepResult(res)
			return res
		}
	}

	start := time.Now()
	var finalErr error
	var finalOutput string

	attempts := 1 + step.Retries
	for attempt := 1; attempt <= attempts; attempt++ {
		cmdStr := e.Context.Interpolate(step.Run)
		workDir := e.Context.Interpolate(step.WorkingDir)

		var ctx context.Context
		var cancel context.CancelFunc

		if step.TimeoutDur > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), step.TimeoutDur)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}

		cmd := buildStepCommand(ctx, cmdStr)

		if workDir != "" {
			cmd.Dir = workDir
		}

		cmd.Env = buildSanitizedEnv(e.Flow.Env, step.Env, e.Context)

		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		err := cmd.Run()
		cancel()

		finalOutput = strings.TrimSpace(buf.String())
		stepDuration := time.Since(start)

		var exitCode int
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}
		e.Context.SetStepResult(step.Name, finalOutput, exitCode, stepDuration)

		if err == nil {
			res := ExecutionResult{
				StepName:   step.Name,
				Duration:   stepDuration,
				Success:    true,
				Output:     finalOutput,
				IsParallel: isParallel,
			}
			e.recordStepResult(res)
			return res
		}

		finalErr = err
		if attempt < attempts && step.RetryDur > 0 {
			time.Sleep(step.RetryDur)
		}
	}

	res := ExecutionResult{
		StepName:   step.Name,
		Duration:   time.Since(start),
		Success:    false,
		Error:      finalErr,
		Output:     finalOutput,
		IsParallel: isParallel,
	}
	e.recordStepResult(res)
	return res
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// buildSanitizedEnv constructs a safe execution environment, preventing ambient host secrets from leaking
func buildSanitizedEnv(flowEnv, stepEnv map[string]string, ctx *Context) []string {
	var safeEnv []string
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			if isSensitiveHostEnv(parts[0]) {
				continue
			}
			safeEnv = append(safeEnv, env)
		}
	}

	for k, v := range flowEnv {
		safeEnv = append(safeEnv, fmt.Sprintf("%s=%s", k, ctx.Interpolate(v)))
	}
	for k, v := range stepEnv {
		safeEnv = append(safeEnv, fmt.Sprintf("%s=%s", k, ctx.Interpolate(v)))
	}
	return safeEnv
}

func isSensitiveHostEnv(key string) bool {
	k := strings.ToUpper(key)
	sensitiveTokens := []string{
		"AWS_SECRET_ACCESS_KEY", "AWS_SESSION_TOKEN", "GITHUB_TOKEN", "GH_TOKEN",
		"GITLAB_TOKEN", "SLACK_TOKEN", "OPENAI_API_KEY", "SSH_AUTH_SOCK",
		"DATABASE_PASSWORD", "PRIVATE_KEY", "SECRET_KEY", "AUTH_TOKEN",
	}
	for _, s := range sensitiveTokens {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}

// buildStepCommand constructs an exec.Cmd using direct tokenization for safe commands,
// falling back to subshell execution only when pipe/redirection operators are present.
func buildStepCommand(ctx context.Context, cmdStr string) *exec.Cmd {
	if !hasShellOperators(cmdStr) {
		tokens := splitCommandTokens(cmdStr)
		if len(tokens) > 0 {
			return exec.CommandContext(ctx, tokens[0], tokens[1:]...)
		}
	}
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd.exe", "/c", cmdStr)
	}
	return exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)
}

func hasShellOperators(cmdStr string) bool {
	operators := []string{"|", "&&", "||", ";", ">", "<", "\n"}
	for _, op := range operators {
		if strings.Contains(cmdStr, op) {
			return true
		}
	}
	return false
}

func splitCommandTokens(cmd string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
				continue
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
				continue
			}
		case ' ', '\t':
			if !inSingle && !inDouble {
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				continue
			}
		}
		current.WriteByte(ch)
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// evalCondition evaluates a basic boolean condition for flow step gating
func evalCondition(expr string) bool {
	clean := strings.TrimSpace(expr)
	if clean == "" || clean == "true" || clean == "1" {
		return true
	}
	if clean == "false" || clean == "0" {
		return false
	}
	if strings.Contains(clean, "==") {
		parts := strings.SplitN(clean, "==", 2)
		left := strings.Trim(strings.TrimSpace(parts[0]), `"'`)
		right := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		return left == right
	}
	if strings.Contains(clean, "!=") {
		parts := strings.SplitN(clean, "!=", 2)
		left := strings.Trim(strings.TrimSpace(parts[0]), `"'`)
		right := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		return left != right
	}
	return clean != ""
}
