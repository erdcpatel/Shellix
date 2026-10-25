package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"shellix/internal/core"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewInitCmd creates the onboarding and shell integration command
func NewInitCmd(app *AppContext) *cobra.Command {
	var workspaceInit bool

	initCmd := &cobra.Command{
		Use:   "init [shell]",
		Short: "Initialize Shellix workbench directories, workspace flows, or shell aliases",
		Long: `Initialize Shellix for fast developer onboarding.

Usage:
  sx init              Setup ~/.shellix with sample flows and verified permissions
  sx init --workspace  Scaffold .shellix/flows/build.yaml in current repository
  sx init zsh          Output zsh integration script (eval "$(sx init zsh)")
  sx init bash         Output bash integration script (eval "$(sx init bash)")
  sx init powershell   Output PowerShell integration snippet`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return outputShellIntegration(args[0])
			}

			if workspaceInit {
				return initWorkspaceFlows()
			}

			return initGlobalEnvironment()
		},
	}

	initCmd.Flags().BoolVarP(&workspaceInit, "workspace", "w", false, "Scaffold .shellix/flows/ in current workspace")
	return initCmd
}

func initGlobalEnvironment() error {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	codeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68"))

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	shellixDir := filepath.Join(home, ".shellix")
	flowsDir := filepath.Join(shellixDir, "flows")
	logsDir := filepath.Join(shellixDir, "logs", "flows")

	_ = core.EnsureSecureDir(shellixDir)
	_ = core.EnsureSecureDir(flowsDir)
	_ = core.EnsureSecureDir(logsDir)

	// Write starter sample flow if missing
	sampleFlowPath := filepath.Join(flowsDir, "verification.yaml")
	if _, err := os.Stat(sampleFlowPath); os.IsNotExist(err) {
		starterFlow := `name: verification
description: Starter automated build and test flow
steps:
  - name: check-worktree
    run: git status --short
  - name: verify-tests
    run: echo "Shellix workspace ready"
`
		_ = os.WriteFile(sampleFlowPath, []byte(starterFlow), 0600)
	}

	fmt.Println(headerStyle.Render("🚀 Shellix Environment Initialized"))
	fmt.Println(metaStyle.Render("─────────────────────────────────────────────────────────────"))
	fmt.Printf(" %s Config Storage:   %s\n", passStyle.Render("✔"), metaStyle.Render(shellixDir))
	fmt.Printf(" %s Flow Store:       %s\n", passStyle.Render("✔"), metaStyle.Render(flowsDir))
	fmt.Printf(" %s Audit Logs Store: %s\n", passStyle.Render("✔"), metaStyle.Render(logsDir))
	fmt.Println()
	fmt.Println("⚡ Shell Integration (Add to your shell profile):")
	if runtime.GOOS == "windows" {
		fmt.Printf("  PowerShell: %s\n", codeStyle.Render(`Invoke-Expression (& sx init powershell)`))
	} else {
		fmt.Printf("  Zsh / Bash: %s\n", codeStyle.Render(`eval "$(sx init zsh)"`))
	}
	fmt.Printf("  Workspace:  %s to scaffold flows in current repo\n", codeStyle.Render(`sx init -w`))
	fmt.Println()
	return nil
}

func initWorkspaceFlows() error {
	passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	wsFlowsDir := filepath.Join(".", ".shellix", "flows")
	if err := os.MkdirAll(wsFlowsDir, 0755); err != nil {
		return err
	}

	buildFlow := filepath.Join(wsFlowsDir, "build.yaml")
	if _, err := os.Stat(buildFlow); os.IsNotExist(err) {
		content := `name: workspace-build
description: Local workspace build and validation pipeline
steps:
  - name: branch-check
    run: echo "Current branch: {{git.branch}}"
  - name: compile
    run: echo "Building workspace..."
  - name: test
    depends_on: [compile]
    run: echo "Running test suite..."
`
		if err := os.WriteFile(buildFlow, []byte(content), 0644); err != nil {
			return err
		}
		fmt.Printf("%s Scaffolded workspace flow: %s\n", passStyle.Render("✔"), buildFlow)
		fmt.Println("Run this flow with: sx flow run build")
	} else {
		fmt.Printf("Workspace flow already exists at %s\n", buildFlow)
	}
	return nil
}

func outputShellIntegration(shellName string) error {
	switch shellName {
	case "zsh", "bash":
		fmt.Println(`# Shellix shell integration
if command -v shellix >/dev/null 2>&1; then
    alias sx="shellix"
fi`)
	case "fish":
		fmt.Println(`# Shellix fish integration
if type -q shellix
    alias sx=shellix
end`)
	case "powershell", "pwsh":
		fmt.Println(`# Shellix PowerShell integration
if (Get-Command shellix -ErrorAction SilentlyContinue) {
    Set-Alias -Name sx -Value shellix
}`)
	default:
		return fmt.Errorf("unsupported shell '%s'. Supported: zsh, bash, fish, powershell", shellName)
	}
	return nil
}
