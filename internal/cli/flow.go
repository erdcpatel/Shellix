package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"shellix/internal/flow"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewFlowCmd handles automation workflow commands
func NewFlowCmd(app *AppContext) *cobra.Command {
	var skipConfirm bool
	var useTUI bool

	flowCmd := &cobra.Command{
		Use:   "flow",
		Short: "Automation workflow engine with parallel builds and audit logging",
		Run: func(cmd *cobra.Command, args []string) {
			listAllFlows()
		},
	}

	flowRunCmd := &cobra.Command{
		Use:   "run <flow-name-or-path>",
		Short: "Execute an automation flow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := resolveFlowPath(args[0])
			f, err := flow.ParseFile(targetPath)
			if err != nil {
				return err
			}

			cwd, _ := os.Getwd()
			branch := "unknown"
			if st, err := app.GitEngine.GetStatus(cwd); err == nil {
				branch = st.Branch
			}

			start := time.Now()
			var runErr error
			defer func() {
				dur := time.Since(start)
				exitCode := 0
				if runErr != nil {
					exitCode = 1
				}
				cmdStr := fmt.Sprintf("sx flow run %s", args[0])
				if useTUI {
					cmdStr += " --tui"
				}
				if store, sErr := getHistoryStore(app); sErr == nil && store != nil {
					_, _ = store.Record(cmdStr, cwd, dur, exitCode)
				}
			}()

			ctx := flow.NewContext(cwd, branch)
			if useTUI {
				runErr = flow.RunFlowTUI(f, ctx)
				return runErr
			}

			executor := flow.NewExecutor(f, ctx)
			_, runErr = executor.Execute()
			return runErr
		},
	}
	flowRunCmd.Flags().BoolVarP(&useTUI, "tui", "t", false, "Launch full-screen interactive Bubble Tea flow dashboard")
	flowRunCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Execute flow without confirmation prompt")

	flowTUICmd := &cobra.Command{
		Use:   "tui <flow-name-or-path>",
		Short: "Execute a flow with interactive full-screen Bubble Tea dashboard",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := resolveFlowPath(args[0])
			f, err := flow.ParseFile(targetPath)
			if err != nil {
				return err
			}

			cwd, _ := os.Getwd()
			branch := "unknown"
			if st, err := app.GitEngine.GetStatus(cwd); err == nil {
				branch = st.Branch
			}

			ctx := flow.NewContext(cwd, branch)
			start := time.Now()
			var runErr error
			defer func() {
				dur := time.Since(start)
				exitCode := 0
				if runErr != nil {
					exitCode = 1
				}
				if store, sErr := getHistoryStore(app); sErr == nil && store != nil {
					_, _ = store.Record(fmt.Sprintf("sx flow tui %s", args[0]), cwd, dur, exitCode)
				}
			}()

			runErr = flow.RunFlowTUI(f, ctx)
			return runErr
		},
	}

	flowValidateCmd := &cobra.Command{
		Use:   "validate <file.yaml>",
		Short: "Validate a flow YAML without running",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := resolveFlowPath(args[0])
			f, err := flow.ParseFile(targetPath)
			if err != nil {
				return err
			}
			fmt.Printf("✔ Flow '%s' is valid (%d steps)\n", f.Name, len(f.Steps))
			return nil
		},
	}

	flowListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all installed and workspace flows",
		RunE: func(cmd *cobra.Command, args []string) error {
			listAllFlows()
			return nil
		},
	}

	flowLogsCmd := &cobra.Command{
		Use:   "logs [filename]",
		Short: "Inspect flow execution audit logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				dir, _ := flow.GetLogDir()
				target := args[0]
				if !filepath.IsAbs(target) {
					target = filepath.Join(dir, target)
				}
				data, err := os.ReadFile(target)
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			logs, err := flow.ListLogs()
			if err != nil {
				return err
			}
			if len(logs) == 0 {
				fmt.Println("No flow execution logs found yet.")
				return nil
			}
			fmt.Printf("Execution Audit Logs (%d available in ~/.shellix/logs/flows):\n", len(logs))
			max := len(logs)
			if max > 10 {
				max = 10
			}
			for i := 0; i < max; i++ {
				fmt.Printf("  • %s\n", filepath.Base(logs[i]))
			}
			fmt.Println("\nView any log with: sx flow logs <filename>")
			return nil
		},
	}

	flowCmd.AddCommand(flowRunCmd, flowValidateCmd, flowListCmd, flowLogsCmd, flowTUICmd)
	return flowCmd
}

func resolveFlowPath(nameOrPath string) string {
	if _, err := os.Stat(nameOrPath); err == nil {
		return nameOrPath
	}

	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(".", ".shellix", "flows", nameOrPath),
		filepath.Join(".", ".shellix", "flows", nameOrPath+".yaml"),
		filepath.Join(".", "tests", "fixtures", "flows", nameOrPath),
		filepath.Join(".", "tests", "fixtures", "flows", nameOrPath+".yaml"),
		filepath.Join(home, ".shellix", "flows", nameOrPath),
		filepath.Join(home, ".shellix", "flows", nameOrPath+".yaml"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return nameOrPath
}

func listAllFlows() {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68"))

	fmt.Println(titleStyle.Render("⚡ Shellix Automation Flows"))
	fmt.Println(metaStyle.Render("─────────────────────────────────────────────────────────────"))

	home, _ := os.UserHomeDir()
	dirs := []struct {
		Label string
		Path  string
	}{
		{"Workspace Flows", filepath.Join(".", ".shellix", "flows")},
		{"Fixture Flows", filepath.Join(".", "tests", "fixtures", "flows")},
		{"User Global Flows", filepath.Join(home, ".shellix", "flows")},
	}

	foundTotal := 0
	for _, d := range dirs {
		entries, err := os.ReadDir(d.Path)
		if err != nil || len(entries) == 0 {
			continue
		}

		fmt.Printf("\n📂 %s (%s):\n", d.Label, metaStyle.Render(d.Path))
		for _, e := range entries {
			ext := filepath.Ext(e.Name())
			if !e.IsDir() && (ext == ".yaml" || ext == ".yml") {
				flowPath := filepath.Join(d.Path, e.Name())
				f, err := flow.ParseFile(flowPath)
				desc := ""
				stepsCount := 0
				if err == nil {
					desc = f.Description
					stepsCount = len(f.Steps)
				}
				base := e.Name()[:len(e.Name())-len(ext)]
				fmt.Printf("   • %-20s %s (%d steps)\n", nameStyle.Render(base), descStyle.Render(desc), stepsCount)
				foundTotal++
			}
		}
	}

	if foundTotal == 0 {
		fmt.Println("No flows found yet. Create a flow YAML in .shellix/flows/ or ~/.shellix/flows/")
	} else {
		fmt.Printf("\nRun any flow: %s\n", metaStyle.Render("sx flow run <flow-name>"))
	}
	fmt.Println()
}
