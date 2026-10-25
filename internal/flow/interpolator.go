package flow

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var varRegex = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

// Context provides variable resolution for step interpolation
type Context struct {
	mu          sync.RWMutex
	ProjectRoot string
	GitBranch   string
	CustomVars  map[string]string
}

// NewContext creates a context populated with runtime variables
func NewContext(projectRoot, gitBranch string) *Context {
	return &Context{
		ProjectRoot: projectRoot,
		GitBranch:   gitBranch,
		CustomVars:  make(map[string]string),
	}
}

// SetVar sets a custom interpolation variable in a thread-safe manner
func (ctx *Context) SetVar(key, val string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.CustomVars[key] = val
}

// SetStepOutput records output from a completed step for downstream step interpolation
func (ctx *Context) SetStepOutput(stepName, output string) {
	ctx.SetStepResult(stepName, output, 0, 0)
}

// SetStepResult records comprehensive step results for downstream step interpolation
func (ctx *Context) SetStepResult(stepName, output string, exitCode int, duration time.Duration) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	ctx.CustomVars["steps."+stepName+".output"] = output
	ctx.CustomVars["steps."+stepName+".exit_code"] = fmt.Sprintf("%d", exitCode)
	if duration > 0 {
		ctx.CustomVars["steps."+stepName+".duration"] = duration.String()
		ctx.CustomVars["prev.duration"] = duration.String()
	}
	ctx.CustomVars["prev.output"] = output
	ctx.CustomVars["prev.exit_code"] = fmt.Sprintf("%d", exitCode)
}

// Interpolate substitutes placeholders like {{project_root}}, {{env.PATH}}, or {{steps.build.output}}
func (ctx *Context) Interpolate(input string) string {
	if input == "" {
		return ""
	}

	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return varRegex.ReplaceAllStringFunc(input, func(placeholder string) string {
		key := strings.TrimSpace(placeholder[2 : len(placeholder)-2])

		// Environment variable prefix: {{env.NAME}}
		if strings.HasPrefix(key, "env.") {
			envKey := strings.TrimPrefix(key, "env.")
			return os.Getenv(envKey)
		}

		switch key {
		case "project_root":
			return ctx.ProjectRoot
		case "git.branch":
			return ctx.GitBranch
		case "timestamp":
			return time.Now().Format(time.RFC3339)
		case "date":
			return time.Now().Format("2006-01-02")
		default:
			if val, ok := ctx.CustomVars[key]; ok {
				return val
			}
			return placeholder
		}
	})
}
