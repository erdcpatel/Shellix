package search

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PrintResults renders formatted search matches to the terminal
func PrintResults(summary *SearchSummary, query string) {
	fileHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E0AF68"))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	matchStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6")).Background(lipgloss.Color("#282A36"))
	summaryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	if summary.TotalMatches == 0 {
		fmt.Printf("No matches found for '%s' in %s (%d files searched)\n",
			query, metaStyle.Render(summary.Duration.String()), summary.TotalFilesSearched)
		return
	}

	currentFile := ""
	for _, m := range summary.Matches {
		if m.Path != currentFile {
			currentFile = m.Path
			fmt.Printf("\n%s\n", fileHeaderStyle.Render("📂 "+currentFile))
		}

		line := m.LineContent
		var highlighted string
		if m.MatchStart >= 0 && m.MatchEnd <= len(line) && m.MatchStart <= m.MatchEnd {
			before := line[:m.MatchStart]
			matched := matchStyle.Render(line[m.MatchStart:m.MatchEnd])
			after := line[m.MatchEnd:]
			highlighted = before + matched + after
		} else {
			highlighted = line
		}

		fmt.Printf("  %s %s\n", lineNumStyle.Render(fmt.Sprintf("%4d:", m.LineNumber)), strings.TrimSpace(highlighted))
	}

	fmt.Printf("\n%s %s\n",
		summaryStyle.Render(fmt.Sprintf("Found %d matches across %d files", summary.TotalMatches, summary.TotalFilesSearched)),
		metaStyle.Render(fmt.Sprintf("(%s)", summary.Duration)),
	)
}
