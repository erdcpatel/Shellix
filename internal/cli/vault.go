package cli

import (
	"fmt"
	"strings"
	"syscall"

	"shellix/internal/vault"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewVaultCmd provides secret and credential management commands
func NewVaultCmd(app *AppContext) *cobra.Command {
	v := vault.NewVault("")

	vaultCmd := &cobra.Command{
		Use:     "vault",
		Aliases: []string{"cred"},
		Short:   "Secure OS credential manager (Windows Credential Manager / macOS Keychain)",
	}

	// 1. sx vault info
	vaultInfoCmd := &cobra.Command{
		Use:   "info",
		Short: "Display active OS credential provider and security status",
		Run: func(cmd *cobra.Command, args []string) {
			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
			keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Width(20)
			valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))

			fmt.Println(titleStyle.Render("🔐 Shellix Credential Manager"))
			fmt.Println()
			fmt.Printf("  %s %s\n", keyStyle.Render("Active Provider:"), valStyle.Render(v.ProviderName()))
			fmt.Printf("  %s %s\n", keyStyle.Render("Service Scope:"), vault.ServiceName)
			fmt.Printf("  %s %d stored\n", keyStyle.Render("Stored Keys:"), len(v.ListKeys()))
		},
	}

	// 2. sx vault set <key> [value]
	vaultSetCmd := &cobra.Command{
		Use:   "set <key> [value]",
		Short: "Store a secret in the OS credential manager",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			var secret string

			if len(args) == 2 {
				secret = args[1]
			} else {
				fmt.Printf("Enter secret for '%s' (input will be hidden): ", key)
				bytePassword, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("failed to read secret: %w", err)
				}
				secret = strings.TrimSpace(string(bytePassword))
			}

			if secret == "" {
				return fmt.Errorf("secret cannot be empty")
			}

			if err := v.Set(key, secret); err != nil {
				return fmt.Errorf("failed to store secret: %w", err)
			}

			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			fmt.Printf("%s Saved credential '%s' into %s\n",
				successStyle.Render("✔ [SAVED]"), key, v.ProviderName())
			return nil
		},
	}

	// 3. sx vault get <key>
	vaultGetCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Retrieve a secret from the OS credential manager",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			val, err := v.Get(key)
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		},
	}

	// 4. sx vault list
	vaultListCmd := &cobra.Command{
		Use:   "list",
		Short: "List credential keys stored in the vault",
		Run: func(cmd *cobra.Command, args []string) {
			keys := v.ListKeys()
			if len(keys) == 0 {
				fmt.Println("No credentials currently stored. Use 'sx vault set <key>' to store one.")
				return
			}
			fmt.Printf("Stored Credentials (%s):\n", v.ProviderName())
			for _, k := range keys {
				fmt.Printf("  • %s\n", k)
			}
		},
	}

	// 5. sx vault rm <key>
	vaultRmCmd := &cobra.Command{
		Use:   "rm <key>",
		Short: "Delete a secret from the credential manager",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if err := v.Delete(key); err != nil {
				return err
			}
			fmt.Printf("✔ Removed credential '%s'\n", key)
			return nil
		},
	}

	vaultCmd.AddCommand(vaultInfoCmd, vaultSetCmd, vaultGetCmd, vaultListCmd, vaultRmCmd)
	return vaultCmd
}
