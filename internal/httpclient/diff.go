package httpclient

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// CompareEndpoints fetches two HTTP endpoints and computes a detailed structural diff
func (c *Client) CompareEndpoints(url1, url2 string, opts RequestOptions) (*DiffResult, error) {
	var resp1, resp2 *ResponseSummary
	var err1, err2 error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		opts1 := opts
		opts1.URL = url1
		resp1, err1 = c.Execute(opts1)
	}()

	go func() {
		defer wg.Done()
		opts2 := opts
		opts2.URL = url2
		resp2, err2 = c.Execute(opts2)
	}()

	wg.Wait()

	if err1 != nil {
		return nil, fmt.Errorf("failed fetching URL 1 (%s): %w", url1, err1)
	}
	if err2 != nil {
		return nil, fmt.Errorf("failed fetching URL 2 (%s): %w", url2, err2)
	}

	res := &DiffResult{
		URL1:          url1,
		URL2:          url2,
		StatusMatch:   resp1.StatusCode == resp2.StatusCode,
		Status1:       resp1.StatusCode,
		Status2:       resp2.StatusCode,
		Latency1:      resp1.Timing.TotalTime,
		Latency2:      resp2.Timing.TotalTime,
		LatencyDelta:  resp2.Timing.TotalTime - resp1.Timing.TotalTime,
		AddedKeys:     make([]string, 0),
		RemovedKeys:   make([]string, 0),
		ChangedValues: make([]ValueMismatch, 0),
	}

	// Compare bodies
	if resp1.IsJSON && resp2.IsJSON {
		var j1, j2 interface{}
		if err := json.Unmarshal(resp1.Body, &j1); err == nil {
			if err := json.Unmarshal(resp2.Body, &j2); err == nil {
				added, removed, changed := DiffJSON(j1, j2, "$")
				res.AddedKeys = added
				res.RemovedKeys = removed
				res.ChangedValues = changed
				res.ContentMatch = len(added) == 0 && len(removed) == 0 && len(changed) == 0
				return res, nil
			}
		}
	}

	// Fallback byte equality
	res.ContentMatch = string(resp1.Body) == string(resp2.Body)
	return res, nil
}

// DiffJSON performs deep recursive structural diff between two JSON objects or arrays
func DiffJSON(v1, v2 interface{}, currentPath string) (added []string, removed []string, changed []ValueMismatch) {
	if reflect.TypeOf(v1) != reflect.TypeOf(v2) {
		changed = append(changed, ValueMismatch{
			Path:   currentPath,
			Value1: v1,
			Value2: v2,
		})
		return
	}

	switch val1 := v1.(type) {
	case map[string]interface{}:
		val2 := v2.(map[string]interface{})

		// Find removed keys and common keys
		for k, item1 := range val1 {
			childPath := currentPath + "." + k
			item2, exists := val2[k]
			if !exists {
				removed = append(removed, childPath)
			} else {
				a, r, c := DiffJSON(item1, item2, childPath)
				added = append(added, a...)
				removed = append(removed, r...)
				changed = append(changed, c...)
			}
		}

		// Find added keys
		for k := range val2 {
			if _, exists := val1[k]; !exists {
				added = append(added, currentPath+"."+k)
			}
		}

	case []interface{}:
		val2 := v2.([]interface{})
		len1 := len(val1)
		len2 := len(val2)

		minLen := len1
		if len2 < minLen {
			minLen = len2
		}

		for i := 0; i < minLen; i++ {
			indexPath := fmt.Sprintf("%s[%d]", currentPath, i)
			a, r, c := DiffJSON(val1[i], val2[i], indexPath)
			added = append(added, a...)
			removed = append(removed, r...)
			changed = append(changed, c...)
		}

		if len1 > len2 {
			for i := len2; i < len1; i++ {
				removed = append(removed, fmt.Sprintf("%s[%d]", currentPath, i))
			}
		} else if len2 > len1 {
			for i := len1; i < len2; i++ {
				added = append(added, fmt.Sprintf("%s[%d]", currentPath, i))
			}
		}

	default:
		if !reflect.DeepEqual(v1, v2) {
			changed = append(changed, ValueMismatch{
				Path:   currentPath,
				Value1: v1,
				Value2: v2,
			})
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	return
}

// RenderDiffReport generates a styled terminal report comparing two responses
func RenderDiffReport(res *DiffResult) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA500"))
	failStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4500"))
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4500"))
	changedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF"))

	var b strings.Builder
	b.WriteString(titleStyle.Render("⚡ API Response & Benchmark Diff Report") + "\n")
	b.WriteString(dimStyle.Render("─────────────────────────────────────────────────────────────") + "\n")
	b.WriteString(fmt.Sprintf("Endpoint 1: %s\n", res.URL1))
	b.WriteString(fmt.Sprintf("Endpoint 2: %s\n\n", res.URL2))

	// 1. Status Code Comparison
	statusIcon := successStyle.Render("✔ MATCH")
	if !res.StatusMatch {
		statusIcon = failStyle.Render("✖ MISMATCH")
	}
	b.WriteString(fmt.Sprintf("Status Code: [%d] vs [%d]  •  %s\n", res.Status1, res.Status2, statusIcon))

	// 2. Latency Benchmarks
	fasterText := "Identical speed"
	if res.LatencyDelta > 0 {
		fasterText = fmt.Sprintf("Endpoint 1 is %s faster", res.LatencyDelta.Round(time.Millisecond))
	} else if res.LatencyDelta < 0 {
		fasterText = fmt.Sprintf("Endpoint 2 is %s faster", (-res.LatencyDelta).Round(time.Millisecond))
	}
	b.WriteString(fmt.Sprintf("Latency:     %s vs %s  •  %s\n",
		res.Latency1.Round(time.Millisecond), res.Latency2.Round(time.Millisecond), warnStyle.Render(fasterText)))

	// 3. Payload Match
	contentBadge := successStyle.Render("✔ IDENTICAL PAYLOAD")
	if !res.ContentMatch {
		contentBadge = failStyle.Render(fmt.Sprintf("✖ DIFF DETECTED (%d added, %d removed, %d changed)",
			len(res.AddedKeys), len(res.RemovedKeys), len(res.ChangedValues)))
	}
	b.WriteString(fmt.Sprintf("Payload:     %s\n\n", contentBadge))

	// 4. Structural Differences
	if len(res.AddedKeys) > 0 {
		b.WriteString(addedStyle.Render("➕ Added Keys in Endpoint 2:") + "\n")
		for _, k := range res.AddedKeys {
			b.WriteString(fmt.Sprintf("   + %s\n", k))
		}
		b.WriteString("\n")
	}

	if len(res.RemovedKeys) > 0 {
		b.WriteString(removedStyle.Render("➖ Removed Keys in Endpoint 2:") + "\n")
		for _, k := range res.RemovedKeys {
			b.WriteString(fmt.Sprintf("   - %s\n", k))
		}
		b.WriteString("\n")
	}

	if len(res.ChangedValues) > 0 {
		b.WriteString(changedStyle.Render("🔄 Value Mismatches:") + "\n")
		for _, c := range res.ChangedValues {
			b.WriteString(fmt.Sprintf("   ~ %s:\n       old: %v\n       new: %v\n", c.Path, c.Value1, c.Value2))
		}
		b.WriteString("\n")
	}

	return b.String()
}
