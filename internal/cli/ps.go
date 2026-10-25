package cli

import (
	"fmt"
	"time"

	"shellix/internal/flow"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewPSCmd creates the top-level process and background service monitor
func NewPSCmd(app *AppContext) *cobra.Command {
	psCmd := &cobra.Command{
		Use:     "ps",
		Aliases: []string{"proc", "processes"},
		Short:   "List active background daemons, scheduled runners, and remote sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
			borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
			headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))

			fmt.Println(titleStyle.Render("⚡ Shellix Background Process & Service Monitor"))
			fmt.Println(borderStyle.Render("─────────────────────────────────────────────────────────────────────────────"))

			// 1. Scheduler Daemon Status
			daemonInfo, isAlive, err := flow.GetDaemonStatus()
			if err != nil {
				fmt.Printf("Scheduler Status: error checking PID: %v\n", err)
			} else if isAlive && daemonInfo != nil {
				uptime := time.Since(daemonInfo.StartedAt).Round(time.Second)
				heartbeat := time.Since(daemonInfo.LastHeartbeat).Round(time.Second)
				fmt.Printf(" %s  %-24s PID: %-8d Uptime: %-12s Heartbeat: %s ago\n",
					greenStyle.Render("●"),
					headerStyle.Render("Flow Scheduler Daemon"),
					daemonInfo.PID,
					uptime,
					heartbeat,
				)
				fmt.Printf("    • Host: %s  • Logs: %s\n", dimStyle.Render(daemonInfo.Hostname), dimStyle.Render(flow.GetSchedulerLogPath()))
			} else {
				fmt.Printf(" %s  %-24s %s\n",
					dimStyle.Render("○"),
					headerStyle.Render("Flow Scheduler Daemon"),
					dimStyle.Render("STOPPED (start with 'sx flow schedule start')"),
				)
			}

			// 2. Native OS Background Service Status
			svcStatus, svcErr := flow.DaemonServiceStatus()
			if svcErr == nil {
				fmt.Printf(" %s  %-24s %s\n",
					dimStyle.Render("⚙"),
					headerStyle.Render("OS Service Hook"),
					dimStyle.Render(svcStatus),
				)
			}

			fmt.Println(borderStyle.Render("─────────────────────────────────────────────────────────────────────────────"))

			// 3. Active Sessions / Tunnels
			sessions := app.SessionMgr.ListSessions()
			if len(sessions) > 0 {
				fmt.Printf("Active Interactive Sessions (%d):\n", len(sessions))
				for _, s := range sessions {
					fmt.Printf("  • [%s] %-15s (%s -> %s) started %s\n",
						s.ID, s.Title, s.Type, s.Target, s.StartedAt.Format("15:04:05"))
				}
				fmt.Println(borderStyle.Render("─────────────────────────────────────────────────────────────────────────────"))
			}

			// 4. Quick Help
			fmt.Println(dimStyle.Render("Useful Commands:"))
			fmt.Println(dimStyle.Render("  • sx flow schedule logs -f   Stream background scheduler logs"))
			fmt.Println(dimStyle.Render("  • sx flow schedule daemon    Manage OS background service"))
			fmt.Println()
			return nil
		},
	}

	return psCmd
}
