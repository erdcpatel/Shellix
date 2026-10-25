package cli

import (
	"fmt"
	"strings"

	"shellix/internal/git"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewBranchCmd constructs the interactive git branch manager
func NewBranchCmd(app *AppContext) *cobra.Command {
	branchCmd := &cobra.Command{
		Use:     "branch",
		Aliases: []string{"b"},
		Short:   "Interactive Git branch switcher and manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return git.RunBranchSwitcher(".")
		},
	}

	branchListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all local and remote branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			branches, err := git.ListBranches(".")
			if err != nil {
				return err
			}
			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
			currentStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

			fmt.Printf("%s (%d branches)\n", titleStyle.Render("🌿 Branches"), len(branches))
			fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────────"))
			for _, b := range branches {
				prefix := "  "
				name := b.Name
				if b.IsCurrent {
					prefix = currentStyle.Render("* ")
					name = currentStyle.Render(b.Name)
				}
				timeStr := dimStyle.Render(b.CommitTime)
				fmt.Printf("%s%-30s  %s  %s\n", prefix, name, dimStyle.Render(b.Subject), timeStr)
			}
			return nil
		},
	}

	branchCheckoutCmd := &cobra.Command{
		Use:   "checkout <branch>",
		Short: "Switch to specified branch",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			branches, err := git.ListBranches(".")
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var list []string
			for _, b := range branches {
				name := b.Name
				if strings.HasPrefix(name, "origin/") {
					name = strings.TrimPrefix(name, "origin/")
				}
				list = append(list, name)
			}
			return list, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return git.CheckoutBranch(".", args[0])
		},
	}

	branchCmd.AddCommand(branchListCmd, branchCheckoutCmd)
	return branchCmd
}

// NewStashCmd constructs the git stash manager
func NewStashCmd(app *AppContext) *cobra.Command {
	stashCmd := &cobra.Command{
		Use:   "stash",
		Short: "Git stash inspection and management",
		RunE: func(cmd *cobra.Command, args []string) error {
			stashes, err := git.ListStashes(".")
			if err != nil {
				return err
			}
			git.PrintStashesTable(stashes)
			return nil
		},
	}

	stashListCmd := &cobra.Command{
		Use:   "list",
		Short: "List saved stashes",
		RunE: func(cmd *cobra.Command, args []string) error {
			stashes, err := git.ListStashes(".")
			if err != nil {
				return err
			}
			git.PrintStashesTable(stashes)
			return nil
		},
	}

	var stashMsg string
	stashPushCmd := &cobra.Command{
		Use:   "push",
		Short: "Save current modified changes to a stash",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := git.SaveStash(".", stashMsg); err != nil {
				return err
			}
			fmt.Println("✨ Changes stashed successfully.")
			return nil
		},
	}
	stashPushCmd.Flags().StringVarP(&stashMsg, "message", "m", "", "Stash description message")

	var stashIndex int
	stashPopCmd := &cobra.Command{
		Use:   "pop [index]",
		Short: "Apply and drop saved stash",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				fmt.Sscanf(args[0], "%d", &stashIndex)
			}
			if err := git.PopStash(".", stashIndex); err != nil {
				return err
			}
			fmt.Printf("✨ Applied and dropped stash@{%d}\n", stashIndex)
			return nil
		},
	}

	stashCmd.AddCommand(stashListCmd, stashPushCmd, stashPopCmd)
	return stashCmd
}
