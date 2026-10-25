package github

import "time"

// GitHubUser represents authenticated GitHub profile metadata
type GitHubUser struct {
	Login       string `json:"login"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PublicRepos int    `json:"public_repos"`
	HTMLURL     string `json:"html_url"`
	Scopes      string `json:"-"`
}

// PullRequest represents GitHub PR metadata
type PullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Draft     bool      `json:"draft"`
	HTMLURL   string    `json:"html_url"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Head struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

// Issue represents GitHub issue metadata
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"` // open, closed
	HTMLURL   string    `json:"html_url"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"labels"`
	Comments int `json:"comments"`
}

// WorkflowRun represents a GitHub Actions CI execution
type WorkflowRun struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	HeadBranch string    `json:"head_branch"`
	Status     string    `json:"status"`     // queued, in_progress, completed
	Conclusion string    `json:"conclusion"` // success, failure, cancelled, timed_out
	Event      string    `json:"event"`
	HTMLURL    string    `json:"html_url"`
	CreatedAt  time.Time `json:"created_at"`
}

// RepoSlug represents owner and repository identifiers
type RepoSlug struct {
	Owner string
	Repo  string
}
