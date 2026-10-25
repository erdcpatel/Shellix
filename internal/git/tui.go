package git

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileEntry represents a file in the TUI file panel
type FileEntry struct {
	Name      string
	Path      string
	IsStaged  bool
	Untracked bool
}

type tuiModel struct {
	repoPath     string
	engine       *Engine
	status       *RepositoryStatus
	files        []FileEntry
	cursor       int
	currentDiff  string
	width        int
	height       int
	isCommitMode bool
	commitInput  textinput.Model
	statusMsg    string
	isErrMsg     bool
}

// LaunchTUI starts the interactive full-screen Git dashboard
func LaunchTUI(repoPath string) error {
	engine := NewEngine(true)
	st, err := engine.GetStatus(repoPath)
	if err != nil {
		return err
	}

	ti := textinput.New()
	ti.Placeholder = "Enter commit message (press Enter to commit, Esc to cancel)"
	ti.Focus()
	ti.CharLimit = 150
	ti.Width = 60

	m := tuiModel{
		repoPath:    repoPath,
		engine:      engine,
		status:      st,
		commitInput: ti,
	}
	m.rebuildFileList()
	m.updateDiff()

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func (m *tuiModel) rebuildFileList() {
	m.files = nil

	// 1. Staged files
	for _, f := range m.status.Staged {
		parts := strings.SplitN(strings.TrimSpace(f), " ", 2)
		name := f
		if len(parts) == 2 {
			name = parts[1]
		}
		m.files = append(m.files, FileEntry{
			Name:     name,
			Path:     name,
			IsStaged: true,
		})
	}

	// 2. Unstaged files
	for _, f := range m.status.Unstaged {
		parts := strings.SplitN(strings.TrimSpace(f), " ", 2)
		name := f
		if len(parts) == 2 {
			name = parts[1]
		}
		m.files = append(m.files, FileEntry{
			Name:     name,
			Path:     name,
			IsStaged: false,
		})
	}

	// 3. Untracked files
	for _, f := range m.status.Untracked {
		m.files = append(m.files, FileEntry{
			Name:      f,
			Path:      f,
			IsStaged:  false,
			Untracked: true,
		})
	}

	if m.cursor >= len(m.files) {
		m.cursor = len(m.files) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *tuiModel) updateDiff() {
	if len(m.files) == 0 || m.cursor >= len(m.files) {
		m.currentDiff = "✔ Working tree is clean. No files to inspect."
		return
	}

	target := m.files[m.cursor]
	diff, err := GetFileDiff(m.repoPath, target.Path, target.IsStaged)
	if err != nil {
		m.currentDiff = fmt.Sprintf("Error fetching diff: %v", err)
		return
	}
	m.currentDiff = diff
}

func (m *tuiModel) refresh() {
	st, err := m.engine.GetStatus(m.repoPath)
	if err == nil {
		m.status = st
		m.rebuildFileList()
		m.updateDiff()
	}
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.isCommitMode {
			switch msg.String() {
			case "esc":
				m.isCommitMode = false
				m.statusMsg = "Commit cancelled"
				m.isErrMsg = false
				return m, nil

			case "enter":
				msgText := strings.TrimSpace(m.commitInput.Value())
				if msgText == "" {
					m.statusMsg = "Commit message cannot be empty"
					m.isErrMsg = true
					return m, nil
				}
				err := Commit(m.repoPath, msgText)
				m.isCommitMode = false
				m.commitInput.Reset()
				if err != nil {
					m.statusMsg = fmt.Sprintf("Commit error: %v", err)
					m.isErrMsg = true
				} else {
					m.statusMsg = fmt.Sprintf("✔ Committed: %s", msgText)
					m.isErrMsg = false
					m.refresh()
				}
				return m, nil
			}

			var cmd tea.Cmd
			m.commitInput, cmd = m.commitInput.Update(msg)
			return m, cmd
		}

		// Normal mode keybindings
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateDiff()
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				m.updateDiff()
			}
			return m, nil

		case "s":
			if len(m.files) > 0 && m.cursor < len(m.files) {
				file := m.files[m.cursor]
				if err := StageFile(m.repoPath, file.Path); err != nil {
					m.statusMsg = fmt.Sprintf("Stage failed: %v", err)
					m.isErrMsg = true
				} else {
					m.statusMsg = fmt.Sprintf("✔ Staged: %s", file.Name)
					m.isErrMsg = false
					m.refresh()
				}
			}
			return m, nil

		case "u":
			if len(m.files) > 0 && m.cursor < len(m.files) {
				file := m.files[m.cursor]
				if err := UnstageFile(m.repoPath, file.Path); err != nil {
					m.statusMsg = fmt.Sprintf("Unstage failed: %v", err)
					m.isErrMsg = true
				} else {
					m.statusMsg = fmt.Sprintf("✔ Unstaged: %s", file.Name)
					m.isErrMsg = false
					m.refresh()
				}
			}
			return m, nil

		case "a":
			if err := StageAll(m.repoPath); err != nil {
				m.statusMsg = fmt.Sprintf("Stage all failed: %v", err)
				m.isErrMsg = true
			} else {
				m.statusMsg = "✔ Staged all changes"
				m.isErrMsg = false
				m.refresh()
			}
			return m, nil

		case "c":
			if len(m.status.Staged) == 0 {
				m.statusMsg = "No staged changes to commit! Tap 's' on a file to stage."
				m.isErrMsg = true
				return m, nil
			}
			m.isCommitMode = true
			m.commitInput.Reset()
			m.commitInput.Focus()
			return m, nil

		case "p":
			m.statusMsg = "Pushing to remote..."
			m.isErrMsg = false
			if err := Push(m.repoPath); err != nil {
				m.statusMsg = fmt.Sprintf("Push failed: %v", err)
				m.isErrMsg = true
			} else {
				m.statusMsg = "✔ Pushed to remote successfully"
				m.isErrMsg = false
				m.refresh()
			}
			return m, nil

		case "P":
			m.statusMsg = "Pulling from remote..."
			m.isErrMsg = false
			if err := Pull(m.repoPath); err != nil {
				m.statusMsg = fmt.Sprintf("Pull failed: %v", err)
				m.isErrMsg = true
			} else {
				m.statusMsg = "✔ Pulled changes successfully"
				m.isErrMsg = false
				m.refresh()
			}
			return m, nil

		case "r":
			m.refresh()
			m.statusMsg = "Refreshed repository status"
			m.isErrMsg = false
			return m, nil
		}
	}

	return m, nil
}

func (m tuiModel) View() string {
	if m.width == 0 {
		m.width = 100
	}
	if m.height == 0 {
		m.height = 30
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Width(m.width)

	headerText := fmt.Sprintf(" 🐚 Shellix Git Dashboard  |  Branch: %s  |  Commit: %s",
		m.status.Branch, m.status.HeadCommit)
	header := headerStyle.Render(headerText)

	contentHeight := m.height - 4
	if contentHeight < 10 {
		contentHeight = 10
	}

	leftWidth := int(float64(m.width)*0.38) - 2
	rightWidth := m.width - leftWidth - 4
	if leftWidth < 25 {
		leftWidth = 25
	}
	if rightWidth < 30 {
		rightWidth = 30
	}

	// 1. Build Left Pane (Files & Summary)
	paneBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7AA2F7")).
		Height(contentHeight).
		Width(leftWidth)

	var leftSb strings.Builder
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	leftSb.WriteString(titleStyle.Render(fmt.Sprintf("Changes (%d files)\n", len(m.files))))
	leftSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("──────────────────────\n"))

	if len(m.files) == 0 {
		leftSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render("\n✔ Working tree clean\n"))
	} else {
		for i, f := range m.files {
			prefix := "  "
			cursorMark := " "
			if i == m.cursor {
				cursorMark = "▸"
			}

			tag := "M"
			tagColor := lipgloss.Color("#F7768E") // red for unstaged
			if f.IsStaged {
				tag = "S"
				tagColor = lipgloss.Color("#9ECE6A") // green for staged
			} else if f.Untracked {
				tag = "?"
				tagColor = lipgloss.Color("#7AA2F7") // blue for untracked
			}

			tagBadge := lipgloss.NewStyle().Bold(true).Foreground(tagColor).Render(fmt.Sprintf("[%s]", tag))
			fileName := f.Name
			if len(fileName) > leftWidth-12 {
				fileName = "..." + fileName[len(fileName)-(leftWidth-15):]
			}

			line := fmt.Sprintf("%s %s %s %s", cursorMark, prefix, tagBadge, fileName)
			if i == m.cursor {
				line = lipgloss.NewStyle().
					Bold(true).
					Background(lipgloss.Color("#28345A")).
					Foreground(lipgloss.Color("#FFFFFF")).
					Width(leftWidth - 4).
					Render(line)
			}
			leftSb.WriteString(line + "\n")
		}
	}
	leftPane := paneBorder.Render(leftSb.String())

	// 2. Build Right Pane (Live Diff)
	diffBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Height(contentHeight).
		Width(rightWidth)

	var rightSb strings.Builder
	diffTitle := "Diff Viewer"
	if len(m.files) > 0 && m.cursor < len(m.files) {
		diffTitle = fmt.Sprintf("Diff: %s", m.files[m.cursor].Name)
	}
	rightSb.WriteString(titleStyle.Render(diffTitle + "\n"))
	rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("──────────────────────\n"))

	// Syntax highlight diff lines
	diffLines := strings.Split(m.currentDiff, "\n")
	maxLines := contentHeight - 3
	for idx, line := range diffLines {
		if idx >= maxLines {
			rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("... [diff truncated]\n"))
			break
		}
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#9ECE6A")).Render(line) + "\n")
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#F7768E")).Render(line) + "\n")
		} else if strings.HasPrefix(line, "@@") {
			rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7DCFFF")).Bold(true).Render(line) + "\n")
		} else {
			rightSb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#A9B1D6")).Render(line) + "\n")
		}
	}
	rightPane := diffBorder.Render(rightSb.String())

	// Join Panels horizontally
	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)

	// 3. Footer / Status bar
	var footer string
	if m.isCommitMode {
		promptStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6"))
		footer = promptStyle.Render("Commit: ") + m.commitInput.View()
	} else {
		var statusText string
		if m.statusMsg != "" {
			color := "#04B575"
			if m.isErrMsg {
				color = "#FF4500"
			}
			statusText = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render(m.statusMsg) + "  |  "
		}

		keyHelp := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Render(
			"[s]tage  [u]nstage  [a]ll  [c]ommit  [p]ush  [P]ull  [r]efresh  [q]uit",
		)
		footer = statusText + keyHelp
	}

	return fmt.Sprintf("%s\n%s\n%s", header, mainBody, footer)
}

// RenderSampleDashboard returns a rendered snapshot string for UI verification and documentation
func RenderSampleDashboard(width, height int) string {
	ti := textinput.New()
	ti.Focus()

	diffSample := `diff --git a/internal/git/tui.go b/internal/git/tui.go
index 8e45a2b..b921ef4 100644
--- a/internal/git/tui.go
+++ b/internal/git/tui.go
@@ -14,6 +14,12 @@ type FileEntry struct {
+	// Interactive TUI Elm Model
+	LaunchTUI(repoPath string) error
+	rebuildFileList()
+	updateDiff()
+	StageFile(repo, file)
+	Commit(repo, message)
 }`

	m := tuiModel{
		repoPath: ".",
		status: &RepositoryStatus{
			Branch:     "main",
			HeadCommit: "c1df587",
			Staged:     []string{"M internal/git/tui.go", "A internal/vault/keyring.go"},
			Unstaged:   []string{"M README.md"},
			Untracked:  []string{"configs/tokyo-night.yaml"},
		},
		commitInput: ti,
		currentDiff: diffSample,
		width:       width,
		height:      height,
		statusMsg:   "✔ Staged: internal/git/tui.go",
	}
	m.rebuildFileList()
	return m.View()
}
