package cli

import (
	"shellix/internal/search"

	"github.com/spf13/cobra"
)

// NewSearchCmd handles cross-repo parallel search
func NewSearchCmd(app *AppContext) *cobra.Command {
	var searchPaths []string
	var searchRegex bool
	var searchMax int

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Blazing-fast multi-repo code and text search",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			engine := search.NewEngine()
			opts := search.SearchOptions{
				Query:      args[0],
				Paths:      searchPaths,
				IsRegex:    searchRegex,
				MaxResults: searchMax,
			}
			summary, err := engine.Search(opts)
			if err != nil {
				return err
			}
			search.PrintResults(summary, args[0])
			return nil
		},
	}

	searchCmd.Flags().StringSliceVarP(&searchPaths, "path", "p", []string{"."}, "Path(s) to search")
	searchCmd.Flags().BoolVarP(&searchRegex, "regex", "r", false, "Enable regular expression search")
	searchCmd.Flags().IntVarP(&searchMax, "max", "m", 200, "Maximum results to display")

	return searchCmd
}
