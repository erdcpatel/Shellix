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

const (
	DefaultGitHubHost = "github.com"
	DefaultAPIBase    = "https://api.github.com"
)

// Client interacts with GitHub and GitHub Enterprise REST APIs over HTTPS via Personal Access Token
type Client struct {
	Host       string
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient initializes a client with host auto-detection from the current repo or defaults to github.com
func NewClient() (*Client, error) {
	host := DefaultGitHubHost
	if slug, err := DetectRepoSlug("."); err == nil && slug.Host != "" {
		host = slug.Host
	}
	return NewClientForHost(host)
}

// NewClientForHost initializes a client targeting a specific GitHub or GitHub Enterprise host
func NewClientForHost(host string) (*Client, error) {
	if host == "" {
		host = DefaultGitHubHost
	}

	token := ResolveTokenForHost(host)
	if token == "" {
		hostMsg := ""
		if host != DefaultGitHubHost {
			hostMsg = fmt.Sprintf(" for host '%s'", host)
		}
		return nil, fmt.Errorf("no GitHub token found%s. Store your PAT securely with: sx github auth <token> --host %s (or set GITHUB_TOKEN)", hostMsg, host)
	}

	// Register token immediately with sanitizer so it is never printed in logs
	core.GlobalSanitizer.RegisterSecret(token)

	apiBase := ResolveAPIBase(host)

	// Clean HTTP transport with proxy awareness
	transport := http.DefaultTransport.(*http.Transport).Clone()

	return &Client{
		Host:    host,
		BaseURL: apiBase,
		Token:   token,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}, nil
}

// ResolveAPIBase determines the REST API endpoint URL for a given GitHub host
func ResolveAPIBase(host string) string {
	if apiURL := strings.TrimSpace(os.Getenv("GITHUB_API_URL")); apiURL != "" {
		return strings.TrimRight(apiURL, "/")
	}

	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" || host == DefaultGitHubHost {
		return DefaultAPIBase
	}

	// GitHub Enterprise Cloud tenant domains (*.ghe.com)
	if strings.HasSuffix(host, ".ghe.com") {
		return fmt.Sprintf("https://api.%s", host)
	}

	// Standard GitHub Enterprise Server (GHES on-premises / private cloud)
	return fmt.Sprintf("https://%s/api/v3", host)
}

// ResolveTokenForHost searches OS Vault (WinCred/Keychain) first, then environment variables for the specified host
func ResolveTokenForHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		host = DefaultGitHubHost
	}

	// 1. Try host-specific key in OS Keyring Vault
	v := vault.NewVault("")
	key := VaultKeyForHost(host)
	if tok, err := v.Get(key); err == nil && strings.TrimSpace(tok) != "" {
		return strings.TrimSpace(tok)
	}

	// 2. If Enterprise, check enterprise-specific environment variables
	if host != DefaultGitHubHost {
		if tok := os.Getenv("GITHUB_ENTERPRISE_TOKEN"); strings.TrimSpace(tok) != "" {
			return strings.TrimSpace(tok)
		}
		if tok := os.Getenv("GH_ENTERPRISE_TOKEN"); strings.TrimSpace(tok) != "" {
			return strings.TrimSpace(tok)
		}
	}

	// 3. Check standard GITHUB_TOKEN or GH_TOKEN
	if tok := os.Getenv("GITHUB_TOKEN"); strings.TrimSpace(tok) != "" {
		return strings.TrimSpace(tok)
	}
	if tok := os.Getenv("GH_TOKEN"); strings.TrimSpace(tok) != "" {
		return strings.TrimSpace(tok)
	}

	// 4. Fallback to default vault key if not already checked
	if host != DefaultGitHubHost {
		if tok, err := v.Get("github_token"); err == nil && strings.TrimSpace(tok) != "" {
			return strings.TrimSpace(tok)
		}
	}

	return ""
}

// ResolveToken returns token for default github.com host
func ResolveToken() string {
	return ResolveTokenForHost(DefaultGitHubHost)
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

// DetectRepoSlug reads the current directory's Git remote and extracts host, owner, and repository
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
	// Matches SCP syntax: git@<host>:<owner>/<repo>(.git)
	scpRegex = regexp.MustCompile(`^(?:[\w\-\.]+@)?([^/:]+):([^/]+)/([^/]+?)(?:\.git)?$`)
	// Matches SSH URL syntax: ssh://(user@)<host>(:port)/<owner>/<repo>(.git)
	sshURLRegex = regexp.MustCompile(`^ssh://(?:[^@]+@)?([^/:]+)(?::\d+)?/([^/]+)/([^/]+?)(?:\.git)?$`)
	// Matches HTTPS/HTTP URL syntax: http(s)://(user:pass@)<host>(:port)/<owner>/<repo>(.git)
	httpURLRegex = regexp.MustCompile(`^https?://(?:[^@]+@)?([^/:]+)(?::\d+)?/([^/]+)/([^/]+?)(?:\.git)?$`)
)

// ParseGitRemoteURL extracts host, owner and repository from SSH or HTTPS URLs (supporting public GitHub and Enterprise)
func ParseGitRemoteURL(url string) (*RepoSlug, error) {
	clean := strings.TrimSpace(url)

	if m := scpRegex.FindStringSubmatch(clean); len(m) == 4 {
		return &RepoSlug{Host: strings.ToLower(m[1]), Owner: m[2], Repo: m[3]}, nil
	}
	if m := sshURLRegex.FindStringSubmatch(clean); len(m) == 4 {
		return &RepoSlug{Host: strings.ToLower(m[1]), Owner: m[2], Repo: m[3]}, nil
	}
	if m := httpURLRegex.FindStringSubmatch(clean); len(m) == 4 {
		return &RepoSlug{Host: strings.ToLower(m[1]), Owner: m[2], Repo: m[3]}, nil
	}

	return nil, fmt.Errorf("unsupported remote URL format: %s", url)
}
