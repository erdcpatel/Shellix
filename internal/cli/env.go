package cli

import (
	"fmt"
	"strings"

	"shellix/internal/envvault"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewEnvCmd constructs the project environment and secret injection CLI
func NewEnvCmd(app *AppContext) *cobra.Command {
	var showSecrets bool

	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Encrypted project environment variables and secret injection (.env.vault)",
		Long:  "AES-256-GCM encrypted local environment store backed by the native OS Vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnvList(".", showSecrets)
		},
	}

	envCmd.Flags().BoolVarP(&showSecrets, "show", "s", false, "Reveal plaintext secrets in list")

	// 1. sx env list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all variables stored in .env.vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnvList(".", showSecrets)
		},
	}
	listCmd.Flags().BoolVarP(&showSecrets, "show", "s", false, "Reveal plaintext secrets")

	// 2. sx env set KEY=VALUE [KEY2=VALUE2...]
	setCmd := &cobra.Command{
		Use:   "set <KEY=VALUE>...",
		Short: "Store or update environment variables in .env.vault",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := envvault.NewManager(".")
			if err != nil {
				return err
			}

			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			for _, arg := range args {
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid format '%s', expected KEY=VALUE", arg)
				}
				key := strings.TrimSpace(parts[0])
				val := parts[1]

				if err := mgr.Set(key, val); err != nil {
					return fmt.Errorf("failed setting %s: %w", key, err)
				}
				fmt.Printf("%s Encrypted and saved %s to .env.vault\n", successStyle.Render("✔"), key)
			}
			return nil
		},
	}

	// 3. sx env get <KEY>
	getCmd := &cobra.Command{
		Use:   "get <KEY>",
		Short: "Retrieve and decrypt an environment variable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := envvault.NewManager(".")
			if err != nil {
				return err
			}

			val, ok := mgr.Get(args[0])
			if !ok {
				return fmt.Errorf("variable '%s' not found in .env.vault", args[0])
			}
			fmt.Println(val)
			return nil
		},
	}

	// 4. sx env delete <KEY>
	deleteCmd := &cobra.Command{
		Use:     "delete <KEY>",
		Aliases: []string{"rm"},
		Short:   "Delete an environment variable from .env.vault",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := envvault.NewManager(".")
			if err != nil {
				return err
			}

			if err := mgr.Delete(args[0]); err != nil {
				return err
			}
			fmt.Printf("✨ Removed %s from .env.vault\n", args[0])
			return nil
		},
	}

	// 5. sx env run -- <command...>
	runCmd := &cobra.Command{
		Use:   "run -- <command> [args...]",
		Short: "Execute a command with decrypted .env.vault secrets injected into environment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := envvault.NewManager(".")
			if err != nil {
				return err
			}

			return mgr.RunWithEnv(args[0], args[1:])
		},
	}

	// 6. sx env export
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Emit shell export commands for eval $(sx env export)",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := envvault.NewManager(".")
			if err != nil {
				return err
			}

			summaries := mgr.ListSummaries(true)
			for _, s := range summaries {
				fmt.Printf("export %s=%q\n", s.Key, s.Value)
			}
			return nil
		},
	}

	envCmd.AddCommand(listCmd, setCmd, getCmd, deleteCmd, runCmd, exportCmd)
	return envCmd
}

func runEnvList(dir string, showSecrets bool) error {
	mgr, err := envvault.NewManager(dir)
	if err != nil {
		return err
	}

	summaries := mgr.ListSummaries(showSecrets)
	if len(summaries) == 0 {
		fmt.Println("No variables found in .env.vault. Add one with: sx env set KEY=VALUE")
		return nil
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D0D0D0"))

	fmt.Printf("%s (%d variables in .env.vault)\n", titleStyle.Render("🔐 Project Environment Vault"), len(summaries))
	fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────────"))
	fmt.Printf("  %-25s %s\n", "KEY", "VALUE")
	fmt.Printf("  %-25s %s\n", "───", "─────")

	for _, s := range summaries {
		displayVal := s.Masked
		if showSecrets {
			displayVal = s.Value
		}
		fmt.Printf("  %-25s %s\n", keyStyle.Render(s.Key), valStyle.Render(displayVal))
	}

	if !showSecrets {
		fmt.Println(dimStyle.Render("\nValues are masked by default. Pass --show to reveal plaintext."))
	}
	fmt.Println("Inject into any process with: sx env run -- <command>")
	return nil
}
