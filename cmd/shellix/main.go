package main

import (
	"fmt"
	"os"

	"shellix/internal/core"
	"shellix/internal/git"
	"shellix/internal/terminal"

	"github.com/spf13/cobra"
)

func main() {
	cfg, err := core.InitConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = core.DefaultConfig()
	}

	gitEngine := git.NewEngine(cfg.Git.UseInProcessEngine)

	rootCmd := &cobra.Command{
		Use:   "shellix",
		Short: "Shellix — Blazing-fast terminal workbench",
		Long:  core.Banner() + "\nA high-performance developer terminal workbench with instant Git engine, automation flows, and productivity tools.",
		Run: func(cmd *cobra.Command, args []string) {
			// Default action: print banner & quick status
			fmt.Print(core.Banner())
			fmt.Println()
			_ = gitEngine.PrintStatus("")
			fmt.Println("\nRun 'sx --help' for commands, or 'sx shell' to launch an interactive PTY shell.")
		},
	}

	// 1. Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print Shellix version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Shellix version %s\n", core.GetVersion())
		},
	}

	// 2. Shell command
	shellCmd := &cobra.Command{
		Use:   "shell",
		Short: "Launch interactive PTY shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			return terminal.RunShell(cfg.Shell.Default)
		},
	}

	// 3. Status command
	statusCmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Quick instant repository status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return gitEngine.PrintStatus("")
		},
	}

	// 4. Git dual-engine command
	gitCmd := &cobra.Command{
		Use:                "git [args...]",
		Aliases:            []string{"g"},
		Short:              "Dual-engine Git runner (ultra-fast in-process status + native fallback)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || (len(args) == 1 && (args[0] == "status" || args[0] == "s")) {
				return gitEngine.PrintStatus("")
			}
			return gitEngine.RunPassthrough(args)
		},
	}

	rootCmd.AddCommand(versionCmd, shellCmd, statusCmd, gitCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
