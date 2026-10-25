package search

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// WriteReportToFile generates and saves a formatted search audit report to disk
func WriteReportToFile(summary *SearchSummary, targetPath string) error {
	dir := filepath.Dir(targetPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed creating report directory: %w", err)
		}
	}

	ext := strings.ToLower(filepath.Ext(targetPath))
	var data []byte
	var err error

	switch ext {
	case ".json":
		data, err = GenerateJSONReport(summary)
	case ".csv":
		data = []byte(GenerateCSVReport(summary))
	default:
		// Default to Markdown for .md or any unspecified extension
		data = []byte(GenerateMarkdownReport(summary))
	}

	if err != nil {
		return err
	}

	return os.WriteFile(targetPath, data, 0644)
}

// GenerateMarkdownReport creates a clean, GitHub-flavored Markdown audit report
func GenerateMarkdownReport(summary *SearchSummary) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Code Search Audit Report: `%s`\n\n", summary.Query))
	b.WriteString(fmt.Sprintf("> **Generated**: %s | **Duration**: %s | **Total Matches**: %d\n\n",
		time.Now().Format("2006-01-02 15:04:05"), summary.DurationStr, summary.TotalMatches))

	b.WriteString(fmt.Sprintf("Scanned **%d folders** (**%d files**) across the workspace. Found matches in **%d folders** across **%d unique files**.\n\n",
		summary.TotalDirsScanned, summary.TotalFilesSearched, len(summary.Folders), len(summary.UniqueFiles)))

	// 1. Folder Overview Table
	b.WriteString("## Folder Breakdown\n\n")
	b.WriteString("| Folder | Matched Files | Total Hits | Directory Path |\n")
	b.WriteString("|---|---|---|---|\n")

	for _, f := range summary.Folders {
		b.WriteString(fmt.Sprintf("| `%s` | %d | %d | `%s` |\n",
			f.Name, f.MatchedFiles, f.TotalHits, f.Path))
	}
	b.WriteString("\n---\n\n")

	// 2. File Checklists and Code Snippets
	b.WriteString("## Match Details\n\n")
	for _, f := range summary.Folders {
		b.WriteString(fmt.Sprintf("### 📁 `%s` (%d files, %d hits)\n\n", f.Name, f.MatchedFiles, f.TotalHits))

		// Group by file within folder
		fileMap := make(map[string][]MatchResult)
		var filesInOrder []string
		for _, m := range f.Matches {
			if _, exists := fileMap[m.Path]; !exists {
				filesInOrder = append(filesInOrder, m.Path)
			}
			fileMap[m.Path] = append(fileMap[m.Path], m)
		}

		for _, filePath := range filesInOrder {
			matches := fileMap[filePath]
			var lines []string
			for _, m := range matches {
				lines = append(lines, strconv.Itoa(m.LineNumber))
			}
			b.WriteString(fmt.Sprintf("- [ ] **`%s`** (Lines: %s)\n", filePath, strings.Join(lines, ", ")))

			for _, m := range matches {
				b.WriteString(fmt.Sprintf("  - Line %d: `%s`\n", m.LineNumber, strings.TrimSpace(m.LineContent)))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// GenerateJSONReport serializes search summary into indented JSON
func GenerateJSONReport(summary *SearchSummary) ([]byte, error) {
	return json.MarshalIndent(summary, "", "  ")
}

// GenerateCSVReport formats all search hits into RFC 4180 CSV
func GenerateCSVReport(summary *SearchSummary) string {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	_ = writer.Write([]string{"Folder", "FilePath", "LineNumber", "Snippet"})

	for _, m := range summary.Matches {
		dir := m.Directory
		if dir == "" {
			dir = "."
		}
		_ = writer.Write([]string{
			dir,
			m.Path,
			strconv.Itoa(m.LineNumber),
			strings.TrimSpace(m.LineContent),
		})
	}

	writer.Flush()
	return buf.String()
}
