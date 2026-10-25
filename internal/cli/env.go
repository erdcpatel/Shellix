package cli

import (
	"fmt"
	"os"
	"strings"

	"shellix/internal/envvault"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewEnvCmd constructs the project environment and secret injection CLI
func NewEnvCmd(app *AppContext) *cobra.Command {
	var showSecrets bool
	var targetEnv string

	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Encrypted project environment variables and secret injection (.env.vault)",
		Long:  "AES-256-GCM encrypted local environment store backed by the native OS Vault (WinCred/Keychain)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnvList(".", targetEnv, showSecrets)
		},
	}

	envCmd.PersistentFlags().StringVarP(&targetEnv, "env", "e", "default", "Target environment profile (default, dev, staging, prod)")
	envCmd.Flags().BoolVarP(&showSecrets, "show", "s", false, "Reveal plaintext secrets in list")

	keyCompletion := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cwd, _ := os.Getwd()
		mgr, err := envvault.NewManager(cwd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		sums := mgr.ListSummariesEnv(targetEnv, false)
		var keys []string
		for _, s := range sums {
			keys = append(keys, s.Key)
		}
		return keys, cobra.ShellCompDirectiveNoFileComp
	}

	// 1. sx env list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all variables stored in .env.vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnvList(".", targetEnv, showSecrets)
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

				if err := mgr.SetEnv(targetEnv, key, val); err != nil {
					return fmt.Errorf("failed setting %s: %w", key, err)
				}
				fmt.Printf("%s Encrypted and saved %s to .env.vault [%s]\n", successStyle.Render("✔"), key, targetEnv)
			}
			return nil
		},
	}

	// 3. sx env get <KEY>
	getCmd := &cobra.Command{
		Use:               "get <KEY>",
		Short:             "Retrieve and decrypt an environment variable",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: keyCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := envvault.NewManager(".")
			if err != nil {
				return err
			}

			val, ok := mgr.GetEnv(targetEnv, args[0])
			if !ok {
				return fmt.Errorf("variable '%s' not found in .env.vault [%s]", args[0], targetEnv)
			}
			fmt.Println(val)
			return nil
		},
	}

	// 4. sx env delete <KEY>
	deleteCmd := &cobra.Command{
		Use:               "delete <KEY>",
		Aliases:           []string{"rm"},
		Short:             "Delete an environment variable from .env.vault",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: keyCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := envvault.NewManager(".")
			if err != nil {
				return err
			}

			if err := mgr.DeleteEnv(targetEnv, args[0]); err != nil {
				return err
			}
			fmt.Printf("✨ Removed %s from .env.vault [%s]\n", args[0], targetEnv)
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

			return mgr.RunWithEnv(targetEnv, args[0], args[1:])
		},
	}

	// 6. sx env export
	var exportShell string
	var exportUnset bool
	exportCmd := &cobra.Command{
		Use:   "export",
		Short: "Emit shell export commands for session injection or hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := envvault.NewManager(".")
			if err != nil {
				return err
			}

			summaries := mgr.ListSummariesEnv(targetEnv, true)
			isPowerShell := strings.EqualFold(exportShell, "powershell") || strings.EqualFold(exportShell, "pwsh")

			for _, s := range summaries {
				if exportUnset {
					if isPowerShell {
						fmt.Printf("Remove-Item -ErrorAction SilentlyContinue env:%s\n", s.Key)
					} else {
						fmt.Printf("unset %s\n", s.Key)
					}
				} else {
					if isPowerShell {
						// Escape PowerShell quotes and string expansion
						escaped := strings.ReplaceAll(s.Value, "`", "``")
						escaped = strings.ReplaceAll(escaped, `"`, "`\"")
						escaped = strings.ReplaceAll(escaped, "$", "`$")
						fmt.Printf("$env:%s = \"%s\"\n", s.Key, escaped)
					} else {
						fmt.Printf("export %s=%q\n", s.Key, s.Value)
					}
				}
			}
			return nil
		},
	}
	exportCmd.Flags().StringVarP(&exportShell, "shell", "s", "zsh", "Shell syntax (powershell, zsh, bash)")
	exportCmd.Flags().BoolVarP(&exportUnset, "unset", "u", false, "Emit unset statements instead of export")

	// 7. sx env hook <powershell|zsh|bash>
	hookCmd := &cobra.Command{
		Use:       "hook [powershell|zsh|bash]",
		Short:     "Generate direnv-style directory auto-load hook script",
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: []string{"powershell", "zsh", "bash"},
		RunE: func(cmd *cobra.Command, args []string) error {
			script, err := envvault.GenerateHook(args[0])
			if err != nil {
				return err
			}
			fmt.Print(script)
			return nil
		},
	}

	// 8. sx env rotate-key
	rotateKeyCmd := &cobra.Command{
		Use:   "rotate-key",
		Short: "Re-encrypt .env.vault with a fresh 256-bit AES key in native OS Vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := envvault.NewManager(".")
			if err != nil {
				return err
			}

			newHex, err := mgr.RotateMasterKey()
			if err != nil {
				return err
			}

			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			fmt.Printf("%s Master encryption key rotated successfully.\n", successStyle.Render("✔"))
			fmt.Printf("  • New Key Fingerprint: %s...\n", newHex[:12])
			fmt.Println("  • Updated in native OS Vault and .env.vault re-encrypted with fresh nonces.")
			return nil
		},
	}

	envCmd.AddCommand(listCmd, setCmd, getCmd, deleteCmd, runCmd, exportCmd, hookCmd, rotateKeyCmd)
	return envCmd
}

func runEnvList(dir, envName string, showSecrets bool) error {
	mgr, err := envvault.NewManager(dir)
	if err != nil {
		return err
	}

	summaries := mgr.ListSummariesEnv(envName, showSecrets)
	allEnvs := mgr.ListEnvironments()

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#D0D0D0"))
	envBadgeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E0AF68"))

	fmt.Printf("%s [%s] (%d variables in .env.vault)\n",
		titleStyle.Render("🔐 Project Environment Vault"),
		envBadgeStyle.Render(envName),
		len(summaries),
	)
	fmt.Println(dimStyle.Render(fmt.Sprintf("Configured environments: %s", strings.Join(allEnvs, ", "))))
	fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────────"))

	if len(summaries) == 0 {
		fmt.Printf("No variables found in environment '%s'. Add one with: sx env set --env %s KEY=VALUE\n\n", envName, envName)
		return nil
	}

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
	fmt.Printf("Inject into any process with: sx env run --env %s -- <command>\n\n", envName)
	return nil
}
