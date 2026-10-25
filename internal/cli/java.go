package cli

import (
	"shellix/internal/java"

	"github.com/spf13/cobra"
)

// NewJavaCmd handles Java discovery and build commands
func NewJavaCmd(app *AppContext) *cobra.Command {
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
	return javaCmd
}
