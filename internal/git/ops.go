package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// StageFile stages a file using git add
func StageFile(repoPath, file string) error {
	cmd := exec.Command("git", "add", file)
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// UnstageFile removes a file from staging using git restore --staged
func UnstageFile(repoPath, file string) error {
	cmd := exec.Command("git", "restore", "--staged", file)
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git restore --staged failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// StageAll stages all changes
func StageAll(repoPath string) error {
	cmd := exec.Command("git", "add", "-A")
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add -A failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Commit creates a commit with the specified message
func Commit(repoPath, message string) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("commit message cannot be empty")
	}
	cmd := exec.Command("git", "commit", "-m", message)
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Push executes git push
func Push(repoPath string) error {
	cmd := exec.Command("git", "push")
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Pull executes git pull
func Pull(repoPath string) error {
	cmd := exec.Command("git", "pull")
	if repoPath != "" {
		cmd.Dir = repoPath
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// GetFileDiff fetches the colored or raw diff for a specific file
func GetFileDiff(repoPath, file string, isStaged bool) (string, error) {
	var cmd *exec.Cmd
	if isStaged {
		cmd = exec.Command("git", "diff", "--cached", "--", file)
	} else {
		cmd = exec.Command("git", "diff", "--", file)
	}
	if repoPath != "" {
		cmd.Dir = repoPath
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	diffText := strings.TrimSpace(out.String())
	if diffText == "" {
		if isStaged {
			return "(File staged; new or unchanged contents)", nil
		}
		return "(Untracked or unmodified file)", nil
	}

	// Normalize CRLF to LF for consistent cross-platform terminal rendering
	diffText = strings.ReplaceAll(diffText, "\r\n", "\n")

	// Truncate gigantic diffs to prevent memory pressure and TUI frame lag
	const maxDiffLines = 500
	lines := strings.Split(diffText, "\n")
	if len(lines) > maxDiffLines {
		truncated := strings.Join(lines[:maxDiffLines], "\n")
		return fmt.Sprintf("%s\n\n▲ [Diff truncated: showing first %d lines of %d total lines. Run 'git diff %s' for full stream]", truncated, maxDiffLines, len(lines), file), nil
	}

	return diffText, nil
}
