package cli

import (
	"github.com/spf13/cobra"
)

// NewGitCmd handles dual-engine Git operations
func NewGitCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:                "git [args...]",
		Aliases:            []string{"g"},
		Short:              "Dual-engine Git runner (ultra-fast in-process status + native fallback)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || (len(args) == 1 && (args[0] == "status" || args[0] == "s")) {
				return app.GitEngine.PrintStatus("")
			}
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
