package cli

import (
	"shellix/internal/git"

	"github.com/spf13/cobra"
)

// NewGitCmd handles dual-engine Git operations and interactive TUI
func NewGitCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:                "git [args...]",
		Aliases:            []string{"g"},
		Short:              "Dual-engine Git runner & full-screen interactive TUI dashboard",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. If run with no args (e.g. 'sx g' or 'sx git'): launch interactive TUI dashboard
			if len(args) == 0 {
				return git.LaunchTUI(".")
			}

			// 2. If run with 'status' or 's': print instant sub-2ms CLI status
			if len(args) == 1 && (args[0] == "status" || args[0] == "s") {
				return app.GitEngine.PrintStatus("")
			}

			// 3. Otherwise: transparent native passthrough
			return app.GitEngine.RunPassthrough(args)
		},
	}
}

// NewStatusCmd handles instant repo status
func NewStatusCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Quick instant repository status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.GitEngine.PrintStatus("")
		},
	}
}
