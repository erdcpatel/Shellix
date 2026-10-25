package cli

import (
	"fmt"

	"shellix/internal/core"

	"github.com/spf13/cobra"
)

// NewRootCmd assembles the root Shellix CLI router
func NewRootCmd(app *AppContext) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "shellix",
		Short: "Shellix — Blazing-fast developer terminal workbench",
		Long:  core.Banner() + "\nA high-performance developer terminal workbench with instant Git engine, automation flows, and productivity tools.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(core.Banner())
			fmt.Println()
			_ = app.GitEngine.PrintStatus("")
			fmt.Println("\nRun 'sx --help' for commands, or 'sx shell' to launch an interactive PTY shell.")
		},
	}

	rootCmd.AddCommand(
		NewVersionCmd(app),
		NewShellCmd(app),
		NewStatusCmd(app),
		NewGitCmd(app),
		NewGitHubCmd(app),
		NewPRCmd(app),
		NewCICmd(app),
		NewFlowCmd(app),
		NewJavaCmd(app),
		NewSSHCmd(app),
		NewVaultCmd(app),
		NewSessionCmd(app),
		NewSearchCmd(app),
		NewBookmarkCmd(app),
		NewSnippetCmd(app),
		NewDoctorCmd(app),
	)

	return rootCmd
}
