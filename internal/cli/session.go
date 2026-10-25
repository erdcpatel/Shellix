package cli

import (
	"fmt"

	"shellix/internal/session"
	"shellix/internal/terminal"

	"github.com/spf13/cobra"
)

// NewSessionCmd monitors active sessions
func NewSessionCmd(app *AppContext) *cobra.Command {
	sessionCmd := &cobra.Command{
		Use:     "session",
		Aliases: []string{"tab"},
		Short:   "Active terminal tabs and remote sessions monitor",
	}

	sessionListCmd := &cobra.Command{
		Use:   "list",
		Short: "List active sessions",
		Run: func(cmd *cobra.Command, args []string) {
			sessions := app.SessionMgr.ListSessions()
			if len(sessions) == 0 {
				fmt.Println("No background sessions currently active.")
				return
			}
			fmt.Println("Active Shellix Sessions & Tabs:")
			for _, s := range sessions {
				fmt.Printf("  • [%s] %-15s (%s -> %s) started %s\n",
					s.ID, s.Title, s.Type, s.Target, s.StartedAt.Format("15:04:05"))
			}
		},
	}

	sessionCmd.AddCommand(sessionListCmd)
	return sessionCmd
}

// NewShellCmd launches the interactive PTY shell
func NewShellCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "shell",
		Short: "Launch interactive PTY shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := app.SessionMgr.RegisterSession("Local Shell", session.SessionLocal, app.Config.Shell.Default)
			defer func() { _ = app.SessionMgr.CloseSession(s.ID) }()
			return terminal.RunShell(app.Config.Shell.Default)
		},
	}
}
