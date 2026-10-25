package github

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGitHubTUIModel(t *testing.T) {
	slug := &RepoSlug{Host: "github.com", Owner: "shellix-dev", Repo: "shellix"}
	m := NewGitHubTUIModel(slug, nil)

	// Populate mock data
	m.prs = []PullRequest{
		{
			Number:    42,
			Title:     "feat: enterprise SSO integration",
			State:     "open",
			HTMLURL:   "https://github.com/shellix-dev/shellix/pull/42",
			Body:      "Adds SAML SSO and GitHub Enterprise support.",
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			Number:    43,
			Title:     "fix: token refresh in credential manager",
			State:     "open",
			HTMLURL:   "https://github.com/shellix-dev/shellix/pull/43",
			Body:      "Fixes token race on Windows Credential Manager.",
			CreatedAt: time.Now().Add(-5 * time.Hour),
		},
	}
	m.prs[0].Head.Ref = "feature/sso"
	m.prs[0].Base.Ref = "main"
	m.prs[0].User.Login = "dhavalpatel"

	m.issues = []Issue{
		{
			Number:    10,
			Title:     "Support custom CA certificates for GHES",
			State:     "open",
			HTMLURL:   "https://github.com/shellix-dev/shellix/issues/10",
			Body:      "Enterprise users need custom root certificates.",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			Comments:  3,
		},
	}
	m.issues[0].User.Login = "contributor"

	m.runs = []WorkflowRun{
		{
			ID:         1234567,
			Name:       "CI Test & Build",
			HeadBranch: "main",
			Status:     "completed",
			Conclusion: "success",
			Event:      "push",
			HTMLURL:    "https://github.com/shellix-dev/shellix/actions/runs/1234567",
			CreatedAt:  time.Now().Add(-10 * time.Minute),
		},
	}

	// 1. Resize window
	resModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 26})
	m = resModel.(GitHubTUIModel)
	if m.width != 100 || m.height != 26 {
		t.Errorf("expected 100x26, got %dx%d", m.width, m.height)
	}

	// 2. Initial view contains PR #42
	view := m.View()
	if !strings.Contains(view, "#42") || !strings.Contains(view, "enterprise SSO") {
		t.Errorf("expected PR #42 in view, got: %s", view)
	}

	// 3. Move cursor down
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = resModel.(GitHubTUIModel)
	if m.prCursor != 1 {
		t.Errorf("expected prCursor 1, got %d", m.prCursor)
	}

	// 4. Switch to Issues tab (press '2')
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = resModel.(GitHubTUIModel)
	if m.activeTab != TabIssues {
		t.Errorf("expected TabIssues, got %v", m.activeTab)
	}
	view = m.View()
	if !strings.Contains(view, "#10") || !strings.Contains(view, "custom CA") {
		t.Errorf("expected Issue #10 in view, got: %s", view)
	}

	// 5. Switch to CI tab (press '3')
	resModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = resModel.(GitHubTUIModel)
	if m.activeTab != TabCI {
		t.Errorf("expected TabCI, got %v", m.activeTab)
	}
	view = m.View()
	if !strings.Contains(view, "CI Test & Build") {
		t.Errorf("expected CI run in view, got: %s", view)
	}
}

func TestRenderSnapshot(t *testing.T) {
	slug := &RepoSlug{Host: "github.internal.corp", Owner: "engineering", Repo: "platform-service"}
	prs := []PullRequest{
		{
			Number:    88,
			Title:     "feat: zero-cgo database driver",
			State:     "open",
			HTMLURL:   "https://github.internal.corp/engineering/platform-service/pull/88",
			Body:      "Replaces CGO sqlite with pure Go driver for sub-millisecond connection setup.",
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			Number:    89,
			Title:     "refactor: improve DAG topological sort",
			State:     "open",
			HTMLURL:   "https://github.internal.corp/engineering/platform-service/pull/89",
			Body:      "Uses Kahn's algorithm with min-heap priority queue.",
			CreatedAt: time.Now().Add(-3 * time.Hour),
		},
	}
	prs[0].Head.Ref = "feature/pure-go"
	prs[0].Base.Ref = "main"
	prs[0].User.Login = "dhavalpatel"

	issues := []Issue{
		{
			Number:    25,
			Title:     "Verify key derivation iterations in native vault",
			State:     "open",
			HTMLURL:   "https://github.internal.corp/engineering/platform-service/issues/25",
			Body:      "Ensure PBKDF2 iteration count is at least 600,000 for key derivation.",
			CreatedAt: time.Now().Add(-12 * time.Hour),
			Comments:  5,
		},
	}
	issues[0].User.Login = "sec-auditor"

	runs := []WorkflowRun{
		{
			ID:         9876543,
			Name:       "Full Verification Pipeline",
			HeadBranch: "main",
			Status:     "completed",
			Conclusion: "success",
			Event:      "push",
			HTMLURL:    "https://github.internal.corp/engineering/platform-service/actions/runs/9876543",
			CreatedAt:  time.Now().Add(-15 * time.Minute),
		},
	}

	snap := RenderSnapshot(slug, prs, issues, runs)
	if !strings.Contains(snap, "platform-service") {
		t.Errorf("expected repo name in snapshot")
	}
	if !strings.Contains(snap, "zero-cgo database driver") {
		t.Errorf("expected PR title in snapshot")
	}

	if os.Getenv("PRINT_SNAPSHOT") == "1" {
		fmt.Print(snap)
	}
}
