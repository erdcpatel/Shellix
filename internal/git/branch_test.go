package git

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func sampleBranches() []BranchInfo {
	return []BranchInfo{
		{
			Name:       "main",
			IsCurrent:  true,
			Upstream:   "origin/main",
			Subject:    "feat: add interactive flow tui and history search",
			CommitTime: "2 hours ago",
		},
		{
			Name:       "feature/github-enterprise",
			IsCurrent:  false,
			Upstream:   "",
			Subject:    "feat: enterprise GHES and GHEC support",
			CommitTime: "1 hour ago",
		},
		{
			Name:       "fix/keyring-probe",
			IsCurrent:  false,
			Upstream:   "",
			Subject:    "fix: direct keyring probe with deferred cleanup",
			CommitTime: "1 day ago",
		},
		{
			Name:       "origin/main",
			IsRemote:   true,
			Subject:    "feat: add interactive flow tui and history search",
			CommitTime: "2 hours ago",
		},
	}
}

func TestBranchSwitcherModel(t *testing.T) {
	branches := sampleBranches()
	m := NewBranchSwitcherModel(branches)

	if len(m.filtered) != len(branches) {
		t.Fatalf("expected %d branches, got %d", len(branches), len(m.filtered))
	}

	// 1. Search for "enterprise"
	m.textInput.SetValue("enterprise")
	m.filter()
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 match for 'enterprise', got %d", len(m.filtered))
	}
	if m.filtered[0].Name != "feature/github-enterprise" {
		t.Errorf("expected feature/github-enterprise, got %s", m.filtered[0].Name)
	}

	// 2. Clear query
	m.textInput.SetValue("")
	m.filter()
	if len(m.filtered) != len(branches) {
		t.Errorf("expected all branches restored, got %d", len(m.filtered))
	}

	// 3. Move cursor down
	mModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = mModel.(BranchSwitcherModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}

	// 4. Select branch with Enter
	mModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mModel.(BranchSwitcherModel)
	if m.Selected == nil || m.Selected.Name != "feature/github-enterprise" {
		t.Errorf("expected feature/github-enterprise selected, got %+v", m.Selected)
	}
}

func TestRenderBranchSnapshot(t *testing.T) {
	branches := sampleBranches()
	snap := RenderBranchSnapshot(branches, "")
	if !strings.Contains(snap, "main") || !strings.Contains(snap, "feature/github-enterprise") {
		t.Errorf("expected branch names in snapshot")
	}

	if os.Getenv("PRINT_SNAPSHOT") == "1" {
		fmt.Print(snap)
	}
}
