package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"shellix/internal/flow"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewFlowCmd handles automation workflow commands
func NewFlowCmd(app *AppContext) *cobra.Command {
	var skipConfirm bool
	var useTUI bool
	var watchMode bool

	flowCmd := &cobra.Command{
		Use:   "flow",
		Short: "Automation workflow engine with parallel builds and audit logging",
		Run: func(cmd *cobra.Command, args []string) {
			listAllFlows()
		},
	}

	flowRunCmd := &cobra.Command{
		Use:               "run <flow-name-or-path>",
		Short:             "Execute an automation flow",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeFlowNames,
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
			if watchMode {
				sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
				defer cancel()
				cfg := flow.WatcherConfig{
					Flow:     f,
					Context:  ctx,
					WatchDir: cwd,
				}
				runErr = flow.RunWatcher(sigCtx, cfg)
				return runErr
			}

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
	flowRunCmd.Flags().BoolVarP(&watchMode, "watch", "w", false, "Watch directory for file changes and re-run flow")
	flowRunCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Execute flow without confirmation prompt")

	var watchExts string
	flowWatchCmd := &cobra.Command{
		Use:               "watch <flow-name-or-path>",
		Short:             "Watch workspace for changes and automatically re-execute flow",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeFlowNames,
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
			sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			var exts []string
			if watchExts != "" {
				for _, e := range strings.Split(watchExts, ",") {
					exts = append(exts, strings.TrimSpace(e))
				}
			}

			cfg := flow.WatcherConfig{
				Flow:       f,
				Context:    ctx,
				WatchDir:   cwd,
				Extensions: exts,
			}
			return flow.RunWatcher(sigCtx, cfg)
		},
	}
	flowWatchCmd.Flags().StringVar(&watchExts, "ext", "", "Comma-separated extensions to watch (e.g. go,yaml,java)")

	flowTUICmd := &cobra.Command{
		Use:               "tui <flow-name-or-path>",
		Short:             "Execute a flow with interactive full-screen Bubble Tea dashboard",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeFlowNames,
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

	var saveGlobal bool
	flowRecordCmd := &cobra.Command{
		Use:   "record [flow-name]",
		Short: "Interactively record shell commands and synthesize a workflow YAML",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flowName := "recorded-workflow"
			if len(args) > 0 {
				flowName = args[0]
			}
			saveDir := filepath.Join(".", ".shellix", "flows")
			if saveGlobal {
				home, _ := os.UserHomeDir()
				saveDir = filepath.Join(home, ".shellix", "flows")
			}
			_, err := flow.RunInteractiveRecorder(flowName, saveDir)
			return err
		},
	}
	flowRecordCmd.Flags().BoolVarP(&saveGlobal, "global", "g", false, "Save synthesized workflow to ~/.shellix/flows/ instead of workspace")

	flowReplayCmd := &cobra.Command{
		Use:               "replay <flow-name-or-path>",
		Short:             "Replay a workflow with interactive dashboard verification",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeFlowNames,
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
			return flow.RunFlowTUI(f, ctx)
		},
	}

	flowScheduleCmd := &cobra.Command{
		Use:   "schedule",
		Short: "Cron-based recurring flow scheduler and background runner",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listScheduledFlows()
		},
	}

	var scheduleCron string
	schedAddCmd := &cobra.Command{
		Use:               "add <flow-name-or-path> <cron-expr>",
		Short:             "Register a recurring cron schedule for an automation flow",
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: completeFlowNames,
		RunE: func(cmd *cobra.Command, args []string) error {
			flowInput := args[0]
			cronExpr := scheduleCron
			if len(args) == 2 {
				cronExpr = args[1]
			}
			if cronExpr == "" {
				return fmt.Errorf("cron expression required (e.g. \"*/15 * * * *\" or \"@daily\")")
			}

			targetPath := resolveFlowPath(flowInput)
			f, err := flow.ParseFile(targetPath)
			if err != nil {
				return fmt.Errorf("cannot resolve flow %q: %w", flowInput, err)
			}

			store, err := flow.NewScheduleStore("")
			if err != nil {
				return err
			}

			job, err := store.Add(f.Name, targetPath, cronExpr)
			if err != nil {
				return err
			}

			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
			keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
			valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

			fmt.Println(titleStyle.Render("⏰ Scheduled Automation Flow Added"))
			fmt.Printf("  • Job ID   : %s\n", keyStyle.Render(job.ID))
			fmt.Printf("  • Flow     : %s\n", keyStyle.Render(job.FlowName))
			fmt.Printf("  • Cron     : %s\n", valStyle.Render(job.CronExpr))
			fmt.Printf("  • Next Run : %s\n", dimStyle.Render(job.NextRun.Format(time.RFC1123)))
			return nil
		},
	}
	schedAddCmd.Flags().StringVarP(&scheduleCron, "cron", "c", "", "Cron expression (e.g. '0 0 * * *', '@hourly', '*/5 * * * *')")

	schedListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all scheduled recurring flows",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listScheduledFlows()
		},
	}

	schedRemoveCmd := &cobra.Command{
		Use:   "remove <id-or-flow-name>",
		Short: "Remove a scheduled flow job",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			store, err := flow.NewScheduleStore("")
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			jobs, err := store.Load()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var list []string
			for _, j := range jobs {
				list = append(list, fmt.Sprintf("%s\t%s (%s)", j.FlowName, j.ID, j.CronExpr))
			}
			return list, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := flow.NewScheduleStore("")
			if err != nil {
				return err
			}

			removed, err := store.Remove(args[0])
			if err != nil {
				return err
			}
			if !removed {
				return fmt.Errorf("no scheduled job found matching %q", args[0])
			}

			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			fmt.Println(successStyle.Render(fmt.Sprintf("✓ Schedule for %q successfully removed", args[0])))
			return nil
		},
	}

	var pollSec int
	schedStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Run the recurring flow scheduler daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := flow.NewScheduleStore("")
			if err != nil {
				return err
			}

			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			fmt.Println(titleStyle.Render("⏰ Shellix Flow Scheduler Daemon Active"))
			fmt.Println(dimStyle.Render(fmt.Sprintf("Checking pending jobs every %d seconds. Press Ctrl+C to terminate.", pollSec)))

			sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			return store.RunDaemon(sigCtx, time.Duration(pollSec)*time.Second, func(job flow.ScheduledJob) error {
				fmt.Printf("[%s] 🚀 Triggering recurring flow: %s (%s)\n", time.Now().Format("15:04:05"), job.FlowName, job.CronExpr)
				targetPath := resolveFlowPath(job.FlowPath)
				f, err := flow.ParseFile(targetPath)
				if err != nil {
					fmt.Printf("   ❌ Flow parse error: %v\n", err)
					return err
				}

				cwd, _ := os.Getwd()
				branch := "unknown"
				if st, err := app.GitEngine.GetStatus(cwd); err == nil {
					branch = st.Branch
				}
				ctx := flow.NewContext(cwd, branch)
				executor := flow.NewExecutor(f, ctx)
				results, err := executor.Execute()
				if err != nil {
					fmt.Printf("   ❌ Flow execution failed: %v\n", err)
					return err
				}
				fmt.Printf("   ✓ Flow executed successfully (%d steps)\n", len(results))
				return nil
			})
		},
	}
	schedStartCmd.Flags().IntVarP(&pollSec, "interval", "i", 2, "Polling check interval in seconds")

	daemonServiceCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage native OS background service for flow scheduler (Windows Task Scheduler / macOS LaunchAgent)",
	}

	daemonInstallCmd := &cobra.Command{
		Use:   "install",
		Short: "Register scheduler as native background service (runs automatically on logon)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := flow.InstallDaemonService(""); err != nil {
				return err
			}
			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			fmt.Printf("%s Flow scheduler registered as native background service.\n", successStyle.Render("✔"))
			fmt.Println("Recurring flows will now execute automatically in the background.")
			return nil
		},
	}

	daemonUninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove scheduler background service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := flow.UninstallDaemonService(); err != nil {
				return err
			}
			fmt.Println("✓ Flow scheduler background service removed.")
			return nil
		},
	}

	daemonStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Query status of background scheduler service",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := flow.DaemonServiceStatus()
			if err != nil {
				return err
			}
			fmt.Printf("Service Status: %s\n", st)
			return nil
		},
	}

	daemonServiceCmd.AddCommand(daemonInstallCmd, daemonUninstallCmd, daemonStatusCmd)
	flowScheduleCmd.AddCommand(schedAddCmd, schedListCmd, schedRemoveCmd, schedStartCmd, daemonServiceCmd)

	flowCmd.AddCommand(flowRunCmd, flowValidateCmd, flowListCmd, flowLogsCmd, flowTUICmd, flowWatchCmd, flowRecordCmd, flowReplayCmd, flowScheduleCmd)
	return flowCmd
}

func listScheduledFlows() error {
	store, err := flow.NewScheduleStore("")
	if err != nil {
		return err
	}

	jobs, err := store.Load()
	if err != nil {
		return err
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))

	fmt.Println(titleStyle.Render("⏰ Shellix Recurring Flow Schedules"))
	fmt.Println(borderStyle.Render("──────────────────────────────────────────────────────────────────────────────────────────"))

	if len(jobs) == 0 {
		fmt.Println("No scheduled flows registered.")
		fmt.Println(dimStyle.Render("Add a schedule with: sx flow schedule add <flow-name> \"*/15 * * * *\""))
		fmt.Println()
		return nil
	}

	fmt.Printf("%-14s %-20s %-16s %-10s %-18s %-18s %-6s\n",
		headerStyle.Render("JOB ID"),
		headerStyle.Render("FLOW"),
		headerStyle.Render("CRON"),
		headerStyle.Render("STATUS"),
		headerStyle.Render("LAST RUN"),
		headerStyle.Render("NEXT RUN"),
		headerStyle.Render("RUNS"),
	)
	fmt.Println(borderStyle.Render("──────────────────────────────────────────────────────────────────────────────────────────"))

	for _, j := range jobs {
		status := greenStyle.Render("ENABLED")
		if !j.Enabled {
			status = dimStyle.Render("PAUSED")
		} else if j.LastStatus == "FAILED" {
			status = redStyle.Render("ERR_LAST")
		}

		lastRun := "-"
		if !j.LastRun.IsZero() {
			lastRun = j.LastRun.Format("01/02 15:04:05")
		}

		nextRun := "-"
		if !j.NextRun.IsZero() {
			nextRun = j.NextRun.Format("01/02 15:04:05")
		}

		fmt.Printf("%-14s %-20s %-16s %-10s %-18s %-18s %-6d\n",
			dimStyle.Render(j.ID),
			j.FlowName,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68")).Render(j.CronExpr),
			status,
			dimStyle.Render(lastRun),
			greenStyle.Render(nextRun),
			j.RunCount,
		)
	}

	fmt.Println(borderStyle.Render("──────────────────────────────────────────────────────────────────────────────────────────"))
	fmt.Printf("%s\n\n", dimStyle.Render("Start daemon runner: sx flow schedule start"))
	return nil
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

func completeFlowNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return getAvailableFlowNames(), cobra.ShellCompDirectiveNoFileComp
}

func getAvailableFlowNames() []string {
	var names []string
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(".", ".shellix", "flows"),
		filepath.Join(".", "tests", "fixtures", "flows"),
		filepath.Join(home, ".shellix", "flows"),
	}

	seen := make(map[string]bool)
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			ext := filepath.Ext(e.Name())
			if !e.IsDir() && (ext == ".yaml" || ext == ".yml") {
				base := e.Name()[:len(e.Name())-len(ext)]
				if !seen[base] {
					seen[base] = true
					names = append(names, base)
				}
			}
		}
	}
	return names
}
