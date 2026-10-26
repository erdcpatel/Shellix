package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"shellix/internal/web"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewWebCmd starts the embedded zero-latency web review and developer workbench
func NewWebCmd(app *AppContext) *cobra.Command {
	var port int
	var host string
	var noBrowser bool

	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Launch zero-latency embedded web review and workbench dashboard",
		Long: `Shellix Embedded Web Workbench is a high-speed, local review console.
It serves an embedded, zero-dependency Linear-dark themed UI for inspecting side-by-side
diffs, browsing data tables, viewing visual pipeline execution waves, and pure-local search.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()

			cfg := web.ServerConfig{
				Host:      host,
				Port:      port,
				Workspace: cwd,
				GitEngine: app.GitEngine,
			}

			server := web.NewServer(cfg)
			actualPort, err := server.Start()
			if err != nil {
				return fmt.Errorf("failed starting web workbench: %w", err)
			}
			defer func() { _ = server.Stop() }()

			serverURL := server.URL()

			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
			borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
			keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00BFFF"))
			valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

			fmt.Println(titleStyle.Render("⚡ Shellix Zero-Latency Web Workbench"))
			fmt.Println(borderStyle.Render("─────────────────────────────────────────────────────────────────────────────"))
			fmt.Printf("  • Local Server   : %s\n", valStyle.Render(serverURL))
			fmt.Printf("  • Bound Port     : %s\n", keyStyle.Render(fmt.Sprintf("%d", actualPort)))
			fmt.Printf("  • Workspace Root : %s\n", dimStyle.Render(cwd))
			fmt.Println(borderStyle.Render("─────────────────────────────────────────────────────────────────────────────"))
			fmt.Println(dimStyle.Render("Press Ctrl+C to stop the web server."))

			if !noBrowser {
				_ = web.OpenBrowser(serverURL)
			}

			// Block until interrupt
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\n" + dimStyle.Render("Shutting down web server gracefully..."))
			return nil
		},
	}

	webCmd.Flags().IntVarP(&port, "port", "p", 7890, "Port to listen on (falls back to next port if busy)")
	webCmd.Flags().StringVar(&host, "host", "127.0.0.1", "Host interface to bind")
	webCmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not automatically launch web browser")

	return webCmd
}
