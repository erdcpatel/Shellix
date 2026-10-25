package jsonview

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// CompareJSONFiles reads two JSON files and calculates structural differences
func CompareJSONFiles(path1, path2 string) (*JSONDiffResult, error) {
	b1, err := os.ReadFile(path1)
	if err != nil {
		return nil, fmt.Errorf("failed to read file 1 (%s): %w", path1, err)
	}

	b2, err := os.ReadFile(path2)
	if err != nil {
		return nil, fmt.Errorf("failed to read file 2 (%s): %w", path2, err)
	}

	res, err := CompareJSONBytes(b1, b2, path1, path2)
	if err != nil {
		return nil, err
	}
	res.Size1 = int64(len(b1))
	res.Size2 = int64(len(b2))
	return res, nil
}

// CompareJSONBytes analyzes structural differences between two JSON payloads
func CompareJSONBytes(b1, b2 []byte, label1, label2 string) (*JSONDiffResult, error) {
	var val1, val2 interface{}
	if err := json.Unmarshal(b1, &val1); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", label1, err)
	}
	if err := json.Unmarshal(b2, &val2); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", label2, err)
	}

	res := &JSONDiffResult{
		File1:         label1,
		File2:         label2,
		Size1:         int64(len(b1)),
		Size2:         int64(len(b2)),
		AddedKeys:     make([]string, 0),
		RemovedKeys:   make([]string, 0),
		ChangedValues: make([]ValueDiff, 0),
	}

	totalKeys, matchedKeys := diffRecursive("", val1, val2, res)
	sort.Strings(res.AddedKeys)
	sort.Strings(res.RemovedKeys)

	if len(res.AddedKeys) == 0 && len(res.RemovedKeys) == 0 && len(res.ChangedValues) == 0 {
		res.Identical = true
		res.MatchPercentage = 100.0
	} else if totalKeys > 0 {
		res.MatchPercentage = (float64(matchedKeys) / float64(totalKeys)) * 100.0
	}

	return res, nil
}

func diffRecursive(path string, val1, val2 interface{}, res *JSONDiffResult) (totalKeys, matchedKeys int) {
	if val1 == nil && val2 == nil {
		return 1, 1
	}

	if val1 == nil || val2 == nil {
		res.ChangedValues = append(res.ChangedValues, ValueDiff{Path: path, Val1: val1, Val2: val2})
		return 1, 0
	}

	// Compare Map / Object
	m1, ok1 := val1.(map[string]interface{})
	m2, ok2 := val2.(map[string]interface{})

	if ok1 && ok2 {
		allKeys := make(map[string]bool)
		for k := range m1 {
			allKeys[k] = true
		}
		for k := range m2 {
			allKeys[k] = true
		}

		for k := range allKeys {
			subPath := k
			if path != "" {
				subPath = path + "." + k
			}

			v1, in1 := m1[k]
			v2, in2 := m2[k]

			if !in1 && in2 {
				res.AddedKeys = append(res.AddedKeys, subPath)
				totalKeys++
				continue
			}
			if in1 && !in2 {
				res.RemovedKeys = append(res.RemovedKeys, subPath)
				totalKeys++
				continue
			}

			subTotal, subMatched := diffRecursive(subPath, v1, v2, res)
			totalKeys += subTotal
			matchedKeys += subMatched
		}
		return totalKeys, matchedKeys
	}

	// Compare Array / Slice
	s1, ok1 := val1.([]interface{})
	s2, ok2 := val2.([]interface{})

	if ok1 && ok2 {
		maxLen := len(s1)
		if len(s2) > maxLen {
			maxLen = len(s2)
		}

		for i := 0; i < maxLen; i++ {
			elemPath := fmt.Sprintf("%s[%d]", path, i)
			if i >= len(s1) {
				res.AddedKeys = append(res.AddedKeys, elemPath)
				totalKeys++
				continue
			}
			if i >= len(s2) {
				res.RemovedKeys = append(res.RemovedKeys, elemPath)
				totalKeys++
				continue
			}

			subTotal, subMatched := diffRecursive(elemPath, s1[i], s2[i], res)
			totalKeys += subTotal
			matchedKeys += subMatched
		}
		return totalKeys, matchedKeys
	}

	// Compare Primitives
	totalKeys = 1
	if reflect.DeepEqual(val1, val2) {
		matchedKeys = 1
	} else {
		res.ChangedValues = append(res.ChangedValues, ValueDiff{Path: path, Val1: val1, Val2: val2})
		matchedKeys = 0
	}

	return totalKeys, matchedKeys
}

// RenderDiffReport returns an ANSI styled comparison report
func (d *JSONDiffResult) RenderDiffReport() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E0AF68"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))

	var out strings.Builder
	out.WriteString(titleStyle.Render("🔍 JSON Structural Diff Report") + "\n")
	out.WriteString(borderStyle.Render("─────────────────────────────────────────────────────────────────────────────") + "\n")

	out.WriteString(fmt.Sprintf("  • File 1 : %s (%s)\n", keyStyle.Render(d.File1), dimStyle.Render(formatBytes(d.Size1))))
	out.WriteString(fmt.Sprintf("  • File 2 : %s (%s)\n", keyStyle.Render(d.File2), dimStyle.Render(formatBytes(d.Size2))))

	matchScore := fmt.Sprintf("%.1f%%", d.MatchPercentage)
	if d.Identical {
		out.WriteString(fmt.Sprintf("  • Status : %s\n", greenStyle.Render("✓ 100% Identical Structure & Values")))
	} else {
		out.WriteString(fmt.Sprintf("  • Match  : %s (Structural Similarity)\n", yellowStyle.Render(matchScore)))
	}

	out.WriteString(borderStyle.Render("─────────────────────────────────────────────────────────────────────────────") + "\n")

	if d.Identical {
		out.WriteString(dimStyle.Render("No schema or value differences detected between the two documents.\n\n"))
		return out.String()
	}

	// 1. Added Keys
	if len(d.AddedKeys) > 0 {
		out.WriteString(fmt.Sprintf("\n%s (%d):\n", greenStyle.Render("➕ Added Keys in File 2"), len(d.AddedKeys)))
		for _, k := range d.AddedKeys {
			out.WriteString(fmt.Sprintf("   + %s\n", greenStyle.Render(k)))
		}
	}

	// 2. Removed Keys
	if len(d.RemovedKeys) > 0 {
		out.WriteString(fmt.Sprintf("\n%s (%d):\n", redStyle.Render("➖ Removed Keys from File 1"), len(d.RemovedKeys)))
		for _, k := range d.RemovedKeys {
			out.WriteString(fmt.Sprintf("   - %s\n", redStyle.Render(k)))
		}
	}

	// 3. Changed Values
	if len(d.ChangedValues) > 0 {
		out.WriteString(fmt.Sprintf("\n%s (%d):\n", yellowStyle.Render("🔄 Value Mismatches"), len(d.ChangedValues)))
		for _, cv := range d.ChangedValues {
			v1Str, _ := json.Marshal(cv.Val1)
			v2Str, _ := json.Marshal(cv.Val2)
			out.WriteString(fmt.Sprintf("   • %-30s %s -> %s\n",
				keyStyle.Render(cv.Path),
				redStyle.Render(string(v1Str)),
				greenStyle.Render(string(v2Str)),
			))
		}
	}

	out.WriteString("\n")
	return out.String()
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
