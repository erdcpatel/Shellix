package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"shellix/internal/palette"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewPaletteCmd constructs the universal command palette launcher
func NewPaletteCmd(app *AppContext) *cobra.Command {
	paletteCmd := &cobra.Command{
		Use:     "palette",
		Aliases: []string{"run", "p", "launcher"},
		Short:   "Universal fuzzy command palette & launcher",
		Long:    "Search and launch flows, java environments, SSH profiles, bookmarks, snippets, and history",
		RunE: func(cmd *cobra.Command, args []string) error {
			actions := palette.CollectAllActions(app.HistoryStore, app.ShortcutMgr, app.SSHMgr)
			m := palette.NewModel(actions)

			p := tea.NewProgram(m, tea.WithAltScreen())
			finalModel, err := p.Run()
			if err != nil {
				return err
			}

			fm, ok := finalModel.(palette.Model)
			if ok && fm.Selected != nil {
				sel := fm.Selected
				badgeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
				cmdStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))

				fmt.Printf("\n🚀 Launching %s: %s\n", badgeStyle.Render(string(sel.Category)), sel.Title)
				fmt.Printf("   Command: %s\n\n", cmdStyle.Render(sel.Command))

				// Execute selected command if it starts with sx or system binary
				parts := strings.Fields(sel.Command)
				if len(parts) > 0 {
					binName := parts[0]
					if binName == "sx" {
						binName = os.Args[0] // execute current shellix binary
					}
					c := exec.Command(binName, parts[1:]...)
					c.Stdin = os.Stdin
					c.Stdout = os.Stdout
					c.Stderr = os.Stderr
					return c.Run()
				}
			}
			return nil
		},
	}

	return paletteCmd
}
