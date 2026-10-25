package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParseGitRemoteURL(t *testing.T) {
	cases := []struct {
		input     string
		wantHost  string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			input:     "git@github.com:shellix-dev/shellix.git",
			wantHost:  "github.com",
			wantOwner: "shellix-dev",
			wantRepo:  "shellix",
		},
		{
			input:     "https://github.com/torvalds/linux.git",
			wantHost:  "github.com",
			wantOwner: "torvalds",
			wantRepo:  "linux",
		},
		{
			input:     "https://token@github.com/myorg/myrepo",
			wantHost:  "github.com",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		// GitHub Enterprise Server (GHES on-premises)
		{
			input:     "git@github.internal.corp:engineering/core-platform.git",
			wantHost:  "github.internal.corp",
			wantOwner: "engineering",
			wantRepo:  "core-platform",
		},
		{
			input:     "https://github.corp.megacorp.com/devops/deployer.git",
			wantHost:  "github.corp.megacorp.com",
			wantOwner: "devops",
			wantRepo:  "deployer",
		},
		// GitHub Enterprise Cloud (GHEC tenant)
		{
			input:     "git@acme.ghe.com:payments/service.git",
			wantHost:  "acme.ghe.com",
			wantOwner: "payments",
			wantRepo:  "service",
		},
		{
			input:     "https://oauth2:token@acme.ghe.com:8443/payments/service.git",
			wantHost:  "acme.ghe.com",
			wantOwner: "payments",
			wantRepo:  "service",
		},
		// SSH URL protocol format
		{
			input:     "ssh://git@github.enterprise.internal/team/app.git",
			wantHost:  "github.enterprise.internal",
			wantOwner: "team",
			wantRepo:  "app",
		},
		{
			input:   "not-a-valid-url",
			wantErr: true,
		},
	}

	for _, c := range cases {
		slug, err := ParseGitRemoteURL(c.input)
		if c.wantErr {
			if err == nil {
				t.Errorf("expected error for %s, got nil", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error for %s: %v", c.input, err)
			continue
		}
		if slug.Host != c.wantHost || slug.Owner != c.wantOwner || slug.Repo != c.wantRepo {
			t.Errorf("for %s: got host=%s, %s/%s; want host=%s, %s/%s",
				c.input, slug.Host, slug.Owner, slug.Repo, c.wantHost, c.wantOwner, c.wantRepo)
		}
	}
}

func TestResolveAPIBase(t *testing.T) {
	// Standard github.com
	if base := ResolveAPIBase("github.com"); base != "https://api.github.com" {
		t.Errorf("expected https://api.github.com, got %s", base)
	}
	if base := ResolveAPIBase(""); base != "https://api.github.com" {
		t.Errorf("expected https://api.github.com for empty host, got %s", base)
	}

	// GitHub Enterprise Cloud (*.ghe.com)
	if base := ResolveAPIBase("acme.ghe.com"); base != "https://api.acme.ghe.com" {
		t.Errorf("expected https://api.acme.ghe.com, got %s", base)
	}

	// GitHub Enterprise Server (GHES on-premises)
	if base := ResolveAPIBase("github.internal.corp"); base != "https://github.internal.corp/api/v3" {
		t.Errorf("expected https://github.internal.corp/api/v3, got %s", base)
	}

	// GITHUB_API_URL override
	os.Setenv("GITHUB_API_URL", "https://custom.api.endpoint")
	defer os.Unsetenv("GITHUB_API_URL")
	if base := ResolveAPIBase("github.com"); base != "https://custom.api.endpoint" {
		t.Errorf("expected override https://custom.api.endpoint, got %s", base)
	}
}

func TestEnterpriseTokenResolution(t *testing.T) {
	host := "github.enterprise.corp"

	// Test environment variable resolution
	os.Setenv("GITHUB_ENTERPRISE_TOKEN", "ent-test-token-12345")
	defer os.Unsetenv("GITHUB_ENTERPRISE_TOKEN")

	tok := ResolveTokenForHost(host)
	if tok != "ent-test-token-12345" {
		t.Errorf("expected ent-test-token-12345, got %s", tok)
	}
}

func TestMockGitHubClient(t *testing.T) {
	// Mock GitHub server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify strict HTTPS headers
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-pat-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path == "/user" {
			w.Header().Set("X-OAuth-Scopes", "repo, workflow")
			user := GitHubUser{
				Login:       "dhavalpatel",
				Name:        "Dhaval Patel",
				PublicRepos: 42,
			}
			_ = json.NewEncoder(w).Encode(user)
			return
		}

		if r.URL.Path == "/repos/shellix-dev/shellix/pulls" {
			prs := []PullRequest{
				{
					Number: 101,
					Title:  "feat: multi-jdk manager",
					State:  "open",
				},
			}
			_ = json.NewEncoder(w).Encode(prs)
			return
		}

		if r.URL.Path == "/repos/shellix-dev/shellix/issues" {
			if r.Method == "POST" {
				var req createIssueRequest
				_ = json.NewDecoder(r.Body).Decode(&req)
				created := Issue{
					Number: 42,
					Title:  req.Title,
					Body:   req.Body,
					State:  "open",
				}
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(created)
				return
			}
			issues := []Issue{
				{
					Number: 42,
					Title:  "Security: verify SSH host keys",
					State:  "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client := &Client{
		Host:       "github.com",
		BaseURL:    server.URL,
		Token:      "test-pat-token",
		HTTPClient: server.Client(),
	}

	// 1. Test GetCurrentUser
	user, err := client.GetCurrentUser()
	if err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}
	if user.Login != "dhavalpatel" {
		t.Errorf("expected login dhavalpatel, got %s", user.Login)
	}
	if user.Scopes != "repo, workflow" {
		t.Errorf("expected scopes repo, workflow, got %s", user.Scopes)
	}

	// 2. Test ListPullRequests
	prs, err := client.ListPullRequests("shellix-dev", "shellix", "open", 10)
	if err != nil {
		t.Fatalf("ListPullRequests failed: %v", err)
	}
	if len(prs) != 1 || prs[0].Number != 101 {
		t.Errorf("expected PR #101, got %+v", prs)
	}

	// 3. Test ListIssues
	issues, err := client.ListIssues("shellix-dev", "shellix", "open", 10)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(issues) != 1 || issues[0].Number != 42 {
		t.Errorf("expected issue #42, got %+v", issues)
	}

	// 4. Test CreateIssue
	created, err := client.CreateIssue("shellix-dev", "shellix", "New issue", "Description")
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if created.Number != 42 || created.Title != "New issue" {
		t.Errorf("expected created issue #42, got %+v", created)
	}
}
