package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"shellix/internal/flow"

	"github.com/spf13/cobra"
)

// NewFlowCmd handles automation workflow commands
func NewFlowCmd(app *AppContext) *cobra.Command {
	flowCmd := &cobra.Command{
		Use:   "flow",
		Short: "Automation workflow engine with parallel builds and audit logging",
	}

	flowRunCmd := &cobra.Command{
		Use:   "run <file.yaml>",
		Short: "Execute an automation flow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := flow.ParseFile(args[0])
			if err != nil {
				return err
			}

			cwd, _ := os.Getwd()
			branch := "unknown"
			if st, err := app.GitEngine.GetStatus(cwd); err == nil {
				branch = st.Branch
			}

			ctx := flow.NewContext(cwd, branch)
			executor := flow.NewExecutor(f, ctx)
			_, err = executor.Execute()
			return err
		},
	}

	flowValidateCmd := &cobra.Command{
		Use:   "validate <file.yaml>",
		Short: "Validate a flow YAML without running",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := flow.ParseFile(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✔ Flow '%s' is valid (%d steps)\n", f.Name, len(f.Steps))
			return nil
		},
	}

	flowListCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed flows in ~/.shellix/flows",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			flowDir := filepath.Join(home, ".shellix", "flows")
			entries, err := os.ReadDir(flowDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No flows installed yet. Create flows under ~/.shellix/flows/*.yaml")
					return nil
				}
				return err
			}
			fmt.Printf("Installed flows in %s:\n", flowDir)
			for _, e := range entries {
				if !e.IsDir() && (filepath.Ext(e.Name()) == ".yaml" || filepath.Ext(e.Name()) == ".yml") {
					fmt.Printf("  • %s\n", e.Name())
				}
			}
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

	flowCmd.AddCommand(flowRunCmd, flowValidateCmd, flowListCmd, flowLogsCmd)
	return flowCmd
}
