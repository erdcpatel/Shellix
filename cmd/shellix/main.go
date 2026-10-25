package main

import (
	"fmt"
	"os"
	"path/filepath"

	"shellix/internal/core"
	"shellix/internal/flow"
	"shellix/internal/git"
	"shellix/internal/search"
	"shellix/internal/shortcuts"
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
	shortcutMgr, _ := shortcuts.NewManager("")

	rootCmd := &cobra.Command{
		Use:   "shellix",
		Short: "Shellix — Blazing-fast terminal workbench",
		Long:  core.Banner() + "\nA high-performance developer terminal workbench with instant Git engine, automation flows, and productivity tools.",
		Run: func(cmd *cobra.Command, args []string) {
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

	// 5. Flow Automation commands
	flowCmd := &cobra.Command{
		Use:   "flow",
		Short: "Automation workflow engine",
	}

	flowRunCmd := &cobra.Command{
		Use:   "run <file.yaml>",
		Short: "Execute an automation flow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := flow.ParseFile(args[0])
			if err != nil {
				return err
			}

			cwd, _ := os.Getwd()
			branch := "unknown"
			if st, err := gitEngine.GetStatus(cwd); err == nil {
				branch = st.Branch
			}

			ctx := flow.NewContext(cwd, branch)
			exec := flow.NewExecutor(f, ctx)
			_, err = exec.Execute()
			return err
		},
	}

	flowValidateCmd := &cobra.Command{
		Use:   "validate <file.yaml>",
		Short: "Validate a flow YAML without running",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := flow.ParseFile(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("✔ Flow '%s' is valid (%d steps)\n", f.Name, len(f.Steps))
			return nil
		},
	}

	flowListCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed flows in ~/.shellix/flows",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			flowDir := filepath.Join(home, ".shellix", "flows")
			entries, err := os.ReadDir(flowDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No flows installed yet. Create flows under ~/.shellix/flows/*.yaml")
					return nil
				}
				return err
			}
			fmt.Printf("Installed flows in %s:\n", flowDir)
			for _, e := range entries {
				if !e.IsDir() && (filepath.Ext(e.Name()) == ".yaml" || filepath.Ext(e.Name()) == ".yml") {
					fmt.Printf("  • %s\n", e.Name())
				}
			}
			return nil
		},
	}
	flowCmd.AddCommand(flowRunCmd, flowValidateCmd, flowListCmd)

	// 6. Search command
	var searchPaths []string
	var searchRegex bool
	var searchMax int

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Blazing-fast multi-repo code and text search",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine := search.NewEngine()
			opts := search.SearchOptions{
				Query:      args[0],
				Paths:      searchPaths,
				IsRegex:    searchRegex,
				MaxResults: searchMax,
			}
			summary, err := engine.Search(opts)
			if err != nil {
				return err
			}
			search.PrintResults(summary, args[0])
			return nil
		},
	}
	searchCmd.Flags().StringSliceVarP(&searchPaths, "path", "p", []string{"."}, "Path(s) to search")
	searchCmd.Flags().BoolVarP(&searchRegex, "regex", "r", false, "Enable regular expression search")
	searchCmd.Flags().IntVarP(&searchMax, "max", "m", 200, "Maximum results to display")

	// 7. Bookmarks command
	bmCmd := &cobra.Command{
		Use:   "bm",
		Short: "Filesystem bookmarks manager",
	}

	bmAddCmd := &cobra.Command{
		Use:   "add <name> [path]",
		Short: "Bookmark a directory",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 2 {
				path = args[1]
			}
			if err := shortcutMgr.AddBookmark(args[0], path); err != nil {
				return err
			}
			bm, _ := shortcutMgr.GetBookmark(args[0])
			fmt.Printf("✔ Bookmarked '%s' -> %s\n", args[0], bm.Path)
			return nil
		},
	}

	bmListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all bookmarks",
		Run: func(cmd *cobra.Command, args []string) {
			bms := shortcutMgr.ListBookmarks()
			if len(bms) == 0 {
				fmt.Println("No bookmarks saved. Use 'sx bm add <name> [path]' to add one.")
				return
			}
			fmt.Println("Saved Bookmarks:")
			for _, b := range bms {
				fmt.Printf("  • %-15s -> %s\n", b.Name, b.Path)
			}
		},
	}

	bmGetCmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Print target path for a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bm, ok := shortcutMgr.GetBookmark(args[0])
			if !ok {
				return fmt.Errorf("bookmark '%s' not found", args[0])
			}
			fmt.Println(bm.Path)
			return nil
		},
	}

	bmRmCmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return shortcutMgr.DeleteBookmark(args[0])
		},
	}
	bmCmd.AddCommand(bmAddCmd, bmListCmd, bmGetCmd, bmRmCmd)

	// 8. Snippets command
	snipCmd := &cobra.Command{
		Use:   "snip",
		Short: "Command snippets manager",
	}

	var snipDesc string
	snipAddCmd := &cobra.Command{
		Use:   "add <name> <command>",
		Short: "Save a reusable command snippet",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := shortcutMgr.AddSnippet(args[0], args[1], snipDesc); err != nil {
				return err
			}
			fmt.Printf("✔ Saved snippet '%s': %s\n", args[0], args[1])
			return nil
		},
	}
	snipAddCmd.Flags().StringVarP(&snipDesc, "desc", "d", "", "Description of snippet")

	snipListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all command snippets",
		Run: func(cmd *cobra.Command, args []string) {
			snips := shortcutMgr.ListSnippets()
			if len(snips) == 0 {
				fmt.Println("No snippets saved. Use 'sx snip add <name> <command>' to create one.")
				return
			}
			fmt.Println("Saved Snippets:")
			for _, s := range snips {
				desc := ""
				if s.Description != "" {
					desc = fmt.Sprintf(" (%s)", s.Description)
				}
				fmt.Printf("  • %-15s : %s%s\n", s.Name, s.Command, desc)
			}
		},
	}

	snipRunCmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Execute a command snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return shortcutMgr.RunSnippet(args[0])
		},
	}

	snipRmCmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a command snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return shortcutMgr.DeleteSnippet(args[0])
		},
	}
	snipCmd.AddCommand(snipAddCmd, snipListCmd, snipRunCmd, snipRmCmd)

	rootCmd.AddCommand(versionCmd, shellCmd, statusCmd, gitCmd, flowCmd, searchCmd, bmCmd, snipCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
