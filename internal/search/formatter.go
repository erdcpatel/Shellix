package search

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PrintResults renders formatted search matches to the terminal based on SearchOptions
func PrintResults(summary *SearchSummary, opts SearchOptions) {
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	summaryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	fileHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E0AF68"))
	dirHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	tableHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	contextNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	matchStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6")).Background(lipgloss.Color("#282A36"))

	if summary.TotalMatches == 0 {
		fmt.Printf("No matches found for '%s' in %s (%d files, %d folders searched)\n",
			opts.Query, metaStyle.Render(summary.DurationStr), summary.TotalFilesSearched, summary.TotalDirsScanned)
		return
	}

	// 1. Dirs Only Mode
	if opts.DirsOnly {
		for _, d := range summary.UniqueDirs {
			fmt.Println(d)
		}
		return
	}

	// 2. Files Only Mode
	if opts.FilesOnly {
		for _, f := range summary.UniqueFiles {
			fmt.Println(f)
		}
		return
	}

	// 3. Count Only Mode
	if opts.CountOnly {
		for _, f := range summary.Folders {
			fmt.Printf("%-24s: %d matches across %d files\n", f.Name, f.TotalHits, f.MatchedFiles)
		}
		fmt.Printf("\nTotal: %d matches across %d files in %d folders (%s)\n",
			summary.TotalMatches, len(summary.UniqueFiles), len(summary.Folders), summary.DurationStr)
		return
	}

	// 4. Directory Grouped Rollup Mode
	if opts.GroupByDir {
		fmt.Printf("\n%s\n", dirHeaderStyle.Render(fmt.Sprintf("📊 Directory Search Summary for %q", opts.Query)))
		fmt.Println(borderStyle.Render("───────────────────────────────────────────────────────────────────────────────"))
		fmt.Printf("Scanned: %d folders | %d files | Duration: %s\n",
			summary.TotalDirsScanned, summary.TotalFilesSearched, metaStyle.Render(summary.DurationStr))
		cleanFolders := summary.TotalDirsScanned - len(summary.Folders)
		if cleanFolders < 0 {
			cleanFolders = 0
		}
		fmt.Printf("Hits:    %d matches across %d files in %d folders (%d clean folders)\n",
			summary.TotalMatches, len(summary.UniqueFiles), len(summary.Folders), cleanFolders)
		fmt.Println(borderStyle.Render("───────────────────────────────────────────────────────────────────────────────"))

		fmt.Printf("%-24s %-15s %-12s %s\n",
			tableHeaderStyle.Render("FOLDER"),
			tableHeaderStyle.Render("MATCHED FILES"),
			tableHeaderStyle.Render("TOTAL HITS"),
			tableHeaderStyle.Render("DIRECTORY PATH"),
		)
		fmt.Println(borderStyle.Render("───────────────────────────────────────────────────────────────────────────────"))
		for _, f := range summary.Folders {
			fmt.Printf("%-24s %-15d %-12d %s\n",
				f.Name, f.MatchedFiles, f.TotalHits, metaStyle.Render(f.Path))
		}
		fmt.Println(borderStyle.Render("───────────────────────────────────────────────────────────────────────────────"))

		// Detailed per-folder match display
		for _, f := range summary.Folders {
			fmt.Printf("\n%s\n", dirHeaderStyle.Render(fmt.Sprintf("📁 %s/ (%d files, %d matches)", f.Name, f.MatchedFiles, f.TotalHits)))

			currentFile := ""
			for _, m := range f.Matches {
				if m.Path != currentFile {
					currentFile = m.Path
					fmt.Printf("  %s\n", fileHeaderStyle.Render(currentFile))
				}

				if len(m.Context) > 0 {
					for _, c := range m.Context {
						if c.IsMatch {
							highlighted := highlightMatch(c.Content, m.MatchStart, m.MatchEnd, matchStyle)
							fmt.Printf("    %s %s\n", lineNumStyle.Render(fmt.Sprintf("%4d:", c.LineNumber)), strings.TrimSpace(highlighted))
						} else {
							fmt.Printf("    %s %s\n", contextNumStyle.Render(fmt.Sprintf("%4d-", c.LineNumber)), strings.TrimSpace(c.Content))
						}
					}
				} else {
					highlighted := highlightMatch(m.LineContent, m.MatchStart, m.MatchEnd, matchStyle)
					fmt.Printf("    %s %s\n", lineNumStyle.Render(fmt.Sprintf("%4d:", m.LineNumber)), strings.TrimSpace(highlighted))
				}
			}
		}

		printSummaryFooter(summary, opts, summaryStyle, metaStyle)
		return
	}

	// 5. Standard Flat Output Mode
	currentFile := ""
	for _, m := range summary.Matches {
		if m.Path != currentFile {
			currentFile = m.Path
			fmt.Printf("\n%s\n", fileHeaderStyle.Render("📂 "+currentFile))
		}

		if len(m.Context) > 0 {
			for _, c := range m.Context {
				if c.IsMatch {
					highlighted := highlightMatch(c.Content, m.MatchStart, m.MatchEnd, matchStyle)
					fmt.Printf("  %s %s\n", lineNumStyle.Render(fmt.Sprintf("%4d:", c.LineNumber)), strings.TrimSpace(highlighted))
				} else {
					fmt.Printf("  %s %s\n", contextNumStyle.Render(fmt.Sprintf("%4d-", c.LineNumber)), strings.TrimSpace(c.Content))
				}
			}
		} else {
			highlighted := highlightMatch(m.LineContent, m.MatchStart, m.MatchEnd, matchStyle)
			fmt.Printf("  %s %s\n", lineNumStyle.Render(fmt.Sprintf("%4d:", m.LineNumber)), strings.TrimSpace(highlighted))
		}
	}

	printSummaryFooter(summary, opts, summaryStyle, metaStyle)
}

func printSummaryFooter(summary *SearchSummary, opts SearchOptions, summaryStyle, metaStyle lipgloss.Style) {
	fmt.Printf("\n%s %s\n",
		summaryStyle.Render(fmt.Sprintf("Found %d matches across %d files in %d folders", summary.TotalMatches, len(summary.UniqueFiles), len(summary.Folders))),
		metaStyle.Render(fmt.Sprintf("(%s)", summary.DurationStr)),
	)
	if opts.ReportFile != "" {
		reportStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
		fmt.Printf("%s %s\n", reportStyle.Render("📄 Search audit report written to:"), opts.ReportFile)
	}
}

func highlightMatch(line string, start, end int, matchStyle lipgloss.Style) string {
	if start >= 0 && end <= len(line) && start <= end {
		before := line[:start]
		matched := matchStyle.Render(line[start:end])
		after := line[end:]
		return before + matched + after
	}
	return line
}
