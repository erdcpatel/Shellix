package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StashEntry represents a saved git stash
type StashEntry struct {
	Index   int    `json:"index"`
	Branch  string `json:"branch"`
	Message string `json:"message"`
}

// ListStashes lists all saved git stashes
func ListStashes(dir string) ([]StashEntry, error) {
	cmd := exec.Command("git", "stash", "list")
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list stashes: %s", strings.TrimSpace(string(out)))
	}

	var stashes []StashEntry
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		// Format: stash@{0}: WIP on main: c1df587 message
		parts := strings.SplitN(l, ":", 3)
		if len(parts) >= 2 {
			idxStr := strings.TrimPrefix(strings.TrimSuffix(parts[0], "}"), "stash@{")
			idx, _ := strconv.Atoi(idxStr)
			branch := strings.TrimSpace(parts[1])
			msg := ""
			if len(parts) >= 3 {
				msg = strings.TrimSpace(parts[2])
			}
			stashes = append(stashes, StashEntry{
				Index:   idx,
				Branch:  branch,
				Message: msg,
			})
		}
	}
	return stashes, nil
}

// SaveStash pushes current unstaged/staged changes into git stash
func SaveStash(dir, message string) error {
	args := []string{"stash", "push"}
	if message != "" {
		args = append(args, "-m", message)
	}
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash push failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// PopStash applies and drops the stash at index (default 0)
func PopStash(dir string, index int) error {
	spec := fmt.Sprintf("stash@{%d}", index)
	cmd := exec.Command("git", "stash", "pop", spec)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash pop failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// DropStash deletes a stash at index
func DropStash(dir string, index int) error {
	spec := fmt.Sprintf("stash@{%d}", index)
	cmd := exec.Command("git", "stash", "drop", spec)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git stash drop failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// PrintStashesTable renders the list of stashes with styling
func PrintStashesTable(stashes []StashEntry) {
	if len(stashes) == 0 {
		fmt.Println("No git stashes found.")
		return
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	idxStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	branchStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))

	fmt.Printf("%s (%d saved)\n", titleStyle.Render("📦 Git Stashes"), len(stashes))
	fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────────"))
	for _, s := range stashes {
		fmt.Printf("  %s %-20s %s\n",
			idxStyle.Render(fmt.Sprintf("stash@{%d}", s.Index)),
			branchStyle.Render(s.Branch),
			s.Message)
	}
	fmt.Println("\nApply stash with: sx stash pop [index]")
}
