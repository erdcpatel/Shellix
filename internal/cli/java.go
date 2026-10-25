package cli

import (
	"fmt"

	"shellix/internal/java"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewJavaCmd handles Java discovery, build, and multi-version switching
func NewJavaCmd(app *AppContext) *cobra.Command {
	javaCmd := &cobra.Command{
		Use:     "java",
		Aliases: []string{"j"},
		Short:   "Java toolchain, multi-version switching, and build automation",
	}

	// 1. sx java env
	javaEnvCmd := &cobra.Command{
		Use:   "env",
		Short: "Inspect active Java runtime, compiler, JAVA_HOME, and build tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			return java.PrintEnv(".")
		},
	}

	// 2. sx java list / ls
	javaListCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all installed Java JDK versions on this machine",
		RunE: func(cmd *cobra.Command, args []string) error {
			jdks, err := java.DiscoverAllJDKs()
			if err != nil {
				return err
			}

			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
			activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			verStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68"))
			pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

			fmt.Println(titleStyle.Render(fmt.Sprintf("☕ Installed Java JDKs (%d discovered)", len(jdks))))
			fmt.Println()

			if len(jdks) == 0 {
				fmt.Println("No JDKs discovered in standard paths.")
				return nil
			}

			for _, j := range jdks {
				marker := "  "
				activeLabel := ""
				if j.IsActive {
					marker = activeStyle.Render("● ")
					activeLabel = activeStyle.Render(" [CURRENT ACTIVE]")
				}

				fmt.Printf("%s%s (%s) — %s%s\n",
					marker,
					verStyle.Render(fmt.Sprintf("Java %-6s", j.Version)),
					j.Vendor,
					pathStyle.Render(j.HomePath),
					activeLabel,
				)
			}

			fmt.Println("\nTo switch version in current shell: eval $(sx j use <version>)")
			fmt.Println("To launch subshell with version:    sx j shell <version>")
			return nil
		},
	}

	// 3. sx java use <version>
	var exportOnly bool
	javaUseCmd := &cobra.Command{
		Use:   "use <version/vendor>",
		Short: "Switch active Java version (generates shell export command)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jdk, err := java.FindMatchingJDK(args[0])
			if err != nil {
				return err
			}

			exportCmd := java.GenerateShellExport(jdk)

			if exportOnly {
				fmt.Println(exportCmd)
				return nil
			}

			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7"))

			fmt.Printf("%s Java %s (%s)\n", successStyle.Render("✔ Selected:"), jdk.Version, jdk.Vendor)
			fmt.Printf("  JAVA_HOME: %s\n\n", jdk.HomePath)
			fmt.Printf("To activate in your current terminal session, run:\n")
			fmt.Printf("  %s\n\n", hintStyle.Render(fmt.Sprintf("eval $(sx j use %s -e)", args[0])))
			fmt.Printf("Or launch an immediate subshell:\n")
			fmt.Printf("  %s\n", hintStyle.Render(fmt.Sprintf("sx j shell %s", args[0])))
			return nil
		},
	}
	javaUseCmd.Flags().BoolVarP(&exportOnly, "export", "e", false, "Output shell export command only (for eval)")

	// 4. sx java shell <version>
	javaShellCmd := &cobra.Command{
		Use:   "shell <version/vendor>",
		Short: "Launch an interactive terminal shell configured with the selected Java version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jdk, err := java.FindMatchingJDK(args[0])
			if err != nil {
				return err
			}

			activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			fmt.Printf("%s Launching subshell with Java %s (%s)...\n",
				activeStyle.Render("⚡ [JAVA SHELL]"), jdk.Version, jdk.HomePath)
			fmt.Println("Type 'exit' to return to your previous shell environment.")
			fmt.Println()

			return java.LaunchSubshellWithJDK(jdk, app.Config.Shell.Default)
		},
	}

	// 5. sx java build
	javaBuildCmd := &cobra.Command{
		Use:                "build [args...]",
		Short:              "Build project automatically using Maven or Gradle wrapper",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return java.Build(".", args)
		},
	}

	// 6. sx java run <file.java>
	javaRunCmd := &cobra.Command{
		Use:                "run <file.java> [args...]",
		Short:              "Execute a single-file Java source file directly",
		DisableFlagParsing: true,
		Args:               cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return java.RunSource(args[0], args[1:])
		},
	}

	javaCmd.AddCommand(javaEnvCmd, javaListCmd, javaUseCmd, javaShellCmd, javaBuildCmd, javaRunCmd)
	return javaCmd
}
