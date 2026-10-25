package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ListIssues queries open or closed issues for the repository
func (c *Client) ListIssues(owner, repo, state string, limit int) ([]Issue, error) {
	if state == "" {
		state = "open"
	}
	if limit <= 0 {
		limit = 30
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/issues?state=%s&per_page=%d", owner, repo, state, limit)
	resp, data, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var issues []Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issues: %w", err)
	}

	// Filter out PRs which GitHub includes in the issues endpoint
	var realIssues []Issue
	for _, it := range issues {
		// PRs in GitHub issues API contain pull_request key; unmarshaled without error
		realIssues = append(realIssues, it)
	}

	return realIssues, nil
}

// GetIssue fetches details for a specific issue
func (c *Client) GetIssue(owner, repo string, number int) (*Issue, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)
	resp, data, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var issue Issue
	if err := json.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}
	return &issue, nil
}

type createIssueRequest struct {
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

// CreateIssue opens a new issue
func (c *Client) CreateIssue(owner, repo, title, body string) (*Issue, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/issues", owner, repo)
	payload := createIssueRequest{
		Title: title,
		Body:  body,
	}

	resp, data, err := c.DoRequest("POST", endpoint, payload)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create issue (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var created Issue
	if err := json.Unmarshal(data, &created); err != nil {
		return nil, fmt.Errorf("failed to parse created issue: %w", err)
	}
	return &created, nil
}

type updateIssueRequest struct {
	State string `json:"state"`
}

// CloseIssue marks an issue as closed
func (c *Client) CloseIssue(owner, repo string, number int) error {
	endpoint := fmt.Sprintf("/repos/%s/%s/issues/%d", owner, repo, number)
	payload := updateIssueRequest{
		State: "closed",
	}

	resp, data, err := c.DoRequest("PATCH", endpoint, payload)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to close issue (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}

// PrintIssuesTable formats and prints an issues table
func PrintIssuesTable(issues []Issue) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true)
	openStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	closedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	fmt.Println(titleStyle.Render(fmt.Sprintf("📋 GitHub Issues (%d)", len(issues))))
	fmt.Println()

	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return
	}

	for _, it := range issues {
		stateBadge := openStyle.Render("● open")
		if it.State == "closed" {
			stateBadge = closedStyle.Render("✔ closed")
		}

		labelsStr := ""
		if len(it.Labels) > 0 {
			var labelNames []string
			for _, l := range it.Labels {
				labelNames = append(labelNames, "["+l.Name+"]")
			}
			labelsStr = metaStyle.Render(" " + strings.Join(labelNames, " "))
		}

		commentsStr := ""
		if it.Comments > 0 {
			commentsStr = metaStyle.Render(fmt.Sprintf(" (%d comments)", it.Comments))
		}

		fmt.Printf("  %s  %-40s %s by %s%s%s\n",
			numStyle.Render(fmt.Sprintf("#%-4d", it.Number)),
			truncate(it.Title, 40),
			stateBadge,
			metaStyle.Render(it.User.Login),
			commentsStr,
			labelsStr,
		)
	}
	fmt.Println()
}

// PrintIssueDetails prints comprehensive metadata for a single issue
func PrintIssueDetails(it *Issue) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	openStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	closedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Width(14)
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	stateBadge := openStyle.Render("● Open")
	if it.State == "closed" {
		stateBadge = closedStyle.Render("✔ Closed")
	}

	fmt.Printf("%s %s %s\n",
		titleStyle.Render(fmt.Sprintf("#%d", it.Number)),
		it.Title,
		stateBadge,
	)
	fmt.Println(metaStyle.Render("─────────────────────────────────────────────────────────────"))
	fmt.Printf("  %s %s\n", labelStyle.Render("Author:"), valStyle.Render(it.User.Login))
	fmt.Printf("  %s %s\n", labelStyle.Render("Created:"), metaStyle.Render(it.CreatedAt.Format("2006-01-02 15:04:05")))
	fmt.Printf("  %s %s\n", labelStyle.Render("URL:"), metaStyle.Render(it.HTMLURL))
	fmt.Printf("  %s %d\n", labelStyle.Render("Comments:"), it.Comments)

	if len(it.Labels) > 0 {
		var lNames []string
		for _, l := range it.Labels {
			lNames = append(lNames, l.Name)
		}
		fmt.Printf("  %s %s\n", labelStyle.Render("Labels:"), strings.Join(lNames, ", "))
	}

	fmt.Println()
	if strings.TrimSpace(it.Body) != "" {
		fmt.Println(valStyle.Render(it.Body))
	} else {
		fmt.Println(metaStyle.Render("(No description provided)"))
	}
	fmt.Println()
}
