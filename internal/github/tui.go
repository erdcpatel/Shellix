package github

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GitHubTab int

const (
	TabPRs GitHubTab = iota
	TabIssues
	TabCI
)

// GitHubTUIModel powers the full-screen interactive GitHub workbench
type GitHubTUIModel struct {
	slug        *RepoSlug
	client      *Client
	activeTab   GitHubTab
	prs         []PullRequest
	issues      []Issue
	runs        []WorkflowRun
	prCursor    int
	issueCursor int
	ciCursor    int
	viewport    viewport.Model
	width       int
	height      int
	loading     bool
	statusMsg   string
}

// NewGitHubTUIModel initializes the model with repository context
func NewGitHubTUIModel(slug *RepoSlug, client *Client) GitHubTUIModel {
	vp := viewport.New(60, 15)
	vp.SetContent("Loading GitHub repository details...")

	return GitHubTUIModel{
		slug:      slug,
		client:    client,
		activeTab: TabPRs,
		viewport:  vp,
		loading:   true,
	}
}

type dataLoadedMsg struct {
	prs    []PullRequest
	issues []Issue
	runs   []WorkflowRun
	err    error
}

func loadGitHubData(client *Client, slug *RepoSlug) tea.Cmd {
	return func() tea.Msg {
		prs, err := client.ListPullRequests(slug.Owner, slug.Repo, "open", 30)
		if err != nil {
			prs = []PullRequest{}
		}

		issues, err := client.ListIssues(slug.Owner, slug.Repo, "open", 30)
		if err != nil {
			issues = []Issue{}
		}

		runs, err := client.ListWorkflowRuns(slug.Owner, slug.Repo, "", 20)
		if err != nil {
			runs = []WorkflowRun{}
		}

		return dataLoadedMsg{
			prs:    prs,
			issues: issues,
			runs:   runs,
			err:    nil,
		}
	}
}

func (m GitHubTUIModel) Init() tea.Cmd {
	if m.client != nil && m.slug != nil {
		return loadGitHubData(m.client, m.slug)
	}
	return nil
}

func (m GitHubTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpWidth := m.width - 48
		if vpWidth < 30 {
			vpWidth = 30
		}
		vpHeight := m.height - 10
		if vpHeight < 8 {
			vpHeight = 8
		}
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
		m.updateViewport()
		return m, nil

	case dataLoadedMsg:
		m.loading = false
		m.prs = msg.prs
		m.issues = msg.issues
		m.runs = msg.runs
		m.updateViewport()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "l", "right":
			m.activeTab = (m.activeTab + 1) % 3
			m.updateViewport()
			return m, nil

		case "shift+tab", "h", "left":
			if m.activeTab == 0 {
				m.activeTab = TabCI
			} else {
				m.activeTab--
			}
			m.updateViewport()
			return m, nil

		case "1":
			m.activeTab = TabPRs
			m.updateViewport()
			return m, nil
		case "2":
			m.activeTab = TabIssues
			m.updateViewport()
			return m, nil
		case "3":
			m.activeTab = TabCI
			m.updateViewport()
			return m, nil

		case "up", "k":
			m.moveCursor(-1)
			m.updateViewport()
			return m, nil

		case "down", "j":
			m.moveCursor(1)
			m.updateViewport()
			return m, nil

		case "o":
			m.openCurrentInBrowser()
			return m, nil

		case "c":
			if m.activeTab == TabPRs && len(m.prs) > 0 && m.prCursor < len(m.prs) {
				branch := m.prs[m.prCursor].Head.Ref
				if branch != "" {
					out, err := exec.Command("git", "checkout", branch).CombinedOutput()
					if err == nil {
						m.statusMsg = fmt.Sprintf("✔ Switched to branch '%s'", branch)
					} else {
						m.statusMsg = fmt.Sprintf("✖ Git checkout failed: %s", strings.TrimSpace(string(out)))
					}
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *GitHubTUIModel) moveCursor(delta int) {
	switch m.activeTab {
	case TabPRs:
		if len(m.prs) == 0 {
			return
		}
		m.prCursor += delta
		if m.prCursor < 0 {
			m.prCursor = 0
		}
		if m.prCursor >= len(m.prs) {
			m.prCursor = len(m.prs) - 1
		}
	case TabIssues:
		if len(m.issues) == 0 {
			return
		}
		m.issueCursor += delta
		if m.issueCursor < 0 {
			m.issueCursor = 0
		}
		if m.issueCursor >= len(m.issues) {
			m.issueCursor = len(m.issues) - 1
		}
	case TabCI:
		if len(m.runs) == 0 {
			return
		}
		m.ciCursor += delta
		if m.ciCursor < 0 {
			m.ciCursor = 0
		}
		if m.ciCursor >= len(m.runs) {
			m.ciCursor = len(m.runs) - 1
		}
	}
}

func (m *GitHubTUIModel) openCurrentInBrowser() {
	var targetURL string
	switch m.activeTab {
	case TabPRs:
		if len(m.prs) > 0 && m.prCursor < len(m.prs) {
			targetURL = m.prs[m.prCursor].HTMLURL
		}
	case TabIssues:
		if len(m.issues) > 0 && m.issueCursor < len(m.issues) {
			targetURL = m.issues[m.issueCursor].HTMLURL
		}
	case TabCI:
		if len(m.runs) > 0 && m.ciCursor < len(m.runs) {
			targetURL = m.runs[m.ciCursor].HTMLURL
		}
	}

	if targetURL == "" {
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL)
	default:
		cmd = exec.Command("xdg-open", targetURL)
	}
	_ = cmd.Start()
	m.statusMsg = fmt.Sprintf("Opened %s in browser", targetURL)
}

func (m *GitHubTUIModel) updateViewport() {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7AA2F7"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF"))
	openBadge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575")).Render("● Open")

	var content string

	switch m.activeTab {
	case TabPRs:
		if len(m.prs) == 0 {
			m.viewport.SetContent("No open Pull Requests found.")
			return
		}
		pr := m.prs[m.prCursor]
		body := strings.TrimSpace(pr.Body)
		if body == "" {
			body = "(No description provided)"
		}
		content = fmt.Sprintf("%s  %s\n%s\n\n%s %s → %s\n%s @%s  •  %s\n%s %s\n\n%s\n%s",
			titleStyle.Render(fmt.Sprintf("#%d %s", pr.Number, pr.Title)),
			openBadge,
			metaStyle.Render(strings.Repeat("─", 50)),
			labelStyle.Render("Branches:"), pr.Head.Ref, pr.Base.Ref,
			labelStyle.Render("Author:"), pr.User.Login, pr.CreatedAt.Format("2006-01-02 15:04"),
			labelStyle.Render("URL:"), metaStyle.Render(pr.HTMLURL),
			titleStyle.Render("Description:"),
			body,
		)

	case TabIssues:
		if len(m.issues) == 0 {
			m.viewport.SetContent("No open Issues found.")
			return
		}
		it := m.issues[m.issueCursor]
		body := strings.TrimSpace(it.Body)
		if body == "" {
			body = "(No description provided)"
		}

		var labelNames []string
		for _, l := range it.Labels {
			labelNames = append(labelNames, "["+l.Name+"]")
		}
		labelsStr := "(None)"
		if len(labelNames) > 0 {
			labelsStr = strings.Join(labelNames, " ")
		}

		content = fmt.Sprintf("%s  %s\n%s\n\n%s @%s  •  %s\n%s %s  •  %s %d\n%s %s\n\n%s\n%s",
			titleStyle.Render(fmt.Sprintf("#%d %s", it.Number, it.Title)),
			openBadge,
			metaStyle.Render(strings.Repeat("─", 50)),
			labelStyle.Render("Author:"), it.User.Login, it.CreatedAt.Format("2006-01-02 15:04"),
			labelStyle.Render("Labels:"), labelsStr, labelStyle.Render("Comments:"), it.Comments,
			labelStyle.Render("URL:"), metaStyle.Render(it.HTMLURL),
			titleStyle.Render("Description:"),
			body,
		)

	case TabCI:
		if len(m.runs) == 0 {
			m.viewport.SetContent("No GitHub Actions workflow runs found.")
			return
		}
		run := m.runs[m.ciCursor]
		statusStr := run.Conclusion
		if statusStr == "" {
			statusStr = run.Status
		}
		badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575")).Render("✔ " + statusStr)
		if statusStr == "failure" {
			badge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4500")).Render("✖ " + statusStr)
		}

		content = fmt.Sprintf("%s  %s\n%s\n\n%s %s\n%s %s  •  %s %s\n%s %s\n\n%s\nWorkflow run ID: %d\nTriggered by event: %s",
			titleStyle.Render(run.Name),
			badge,
			metaStyle.Render(strings.Repeat("─", 50)),
			labelStyle.Render("Branch:"), run.HeadBranch,
			labelStyle.Render("Event:"), run.Event, labelStyle.Render("Created:"), run.CreatedAt.Format("2006-01-02 15:04"),
			labelStyle.Render("URL:"), metaStyle.Render(run.HTMLURL),
			titleStyle.Render("Run Information:"),
			run.ID, run.Event,
		)
	}

	m.viewport.SetContent(content)
}

func (m GitHubTUIModel) View() string {
	if m.width == 0 {
		m.width = 100
		m.height = 26
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	borderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3A3F58")).Padding(0, 1)
	activeTabStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF")).Background(lipgloss.Color("#252836")).Padding(0, 2)
	inactiveTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Padding(0, 2)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true)
	greenBadge := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	redBadge := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")).Bold(true)

	repoName := "Repository"
	hostName := DefaultGitHubHost
	if m.slug != nil {
		repoName = fmt.Sprintf("%s/%s", m.slug.Owner, m.slug.Repo)
		if m.slug.Host != "" {
			hostName = m.slug.Host
		}
	}

	// 1. Header
	header := fmt.Sprintf("🐙 Shellix GitHub Workbench  •  %s  [%s]",
		titleStyle.Render(repoName), dimStyle.Render(hostName))

	// 2. Tabs
	t1 := inactiveTabStyle.Render(fmt.Sprintf("[1] PRs (%d)", len(m.prs)))
	t2 := inactiveTabStyle.Render(fmt.Sprintf("[2] Issues (%d)", len(m.issues)))
	t3 := inactiveTabStyle.Render(fmt.Sprintf("[3] CI Runs (%d)", len(m.runs)))

	switch m.activeTab {
	case TabPRs:
		t1 = activeTabStyle.Render(fmt.Sprintf("[1] PRs (%d)", len(m.prs)))
	case TabIssues:
		t2 = activeTabStyle.Render(fmt.Sprintf("[2] Issues (%d)", len(m.issues)))
	case TabCI:
		t3 = activeTabStyle.Render(fmt.Sprintf("[3] CI Runs (%d)", len(m.runs)))
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, t1, " ", t2, " ", t3)

	// 3. Left Panel Items List
	var rows []string
	listWidth := 40
	listHeight := m.height - 9
	if listHeight < 8 {
		listHeight = 8
	}

	switch m.activeTab {
	case TabPRs:
		for i, pr := range m.prs {
			prefix := "  "
			if i == m.prCursor {
				prefix = cursorStyle.Render("▶ ")
			}
			icon := greenBadge.Render("●")
			titleText := pr.Title
			if len(titleText) > 22 {
				titleText = titleText[:21] + "…"
			}
			rows = append(rows, fmt.Sprintf("%s%s #%-4d %s", prefix, icon, pr.Number, titleText))
		}
	case TabIssues:
		for i, it := range m.issues {
			prefix := "  "
			if i == m.issueCursor {
				prefix = cursorStyle.Render("▶ ")
			}
			icon := greenBadge.Render("●")
			titleText := it.Title
			if len(titleText) > 22 {
				titleText = titleText[:21] + "…"
			}
			rows = append(rows, fmt.Sprintf("%s%s #%-4d %s", prefix, icon, it.Number, titleText))
		}
	case TabCI:
		for i, run := range m.runs {
			prefix := "  "
			if i == m.ciCursor {
				prefix = cursorStyle.Render("▶ ")
			}
			icon := greenBadge.Render("✔")
			if run.Conclusion == "failure" {
				icon = redBadge.Render("✖")
			}
			nameText := run.Name
			if len(nameText) > 22 {
				nameText = nameText[:21] + "…"
			}
			rows = append(rows, fmt.Sprintf("%s%s %s", prefix, icon, nameText))
		}
	}

	if len(rows) == 0 {
		rows = append(rows, dimStyle.Render("  (No items to show)"))
	}

	leftPanel := borderStyle.Width(listWidth).Height(listHeight).Render(
		strings.Join(rows, "\n"),
	)

	// 4. Right Panel Viewport
	rightWidth := m.width - listWidth - 6
	if rightWidth < 30 {
		rightWidth = 30
	}
	rightPanel := borderStyle.Width(rightWidth).Height(listHeight).Render(
		m.viewport.View(),
	)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)

	// 5. Footer & Status
	status := ""
	if m.statusMsg != "" {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render(m.statusMsg) + "  •  "
	}
	footer := dimStyle.Render("tab/1-3: switch view  •  ↑/↓: navigate  •  c: checkout branch  •  o: open browser  •  q: quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabBar,
		panels,
		status+footer,
	)
}

// RunGitHubTUI launches the interactive Bubble Tea program
func RunGitHubTUI(slug *RepoSlug, client *Client) error {
	m := NewGitHubTUIModel(slug, client)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RenderSnapshot renders an ANSI string snapshot of the GitHub TUI dashboard for testing & docs
func RenderSnapshot(slug *RepoSlug, prs []PullRequest, issues []Issue, runs []WorkflowRun) string {
	m := NewGitHubTUIModel(slug, nil)
	m.width = 100
	m.height = 24
	m.prs = prs
	m.issues = issues
	m.runs = runs
	m.loading = false
	m.viewport.Width = m.width - 48
	m.viewport.Height = m.height - 10
	m.updateViewport()
	return m.View()
}
