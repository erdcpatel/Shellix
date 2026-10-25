package flow

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ExecutionWave groups steps that can run concurrently within the execution sequence
type ExecutionWave struct {
	WaveIndex  int
	Steps      []Step
	IsParallel bool
}

// ComputeExecutionWaves analyzes step dependencies and returns chronological parallel waves
func ComputeExecutionWaves(f *Flow) []ExecutionWave {
	if len(f.Steps) == 0 {
		return nil
	}

	satisfied := make(map[string]bool)
	remaining := make([]Step, len(f.Steps))
	copy(remaining, f.Steps)

	var waves []ExecutionWave
	waveIdx := 1

	for len(remaining) > 0 {
		var currentWave []Step
		var nextRemaining []Step

		for _, step := range remaining {
			// Check if all dependencies are satisfied
			allSatisfied := true
			for _, dep := range step.DependsOn {
				if !satisfied[dep] {
					allSatisfied = false
					break
				}
			}

			if allSatisfied {
				currentWave = append(currentWave, step)
			} else {
				nextRemaining = append(nextRemaining, step)
			}
		}

		if len(currentWave) == 0 {
			// Circular or unreachable dependency fallback
			currentWave = remaining
			nextRemaining = nil
		}

		for _, s := range currentWave {
			satisfied[s.Name] = true
		}

		isParallel := len(currentWave) > 1
		waves = append(waves, ExecutionWave{
			WaveIndex:  waveIdx,
			Steps:      currentWave,
			IsParallel: isParallel,
		})

		remaining = nextRemaining
		waveIdx++
	}

	return waves
}

// RenderFlowPlan renders an execution blueprint with parameters, waves, and conditions
func RenderFlowPlan(f *Flow, ctx *Context) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	waveHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	stepNameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	cmdStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
	paramKeyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E0AF68"))
	paramValStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))

	var out strings.Builder
	out.WriteString(titleStyle.Render(fmt.Sprintf("📋 Execution Plan: %s", f.Name)) + "\n")
	if f.Description != "" {
		out.WriteString(dimStyle.Render("   "+f.Description) + "\n")
	}
	out.WriteString(borderStyle.Render("─────────────────────────────────────────────────────────────────────────────") + "\n")

	// 1. Resolved Parameters
	mergedParams := make(map[string]string)
	for k, v := range f.Params {
		mergedParams[k] = v
	}
	if ctx != nil {
		for k, v := range ctx.Params {
			mergedParams[k] = v
		}
	}

	if len(mergedParams) > 0 {
		out.WriteString("Parameters:\n")
		for k, v := range mergedParams {
			out.WriteString(fmt.Sprintf("  • %s = %s\n", paramKeyStyle.Render(k), paramValStyle.Render(v)))
		}
		out.WriteString(borderStyle.Render("─────────────────────────────────────────────────────────────────────────────") + "\n")
	}

	// 2. Waves
	waves := ComputeExecutionWaves(f)
	for _, wave := range waves {
		waveType := "Sequential"
		if wave.IsParallel {
			waveType = "Parallel (Concurrent Wave)"
		}

		out.WriteString(fmt.Sprintf("%s (%s, %d steps)\n",
			waveHeaderStyle.Render(fmt.Sprintf("🌊 Wave %d", wave.WaveIndex)),
			dimStyle.Render(waveType),
			len(wave.Steps),
		))

		for i, s := range wave.Steps {
			branchSymbol := "├── "
			if i == len(wave.Steps)-1 {
				branchSymbol = "└── "
			}

			cmdDisplay := s.Run
			if ctx != nil {
				cmdDisplay = ctx.Interpolate(s.Run)
			}
			if len(cmdDisplay) > 50 {
				cmdDisplay = cmdDisplay[:47] + "..."
			}

			out.WriteString(fmt.Sprintf("  %s[%s] %s\n",
				dimStyle.Render(branchSymbol),
				stepNameStyle.Render(s.Name),
				cmdStyle.Render(cmdDisplay),
			))

			// Details
			var meta []string
			if len(s.DependsOn) > 0 {
				meta = append(meta, fmt.Sprintf("depends_on: [%s]", strings.Join(s.DependsOn, ", ")))
			}
			if s.If != "" {
				meta = append(meta, fmt.Sprintf("if: %s", s.If))
			}
			if s.Timeout != "" {
				meta = append(meta, fmt.Sprintf("timeout: %s", s.Timeout))
			}
			if s.Retries > 0 {
				meta = append(meta, fmt.Sprintf("retries: %d", s.Retries))
			}
			if s.Prompt != "" {
				promptDisplay := s.Prompt
				if ctx != nil {
					promptDisplay = ctx.Interpolate(s.Prompt)
				}
				meta = append(meta, warnStyle.Render(fmt.Sprintf("approval_gate: %q", promptDisplay)))
			}
			if len(s.Artifacts) > 0 {
				meta = append(meta, fmt.Sprintf("artifacts: [%s]", strings.Join(s.Artifacts, ", ")))
			}

			if len(meta) > 0 {
				padding := "  │    "
				if i == len(wave.Steps)-1 {
					padding = "       "
				}
				out.WriteString(fmt.Sprintf("%s%s\n", padding, dimStyle.Render(strings.Join(meta, " • "))))
			}
		}
		out.WriteString("\n")
	}

	return out.String()
}

// RenderMermaid generates a valid Mermaid diagram representation of the workflow DAG
func RenderMermaid(f *Flow) string {
	var b strings.Builder
	b.WriteString("```mermaid\n")
	b.WriteString("flowchart TD\n")

	for _, s := range f.Steps {
		nodeShape := fmt.Sprintf("%s[\"%s\"]", s.Name, s.Name)
		if s.Prompt != "" {
			nodeShape = fmt.Sprintf("%s{{\"⚠️ %s (Approval)\"}}", s.Name, s.Name)
		}

		if len(s.DependsOn) == 0 {
			b.WriteString(fmt.Sprintf("  Start --> %s\n", nodeShape))
		} else {
			for _, dep := range s.DependsOn {
				b.WriteString(fmt.Sprintf("  %s --> %s\n", dep, nodeShape))
			}
		}
	}

	b.WriteString("```\n")
	return b.String()
}
