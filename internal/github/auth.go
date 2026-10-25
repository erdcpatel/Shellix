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

// VaultKeyForHost returns the storage key for a given GitHub host in the OS Vault
func VaultKeyForHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" || host == "github.com" {
		return "github_token"
	}
	return fmt.Sprintf("github_token_%s", host)
}

// SavePAT securely stores the personal access token for default host into the OS Vault
func SavePAT(token string) error {
	return SavePATForHost("github.com", token)
}

// SavePATForHost securely stores the personal access token for a specific host (e.g. GitHub Enterprise)
func SavePATForHost(host, token string) error {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return fmt.Errorf("token cannot be empty")
	}

	core.GlobalSanitizer.RegisterSecret(trimmed)

	v := vault.NewVault("")
	return v.Set(VaultKeyForHost(host), trimmed)
}

// DeletePAT removes the personal access token for default host from the OS Vault
func DeletePAT() error {
	return DeletePATForHost("github.com")
}

// DeletePATForHost removes the personal access token for a specific host from the OS Vault
func DeletePATForHost(host string) error {
	v := vault.NewVault("")
	return v.Delete(VaultKeyForHost(host))
}

// PrintUserInfo renders authenticated user profile and token scopes
func PrintUserInfo(user *GitHubUser, host ...string) {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Width(16)
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))

	targetHost := "github.com"
	if len(host) > 0 && host[0] != "" {
		targetHost = host[0]
	}

	header := fmt.Sprintf("GitHub Authentication [%s] (HTTPS + PAT)", targetHost)
	fmt.Println(titleStyle.Render(header))
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
