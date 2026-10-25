package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// RepositoryStatus represents instant in-memory Git repo metadata
type RepositoryStatus struct {
	Path        string
	Branch      string
	HeadCommit  string
	HeadMessage string
	HeadAuthor  string
	HeadTime    time.Time
	Staged      []string
	Unstaged    []string
	Untracked   []string
	IsClean     bool
}

// Engine encapsulates the dual-engine Git executor
type Engine struct {
	UseInProcess bool
}

// NewEngine creates a new Git engine instance
func NewEngine(useInProcess bool) *Engine {
	return &Engine{
		UseInProcess: useInProcess,
	}
}

// GetStatus uses in-process go-git to query repository status with zero fork overhead
func (e *Engine) GetStatus(repoPath string) (*RepositoryStatus, error) {
	if repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		repoPath = cwd
	}

	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, fmt.Errorf("not a git repository (or any parent directory)")
		}
		return nil, err
	}

	status := &RepositoryStatus{
		Path: repoPath,
	}

	// 1. Current Branch
	headRef, err := repo.Head()
	if err == nil {
		if headRef.Name().IsBranch() {
			status.Branch = headRef.Name().Short()
		} else {
			status.Branch = fmt.Sprintf("HEAD (%s)", headRef.Hash().String()[:7])
		}

		// 2. Head Commit
		commit, err := repo.CommitObject(headRef.Hash())
		if err == nil {
			status.HeadCommit = commit.Hash.String()[:7]
			status.HeadMessage = strings.Split(commit.Message, "\n")[0]
			status.HeadAuthor = commit.Author.Name
			status.HeadTime = commit.Author.When
		}
	} else if errors.Is(err, plumbing.ErrReferenceNotFound) {
		status.Branch = "main (empty repo)"
	}

	// 3. Worktree status
	wt, err := repo.Worktree()
	if err == nil {
		wtStatus, err := wt.Status()
		if err == nil {
			status.IsClean = wtStatus.IsClean()
			for file, fileStatus := range wtStatus {
				if fileStatus.Staging != git.Unmodified && fileStatus.Staging != git.Untracked {
					status.Staged = append(status.Staged, fmt.Sprintf("%c %s", fileStatus.Staging, file))
				}
				if fileStatus.Worktree != git.Unmodified && fileStatus.Worktree != git.Untracked {
					status.Unstaged = append(status.Unstaged, fmt.Sprintf("%c %s", fileStatus.Worktree, file))
				}
				if fileStatus.Worktree == git.Untracked || fileStatus.Staging == git.Untracked {
					status.Untracked = append(status.Untracked, file)
				}
			}
		}
	}

	return status, nil
}

// PrintStatus renders a modern, high-speed status view
func (e *Engine) PrintStatus(repoPath string) error {
	start := time.Now()
	status, err := e.GetStatus(repoPath)
	duration := time.Since(start)

	if err != nil {
		return err
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1)

	branchStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#04B575"))

	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262"))

	tagStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	fmt.Println(titleStyle.Render("⚡ Shellix Instant Git Status"))
	fmt.Printf("On branch: %s  %s\n", branchStyle.Render(status.Branch), tagStyle.Render(fmt.Sprintf("[%s]", duration)))

	if status.HeadCommit != "" {
		fmt.Printf("Latest: %s %s (%s)\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68")).Render(status.HeadCommit),
			status.HeadMessage,
			metaStyle.Render(status.HeadAuthor),
		)
	}
	fmt.Println()

	if status.IsClean {
		cleanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
		fmt.Println(cleanStyle.Render("✔ Working tree clean, nothing to commit"))
		return nil
	}

	stagedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A"))
	unstagedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E"))
	untrackedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7"))

	if len(status.Staged) > 0 {
		fmt.Println(stagedStyle.Render("Changes to be committed:"))
		for _, f := range status.Staged {
			fmt.Printf("  %s\n", stagedStyle.Render(f))
		}
		fmt.Println()
	}

	if len(status.Unstaged) > 0 {
		fmt.Println(unstagedStyle.Render("Changes not staged for commit:"))
		for _, f := range status.Unstaged {
			fmt.Printf("  %s\n", unstagedStyle.Render(f))
		}
		fmt.Println()
	}

	if len(status.Untracked) > 0 {
		fmt.Println(untrackedStyle.Render("Untracked files:"))
		for _, f := range status.Untracked {
			fmt.Printf("  %s\n", untrackedStyle.Render("? "+f))
		}
		fmt.Println()
	}

	return nil
}

// RunPassthrough passes arbitrary Git commands directly to the native git executable
func (e *Engine) RunPassthrough(args []string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
