package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"shellix/internal/core"
	"shellix/internal/vault"
)

const DefaultAPIBase = "https://api.github.com"

// Client interacts strictly with GitHub REST APIs over HTTPS via Personal Access Token
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient initializes a client with token discovery from the native OS Vault or environment
func NewClient() (*Client, error) {
	token := ResolveToken()
	if token == "" {
		return nil, fmt.Errorf("no GitHub token found. Store your PAT securely with: sx github auth <token> (or set GITHUB_TOKEN)")
	}

	// Register token immediately with sanitizer so it is never printed in logs
	core.GlobalSanitizer.RegisterSecret(token)

	return &Client{
		BaseURL: DefaultAPIBase,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

// ResolveToken searches OS Vault (WinCred/Keychain) first, then environment variables
func ResolveToken() string {
	// 1. Try OS Keyring Vault
	v := vault.NewVault("")
	if tok, err := v.Get("github_token"); err == nil && strings.TrimSpace(tok) != "" {
		return strings.TrimSpace(tok)
	}

	// 2. Try GITHUB_TOKEN
	if tok := os.Getenv("GITHUB_TOKEN"); strings.TrimSpace(tok) != "" {
		return strings.TrimSpace(tok)
	}

	// 3. Try GH_TOKEN
	if tok := os.Getenv("GH_TOKEN"); strings.TrimSpace(tok) != "" {
		return strings.TrimSpace(tok)
	}

	return ""
}

// DoRequest performs an authenticated HTTPS request
func (c *Client) DoRequest(method, endpoint string, body interface{}) (*http.Response, []byte, error) {
	url := c.BaseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Shellix-Developer-Workbench")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTPS request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return resp, respBytes, nil
}

// DetectRepoSlug reads the current directory's Git remote and extracts owner/repo
func DetectRepoSlug(dir string) (*RepoSlug, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to detect git remote origin: %s", strings.TrimSpace(string(out)))
	}

	rawURL := strings.TrimSpace(string(out))
	return ParseGitRemoteURL(rawURL)
}

var (
	// git@github.com:owner/repo.git
	sshRegex = regexp.MustCompile(`^git@github\.com:([^/]+)/([^/]+?)(?:\.git)?$`)
	// https://github.com/owner/repo.git or https://token@github.com/owner/repo.git
	httpsRegex = regexp.MustCompile(`^https://(?:[^@]+@)?github\.com/([^/]+)/([^/]+?)(?:\.git)?$`)
)

// ParseGitRemoteURL extracts owner and repository from SSH or HTTPS GitHub URLs
func ParseGitRemoteURL(url string) (*RepoSlug, error) {
	clean := strings.TrimSpace(url)

	if m := sshRegex.FindStringSubmatch(clean); len(m) == 3 {
		return &RepoSlug{Owner: m[1], Repo: m[2]}, nil
	}
	if m := httpsRegex.FindStringSubmatch(clean); len(m) == 3 {
		return &RepoSlug{Owner: m[1], Repo: m[2]}, nil
	}

	return nil, fmt.Errorf("unsupported remote URL format (expected github.com): %s", url)
}
