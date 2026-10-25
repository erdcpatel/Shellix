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
			fmt.Println()
			fmt.Println("⚡ Quick Superpowers:")
			fmt.Println("  • sx g             Launch interactive Git TUI dashboard (staging & diffs)")
			fmt.Println("  • sx flow          List and run automation flows with parallel builds")
			fmt.Println("  • sx java ls       List all discovered JDKs & switch JAVA_HOME")
			fmt.Println("  • sx search <q>    High-speed parallel code search (.gitignore-aware)")
			fmt.Println("  • sx doctor        Comprehensive system health & environment diagnostics")
			fmt.Println("  • sx vault info    Inspect native OS Credential Manager")
			fmt.Println("  • sx shell         Launch interactive subshell with VT processing")
			fmt.Println("\nRun 'sx --help' for the complete command tree.")
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
