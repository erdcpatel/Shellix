package palette

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func sampleActions() []PaletteAction {
	return []PaletteAction{
		{
			ID:          "flow-verify",
			Title:       "Run verification-pipeline flow",
			Description: "Automated build, test and search",
			Category:    CatFlow,
			Command:     "sx flow run verification-pipeline --tui",
		},
		{
			ID:          "git-tui",
			Title:       "Interactive Git Dashboard",
			Description: "Staging, diffing, and commits",
			Category:    CatCore,
			Command:     "sx g",
		},
		{
			ID:          "search-code",
			Title:       "Parallel Code Search",
			Description: "High-speed multi-repo search",
			Category:    CatCore,
			Command:     "sx search",
		},
		{
			ID:          "ssh-tunnel-bastion",
			Title:       "Open SSH tunnel to prod-bastion",
			Description: "Port forward 5432:localhost:5432",
			Category:    CatSSH,
			Command:     "sx ssh tunnel prod-bastion 5432:localhost:5432",
		},
		{
			ID:          "github-tui",
			Title:       "GitHub Workbench Dashboard",
			Description: "Pull requests, issues, and CI runs",
			Category:    CatCore,
			Command:     "sx gh",
		},
		{
			ID:          "bm-shellix",
			Title:       "Jump to bookmark: shellix",
			Description: "/Users/dhavalpatel/workspace/shellix",
			Category:    CatBookmark,
			Command:     "sx bm get shellix",
		},
		{
			ID:          "snip-test",
			Title:       "Run snippet: test-suite",
			Description: "go test -v ./...",
			Category:    CatSnippet,
			Command:     "sx snip run test-suite",
		},
	}
}

func TestPaletteFilteringAndNavigation(t *testing.T) {
	actions := sampleActions()
	m := NewModel(actions)

	if len(m.filtered) != len(actions) {
		t.Fatalf("expected %d initial actions, got %d", len(actions), len(m.filtered))
	}

	// 1. Filter by "code search"
	m.textInput.SetValue("code search")
	m.filter()
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 match for 'code search', got %d", len(m.filtered))
	}
	if m.filtered[0].Category != CatCore {
		t.Errorf("expected CatCore, got %v", m.filtered[0].Category)
	}

	// 2. Filter by "tunnel"
	m.textInput.SetValue("tunnel")
	m.filter()
	if len(m.filtered) != 1 || m.filtered[0].Category != CatSSH {
		t.Errorf("expected 1 match for 'tunnel', got %d", len(m.filtered))
	}

	// 3. Clear filter
	m.textInput.SetValue("")
	m.filter()
	if len(m.filtered) != len(actions) {
		t.Errorf("expected all actions restored on clear, got %d", len(m.filtered))
	}

	// 4. Down navigation
	mModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = mModel.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after down key, got %d", m.cursor)
	}

	// 5. Select action with Enter
	mModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mModel.(Model)
	if m.Selected == nil || m.Selected.ID != "git-tui" {
		t.Errorf("expected git-tui selected, got %+v", m.Selected)
	}
}

func TestRenderSnapshot(t *testing.T) {
	actions := sampleActions()
	snap := RenderSnapshot(actions, "flow")
	if !strings.Contains(snap, "verification-pipeline") {
		t.Errorf("expected verification-pipeline in snapshot")
	}

	allSnap := RenderSnapshot(actions, "")
	if os.Getenv("PRINT_SNAPSHOT") == "1" {
		fmt.Print(allSnap)
	}
}
