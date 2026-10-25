package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type workflowRunsResponse struct {
	TotalCount   int           `json:"total_count"`
	WorkflowRuns []WorkflowRun `json:"workflow_runs"`
}

// ListWorkflowRuns queries GitHub Actions runs for the specified repository
func (c *Client) ListWorkflowRuns(owner, repo, branch string, limit int) ([]WorkflowRun, error) {
	if limit <= 0 {
		limit = 10
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/actions/runs?per_page=%d", owner, repo, limit)
	if branch != "" {
		endpoint += fmt.Sprintf("&branch=%s", branch)
	}

	resp, data, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var res workflowRunsResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("failed to parse workflow runs: %w", err)
	}

	return res.WorkflowRuns, nil
}

// PrintWorkflowRuns formats CI checks cleanly
func PrintWorkflowRuns(runs []WorkflowRun, owner, repo, branch string) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4500"))
	progressStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA500"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	target := fmt.Sprintf("%s/%s", owner, repo)
	if branch != "" {
		target += fmt.Sprintf(" (branch: %s)", branch)
	}
	fmt.Println(titleStyle.Render(fmt.Sprintf("⚡ GitHub Actions CI Runs for %s", target)))
	fmt.Println()

	if len(runs) == 0 {
		fmt.Println("No recent workflow runs found.")
		return
	}

	for _, run := range runs {
		var statusBadge string
		switch run.Conclusion {
		case "success":
			statusBadge = successStyle.Render("✔ Success")
		case "failure":
			statusBadge = failStyle.Render("✖ Failed ")
		case "cancelled":
			statusBadge = metaStyle.Render("⊘ Cancel ")
		default:
			if run.Status == "in_progress" {
				statusBadge = progressStyle.Render("⏳ Running")
			} else {
				statusBadge = metaStyle.Render(fmt.Sprintf("• %s", run.Status))
			}
		}

		fmt.Printf("  %s  %-30s  branch: %-15s  (%s)\n",
			statusBadge,
			truncate(run.Name, 30),
			run.HeadBranch,
			metaStyle.Render(run.CreatedAt.Format("15:04:05")),
		)
	}
}
