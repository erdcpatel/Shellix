package jsonview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorKey        = lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF")).Bold(true)
	colorString     = lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
	colorNumber     = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B"))
	colorBool       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
	colorNull       = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Italic(true)
	colorPunctuation = lipgloss.NewStyle().Foreground(lipgloss.Color("#ABB2BF"))
)

// ColorizeJSON indents and applies syntax highlighting to raw JSON bytes
func ColorizeJSON(input []byte) (string, error) {
	var val interface{}
	if err := json.Unmarshal(input, &val); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	pretty, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return "", err
	}

	return highlightIndentedJSON(string(pretty)), nil
}

// StreamColorize reads JSON from reader and writes highlighted output to writer
func StreamColorize(r io.Reader, w io.Writer) error {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return err
	}

	highlighted, err := ColorizeJSON(buf.Bytes())
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(w, highlighted)
	return err
}

func highlightIndentedJSON(jsonStr string) string {
	lines := strings.Split(jsonStr, "\n")
	var out strings.Builder

	for i, line := range lines {
		if i > 0 {
			out.WriteString("\n")
		}

		trimmed := strings.TrimLeft(line, " ")
		indentLen := len(line) - len(trimmed)
		indent := strings.Repeat(" ", indentLen)

		out.WriteString(indent)
		out.WriteString(highlightLine(trimmed))
	}

	return out.String()
}

func highlightLine(line string) string {
	// Check if this is a "key": value line
	colonIdx := strings.Index(line, "\": ")
	if strings.HasPrefix(line, "\"") && colonIdx != -1 {
		keyPart := line[:colonIdx+1]
		rest := line[colonIdx+1:] // ": value..."

		var out strings.Builder
		out.WriteString(colorKey.Render(keyPart))
		out.WriteString(colorPunctuation.Render(": "))

		valuePart := strings.TrimSpace(rest[2:])
		out.WriteString(highlightValueToken(valuePart))
		return out.String()
	}

	return highlightValueToken(line)
}

func highlightValueToken(token string) string {
	trailingComma := strings.HasSuffix(token, ",")
	clean := strings.TrimSuffix(token, ",")

	var highlighted string
	switch {
	case clean == "{" || clean == "}" || clean == "[" || clean == "]" || clean == "{}" || clean == "[]":
		highlighted = colorPunctuation.Render(clean)
	case strings.HasPrefix(clean, "\"") && strings.HasSuffix(clean, "\""):
		highlighted = colorString.Render(clean)
	case clean == "true" || clean == "false":
		highlighted = colorBool.Render(clean)
	case clean == "null":
		highlighted = colorNull.Render(clean)
	default:
		// Check if number
		highlighted = colorNumber.Render(clean)
	}

	if trailingComma {
		highlighted += colorPunctuation.Render(",")
	}

	return highlighted
}
