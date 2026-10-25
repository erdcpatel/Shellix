package git

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIModelRender(t *testing.T) {
	ti := textinput.New()
	ti.Focus()

	model := tuiModel{
		repoPath: ".",
		status: &RepositoryStatus{
			Branch:     "main",
			HeadCommit: "c1df587",
			IsClean:    false,
			Staged:     []string{"M internal/git/tui.go"},
			Unstaged:   []string{"M README.md"},
			Untracked:  []string{"newfile.txt"},
		},
		commitInput: ti,
		width:       100,
		height:      24,
	}

	model.rebuildFileList()
	if len(model.files) != 3 {
		t.Fatalf("expected 3 files in list, got %d", len(model.files))
	}

	view := model.View()
	if !strings.Contains(view, "Shellix Git Dashboard") {
		t.Errorf("view missing dashboard title: %s", view)
	}
	if !strings.Contains(view, "Changes (3 files)") {
		t.Errorf("view missing changes count: %s", view)
	}
	if !strings.Contains(view, "Diff:") {
		t.Errorf("view missing Diff header: %s", view)
	}
	if !strings.Contains(view, "tui.go") {
		t.Errorf("view missing staged file name: %s", view)
	}
}

func TestTUIKeyNavigation(t *testing.T) {
	ti := textinput.New()
	ti.Focus()

	model := tuiModel{
		repoPath: ".",
		status: &RepositoryStatus{
			Branch:     "main",
			HeadCommit: "c1df587",
			Staged:     []string{"M file1.go"},
			Unstaged:   []string{"M file2.go"},
		},
		commitInput: ti,
		width:       100,
		height:      24,
	}
	model.rebuildFileList()

	// Initial cursor is 0
	if model.cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", model.cursor)
	}

	// Down key -> cursor becomes 1
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	m := updated.(tuiModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1 after down key, got %d", m.cursor)
	}

	// Up key -> cursor returns to 0
	updated2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m2 := updated2.(tuiModel)
	if m2.cursor != 0 {
		t.Errorf("expected cursor 0 after up key, got %d", m2.cursor)
	}
}
