package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"shellix/internal/jsonview"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewJSONCmd manages JSON querying, formatting, and structural diffing
func NewJSONCmd(app *AppContext) *cobra.Command {
	jsonCmd := &cobra.Command{
		Use:   "json",
		Short: "Streaming JSON parser, JSONPath query engine, and structural diff",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return jsonview.StreamColorize(os.Stdin, os.Stdout)
			}
			f, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer f.Close()
			return jsonview.StreamColorize(f, os.Stdout)
		},
	}

	var rawOutput bool
	queryCmd := &cobra.Command{
		Use:   "query <json-path> [file]",
		Short: "Query JSON payload using JSONPath expressions (e.g. '.data.items[0]')",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			var reader io.Reader = os.Stdin

			if len(args) == 2 && args[1] != "-" {
				f, err := os.Open(args[1])
				if err != nil {
					return fmt.Errorf("failed to open file %s: %w", args[1], err)
				}
				defer f.Close()
				reader = f
			}

			val, err := jsonview.QueryFromReader(reader, path)
			if err != nil {
				return err
			}

			formatted, err := jsonview.FormatQueryResult(val, !rawOutput)
			if err != nil {
				return err
			}

			if rawOutput {
				fmt.Println(formatted)
			} else {
				highlighted, hErr := jsonview.ColorizeJSON([]byte(formatted))
				if hErr == nil {
					fmt.Println(highlighted)
				} else {
					fmt.Println(formatted)
				}
			}
			return nil
		},
	}
	queryCmd.Flags().BoolVarP(&rawOutput, "raw", "r", false, "Print raw unformatted output")

	formatCmd := &cobra.Command{
		Use:   "format [file]",
		Short: "Indent and colorize raw JSON from file or stdin",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var r io.Reader = os.Stdin
			if len(args) == 1 && args[0] != "-" {
				f, err := os.Open(args[0])
				if err != nil {
					return err
				}
				defer f.Close()
				r = f
			}
			return jsonview.StreamColorize(r, os.Stdout)
		},
	}

	diffCmd := &cobra.Command{
		Use:   "diff <file1> <file2>",
		Short: "Compute structural JSON diff and field comparison report between two documents",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := jsonview.CompareJSONFiles(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Print(res.RenderDiffReport())
			return nil
		},
	}

	jsonCmd.AddCommand(queryCmd, formatCmd, diffCmd)
	return jsonCmd
}

// NewViewCmd provides high-throughput streaming large file viewing
func NewViewCmd(app *AppContext) *cobra.Command {
	var maxLines int
	var tailMode bool
	var followMode bool
	var lineNums bool
	var searchPattern string

	viewCmd := &cobra.Command{
		Use:   "view <file>",
		Short: "High-throughput streaming viewer for large files and logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := jsonview.ViewerConfig{
				FilePath:        args[0],
				MaxLines:        maxLines,
				Tail:            tailMode,
				Follow:          followMode,
				ShowLineNumbers: lineNums,
				SearchPattern:   searchPattern,
			}

			sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			stat, err := os.Stat(args[0])
			if err == nil {
				titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
				dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
				modeDesc := "Head Stream"
				if tailMode {
					modeDesc = "Tail Seek"
				}
				if followMode {
					modeDesc += " (Follow)"
				}
				fmt.Printf("%s %s [%s, %s lines limit]\n",
					titleStyle.Render("📄 Shellix Streaming Viewer:"),
					args[0],
					dimStyle.Render(modeDesc),
					dimStyle.Render(fmt.Sprintf("%d", maxLines)),
				)
				fmt.Println(dimStyle.Render(fmt.Sprintf("Size: %s | Press Ctrl+C to exit", formatFileSize(stat.Size()))))
				fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────────────"))
			}

			return jsonview.StreamLargeFile(sigCtx, os.Stdout, cfg)
		},
	}

	viewCmd.Flags().IntVarP(&maxLines, "lines", "n", 50, "Number of lines to display")
	viewCmd.Flags().BoolVarP(&tailMode, "tail", "t", false, "Display the last N lines (chunked reverse seek)")
	viewCmd.Flags().BoolVarP(&followMode, "follow", "f", false, "Continuously stream new lines appended to file")
	viewCmd.Flags().BoolVarP(&lineNums, "numbers", "l", true, "Print line numbers")
	viewCmd.Flags().StringVarP(&searchPattern, "search", "s", "", "Highlight lines matching search text")

	return viewCmd
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
