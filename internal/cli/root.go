package cli

import (
	"fmt"

	"shellix/internal/core"
	"shellix/internal/session"
	"shellix/internal/terminal"

	"github.com/spf13/cobra"
)

// NewRootCmd assembles the root Shellix CLI router
func NewRootCmd(app *AppContext) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "shellix",
		Aliases: []string{"sx", "slx"},
		Version: core.GetVersion(),
		Short:   "Shellix — Blazing-fast developer terminal workbench",
		Long:    core.Banner() + "\nA high-performance developer terminal workbench with instant Git engine, automation flows, and productivity tools.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Print(core.Banner())
			fmt.Println()
			if app != nil && app.GitEngine != nil {
				_ = app.GitEngine.PrintStatus("")
				fmt.Println()
			}
			fmt.Println("⚡ Quick Superpowers:")
			fmt.Println("  • sx g             Launch interactive Git TUI dashboard (staging & diffs)")
			fmt.Println("  • sx flow          List and run automation flows with parallel builds")
			fmt.Println("  • sx palette       Universal fuzzy command palette across all tools")
			fmt.Println("  • sx history       Search execution history & view command statistics")
			fmt.Println("  • sx java ls       List all discovered JDKs & switch JAVA_HOME")
			fmt.Println("  • sx search <q>    High-speed parallel code search (.gitignore-aware)")
			fmt.Println("  • sx doctor        Comprehensive system health & environment diagnostics")
			fmt.Println("  • sx vault info    Inspect native OS Credential Manager")
			fmt.Println("  • sx shell         Launch interactive subshell with VT processing")
			fmt.Println("\nRun 'sx --help' (or 'slx --help') for the complete command tree.")
			fmt.Println()

			defaultShell := ""
			if app != nil && app.Config != nil {
				defaultShell = app.Config.Shell.Default
			}

			runner := terminal.RunShell
			if app != nil && app.ShellRunner != nil {
				runner = app.ShellRunner
			}

			if app != nil && app.SessionMgr != nil {
				s := app.SessionMgr.RegisterSession("Interactive Shell", session.SessionLocal, defaultShell)
				defer func() { _ = app.SessionMgr.CloseSession(s.ID) }()
			}

			return runner(defaultShell)
		},
	}

	rootCmd.AddCommand(
		NewVersionCmd(app),
		NewShellCmd(app),
		NewStatusCmd(app),
		NewGitCmd(app),
		NewBranchCmd(app),
		NewStashCmd(app),
		NewGitHubCmd(app),
		NewPRCmd(app),
		NewCICmd(app),
		NewIssueCmd(app),
		NewFlowCmd(app),
		NewJavaCmd(app),
		NewSSHCmd(app),
		NewVaultCmd(app),
		NewEnvCmd(app),
		NewSessionCmd(app),
		NewSearchCmd(app),
		NewBookmarkCmd(app),
		NewSnippetCmd(app),
		NewHistoryCmd(app),
		NewHTTPCmd(app),
		NewPaletteCmd(app),
		NewDoctorCmd(app),
		NewInitCmd(app),
		NewJSONCmd(app),
		NewViewCmd(app),
		NewParquetCmd(app),
		NewCompletionCmd(),
		NewPSCmd(app),
		NewWebCmd(app),
	)

	return rootCmd
}
