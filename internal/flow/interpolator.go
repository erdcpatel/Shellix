package flow

import (
	"os"
	"regexp"
	"strings"
	"time"
)

var varRegex = regexp.MustCompile(`\{\{([^{}]+)\}\}`)

// Context provides variable resolution for step interpolation
type Context struct {
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

// Interpolate substitutes placeholders like {{project_root}} or {{env.PATH}}
func (ctx *Context) Interpolate(input string) string {
	if input == "" {
		return ""
	}

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
