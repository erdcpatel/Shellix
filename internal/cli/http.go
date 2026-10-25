package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"shellix/internal/httpclient"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// NewHTTPCmd constructs the HTTP & API client CLI
func NewHTTPCmd(app *AppContext) *cobra.Command {
	var headers []string
	var data string
	var bearer string
	var timeoutStr string
	var insecure bool
	var verbose bool
	var outputFile string

	httpCmd := &cobra.Command{
		Use:     "http",
		Aliases: []string{"api", "curl"},
		Short:   "High-speed HTTP & API terminal client with response diffing",
		Long:    "Modern developer API client with latency benchmarks, syntax highlighting, and endpoint comparison",
	}

	httpCmd.PersistentFlags().StringArrayVarP(&headers, "header", "H", []string{}, "Custom request header (key:value)")
	httpCmd.PersistentFlags().StringVarP(&data, "data", "d", "", "HTTP request body")
	httpCmd.PersistentFlags().StringVarP(&bearer, "bearer", "b", "", "Bearer authorization token")
	httpCmd.PersistentFlags().StringVarP(&timeoutStr, "timeout", "t", "15s", "Request timeout duration")
	httpCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "Allow insecure TLS connections")
	httpCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show headers and timing breakdown")
	httpCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "", "Save response payload to file")

	makeMethodCmd := func(method string) *cobra.Command {
		return &cobra.Command{
			Use:   strings.ToLower(method) + " <url>",
			Short: fmt.Sprintf("Perform HTTP %s request", method),
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				timeout, _ := time.ParseDuration(timeoutStr)
				client := httpclient.NewClient(timeout, insecure)

				hdrMap := parseHeaders(headers)
				opts := httpclient.RequestOptions{
					Method:      method,
					URL:         args[0],
					Headers:     hdrMap,
					Body:        []byte(data),
					BearerToken: bearer,
					Timeout:     timeout,
					InsecureTLS: insecure,
				}

				resp, err := client.Execute(opts)
				if err != nil {
					return err
				}

				return printResponse(resp, verbose, outputFile)
			},
		}
	}

	// 1. Standard methods: sx http get, post, put, delete, patch
	httpCmd.AddCommand(
		makeMethodCmd("GET"),
		makeMethodCmd("POST"),
		makeMethodCmd("PUT"),
		makeMethodCmd("DELETE"),
		makeMethodCmd("PATCH"),
	)

	// 2. Generic request: sx http request <METHOD> <URL>
	requestCmd := &cobra.Command{
		Use:   "request <method> <url>",
		Short: "Execute custom HTTP request",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			timeout, _ := time.ParseDuration(timeoutStr)
			client := httpclient.NewClient(timeout, insecure)

			hdrMap := parseHeaders(headers)
			opts := httpclient.RequestOptions{
				Method:      strings.ToUpper(args[0]),
				URL:         args[1],
				Headers:     hdrMap,
				Body:        []byte(data),
				BearerToken: bearer,
				Timeout:     timeout,
				InsecureTLS: insecure,
			}

			resp, err := client.Execute(opts)
			if err != nil {
				return err
			}

			return printResponse(resp, verbose, outputFile)
		},
	}
	httpCmd.AddCommand(requestCmd)

	// 3. Diff endpoints: sx http diff <url1> <url2>
	diffCmd := &cobra.Command{
		Use:   "diff <url1> <url2>",
		Short: "Compare responses and benchmark latency between two API endpoints",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			timeout, _ := time.ParseDuration(timeoutStr)
			client := httpclient.NewClient(timeout, insecure)

			hdrMap := parseHeaders(headers)
			opts := httpclient.RequestOptions{
				Method:      "GET",
				Headers:     hdrMap,
				Body:        []byte(data),
				BearerToken: bearer,
				Timeout:     timeout,
				InsecureTLS: insecure,
			}

			fmt.Println("🚀 Comparing API endpoints...")
			diffRes, err := client.CompareEndpoints(args[0], args[1], opts)
			if err != nil {
				return err
			}

			report := httpclient.RenderDiffReport(diffRes)
			fmt.Print(report)
			return nil
		},
	}
	httpCmd.AddCommand(diffCmd)

	return httpCmd
}

func parseHeaders(raw []string) map[string]string {
	m := make(map[string]string)
	for _, h := range raw {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m
}

func printResponse(resp *httpclient.ResponseSummary, verbose bool, outputFile string) error {
	statusStyle := lipgloss.NewStyle().Bold(true)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		statusStyle = statusStyle.Foreground(lipgloss.Color("#04B575"))
	} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		statusStyle = statusStyle.Foreground(lipgloss.Color("#FFA500"))
	} else {
		statusStyle = statusStyle.Foreground(lipgloss.Color("#FF4500"))
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	metaStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00BFFF"))

	fmt.Printf("%s  •  %s  •  %s\n",
		statusStyle.Render(fmt.Sprintf("%d %s", resp.StatusCode, resp.StatusText)),
		dimStyle.Render(fmt.Sprintf("%d bytes", resp.ContentLength)),
		metaStyle.Render(fmt.Sprintf("latency: %s", resp.Timing.TotalTime.Round(time.Millisecond))),
	)

	if verbose {
		fmt.Println(dimStyle.Render("── Response Headers ─────────────────────────────────────────"))
		for k, v := range resp.Headers {
			fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
		}
		if resp.Timing.TTFB > 0 {
			fmt.Println(dimStyle.Render("── Network Latency Breakdown ────────────────────────────────"))
			if resp.Timing.DNSDuration > 0 {
				fmt.Printf("  DNS Lookup:     %s\n", resp.Timing.DNSDuration.Round(100*time.Microsecond))
			}
			if resp.Timing.ConnDuration > 0 {
				fmt.Printf("  TCP Connect:    %s\n", resp.Timing.ConnDuration.Round(100*time.Microsecond))
			}
			if resp.Timing.TLSDuration > 0 {
				fmt.Printf("  TLS Handshake:  %s\n", resp.Timing.TLSDuration.Round(100*time.Microsecond))
			}
			fmt.Printf("  Time to First Byte (TTFB): %s\n", resp.Timing.TTFB.Round(100*time.Microsecond))
		}
		fmt.Println(dimStyle.Render("─────────────────────────────────────────────────────────────"))
	}

	// Output file handling
	if outputFile != "" {
		if err := os.WriteFile(outputFile, resp.Body, 0644); err != nil {
			return fmt.Errorf("failed to save response body to %s: %w", outputFile, err)
		}
		fmt.Printf("✨ Payload saved to %s\n", outputFile)
		return nil
	}

	// Pretty-print JSON if applicable
	if resp.IsJSON && len(resp.Body) > 0 {
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, resp.Body, "", "  "); err == nil {
			fmt.Println(pretty.String())
			return nil
		}
	}

	fmt.Println(string(resp.Body))
	return nil
}
