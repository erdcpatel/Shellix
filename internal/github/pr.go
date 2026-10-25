package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ListPullRequests queries open or closed PRs for the repository
func (c *Client) ListPullRequests(owner, repo, state string, limit int) ([]PullRequest, error) {
	if state == "" {
		state = "open"
	}
	if limit <= 0 {
		limit = 30
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/pulls?state=%s&per_page=%d", owner, repo, state, limit)
	resp, data, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var prs []PullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse pull requests: %w", err)
	}
	return prs, nil
}

// GetPullRequest fetches full details for a single PR
func (c *Client) GetPullRequest(owner, repo string, number int) (*PullRequest, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
	resp, data, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse pull request: %w", err)
	}
	return &pr, nil
}

type createPRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Draft bool   `json:"draft"`
}

// CreatePullRequest opens a new pull request on GitHub
func (c *Client) CreatePullRequest(owner, repo, title, body, head, base string, draft bool) (*PullRequest, error) {
	if base == "" {
		base = "main"
	}

	reqPayload := createPRRequest{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
		Draft: draft,
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/pulls", owner, repo)
	resp, data, err := c.DoRequest("POST", endpoint, reqPayload)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create pull request (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var pr PullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse created PR: %w", err)
	}
	return &pr, nil
}

// PrintPRList formats pull requests cleanly for the terminal
func PrintPRList(prs []PullRequest, owner, repo string) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	numStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	authorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68"))
	openStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	draftStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	fmt.Println(titleStyle.Render(fmt.Sprintf("Pull Requests for %s/%s (%d)", owner, repo, len(prs))))
	fmt.Println()

	if len(prs) == 0 {
		fmt.Println("No open pull requests found.")
		return
	}

	for _, pr := range prs {
		statusBadge := openStyle.Render("● Open")
		if pr.Draft {
			statusBadge = draftStyle.Render("○ Draft")
		}

		fmt.Printf("  %s  %-40s %s\n",
			numStyle.Render(fmt.Sprintf("#%-4d", pr.Number)),
			truncate(pr.Title, 40),
			statusBadge,
		)
		fmt.Printf("        %s → %s  by @%s  (%s)\n",
			branchStyle.Render(pr.Head.Ref),
			branchStyle.Render(pr.Base.Ref),
			authorStyle.Render(pr.User.Login),
			authorStyle.Render(pr.CreatedAt.Format("Jan 02")),
		)
	}
}

// PrintPRDetail renders full details of a PR
func PrintPRDetail(pr *PullRequest) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Width(14)
	urlStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Underline(true)

	fmt.Printf("\n#%d  %s\n\n", pr.Number, titleStyle.Render(pr.Title))
	fmt.Printf("  %s %s\n", labelStyle.Render("Status:"), strings.ToUpper(pr.State))
	fmt.Printf("  %s @%s\n", labelStyle.Render("Author:"), pr.User.Login)
	fmt.Printf("  %s %s → %s\n", labelStyle.Render("Branches:"), pr.Head.Ref, pr.Base.Ref)
	fmt.Printf("  %s %s\n", labelStyle.Render("URL:"), urlStyle.Render(pr.HTMLURL))

	if pr.Body != "" {
		fmt.Println("\n── Description ──────────────────────────────────────────")
		fmt.Println(strings.TrimSpace(pr.Body))
		fmt.Println("─────────────────────────────────────────────────────────")
	}
}

func truncate(text string, max int) string {
	if len(text) > max {
		return text[:max-3] + "..."
	}
	return text
}
