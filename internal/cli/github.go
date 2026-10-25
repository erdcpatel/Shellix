package cli

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"

	"shellix/internal/github"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// NewGitHubCmd handles GitHub API operations strictly via HTTPS + PAT
func NewGitHubCmd(app *AppContext) *cobra.Command {
	ghCmd := &cobra.Command{
		Use:     "github",
		Aliases: []string{"gh"},
		Short:   "GitHub integration via HTTPS + Personal Access Token",
	}

	// 1. sx github auth [token]
	ghAuthCmd := &cobra.Command{
		Use:   "auth [token]",
		Short: "Store your GitHub Personal Access Token into the native OS Vault",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var token string
			if len(args) == 1 {
				token = args[0]
			} else {
				fmt.Print("Enter your GitHub Personal Access Token (input hidden): ")
				byteToken, err := term.ReadPassword(int(syscall.Stdin))
				fmt.Println()
				if err != nil {
					return fmt.Errorf("failed to read token: %w", err)
				}
				token = strings.TrimSpace(string(byteToken))
			}

			if token == "" {
				return fmt.Errorf("token cannot be empty")
			}

			if err := github.SavePAT(token); err != nil {
				return fmt.Errorf("failed to save token in vault: %w", err)
			}

			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			fmt.Printf("%s GitHub PAT securely stored in %s\n",
				successStyle.Render("✔ [AUTHENTICATED]"), app.Config.Shell.Default)
			fmt.Println("You can now run: sx github whoami, sx pr list, sx ci status")
			return nil
		},
	}

	// 2. sx github whoami
	ghWhoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "Verify authentication and inspect GitHub user profile via HTTPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := github.NewClient()
			if err != nil {
				return err
			}

			user, err := client.GetCurrentUser()
			if err != nil {
				return err
			}

			github.PrintUserInfo(user)
			return nil
		},
	}

	ghCmd.AddCommand(ghAuthCmd, ghWhoamiCmd)
	return ghCmd
}

// NewPRCmd handles Pull Request operations
func NewPRCmd(app *AppContext) *cobra.Command {
	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "GitHub Pull Requests management",
	}

	// 1. sx pr list / ls
	var prState string
	var prLimit int
	prListCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pull requests for current repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, err := github.DetectRepoSlug(".")
			if err != nil {
				return err
			}

			client, err := github.NewClient()
			if err != nil {
				return err
			}

			prs, err := client.ListPullRequests(slug.Owner, slug.Repo, prState, prLimit)
			if err != nil {
				return err
			}

			github.PrintPRList(prs, slug.Owner, slug.Repo)
			return nil
		},
	}
	prListCmd.Flags().StringVarP(&prState, "state", "s", "open", "Filter by state: open, closed, all")
	prListCmd.Flags().IntVarP(&prLimit, "limit", "l", 20, "Maximum pull requests to list")

	// 2. sx pr view <number>
	prViewCmd := &cobra.Command{
		Use:   "view <number>",
		Short: "View pull request details and description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			num, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}

			slug, err := github.DetectRepoSlug(".")
			if err != nil {
				return err
			}

			client, err := github.NewClient()
			if err != nil {
				return err
			}

			pr, err := client.GetPullRequest(slug.Owner, slug.Repo, num)
			if err != nil {
				return err
			}

			github.PrintPRDetail(pr)
			return nil
		},
	}

	// 3. sx pr create
	var prTitle, prBody, prHead, prBase string
	var prDraft bool
	prCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Open a new pull request on GitHub via HTTPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, err := github.DetectRepoSlug(".")
			if err != nil {
				return err
			}

			// Auto-detect current branch for head if omitted
			if prHead == "" {
				st, err := app.GitEngine.GetStatus(".")
				if err != nil || st.Branch == "" {
					return fmt.Errorf("unable to detect current branch; please specify --head <branch>")
				}
				prHead = st.Branch
			}

			if prTitle == "" {
				return fmt.Errorf("please provide a PR title via --title <string>")
			}

			client, err := github.NewClient()
			if err != nil {
				return err
			}

			createdPR, err := client.CreatePullRequest(slug.Owner, slug.Repo, prTitle, prBody, prHead, prBase, prDraft)
			if err != nil {
				return err
			}

			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Underline(true)

			fmt.Printf("%s Created Pull Request #%d: %s\n",
				successStyle.Render("✔ [PR CREATED]"), createdPR.Number, createdPR.Title)
			fmt.Printf("  URL: %s\n", urlStyle.Render(createdPR.HTMLURL))
			return nil
		},
	}
	prCreateCmd.Flags().StringVarP(&prTitle, "title", "t", "", "Pull request title (required)")
	prCreateCmd.Flags().StringVarP(&prBody, "body", "b", "", "Pull request description / body")
	prCreateCmd.Flags().StringVar(&prHead, "head", "", "Branch containing changes (defaults to current branch)")
	prCreateCmd.Flags().StringVar(&prBase, "base", "main", "Target branch into which changes should be merged")
	prCreateCmd.Flags().BoolVarP(&prDraft, "draft", "d", false, "Create as draft pull request")

	prCmd.AddCommand(prListCmd, prViewCmd, prCreateCmd)
	return prCmd
}

// NewCICmd handles GitHub Actions status
func NewCICmd(app *AppContext) *cobra.Command {
	var ciBranch string
	var ciLimit int

	ciCmd := &cobra.Command{
		Use:     "ci",
		Aliases: []string{"actions"},
		Short:   "GitHub Actions CI workflow checks via HTTPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, err := github.DetectRepoSlug(".")
			if err != nil {
				return err
			}

			if ciBranch == "" {
				if st, err := app.GitEngine.GetStatus("."); err == nil && st.Branch != "" {
					ciBranch = st.Branch
				}
			}

			client, err := github.NewClient()
			if err != nil {
				return err
			}

			runs, err := client.ListWorkflowRuns(slug.Owner, slug.Repo, ciBranch, ciLimit)
			if err != nil {
				return err
			}

			github.PrintWorkflowRuns(runs, slug.Owner, slug.Repo, ciBranch)
			return nil
		},
	}
	ciCmd.Flags().StringVarP(&ciBranch, "branch", "b", "", "Filter workflow runs by branch (defaults to current branch)")
	ciCmd.Flags().IntVarP(&ciLimit, "limit", "l", 10, "Maximum runs to display")

	return ciCmd
}
