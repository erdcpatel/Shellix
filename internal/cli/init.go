package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"shellix/internal/core"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewInitCmd creates the onboarding and shell integration command
func NewInitCmd(app *AppContext) *cobra.Command {
	var workspaceInit bool

	initCmd := &cobra.Command{
		Use:   "init [shell|desktop]",
		Short: "Initialize Shellix workbench directories, workspace flows, or shell aliases",
		Long: `Initialize Shellix for fast developer onboarding.

Usage:
  sx init              Setup ~/.shellix with sample flows and verified permissions
  sx init --workspace  Scaffold .shellix/flows/build.yaml in current repository
  sx init desktop      Generate Windows Desktop Shortcut (Shellix.lnk) for one-click launch
  sx init zsh          Output zsh integration script (eval "$(sx init zsh)")
  sx init bash         Output bash integration script (eval "$(sx init bash)")
  sx init powershell   Output PowerShell integration snippet`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if args[0] == "desktop" {
					return createDesktopShortcut()
				}
				return outputShellIntegration(args[0])
			}

			if workspaceInit {
				return initWorkspaceFlows()
			}

			return initGlobalEnvironment()
		},
	}

	initCmd.Flags().BoolVarP(&workspaceInit, "workspace", "w", false, "Scaffold .shellix/flows/ in current workspace")

	initCmd.AddCommand(&cobra.Command{
		Use:   "desktop",
		Short: "Generate Windows Desktop Shortcut (Shellix.lnk) for one-click launch",
		RunE: func(cmd *cobra.Command, args []string) error {
			return createDesktopShortcut()
		},
	})

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
	case "zsh":
		fmt.Println(`# Shellix Zsh integration
if command -v shellix >/dev/null 2>&1; then
    alias sx="shellix"
    alias slx="shellix"
fi
if command -v sx >/dev/null 2>&1; then
    source <(sx completion zsh 2>/dev/null)
fi`)
	case "bash":
		fmt.Println(`# Shellix Bash integration
if command -v shellix >/dev/null 2>&1; then
    alias sx="shellix"
    alias slx="shellix"
fi
if command -v sx >/dev/null 2>&1; then
    source <(sx completion bash 2>/dev/null)
fi`)
	case "fish":
		fmt.Println(`# Shellix Fish integration
if type -q shellix
    alias sx=shellix
    alias slx=shellix
end
if type -q sx
    sx completion fish | source
end`)
	case "powershell", "pwsh":
		fmt.Println(`# Shellix PowerShell integration (Windows First-Class)
if (Get-Command shellix -ErrorAction SilentlyContinue) {
    Set-Alias -Name sx -Value shellix
    Set-Alias -Name slx -Value shellix
}
if (Get-Command sx -ErrorAction SilentlyContinue) {
    sx completion powershell | Out-String | Invoke-Expression
}
if (Get-Command slx -ErrorAction SilentlyContinue) {
    slx completion powershell | Out-String | Invoke-Expression
}`)
	default:
		return fmt.Errorf("unsupported shell '%s'. Supported: powershell, zsh, bash, fish", shellName)
	}
	return nil
}

// GenerateWindowsShortcutScript produces the PowerShell script that uses WScript.Shell
// to instantiate a Desktop shortcut (.lnk) pointing to the Shellix binary.
func GenerateWindowsShortcutScript(targetPath, workingDir string) string {
	escapedTarget := strings.ReplaceAll(targetPath, "'", "''")
	escapedWorkingDir := strings.ReplaceAll(workingDir, "'", "''")
	return fmt.Sprintf(`$ws = New-Object -ComObject WScript.Shell; `+
		`$desktop = $ws.SpecialFolders.Item('Desktop'); `+
		`$shortcutPath = Join-Path $desktop 'Shellix.lnk'; `+
		`$shortcut = $ws.CreateShortcut($shortcutPath); `+
		`$shortcut.TargetPath = '%s'; `+
		`$shortcut.WorkingDirectory = '%s'; `+
		`$shortcut.IconLocation = '%s,0'; `+
		`$shortcut.Description = 'Shellix Developer Terminal Workbench'; `+
		`$shortcut.Save(); `+
		`Write-Output $shortcutPath`, escapedTarget, escapedWorkingDir, escapedTarget)
}

var shortcutCommandRunner = func(psCmd, script string) ([]byte, error) {
	cmd := exec.Command(psCmd, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	return cmd.CombinedOutput()
}

func createDesktopShortcut() error {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	passStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68"))

	if runtime.GOOS != "windows" {
		fmt.Println(headerStyle.Render("🖥️ Shellix Desktop Shortcut"))
		fmt.Println(metaStyle.Render("─────────────────────────────────────────────────────────────"))
		fmt.Printf(" %s Windows Desktop shortcut (.lnk) generation via WScript.Shell is designed for Windows (detected: %s).\n", warnStyle.Render("!"), runtime.GOOS)
		return nil
	}

	exePath, err := os.Executable()
	if err != nil {
		exePath = os.Args[0]
	}
	if abs, err := filepath.Abs(exePath); err == nil {
		exePath = abs
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = filepath.Dir(exePath)
	}

	script := GenerateWindowsShortcutScript(exePath, homeDir)

	psCmd := "powershell"
	if _, err := exec.LookPath("pwsh.exe"); err == nil {
		psCmd = "pwsh"
	}

	out, err := shortcutCommandRunner(psCmd, script)
	if err != nil {
		return fmt.Errorf("failed to create desktop shortcut: %w (output: %s)", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var createdPath string
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasSuffix(strings.ToLower(trimmed), ".lnk") {
			createdPath = trimmed
			break
		}
	}
	if createdPath == "" {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			createdPath = strings.TrimSpace(lines[len(lines)-1])
		} else {
			createdPath = filepath.Join(homeDir, "Desktop", "Shellix.lnk")
		}
	}

	fmt.Println(headerStyle.Render("🚀 Shellix Desktop Shortcut Created"))
	fmt.Println(metaStyle.Render("─────────────────────────────────────────────────────────────"))
	fmt.Printf(" %s Shortcut:   %s\n", passStyle.Render("✔"), metaStyle.Render(createdPath))
	fmt.Printf(" %s Target:     %s\n", passStyle.Render("✔"), metaStyle.Render(exePath))
	fmt.Println("\nLaunch Shellix directly from your Desktop with one click.")
	return nil
}
