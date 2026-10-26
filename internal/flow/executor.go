package flow

import (
	"bufio"
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

	"shellix/internal/core"

	"github.com/charmbracelet/lipgloss"
)

// Executor manages execution of a Flow
type Executor struct {
	Flow        *Flow
	Context     *Context
	Silent      bool
	AutoConfirm bool
	RunID       string
	Logger      *FlowLogger
	OnEvent     func(event FlowEvent)
	mu          sync.RWMutex
	stepResults map[string]ExecutionResult
}

// NewExecutor creates a runner for a given flow and context
func NewExecutor(f *Flow, ctx *Context) *Executor {
	logger, _ := NewFlowLogger(f.Name)
	if ctx != nil && f.Params != nil {
		ctx.ApplyFlowDefaults(f.Params)
	}
	return &Executor{
		Flow:        f,
		Context:     ctx,
		RunID:       fmt.Sprintf("run_%s", time.Now().Format("20060102_150405")),
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
			batchStepNames := map[string]bool{current.Name: true}
			j := i + 1
			for j < len(e.Flow.Steps) && e.Flow.Steps[j].Parallel {
				hasDepInBatch := false
				for _, dep := range e.Flow.Steps[j].DependsOn {
					if batchStepNames[dep] {
						hasDepInBatch = true
						break
					}
				}
				if hasDepInBatch {
					break
				}
				batch = append(batch, e.Flow.Steps[j])
				batchStepNames[e.Flow.Steps[j].Name] = true
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

				if res.Skipped {
					if !e.Silent {
						fmt.Printf("   %s %s (%s)\n",
							warnStyle.Render("⏭ [PARALLEL SKIP]"), st.Name, core.GlobalSanitizer.Sanitize(res.Output))
					}
					continue
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
							fmt.Printf("   Output:\n%s\n", indent(core.GlobalSanitizer.Sanitize(res.Output), "     "))
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
				fmt.Printf("   %s (%s)\n\n", warnStyle.Render("⏭ Skipped"), core.GlobalSanitizer.Sanitize(res.Output))
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
					fmt.Printf("   Output:\n%s\n", indent(core.GlobalSanitizer.Sanitize(res.Output), "     "))
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

	// Trigger notifications
	if success {
		if e.Flow.Notify.OnSuccess {
			_ = SendDesktopNotification("Shellix Flow Succeeded", fmt.Sprintf("Flow '%s' completed successfully in %s", e.Flow.Name, totalDuration.Round(time.Millisecond)), true)
		}
	} else {
		if e.Flow.Notify.OnFailure {
			_ = SendDesktopNotification("Shellix Flow Failed", fmt.Sprintf("Flow '%s' failed in %s", e.Flow.Name, totalDuration.Round(time.Millisecond)), false)
		}
	}
	if e.Flow.Notify.Webhook != "" {
		errMsg := ""
		if flowErr != nil {
			errMsg = flowErr.Error()
		}
		_ = SendWebhookNotification(e.Flow.Notify.Webhook, e.Flow.Name, success, totalDuration, errMsg)
	}

	if e.OnEvent != nil {
		e.OnEvent(FlowEvent{
			Type:     EventFlowDone,
			Duration: totalDuration,
			Error:    flowErr,
		})
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

func (e *Executor) finishStep(res ExecutionResult, isParallel bool) ExecutionResult {
	e.recordStepResult(res)
	if e.OnEvent != nil {
		e.OnEvent(FlowEvent{
			Type:       EventStepDone,
			StepName:   res.StepName,
			Result:     &res,
			Duration:   res.Duration,
			Error:      res.Error,
			IsParallel: isParallel,
		})
	}
	return res
}

func (e *Executor) executeStep(step *Step, isParallel bool) ExecutionResult {
	if e.OnEvent != nil {
		e.OnEvent(FlowEvent{
			Type:       EventStepStart,
			StepName:   step.Name,
			IsParallel: isParallel,
		})
	}

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
			return e.finishStep(res, isParallel)
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
			return e.finishStep(res, isParallel)
		}
	}

	// 3. Manual approval gate
	if step.Prompt != "" {
		promptText := e.Context.Interpolate(step.Prompt)
		if e.AutoConfirm {
			if !e.Silent {
				fmt.Printf("   [Auto-approved gate: %q]\n", promptText)
			}
		} else {
			warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA500"))
			fmt.Printf("\n%s %s [y/N]: ", warnStyle.Render("⚠️  Approval Gate:"), promptText)
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			trimmed := strings.ToLower(strings.TrimSpace(line))
			if trimmed != "y" && trimmed != "yes" {
				res := ExecutionResult{
					StepName:   step.Name,
					Duration:   0,
					Success:    false,
					Skipped:    true,
					Error:      fmt.Errorf("manual approval declined for step '%s'", step.Name),
					Output:     "Manual approval declined by user",
					IsParallel: isParallel,
				}
				return e.finishStep(res, isParallel)
			}
		}
	}

	start := time.Now()
	var finalErr error
	var finalOutput string

	attempts := 1 + step.Retries
	for attempt := 1; attempt <= attempts; attempt++ {
		rawRun := normalizeScript(step.Run)
		workDir := normalizeScript(e.Context.Interpolate(step.WorkingDir))

		var ctx context.Context
		var cancel context.CancelFunc

		if step.TimeoutDur > 0 {
			ctx, cancel = context.WithTimeout(context.Background(), step.TimeoutDur)
		} else {
			ctx, cancel = context.WithCancel(context.Background())
		}

		cmd, cleanup, cmdErr := e.buildStepExecution(ctx, rawRun)
		if cmdErr != nil {
			cancel()
			finalErr = cmdErr
			break
		}

		if workDir != "" {
			cmd.Dir = workDir
		}

		cmd.Env = buildSanitizedEnv(e.Flow.Env, step.Env, e.Context)

		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf

		err := cmd.Run()
		cleanup()
		if ctx.Err() == context.DeadlineExceeded {
			err = fmt.Errorf("step '%s' timed out after %s", step.Name, step.TimeoutDur)
		}
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
			var collected []string
			if len(step.Artifacts) > 0 {
				targetDir := workDir
				if targetDir == "" {
					targetDir = e.Context.ProjectRoot
				}
				if targetDir == "" {
					targetDir, _ = os.Getwd()
				}
				collected, _ = CollectArtifacts(e.Flow.Name, e.RunID, targetDir, step.Artifacts)
			}
			res := ExecutionResult{
				StepName:       step.Name,
				Duration:       stepDuration,
				Success:        true,
				Output:         finalOutput,
				IsParallel:     isParallel,
				PromptApproved: step.Prompt != "",
				CollectedFiles: collected,
			}
			return e.finishStep(res, isParallel)
		}

		finalErr = err
		if attempt < attempts && step.RetryDur > 0 {
			sleepDur := step.RetryDur
			if strings.EqualFold(step.Backoff, "exponential") {
				multiplier := 1 << (attempt - 1)
				sleepDur = step.RetryDur * time.Duration(multiplier)
			}
			time.Sleep(sleepDur)
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
	return e.finishStep(res, isParallel)
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

	// Export flow parameters into the step execution environment safely
	ctx.mu.RLock()
	for k, v := range ctx.Params {
		cleanKey := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(k, ".", "_"), "-", "_"))
		safeEnv = append(safeEnv, fmt.Sprintf("PARAM_%s=%s", cleanKey, v))
		safeEnv = append(safeEnv, fmt.Sprintf("SHELLIX_PARAM_%s=%s", cleanKey, v))
	}
	ctx.mu.RUnlock()

	return safeEnv
}

func isSensitiveHostEnv(key string) bool {
	return core.IsSensitiveEnvKey(key)
}

func normalizeScript(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// buildStepExecution builds an exec.Cmd using safe tokenization for parameter isolation,
// CRLF line normalization, subshell parameter escaping, and reliable Windows batch execution for multiline scripts.
func (e *Executor) buildStepExecution(ctx context.Context, rawRun string) (*exec.Cmd, func(), error) {
	cleanRun := normalizeScript(rawRun)

	// If the template itself does NOT contain shell operators:
	// Use direct execution to guarantee safe parameter interpolation and avoid command injection!
	if !hasShellOperators(cleanRun) {
		rawTokens := splitCommandTokens(cleanRun)
		if len(rawTokens) > 0 {
			interpolatedTokens := make([]string, len(rawTokens))
			for i, tok := range rawTokens {
				interpolatedTokens[i] = e.Context.Interpolate(tok)
			}
			cmd := exec.CommandContext(ctx, interpolatedTokens[0], interpolatedTokens[1:]...)
			return cmd, func() {}, nil
		}
	}

	// Step template explicitly contains shell operators or multiline script.
	// Use interpolateForSubshell to safely escape parameter placeholders.
	cmdStr := e.interpolateForSubshell(cleanRun, runtime.GOOS == "windows")
	cmdStr = normalizeScript(cmdStr)

	if runtime.GOOS == "windows" {
		// On Windows, cmd.exe /c cannot execute raw multiline strings.
		// Write to a temporary .bat script for atomic, reliable batch execution.
		if strings.Contains(cmdStr, "\n") {
			tmpFile, err := os.CreateTemp("", "shellix-step-*.bat")
			if err != nil {
				return nil, func() {}, fmt.Errorf("failed creating temporary script: %w", err)
			}
			tmpPath := tmpFile.Name()
			batContent := "@echo off\r\n" + strings.ReplaceAll(cmdStr, "\n", "\r\n") + "\r\n"
			if _, err := tmpFile.WriteString(batContent); err != nil {
				_ = tmpFile.Close()
				_ = os.Remove(tmpPath)
				return nil, func() {}, fmt.Errorf("failed writing temporary script: %w", err)
			}
			_ = tmpFile.Close()

			cleanup := func() {
				_ = os.Remove(tmpPath)
			}
			cmd := exec.CommandContext(ctx, "cmd.exe", "/c", tmpPath)
			return cmd, cleanup, nil
		}
		return exec.CommandContext(ctx, "cmd.exe", "/c", cmdStr), func() {}, nil
	}

	return exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr), func() {}, nil
}

// interpolateForSubshell interpolates template variables while safely escaping parameters
// to prevent subshell command injection vulnerabilities.
func (e *Executor) interpolateForSubshell(script string, isWindows bool) string {
	if script == "" {
		return ""
	}

	var sb strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(script); {
		ch := script[i]

		// Handle backslash escapes inside double quotes or outside quotes
		if ch == '\\' && !inSingle && i+1 < len(script) {
			sb.WriteByte(ch)
			sb.WriteByte(script[i+1])
			i += 2
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			sb.WriteByte(ch)
			i++
			continue
		}

		if ch == '"' && !inSingle {
			inDouble = !inDouble
			sb.WriteByte(ch)
			i++
			continue
		}

		// Look for template placeholder opening {{
		if ch == '{' && i+1 < len(script) && script[i+1] == '{' {
			end := strings.Index(script[i+2:], "}}")
			if end != -1 {
				placeholder := script[i : i+2+end+2]
				key := strings.TrimSpace(script[i+2 : i+2+end])

				// If it's a dynamic parameter (params.KEY or param.KEY)
				if strings.HasPrefix(key, "params.") || strings.HasPrefix(key, "param.") {
					pKey := strings.TrimPrefix(key, "params.")
					pKey = strings.TrimPrefix(pKey, "param.")

					e.Context.mu.RLock()
					val, ok := e.Context.Params[pKey]
					e.Context.mu.RUnlock()

					if ok {
						escaped := escapeParamForShell(val, inDouble, inSingle, isWindows)
						sb.WriteString(escaped)
						i += len(placeholder)
						continue
					}
				}

				// Other variables (e.g. project_root, git.branch, env.VAR, steps.X.output)
				val := e.Context.Interpolate(placeholder)
				sb.WriteString(val)
				i += len(placeholder)
				continue
			}
		}

		sb.WriteByte(ch)
		i++
	}

	return sb.String()
}

func escapeParamForShell(val string, inDouble, inSingle, isWindows bool) string {
	if isWindows {
		escaped := strings.ReplaceAll(val, `"`, `""`)
		escaped = strings.ReplaceAll(escaped, `%`, `%%`)
		if inDouble {
			return escaped
		}
		return `"` + escaped + `"`
	}

	// POSIX /bin/sh quoting rules
	if inSingle {
		return strings.ReplaceAll(val, `'`, `'\''`)
	}

	if inDouble {
		r := strings.NewReplacer(
			`\`, `\\`,
			`"`, `\"`,
			`$`, `\$`,
			"`", "\\`",
		)
		return r.Replace(val)
	}

	// Outside quotes: wrap in single quotes (completely literal in POSIX)
	return `'` + strings.ReplaceAll(val, `'`, `'\''`) + `'`
}

// buildStepCommand constructs an exec.Cmd using direct tokenization for safe commands,
// falling back to subshell execution only when pipe/redirection operators are present.
func buildStepCommand(ctx context.Context, cmdStr string) *exec.Cmd {
	clean := normalizeScript(cmdStr)
	if !hasShellOperators(clean) {
		tokens := splitCommandTokens(clean)
		if len(tokens) > 0 {
			return exec.CommandContext(ctx, tokens[0], tokens[1:]...)
		}
	}
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd.exe", "/c", clean)
	}
	return exec.CommandContext(ctx, "/bin/sh", "-c", clean)
}

func hasShellOperators(cmdStr string) bool {
	inSingle := false
	inDouble := false
	for i := 0; i < len(cmdStr); i++ {
		ch := cmdStr[i]
		if ch == '\\' && !inSingle && i+1 < len(cmdStr) {
			i++
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if !inSingle && !inDouble {
			if ch == '\n' || ch == '|' || ch == ';' || ch == '>' || ch == '<' {
				return true
			}
			if ch == '&' && i+1 < len(cmdStr) && cmdStr[i+1] == '&' {
				return true
			}
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
		if ch == '\\' && !inSingle && i+1 < len(cmd) {
			current.WriteByte(cmd[i+1])
			i++
			continue
		}
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
