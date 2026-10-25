package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseGitRemoteURL(t *testing.T) {
	cases := []struct {
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			input:     "git@github.com:shellix-dev/shellix.git",
			wantOwner: "shellix-dev",
			wantRepo:  "shellix",
		},
		{
			input:     "https://github.com/torvalds/linux.git",
			wantOwner: "torvalds",
			wantRepo:  "linux",
		},
		{
			input:     "https://token@github.com/myorg/myrepo",
			wantOwner: "myorg",
			wantRepo:  "myrepo",
		},
		{
			input:   "https://gitlab.com/owner/repo.git",
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
		if slug.Owner != c.wantOwner || slug.Repo != c.wantRepo {
			t.Errorf("for %s: got %s/%s, want %s/%s", c.input, slug.Owner, slug.Repo, c.wantOwner, c.wantRepo)
		}
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
