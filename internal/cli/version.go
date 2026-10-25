package cli

import (
	"fmt"

	"shellix/internal/core"

	"github.com/spf13/cobra"
)

// NewVersionCmd prints application build information
func NewVersionCmd(app *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Shellix version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Shellix version %s\n", core.GetVersion())
		},
	}
}
