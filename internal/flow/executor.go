package flow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Executor manages execution of a Flow
type Executor struct {
	Flow    *Flow
	Context *Context
	Silent  bool
}

// NewExecutor creates a runner for a given flow and context
func NewExecutor(f *Flow, ctx *Context) *Executor {
	return &Executor{
		Flow:    f,
		Context: ctx,
	}
}

// Execute runs all steps in the flow sequentially according to failure policies
func (e *Executor) Execute() ([]ExecutionResult, error) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	stepHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))
	errStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4500"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	if !e.Silent {
		fmt.Println(titleStyle.Render(fmt.Sprintf("🚀 Executing Flow: %s", e.Flow.Name)))
		if e.Flow.Description != "" {
			fmt.Printf("   %s\n", metaStyle.Render(e.Flow.Description))
		}
		fmt.Printf("   Steps: %d\n\n", len(e.Flow.Steps))
	}

	flowStart := time.Now()
	var results []ExecutionResult

	for i, step := range e.Flow.Steps {
		stepNum := fmt.Sprintf("[%d/%d]", i+1, len(e.Flow.Steps))
		if !e.Silent {
			fmt.Printf("%s %s ...\n", metaStyle.Render(stepNum), stepHeaderStyle.Render(step.Name))
		}

		res := e.executeStep(&step)
		results = append(results, res)

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

			if step.OnFailure == OnFailureContinue {
				if !e.Silent {
					fmt.Println(warnStyle.Render("   ⚠ Step failed with 'continue' policy; proceeding..."))
					fmt.Println()
				}
				continue
			}

			return results, fmt.Errorf("flow aborted at step '%s': %w", step.Name, res.Error)
		}
	}

	totalDuration := time.Since(flowStart)
	if !e.Silent {
		fmt.Println(successStyle.Render(fmt.Sprintf("✨ Flow '%s' finished successfully in %s", e.Flow.Name, totalDuration)))
	}

	return results, nil
}

func (e *Executor) executeStep(step *Step) ExecutionResult {
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

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "cmd.exe", "/c", cmdStr)
		} else {
			cmd = exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)
		}

		if workDir != "" {
			cmd.Dir = workDir
		}

		// Environment setup
		cmd.Env = os.Environ()
		for k, v := range e.Flow.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, e.Context.Interpolate(v)))
		}
		for k, v := range step.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, e.Context.Interpolate(v)))
		}

		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		err := cmd.Run()
		cancel()

		finalOutput = strings.TrimSpace(buf.String())

		if err == nil {
			return ExecutionResult{
				StepName: step.Name,
				Duration: time.Since(start),
				Success:  true,
				Output:   finalOutput,
			}
		}

		finalErr = err
		if attempt < attempts && step.RetryDur > 0 {
			time.Sleep(step.RetryDur)
		}
	}

	return ExecutionResult{
		StepName: step.Name,
		Duration: time.Since(start),
		Success:  false,
		Error:    finalErr,
		Output:   finalOutput,
	}
}

func indent(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
