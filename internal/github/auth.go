package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"shellix/internal/core"
	"shellix/internal/vault"

	"github.com/charmbracelet/lipgloss"
)

// GetCurrentUser verifies token authentication against https://api.github.com/user
func (c *Client) GetCurrentUser() (*GitHubUser, error) {
	resp, data, err := c.DoRequest("GET", "/user", nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("GitHub token is invalid or expired (HTTP 401)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var user GitHubUser
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}

	user.Scopes = resp.Header.Get("X-OAuth-Scopes")
	return &user, nil
}

// SavePAT securely stores the personal access token into the OS Vault
func SavePAT(token string) error {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return fmt.Errorf("token cannot be empty")
	}

	core.GlobalSanitizer.RegisterSecret(trimmed)

	v := vault.NewVault("")
	return v.Set("github_token", trimmed)
}

// PrintUserInfo renders authenticated user profile and token scopes
func PrintUserInfo(user *GitHubUser) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Width(16)
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	fmt.Println(titleStyle.Render("GitHub Authentication (HTTPS + PAT)"))
	fmt.Println()
	fmt.Printf("  %s %s (@%s)\n", labelStyle.Render("Logged in as:"), valStyle.Render(user.Name), user.Login)
	fmt.Printf("  %s %s\n", labelStyle.Render("Profile URL:"), user.HTMLURL)
	fmt.Printf("  %s %d\n", labelStyle.Render("Public Repos:"), user.PublicRepos)

	scopes := user.Scopes
	if scopes == "" {
		scopes = metaStyle.Render("(default / no explicit scopes)")
	}
	fmt.Printf("  %s %s\n", labelStyle.Render("Token Scopes:"), scopes)
}
