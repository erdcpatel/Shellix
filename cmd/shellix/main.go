package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"shellix/internal/core"
	"shellix/internal/flow"
	"shellix/internal/git"
	"shellix/internal/java"
	"shellix/internal/search"
	"shellix/internal/session"
	"shellix/internal/shortcuts"
	"shellix/internal/ssh"
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
	sshMgr, _ := ssh.NewProfileManager("")
	sessionMgr := session.NewManager()

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
			s := sessionMgr.RegisterSession("Local Shell", session.SessionLocal, cfg.Shell.Default)
			defer func() { _ = sessionMgr.CloseSession(s.ID) }()
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
		Short: "Automation workflow engine with parallel builds and audit logging",
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

	flowLogsCmd := &cobra.Command{
		Use:   "logs [filename]",
		Short: "Inspect flow execution audit logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				dir, _ := flow.GetLogDir()
				target := args[0]
				if !filepath.IsAbs(target) {
					target = filepath.Join(dir, target)
				}
				data, err := os.ReadFile(target)
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			logs, err := flow.ListLogs()
			if err != nil {
				return err
			}
			if len(logs) == 0 {
				fmt.Println("No flow execution logs found yet.")
				return nil
			}
			fmt.Printf("Execution Audit Logs (%d available in ~/.shellix/logs/flows):\n", len(logs))
			max := len(logs)
			if max > 10 {
				max = 10
			}
			for i := 0; i < max; i++ {
				fmt.Printf("  • %s\n", filepath.Base(logs[i]))
			}
			fmt.Println("\nView any log with: sx flow logs <filename>")
			return nil
		},
	}

	flowCmd.AddCommand(flowRunCmd, flowValidateCmd, flowListCmd, flowLogsCmd)

	// 6. Java Toolchain commands
	javaCmd := &cobra.Command{
		Use:     "java",
		Aliases: []string{"j"},
		Short:   "Java toolchain, environment detection, and build helper",
	}

	javaEnvCmd := &cobra.Command{
		Use:   "env",
		Short: "Inspect active Java runtime, compiler, JAVA_HOME, and build tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			return java.PrintEnv(".")
		},
	}

	javaBuildCmd := &cobra.Command{
		Use:                "build [args...]",
		Short:              "Build project automatically using Maven or Gradle wrapper",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return java.Build(".", args)
		},
	}

	javaRunCmd := &cobra.Command{
		Use:                "run <file.java> [args...]",
		Short:              "Execute a single-file Java source file directly",
		DisableFlagParsing: true,
		Args:               cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return java.RunSource(args[0], args[1:])
		},
	}
	javaCmd.AddCommand(javaEnvCmd, javaBuildCmd, javaRunCmd)

	// 7. SSH commands
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Multi-server SSH connection manager and parallel command executor",
	}

	var sshHost, sshUser, sshKey, sshPass string
	var sshPort int

	sshAddCmd := &cobra.Command{
		Use:   "add <alias>",
		Short: "Save a remote Linux/Unix server target",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := ssh.HostProfile{
				Name:     args[0],
				Host:     sshHost,
				Port:     sshPort,
				User:     sshUser,
				KeyPath:  sshKey,
				Password: sshPass,
			}
			if err := sshMgr.AddHost(p); err != nil {
				return err
			}
			fmt.Printf("✔ Saved SSH target '%s' (%s@%s:%d)\n", p.Name, p.User, p.Host, p.Port)
			return nil
		},
	}
	sshAddCmd.Flags().StringVar(&sshHost, "host", "", "Remote server hostname or IP")
	sshAddCmd.Flags().StringVar(&sshUser, "user", "", "Remote SSH user")
	sshAddCmd.Flags().IntVar(&sshPort, "port", 22, "SSH port")
	sshAddCmd.Flags().StringVar(&sshKey, "key", "", "Path to private key (defaults to ~/.ssh/id_rsa or id_ed25519)")
	sshAddCmd.Flags().StringVar(&sshPass, "pass", "", "Password (optional)")
	_ = sshAddCmd.MarkFlagRequired("host")
	_ = sshAddCmd.MarkFlagRequired("user")

	sshListCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured SSH server profiles",
		Run: func(cmd *cobra.Command, args []string) {
			hosts := sshMgr.ListHosts()
			if len(hosts) == 0 {
				fmt.Println("No SSH hosts configured. Use 'sx ssh add <name> --host <ip> --user <user>' to add one.")
				return
			}
			fmt.Println("Configured SSH Server Targets:")
			for _, h := range hosts {
				fmt.Printf("  • %-16s %s@%s:%d\n", h.Name, h.User, h.Host, h.Port)
			}
		},
	}

	sshConnectCmd := &cobra.Command{
		Use:   "connect <alias>",
		Short: "Connect to a remote server with an interactive PTY session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hp, ok := sshMgr.GetHost(args[0])
			if !ok {
				return fmt.Errorf("host profile '%s' not found", args[0])
			}
			s := sessionMgr.RegisterSession(hp.Name, session.SessionSSH, hp.Host)
			defer func() { _ = sessionMgr.CloseSession(s.ID) }()
			return ssh.ConnectInteractive(hp)
		},
	}

	var targetHostsStr string
	sshExecCmd := &cobra.Command{
		Use:   "exec \"<command>\"",
		Short: "Execute a command concurrently across multiple Linux servers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetHostsStr == "" {
				return fmt.Errorf("please provide targets via --hosts <h1,h2,...>")
			}
			targets := strings.Split(targetHostsStr, ",")
			var profiles []ssh.HostProfile
			for _, t := range targets {
				name := strings.TrimSpace(t)
				hp, ok := sshMgr.GetHost(name)
				if !ok {
					return fmt.Errorf("configured host '%s' not found", name)
				}
				profiles = append(profiles, hp)
			}

			_ = ssh.MultiExec(profiles, args[0])
			return nil
		},
	}
	sshExecCmd.Flags().StringVar(&targetHostsStr, "hosts", "", "Comma-separated list of host aliases")
	_ = sshExecCmd.MarkFlagRequired("hosts")

	sshCmd.AddCommand(sshAddCmd, sshListCmd, sshConnectCmd, sshExecCmd)

	// 8. Session / Tab manager commands
	sessionCmd := &cobra.Command{
		Use:     "session",
		Aliases: []string{"tab"},
		Short:   "Active terminal tabs and remote sessions monitor",
	}

	sessionListCmd := &cobra.Command{
		Use:   "list",
		Short: "List active sessions",
		Run: func(cmd *cobra.Command, args []string) {
			sessions := sessionMgr.ListSessions()
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

	// 9. Search command
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

	// 10. Bookmarks command
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

	// 11. Snippets command
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

	rootCmd.AddCommand(versionCmd, shellCmd, statusCmd, gitCmd, flowCmd, javaCmd, sshCmd, sessionCmd, searchCmd, bmCmd, snipCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
