package palette

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model powers the interactive fuzzy command palette
type Model struct {
	actions   []PaletteAction
	filtered  []PaletteAction
	cursor    int
	textInput textinput.Model
	width     int
	height    int
	Selected  *PaletteAction
	Quitted   bool
}

// NewModel initializes the command palette with available actions
func NewModel(actions []PaletteAction) Model {
	ti := textinput.New()
	ti.Placeholder = "Type to search flows, java, ssh, bookmarks, history..."
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 60

	m := Model{
		actions:   actions,
		textInput: ti,
		cursor:    0,
	}
	m.filter()
	return m
}

func (m *Model) filter() {
	query := strings.TrimSpace(strings.ToLower(m.textInput.Value()))
	if query == "" {
		m.filtered = make([]PaletteAction, len(m.actions))
		copy(m.filtered, m.actions)
		m.cursor = 0
		return
	}

	var res []PaletteAction
	for _, a := range m.actions {
		fullText := strings.ToLower(fmt.Sprintf("%s %s %s %s", a.Category, a.Title, a.Description, a.Command))
		if strings.Contains(fullText, query) {
			res = append(res, a)
		}
	}
	m.filtered = res
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	prevVal := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)
	if m.textInput.Value() != prevVal {
		m.filter()
	}
	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 {
		m.width = 100
		m.height = 24
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	borderStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#3A3F58")).Padding(0, 1)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true)
	badgeStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)

	header := fmt.Sprintf("⚡ Shellix Command Palette  •  %s available actions",
		dimStyle.Render(fmt.Sprintf("%d", len(m.actions))))

	inputBox := borderStyle.Width(m.width - 6).Render(
		m.textInput.View(),
	)

	maxRows := m.height - 10
	if maxRows < 5 {
		maxRows = 5
	}
	if maxRows > 14 {
		maxRows = 14
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
		a := m.filtered[i]
		prefix := "  "
		rowStyle := lipgloss.NewStyle()
		if i == m.cursor {
			prefix = cursorStyle.Render("▶ ")
			rowStyle = rowStyle.Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
		} else {
			rowStyle = rowStyle.Foreground(lipgloss.Color("#D0D0D0"))
		}

		// Category badge colors
		bColor := "#7AA2F7"
		switch a.Category {
		case CatFlow:
			bColor = "#FF79C6"
		case CatSSH:
			bColor = "#00BFFF"
		case CatBookmark:
			bColor = "#04B575"
		case CatSnippet:
			bColor = "#BD93F9"
		case CatHistory:
			bColor = "#6272A4"
		}

		badge := badgeStyle.Foreground(lipgloss.Color("#1A1B26")).Background(lipgloss.Color(bColor)).Render(fmt.Sprintf("%-8s", a.Category))
		title := a.Title
		if len(title) > 34 {
			title = title[:33] + "…"
		}
		desc := dimStyle.Render(a.Description)
		if len(a.Description) > 30 {
			desc = dimStyle.Render(a.Description[:29] + "…")
		}

		row := fmt.Sprintf("%s%s  %-35s  %s", prefix, badge, rowStyle.Render(title), desc)
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		rows = append(rows, dimStyle.Render("  No matching actions found. Try typing another keyword."))
	}

	listPanel := borderStyle.Width(m.width - 6).Height(maxRows + 2).Render(
		strings.Join(rows, "\n"),
	)

	footer := dimStyle.Render("↑/↓: navigate  •  enter: execute action  •  esc: close palette")

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(header),
		"",
		inputBox,
		listPanel,
		"",
		footer,
	)
}

// RenderSnapshot generates an ANSI string representation for visual test verification and documentation
func RenderSnapshot(actions []PaletteAction, query string) string {
	m := NewModel(actions)
	m.width = 100
	m.height = 24
	if query != "" {
		m.textInput.SetValue(query)
		m.filter()
	}
	return m.View()
}
