package jsonview

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// StreamLargeFile efficiently streams content from a large file with optional tail, line numbers, and follow
func StreamLargeFile(ctx context.Context, w io.Writer, cfg ViewerConfig) error {
	var file *os.File
	var err error

	if cfg.FilePath == "-" || cfg.FilePath == "" {
		file = os.Stdin
	} else {
		file, err = os.Open(cfg.FilePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", cfg.FilePath, err)
		}
		defer file.Close()
	}

	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
	highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color("#F1FA8C")).Foreground(lipgloss.Color("#000000")).Bold(true)

	printLine := func(lineNum int, text string) {
		if cfg.SearchPattern != "" && strings.Contains(text, cfg.SearchPattern) {
			text = strings.ReplaceAll(text, cfg.SearchPattern, highlightStyle.Render(cfg.SearchPattern))
		}

		if cfg.ShowLineNumbers && lineNum > 0 {
			fmt.Fprintf(w, "%s │ %s\n", numStyle.Render(fmt.Sprintf("%6d", lineNum)), text)
		} else {
			fmt.Fprintln(w, text)
		}
	}

	maxLines := cfg.MaxLines
	if maxLines <= 0 {
		maxLines = 100
	}

	// Tail mode for seekable files: seek from end backwards to avoid reading multi-gigabyte file
	if cfg.Tail && file != os.Stdin {
		lines, startLineNum, err := tailSeekLines(file, maxLines)
		if err == nil {
			for i, line := range lines {
				printLine(startLineNum+i, line)
			}

			if cfg.Follow {
				return followFile(ctx, file, w, printLine, startLineNum+len(lines))
			}
			return nil
		}
		// Fallback to streaming scan if seek fails
	}

	// Forward streaming
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // Allow lines up to 10MB

	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if !cfg.Tail && cfg.MaxLines > 0 && lineCount > cfg.MaxLines {
			break
		}
		printLine(lineCount, scanner.Text())
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("read error: %w", err)
	}

	if cfg.Follow && file != os.Stdin {
		return followFile(ctx, file, w, printLine, lineCount+1)
	}

	return nil
}

// tailSeekLines seeks from the end of a file in 64KB chunks to find the last n lines
func tailSeekLines(f *os.File, n int) ([]string, int, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		return []string{}, 1, nil
	}

	const chunkSize int64 = 64 * 1024
	var offset int64 = fileSize
	var linesFound []string
	var remainder []byte

	for offset > 0 && len(linesFound) <= n {
		readSize := chunkSize
		if offset < chunkSize {
			readSize = offset
		}
		offset -= readSize

		_, err := f.Seek(offset, io.SeekStart)
		if err != nil {
			return nil, 0, err
		}

		buf := make([]byte, readSize)
		_, err = io.ReadFull(f, buf)
		if err != nil {
			return nil, 0, err
		}

		if len(remainder) > 0 {
			buf = append(buf, remainder...)
			remainder = nil
		}

		// Split by newline
		parts := bytes.Split(buf, []byte("\n"))
		if offset > 0 {
			// First part might be incomplete, save as remainder
			remainder = parts[0]
			parts = parts[1:]
		}

		// Prepend lines
		for i := len(parts) - 1; i >= 0; i-- {
			line := string(parts[i])
			if i == len(parts)-1 && line == "" && len(linesFound) == 0 {
				continue // skip trailing newline
			}
			linesFound = append([]string{line}, linesFound...)
			if len(linesFound) >= n {
				break
			}
		}
	}

	// Reset seek position to end of file for follow mode
	_, _ = f.Seek(0, io.SeekEnd)

	if len(linesFound) > n {
		linesFound = linesFound[len(linesFound)-n:]
	}

	return linesFound, 1, nil
}

func followFile(ctx context.Context, f *os.File, w io.Writer, printLine func(int, string), startLine int) error {
	reader := bufio.NewReader(f)
	currentLine := startLine

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(200 * time.Millisecond)
					continue
				}
				return err
			}
			printLine(currentLine, strings.TrimRight(line, "\r\n"))
			currentLine++
		}
	}
}
