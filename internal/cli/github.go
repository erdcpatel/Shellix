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
		Short:   "GitHub & GitHub Enterprise integration via HTTPS + Personal Access Token",
	}

	// 1. sx github auth [token] [--host <host>]
	var authHost string
	ghAuthCmd := &cobra.Command{
		Use:   "auth [token]",
		Short: "Store your GitHub or GitHub Enterprise PAT into the native OS Vault",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var token string
			if len(args) == 1 {
				token = args[0]
			} else {
				fmt.Printf("Enter GitHub Personal Access Token for '%s' (input hidden): ", authHost)
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

			if err := github.SavePATForHost(authHost, token); err != nil {
				return fmt.Errorf("failed to save token in vault: %w", err)
			}

			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			fmt.Printf("%s GitHub PAT for host '%s' securely stored in native OS Vault\n",
				successStyle.Render("✔ [AUTHENTICATED]"), authHost)
			fmt.Printf("You can now run: sx github whoami --host %s, sx pr list, sx ci status\n", authHost)
			return nil
		},
	}
	ghAuthCmd.Flags().StringVar(&authHost, "host", "github.com", "GitHub hostname (e.g. github.com, enterprise.internal, or tenant.ghe.com)")

	// 2. sx github whoami [--host <host>]
	var whoamiHost string
	ghWhoamiCmd := &cobra.Command{
		Use:   "whoami",
		Short: "Verify authentication and inspect GitHub user profile via HTTPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetHost := whoamiHost
			if targetHost == "" {
				if slug, err := github.DetectRepoSlug("."); err == nil && slug.Host != "" {
					targetHost = slug.Host
				} else {
					targetHost = "github.com"
				}
			}

			client, err := github.NewClientForHost(targetHost)
			if err != nil {
				return err
			}

			user, err := client.GetCurrentUser()
			if err != nil {
				return err
			}

			github.PrintUserInfo(user, targetHost)
			return nil
		},
	}
	ghWhoamiCmd.Flags().StringVar(&whoamiHost, "host", "", "GitHub hostname (default: auto-detected from repository remote)")

	// 3. sx github logout [--host <host>]
	var logoutHost string
	ghLogoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove GitHub Personal Access Token from the native OS Vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := github.DeletePATForHost(logoutHost); err != nil {
				return fmt.Errorf("failed to delete token: %w", err)
			}
			fmt.Printf("✨ Removed GitHub PAT for host '%s' from native OS Vault.\n", logoutHost)
			return nil
		},
	}
	ghLogoutCmd.Flags().StringVar(&logoutHost, "host", "github.com", "GitHub hostname")

	ghCmd.AddCommand(ghAuthCmd, ghWhoamiCmd, ghLogoutCmd)
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

// NewIssueCmd handles GitHub issues management
func NewIssueCmd(app *AppContext) *cobra.Command {
	var issueState string
	var issueLimit int

	issueCmd := &cobra.Command{
		Use:     "issue",
		Aliases: []string{"issues"},
		Short:   "GitHub Issues management via HTTPS + PAT",
	}

	// 1. sx issue list
	issueListCmd := &cobra.Command{
		Use:   "list",
		Short: "List repository issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, err := github.DetectRepoSlug(".")
			if err != nil {
				return err
			}
			client, err := github.NewClient()
			if err != nil {
				return err
			}
			issues, err := client.ListIssues(slug.Owner, slug.Repo, issueState, issueLimit)
			if err != nil {
				return err
			}
			github.PrintIssuesTable(issues)
			return nil
		},
	}
	issueListCmd.Flags().StringVarP(&issueState, "state", "s", "open", "Filter by state: open, closed, all")
	issueListCmd.Flags().IntVarP(&issueLimit, "limit", "l", 20, "Maximum issues to display")

	// 2. sx issue view <number>
	issueViewCmd := &cobra.Command{
		Use:   "view <number>",
		Short: "View details of a specific issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			num, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			slug, err := github.DetectRepoSlug(".")
			if err != nil {
				return err
			}
			client, err := github.NewClient()
			if err != nil {
				return err
			}
			it, err := client.GetIssue(slug.Owner, slug.Repo, num)
			if err != nil {
				return err
			}
			github.PrintIssueDetails(it)
			return nil
		},
	}

	// 3. sx issue create
	var itTitle, itBody string
	issueCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Open a new GitHub issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(itTitle) == "" {
				return fmt.Errorf("issue title is required (--title)")
			}
			slug, err := github.DetectRepoSlug(".")
			if err != nil {
				return err
			}
			client, err := github.NewClient()
			if err != nil {
				return err
			}
			created, err := client.CreateIssue(slug.Owner, slug.Repo, itTitle, itBody)
			if err != nil {
				return err
			}
			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			fmt.Printf("%s Created issue #%d: %s\n", successStyle.Render("✔"), created.Number, created.Title)
			fmt.Printf("  URL: %s\n", created.HTMLURL)
			return nil
		},
	}
	issueCreateCmd.Flags().StringVarP(&itTitle, "title", "t", "", "Issue title (required)")
	issueCreateCmd.Flags().StringVarP(&itBody, "body", "b", "", "Issue description / body")

	// 4. sx issue close <number>
	issueCloseCmd := &cobra.Command{
		Use:   "close <number>",
		Short: "Close a GitHub issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			num, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue number: %s", args[0])
			}
			slug, err := github.DetectRepoSlug(".")
			if err != nil {
				return err
			}
			client, err := github.NewClient()
			if err != nil {
				return err
			}
			if err := client.CloseIssue(slug.Owner, slug.Repo, num); err != nil {
				return err
			}
			successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
			fmt.Printf("%s Closed issue #%d in %s/%s\n", successStyle.Render("✔"), num, slug.Owner, slug.Repo)
			return nil
		},
	}

	issueCmd.AddCommand(issueListCmd, issueViewCmd, issueCreateCmd, issueCloseCmd)
	return issueCmd
}
