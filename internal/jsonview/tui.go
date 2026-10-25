package jsonview

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TreeNode represents a single node in the collapsible JSON hierarchy
type TreeNode struct {
	Key        string
	Path       string
	Type       string
	ValueStr   string
	Expanded   bool
	Depth      int
	Children   []*TreeNode
	ParentPath string
}

// JSONTUIModel is the interactive Bubble Tea model for JSON browsing
type JSONTUIModel struct {
	root        *TreeNode
	visibleRows []*TreeNode
	cursor      int
	viewportTop int
	height      int
	width       int
	filter      string
	filtering   bool
	statusMsg   string
	sourceLabel string
}

// NewJSONTUIModel initializes the tree from raw JSON bytes
func NewJSONTUIModel(data []byte, label string) (*JSONTUIModel, error) {
	var val interface{}
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	root := buildTreeNode("root", "$", val, 0)
	root.Expanded = true

	m := &JSONTUIModel{
		root:        root,
		height:      24,
		width:       80,
		sourceLabel: label,
	}
	m.rebuildVisibleRows()
	return m, nil
}

func buildTreeNode(key, path string, val interface{}, depth int) *TreeNode {
	node := &TreeNode{
		Key:      key,
		Path:     path,
		Depth:    depth,
		Expanded: depth < 2, // Auto-expand first 2 levels
	}

	switch v := val.(type) {
	case map[string]interface{}:
		node.Type = "object"
		node.ValueStr = fmt.Sprintf("{%d items}", len(v))

		var keys []string
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			subPath := path + "." + k
			if path == "$" {
				subPath = "." + k
			}
			child := buildTreeNode(k, subPath, v[k], depth+1)
			node.Children = append(node.Children, child)
		}

	case []interface{}:
		node.Type = "array"
		node.ValueStr = fmt.Sprintf("[%d items]", len(v))

		for i, elem := range v {
			subPath := fmt.Sprintf("%s[%d]", path, i)
			child := buildTreeNode(fmt.Sprintf("[%d]", i), subPath, elem, depth+1)
			node.Children = append(node.Children, child)
		}

	case string:
		node.Type = "string"
		node.ValueStr = fmt.Sprintf("%q", v)
	case float64:
		node.Type = "number"
		node.ValueStr = fmt.Sprintf("%v", v)
	case bool:
		node.Type = "bool"
		node.ValueStr = fmt.Sprintf("%v", v)
	case nil:
		node.Type = "null"
		node.ValueStr = "null"
	default:
		node.Type = "unknown"
		node.ValueStr = fmt.Sprintf("%v", v)
	}

	return node
}

func (m *JSONTUIModel) rebuildVisibleRows() {
	var rows []*TreeNode
	var collect func(node *TreeNode)
	collect = func(n *TreeNode) {
		if n != m.root {
			if m.filter == "" || strings.Contains(strings.ToLower(n.Key), strings.ToLower(m.filter)) || strings.Contains(strings.ToLower(n.ValueStr), strings.ToLower(m.filter)) {
				rows = append(rows, n)
			}
		}
		if n.Expanded {
			for _, child := range n.Children {
				collect(child)
			}
		}
	}

	collect(m.root)
	m.visibleRows = rows

	if m.cursor >= len(m.visibleRows) {
		m.cursor = len(m.visibleRows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m JSONTUIModel) Init() tea.Cmd {
	return nil
}

func (m JSONTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "enter", "esc":
				m.filtering = false
			case "backspace":
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
					m.rebuildVisibleRows()
				}
			default:
				if len(msg.String()) == 1 {
					m.filter += msg.String()
					m.rebuildVisibleRows()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			if m.cursor < m.viewportTop {
				m.viewportTop = m.cursor
			}

		case "down", "j":
			if m.cursor < len(m.visibleRows)-1 {
				m.cursor++
			}
			visibleLimit := m.height - 6
			if visibleLimit > 0 && m.cursor >= m.viewportTop+visibleLimit {
				m.viewportTop = m.cursor - visibleLimit + 1
			}

		case "right", "l", "enter":
			if len(m.visibleRows) > 0 {
				curr := m.visibleRows[m.cursor]
				if len(curr.Children) > 0 {
					curr.Expanded = !curr.Expanded
					m.rebuildVisibleRows()
				}
			}

		case "left", "h":
			if len(m.visibleRows) > 0 {
				curr := m.visibleRows[m.cursor]
				if curr.Expanded && len(curr.Children) > 0 {
					curr.Expanded = false
					m.rebuildVisibleRows()
				}
			}

		case "e":
			setAllExpanded(m.root, true)
			m.rebuildVisibleRows()
			m.statusMsg = "Expanded all nodes"

		case "c":
			setAllExpanded(m.root, false)
			m.root.Expanded = true
			m.rebuildVisibleRows()
			m.statusMsg = "Collapsed all nodes"

		case "/":
			m.filtering = true
			m.filter = ""
			m.rebuildVisibleRows()

		case "y":
			if len(m.visibleRows) > 0 {
				curr := m.visibleRows[m.cursor]
				m.statusMsg = fmt.Sprintf("Copied path: %s", curr.Path)
			}
		}
	}

	return m, nil
}

func setAllExpanded(n *TreeNode, expanded bool) {
	n.Expanded = expanded
	for _, child := range n.Children {
		setAllExpanded(child, expanded)
	}
}

func (m JSONTUIModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	cursorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68"))

	var s strings.Builder
	s.WriteString(titleStyle.Render(fmt.Sprintf("🌳 Shellix Interactive JSON Explorer (%s)", m.sourceLabel)) + "\n")

	if m.filtering {
		s.WriteString(fmt.Sprintf("Filter: %s█\n", m.filter))
	} else if m.filter != "" {
		s.WriteString(dimStyle.Render(fmt.Sprintf("Filter: %s (press / to clear)\n", m.filter)))
	} else {
		s.WriteString(dimStyle.Render("Keys: [↑/↓] Navigate  [Enter/→] Expand/Collapse  [e] Expand All  [c] Collapse  [/] Filter  [q] Quit\n"))
	}
	s.WriteString(dimStyle.Render("─────────────────────────────────────────────────────────────────────────────\n"))

	if len(m.visibleRows) == 0 {
		s.WriteString("  No matching JSON nodes found.\n")
		return s.String()
	}

	maxVisible := m.height - 6
	if maxVisible <= 0 {
		maxVisible = 18
	}

	endIdx := m.viewportTop + maxVisible
	if endIdx > len(m.visibleRows) {
		endIdx = len(m.visibleRows)
	}

	for i := m.viewportTop; i < endIdx; i++ {
		node := m.visibleRows[i]
		isCursor := i == m.cursor

		prefix := "  "
		if isCursor {
			prefix = cursorStyle.Render("▶ ")
		}

		indent := strings.Repeat("  ", node.Depth-1)

		toggle := "  "
		if len(node.Children) > 0 {
			if node.Expanded {
				toggle = "▼ "
			} else {
				toggle = "▶ "
			}
		}

		var line string
		if len(node.Children) > 0 {
			line = fmt.Sprintf("%s%s%s%s %s",
				prefix,
				indent,
				dimStyle.Render(toggle),
				keyStyle.Render(node.Key),
				typeStyle.Render(node.ValueStr),
			)
		} else {
			line = fmt.Sprintf("%s%s%s%s: %s",
				prefix,
				indent,
				dimStyle.Render("  "),
				keyStyle.Render(node.Key),
				highlightValueToken(node.ValueStr),
			)
		}

		s.WriteString(line + "\n")
	}

	s.WriteString(dimStyle.Render("─────────────────────────────────────────────────────────────────────────────\n"))
	if m.statusMsg != "" {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render(m.statusMsg) + "\n")
	} else if len(m.visibleRows) > 0 {
		selected := m.visibleRows[m.cursor]
		s.WriteString(dimStyle.Render(fmt.Sprintf("Path: %s (%s)", selected.Path, selected.Type)) + "\n")
	}

	return s.String()
}

// RunJSONTUI starts interactive tree explorer
func RunJSONTUI(r io.Reader, label string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	model, err := NewJSONTUIModel(data, label)
	if err != nil {
		return err
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
