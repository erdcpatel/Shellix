package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCmd generates shell completion scripts for powershell, zsh, bash, and fish
func NewCompletionCmd() *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [powershell|zsh|bash|fish]",
		Short: "Generate shell auto-completion scripts (PowerShell, Zsh, Bash, Fish)",
		Long: `Generate shell auto-completion scripts for Shellix (sx).

PowerShell (Windows - Default & Recommended):
  # To load completions for the current session:
  sx completion powershell | Out-String | Invoke-Expression

  # To load completions for every session, add to your $PROFILE:
  if (Get-Command sx -ErrorAction SilentlyContinue) {
      sx completion powershell | Out-String | Invoke-Expression
  }

Zsh (macOS):
  # To load completions in your current session:
  source <(sx completion zsh)

  # To load completions permanently:
  sx completion zsh > "${fpath[1]}/_sx"

Bash:
  # To load completions in your current session:
  source <(sx completion bash)

  # To load completions permanently:
  sx completion bash > /etc/bash_completion.d/sx
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"powershell", "zsh", "bash", "fish"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			default:
				return fmt.Errorf("unsupported shell %q (supported: powershell, zsh, bash, fish)", args[0])
			}
		},
	}

	return completionCmd
}
