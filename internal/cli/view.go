package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"shellix/internal/dataview"
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

	tuiCmd := &cobra.Command{
		Use:   "tui [file]",
		Short: "Interactive collapsible tree explorer for JSON documents",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var r io.Reader = os.Stdin
			label := "stdin"
			if len(args) == 1 && args[0] != "-" {
				f, err := os.Open(args[0])
				if err != nil {
					return err
				}
				defer f.Close()
				r = f
				label = args[0]
			}
			return jsonview.RunJSONTUI(r, label)
		},
	}

	jsonCmd.AddCommand(queryCmd, formatCmd, diffCmd, tuiCmd)
	return jsonCmd
}

// NewViewCmd provides high-throughput streaming large file and tabular data viewing
func NewViewCmd(app *AppContext) *cobra.Command {
	var maxLines int
	var headCount int
	var tailMode bool
	var followMode bool
	var lineNums bool
	var searchPattern string
	var schemaOnly bool
	var columnsStr string
	var exportFormat string

	viewCmd := &cobra.Command{
		Use:   "view <file>",
		Short: "High-throughput streaming viewer for Parquet, CSV, Excel, and large logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			cleanExt := strings.ToLower(filepath.Ext(filePath))
			isTabular := cleanExt == ".parquet" || cleanExt == ".csv" || cleanExt == ".tsv" || cleanExt == ".xlsx" ||
				strings.Contains(strings.ToLower(filePath), ".parquet.") ||
				strings.Contains(strings.ToLower(filePath), ".csv.")

			if !isTabular {
				if f, err := os.Open(filePath); err == nil {
					buf := make([]byte, 4)
					n, _ := f.Read(buf)
					_ = f.Close()
					if n >= 2 && buf[0] == 0x1F && buf[1] == 0x8B {
						isTabular = true
					} else if n >= 4 && string(buf[:4]) == "PAR1" {
						isTabular = true
					}
				}
			}

			if isTabular {
				var cols []string
				if columnsStr != "" {
					for _, c := range strings.Split(columnsStr, ",") {
						if clean := strings.TrimSpace(c); clean != "" {
							cols = append(cols, clean)
						}
					}
				}
				limit := headCount
				if limit <= 0 {
					limit = maxLines
				}
				opts := dataview.InspectOptions{
					Columns: cols,
					Limit:   limit,
					Filter:  searchPattern,
				}
				summary, err := dataview.InspectFile(filePath, opts)
				if err != nil {
					return fmt.Errorf("failed inspecting tabular file: %w", err)
				}

				if exportFormat != "" {
					out, err := dataview.ExportRows(summary, exportFormat)
					if err != nil {
						return err
					}
					_, _ = os.Stdout.Write(out)
					fmt.Println()
					return nil
				}

				printTabularSummary(summary, schemaOnly)
				return nil
			}

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
	viewCmd.Flags().IntVar(&headCount, "head", 25, "Preview first N rows of tabular data (Parquet/CSV)")
	viewCmd.Flags().BoolVarP(&tailMode, "tail", "t", false, "Display the last N lines (chunked reverse seek)")
	viewCmd.Flags().BoolVarP(&followMode, "follow", "f", false, "Continuously stream new lines appended to file")
	viewCmd.Flags().BoolVarP(&lineNums, "numbers", "l", true, "Print line numbers")
	viewCmd.Flags().StringVarP(&searchPattern, "search", "s", "", "Filter rows or highlight lines matching search text")
	viewCmd.Flags().BoolVar(&schemaOnly, "schema", false, "Print tabular schema definition and metadata only")
	viewCmd.Flags().StringVarP(&columnsStr, "columns", "c", "", "Comma-separated list of columns to project (e.g. 'id,service,latency_ms')")
	viewCmd.Flags().StringVar(&exportFormat, "export", "", "Export projected rows to 'csv' or 'json'")

	return viewCmd
}

// NewParquetCmd provides a dedicated shortcut command for Parquet viewing
func NewParquetCmd(app *AppContext) *cobra.Command {
	cmd := NewViewCmd(app)
	cmd.Use = "parquet <file>"
	cmd.Short = "Inspect, filter, and export Apache Parquet files (.parquet, .parquet.gz)"
	return cmd
}

func printTabularSummary(summary *dataview.DataFileSummary, schemaOnly bool) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	badgeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))
	colStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))

	fmt.Printf("%s %s\n", headerStyle.Render("📊 Shellix Tabular Data Viewer:"), summary.FileName)
	fmt.Printf("   Format: %s  •  Compression: %s  •  Total Rows: %s  •  Columns: %d  •  Size: %s\n",
		badgeStyle.Render(strings.ToUpper(summary.Format)),
		warnStyle.Render(summary.Compression),
		badgeStyle.Render(fmt.Sprintf("%d", summary.TotalRows)),
		summary.TotalCols,
		formatFileSize(summary.FileSize),
	)
	fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────────────"))

	if schemaOnly {
		fmt.Println(colStyle.Render("SCHEMA DEFINITION:"))
		for i, col := range summary.Columns {
			fmt.Printf("  %3d. %-24s %s\n", i+1, col.Name, dimStyle.Render(col.Type))
		}
		return
	}

	if len(summary.Rows) == 0 {
		fmt.Println(dimStyle.Render("No rows found matching criteria."))
		return
	}

	headers := summary.SelectedColumns
	if len(headers) == 0 {
		for _, col := range summary.Columns {
			headers = append(headers, col.Name)
		}
	}

	var headerRow []string
	for _, h := range headers {
		headerRow = append(headerRow, colStyle.Render(h))
	}
	fmt.Println(strings.Join(headerRow, "  |  "))
	fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────────────"))

	for _, r := range summary.Rows {
		var line []string
		for _, h := range headers {
			val := fmt.Sprintf("%v", r[h])
			if len(val) > 28 {
				val = val[:25] + "..."
			}
			line = append(line, val)
		}
		fmt.Println(strings.Join(line, "  |  "))
	}

	if summary.HasMore {
		fmt.Println(dimStyle.Render(fmt.Sprintf("... Showing %d of %d rows (use --head or --offset to view more)", len(summary.Rows), summary.TotalRows)))
	}
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
