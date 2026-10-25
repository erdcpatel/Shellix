package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewBookmarkCmd handles filesystem path bookmarks
func NewBookmarkCmd(app *AppContext) *cobra.Command {
	bmCmd := &cobra.Command{
		Use:   "bm",
		Short: "Filesystem bookmarks manager",
	}

	bmAddCmd := &cobra.Command{
		Use:   "add <name> [path]",
		Short: "Bookmark a directory",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 2 {
				path = args[1]
			}
			if err := app.ShortcutMgr.AddBookmark(args[0], path); err != nil {
				return err
			}
			bm, _ := app.ShortcutMgr.GetBookmark(args[0])
			fmt.Printf("✔ Bookmarked '%s' -> %s\n", args[0], bm.Path)
			return nil
		},
	}

	bmListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all bookmarks",
		Run: func(cmd *cobra.Command, args []string) {
			bms := app.ShortcutMgr.ListBookmarks()
			if len(bms) == 0 {
				fmt.Println("No bookmarks saved. Use 'sx bm add <name> [path]' to add one.")
				return
			}
			fmt.Println("Saved Bookmarks:")
			for _, b := range bms {
				fmt.Printf("  • %-15s -> %s\n", b.Name, b.Path)
			}
		},
	}

	bmGetCmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Print target path for a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bm, ok := app.ShortcutMgr.GetBookmark(args[0])
			if !ok {
				return fmt.Errorf("bookmark '%s' not found", args[0])
			}
			fmt.Println(bm.Path)
			return nil
		},
	}

	bmRmCmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.ShortcutMgr.DeleteBookmark(args[0])
		},
	}

	bmCmd.AddCommand(bmAddCmd, bmListCmd, bmGetCmd, bmRmCmd)
	return bmCmd
}

// NewSnippetCmd handles reusable command snippets
func NewSnippetCmd(app *AppContext) *cobra.Command {
	snipCmd := &cobra.Command{
		Use:   "snip",
		Short: "Command snippets manager",
	}

	var snipDesc string
	snipAddCmd := &cobra.Command{
		Use:   "add <name> <command>",
		Short: "Save a reusable command snippet",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.ShortcutMgr.AddSnippet(args[0], args[1], snipDesc); err != nil {
				return err
			}
			fmt.Printf("✔ Saved snippet '%s': %s\n", args[0], args[1])
			return nil
		},
	}
	snipAddCmd.Flags().StringVarP(&snipDesc, "desc", "d", "", "Description of snippet")

	snipListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all command snippets",
		Run: func(cmd *cobra.Command, args []string) {
			snips := app.ShortcutMgr.ListSnippets()
			if len(snips) == 0 {
				fmt.Println("No snippets saved. Use 'sx snip add <name> <command>' to create one.")
				return
			}
			fmt.Println("Saved Snippets:")
			for _, s := range snips {
				desc := ""
				if s.Description != "" {
					desc = fmt.Sprintf(" (%s)", s.Description)
				}
				fmt.Printf("  • %-15s : %s%s\n", s.Name, s.Command, desc)
			}
		},
	}

	snipRunCmd := &cobra.Command{
		Use:   "run <name>",
		Short: "Execute a command snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.ShortcutMgr.RunSnippet(args[0])
		},
	}

	snipRmCmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a command snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.ShortcutMgr.DeleteSnippet(args[0])
		},
	}

	snipCmd.AddCommand(snipAddCmd, snipListCmd, snipRunCmd, snipRmCmd)
	return snipCmd
}
