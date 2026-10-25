package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BranchInfo holds metadata for a Git branch
type BranchInfo struct {
	Name       string `json:"name"`
	IsCurrent  bool   `json:"is_current"`
	IsRemote   bool   `json:"is_remote"`
	Upstream   string `json:"upstream"`
	Subject    string `json:"subject"`
	CommitTime string `json:"commit_time"`
}

// ListBranches enumerates local and remote branches with commit summaries
func ListBranches(dir string) ([]BranchInfo, error) {
	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)|%(HEAD)|%(upstream:short)|%(subject)|%(committerdate:relative)")
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %s", strings.TrimSpace(string(out)))
	}

	var branches []BranchInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		parts := strings.Split(l, "|")
		name := ""
		isCurrent := false
		upstream := ""
		subject := ""
		commitTime := ""

		if len(parts) > 0 {
			name = parts[0]
		}
		if len(parts) > 1 && parts[1] == "*" {
			isCurrent = true
		}
		if len(parts) > 2 {
			upstream = parts[2]
		}
		if len(parts) > 3 {
			subject = parts[3]
		}
		if len(parts) > 4 {
			commitTime = parts[4]
		}

		isRemote := strings.HasPrefix(name, "origin/") || strings.HasPrefix(name, "remotes/")
		branches = append(branches, BranchInfo{
			Name:       name,
			IsCurrent:  isCurrent,
			IsRemote:   isRemote,
			Upstream:   upstream,
			Subject:    subject,
			CommitTime: commitTime,
		})
	}

	return branches, nil
}

// BranchSwitcherModel provides an interactive fuzzy branch switcher
type BranchSwitcherModel struct {
	branches  []BranchInfo
	filtered  []BranchInfo
	cursor    int
	textInput textinput.Model
	width     int
	height    int
	Selected  *BranchInfo
	StatusMsg string
	Quitted   bool
}

// NewBranchSwitcherModel initializes the interactive branch switcher
func NewBranchSwitcherModel(branches []BranchInfo) BranchSwitcherModel {
	ti := textinput.New()
	ti.Placeholder = "Search branches (e.g. main, feature, release)..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 50

	m := BranchSwitcherModel{
		branches:  branches,
		textInput: ti,
		cursor:    0,
	}
	m.filter()
	return m
}

func (m *BranchSwitcherModel) filter() {
	q := strings.TrimSpace(strings.ToLower(m.textInput.Value()))
	if q == "" {
		m.filtered = make([]BranchInfo, len(m.branches))
		copy(m.filtered, m.branches)
		m.cursor = 0
		return
	}

	var res []BranchInfo
	for _, b := range m.branches {
		full := strings.ToLower(fmt.Sprintf("%s %s %s", b.Name, b.Upstream, b.Subject))
		if strings.Contains(full, q) {
			res = append(res, b)
		}
	}
	m.filtered = res
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
}

func (m BranchSwitcherModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m BranchSwitcherModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.Quitted = true
			return m, tea.Quit

		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil

		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.Selected = &m.filtered[m.cursor]
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	prev := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)
	if m.textInput.Value() != prev {
		m.filter()
	}
	return m, cmd
}

func (m BranchSwitcherModel) View() string {
	if m.width == 0 {
		m.width = 100
		m.height = 24
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	borderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3A3F58")).Padding(0, 1)
	currentBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true)
	remoteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	header := fmt.Sprintf("🌿 Shellix Branch Switcher  •  %s branches found",
		dimStyle.Render(fmt.Sprintf("%d", len(m.branches))))

	inputBox := borderStyle.Width(m.width - 6).Render(
		m.textInput.View(),
	)

	maxRows := m.height - 10
	if maxRows < 5 {
		maxRows = 5
	}
	if maxRows > 12 {
		maxRows = 12
	}

	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	var rows []string
	for i := start; i < end; i++ {
		b := m.filtered[i]
		prefix := "  "
		if i == m.cursor {
			prefix = cursorStyle.Render("▶ ")
		}

		icon := "  "
		nameStyle := lipgloss.NewStyle()
		if b.IsCurrent {
			icon = currentBadge.Render("* ")
			nameStyle = currentBadge
		} else if b.IsRemote {
			icon = remoteStyle.Render("☁ ")
			nameStyle = remoteStyle
		}

		bName := b.Name
		if len(bName) > 28 {
			bName = bName[:27] + "…"
		}

		subject := b.Subject
		if len(subject) > 35 {
			subject = subject[:34] + "…"
		}

		timeStr := dimStyle.Render(b.CommitTime)
		row := fmt.Sprintf("%s%s%-30s  %-36s  %s", prefix, icon, nameStyle.Render(bName), dimStyle.Render(subject), timeStr)
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		rows = append(rows, dimStyle.Render("  No branches matched search query."))
	}

	listPanel := borderStyle.Width(m.width - 6).Height(maxRows + 2).Render(
		strings.Join(rows, "\n"),
	)

	footer := dimStyle.Render("↑/↓: navigate  •  enter: checkout branch  •  esc: close")

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(header),
		"",
		inputBox,
		listPanel,
		"",
		footer,
	)
}

// RunBranchSwitcher launches interactive branch checkout TUI
func RunBranchSwitcher(dir string) error {
	branches, err := ListBranches(dir)
	if err != nil {
		return err
	}

	m := NewBranchSwitcherModel(branches)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	fm, ok := finalModel.(BranchSwitcherModel)
	if ok && fm.Selected != nil {
		targetBranch := fm.Selected.Name
		if strings.HasPrefix(targetBranch, "origin/") {
			targetBranch = strings.TrimPrefix(targetBranch, "origin/")
		}

		out, err := exec.Command("git", "checkout", targetBranch).CombinedOutput()
		if err != nil {
			return fmt.Errorf("git checkout failed: %s", strings.TrimSpace(string(out)))
		}
		fmt.Printf("✨ Switched to branch '%s'\n", targetBranch)
	}
	return nil
}

// CheckoutBranch checks out the specified branch
func CheckoutBranch(dir, targetBranch string) error {
	branch := targetBranch
	if strings.HasPrefix(branch, "origin/") {
		branch = strings.TrimPrefix(branch, "origin/")
	}
	cmd := exec.Command("git", "checkout", branch)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %s", strings.TrimSpace(string(out)))
	}
	fmt.Printf("✨ Switched to branch '%s'\n", branch)
	return nil
}

// RenderBranchSnapshot renders an ANSI snapshot for documentation and test verification
func RenderBranchSnapshot(branches []BranchInfo, query string) string {
	m := NewBranchSwitcherModel(branches)
	m.width = 100
	m.height = 22
	if query != "" {
		m.textInput.SetValue(query)
		m.filter()
	}
	return m.View()
}
