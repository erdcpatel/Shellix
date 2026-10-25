package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"shellix/internal/history"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewHistoryCmd constructs the command history management CLI
func NewHistoryCmd(app *AppContext) *cobra.Command {
	var limit int

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Search and inspect command execution history and statistics",
		Long:  "Fast, in-memory searchable command execution history with deduplication and metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryList(app, limit)
		},
	}

	historyCmd.PersistentFlags().IntVarP(&limit, "limit", "n", 20, "Maximum number of entries to display")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recent command history",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryList(app, limit)
		},
	}

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search command history matching keyword",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistorySearch(app, args[0], limit)
		},
	}

	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Display command history execution metrics and top commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistoryStats(app)
		},
	}

	var skipConfirm bool
	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear all saved command history",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !skipConfirm {
				fmt.Print("Are you sure you want to clear all Shellix command history? [y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				ans, _ := reader.ReadString('\n')
				ans = strings.TrimSpace(strings.ToLower(ans))
				if ans != "y" && ans != "yes" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			store, err := getHistoryStore(app)
			if err != nil {
				return err
			}
			if err := store.Clear(); err != nil {
				return fmt.Errorf("failed to clear history: %w", err)
			}
			fmt.Println("✨ Command history cleared successfully.")
			return nil
		},
	}
	clearCmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")

	historyCmd.AddCommand(listCmd, searchCmd, statsCmd, clearCmd)
	return historyCmd
}

func getHistoryStore(app *AppContext) (*history.Store, error) {
	if app.HistoryStore != nil {
		return app.HistoryStore, nil
	}
	store, err := history.NewStore("")
	if err != nil {
		return nil, err
	}
	app.HistoryStore = store
	return store, nil
}

func runHistoryList(app *AppContext, limit int) error {
	store, err := getHistoryStore(app)
	if err != nil {
		return err
	}

	entries := store.GetRecent(limit)
	if len(entries) == 0 {
		fmt.Println("No command history recorded yet.")
		return nil
	}

	renderHistoryTable("Recent Command History", entries, "")
	return nil
}

func runHistorySearch(app *AppContext, query string, limit int) error {
	store, err := getHistoryStore(app)
	if err != nil {
		return err
	}

	entries := store.Search(query, limit)
	if len(entries) == 0 {
		fmt.Printf("No command history found matching '%s'.\n", query)
		return nil
	}

	renderHistoryTable(fmt.Sprintf("History Search: '%s'", query), entries, query)
	return nil
}

func runHistoryStats(app *AppContext) error {
	store, err := getHistoryStore(app)
	if err != nil {
		return err
	}

	stats := store.GetStats()

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	statLabelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0"))
	statValStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA500"))
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4500"))
	borderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3A3F58")).Padding(0, 1)

	rateColor := successStyle
	if stats.SuccessRate < 80.0 {
		rateColor = warnStyle
	}
	if stats.SuccessRate < 50.0 {
		rateColor = failStyle
	}

	summary := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("📊 Shellix Command Execution Analytics"),
		statLabelStyle.Render("─────────────────────────────────────────────────────────────"),
		fmt.Sprintf("%-22s %s", statLabelStyle.Render("Total Commands:"), statValStyle.Render(fmt.Sprintf("%d", stats.TotalCommands))),
		fmt.Sprintf("%-22s %s", statLabelStyle.Render("Unique Commands:"), statValStyle.Render(fmt.Sprintf("%d", stats.UniqueCommands))),
		fmt.Sprintf("%-22s %s", statLabelStyle.Render("Successful / Failed:"), fmt.Sprintf("%s / %s", successStyle.Render(fmt.Sprintf("%d", stats.SuccessCount)), failStyle.Render(fmt.Sprintf("%d", stats.FailureCount)))),
		fmt.Sprintf("%-22s %s", statLabelStyle.Render("Success Rate:"), rateColor.Render(fmt.Sprintf("%.1f%%", stats.SuccessRate))),
		fmt.Sprintf("%-22s %s", statLabelStyle.Render("Average Latency:"), statValStyle.Render(stats.AvgDuration.Round(time.Millisecond).String())),
	)

	fmt.Println(borderStyle.Render(summary))
	fmt.Println()

	if len(stats.TopCommands) > 0 {
		fmt.Println(headerStyle.Render("🔥 Top Executed Commands:"))
		fmt.Printf("   %-4s %-8s %-12s %-14s %s\n", "RANK", "COUNT", "SUCCESS", "LAST USED", "COMMAND")
		fmt.Printf("   %-4s %-8s %-12s %-14s %s\n", "────", "─────", "───────", "─────────", "────────────────────────────────")

		for i, tc := range stats.TopCommands {
			succRate := 0.0
			if tc.Count > 0 {
				succRate = float64(tc.Successes) / float64(tc.Count) * 100.0
			}
			rateStr := fmt.Sprintf("%.0f%%", succRate)
			if succRate >= 90.0 {
				rateStr = successStyle.Render(rateStr)
			} else {
				rateStr = warnStyle.Render(rateStr)
			}

			relTime := formatRelativeTime(tc.LastUsed)
			cmdDisplay := tc.Command
			if len(cmdDisplay) > 45 {
				cmdDisplay = cmdDisplay[:42] + "..."
			}

			fmt.Printf("   #%-3d %-8d %-12s %-14s %s\n",
				i+1, tc.Count, rateStr, relTime, cmdDisplay)
		}
		fmt.Println()
	}

	return nil
}

func renderHistoryTable(title string, entries []history.HistoryEntry, highlightQuery string) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	successIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render("✔")
	failIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")).Render("✖")
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true)
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Bold(true)

	fmt.Printf("%s (%d entries)\n", titleStyle.Render("📜 "+title), len(entries))
	fmt.Println(dimStyle.Render("────────────────────────────────────────────────────────────────────────────────────────"))
	fmt.Printf("  %-3s %-10s %-10s %-20s %s\n", "ST", "TIME", "DURATION", "DIRECTORY", "COMMAND")
	fmt.Printf("  %-3s %-10s %-10s %-20s %s\n", "──", "────", "────────", "─────────", "──────────────────────────────────")

	home, _ := os.UserHomeDir()

	for _, e := range entries {
		icon := successIcon
		if e.ExitCode != 0 {
			icon = failIcon
		}

		timeStr := formatRelativeTime(e.Timestamp)
		durStr := e.Duration.Round(time.Millisecond).String()
		if e.Duration < time.Millisecond {
			durStr = "<1ms"
		}

		dirStr := e.Directory
		if home != "" && strings.HasPrefix(dirStr, home) {
			dirStr = "~" + strings.TrimPrefix(dirStr, home)
		}
		if len(dirStr) > 18 {
			dirStr = "…" + dirStr[len(dirStr)-17:]
		}

		cmdStr := e.Command
		if highlightQuery != "" {
			idx := strings.Index(strings.ToLower(cmdStr), strings.ToLower(highlightQuery))
			if idx >= 0 {
				matched := cmdStr[idx : idx+len(highlightQuery)]
				cmdStr = cmdStr[:idx] + highlightStyle.Render(matched) + cmdStr[idx+len(highlightQuery):]
			}
		}

		countBadge := ""
		if e.Count > 1 {
			countBadge = countStyle.Render(fmt.Sprintf(" [x%d]", e.Count))
		}

		fmt.Printf("  %s   %-10s %-10s %-20s %s%s\n",
			icon, dimStyle.Render(timeStr), dimStyle.Render(durStr), dimStyle.Render(dirStr), cmdStr, countBadge)
	}
	fmt.Println()
}

func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return t.Format("2006-01-02 15:04")
}
