package web

import (
	"regexp"
	"strconv"
	"strings"
)

// DiffLine represents a single line in a unified diff
type DiffLine struct {
	Type    string `json:"type"` // "hunk", "add", "del", "ctx"
	OldNum  int    `json:"old_num,omitempty"`
	NewNum  int    `json:"new_num,omitempty"`
	Content string `json:"content"`
}

// SplitDiffRow represents an aligned row for side-by-side split diff viewing
type SplitDiffRow struct {
	IsHunk       bool   `json:"is_hunk"`
	LeftNum      int    `json:"left_num,omitempty"`
	LeftType     string `json:"left_type,omitempty"` // "del", "ctx", ""
	LeftContent  string `json:"left_content,omitempty"`
	RightNum     int    `json:"right_num,omitempty"`
	RightType    string `json:"right_type,omitempty"` // "add", "ctx", ""
	RightContent string `json:"right_content,omitempty"`
}

// ParsedDiff contains structured representation of a file diff
type ParsedDiff struct {
	Raw       string         `json:"raw"`
	Additions int            `json:"additions"`
	Deletions int            `json:"deletions"`
	Lines     []DiffLine     `json:"lines"`
	SplitRows []SplitDiffRow `json:"split_rows"`
}

var hunkHeaderRegex = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// ParseUnifiedDiff converts raw git diff output into structured line and split-row models
func ParseUnifiedDiff(rawDiff string) ParsedDiff {
	lines := strings.Split(rawDiff, "\n")
	var diffLines []DiffLine
	var splitRows []SplitDiffRow

	oldLine := 1
	newLine := 1
	addCount := 0
	delCount := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			matches := hunkHeaderRegex.FindStringSubmatch(line)
			if len(matches) >= 3 {
				oldLine, _ = strconv.Atoi(matches[1])
				newLine, _ = strconv.Atoi(matches[2])
			}

			diffLines = append(diffLines, DiffLine{
				Type:    "hunk",
				Content: line,
			})
			splitRows = append(splitRows, SplitDiffRow{
				IsHunk:       true,
				LeftContent:  line,
				RightContent: line,
			})
			continue
		}

		if strings.HasPrefix(line, "+") {
			addCount++
			content := line[1:]
			diffLines = append(diffLines, DiffLine{
				Type:    "add",
				NewNum:  newLine,
				Content: content,
			})
			splitRows = append(splitRows, SplitDiffRow{
				RightNum:     newLine,
				RightType:    "add",
				RightContent: content,
			})
			newLine++
		} else if strings.HasPrefix(line, "-") {
			delCount++
			content := line[1:]
			diffLines = append(diffLines, DiffLine{
				Type:    "del",
				OldNum:  oldLine,
				Content: content,
			})
			splitRows = append(splitRows, SplitDiffRow{
				LeftNum:     oldLine,
				LeftType:    "del",
				LeftContent: content,
			})
			oldLine++
		} else if strings.HasPrefix(line, " ") || (len(line) > 0 && !strings.HasPrefix(line, "\\")) {
			content := line
			if strings.HasPrefix(line, " ") {
				content = line[1:]
			}
			diffLines = append(diffLines, DiffLine{
				Type:    "ctx",
				OldNum:  oldLine,
				NewNum:  newLine,
				Content: content,
			})
			splitRows = append(splitRows, SplitDiffRow{
				LeftNum:      oldLine,
				LeftType:     "ctx",
				LeftContent:  content,
				RightNum:     newLine,
				RightType:    "ctx",
				RightContent: content,
			})
			oldLine++
			newLine++
		}
	}

	return ParsedDiff{
		Raw:       rawDiff,
		Additions: addCount,
		Deletions: delCount,
		Lines:     diffLines,
		SplitRows: splitRows,
	}
}
