package flow

import (
	"fmt"
	"strings"
	"time"

	"shellix/internal/core"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StepUIStatus reflects real-time visual progress of a workflow step
type StepUIStatus int

const (
	StepStatusPending StepUIStatus = iota
	StepStatusRunning
	StepStatusDone
	StepStatusSkipped
	StepStatusFailed
)

// StepUIState holds the visual and log state of a single step
type StepUIState struct {
	Name       string
	Status     StepUIStatus
	Duration   time.Duration
	Output     string
	Error      error
	IsParallel bool
}

type flowTUIModel struct {
	flow          *Flow
	ctx           *Context
	executor      *Executor
	steps         []StepUIState
	cursor        int
	spinner       spinner.Model
	viewport      viewport.Model
	width         int
	height        int
	isDone        bool
	flowErr       error
	totalDuration time.Duration
	eventChan     chan FlowEvent
	startTime     time.Time
}

// flowEventMsg wraps FlowEvent as a Bubble Tea message
type flowEventMsg FlowEvent

// tickMsg updates the live duration clock
type tickMsg time.Time

func newFlowTUIModel(f *Flow, ctx *Context) flowTUIModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF"))

	stepStates := make([]StepUIState, len(f.Steps))
	for i, s := range f.Steps {
		stepStates[i] = StepUIState{
			Name:       s.Name,
			Status:     StepStatusPending,
			IsParallel: s.Parallel,
		}
	}

	vp := viewport.New(60, 15)
	vp.SetContent("Waiting for step execution to begin...")

	eventChan := make(chan FlowEvent, 128)

	exec := NewExecutor(f, ctx)
	exec.Silent = true
	exec.OnEvent = func(ev FlowEvent) {
		eventChan <- ev
	}

	return flowTUIModel{
		flow:      f,
		ctx:       ctx,
		executor:  exec,
		steps:     stepStates,
		cursor:    0,
		spinner:   sp,
		viewport:  vp,
		eventChan: eventChan,
		startTime: time.Now(),
	}
}

// RunFlowTUI launches the interactive full-screen Bubble Tea dashboard for workflow execution
func RunFlowTUI(f *Flow, ctx *Context) error {
	m := newFlowTUIModel(f, ctx)

	p := tea.NewProgram(m, tea.WithAltScreen())

	// Run flow execution in background goroutine
	go func() {
		_, _ = m.executor.Execute()
	}()

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	fm, ok := finalModel.(flowTUIModel)
	if ok && fm.flowErr != nil {
		return fm.flowErr
	}
	return nil
}

// RenderSnapshot generates an ANSI formatted render of the Flow TUI for terminal captures and tests
func RenderSnapshot(f *Flow, ctx *Context) string {
	m := newFlowTUIModel(f, ctx)
	m.width = 100
	m.height = 24
	m.viewport.Width = m.width - 44
	m.viewport.Height = m.height - 10
	m.isDone = true
	m.totalDuration = 1170 * time.Millisecond
	for i := range m.steps {
		m.steps[i].Status = StepStatusDone
		m.steps[i].Duration = time.Duration(120+i*210) * time.Millisecond
	}
	if len(m.steps) > 1 {
		m.cursor = 1
		m.steps[1].Output = "=== RUN   TestFlowTUIModel\n--- PASS: TestFlowTUIModel (0.00s)\n=== RUN   TestDAGDependencyExecution\n--- PASS: TestDAGDependencyExecution (0.01s)\nPASS\nok  \tshellix/internal/flow  0.454s"
	}
	m.updateViewportContent()
	return m.View()
}

func (m flowTUIModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		waitForFlowEvent(m.eventChan),
		tickDuration(),
	)
}

func waitForFlowEvent(ch chan FlowEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return flowEventMsg(ev)
	}
}

func tickDuration() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m flowTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpWidth := m.width - 44
		if vpWidth < 30 {
			vpWidth = 30
		}
		vpHeight := m.height - 10
		if vpHeight < 8 {
			vpHeight = 8
		}
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
		m.updateViewportContent()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateViewportContent()
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.steps)-1 {
				m.cursor++
				m.updateViewportContent()
			}
			return m, nil
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tickMsg:
		if !m.isDone {
			cmds = append(cmds, tickDuration())
		}

	case flowEventMsg:
		switch msg.Type {
		case EventStepStart:
			for i := range m.steps {
				if m.steps[i].Name == msg.StepName {
					m.steps[i].Status = StepStatusRunning
					m.cursor = i
					m.updateViewportContent()
					break
				}
			}

		case EventStepDone:
			for i := range m.steps {
				if m.steps[i].Name == msg.StepName {
					m.steps[i].Duration = msg.Duration
					if msg.Result != nil {
						m.steps[i].Output = core.GlobalSanitizer.Sanitize(msg.Result.Output)
						if msg.Result.Skipped {
							m.steps[i].Status = StepStatusSkipped
						} else if msg.Result.Success {
							m.steps[i].Status = StepStatusDone
						} else {
							m.steps[i].Status = StepStatusFailed
							m.steps[i].Error = msg.Result.Error
						}
					}
					m.updateViewportContent()
					break
				}
			}

		case EventFlowDone:
			m.isDone = true
			m.flowErr = msg.Error
			m.totalDuration = msg.Duration
			m.updateViewportContent()
		}

		if !m.isDone {
			cmds = append(cmds, waitForFlowEvent(m.eventChan))
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *flowTUIModel) updateViewportContent() {
	if len(m.steps) == 0 || m.cursor >= len(m.steps) {
		m.viewport.SetContent("(No steps)")
		return
	}

	current := m.steps[m.cursor]
	if strings.TrimSpace(current.Output) == "" {
		if current.Status == StepStatusRunning {
			m.viewport.SetContent("Step is executing...")
		} else if current.Status == StepStatusPending {
			m.viewport.SetContent("Waiting to execute...")
		} else if current.Status == StepStatusSkipped {
			m.viewport.SetContent("Step skipped.")
		} else {
			m.viewport.SetContent("(No output generated)")
		}
	} else {
		m.viewport.SetContent(current.Output)
	}
	m.viewport.GotoBottom()
}

func (m flowTUIModel) View() string {
	if m.width == 0 {
		m.width = 100
		m.height = 30
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	timerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true)
	doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500")).Bold(true)
	skipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Bold(true)
	panelBorder := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3A3F58")).Padding(0, 1)

	// Elapsed time calculation
	elapsed := time.Since(m.startTime)
	if m.isDone && m.totalDuration > 0 {
		elapsed = m.totalDuration
	}

	// 1. Header & Progress
	completedCount := 0
	for _, s := range m.steps {
		if s.Status == StepStatusDone || s.Status == StepStatusSkipped {
			completedCount++
		}
	}

	headerText := fmt.Sprintf("🚀 Flow: %s  •  Elapsed: %s  •  Progress: [%d/%d]",
		m.flow.Name, timerStyle.Render(elapsed.Round(100*time.Millisecond).String()), completedCount, len(m.steps))
	if m.isDone {
		if m.flowErr == nil {
			headerText += doneStyle.Render("  ✨ [PASSED]")
		} else {
			headerText += failStyle.Render("  🚨 [FAILED]")
		}
	}

	// 2. Left Panel: Step List
	var stepRows []string
	stepListWidth := 36
	for i, s := range m.steps {
		prefix := "  "
		if i == m.cursor {
			prefix = cursorStyle.Render("▶ ")
		}

		icon := metaStyle.Render("⏳")
		switch s.Status {
		case StepStatusRunning:
			icon = m.spinner.View()
		case StepStatusDone:
			icon = doneStyle.Render("✔")
		case StepStatusSkipped:
			icon = skipStyle.Render("⏭")
		case StepStatusFailed:
			icon = failStyle.Render("✖")
		}

		nameStr := s.Name
		if len(nameStr) > 16 {
			nameStr = nameStr[:15] + "…"
		}

		durStr := ""
		if s.Duration > 0 {
			durStr = metaStyle.Render(fmt.Sprintf("(%s)", s.Duration.Round(time.Millisecond)))
		}

		parallelBadge := ""
		if s.IsParallel {
			parallelBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")).Render("⚡")
		}

		row := fmt.Sprintf("%s%s %-16s %s %s", prefix, icon, nameStr, parallelBadge, durStr)
		stepRows = append(stepRows, row)
	}

	stepListPanel := panelBorder.Width(stepListWidth).Height(m.height - 8).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7AA2F7")).Render("Steps Execution"),
			metaStyle.Render("────────────────────────────────"),
			strings.Join(stepRows, "\n"),
		),
	)

	// 3. Right Panel: Step Logs Viewport
	selectedStepName := "None"
	if len(m.steps) > 0 && m.cursor < len(m.steps) {
		selectedStepName = m.steps[m.cursor].Name
	}
	vpHeader := fmt.Sprintf("Live Logs: %s", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF")).Render(selectedStepName))
	logPanel := panelBorder.Width(m.width - stepListWidth - 8).Height(m.height - 8).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			vpHeader,
			metaStyle.Render(strings.Repeat("─", m.width-stepListWidth-12)),
			m.viewport.View(),
		),
	)

	// Combine left and right panels
	panels := lipgloss.JoinHorizontal(lipgloss.Top, stepListPanel, " ", logPanel)

	// 4. Footer Help
	footer := metaStyle.Render("↑/↓: select step  •  pgup/pgdn: scroll log  •  q/esc: quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(headerText),
		"",
		panels,
		"",
		footer,
	)
}
