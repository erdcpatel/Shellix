package cli

import (
	"strings"

	"shellix/internal/search"

	"github.com/spf13/cobra"
)

// NewSearchCmd handles folder-based parallel code and directory search
func NewSearchCmd(app *AppContext) *cobra.Command {
	var searchPaths []string
	var searchRegex bool
	var searchCaseSensitive bool
	var searchWholeWord bool
	var searchMax int
	var searchExt string
	var searchExclude []string
	var searchContext int
	var searchAfterContext int
	var searchBeforeContext int
	var searchGroupByDir bool
	var searchDirsOnly bool
	var searchFilesOnly bool
	var searchCountOnly bool
	var searchReportFile string

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Blazing-fast parallel folder, directory, and code search with audit reporting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine := search.NewEngine()

			var exts []string
			if searchExt != "" {
				for _, e := range strings.Split(searchExt, ",") {
					cleaned := strings.TrimSpace(e)
					if cleaned != "" {
						exts = append(exts, cleaned)
					}
				}
			}

			cBefore := searchBeforeContext
			cAfter := searchAfterContext
			if searchContext > 0 {
				cBefore = searchContext
				cAfter = searchContext
			}

			opts := search.SearchOptions{
				Query:         args[0],
				Paths:         searchPaths,
				IsRegex:       searchRegex,
				CaseSensitive: searchCaseSensitive,
				WholeWord:     searchWholeWord,
				MaxResults:    searchMax,
				Extensions:    exts,
				ExcludeGlobs:  searchExclude,
				ContextBefore: cBefore,
				ContextAfter:  cAfter,
				GroupByDir:    searchGroupByDir,
				DirsOnly:      searchDirsOnly,
				FilesOnly:     searchFilesOnly,
				CountOnly:     searchCountOnly,
				ReportFile:    searchReportFile,
			}

			summary, err := engine.Search(opts)
			if err != nil {
				return err
			}

			search.PrintResults(summary, opts)
			return nil
		},
	}

	searchCmd.Flags().StringSliceVarP(&searchPaths, "path", "p", []string{"."}, "Path(s) to search")
	searchCmd.Flags().BoolVarP(&searchRegex, "regex", "r", false, "Enable regular expression search")
	searchCmd.Flags().BoolVarP(&searchCaseSensitive, "case-sensitive", "s", false, "Perform case-sensitive search (defaults to case-insensitive)")
	searchCmd.Flags().BoolVarP(&searchWholeWord, "word", "w", false, "Match only whole words")
	searchCmd.Flags().IntVarP(&searchMax, "max", "m", 500, "Maximum matching results to collect")
	searchCmd.Flags().StringVar(&searchExt, "ext", "", "Comma-separated file extensions to search (e.g. 'go,java,json')")
	searchCmd.Flags().StringArrayVar(&searchExclude, "exclude", nil, "Glob pattern(s) to exclude from search (e.g. '*.min.js')")
	searchCmd.Flags().IntVarP(&searchContext, "context", "C", 0, "Number of context lines before and after match")
	searchCmd.Flags().IntVarP(&searchAfterContext, "after-context", "A", 0, "Number of context lines after match")
	searchCmd.Flags().IntVarP(&searchBeforeContext, "before-context", "B", 0, "Number of context lines before match")
	searchCmd.Flags().BoolVar(&searchGroupByDir, "by-dir", false, "Group matches by top-level project folder with rollup summary table")
	searchCmd.Flags().BoolVar(&searchDirsOnly, "dirs-only", false, "Print only unique matching folder paths")
	searchCmd.Flags().BoolVarP(&searchFilesOnly, "files-only", "l", false, "Print only unique matching file paths")
	searchCmd.Flags().BoolVarP(&searchCountOnly, "count", "c", false, "Print match count per folder instead of lines")
	searchCmd.Flags().StringVar(&searchReportFile, "report", "", "Save search audit report to disk (.md, .json, or .csv)")

	return searchCmd
}
